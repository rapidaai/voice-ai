// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
)

type IdleTimeoutEvent struct {
	ContextID  string
	Deadline   time.Time
	Count      uint64
	Generation uint64
}

type IdleTimeoutOptions = WatchdogOptions
type IdleTimeoutOption = Option

type IdleTimeoutWatchdog struct {
	mu      sync.Mutex
	options IdleTimeoutOptions

	timer *time.Timer

	generation       uint64
	active           bool
	contextID        string
	deadline         time.Time
	idleTimeoutCount uint64
	expired          IdleTimeoutEvent
}

func NewIdleTimeoutWatchdog(opts ...IdleTimeoutOption) *IdleTimeoutWatchdog {
	options := IdleTimeoutOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.applyWatchdogOptions(&options)
	}

	if !validator.NonNil(options.PacketContext) {
		options.PacketContext = context.Background()
	}
	if !validator.NotBlank(options.RecordScope.String()) {
		options.RecordScope = internal_type.ObservabilityRecordScopeConversation
	}

	watchdog := &IdleTimeoutWatchdog{
		options: options,
	}

	if options.OnPacket != nil {
		_ = options.OnPacket(
			options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				Scope: options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "idle-timeout-watchdog: initialization completed",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"watchdog":  "idle_timeout",
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return watchdog
}

func (w *IdleTimeoutWatchdog) Start(contextID string, timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.expired = IdleTimeoutEvent{}
	w.active = true
	w.contextID = contextID
	w.deadline = time.Now().Add(timeout)

	generation := w.generation
	w.timer = time.AfterFunc(timeout, func() {
		w.expire(generation)
	})

	return true
}

func (w *IdleTimeoutWatchdog) Extend(contextID string, duration time.Duration) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if duration <= 0 {
		return false
	}
	if w.active {
		if w.contextID != contextID {
			return false
		}
	} else {
		// A queued expiry remains extendable until the session admits it.
		if w.expired.Generation == 0 || w.expired.ContextID != contextID {
			return false
		}
		w.contextID = w.expired.ContextID
		w.deadline = w.expired.Deadline
	}
	if w.timer != nil {
		w.timer.Stop()
	}
	w.deadline = w.deadline.Add(duration)
	w.generation++
	w.expired = IdleTimeoutEvent{}
	w.active = true
	generation := w.generation
	w.timer = time.AfterFunc(time.Until(w.deadline), func() {
		w.expire(generation)
	})

	return true
}

func (w *IdleTimeoutWatchdog) Stop(resetCount bool) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	wasActive := w.active
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.expired = IdleTimeoutEvent{}
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	if resetCount {
		w.idleTimeoutCount = 0
	}

	return wasActive
}

func (w *IdleTimeoutWatchdog) Cancel() bool {
	return w.Stop(false)
}

func (w *IdleTimeoutWatchdog) Count() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.idleTimeoutCount
}

func (w *IdleTimeoutWatchdog) IncrementCount() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.idleTimeoutCount++
	return w.idleTimeoutCount
}

// AcceptExpiry consumes only the current countdown's emitted expiry.
// The deadline also fences replacement watchdogs whose generation starts over.
func (w *IdleTimeoutWatchdog) AcceptExpiry(packet internal_type.IdleTimeoutExpiredPacket) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.options.PacketContext.Err() != nil || packet.Generation == 0 ||
		packet.Generation != w.expired.Generation || packet.ContextID != w.expired.ContextID ||
		!packet.Deadline.Equal(w.expired.Deadline) || packet.Count != w.expired.Count ||
		packet.Count != w.idleTimeoutCount {
		return false
	}
	w.expired = IdleTimeoutEvent{}
	return true
}

func (w *IdleTimeoutWatchdog) expire(generation uint64) {
	w.mu.Lock()
	if !w.active || w.generation != generation {
		w.mu.Unlock()
		return
	}

	event := IdleTimeoutEvent{
		ContextID:  w.contextID,
		Deadline:   w.deadline,
		Count:      w.idleTimeoutCount,
		Generation: generation,
	}
	w.expired = event
	w.timer = nil
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.mu.Unlock()

	if w.options.OnPacket != nil {
		_ = w.options.OnPacket(
			w.options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				ContextID: event.ContextID,
				Scope:     w.options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "idle-timeout-watchdog: deadline expired",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"watchdog":  "idle_timeout",
						"count":     observability.AttributeValue(event.Count),
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.IdleTimeoutExpiredPacket{
				ContextID: event.ContextID, Count: event.Count,
				Generation: event.Generation, Deadline: event.Deadline,
			},
		)
	}
}
