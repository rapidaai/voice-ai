// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_outbound "github.com/rapidaai/api/assistant-api/sip/internal/outbound"
	"github.com/rapidaai/pkg/commons"
)

// Server wraps sipgo for handling SIP signaling.
type Server struct {
	mu          sync.RWMutex
	lifecycleMu sync.Mutex
	logger      commons.Logger
	state       atomic.Int32
	listenerWG  sync.WaitGroup
	closeOnce   sync.Once

	userAgent    *sipgo.UserAgent
	server       *sipgo.Server
	client       *sipgo.Client
	listenConfig *ListenConfig // Shared server listen config (address, port, transport)

	rtpPortRangeStart    int
	rtpPortRangeEnd      int
	symmetricRTP         bool
	ignoreLocalAddrInSDP bool
	rtpPortStats         *RTPPortStats

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
	pendingInvites map[inboundInviteKey]*pendingInvite
	// cancelledInvites tracks early-dialog INVITEs that received CANCEL while
	// INVITE processing is still in-flight.
	cancelledInvites                 map[inboundInviteKey]bool
	rejectedInvites                  map[inboundInviteKey]inboundRejectedInvite
	sessionCount                     atomic.Int64
	inboundACKTimeout                time.Duration
	inboundFinalResponseRetryInitial time.Duration
	inboundFinalResponseRetryMax     time.Duration
	inboundRingingInterval           time.Duration

	// Middlewares are called by index for each incoming INVITE.
	middlewares []Middleware

	// Event callbacks
	onApplicationReady   func(session *Session, requestURI string, callAddress CallAddress) error
	onApplicationCleanup func(session *Session)
	onInvite             func(session *Session, requestURI string, callAddress CallAddress) error
	onBye                func(session *Session) error
	onCancel             func(session *Session) error
	onError              func(session *Session, err error)

	ctx    context.Context
	cancel context.CancelFunc
}

type pendingInvite struct {
	req                  *sip.Request
	tx                   sip.ServerTransaction
	finalResponseStarted bool
}

type inboundInviteKey struct {
	callID  string
	fromTag string
}

type inboundRejectedInvite struct {
	statusCode     int
	reason         string
	includeContact bool
	expiresAt      time.Time
}

// ListenConfig holds shared server configuration (not tenant-specific)
type ListenConfig struct {
	Address    string `json:"address" mapstructure:"address"`         // Bind address (e.g. 0.0.0.0)
	ExternalIP string `json:"external_ip" mapstructure:"external_ip"` // Public/reachable IP for SDP and Contact headers
	// AllowLoopbackExternalIP permits localhost advertised addresses in local test environments.
	AllowLoopbackExternalIP bool      `json:"allow_loopback_external_ip" mapstructure:"allow_loopback_external_ip"`
	Port                    int       `json:"port" mapstructure:"port"`
	Transport               Transport `json:"transport" mapstructure:"transport"`
}

// GetExternalIP returns the external/advertised IP for SDP and SIP Contact headers.
// ExternalIP must be explicitly configured (SIP__EXTERNAL_IP) for production use.
// Falls back to Address only if ExternalIP is not set.
func (c *ListenConfig) GetExternalIP() string {
	if c == nil {
		return ""
	}
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
	if c == nil {
		return ""
	}
	return c.Address
}

// GetListenAddr returns the address to listen on
func (c *ListenConfig) GetListenAddr() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

func (c *ListenConfig) SIPContactHeader() sip.ContactHeader {
	return internal_outbound.BuildContactHeader(internal_outbound.ContactConfig{
		ExternalIP: c.GetExternalIP(),
		Port:       c.Port,
		Transport:  internal_outbound.Transport(c.Transport),
	})
}

// ServerConfig holds configuration for creating a SIP server
// Multi-tenant: Only holds shared listen config, tenant config resolved per-call
type ServerConfig struct {
	ListenConfig         *ListenConfig // Shared server listen configuration
	Middlewares          []Middleware  // Resolves tenant-specific config per-call
	Logger               commons.Logger
	RTPPortRangeStart    int  // Start of RTP port range.
	RTPPortRangeEnd      int  // End of RTP port range, inclusive.
	SymmetricRTP         bool // Updates the remote RTP target from received packet sources.
	IgnoreLocalAddrInSDP bool // Enables symmetric RTP when SDP advertises a private remote address.
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("server config is required")
	}
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
	if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
		return fmt.Errorf("rtp_port_range must be specified")
	}
	if c.RTPPortRangeStart > c.RTPPortRangeEnd {
		return fmt.Errorf("rtp_port_range_start must be less than or equal to rtp_port_range_end")
	}
	return nil
}

// NewServer creates a new shared SIP server instance
// Multi-tenant: Server listens on shared address, config resolved per-call via middleware.
func NewServer(ctx context.Context, cfg *ServerConfig) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, NewSIPError("NewServer", "", "configuration validation failed", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	serverCtx, cancel := context.WithCancel(ctx)
	listenConfig := cloneListenConfig(cfg.ListenConfig)

	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(internal_outbound.SIPUserAgent),
		sipgo.WithUserAgentTransactionLayerOptions(
			sip.WithTransactionLayerUnhandledResponseHandler(func(*sip.Response) {}),
		),
	)
	if err != nil {
		cancel()
		return nil, NewSIPError("NewServer", "", "failed to create SIP user agent", err)
	}

	server, err := sipgo.NewServer(ua)
	if err != nil {
		cancel()
		_ = ua.Close()
		return nil, NewSIPError("NewServer", "", "failed to create SIP server", err)
	}

	resolvedIP := listenConfig.GetExternalIP()

	// Use the external/public IP for SIP Via/Contact headers so remote peers can reach us
	clientOpts := []sipgo.ClientOption{
		sipgo.WithClientHostname(resolvedIP),
	}
	if listenConfig.Port > 0 {
		clientOpts = append(clientOpts, sipgo.WithClientPort(listenConfig.Port))
	}

	client, err := sipgo.NewClient(ua, clientOpts...)
	if err != nil {
		cancel()
		_ = ua.Close()
		return nil, NewSIPError("NewServer", "", "failed to create SIP client", err)
	}

	// Build the Contact header used for outbound dialog sessions.
	// Uses the external IP so the remote side can route subsequent requests back to us.
	contactHDR := listenConfig.SIPContactHeader()

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
		logger:                           cfg.Logger,
		userAgent:                        ua,
		server:                           server,
		client:                           client,
		listenConfig:                     listenConfig,
		rtpPortRangeStart:                cfg.RTPPortRangeStart,
		rtpPortRangeEnd:                  cfg.RTPPortRangeEnd,
		symmetricRTP:                     cfg.SymmetricRTP,
		ignoreLocalAddrInSDP:             cfg.IgnoreLocalAddrInSDP,
		rtpPortStats:                     &RTPPortStats{},
		dialogClientCache:                dialogClientCache,
		dialogServerCache:                dialogServerCache,
		middlewares:                      append([]Middleware(nil), cfg.Middlewares...),
		sessions:                         make(map[string]*Session),
		lifecycles:                       make(map[string]*CallLifecycle),
		pendingInvites:                   make(map[inboundInviteKey]*pendingInvite),
		cancelledInvites:                 make(map[inboundInviteKey]bool),
		rejectedInvites:                  make(map[inboundInviteKey]inboundRejectedInvite),
		inboundACKTimeout:                defaultInboundACKTimeout,
		inboundFinalResponseRetryInitial: defaultInboundFinalResponseRetryInitial,
		inboundFinalResponseRetryMax:     defaultInboundFinalResponseRetryMax,
		inboundRingingInterval:           defaultInboundRingingInterval,
		ctx:                              serverCtx,
		cancel:                           cancel,
	}

	s.state.Store(int32(ServerStateCreated))
	s.registerHandlers()

	return s, nil
}

func (s *Server) useSymmetricRTPForRemoteIP(remoteIP string) bool {
	if s == nil {
		return false
	}
	if s.symmetricRTP {
		return true
	}
	if !s.ignoreLocalAddrInSDP {
		return false
	}
	remoteAddr, err := netip.ParseAddr(remoteIP)
	return err == nil && remoteAddr.IsPrivate()
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
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if !s.state.CompareAndSwap(int32(ServerStateCreated), int32(ServerStateRunning)) {
		return fmt.Errorf("server is not in created state")
	}

	listenAddr := s.listenConfig.GetListenAddr()
	transport := s.listenConfig.Transport.String()
	if transport == "" {
		transport = "udp"
	}

	ready := make(chan struct{})
	result := make(chan error, 1)
	var listenerReady atomic.Bool
	listenCtx := context.WithValue(s.ctx, sipgo.ListenReadyCtxKey, sipgo.ListenReadyFuncCtxValue(func(string, string) {
		listenerReady.Store(true)
		close(ready)
	}))

	s.listenerWG.Add(1)
	go func() {
		err := s.server.ListenAndServe(listenCtx, transport, listenAddr)
		result <- err
		s.listenerWG.Done()
		if !listenerReady.Load() {
			return
		}
		if err != nil && s.ctx.Err() == nil {
			s.logger.Errorw("SIP server stopped unexpectedly",
				"error", err,
				"address", listenAddr)
		}
		s.Stop()
	}()

	select {
	case <-ready:
		s.logger.Infow("SIP server started",
			"address", listenAddr,
			"transport", transport)
		return nil
	case err := <-result:
		s.state.CompareAndSwap(int32(ServerStateRunning), int32(ServerStateStopped))
		if s.cancel != nil {
			s.cancel()
		}
		s.closeTransport()
		s.listenerWG.Wait()
		if err == nil {
			err = fmt.Errorf("listener stopped before becoming ready")
		}
		return NewSIPError("Start", "", "failed to start SIP listener", err)
	case <-s.ctx.Done():
		s.state.CompareAndSwap(int32(ServerStateRunning), int32(ServerStateStopped))
		s.closeTransport()
		s.listenerWG.Wait()
		return NewSIPError("Start", "", "SIP listener startup cancelled", s.ctx.Err())
	}
}

// Stop stops the SIP server gracefully
func (s *Server) Stop() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	for {
		state := ServerState(s.state.Load())
		if state == ServerStateStopped {
			return
		}
		if s.state.CompareAndSwap(int32(state), int32(ServerStateStopped)) {
			break
		}
	}

	s.logger.Infow("Stopping SIP server")

	// Cancel context first to stop accepting new calls
	if s.cancel != nil {
		s.cancel()
	}

	// End all active sessions
	s.mu.RLock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.mu.RUnlock()

	for _, session := range sessions {
		_ = s.EndCallWithReason(session, LifecycleReasonServerStop)
	}

	s.closeTransport()
	s.listenerWG.Wait()

	s.logger.Infow("SIP server stopped", "sessions_ended", len(sessions))
}

func (s *Server) closeTransport() {
	s.closeOnce.Do(func() {
		if s.userAgent != nil {
			_ = s.userAgent.Close()
		}
	})
}

// SetMiddlewares sets the ordered middleware list for all SIP requests.
//
// Example:
//
//	server.SetMiddlewares(
//	    []Middleware{RouteMiddleware, VaultMiddleware},
//	)
func (s *Server) SetMiddlewares(middlewares []Middleware) {
	filtered := make([]Middleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if middleware != nil {
			filtered = append(filtered, middleware)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middlewares = filtered
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	return s.state.Load() == int32(ServerStateRunning)
}

// Client returns the underlying sipgo client for outbound requests (e.g., REGISTER).
func (s *Server) Client() *sipgo.Client {
	return s.client
}

// ListenConfig returns the shared server listen configuration.
func (s *Server) GetListenConfig() *ListenConfig {
	return cloneListenConfig(s.listenConfig)
}

func cloneListenConfig(config *ListenConfig) *ListenConfig {
	if config == nil {
		return nil
	}
	clone := *config
	return &clone
}

// SessionCount returns the number of active sessions
func (s *Server) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

func (s *Server) SetOnApplicationReady(fn func(session *Session, requestURI string, callAddress CallAddress) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onApplicationReady = fn
}

func (s *Server) SetOnApplicationCleanup(fn func(session *Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onApplicationCleanup = fn
}

// SetOnInvite sets the callback for answered INVITE requests.
func (s *Server) SetOnInvite(fn func(session *Session, requestURI string, callAddress CallAddress) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onInvite = fn
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
