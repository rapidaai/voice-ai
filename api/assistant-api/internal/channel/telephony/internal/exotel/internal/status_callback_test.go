// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestNewStatusCallback_MissingStatusUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{"CallSid": "call-id"}, "")

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackStatusMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackStatusMissing)
	}
}

func TestNewStatusCallback_MapsStatusToEvent(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{
		"CallSid": "call-id",
		"Status":  "in-progress",
	}, "")

	if err != nil {
		t.Fatalf("NewStatusCallback() error = %v", err)
	}
	if callback.Event != "answered" {
		t.Fatalf("Event=%q want answered", callback.Event)
	}
	if callback.Status != "in-progress" {
		t.Fatalf("Status=%q want in-progress", callback.Status)
	}
}

func TestNewStatusCallback_AllowsUnknownStatus(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{
		"CallSid": "call-id",
		"Status":  "provider-specific",
	}, "")

	if err != nil {
		t.Fatalf("NewStatusCallback() error = %v", err)
	}
	if callback.Event.String() != "provider-specific" {
		t.Fatalf("Event=%q want provider-specific", callback.Event)
	}
}

func TestStatusInfo_FailedStatusKeepsStatusFailureReason(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{
		"CallSid": "call-id",
		"Status":  "no-answer",
	}, "")
	if err != nil {
		t.Fatalf("NewStatusCallback() error = %v", err)
	}

	statusInfo := callback.StatusInfo()

	if statusInfo.Event.String() != "completed" {
		t.Fatalf("Event=%q want completed", statusInfo.Event)
	}
	if statusInfo.Error == nil {
		t.Fatal("Error=nil want failed status error")
	}
	if statusInfo.Error.Reason != "no-answer" {
		t.Fatalf("Reason=%q want no-answer", statusInfo.Error.Reason)
	}
}
