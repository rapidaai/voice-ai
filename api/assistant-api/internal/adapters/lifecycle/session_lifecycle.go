// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

type SessionState uint8

func (s SessionState) String() string {
	switch s {
	case StateNew:
		return "new"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateSwitching:
		return "switching"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type SessionEvent uint8

func (e SessionEvent) String() string {
	switch e {
	case EventConnectRequested:
		return "connect_requested"
	case EventInitializationCompleted:
		return "initialization_completed"
	case EventInitializationFailed:
		return "initialization_failed"
	case EventSwitchRequested:
		return "switch_requested"
	case EventSwitchCompleted:
		return "switch_completed"
	case EventSwitchFailedRecoverable:
		return "switch_failed_recoverable"
	case EventSwitchFailedFatal:
		return "switch_failed_fatal"
	case EventDisconnectRequested:
		return "disconnect_requested"
	case EventDisconnectCompleted:
		return "disconnect_completed"
	default:
		return "unknown"
	}
}

type SessionLifecycle interface {
	Current() SessionState
	CanBe(SessionEvent) bool
	Transition(SessionEvent) error
	ConfigureTimeouts(context.Context, string, *internal_assistant_entity.AssistantDeploymentBehavior, func(context.Context, ...internal_type.Packet) error) error
	StartIdleTimeout(internal_type.StartIdleTimeoutPacket, MessageLifecycle)
	StopIdleTimeout(internal_type.StopIdleTimeoutPacket)
	ExtendIdleTimeout(string, time.Duration)
	IdleTimeoutCount() uint64
	IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket) (internal_type.InjectMessagePacket, *protos.ConversationDisconnection)
	CloseTimeouts()
	ToolCall(internal_type.LLMToolCallPacket) []internal_type.Packet
	MaxSessionExpired(internal_type.MaxSessionExpiredPacket) *protos.ConversationDisconnection
}

type sessionLifecycle struct {
	mu    sync.RWMutex
	state SessionState

	idleTimeoutWatchdog     *watchdog.IdleTimeoutWatchdog
	maxSessionWatchdog      *watchdog.MaxSessionWatchdog
	idleTimeoutDuration     time.Duration
	idleTimeoutBackoff      uint64
	idleTimeoutPrompt       string
	idleTimeoutContextID    string
	maxSessionContextID     string
	idleTimeoutMessageOwner MessageLifecycle
	timeoutGeneration       uint64
	timeoutDone             <-chan struct{}
	stopTimeoutCancellation func() bool
}

func NewSessionLifecycle() SessionLifecycle {
	return NewSessionLifecycleWithState(StateNew)
}

func NewSessionLifecycleWithState(initial SessionState) SessionLifecycle {
	return &sessionLifecycle{state: initial}
}

func (l *sessionLifecycle) Current() SessionState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *sessionLifecycle) CanBe(event SessionEvent) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, err := nextSessionState(l.state, event)
	return err == nil
}

func (l *sessionLifecycle) Transition(event SessionEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	next, err := nextSessionState(l.state, event)
	if err != nil {
		return err
	}
	l.state = next
	return nil
}

func (l *sessionLifecycle) ConfigureTimeouts(
	ctx context.Context,
	contextID string,
	behavior *internal_assistant_entity.AssistantDeploymentBehavior,
	onPacket func(context.Context, ...internal_type.Packet) error,
) error {
	var idleTimeoutDuration, maxSessionDuration time.Duration
	if behavior != nil && behavior.IdleTimeout != nil {
		idleTimeoutSeconds, err := utils.Uint64ToInt64(*behavior.IdleTimeout)
		if err != nil {
			return fmt.Errorf("invalid idle timeout: %w", err)
		}
		if idleTimeoutSeconds > math.MaxInt64/int64(time.Second) {
			return fmt.Errorf("idle timeout %d seconds exceeds duration range", idleTimeoutSeconds)
		}
		idleTimeoutDuration = time.Duration(idleTimeoutSeconds) * time.Second
	}
	if behavior != nil && behavior.MaxSessionDuration != nil {
		maxSessionSeconds, err := utils.Uint64ToInt64(*behavior.MaxSessionDuration)
		if err != nil {
			return fmt.Errorf("invalid maximum session duration: %w", err)
		}
		if maxSessionSeconds > math.MaxInt64/int64(time.Second) {
			return fmt.Errorf("maximum session duration %d seconds exceeds duration range", maxSessionSeconds)
		}
		maxSessionDuration = time.Duration(maxSessionSeconds) * time.Second
	}

	l.mu.Lock()
	if l.stopTimeoutCancellation != nil {
		l.stopTimeoutCancellation()
		l.stopTimeoutCancellation = nil
	}
	if l.idleTimeoutWatchdog != nil {
		l.idleTimeoutWatchdog.Cancel()
		l.idleTimeoutWatchdog = nil
	}
	if l.maxSessionWatchdog != nil {
		l.maxSessionWatchdog.Cancel()
		l.maxSessionWatchdog = nil
	}
	l.timeoutGeneration++
	generation := l.timeoutGeneration
	l.idleTimeoutContextID = ""
	l.maxSessionContextID = ""
	l.idleTimeoutMessageOwner = nil
	l.timeoutDone = ctx.Done()
	l.idleTimeoutDuration = idleTimeoutDuration
	l.idleTimeoutBackoff = 0
	l.idleTimeoutPrompt = "Are you still there?"
	if behavior != nil {
		if behavior.IdleTimeoutBackoff != nil {
			l.idleTimeoutBackoff = *behavior.IdleTimeoutBackoff
		}
		if behavior.IdleTimeoutMessage != nil && validator.NotBlank(*behavior.IdleTimeoutMessage) {
			l.idleTimeoutPrompt = *behavior.IdleTimeoutMessage
		}
	}
	l.mu.Unlock()

	if ctx.Err() != nil {
		return nil
	}
	forwardPacket := func(packetCtx context.Context, packets ...internal_type.Packet) error {
		l.mu.RLock()
		isCurrent := generation == l.timeoutGeneration
		l.mu.RUnlock()
		if !isCurrent || packetCtx.Err() != nil || onPacket == nil {
			return nil
		}
		return onPacket(packetCtx, packets...)
	}
	// Watchdog constructors emit packets, so callbacks must run outside the lifecycle lock.
	idle := watchdog.NewIdleTimeoutWatchdog(watchdog.WithPacketContext(ctx), watchdog.WithOnPacket(forwardPacket))
	maxSession := watchdog.NewMaxSessionWatchdog(watchdog.WithPacketContext(ctx), watchdog.WithOnPacket(forwardPacket))

	l.mu.Lock()
	defer l.mu.Unlock()
	if generation != l.timeoutGeneration || ctx.Err() != nil {
		return nil
	}
	l.idleTimeoutWatchdog = idle
	l.maxSessionWatchdog = maxSession
	if maxSession.Start(contextID, maxSessionDuration) {
		l.maxSessionContextID = contextID
	}
	l.stopTimeoutCancellation = context.AfterFunc(ctx, func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if generation != l.timeoutGeneration {
			return
		}
		idle.Cancel()
		maxSession.Cancel()
		l.idleTimeoutWatchdog = nil
		l.maxSessionWatchdog = nil
		l.idleTimeoutContextID = ""
		l.maxSessionContextID = ""
		l.idleTimeoutMessageOwner = nil
		l.stopTimeoutCancellation = nil
		l.timeoutGeneration++
	})
	return nil
}

func (l *sessionLifecycle) StartIdleTimeout(p internal_type.StartIdleTimeoutPacket, message MessageLifecycle) {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.timeoutDone:
		return
	default:
	}
	if l.idleTimeoutWatchdog == nil || l.idleTimeoutDuration <= 0 || message == nil || !message.CanStartIdleTimeout(p.ContextID) {
		return
	}
	l.idleTimeoutWatchdog.Start(p.ContextID, l.idleTimeoutDuration)
	l.idleTimeoutContextID = p.ContextID
	l.idleTimeoutMessageOwner = message
}

func (l *sessionLifecycle) StopIdleTimeout(p internal_type.StopIdleTimeoutPacket) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p.ContextID != "" && l.idleTimeoutContextID != "" && p.ContextID != l.idleTimeoutContextID {
		return
	}
	if l.idleTimeoutWatchdog != nil {
		l.idleTimeoutWatchdog.Stop(p.ResetCount)
	}
	l.idleTimeoutContextID = ""
	l.idleTimeoutMessageOwner = nil
}

func (l *sessionLifecycle) ExtendIdleTimeout(contextID string, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.timeoutDone:
		return
	default:
	}
	if l.idleTimeoutWatchdog != nil {
		l.idleTimeoutWatchdog.Extend(contextID, duration)
	}
}

func (l *sessionLifecycle) IdleTimeoutCount() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.idleTimeoutWatchdog == nil {
		return 0
	}
	return l.idleTimeoutWatchdog.Count()
}

func (l *sessionLifecycle) IdleTimeoutExpired(p internal_type.IdleTimeoutExpiredPacket) (internal_type.InjectMessagePacket, *protos.ConversationDisconnection) {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.timeoutDone:
		return internal_type.InjectMessagePacket{}, nil
	default:
	}
	if l.idleTimeoutWatchdog == nil || l.idleTimeoutDuration <= 0 || p.ContextID == "" || p.ContextID != l.idleTimeoutContextID {
		return internal_type.InjectMessagePacket{}, nil
	}
	if l.idleTimeoutMessageOwner != nil && !l.idleTimeoutMessageOwner.CanStartIdleTimeout(p.ContextID) {
		return internal_type.InjectMessagePacket{}, nil
	}
	if !l.idleTimeoutWatchdog.AcceptExpiry(p) {
		return internal_type.InjectMessagePacket{}, nil
	}
	l.idleTimeoutWatchdog.Cancel()
	l.idleTimeoutContextID = ""
	l.idleTimeoutMessageOwner = nil
	if l.idleTimeoutBackoff > 0 && l.idleTimeoutWatchdog.Count() >= l.idleTimeoutBackoff {
		return internal_type.InjectMessagePacket{}, &protos.ConversationDisconnection{
			Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT,
		}
	}
	l.idleTimeoutWatchdog.IncrementCount()
	return internal_type.InjectMessagePacket{ContextID: p.ContextID, Text: l.idleTimeoutPrompt}, nil
}

func (l *sessionLifecycle) CloseTimeouts() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.stopTimeoutCancellation != nil {
		l.stopTimeoutCancellation()
		l.stopTimeoutCancellation = nil
	}
	if l.idleTimeoutWatchdog != nil {
		l.idleTimeoutWatchdog.Cancel()
		l.idleTimeoutWatchdog = nil
	}
	if l.maxSessionWatchdog != nil {
		l.maxSessionWatchdog.Cancel()
		l.maxSessionWatchdog = nil
	}
	l.idleTimeoutContextID = ""
	l.maxSessionContextID = ""
	l.idleTimeoutMessageOwner = nil
	l.timeoutGeneration++
}

func (l *sessionLifecycle) ToolCall(p internal_type.LLMToolCallPacket) []internal_type.Packet {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p.Action == protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED {
		return nil
	}
	if l.maxSessionWatchdog != nil {
		l.maxSessionWatchdog.Cancel()
	}
	l.maxSessionContextID = ""
	return []internal_type.Packet{internal_type.StopIdleTimeoutPacket{ContextID: p.ContextID, ResetCount: true}}
}

func (l *sessionLifecycle) MaxSessionExpired(p internal_type.MaxSessionExpiredPacket) *protos.ConversationDisconnection {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.timeoutDone:
		return nil
	default:
	}
	if p.ContextID == "" || p.ContextID != l.maxSessionContextID {
		return nil
	}
	l.maxSessionContextID = ""
	if l.maxSessionWatchdog != nil {
		l.maxSessionWatchdog.Cancel()
	}
	return &protos.ConversationDisconnection{Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION}
}

func nextSessionState(current SessionState, event SessionEvent) (SessionState, error) {
	switch current {
	case StateNew:
		switch event {
		case EventConnectRequested:
			return StateInitializing, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateInitializing:
		switch event {
		case EventInitializationCompleted:
			return StateReady, nil
		case EventInitializationFailed:
			return StateFailed, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateReady:
		switch event {
		case EventSwitchRequested:
			return StateSwitching, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateSwitching:
		switch event {
		case EventSwitchCompleted:
			return StateReady, nil
		case EventSwitchFailedRecoverable:
			return StateReady, nil
		case EventSwitchFailedFatal:
			return StateFailed, nil
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateFailed:
		switch event {
		case EventDisconnectRequested:
			return StateDisconnecting, nil
		}
	case StateDisconnecting:
		if event == EventDisconnectCompleted {
			return StateDisconnected, nil
		}
	case StateDisconnected:
	}

	return current, fmt.Errorf("invalid session lifecycle transition: state=%s event=%s", current.String(), event.String())
}
