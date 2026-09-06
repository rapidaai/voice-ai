// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

// Registration describes a SIP registration to be maintained with an external registrar.
type Registration struct {
	DID          string  // Phone number / DID being registered (e.g., "+15551234567")
	Config       *Config // SIP provider credentials (server, username, password, realm, domain)
	DeploymentID uint64  // Deployment that owns this registration.
	AssistantID  uint64  // Assistant that owns this DID
	ExpiresIn    uint32  // Desired registration duration in seconds (0 = use default)
}

// Validate checks that the registration has the minimum required fields.
func (r *Registration) Validate() error {
	if !validator.NonNil(r) || !validator.NotBlank(r.DID) {
		return ErrMissingDID
	}
	if !validator.NonNil(r.Config) || !validator.NotBlank(r.Config.Server) {
		return ErrMissingServer
	}
	return nil
}

// activeRegistration tracks a live registration and its renewal timer.
type activeRegistration struct {
	reg                  *Registration
	cancel               context.CancelFunc
	expiresAt            time.Time
	grantedExpirySeconds uint32
	callID               string
	cseq                 uint32

	renewalRetryCount int
	lastRenewalError  error
	failureClass      RegistrationFailureClass
	failureReason     RegistrationFailureReason
	statusCode        int
	statusText        string
}

func (active *activeRegistration) expired(now time.Time) bool {
	return now.After(active.expiresAt.Add(registrationExpiryGrace(active.grantedExpirySeconds)))
}

// RegistrationClient manages outbound SIP REGISTER transactions.
// Each registration is maintained with periodic renewal and supports digest auth.
// Thread-safe: all methods can be called concurrently.
type RegistrationClient struct {
	client       *sipgo.Client
	listenConfig *ListenConfig
	logger       commons.Logger
	observer     RegistrationObserver

	mu            sync.RWMutex
	registrations map[string]*activeRegistration // keyed by DID
}

// NewRegistrationClient creates a registration client using the shared sipgo client.
func NewRegistrationClient(client *sipgo.Client, listenConfig *ListenConfig, logger commons.Logger) *RegistrationClient {
	return &RegistrationClient{
		client:        client,
		listenConfig:  cloneListenConfig(listenConfig),
		logger:        logger,
		registrations: make(map[string]*activeRegistration),
	}
}

func (rc *RegistrationClient) SetObserver(observer RegistrationObserver) {
	rc.mu.Lock()
	rc.observer = observer
	rc.mu.Unlock()
}

// Register sends a REGISTER request and maintains the registration with periodic renewal.
// Handles 401/407 digest auth challenges automatically via sipgo's DoDigestAuth.
// Idempotent: calling Register for an already-registered DID replaces the existing registration.
func (rc *RegistrationClient) Register(ctx context.Context, reg *Registration) error {
	if err := reg.Validate(); err != nil {
		return newRegistrationValidationError(err)
	}

	expiresIn := reg.ExpiresIn
	if expiresIn == 0 {
		expiresIn = defaultRegisterExpiry
	}

	// Stable Call-ID per binding (RFC 3261 §10.2)
	bindingCallID := fmt.Sprintf("reg-%s-%d", reg.DID, time.Now().UnixNano())
	var cseq uint32 = 1

	grantedExpiry, err := rc.sendRegister(ctx, reg, expiresIn, bindingCallID, cseq)
	if err != nil {
		return fmt.Errorf("%w: DID %s at %s: %w", ErrRegistrationFailed, reg.DID, reg.Config.Server, err)
	}

	regCtx, cancelReg := context.WithCancel(ctx)

	rc.mu.Lock()
	if existing, ok := rc.registrations[reg.DID]; ok {
		existing.cancel()
	}
	rc.registrations[reg.DID] = &activeRegistration{
		reg:                  reg,
		cancel:               cancelReg,
		expiresAt:            time.Now().Add(time.Duration(grantedExpiry) * time.Second),
		grantedExpirySeconds: grantedExpiry,
		callID:               bindingCallID,
		cseq:                 cseq + 1,
	}
	rc.mu.Unlock()

	go rc.renewLoop(regCtx, reg, grantedExpiry)

	rc.logger.Infow("SIP registration active",
		"did", reg.DID,
		"server", reg.Config.Server,
		"assistant_id", reg.AssistantID,
		"expires_in", grantedExpiry)

	return nil
}

// Unregister sends a REGISTER with Expires: 0 to remove the registration.
// Idempotent: returns nil if the DID is not registered.
func (rc *RegistrationClient) Unregister(ctx context.Context, did string) error {
	rc.mu.RLock()
	active, ok := rc.registrations[did]
	rc.mu.RUnlock()

	if !ok {
		return nil
	}

	unregCtx, cancel := contextWithTimeout(ctx, active.reg.Config.EffectiveRegisterTimeout())
	defer cancel()

	if _, err := rc.sendRegister(unregCtx, active.reg, 0, active.callID, active.cseq); err != nil {
		rc.logger.Warnw("Failed to send REGISTER Expires:0",
			"did", did,
			"error", err)
		rc.notifyUnregisterFailed(ctx, active, err)
		return err
	}

	rc.mu.Lock()
	if current, ok := rc.registrations[did]; ok && current == active {
		active.cancel()
		delete(rc.registrations, did)
	}
	rc.mu.Unlock()

	rc.logger.Infow("SIP registration removed", "did", did)
	return nil
}

// UnregisterAll unregisters all active registrations. Called during shutdown.
func (rc *RegistrationClient) UnregisterAll(ctx context.Context) {
	rc.mu.RLock()
	dids := make([]string, 0, len(rc.registrations))
	for did := range rc.registrations {
		dids = append(dids, did)
	}
	rc.mu.RUnlock()

	for _, did := range dids {
		if err := rc.Unregister(ctx, did); err != nil {
			rc.logger.Warnw("Shutdown: failed to unregister DID",
				"did", did,
				"error", err)
		}
	}
}

// IsRegistered returns true if the given DID has an active registration.
func (rc *RegistrationClient) IsRegistered(did string) bool {
	return rc.Snapshot(did).Active
}

func (rc *RegistrationClient) Snapshot(did string) RegistrationSnapshot {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	active, ok := rc.registrations[did]
	if !ok {
		return RegistrationSnapshot{DID: did}
	}
	expired := active.expired(time.Now())
	return RegistrationSnapshot{
		DID:               did,
		Active:            !expired,
		Healthy:           !expired && active.lastRenewalError == nil,
		ExpiresAt:         active.expiresAt,
		RenewalRetryCount: active.renewalRetryCount,
		LastRenewalError:  active.lastRenewalError,
		FailureClass:      active.failureClass,
		FailureReason:     active.failureReason,
	}
}

// ActiveCount returns the number of active registrations.
func (rc *RegistrationClient) ActiveCount() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.registrations)
}

// GetRegisteredDIDs returns all currently registered DIDs.
func (rc *RegistrationClient) GetRegisteredDIDs() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	dids := make([]string, 0, len(rc.registrations))
	for did := range rc.registrations {
		dids = append(dids, did)
	}
	return dids
}

// sendRegister constructs and sends a REGISTER request, handling digest auth if challenged.
// Returns the granted expiry from the 200 OK response.
func (rc *RegistrationClient) sendRegister(ctx context.Context, reg *Registration, expiresIn uint32, bindingCallID string, cseq uint32) (uint32, error) {
	return rc.sendRegisterWithMinExpires(ctx, reg, expiresIn, bindingCallID, cseq, true)
}

func (rc *RegistrationClient) sendRegisterWithMinExpires(
	ctx context.Context,
	reg *Registration,
	expiresIn uint32,
	bindingCallID string,
	cseq uint32,
	allowMinExpiresRetry bool,
) (uint32, error) {
	cfg := reg.Config

	domain := cfg.Domain
	if !validator.NotBlank(domain) {
		domain = cfg.Server
	}

	scheme := "sip"
	if cfg.Transport == TransportTLS {
		scheme = "sips"
	}

	// Request-URI: the registrar address
	registrar := sip.Uri{
		Scheme: scheme,
		Host:   cfg.Server,
		Port:   cfg.Port,
	}

	req := sip.NewRequest(sip.REGISTER, registrar)

	// To/From: the AOR (Address of Record) being registered.
	// Per RFC 3261 §10.2, To and From are identical for REGISTER.
	aor := sip.Uri{
		Scheme: scheme,
		User:   normalizeUser(reg.DID),
		Host:   domain,
	}

	toHdr := &sip.ToHeader{Address: aor}
	fromHdr := &sip.FromHeader{
		Address: aor,
		Params:  sip.NewParams(),
	}
	fromHdr.Params.Add("tag", sip.GenerateTagN(16))

	req.AppendHeader(toHdr)
	req.AppendHeader(fromHdr)

	// Contact: where the registrar should route INVITEs for this DID
	externalIP := rc.listenConfig.GetExternalIP()
	if err := validateRegistrationContactAddress(rc.listenConfig, externalIP); err != nil {
		return 0, err
	}
	contactHdr := &sip.ContactHeader{
		Address: sip.Uri{
			Scheme: scheme,
			User:   normalizeUser(reg.DID),
			Host:   externalIP,
			Port:   rc.listenConfig.Port,
		},
	}
	req.AppendHeader(contactHdr)

	// Expires
	expiresHdr := sip.ExpiresHeader(expiresIn)
	req.AppendHeader(&expiresHdr)

	req.AppendHeader(&sip.CSeqHeader{SeqNo: cseq, MethodName: sip.REGISTER})

	callID := sip.CallIDHeader(bindingCallID)
	req.AppendHeader(&callID)

	// Max-Forwards
	maxFwd := sip.MaxForwardsHeader(70)
	req.AppendHeader(&maxFwd)

	// Apply timeout — respect parent context deadline if shorter
	reqCtx, cancel := contextWithTimeout(ctx, cfg.EffectiveRegisterTimeout())
	defer cancel()

	rc.logger.Debugw("Sending REGISTER",
		"did", reg.DID,
		"registrar", registrar.String(),
		"contact", contactHdr.Address.String(),
		"expires", expiresIn)

	resp, err := rc.client.Do(reqCtx, req)
	if err != nil {
		return 0, newRegistrationTransportError(reqCtx, err)
	}

	// Handle digest auth challenges (401 WWW-Authenticate / 407 Proxy-Authenticate)
	if resp.StatusCode == 401 || resp.StatusCode == 407 {
		rc.logger.Debugw("REGISTER auth challenge",
			"did", reg.DID,
			"status", resp.StatusCode)

		resp, err = rc.client.DoDigestAuth(reqCtx, req, resp, sipgo.DigestAuth{
			Username: cfg.Username,
			Password: cfg.Password,
		})
		if err != nil {
			return 0, newRegistrationAuthError(0, "", err)
		}
	}

	// Second 401/407 after digest auth means credentials are wrong
	if resp.StatusCode == 401 || resp.StatusCode == 407 {
		return 0, newRegistrationAuthError(resp.StatusCode, resp.Reason, ErrAuthFailed)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 423 && allowMinExpiresRetry {
			if minExpires := parseMinExpires(resp); minExpires > expiresIn {
				return rc.sendRegisterWithMinExpires(ctx, reg, minExpires, bindingCallID, cseq+1, false)
			}
		}
		return 0, newRegistrationResponseError(resp.StatusCode, resp.Reason)
	}

	// Parse granted expiry. Per RFC 3261 §10.2.4, the registrar may return the
	// granted duration either as a Contact;expires=N parameter or a top-level
	// Expires header. Contact-level takes precedence when present.
	grantedExpiry := expiresIn
	if contact := resp.GetHeader("Contact"); contact != nil {
		if exp := parseContactExpires(contact.Value()); exp > 0 {
			grantedExpiry = exp
		}
	}
	if grantedExpiry == expiresIn {
		if hdr := resp.GetHeader("Expires"); hdr != nil {
			if parsed, err := parsePositiveUint32(hdr.Value()); err == nil {
				grantedExpiry = parsed
			}
		}
	}

	return grantedExpiry, nil
}

// renewLoop periodically re-registers before the registration expires.
// Re-registers at renewalFraction (80%) of the granted expiry time.
// On failure, retries every renewRetryInterval (30s) until successful or cancelled.
func (rc *RegistrationClient) renewLoop(ctx context.Context, reg *Registration, expiresIn uint32) {
	renewInterval := time.Duration(float64(expiresIn)*renewalFraction) * time.Second
	timer := time.NewTimer(renewInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			rc.mu.RLock()
			active, ok := rc.registrations[reg.DID]
			rc.mu.RUnlock()
			if !ok {
				return
			}

			renewCtx, cancel := contextWithTimeout(ctx, reg.Config.EffectiveRegisterTimeout())
			grantedExpiry, err := rc.sendRegister(renewCtx, reg, expiresIn, active.callID, active.cseq)
			cancel()

			if err != nil {
				nextRetryAt := time.Now().Add(renewRetryInterval)
				expired, current := rc.markRenewalFailed(ctx, active, err, nextRetryAt)
				rc.logger.Warnw("Re-registration failed",
					"did", reg.DID,
					"error", err,
					"retry_in", renewRetryInterval)
				if expired || !current {
					return
				}
				timer.Reset(renewRetryInterval)
				continue
			}

			var event RegistrationEvent
			renewed := false
			rc.mu.Lock()
			if current, ok := rc.registrations[reg.DID]; ok && current == active {
				active.expiresAt = time.Now().Add(time.Duration(grantedExpiry) * time.Second)
				active.grantedExpirySeconds = grantedExpiry
				active.cseq++
				active.renewalRetryCount = 0
				active.lastRenewalError = nil
				active.failureClass = ""
				active.failureReason = ""
				active.statusCode = 0
				active.statusText = ""
				event = rc.registrationEvent(active)
				renewed = true
			}
			rc.mu.Unlock()
			if !renewed {
				return
			}
			rc.notifyRenewed(ctx, event)

			expiresIn = grantedExpiry
			renewInterval = time.Duration(float64(grantedExpiry)*renewalFraction) * time.Second
			timer.Reset(renewInterval)

			rc.logger.Debugw("Re-registration successful",
				"did", reg.DID,
				"granted_expiry", grantedExpiry,
				"next_renewal_in", renewInterval)
		}
	}
}

// contextWithTimeout creates a context with the given timeout, but respects
// the parent context's deadline if it is sooner. This follows the LiveKit
// pattern of never extending beyond the caller's deadline.
func contextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(parent, timeout)
}

// normalizeUser strips the "+" prefix from a DID for the SIP URI user part.
// Some registrars reject "+" in the userinfo field.
func normalizeUser(did string) string {
	return strings.TrimPrefix(did, "+")
}

func (rc *RegistrationClient) markRenewalFailed(
	ctx context.Context,
	active *activeRegistration,
	err error,
	nextRetryAt time.Time,
) (bool, bool) {
	registrationError := registrationErrorFrom(err)
	retryCount := 0
	expired := false
	registrationCurrent := false
	var event RegistrationEvent

	rc.mu.Lock()
	if currentActive, ok := rc.registrations[active.reg.DID]; ok && currentActive == active {
		registrationCurrent = true
		active.renewalRetryCount++
		active.lastRenewalError = err
		active.failureClass = RegistrationFailureClassRenewal
		active.failureReason = RegistrationFailureReasonRenewalFailed
		active.statusCode = registrationError.StatusCode
		active.statusText = registrationError.StatusText
		retryCount = active.renewalRetryCount
		expired = active.expired(time.Now())
		event = rc.registrationEvent(active)
		if expired {
			active.cancel()
			delete(rc.registrations, active.reg.DID)
		}
	}
	rc.mu.Unlock()
	if !registrationCurrent {
		return false, false
	}

	event.RetryCount = retryCount
	event.NextRetryAt = nextRetryAt
	event.Error = err
	event.FailureClass = RegistrationFailureClassRenewal
	event.FailureReason = RegistrationFailureReasonRenewalFailed
	event.StatusCode = registrationError.StatusCode
	event.StatusText = registrationError.StatusText

	if expired {
		rc.notifyExpired(ctx, event)
		return true, true
	}
	rc.notifyRenewalFailed(ctx, event)
	return false, true
}

func (rc *RegistrationClient) notifyRenewed(ctx context.Context, event RegistrationEvent) {
	observer := rc.registrationObserver()
	observer.RegistrationRenewed(ctx, event)
}

func (rc *RegistrationClient) notifyRenewalFailed(ctx context.Context, event RegistrationEvent) {
	observer := rc.registrationObserver()
	observer.RegistrationRenewalFailed(ctx, event)
}

func (rc *RegistrationClient) notifyExpired(ctx context.Context, event RegistrationEvent) {
	observer := rc.registrationObserver()
	observer.RegistrationExpired(ctx, event)
}

func (rc *RegistrationClient) notifyUnregisterFailed(ctx context.Context, active *activeRegistration, err error) {
	registrationError := registrationErrorFrom(err)
	event := rc.registrationEvent(active)
	event.Error = err
	event.FailureClass = RegistrationFailureClassUnregister
	event.FailureReason = RegistrationFailureReasonUnregisterFailed
	event.StatusCode = registrationError.StatusCode
	event.StatusText = registrationError.StatusText

	observer := rc.registrationObserver()
	observer.RegistrationUnregisterFailed(ctx, event)
}

func (rc *RegistrationClient) registrationObserver() RegistrationObserver {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.observer
}

func (rc *RegistrationClient) registrationEvent(active *activeRegistration) RegistrationEvent {
	return RegistrationEvent{
		DID:           active.reg.DID,
		DeploymentID:  active.reg.DeploymentID,
		AssistantID:   active.reg.AssistantID,
		Server:        active.reg.Config.Server,
		ExpiresAt:     active.expiresAt,
		GrantedExpiry: active.grantedExpirySeconds,
	}
}

func registrationErrorFrom(err error) *RegistrationError {
	var registrationError *RegistrationError
	if errors.As(err, &registrationError) {
		return registrationError
	}
	return &RegistrationError{
		Class:     RegistrationFailureClassTransient,
		Reason:    RegistrationFailureReasonRegistrarUnreachable,
		Retryable: true,
		Cause:     err,
	}
}

func registrationExpiryGrace(expiresIn uint32) time.Duration {
	if expiresIn == 0 {
		return maxRegistrationExpiryGrace
	}
	grace := time.Duration(math.Ceil(float64(expiresIn)*0.2)) * time.Second
	if grace <= 0 || grace > maxRegistrationExpiryGrace {
		return maxRegistrationExpiryGrace
	}
	return grace
}

func validateRegistrationContactAddress(listenConfig *ListenConfig, address string) error {
	if !validator.NonNil(listenConfig) || !validator.NotBlank(address) {
		return newRegistrationContactAddressError(address)
	}
	ip := net.ParseIP(strings.TrimSpace(address))
	if ip == nil || ip.IsUnspecified() {
		return newRegistrationContactAddressError(address)
	}
	if ip.IsLoopback() && !listenConfig.AllowLoopbackExternalIP {
		return newRegistrationContactAddressError(address)
	}
	return nil
}

func newRegistrationContactAddressError(address string) *RegistrationError {
	return &RegistrationError{
		Class:     RegistrationFailureClassConfig,
		Reason:    RegistrationFailureReasonInvalidContactAddress,
		Retryable: false,
		Cause:     fmt.Errorf("invalid SIP registration contact address: %s", address),
	}
}

func newRegistrationValidationError(err error) *RegistrationError {
	reason := RegistrationFailureReasonInvalidSIPConfig
	if errors.Is(err, ErrMissingDID) {
		reason = RegistrationFailureReasonMissingDID
	}
	if errors.Is(err, ErrMissingServer) {
		reason = RegistrationFailureReasonMissingSIPServer
	}
	return &RegistrationError{
		Class:     RegistrationFailureClassConfig,
		Reason:    reason,
		Retryable: false,
		Cause:     err,
	}
}

func newRegistrationTransportError(ctx context.Context, err error) *RegistrationError {
	reason := RegistrationFailureReasonTransportError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		reason = RegistrationFailureReasonRegisterTimeout
	}
	return &RegistrationError{
		Class:     RegistrationFailureClassNetwork,
		Reason:    reason,
		Retryable: true,
		Cause:     err,
	}
}

func newRegistrationAuthError(statusCode int, statusText string, err error) *RegistrationError {
	cause := err
	if err != nil && !errors.Is(err, ErrAuthFailed) {
		cause = fmt.Errorf("%w: %w", ErrAuthFailed, err)
	}
	return &RegistrationError{
		Class:      RegistrationFailureClassAuth,
		Reason:     RegistrationFailureReasonAuthFailed,
		StatusCode: statusCode,
		StatusText: statusText,
		Retryable:  false,
		Cause:      cause,
	}
}

func newRegistrationResponseError(statusCode int, statusText string) *RegistrationError {
	if isPermanentSIPResponse(statusCode) {
		return &RegistrationError{
			Class:      RegistrationFailureClassRejected,
			Reason:     RegistrationFailureReasonRegistrarRejected,
			StatusCode: statusCode,
			StatusText: statusText,
			Retryable:  false,
			Cause:      ErrPermanentFailure,
		}
	}
	return &RegistrationError{
		Class:      RegistrationFailureClassTransient,
		Reason:     RegistrationFailureReasonRegistrarUnreachable,
		StatusCode: statusCode,
		StatusText: statusText,
		Retryable:  true,
		Cause:      ErrRegistrationFailed,
	}
}

// isPermanentSIPResponse returns true for SIP response codes that indicate a
// configuration or authorization problem that will not resolve by retrying.
func isPermanentSIPResponse(code int) bool {
	if !validator.Between(code, 300, 699) {
		return false
	}
	switch code {
	case 403, // Forbidden — blocked, wrong credentials, or policy rejection
		404, // Not Found — DID/AOR unknown to registrar
		405, // Method Not Allowed — provider does not support REGISTER
		410, // Gone — resource permanently removed
		416, // Unsupported URI Scheme
		484, // Address Incomplete
		485, // Ambiguous
		604, // Does Not Exist Anywhere
		606: // Not Acceptable
		return true
	default:
		return false
	}
}

// parseContactExpires extracts the expires parameter from a Contact header value.
// Handles: <sip:user@host>;expires=3600 and <sip:user@host;expires=3600>
func parseContactExpires(contact string) uint32 {
	lower := strings.ToLower(contact)
	idx := strings.Index(lower, "expires=")
	if idx < 0 {
		return 0
	}
	val := contact[idx+8:]
	end := strings.IndexAny(val, ";, \t>")
	if end > 0 {
		val = val[:end]
	}
	parsed, err := parsePositiveUint32(val)
	if err != nil {
		return 0
	}
	return parsed
}

func parseMinExpires(resp *sip.Response) uint32 {
	if resp == nil {
		return 0
	}
	header := resp.GetHeader("Min-Expires")
	if header == nil {
		return 0
	}
	minExpires, err := parsePositiveUint32(header.Value())
	if err != nil {
		return 0
	}
	return minExpires
}

func parsePositiveUint32(value string) (uint32, error) {
	parsed, err := utils.StringToUint32(value)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid positive uint32: %q", value)
	}
	return parsed, nil
}
