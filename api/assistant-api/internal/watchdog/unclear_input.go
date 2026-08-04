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

type UnclearInputOptions = WatchdogOptions
type UnclearInputOption = Option

type UnclearInputWatchdog struct {
	mu      sync.Mutex
	options UnclearInputOptions

	timer *time.Timer

	generation uint64
	active     bool
	contextID  string
	deadline   time.Time
}

func NewUnclearInputWatchdog(opts ...UnclearInputOption) *UnclearInputWatchdog {
	options := UnclearInputOptions{}
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

	watchdog := &UnclearInputWatchdog{
		options: options,
	}

	if options.OnPacket != nil {
		_ = options.OnPacket(
			options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				Scope: options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "unclear-input-watchdog: initialization completed",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"watchdog":  "unclear_input",
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return watchdog
}

func (w *UnclearInputWatchdog) Start(contextID string, timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.startLocked(contextID, timeout)
	return true
}

func (w *UnclearInputWatchdog) Extend(contextID string, timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active || w.contextID != contextID {
		return false
	}

	w.startLocked(contextID, timeout)
	return true
}

func (w *UnclearInputWatchdog) Stop() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	wasActive := w.active
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}

	return wasActive
}

func (w *UnclearInputWatchdog) Cancel() bool {
	return w.Stop()
}

func (w *UnclearInputWatchdog) startLocked(contextID string, timeout time.Duration) {
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = true
	w.contextID = contextID
	w.deadline = time.Now().Add(timeout)

	generation := w.generation
	w.timer = time.AfterFunc(timeout, func() {
		w.expire(generation)
	})
}

func (w *UnclearInputWatchdog) expire(generation uint64) {
	w.mu.Lock()
	if !w.active || w.generation != generation {
		w.mu.Unlock()
		return
	}

	contextID := w.contextID
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
				ContextID: contextID,
				Scope:     w.options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "unclear-input-watchdog: deadline expired",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"watchdog":  "unclear_input",
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.UnclearInputExpiredPacket{ContextID: contextID},
		)
	}
}
