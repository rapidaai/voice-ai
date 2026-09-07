// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrClosed = errors.New("channel is closed")
	ErrEmpty  = errors.New("channel is empty")
)

// CapacityPolicy defines the initial and maximum number of buffered values.
// Construct policies with FixedCapacity or GrowingCapacity.
type CapacityPolicy struct {
	initial int
	maximum int
}

// FixedCapacity keeps the channel at a constant capacity. Once full, the
// configured OverflowPolicy decides how subsequent sends are handled.
func FixedCapacity(capacity int) CapacityPolicy {
	return CapacityPolicy{initial: capacity, maximum: capacity}
}

// GrowingCapacity expands the channel geometrically from initialCapacity up
// to maximumCapacity. The OverflowPolicy applies only after that limit.
func GrowingCapacity(initialCapacity, maximumCapacity int) CapacityPolicy {
	return CapacityPolicy{initial: initialCapacity, maximum: maximumCapacity}
}

// OverflowPolicy defines how a Channel behaves after reaching its maximum
// capacity.
type OverflowPolicy uint8

const (
	// BlockWhenFull waits until a receiver creates space or the context ends.
	BlockWhenFull OverflowPolicy = iota + 1
	// RejectNewestWhenFull preserves buffered values and rejects the new value.
	RejectNewestWhenFull
	// ReplaceOldestWhenFull discards the oldest value and accepts the new value.
	ReplaceOldestWhenFull
)

// SendStatus describes how a value was handled.
type SendStatus uint8

const (
	// Enqueued indicates that the value was added without discarding a value.
	Enqueued SendStatus = iota + 1
	// ReplacedOldest indicates that the oldest value was discarded.
	ReplacedOldest
	// Rejected indicates that the incoming value was discarded.
	Rejected
)

// Config defines the fixed behavior of a Channel.
type Config struct {
	CapacityPolicy CapacityPolicy
	OverflowPolicy OverflowPolicy
}

// Option configures value-specific channel behavior.
type Option[T any] func(*Channel[T])

// WithReplacementFilter limits ReplaceOldestWhenFull to the oldest buffered
// value accepted by match. The incoming value is rejected when none match.
func WithReplacementFilter[T any](match func(T) bool) Option[T] {
	return func(channel *Channel[T]) {
		channel.replacementFilter = match
	}
}

// SendResult describes the outcome of a successful policy decision. Discarded
// is populated when Status is ReplacedOldest or Rejected.
type SendResult[T any] struct {
	Status    SendStatus
	Discarded T
}

// Channel is a concurrent bounded FIFO with explicit capacity and overflow
// policies. Its zero value is not usable; construct it with New.
//
// Channel owns its ring buffer. Send, Receive, and Close serialize all state
// transitions, so replacing the oldest value is atomic with receiving values.
type Channel[T any] struct {
	mu       sync.Mutex
	values   []T
	head     int
	length   int
	maximum  int
	policy   OverflowPolicy
	notEmpty *sync.Cond
	notFull  *sync.Cond
	ready    chan struct{}
	isClosed bool

	replacementFilter func(T) bool
}

// New creates a channel with validated capacity and overflow policies.
func New[T any](config Config, options ...Option[T]) (*Channel[T], error) {
	if config.CapacityPolicy.initial <= 0 {
		return nil, fmt.Errorf("channel initial capacity must be greater than zero: %d", config.CapacityPolicy.initial)
	}
	if config.CapacityPolicy.maximum < config.CapacityPolicy.initial {
		return nil, fmt.Errorf(
			"channel maximum capacity must be at least initial capacity: initial=%d maximum=%d",
			config.CapacityPolicy.initial,
			config.CapacityPolicy.maximum,
		)
	}
	if config.OverflowPolicy < BlockWhenFull || config.OverflowPolicy > ReplaceOldestWhenFull {
		return nil, fmt.Errorf("invalid channel overflow policy: %d", config.OverflowPolicy)
	}

	channel := &Channel[T]{
		values:  make([]T, config.CapacityPolicy.initial),
		maximum: config.CapacityPolicy.maximum,
		policy:  config.OverflowPolicy,
		ready:   make(chan struct{}, 1),
	}
	channel.notEmpty = sync.NewCond(&channel.mu)
	channel.notFull = sync.NewCond(&channel.mu)
	for _, option := range options {
		if option != nil {
			option(channel)
		}
	}
	return channel, nil
}

// Send adds value according to the configured capacity and overflow policies.
// It returns ErrClosed after Close and returns the context error when canceled.
func (c *Channel[T]) Send(ctx context.Context, value T) (SendResult[T], error) {
	if ctx == nil {
		return SendResult[T]{}, errors.New("channel send requires a context")
	}

	for {
		c.mu.Lock()
		if c.isClosed {
			c.mu.Unlock()
			return SendResult[T]{}, ErrClosed
		}
		if err := ctx.Err(); err != nil {
			c.mu.Unlock()
			return SendResult[T]{}, err
		}
		if c.length < len(c.values) {
			c.push(value)
			c.notEmpty.Signal()
			c.updateReady()
			c.mu.Unlock()
			return SendResult[T]{Status: Enqueued}, nil
		}
		if len(c.values) < c.maximum {
			c.grow()
			c.push(value)
			c.notEmpty.Signal()
			c.updateReady()
			c.mu.Unlock()
			return SendResult[T]{Status: Enqueued}, nil
		}

		switch c.policy {
		case RejectNewestWhenFull:
			c.mu.Unlock()
			return SendResult[T]{Status: Rejected, Discarded: value}, nil
		case ReplaceOldestWhenFull:
			discarded, replaced := c.replaceOldest(value)
			if !replaced {
				c.mu.Unlock()
				return SendResult[T]{Status: Rejected, Discarded: value}, nil
			}
			c.updateReady()
			c.mu.Unlock()
			return SendResult[T]{Status: ReplacedOldest, Discarded: discarded}, nil
		case BlockWhenFull:
			c.wait(ctx, c.notFull)
			c.mu.Unlock()
		default:
			c.mu.Unlock()
			return SendResult[T]{}, fmt.Errorf("invalid channel overflow policy: %d", c.policy)
		}
	}
}

// Receive waits for and removes the oldest buffered value. After Close, it
// continues draining buffered values and then returns ErrClosed.
func (c *Channel[T]) Receive(ctx context.Context) (T, error) {
	var zero T
	if ctx == nil {
		return zero, errors.New("channel receive requires a context")
	}

	for {
		c.mu.Lock()
		if err := ctx.Err(); err != nil {
			c.mu.Unlock()
			return zero, err
		}
		if c.length > 0 {
			value := c.pop()
			c.notFull.Signal()
			c.updateReady()
			c.mu.Unlock()
			return value, nil
		}
		if c.isClosed {
			c.mu.Unlock()
			return zero, ErrClosed
		}

		c.wait(ctx, c.notEmpty)
		c.mu.Unlock()
	}
}

// TryReceive removes the oldest buffered value without waiting.
func (c *Channel[T]) TryReceive() (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.length > 0 {
		value := c.pop()
		c.notFull.Signal()
		c.updateReady()
		return value, nil
	}

	var zero T
	if c.isClosed {
		return zero, ErrClosed
	}
	return zero, ErrEmpty
}

// Ready is notified while values are available or after the channel closes.
// A receiver must call TryReceive after a notification.
func (c *Channel[T]) Ready() <-chan struct{} {
	return c.ready
}

// Len returns a concurrent snapshot of the buffered value count.
func (c *Channel[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.length
}

// Capacity returns the channel's current buffer capacity.
func (c *Channel[T]) Capacity() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.values)
}

// Drain removes every buffered value and returns the number removed.
func (c *Channel[T]) Drain() int {
	return c.RemoveIf(nil)
}

// RemoveIf removes buffered values accepted by match while preserving order.
// A nil match removes every buffered value.
func (c *Channel[T]) RemoveIf(match func(T) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.length == 0 {
		return 0
	}

	values := make([]T, len(c.values))
	kept := 0
	removed := 0
	for index := range c.length {
		value := c.values[(c.head+index)%len(c.values)]
		if match == nil || match(value) {
			removed++
			continue
		}
		values[kept] = value
		kept++
	}
	c.values = values
	c.head = 0
	c.length = kept
	if removed > 0 {
		c.notFull.Broadcast()
	}
	c.updateReady()
	return removed
}

// Close rejects future sends, wakes blocked operations, and allows receivers to
// drain values that were accepted before closure. It is safe to call repeatedly.
func (c *Channel[T]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	c.notEmpty.Broadcast()
	c.notFull.Broadcast()
	c.updateReady()
}

func (c *Channel[T]) push(value T) {
	index := (c.head + c.length) % len(c.values)
	c.values[index] = value
	c.length++
}

func (c *Channel[T]) pop() T {
	value := c.values[c.head]
	var zero T
	c.values[c.head] = zero
	c.head = (c.head + 1) % len(c.values)
	c.length--
	return value
}

func (c *Channel[T]) replaceOldest(value T) (T, bool) {
	if c.replacementFilter == nil {
		discarded := c.pop()
		c.push(value)
		return discarded, true
	}

	for offset := range c.length {
		index := (c.head + offset) % len(c.values)
		if !c.replacementFilter(c.values[index]) {
			continue
		}

		discarded := c.values[index]
		for nextOffset := offset; nextOffset < c.length-1; nextOffset++ {
			current := (c.head + nextOffset) % len(c.values)
			next := (c.head + nextOffset + 1) % len(c.values)
			c.values[current] = c.values[next]
		}
		var zero T
		tail := (c.head + c.length - 1) % len(c.values)
		c.values[tail] = zero
		c.length--
		c.push(value)
		return discarded, true
	}

	var zero T
	return zero, false
}

func (c *Channel[T]) grow() {
	capacity := min(len(c.values)*2, c.maximum)
	values := make([]T, capacity)
	for index := range c.length {
		values[index] = c.values[(c.head+index)%len(c.values)]
	}
	c.values = values
	c.head = 0
}

func (c *Channel[T]) wait(ctx context.Context, condition *sync.Cond) {
	if ctx.Done() == nil {
		condition.Wait()
		return
	}

	stop := context.AfterFunc(ctx, func() {
		condition.L.Lock()
		condition.Broadcast()
		condition.L.Unlock()
	})
	condition.Wait()
	stop()
}

func (c *Channel[T]) updateReady() {
	if c.length > 0 || c.isClosed {
		select {
		case c.ready <- struct{}{}:
		default:
		}
		return
	}

	select {
	case <-c.ready:
	default:
	}
}
