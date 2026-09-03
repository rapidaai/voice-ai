// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emiago/sipgo/sip"
)

func TestRegistrationValidationErrorClassifiesMissingFields(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedReason RegistrationFailureReason
	}{
		{
			name:           "missing DID",
			err:            ErrMissingDID,
			expectedReason: RegistrationFailureReasonMissingDID,
		},
		{
			name:           "missing server",
			err:            ErrMissingServer,
			expectedReason: RegistrationFailureReasonMissingSIPServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrationError := newRegistrationValidationError(tt.err)
			if registrationError.Class != RegistrationFailureClassConfig {
				t.Fatalf("expected class=%s, got %s", RegistrationFailureClassConfig, registrationError.Class)
			}
			if registrationError.Reason != tt.expectedReason {
				t.Fatalf("expected reason=%s, got %s", tt.expectedReason, registrationError.Reason)
			}
			if registrationError.Retryable {
				t.Fatal("validation errors should not be retryable")
			}
			if !errors.Is(registrationError, tt.err) {
				t.Fatalf("expected errors.Is to match %v", tt.err)
			}
		})
	}
}

func TestRegistrationResponseErrorClassifiesRegistrarResponses(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		statusText     string
		expectedClass  RegistrationFailureClass
		expectedReason RegistrationFailureReason
		expectedCause  error
		retryable      bool
	}{
		{
			name:           "permanent rejection",
			statusCode:     403,
			statusText:     "Forbidden",
			expectedClass:  RegistrationFailureClassRejected,
			expectedReason: RegistrationFailureReasonRegistrarRejected,
			expectedCause:  ErrPermanentFailure,
			retryable:      false,
		},
		{
			name:           "transient registrar failure",
			statusCode:     503,
			statusText:     "Service Unavailable",
			expectedClass:  RegistrationFailureClassTransient,
			expectedReason: RegistrationFailureReasonRegistrarUnreachable,
			expectedCause:  ErrRegistrationFailed,
			retryable:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrationError := newRegistrationResponseError(tt.statusCode, tt.statusText)
			if registrationError.Class != tt.expectedClass {
				t.Fatalf("expected class=%s, got %s", tt.expectedClass, registrationError.Class)
			}
			if registrationError.Reason != tt.expectedReason {
				t.Fatalf("expected reason=%s, got %s", tt.expectedReason, registrationError.Reason)
			}
			if registrationError.StatusCode != tt.statusCode {
				t.Fatalf("expected status_code=%d, got %d", tt.statusCode, registrationError.StatusCode)
			}
			if registrationError.StatusText != tt.statusText {
				t.Fatalf("expected status_text=%s, got %s", tt.statusText, registrationError.StatusText)
			}
			if registrationError.Retryable != tt.retryable {
				t.Fatalf("expected retryable=%v, got %v", tt.retryable, registrationError.Retryable)
			}
			if !errors.Is(registrationError, tt.expectedCause) {
				t.Fatalf("expected errors.Is to match %v", tt.expectedCause)
			}
		})
	}
}

func TestRegistrationAuthErrorPreservesSentinel(t *testing.T) {
	registrationError := newRegistrationAuthError(0, "", digestAuthTestError{})
	if registrationError.Class != RegistrationFailureClassAuth {
		t.Fatalf("expected class=%s, got %s", RegistrationFailureClassAuth, registrationError.Class)
	}
	if registrationError.Reason != RegistrationFailureReasonAuthFailed {
		t.Fatalf("expected reason=%s, got %s", RegistrationFailureReasonAuthFailed, registrationError.Reason)
	}
	if !errors.Is(registrationError, ErrAuthFailed) {
		t.Fatal("expected auth registration error to match ErrAuthFailed")
	}
}

func TestRegistrationTransportErrorClassifiesTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	registrationError := newRegistrationTransportError(ctx, context.DeadlineExceeded)
	if registrationError.Class != RegistrationFailureClassNetwork {
		t.Fatalf("expected class=%s, got %s", RegistrationFailureClassNetwork, registrationError.Class)
	}
	if registrationError.Reason != RegistrationFailureReasonRegisterTimeout {
		t.Fatalf("expected reason=%s, got %s", RegistrationFailureReasonRegisterTimeout, registrationError.Reason)
	}
	if !registrationError.Retryable {
		t.Fatal("transport timeout should be retryable")
	}
}

func TestRegistrationSnapshotReflectsRenewalHealth(t *testing.T) {
	now := time.Now()
	client := &RegistrationClient{
		registrations: map[string]*activeRegistration{
			"+15551234567": {
				reg: &Registration{
					DID: "+15551234567",
				},
				expiresAt:            now.Add(time.Minute),
				grantedExpirySeconds: 120,
				renewalRetryCount:    2,
				lastRenewalError:     ErrRegistrationFailed,
				failureClass:         RegistrationFailureClassRenewal,
				failureReason:        RegistrationFailureReasonRenewalFailed,
			},
		},
	}

	snapshot := client.Snapshot("+15551234567")
	if !snapshot.Active {
		t.Fatal("expected registration to remain active before expiry")
	}
	if snapshot.Healthy {
		t.Fatal("expected registration with renewal error to be unhealthy")
	}
	if snapshot.RenewalRetryCount != 2 {
		t.Fatalf("expected retry count 2, got %d", snapshot.RenewalRetryCount)
	}
	if snapshot.FailureClass != RegistrationFailureClassRenewal {
		t.Fatalf("expected failure class=%s, got %s", RegistrationFailureClassRenewal, snapshot.FailureClass)
	}
}

func TestRegistrationSnapshotMarksExpiredInactive(t *testing.T) {
	client := &RegistrationClient{
		registrations: map[string]*activeRegistration{
			"+15551234567": {
				reg: &Registration{
					DID: "+15551234567",
				},
				expiresAt:            time.Now().Add(-2 * maxRegistrationExpiryGrace),
				grantedExpirySeconds: 120,
			},
		},
	}

	snapshot := client.Snapshot("+15551234567")
	if snapshot.Active {
		t.Fatal("expected expired registration to be inactive")
	}
	if snapshot.Healthy {
		t.Fatal("expected expired registration to be unhealthy")
	}
}

func TestRegistrationSnapshotUsesDynamicExpiryGrace(t *testing.T) {
	client := &RegistrationClient{
		registrations: map[string]*activeRegistration{
			"+15551234567": {
				reg: &Registration{
					DID: "+15551234567",
				},
				expiresAt:            time.Now().Add(-3 * time.Second),
				grantedExpirySeconds: 10,
			},
		},
	}

	snapshot := client.Snapshot("+15551234567")
	if snapshot.Active {
		t.Fatal("expected short-expiry registration to become inactive after dynamic grace")
	}
	if snapshot.Healthy {
		t.Fatal("expected short-expiry registration to be unhealthy after dynamic grace")
	}
}

func TestMarkRenewalFailedBeforeExpiryEmitsFailure(t *testing.T) {
	observer := &registrationObserverSpy{}
	active := &activeRegistration{
		reg: &Registration{
			DID:          "+15551234567",
			DeploymentID: 101,
			AssistantID:  201,
			Config:       &Config{Server: "registrar.example.com"},
		},
		expiresAt:            time.Now().Add(time.Minute),
		grantedExpirySeconds: 120,
	}
	client := &RegistrationClient{
		observer: observer,
		registrations: map[string]*activeRegistration{
			"+15551234567": active,
		},
	}

	expired, current := client.markRenewalFailed(context.Background(), active, ErrRegistrationFailed, time.Now().Add(renewRetryInterval))
	if !current {
		t.Fatal("expected renewal failure to target current registration")
	}
	if expired {
		t.Fatal("expected renewal failure before expiry to remain active")
	}
	if len(observer.renewalFailures) != 1 {
		t.Fatalf("expected one renewal failure event, got %d", len(observer.renewalFailures))
	}
	if !client.IsRegistered("+15551234567") {
		t.Fatal("expected registration to remain active")
	}
	if active.renewalRetryCount != 1 {
		t.Fatalf("expected retry count 1, got %d", active.renewalRetryCount)
	}
}

func TestMarkRenewalFailedAfterGraceExpiresRegistration(t *testing.T) {
	observer := &registrationObserverSpy{}
	active := &activeRegistration{
		reg: &Registration{
			DID:          "+15551234567",
			DeploymentID: 101,
			AssistantID:  201,
			Config:       &Config{Server: "registrar.example.com"},
		},
		cancel:               func() {},
		expiresAt:            time.Now().Add(-2 * maxRegistrationExpiryGrace),
		grantedExpirySeconds: 120,
	}
	client := &RegistrationClient{
		observer: observer,
		registrations: map[string]*activeRegistration{
			"+15551234567": active,
		},
	}

	expired, current := client.markRenewalFailed(context.Background(), active, ErrRegistrationFailed, time.Now().Add(renewRetryInterval))
	if !current {
		t.Fatal("expected renewal failure to target current registration")
	}
	if !expired {
		t.Fatal("expected renewal failure after grace to expire registration")
	}
	if len(observer.expired) != 1 {
		t.Fatalf("expected one expired event, got %d", len(observer.expired))
	}
	if client.IsRegistered("+15551234567") {
		t.Fatal("expected expired registration to be removed")
	}
}

func TestMarkRenewalFailedIgnoresStaleRegistration(t *testing.T) {
	observer := &registrationObserverSpy{}
	stale := &activeRegistration{
		reg: &Registration{
			DID:          "+15551234567",
			DeploymentID: 101,
			AssistantID:  201,
			Config:       &Config{Server: "registrar.example.com"},
		},
		cancel:               func() {},
		expiresAt:            time.Now().Add(-2 * maxRegistrationExpiryGrace),
		grantedExpirySeconds: 120,
	}
	currentRegistration := &activeRegistration{
		reg: &Registration{
			DID:          "+15551234567",
			DeploymentID: 101,
			AssistantID:  201,
			Config:       &Config{Server: "registrar.example.com"},
		},
		cancel:               func() {},
		expiresAt:            time.Now().Add(time.Minute),
		grantedExpirySeconds: 120,
	}
	client := &RegistrationClient{
		observer: observer,
		registrations: map[string]*activeRegistration{
			"+15551234567": currentRegistration,
		},
	}

	expired, registrationCurrent := client.markRenewalFailed(context.Background(), stale, ErrRegistrationFailed, time.Now().Add(renewRetryInterval))
	if registrationCurrent {
		t.Fatal("expected stale renewal failure to be ignored")
	}
	if expired {
		t.Fatal("stale registration should not expire current registration")
	}
	if len(observer.renewalFailures) != 0 {
		t.Fatalf("expected no stale renewal failure events, got %d", len(observer.renewalFailures))
	}
	if len(observer.expired) != 0 {
		t.Fatalf("expected no stale expired events, got %d", len(observer.expired))
	}
	if client.Snapshot("+15551234567").LastRenewalError != nil {
		t.Fatal("stale renewal failure should not mutate current registration")
	}
}

func TestValidateRegistrationContactAddress(t *testing.T) {
	tests := []struct {
		name        string
		config      *ListenConfig
		address     string
		expectError bool
	}{
		{
			name:        "empty",
			config:      &ListenConfig{},
			address:     "",
			expectError: true,
		},
		{
			name:        "unspecified",
			config:      &ListenConfig{},
			address:     "0.0.0.0",
			expectError: true,
		},
		{
			name:        "loopback blocked",
			config:      &ListenConfig{},
			address:     "127.0.0.1",
			expectError: true,
		},
		{
			name:        "loopback allowed",
			config:      &ListenConfig{AllowLoopbackExternalIP: true},
			address:     "127.0.0.1",
			expectError: false,
		},
		{
			name:        "public ip",
			config:      &ListenConfig{},
			address:     "203.0.113.10",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistrationContactAddress(tt.config, tt.address)
			if tt.expectError && err == nil {
				t.Fatal("expected validation error")
			}
			if tt.expectError {
				var registrationError *RegistrationError
				if !errors.As(err, &registrationError) {
					t.Fatalf("expected RegistrationError, got %T", err)
				}
				if registrationError.Reason != RegistrationFailureReasonInvalidContactAddress {
					t.Fatalf("expected reason=%s, got %s", RegistrationFailureReasonInvalidContactAddress, registrationError.Reason)
				}
			}
			if !tt.expectError && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestParseMinExpires(t *testing.T) {
	resp := sip.NewResponse(423, "Interval Too Brief")
	resp.AppendHeader(sip.NewHeader("Min-Expires", "120"))

	if minExpires := parseMinExpires(resp); minExpires != 120 {
		t.Fatalf("expected min expires 120, got %d", minExpires)
	}
}

func TestParseRegistrationExpiryRejectsUint32Overflow(t *testing.T) {
	if expires := parseContactExpires("<sip:user@example.com>;expires=4294967296"); expires != 0 {
		t.Fatalf("expected overflowing contact expires to be ignored, got %d", expires)
	}

	resp := sip.NewResponse(423, "Interval Too Brief")
	resp.AppendHeader(sip.NewHeader("Min-Expires", "4294967296"))
	if minExpires := parseMinExpires(resp); minExpires != 0 {
		t.Fatalf("expected overflowing min expires to be ignored, got %d", minExpires)
	}
}

type digestAuthTestError struct{}

func (digestAuthTestError) Error() string { return "digest calculation failed" }

type registrationObserverSpy struct {
	renewed         []RegistrationEvent
	renewalFailures []RegistrationEvent
	expired         []RegistrationEvent
	unregisterFails []RegistrationEvent
}

func (s *registrationObserverSpy) RegistrationRenewed(_ context.Context, event RegistrationEvent) {
	s.renewed = append(s.renewed, event)
}

func (s *registrationObserverSpy) RegistrationRenewalFailed(_ context.Context, event RegistrationEvent) {
	s.renewalFailures = append(s.renewalFailures, event)
}

func (s *registrationObserverSpy) RegistrationExpired(_ context.Context, event RegistrationEvent) {
	s.expired = append(s.expired, event)
}

func (s *registrationObserverSpy) RegistrationUnregisterFailed(_ context.Context, event RegistrationEvent) {
	s.unregisterFails = append(s.unregisterFails, event)
}
