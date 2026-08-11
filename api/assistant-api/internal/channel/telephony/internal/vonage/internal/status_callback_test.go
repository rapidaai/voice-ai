// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vonage

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestNewStatusCallback_MissingStatusUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{"uuid": "call-id"}, "")

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackStatusMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackStatusMissing)
	}
}

func TestVonageWebSocketEvent_UnmarshalsTypedEvent(t *testing.T) {
	var event VonageWebSocketEvent

	if err := json.Unmarshal([]byte(`{"event":"websocket:connected"}`), &event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Event != EventTypeWebSocketConnected {
		t.Fatalf("event=%q want %q", event.Event, EventTypeWebSocketConnected)
	}
}
