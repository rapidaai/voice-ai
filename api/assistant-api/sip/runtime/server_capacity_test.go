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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedCallAdmissionClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedCallAdmissionClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedCallAdmissionClock) Add(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

func TestServerCallAdmissionUnlimitedWhenUnset(t *testing.T) {
	server := &Server{}

	require.True(t, server.tryAcquireCallSlot())
	require.True(t, server.tryAcquireCallSlot())
	server.releaseCallSlot()

	assert.Equal(t, int64(0), server.activeCallAdmissions.Load())
}

func TestServerCallAdmissionRejectsAtLimitConcurrently(t *testing.T) {
	server := &Server{maxConcurrentCalls: 1}
	start := make(chan struct{})
	var accepted atomic.Int64
	var waitGroup sync.WaitGroup

	for range 16 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			if server.tryAcquireCallSlot() {
				accepted.Add(1)
			}
		}()
	}

	close(start)
	waitGroup.Wait()

	assert.Equal(t, int64(1), accepted.Load())
	assert.Equal(t, int64(1), server.activeCallAdmissions.Load())
}

func TestServerCallAdmissionReleasesCapacity(t *testing.T) {
	server := &Server{maxConcurrentCalls: 1}

	require.True(t, server.tryAcquireCallSlot())
	require.False(t, server.tryAcquireCallSlot())

	server.releaseCallSlot()

	require.True(t, server.tryAcquireCallSlot())
	assert.Equal(t, int64(1), server.activeCallAdmissions.Load())
}

func TestServerCallAdmissionCapacityRejectDoesNotConsumeSetupRate(t *testing.T) {
	server := &Server{
		maxConcurrentCalls:  1,
		callAdmissionCPS:    1,
		callAdmissionBurst:  1,
		callAdmissionClock:  &fixedCallAdmissionClock{now: time.Unix(100, 0)},
		callAdmissionLast:   time.Unix(100, 0),
		callAdmissionTokens: 1,
	}
	server.activeCallAdmissions.Store(1)

	releaseAdmission, err := server.acquireNewCallAdmission()

	require.Nil(t, releaseAdmission)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSIPCallCapacityExceeded))
	assert.Equal(t, float64(1), server.callAdmissionTokens)
	assert.Equal(t, int64(1), server.activeCallAdmissions.Load())
	assert.Equal(t, uint64(1), server.CallAdmissionStats().CapacityRejections)
	assert.Zero(t, server.CallAdmissionStats().RateRejections)
}

func TestServerCallAdmissionRateRejectReleasesCapacity(t *testing.T) {
	server := &Server{
		maxConcurrentCalls:  1,
		callAdmissionCPS:    1,
		callAdmissionBurst:  1,
		callAdmissionClock:  &fixedCallAdmissionClock{now: time.Unix(100, 0)},
		callAdmissionLast:   time.Unix(100, 0),
		callAdmissionTokens: 0,
	}

	releaseAdmission, err := server.acquireNewCallAdmission()

	require.Nil(t, releaseAdmission)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSIPCallRateExceeded))
	assert.Equal(t, int64(0), server.activeCallAdmissions.Load())
	assert.Zero(t, server.CallAdmissionStats().CapacityRejections)
	assert.Equal(t, uint64(1), server.CallAdmissionStats().RateRejections)
}

func TestServerRegisterAdmittedSessionReleasesCapacityOnEnd(t *testing.T) {
	server := newServerForCommandTests(t)
	server.maxConcurrentCalls = 1
	require.True(t, server.tryAcquireCallSlot())
	session := newTestSession(t, "capacity-release", CallDirectionInbound)

	server.registerSessionWithAdmission(session, session.GetCallID(), true)
	session.End()

	assert.Equal(t, int64(0), server.activeCallAdmissions.Load())
}

func TestInboundInviteRejectsWhenCallCapacityFull(t *testing.T) {
	server := newServerForCommandTests(t)
	server.maxConcurrentCalls = 1
	server.activeCallAdmissions.Store(1)
	request := newInboundInviteRequest("inbound-capacity-full")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 100, transaction.responses[0].StatusCode)
	assert.Equal(t, 503, transaction.lastStatus())
	retryAfter := transaction.responses[len(transaction.responses)-1].GetHeader(sipHeaderRetryAfter)
	require.NotNil(t, retryAfter)
	assert.Equal(t, "1", retryAfter.Value())
	assert.Equal(t, int64(1), server.activeCallAdmissions.Load())
	assert.Equal(t, uint64(1), server.CallAdmissionStats().CapacityRejections)
	_, exists := server.GetSession("inbound-capacity-full")
	assert.False(t, exists)
}

func TestMakeCallRejectsWhenCallCapacityFull(t *testing.T) {
	server := &Server{maxConcurrentCalls: 1}
	server.activeCallAdmissions.Store(1)
	server.state.Store(int32(ServerStateRunning))

	session, err := server.MakeCall(
		context.Background(),
		testOutboundConfig(),
		"+15551234567",
		"+15557654321",
		MakeCallOptions{},
	)

	require.Nil(t, session)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSIPCallCapacityExceeded))
	assert.Equal(t, int64(1), server.activeCallAdmissions.Load())
	assert.Equal(t, uint64(1), server.CallAdmissionStats().CapacityRejections)
}

func TestMakeTransferBridgeCallRejectsWhenCallCapacityFull(t *testing.T) {
	server := &Server{maxConcurrentCalls: 1}
	server.activeCallAdmissions.Store(1)
	server.state.Store(int32(ServerStateRunning))

	session, err := server.MakeTransferBridgeCall(
		context.Background(),
		testOutboundConfig(),
		"+15551234567",
		"+15557654321",
		TransferBridgeCallOptions{ParentCallID: "transfer-parent", Attempt: 1, TotalAttempts: 1},
	)

	require.Nil(t, session)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSIPCallCapacityExceeded))
	assert.Equal(t, uint64(1), server.CallAdmissionStats().CapacityRejections)
}

func TestServerCallSetupRateAdmissionAllowsBurstAndRefill(t *testing.T) {
	clock := &fixedCallAdmissionClock{now: time.Unix(100, 0)}
	server := &Server{
		callAdmissionCPS:    2,
		callAdmissionBurst:  2,
		callAdmissionClock:  clock,
		callAdmissionTokens: 2,
	}

	require.True(t, server.tryAcquireCallSetupRate())
	require.True(t, server.tryAcquireCallSetupRate())
	require.False(t, server.tryAcquireCallSetupRate())

	clock.Add(500 * time.Millisecond)
	require.True(t, server.tryAcquireCallSetupRate())
	require.False(t, server.tryAcquireCallSetupRate())

	clock.Add(time.Second)
	require.True(t, server.tryAcquireCallSetupRate())
	require.True(t, server.tryAcquireCallSetupRate())
	require.False(t, server.tryAcquireCallSetupRate())
}

func TestInboundInviteRejectsWhenCallSetupRateExceeded(t *testing.T) {
	server := newServerForCommandTests(t)
	server.callAdmissionCPS = 1
	server.callAdmissionBurst = 1
	server.callAdmissionClock = &fixedCallAdmissionClock{now: time.Unix(100, 0)}
	server.callAdmissionLast = time.Unix(100, 0)
	server.callAdmissionTokens = 0
	request := newInboundInviteRequest("inbound-rate-full")
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 100, transaction.responses[0].StatusCode)
	assert.Equal(t, 503, transaction.lastStatus())
	retryAfter := transaction.responses[len(transaction.responses)-1].GetHeader(sipHeaderRetryAfter)
	require.NotNil(t, retryAfter)
	assert.Equal(t, "1", retryAfter.Value())
	_, exists := server.GetSession("inbound-rate-full")
	assert.False(t, exists)
	assert.Equal(t, uint64(1), server.CallAdmissionStats().RateRejections)

	retransmission := newTestServerTx()
	server.handleInvite(request, retransmission)

	require.NotEmpty(t, retransmission.responses)
	assert.Equal(t, 503, retransmission.lastStatus())
	retryAfter = retransmission.responses[len(retransmission.responses)-1].GetHeader(sipHeaderRetryAfter)
	require.NotNil(t, retryAfter)
	assert.Equal(t, "1", retryAfter.Value())
}

func TestInboundInviteRetransmissionDoesNotConsumeCallSetupRate(t *testing.T) {
	server := newServerForCommandTests(t)
	server.callAdmissionCPS = 1
	server.callAdmissionBurst = 1
	server.callAdmissionClock = &fixedCallAdmissionClock{now: time.Unix(100, 0)}
	server.callAdmissionLast = time.Unix(100, 0)
	server.callAdmissionTokens = 0
	request := newInboundInviteRequest("inbound-rate-retransmit")
	transaction := newTestServerTx()
	key := inboundInviteKey{callID: "inbound-rate-retransmit", fromTag: "fromtag"}
	require.True(t, server.setPendingInviteIfAbsent(key, request, newTestServerTx()))

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 100, transaction.lastStatus())
	assert.Equal(t, 0, transaction.statusCount(503))
	assert.Equal(t, float64(0), server.callAdmissionTokens)
}

func TestServerSetPendingInviteIfAbsentIsAtomic(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-pending-atomic")
	key := inboundInviteKey{callID: "inbound-pending-atomic", fromTag: "fromtag"}
	start := make(chan struct{})
	var accepted atomic.Int64
	var waitGroup sync.WaitGroup

	for range 16 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			if server.setPendingInviteIfAbsent(key, request, newTestServerTx()) {
				accepted.Add(1)
			}
		}()
	}

	close(start)
	waitGroup.Wait()

	server.mu.RLock()
	pendingCount := len(server.pendingInvites)
	server.mu.RUnlock()
	assert.Equal(t, int64(1), accepted.Load())
	assert.Equal(t, 1, pendingCount)
}

func TestMakeCallRejectsWhenCallSetupRateExceeded(t *testing.T) {
	server := &Server{
		callAdmissionCPS:    1,
		callAdmissionBurst:  1,
		callAdmissionClock:  &fixedCallAdmissionClock{now: time.Unix(100, 0)},
		callAdmissionLast:   time.Unix(100, 0),
		callAdmissionTokens: 0,
	}
	server.state.Store(int32(ServerStateRunning))

	session, err := server.MakeCall(
		context.Background(),
		testOutboundConfig(),
		"+15551234567",
		"+15557654321",
		MakeCallOptions{},
	)

	require.Nil(t, session)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSIPCallRateExceeded))
	assert.Equal(t, uint64(1), server.CallAdmissionStats().RateRejections)
}

func TestInboundInviteReleasesAdmissionWhenMediaOfferFails(t *testing.T) {
	server := newServerForCommandTests(t)
	server.maxConcurrentCalls = 1
	request := newInboundInviteRequest("inbound-release-media-failure")
	request.SetBody([]byte(unsupportedInboundOfferSDP()))
	transaction := newTestServerTx()

	server.handleInvite(request, transaction)

	require.NotEmpty(t, transaction.responses)
	assert.Equal(t, 488, transaction.lastStatus())
	assert.Equal(t, int64(0), server.CallAdmissionStats().ActiveCalls)
	require.True(t, server.tryAcquireCallSlot())
	server.releaseCallSlot()
}

func TestMakeCallReleasesAdmissionWhenMediaPrepareFails(t *testing.T) {
	server := newServerForCommandTests(t)
	server.maxConcurrentCalls = 1
	server.rtpPortRangeStart = 0
	server.rtpPortRangeEnd = 0
	server.state.Store(int32(ServerStateRunning))

	session, err := server.MakeCall(
		context.Background(),
		testOutboundConfig(),
		"+15551234567",
		"+15557654321",
		MakeCallOptions{},
	)

	require.Nil(t, session)
	require.Error(t, err)
	require.Eventually(t, func() bool {
		return server.CallAdmissionStats().ActiveCalls == 0
	}, time.Second, 10*time.Millisecond)
	require.True(t, server.tryAcquireCallSlot())
	server.releaseCallSlot()
}
