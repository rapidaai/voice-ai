// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/require"
)

type failingTalker struct {
	internal_type.Talking
	err error
}

func (talker failingTalker) Talk(context.Context, *types.Authentication) error {
	return talker.err
}

func TestSIPPreparedCallRuntimeReturnsTalkerError(t *testing.T) {
	expected := errors.New("talker failed")
	runtime := &sipPreparedCallRuntime{
		talkContext: context.Background(),
		talker:      failingTalker{err: expected},
	}

	require.ErrorIs(t, runtime.runTalker(), expected)
}

type preparedRuntimeTestRuntime struct {
	started chan struct{}
	closed  atomic.Bool
}

func (runtime *preparedRuntimeTestRuntime) Start(context.Context) error {
	close(runtime.started)
	return nil
}

func (runtime *preparedRuntimeTestRuntime) Close(context.Context) {
	runtime.closed.Store(true)
}

type preparedRuntimeTestObserver struct {
	closed atomic.Bool
}

func (observer *preparedRuntimeTestObserver) Record(context.Context, observability.Scope, ...observability.Record) error {
	return nil
}

func (observer *preparedRuntimeTestObserver) AddCollectors(...observability.Collector) error {
	return nil
}

func (observer *preparedRuntimeTestObserver) Close(context.Context) error {
	observer.closed.Store(true)
	return nil
}

func TestStartPreparedSessionConsumesPreparedRuntime(t *testing.T) {
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionOutbound),
		sip_runtime.WithSessionCallID("prepared-call"),
	)
	require.NoError(t, err)

	runtime := &preparedRuntimeTestRuntime{started: make(chan struct{})}
	observer := &preparedRuntimeTestObserver{}
	dispatcher := New(
		WithLogger(newPipelineTestLogger(t)),
		WithTransferServer(&fakeTransferServer{}),
	)
	dispatcher.preparedSessions["prepared-call"] = &preparedSession{
		stage: SessionEstablishedPipeline{
			ID:        "prepared-call",
			Session:   session,
			Direction: sip_runtime.CallDirectionOutbound,
		},
		setup: &CallSetupResult{
			AssistantID:    1,
			ConversationID: 2,
			CallContext: &callcontext.CallContext{
				ContextID:    "ctx-prepared-call",
				CallerNumber: "+15551234567",
				FromNumber:   "+15557654321",
			},
		},
		observer: observer,
		runtime:  runtime,
	}

	err = dispatcher.StartPreparedSession(context.Background(), SessionEstablishedPipeline{ID: "prepared-call"})
	require.NoError(t, err)

	select {
	case <-runtime.started:
	case <-time.After(time.Second):
		t.Fatal("prepared runtime did not start")
	}
	require.Eventually(t, observer.closed.Load, time.Second, 10*time.Millisecond)

	dispatcher.preparedMu.Lock()
	_, exists := dispatcher.preparedSessions["prepared-call"]
	dispatcher.preparedMu.Unlock()
	require.False(t, exists)
	require.False(t, runtime.closed.Load())
}

func TestStartPreparedSessionRequiresPreparedRuntime(t *testing.T) {
	dispatcher := New(WithLogger(newPipelineTestLogger(t)))

	err := dispatcher.StartPreparedSession(context.Background(), SessionEstablishedPipeline{ID: "missing-call"})

	require.ErrorContains(t, err, "prepared SIP session not found")
}
