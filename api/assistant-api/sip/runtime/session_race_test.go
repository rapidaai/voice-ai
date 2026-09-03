// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSessionConfig() *Config {
	return &Config{
		Server:            "127.0.0.1",
		Port:              5060,
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10020,
	}
}

func newInboundTestSession(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession(context.Background(),
		WithSessionConfig(testSessionConfig()),
		WithSessionDirection(CallDirectionInbound),
	)
	require.NoError(t, err)
	return s
}

func TestSessionSetRTPHandlerStopsReplacedHandler(t *testing.T) {
	t.Parallel()
	s := newInboundTestSession(t)
	first := newTestRTPHandler()
	second := newTestRTPHandler()

	s.SetRTPHandler(first)
	s.SetRTPHandler(first)
	assert.True(t, first.running.Load())

	s.SetRTPHandler(second)
	assert.False(t, first.running.Load())
	assert.True(t, second.running.Load())

	s.End()
	assert.False(t, second.running.Load())
}

func TestSessionConcurrentEndAndSetState(t *testing.T) {
	t.Parallel()

	s := newInboundTestSession(t)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 500; j++ {
				s.SetState(CallStateRinging)
				s.SetState(CallStateConnected)
			}
		}()
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 200; j++ {
				s.End()
			}
		}()
	}

	close(start)
	wg.Wait()

	require.True(t, s.IsEnded())
}

func TestSessionConcurrentErrorSendAndEnd(t *testing.T) {
	t.Parallel()

	s := newInboundTestSession(t)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for j := 0; j < 1000; j++ {
				s.SendError(errors.New("test error"))
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 50; i++ {
			s.End()
		}
	}()

	close(start)
	wg.Wait()

	require.True(t, s.IsEnded())
}

func TestSessionConcurrentMetadataAndLifecycle(t *testing.T) {
	t.Parallel()

	s := newInboundTestSession(t)

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for j := 0; j < 600; j++ {
				s.SetMetadata("k", worker*1000+j)
				_, _ = s.GetMetadata("k")
				_ = s.GetState()
				_ = s.GetInfo()
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 100; i++ {
			s.SetState(CallStateRinging)
			s.SetState(CallStateConnected)
			time.Sleep(50 * time.Microsecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		time.Sleep(2 * time.Millisecond)
		s.End()
	}()

	close(start)
	wg.Wait()

	require.True(t, s.IsEnded())
}
