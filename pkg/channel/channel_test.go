// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "zero capacity",
			config: Config{
				CapacityPolicy: FixedCapacity(0),
				OverflowPolicy: ReplaceOldestWhenFull,
			},
		},
		{
			name: "missing policy",
			config: Config{
				CapacityPolicy: FixedCapacity(1),
			},
		},
		{
			name: "maximum below initial capacity",
			config: Config{
				CapacityPolicy: GrowingCapacity(2, 1),
				OverflowPolicy: ReplaceOldestWhenFull,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel, err := New[int](test.config)
			assert.Nil(t, channel)
			assert.Error(t, err)
		})
	}
}

func TestChannelBlocksWhenFull(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)

	resultCh := make(chan SendResult[int], 1)
	errorCh := make(chan error, 1)
	go func() {
		result, sendErr := channel.Send(t.Context(), 2)
		resultCh <- result
		errorCh <- sendErr
	}()

	select {
	case <-resultCh:
		t.Fatal("send completed before space became available")
	case <-time.After(10 * time.Millisecond):
	}

	value, err := channel.Receive(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, value)
	assert.Equal(t, Enqueued, (<-resultCh).Status)
	assert.NoError(t, <-errorCh)

	value, err = channel.Receive(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, value)
}

func TestChannelBlockWhenFullHonorsCancellation(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = channel.Send(ctx, 2)
	assert.ErrorIs(t, err, context.Canceled)

	value, err := channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, value)
}

func TestChannelRejectsNewestWhenFull(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(2), OverflowPolicy: RejectNewestWhenFull})
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 2)
	require.NoError(t, err)

	result, err := channel.Send(t.Context(), 3)
	require.NoError(t, err)
	assert.Equal(t, Rejected, result.Status)
	assert.Equal(t, 3, result.Discarded)

	value, err := channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, value)
	value, err = channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, value)
}

func TestChannelReplacesOldestWhenFull(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(2), OverflowPolicy: ReplaceOldestWhenFull})
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 2)
	require.NoError(t, err)

	result, err := channel.Send(t.Context(), 3)
	require.NoError(t, err)
	assert.Equal(t, ReplacedOldest, result.Status)
	assert.Equal(t, 1, result.Discarded)

	value, err := channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, value)
	value, err = channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, value)
}

func TestChannelReplacesOldestMatchingValueWhenFull(t *testing.T) {
	type value struct {
		name        string
		replaceable bool
	}
	channel, err := New[value](
		Config{CapacityPolicy: FixedCapacity(3), OverflowPolicy: ReplaceOldestWhenFull},
		WithReplacementFilter(func(value value) bool { return value.replaceable }),
	)
	require.NoError(t, err)
	for _, queued := range []value{
		{name: "control"},
		{name: "old-audio", replaceable: true},
		{name: "transcript"},
	} {
		_, err = channel.Send(t.Context(), queued)
		require.NoError(t, err)
	}

	result, err := channel.Send(t.Context(), value{name: "new-audio", replaceable: true})
	require.NoError(t, err)
	assert.Equal(t, ReplacedOldest, result.Status)
	assert.Equal(t, "old-audio", result.Discarded.name)

	for _, expected := range []string{"control", "transcript", "new-audio"} {
		queued, receiveErr := channel.Receive(t.Context())
		require.NoError(t, receiveErr)
		assert.Equal(t, expected, queued.name)
	}
}

func TestChannelRejectsNewestWhenNoBufferedValueMatchesReplacementFilter(t *testing.T) {
	channel, err := New[int](
		Config{CapacityPolicy: FixedCapacity(2), OverflowPolicy: ReplaceOldestWhenFull},
		WithReplacementFilter(func(value int) bool { return value < 0 }),
	)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 2)
	require.NoError(t, err)

	result, err := channel.Send(t.Context(), 3)
	require.NoError(t, err)
	assert.Equal(t, Rejected, result.Status)
	assert.Equal(t, 3, result.Discarded)

	for _, expected := range []int{1, 2} {
		queued, receiveErr := channel.Receive(t.Context())
		require.NoError(t, receiveErr)
		assert.Equal(t, expected, queued)
	}
}

func TestChannelCloseRejectsFutureSendsAndDrains(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: ReplaceOldestWhenFull})
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)

	channel.Close()
	channel.Close()

	_, err = channel.Send(t.Context(), 2)
	assert.ErrorIs(t, err, ErrClosed)
	value, err := channel.Receive(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, value)
	_, err = channel.Receive(t.Context())
	assert.ErrorIs(t, err, ErrClosed)
}

func TestChannelCloseUnblocksSenderAndReceiver(t *testing.T) {
	waitingSender, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)
	_, err = waitingSender.Send(t.Context(), 1)
	require.NoError(t, err)
	sendErrorCh := make(chan error, 1)
	go func() {
		_, sendErr := waitingSender.Send(t.Context(), 2)
		sendErrorCh <- sendErr
	}()

	waitingReceiver, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)
	receiveErrorCh := make(chan error, 1)
	go func() {
		_, receiveErr := waitingReceiver.Receive(t.Context())
		receiveErrorCh <- receiveErr
	}()

	waitingSender.Close()
	waitingReceiver.Close()

	assert.ErrorIs(t, <-sendErrorCh, ErrClosed)
	assert.ErrorIs(t, <-receiveErrorCh, ErrClosed)
}

func TestChannelReceiveHonorsCancellation(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = channel.Receive(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestChannelRequiresContext(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(1), OverflowPolicy: BlockWhenFull})
	require.NoError(t, err)
	var missingContext context.Context

	_, err = channel.Send(missingContext, 1)
	assert.Error(t, err)
	_, err = channel.Receive(missingContext)
	assert.Error(t, err)
}

func TestChannelConcurrentSendAndReceive(t *testing.T) {
	channel, err := New[int](Config{CapacityPolicy: FixedCapacity(8), OverflowPolicy: ReplaceOldestWhenFull})
	require.NoError(t, err)

	var writers sync.WaitGroup
	for writer := range 8 {
		writers.Add(1)
		go func(value int) {
			defer writers.Done()
			for range 100 {
				_, sendErr := channel.Send(t.Context(), value)
				if sendErr != nil && !errors.Is(sendErr, ErrClosed) {
					t.Errorf("send failed: %v", sendErr)
				}
			}
		}(writer)
	}

	writers.Wait()
	channel.Close()

	count := 0
	for {
		_, receiveErr := channel.Receive(t.Context())
		if errors.Is(receiveErr, ErrClosed) {
			break
		}
		require.NoError(t, receiveErr)
		count++
	}
	assert.LessOrEqual(t, count, 8)
}

func TestChannelGrowsUntilMaximumCapacity(t *testing.T) {
	channel, err := New[int](Config{
		CapacityPolicy: GrowingCapacity(1, 4),
		OverflowPolicy: RejectNewestWhenFull,
	})
	require.NoError(t, err)

	for value := range 4 {
		result, sendErr := channel.Send(t.Context(), value)
		require.NoError(t, sendErr)
		assert.Equal(t, Enqueued, result.Status)
	}

	result, err := channel.Send(t.Context(), 4)
	require.NoError(t, err)
	assert.Equal(t, Rejected, result.Status)
	assert.Equal(t, 4, result.Discarded)

	for expected := range 4 {
		value, receiveErr := channel.Receive(t.Context())
		require.NoError(t, receiveErr)
		assert.Equal(t, expected, value)
	}
}

func TestChannelPreservesOrderWhenGrowingWrappedBuffer(t *testing.T) {
	channel, err := New[int](Config{
		CapacityPolicy: GrowingCapacity(2, 4),
		OverflowPolicy: RejectNewestWhenFull,
	})
	require.NoError(t, err)

	_, err = channel.Send(t.Context(), 1)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 2)
	require.NoError(t, err)
	_, err = channel.Receive(t.Context())
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 3)
	require.NoError(t, err)
	_, err = channel.Send(t.Context(), 4)
	require.NoError(t, err)

	for _, expected := range []int{2, 3, 4} {
		value, receiveErr := channel.Receive(t.Context())
		require.NoError(t, receiveErr)
		assert.Equal(t, expected, value)
	}
}
