// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func TestUnclearInputWatchdog_StartExpiresWhenDeadlinePasses(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	unclearInputWatchdog := NewUnclearInputWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, unclearInputWatchdog.Start("ctx-unclear", 25*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-unclear", observabilityLogPacket.ContextID)
		assert.Equal(t, "unclear-input-watchdog: deadline expired", observabilityLogPacket.Record.Message)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("unclear input watchdog did not expire")
	}

	select {
	case packet := <-pushedPackets:
		unclearInputExpiredPacket, ok := packet.(internal_type.UnclearInputExpiredPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-unclear", unclearInputExpiredPacket.ContextID)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("unclear input watchdog did not push expired packet")
	}

	assert.False(t, unclearInputWatchdog.Stop())
}

func TestUnclearInputWatchdog_ExtendResetsDeadline(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	unclearInputWatchdog := NewUnclearInputWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, unclearInputWatchdog.Start("ctx-extend", 35*time.Millisecond))
	time.Sleep(20 * time.Millisecond)
	require.True(t, unclearInputWatchdog.Extend("ctx-extend", 80*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		t.Fatalf("unclear input watchdog pushed packet before extended deadline: %+v", packet)
	case <-time.After(55 * time.Millisecond):
	}

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-extend", observabilityLogPacket.ContextID)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("unclear input watchdog did not expire after extended deadline")
	}
}

func TestUnclearInputWatchdog_StopCancelsExpiration(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	unclearInputWatchdog := NewUnclearInputWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, unclearInputWatchdog.Start("ctx-stop", 40*time.Millisecond))
	require.True(t, unclearInputWatchdog.Stop())

	select {
	case packet := <-pushedPackets:
		t.Fatalf("unclear input watchdog pushed packet after stop: %+v", packet)
	case <-time.After(90 * time.Millisecond):
	}
}

func TestUnclearInputWatchdog_ExtendIgnoresInactiveAndDifferentContext(t *testing.T) {
	unclearInputWatchdog := NewUnclearInputWatchdog()

	require.False(t, unclearInputWatchdog.Extend("ctx-inactive", time.Second))

	require.True(t, unclearInputWatchdog.Start("ctx-active", 40*time.Millisecond))
	require.False(t, unclearInputWatchdog.Extend("ctx-other", time.Second))
	require.True(t, unclearInputWatchdog.Stop())
}

func TestUnclearInputWatchdog_ConstructorPushesInitializationInfo(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 1)

	NewUnclearInputWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))

	packet := <-pushedPackets
	observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
	require.True(t, ok)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, observabilityLogPacket.Scope)
	assert.Equal(t, observability.LevelInfo, observabilityLogPacket.Record.Level)
	assert.Equal(t, "unclear-input-watchdog: initialization completed", observabilityLogPacket.Record.Message)
	assert.Equal(t, "unclear_input", observabilityLogPacket.Record.Attributes["watchdog"])
}
