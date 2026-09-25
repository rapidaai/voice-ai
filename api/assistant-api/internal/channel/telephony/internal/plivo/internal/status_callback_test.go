// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestNewStatusCallback_MissingStatusUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{"CallUUID": "abc"}, "")

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackStatusMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackStatusMissing)
	}
}

func TestNewStatusCallback_ParsesFields(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{
		"CallUUID":    "call-uuid-123",
		"CallStatus":  "completed",
		"Duration":    "42",
		"HangupCause": "NORMAL_CLEARING",
	}, "raw=payload")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callback.ChannelUUID != "call-uuid-123" {
		t.Errorf("ChannelUUID=%s want call-uuid-123", callback.ChannelUUID)
	}
	if callback.Event != "completed" {
		t.Errorf("Event=%s want completed", callback.Event)
	}
	if callback.Duration == nil {
		t.Fatal("expected duration, got nil")
	}
	if callback.RawPayload != "raw=payload" {
		t.Errorf("RawPayload=%s want raw=payload", callback.RawPayload)
	}
}

func TestStatusInfo_CleanHangupCompletesWithoutError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{
		"CallUUID":    "call-uuid-123",
		"CallStatus":  "completed",
		"HangupCause": "NORMAL_CLEARING",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info := callback.StatusInfo()
	if !info.Completed {
		t.Error("expected completed=true for NORMAL_CLEARING hangup")
	}
	if info.Error != nil {
		t.Errorf("expected no error, got %+v", info.Error)
	}
}

func TestStatusInfo_FailureStates(t *testing.T) {
	tests := []struct {
		name       string
		details    utils.Option
		wantFailed bool
		wantReason string
	}{
		{
			name:       "busy status",
			details:    utils.Option{"CallUUID": "u", "CallStatus": "busy"},
			wantFailed: true,
			wantReason: "busy",
		},
		{
			name:       "no-answer status",
			details:    utils.Option{"CallUUID": "u", "CallStatus": "no-answer"},
			wantFailed: true,
			wantReason: "no-answer",
		},
		{
			name:       "abnormal hangup cause",
			details:    utils.Option{"CallUUID": "u", "CallStatus": "hangup", "HangupCause": "USER_BUSY"},
			wantFailed: true,
			wantReason: "USER_BUSY",
		},
		{
			name:       "normal clearing not failed",
			details:    utils.Option{"CallUUID": "u", "CallStatus": "hangup", "HangupCause": "NORMAL_CLEARING"},
			wantFailed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callback, err := NewStatusCallback(tt.details, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := callback.Failed(); got != tt.wantFailed {
				t.Errorf("Failed()=%t want %t", got, tt.wantFailed)
			}
			info := callback.StatusInfo()
			if tt.wantFailed {
				if info.Error == nil {
					t.Fatal("expected StatusError, got nil")
				}
				if info.Error.Reason != tt.wantReason {
					t.Errorf("reason=%s want %s", info.Error.Reason, tt.wantReason)
				}
			} else if info.Error != nil {
				t.Errorf("expected no error, got %+v", info.Error)
			}
		})
	}
}

func TestStatusInfo_NormalHangupCausesAreNotFailures(t *testing.T) {
	// Plivo documents these as normal terminations, so they must complete cleanly
	// rather than be classified as failures. The XML case is the important one: the
	// SIP-style HangupCause is not itself normal, but the cause name marks it clean.
	cases := []struct {
		name    string
		details utils.Option
	}{
		{"sip normal clearing", utils.Option{"CallUUID": "u", "CallStatus": "completed", "HangupCause": "NORMAL_CLEARING"}},
		{"normal hangup name", utils.Option{"CallUUID": "u", "CallStatus": "completed", "HangupCauseName": "Normal Hangup"}},
		{"end of xml instructions", utils.Option{"CallUUID": "u", "CallStatus": "completed", "HangupCause": "END_OF_SESSION", "HangupCauseName": "End Of XML Instructions"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			callback, err := NewStatusCallback(tt.details, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if callback.Failed() {
				t.Error("Failed()=true, want false for a documented normal cause")
			}
			info := callback.StatusInfo()
			if !info.Completed {
				t.Error("Completed=false, want true for a documented normal cause")
			}
			if info.Error != nil {
				t.Errorf("expected no StatusError, got %+v", info.Error)
			}
		})
	}
}
