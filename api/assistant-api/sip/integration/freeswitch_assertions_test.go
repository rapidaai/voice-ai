// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package sip_integration

import (
	"sync"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func waitForCallState(t *testing.T, session *sip_runtime.Session, expected sip_runtime.CallState, timeout time.Duration) {
	t.Helper()
	require.Eventuallyf(t, func() bool {
		return session.GetInfo().State == expected
	}, timeout, 50*time.Millisecond, "expected call %s to reach state %s, current state is %s", session.GetCallID(), expected, session.GetInfo().State)
}

func waitForTerminalCallState(t *testing.T, session *sip_runtime.Session, timeout time.Duration) {
	t.Helper()
	require.Eventuallyf(t, func() bool {
		return session.GetInfo().State.IsTerminal()
	}, timeout, 50*time.Millisecond, "expected call %s to reach terminal state, current state is %s", session.GetCallID(), session.GetInfo().State)
}

type integrationOutboundStatusRecorder struct {
	mu      sync.Mutex
	updates []internal_type.ProviderCallStatusUpdate
}

func newIntegrationOutboundStatusRecorder() *integrationOutboundStatusRecorder {
	return &integrationOutboundStatusRecorder{}
}

func (r *integrationOutboundStatusRecorder) Record(update internal_type.ProviderCallStatusUpdate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates = append(r.updates, update)
}

func (r *integrationOutboundStatusRecorder) LastStatus(t *testing.T, callStatus string) internal_type.ProviderCallStatusUpdate {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()

	for index := len(r.updates) - 1; index >= 0; index-- {
		if r.updates[index].CallStatus == callStatus {
			return r.updates[index]
		}
	}
	require.Failf(t, "missing provider status update", "call status %q was not recorded in %#v", callStatus, r.updates)
	return internal_type.ProviderCallStatusUpdate{}
}

func receiveInboundSession(t *testing.T, sessions <-chan *sip_runtime.Session, timeout time.Duration) *sip_runtime.Session {
	t.Helper()
	select {
	case session := <-sessions:
		require.NotNil(t, session)
		return session
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for inbound session")
		return nil
	}
}
