// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"fmt"
	"testing"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
)

func TestRegistrationStatusUpdateFromError_PreservesTypedClassification(t *testing.T) {
	m := &manager{instanceID: "test-instance"}
	err := fmt.Errorf("%w: %w", sip_runtime.ErrRegistrationFailed, &sip_runtime.RegistrationError{
		Class:      RegistrationFailureClassRejected,
		Reason:     RegistrationFailureReasonRegistrarRejected,
		StatusCode: 403,
		StatusText: "Forbidden",
		Cause:      sip_runtime.ErrPermanentFailure,
	})

	update := m.registrationStatusUpdateFromError(err)
	if update.Status != StatusRejected {
		t.Fatalf("expected status=%s, got %s", StatusRejected, update.Status)
	}
	if update.FailureClass != RegistrationFailureClassRejected {
		t.Fatalf("expected class=%s, got %s", RegistrationFailureClassRejected, update.FailureClass)
	}
	if update.FailureReason != RegistrationFailureReasonRegistrarRejected {
		t.Fatalf("expected reason=%s, got %s", RegistrationFailureReasonRegistrarRejected, update.FailureReason)
	}
	if update.ResponseCode != 403 || update.ResponseText != "Forbidden" {
		t.Fatalf("unexpected response metadata: %d %s", update.ResponseCode, update.ResponseText)
	}
	if update.OwnerInstance != "test-instance" {
		t.Fatalf("expected owner instance to be preserved, got %s", update.OwnerInstance)
	}
}

func TestRegistrationStatusUpdateFromError_ClassifiesLegacyAuthError(t *testing.T) {
	m := &manager{instanceID: "test-instance"}

	update := m.registrationStatusUpdateFromError(sip_runtime.ErrAuthFailed)
	if update.Status != StatusFailed {
		t.Fatalf("expected status=%s, got %s", StatusFailed, update.Status)
	}
	if update.FailureClass != RegistrationFailureClassAuth {
		t.Fatalf("expected class=%s, got %s", RegistrationFailureClassAuth, update.FailureClass)
	}
	if update.FailureReason != RegistrationFailureReasonAuthFailed {
		t.Fatalf("expected reason=%s, got %s", RegistrationFailureReasonAuthFailed, update.FailureReason)
	}
}
