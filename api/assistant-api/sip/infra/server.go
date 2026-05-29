// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rapidaai/pkg/commons"
	"github.com/redis/go-redis/v9"
)

// ServerState represents the state of the SIP server
type ServerState int32

const (
	ServerStateCreated ServerState = iota
	ServerStateRunning
	ServerStateStopped
)

// SIPRequestContext contains information about an incoming SIP request.
// Used by the middleware chain to resolve config for every SIP request.
//
// Middleware enriches this context as it flows through the chain:
//
//	routingMiddleware   → resolves assistant by DID lookup, sets Extra["auth"]
//	assistantMiddleware → loads assistant entity, sets Extra["assistant"]
//	vaultConfigResolver → fetches SIP config from vault, sets Extra["sip_config"]
type SIPRequestContext struct {
	Method  string // SIP method (INVITE, REGISTER, BYE, etc.)
	CallID  string
	FromURI string
	ToURI   string
	SDPInfo *SDPMediaInfo

	// Authentication fields extracted from URI userinfo
	// Parsed from: sip:{assistantID}:{apiKey}@host
	APIKey      string // API key (password part of userinfo)
	AssistantID string // Assistant ID (user part of userinfo)

	// Extra holds middleware-resolved state (auth principal, assistant entity, etc.).
	// Using interface{} keeps the infra package decoupled from business types.
	// Keys: "auth" → types.SimplePrinciple, "assistant" → *Assistant, "sip_config" → *Config
	Extra map[string]interface{}
}

// Set stores a value in the middleware context.
func (c *SIPRequestContext) Set(key string, value interface{}) {
	if c.Extra == nil {
		c.Extra = make(map[string]interface{})
	}
	c.Extra[key] = value
}

// Get retrieves a value from the middleware context.
func (c *SIPRequestContext) Get(key string) (interface{}, bool) {
	if c.Extra == nil {
		return nil, false
	}
	v, ok := c.Extra[key]
	return v, ok
}

// InviteResult contains the resolved configuration for handling the call
type InviteResult struct {
	Config      *Config // Tenant-specific config (RTP ports, credentials, etc.)
	ShouldAllow bool    // Whether to accept the call
	RejectCode  int     // SIP response code if rejecting (e.g., 403, 404)
	RejectMsg   string  // Optional message for rejection

	// Extra carries middleware-resolved state (auth, assistant, etc.) back to the
	// infra layer so it can be stored as session metadata. The server copies this
	// map onto the Session after creation, making it available to onInvite handlers.
	Extra map[string]interface{}
}

// Reject creates an InviteResult that rejects the call with the given SIP code and message.
func Reject(code int, msg string) *InviteResult {
	return &InviteResult{ShouldAllow: false, RejectCode: code, RejectMsg: msg}
}

// Allow creates an InviteResult that accepts the call with the resolved config.
func Allow(config *Config) *InviteResult {
	return &InviteResult{ShouldAllow: true, Config: config}
}

// AllowWithExtra creates an InviteResult that accepts the call and carries
// resolved state (auth principal, assistant entity, etc.) so the infra layer
// can propagate it to session metadata.
func AllowWithExtra(config *Config, extra map[string]interface{}) *InviteResult {
	return &InviteResult{ShouldAllow: true, Config: config, Extra: extra}
}

// ConfigResolver resolves tenant-specific config from a SIP request.
// Returns an InviteResult with Config (for RTP/SIP setup) and optionally Extra
// (auth, assistant, etc. to be stored as session metadata).
type ConfigResolver func(ctx *SIPRequestContext) (*InviteResult, error)

// Middleware processes a SIP request context and either enriches it or rejects it.
// Each middleware receives the context, enriches it (e.g., sets auth, assistant),
// and calls next() to continue the chain. Returning an InviteResult with
// ShouldAllow=false short-circuits the chain.
//
// Example chain for INVITE:
//
//	routingMiddleware → assistantMiddleware → vaultConfigResolver
type Middleware func(ctx *SIPRequestContext, next func() (*InviteResult, error)) (*InviteResult, error)

// MiddlewareChain composes a slice of Middleware into a single ConfigResolver.
// The chain executes each middleware in order; the final handler is called
// if all middlewares pass. This replaces the monolithic ConfigResolver approach
// with composable, testable middleware functions.
func MiddlewareChain(middlewares []Middleware, final ConfigResolver) ConfigResolver {
	return func(ctx *SIPRequestContext) (*InviteResult, error) {
		var run func(i int) (*InviteResult, error)
		run = func(i int) (*InviteResult, error) {
			if i >= len(middlewares) {
				return final(ctx)
			}
			return middlewares[i](ctx, func() (*InviteResult, error) {
				return run(i + 1)
			})
		}
		return run(0)
	}
}

// Server wraps sipgo for handling SIP signaling
// Uses native SIP signaling (UDP/TCP/TLS) - no WebSocket needed
// Multi-tenant: Config is resolved per-call via ConfigResolver callback
type Server struct {
	mu     sync.RWMutex
	logger commons.Logger
	state  atomic.Int32

	ua           *sipgo.UserAgent
	server       *sipgo.Server
	client       *sipgo.Client
	listenConfig *ListenConfig     // Shared server listen config (address, port, transport)
	rtpAllocator *RTPPortAllocator // Allocates RTP ports from configured range

	// Outbound dialog cache — routes incoming BYE/re-INVITE to the correct
	// DialogClientSession. Without this, BYE from the remote side is handled
	// only at the Session level and the sipgo dialog stays in limbo.
	dialogClientCache *sipgo.DialogClientCache

	// Inbound dialog cache — manages UAS dialog state for inbound calls so we
	// can send BYE when the assistant ends the conversation. Without this,
	// ending an inbound call only does local cleanup and the remote PBX keeps
	// the call alive until timeout.
	dialogServerCache *sipgo.DialogServerCache

	sessions map[string]*Session
	// lifecycles owns state transitions for active calls.
	lifecycles map[string]*CallLifecycle
	// pendingInvites keeps active INVITE server transactions until a final
	// response is sent, so CANCEL can terminate the original INVITE with 487.
	pendingInvites map[string]*pendingInvite
	// cancelledInvites tracks call-ids that received CANCEL while INVITE
	// processing is still in-flight.
	cancelledInvites map[string]bool
	sessionCount     atomic.Int64

	// Multi-tenant config resolver - called for each incoming INVITE
	configResolver ConfigResolver

	// Event callbacks
	onInvite func(session *Session, fromURI, toURI string) error
	onCreated func(session *Session, fromURI, toURI string)
	onRinging func(session *Session, fromURI, toURI string)
	onAnswered func(session *Session, fromURI, toURI string)
	onBye    func(session *Session) error
	onCancel func(session *Session) error
	onError  func(session *Session, err error)

	ctx    context.Context
	cancel context.CancelFunc
}

type pendingInvite struct {
	req *sip.Request
	tx  sip.ServerTransaction
}

// ListenConfig holds shared server configuration (not tenant-specific)
type ListenConfig struct {
	Address    string    `json:"address" mapstructure:"address"`         // Bind address (e.g. 0.0.0.0)
	ExternalIP string    `json:"external_ip" mapstructure:"external_ip"` // Public/reachable IP for SDP and Contact headers
	Port       int       `json:"port" mapstructure:"port"`
	Transport  Transport `json:"transport" mapstructure:"transport"`
}

// GetExternalIP returns the external/advertised IP for SDP and SIP Contact headers.
// ExternalIP must be explicitly configured (SIP__EXTERNAL_IP) for production use.
// Falls back to Address only if ExternalIP is not set.
func (c *ListenConfig) GetExternalIP() string {
	if c.ExternalIP != "" {
		return c.ExternalIP
	}
	return c.Address
}

// GetBindAddress returns the address to bind RTP sockets to.
// This is the actual local interface address (e.g. 0.0.0.0) — NOT the
// external/public IP. RTP sockets must bind to a local interface, while
// the external IP is only advertised in SDP so the remote peer knows
// where to send its RTP packets.
func (c *ListenConfig) GetBindAddress() string {
	return c.Address
}

// GetListenAddr returns the address to listen on
func (c *ListenConfig) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

// ServerConfig holds configuration for creating a SIP server
// Multi-tenant: Only holds shared listen config, tenant config resolved per-call
type ServerConfig struct {
	ListenConfig      *ListenConfig  // Shared server listen configuration
	ConfigResolver    ConfigResolver // Resolves tenant-specific config per-call
	Logger            commons.Logger
	RedisClient       *redis.Client // Redis client for distributed RTP port allocation
	RTPPortRangeStart int           // Start of RTP port range (even, >= 1024)
	RTPPortRangeEnd   int           // End of RTP port range (exclusive)
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.ListenConfig == nil {
		return fmt.Errorf("listen config is required")
	}
	if c.ListenConfig.Address == "" {
		return fmt.Errorf("listen address is required")
	}
	if c.ListenConfig.Port <= 0 || c.ListenConfig.Port > 65535 {
		return fmt.Errorf("invalid listen port: %d", c.ListenConfig.Port)
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if c.RedisClient == nil {
		return fmt.Errorf("redis client is required for distributed RTP port allocation")
	}
	if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
		return fmt.Errorf("rtp_port_range must be specified")
	}
	if c.RTPPortRangeStart >= c.RTPPortRangeEnd {
		return fmt.Errorf("rtp_port_range_start must be less than rtp_port_range_end")
	}
	return nil
}

// NewServer creates a new shared SIP server instance
// Multi-tenant: Server listens on shared address, config resolved per-call via ConfigResolver
func NewServer(ctx context.Context, cfg *ServerConfig) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, NewSIPError("NewServer", "", "configuration validation failed", err)
	}

	serverCtx, cancel := context.WithCancel(ctx)

	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent("RapidaVoiceAI"),
		sipgo.WithUserAgentTransactionLayerOptions(
			sip.WithTransactionLayerUnhandledResponseHandler(func(r *sip.Response) {
				// Absorb retransmissions silently — these are expected when
				// the remote side retransmits responses before the transaction completes
				cfg.Logger.Debug("Unhandled SIP response (retransmission)",
					"status", r.StatusCode,
					"reason", r.Reason)
			}),
		),
	)
	if err != nil {
		cancel()
		return nil, NewSIPError("NewServer", "", "failed to create SIP user agent", err)
	}

	server, err := sipgo.NewServer(ua)
	if err != nil {
		cancel()
		return nil, NewSIPError("NewServer", "", "failed to create SIP server", err)
	}

	// Log full ListenConfig so config issues are immediately visible
	resolvedIP := cfg.ListenConfig.GetExternalIP()
	cfg.Logger.Infow("SIP server config",
		"bind_address", cfg.ListenConfig.Address,
		"external_ip_config", cfg.ListenConfig.ExternalIP,
		"external_ip_resolved", resolvedIP,
		"port", cfg.ListenConfig.Port,
		"transport", cfg.ListenConfig.Transport)
	if resolvedIP == "" || resolvedIP == "0.0.0.0" || resolvedIP == "::" {
		cfg.Logger.Warn("SIP ExternalIP not configured — SDP will advertise bind address which may be unroutable. Set SIP__EXTERNAL_IP in config.")
	}

	// Use the external/public IP for SIP Via/Contact headers so remote peers can reach us
	clientOpts := []sipgo.ClientOption{
		sipgo.WithClientHostname(resolvedIP),
	}
	if cfg.ListenConfig.Port > 0 {
		clientOpts = append(clientOpts, sipgo.WithClientPort(cfg.ListenConfig.Port))
	}

	client, err := sipgo.NewClient(ua, clientOpts...)
	if err != nil {
		cancel()
		return nil, NewSIPError("NewServer", "", "failed to create SIP client", err)
	}

	// Create Redis-backed distributed RTP port allocator
	rtpAllocator := NewRTPPortAllocator(cfg.RedisClient, cfg.Logger, cfg.RTPPortRangeStart, cfg.RTPPortRangeEnd)
	if err := rtpAllocator.Init(serverCtx); err != nil {
		cancel()
		return nil, NewSIPError("NewServer", "", "failed to initialize RTP port allocator", err)
	}

	// Build the Contact header used for outbound dialog sessions.
	// Uses the external IP so the remote side can route subsequent requests back to us.
	contactHDR := sip.ContactHeader{
		Address: sip.Uri{
			Scheme: "sip",
			Host:   cfg.ListenConfig.GetExternalIP(),
			Port:   cfg.ListenConfig.Port,
		},
	}

	// Create dialog client cache — routes incoming BYE/re-INVITE for outbound dialogs
	// to the correct DialogClientSession. This is essential for proper dialog lifecycle:
	// without it, BYE from the remote side never terminates the sipgo dialog, and
	// re-INVITE responses lack proper dialog context (Contact, To-tag).
	dialogClientCache := sipgo.NewDialogClientCache(client, contactHDR)

	// Create dialog server cache — manages UAS dialog state for inbound calls.
	// This allows us to send BYE when the assistant ends an inbound conversation,
	// properly tearing down the call on the remote PBX side.
	dialogServerCache := sipgo.NewDialogServerCache(client, contactHDR)

	s := &Server{
		logger:            cfg.Logger,
		ua:                ua,
		server:            server,
		client:            client,
		listenConfig:      cfg.ListenConfig,
		rtpAllocator:      rtpAllocator,
		dialogClientCache: dialogClientCache,
		dialogServerCache: dialogServerCache,
		configResolver:    cfg.ConfigResolver,
		sessions:          make(map[string]*Session),
		lifecycles:        make(map[string]*CallLifecycle),
		pendingInvites:    make(map[string]*pendingInvite),
		cancelledInvites:  make(map[string]bool),
		ctx:               serverCtx,
		cancel:            cancel,
	}

	s.state.Store(int32(ServerStateCreated))
	s.registerHandlers()

	return s, nil
}

func (s *Server) registerHandlers() {
	s.server.OnInvite(s.handleInvite)
	s.server.OnAck(s.handleAck)
	s.server.OnBye(s.handleBye)
	s.server.OnCancel(s.handleCancel)
	s.server.OnRegister(s.handleRegister)
	s.server.OnOptions(s.handleOptions)

	// Handle UPDATE — Asterisk sends UPDATE for direct_media negotiation and session timers.
	// Without this handler, sipgo responds 405 Method Not Allowed, which causes Asterisk to
	// tear down the bridge (the exact symptom: call disconnects ~2ms after answer).
	s.server.OnUpdate(s.handleUpdate)

	// Handle INFO — some PBXes send INFO for DTMF relay (RFC 2833) or session information.
	s.server.OnInfo(s.handleInfo)

	// Handle NOTIFY — sent for REFER progress, subscription events, and MWI.
	s.server.OnNotify(s.handleNotify)

	// Handle REFER — call transfer requests from the remote side.
	s.server.OnRefer(s.handleRefer)

	// Handle SUBSCRIBE — Twilio sends SUBSCRIBE for dialog-info and presence events.
	// Reject cleanly to prevent retry loops.
	s.server.OnSubscribe(s.handleSubscribe)

	// Handle MESSAGE — FreeSWITCH sends MESSAGE for T.38 fax or text-based events.
	s.server.OnMessage(s.handleMessage)

	// Catch-all for any SIP method we don't explicitly handle. Without this,
	// sipgo responds 405 Method Not Allowed which can cause Asterisk to tear down calls.
	// For in-dialog requests (known Call-ID), respond 200 OK to keep the dialog alive.
	// For out-of-dialog requests, respond 405 as before.
	s.server.OnNoRoute(s.handleUnknownRequest)
}

// Start begins listening for SIP traffic
func (s *Server) Start() error {
	if !s.state.CompareAndSwap(int32(ServerStateCreated), int32(ServerStateRunning)) {
		return fmt.Errorf("server is not in created state")
	}

	listenAddr := s.listenConfig.GetListenAddr()
	transport := s.listenConfig.Transport.String()
	if transport == "" {
		transport = "udp"
	}

	go func() {
		err := s.server.ListenAndServe(s.ctx, transport, listenAddr)
		if err != nil && s.state.Load() == int32(ServerStateRunning) {
			s.logger.Errorw("SIP server stopped unexpectedly",
				"error", err,
				"address", listenAddr)
			s.state.Store(int32(ServerStateStopped))
		}
	}()

	s.logger.Infow("SIP server started (multi-tenant)",
		"address", listenAddr,
		"transport", transport)

	return nil
}

// Stop stops the SIP server gracefully
func (s *Server) Stop() {
	if !s.state.CompareAndSwap(int32(ServerStateRunning), int32(ServerStateStopped)) {
		return // Already stopped or not running
	}

	s.logger.Infow("Stopping SIP server")

	// Cancel context first to stop accepting new calls
	s.cancel()

	// End all active sessions
	s.mu.Lock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.sessions = make(map[string]*Session)
	s.mu.Unlock()

	for _, session := range sessions {
		s.beginEnding(session, "server_stop")
		session.End()
	}

	// Release all RTP ports allocated by this instance back to Redis
	s.rtpAllocator.ReleaseAll(context.Background())

	s.logger.Infow("SIP server stopped", "sessions_ended", len(sessions))
}

// SetConfigResolver sets the callback for resolving tenant-specific config.
// For middleware-based auth, use SetMiddlewares instead.
func (s *Server) SetConfigResolver(resolver ConfigResolver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configResolver = resolver
}

// SetMiddlewares composes a middleware chain with a final handler and sets it
// as the config resolver. This is the preferred way to set up authentication
// for all SIP requests.
//
// Example:
//
//	server.SetMiddlewares(
//	    []Middleware{routingMiddleware, assistantMiddleware},
//	    vaultConfigResolver,
//	)
func (s *Server) SetMiddlewares(middlewares []Middleware, final ConfigResolver) {
	s.SetConfigResolver(MiddlewareChain(middlewares, final))
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	return s.state.Load() == int32(ServerStateRunning)
}

// AllocateRTPPort allocates an available RTP port from the shared pool.
// Callers must call ReleaseRTPPort when the port is no longer needed.
func (s *Server) AllocateRTPPort() (int, error) {
	return s.rtpAllocator.Allocate()
}

// ReleaseRTPPort returns an RTP port to the shared pool.
func (s *Server) ReleaseRTPPort(port int) {
	s.rtpAllocator.Release(port)
}

// Client returns the underlying sipgo client for outbound requests (e.g., REGISTER).
func (s *Server) Client() *sipgo.Client {
	return s.client
}

// ListenConfig returns the shared server listen configuration.
func (s *Server) GetListenConfig() *ListenConfig {
	return s.listenConfig
}

// SessionCount returns the number of active sessions
func (s *Server) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// SetOnInvite sets the callback for incoming INVITE requests
func (s *Server) SetOnInvite(fn func(session *Session, fromURI, toURI string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onInvite = fn
}

// SetOnCreated sets the callback when a call session is created/registered.
func (s *Server) SetOnCreated(fn func(session *Session, fromURI, toURI string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCreated = fn
}

// SetOnRinging sets the callback when a call reaches ringing state.
func (s *Server) SetOnRinging(fn func(session *Session, fromURI, toURI string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onRinging = fn
}

// SetOnAnswered sets the callback when a call is answered/connected.
func (s *Server) SetOnAnswered(fn func(session *Session, fromURI, toURI string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onAnswered = fn
}

// SetOnBye sets the callback for BYE requests
func (s *Server) SetOnBye(fn func(session *Session) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onBye = fn
}

// SetOnCancel sets the callback for CANCEL requests
func (s *Server) SetOnCancel(fn func(session *Session) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCancel = fn
}

// SetOnError sets the callback for error events
func (s *Server) SetOnError(fn func(session *Session, err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = fn
}
