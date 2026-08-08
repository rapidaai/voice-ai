// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type MessageState string

const (
	MessageStateInterrupt MessageState = "interrupt"

	MessageStateUserIdle      MessageState = "user_idle"
	MessageStateUserListening MessageState = "user_listening"
	MessageStateUserSpeaking  MessageState = "user_speaking"
	MessageStateUserThinking  MessageState = "user_thinking"
	MessageStateUserFinished  MessageState = "user_finished"
	MessageStateUserPrompted  MessageState = "user_prompted"

	MessageStateAssistantGenerating MessageState = "assistant_generating"
	MessageStateAssistantGenerated  MessageState = "assistant_generated"
	MessageStateAssistantSpeaking   MessageState = "assistant_speaking"
	MessageStateAssistantFinished   MessageState = "assistant_finished"
	MessageStateAssistantIdle       MessageState = "assistant_idle"
	MessageStateAssistantPrompted   MessageState = "assistant_prompted"
)

var (
	ErrEmptyContextID    = errors.New("empty context id")
	ErrStaleContext      = errors.New("stale context")
	ErrInvalidTransition = errors.New("invalid message lifecycle transition")
)

type MessageLifecycle interface {
	ContextID() string
	RotateContext() (string, string, error)
	Mode() type_enums.MessageMode
	SetMode(type_enums.MessageMode)
	State() MessageState
	UserIdle(string) error
	UserListening(string) error
	UserSpeaking(string) error
	UserThinking(string) error
	UserFinished(string) error
	UserPrompted(string) error
	UserPromptCount() uint64
	AssistantGenerating(string) error
	AssistantGenerated(string) error
	AssistantSpeaking(string) error
	AssistantFinished(string) error
	AssistantIdle(string) error
	AssistantPrompted(string) error
	AssistantPromptCount() uint64
	BeginInterrupt(string) error
	CancelInterrupt(string) error
}

type messageLifecycle struct {
	mu               sync.RWMutex
	contextID        string
	mode             type_enums.MessageMode
	state            MessageState
	userPrompts      uint64
	assistantPrompts uint64
}

func NewMessageLifecycle() MessageLifecycle {
	return NewMessageLifecycleWithContext(uuid.NewString(), type_enums.TextMode)
}

func NewMessageLifecycleWithContext(
	initialContextID string,
	initialMode type_enums.MessageMode,
) MessageLifecycle {
	if initialContextID == "" {
		initialContextID = uuid.NewString()
	}
	if initialMode == "" {
		initialMode = type_enums.TextMode
	}
	return &messageLifecycle{
		contextID: initialContextID,
		mode:      initialMode,
		state:     MessageStateAssistantIdle,
	}
}

func (l *messageLifecycle) ContextID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.contextID
}

func (l *messageLifecycle) RotateContext() (string, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	oldContextID := l.contextID
	newContextID := uuid.NewString()
	l.contextID = newContextID
	l.state = MessageStateAssistantIdle
	return oldContextID, newContextID, nil
}

func (l *messageLifecycle) Mode() type_enums.MessageMode {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mode
}

func (l *messageLifecycle) SetMode(mode type_enums.MessageMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mode = mode
}

func (l *messageLifecycle) State() MessageState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

func (l *messageLifecycle) UserIdle(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateInterrupt, MessageStateUserIdle:
		l.state = MessageStateUserIdle
		return nil
	default:
		return fmt.Errorf("%w: user_idle from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserListening(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
		l.state = MessageStateUserListening
		return nil
	default:
		return fmt.Errorf("%w: user_listening from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserSpeaking(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
		l.state = MessageStateUserSpeaking
		return nil
	default:
		return fmt.Errorf("%w: user_speaking from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserThinking(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		l.state = MessageStateUserThinking
		return nil
	default:
		return fmt.Errorf("%w: user_thinking from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserFinished(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking, MessageStateUserFinished:
		l.state = MessageStateUserFinished
		return nil
	default:
		return fmt.Errorf("%w: user_finished from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserPrompted(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		l.state = MessageStateUserPrompted
		l.userPrompts++
		return nil
	default:
		return fmt.Errorf("%w: user_prompted from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) UserPromptCount() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.userPrompts
}

func (l *messageLifecycle) AssistantGenerating(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateAssistantFinished, MessageStateAssistantPrompted, MessageStateUserFinished, MessageStateUserPrompted, MessageStateAssistantGenerating:
		l.state = MessageStateAssistantGenerating
		return nil
	default:
		return fmt.Errorf("%w: assistant_generating from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantGenerated(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated:
		l.state = MessageStateAssistantGenerated
		return nil
	default:
		return fmt.Errorf("%w: assistant_generated from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantSpeaking(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking, MessageStateAssistantPrompted:
		l.state = MessageStateAssistantSpeaking
		return nil
	default:
		return fmt.Errorf("%w: assistant_speaking from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantFinished(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking, MessageStateAssistantFinished:
		l.state = MessageStateAssistantFinished
		return nil
	default:
		return fmt.Errorf("%w: assistant_finished from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantIdle(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantFinished, MessageStateAssistantIdle:
		l.state = MessageStateAssistantIdle
		return nil
	default:
		return fmt.Errorf("%w: assistant_idle from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantPrompted(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantIdle:
		l.state = MessageStateAssistantPrompted
		l.assistantPrompts++
		return nil
	default:
		return fmt.Errorf("%w: assistant_prompted from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) AssistantPromptCount() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.assistantPrompts
}

func (l *messageLifecycle) BeginInterrupt(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking, MessageStateAssistantPrompted:
		l.state = MessageStateInterrupt
		return nil
	default:
		return fmt.Errorf("%w: interrupt from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) CancelInterrupt(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	switch l.state {
	case MessageStateInterrupt, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		l.state = MessageStateAssistantSpeaking
		return nil
	default:
		return fmt.Errorf("%w: cancel_interrupt from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) validateContextLocked(contextID string) error {
	if contextID == "" {
		return ErrEmptyContextID
	}
	if contextID != l.contextID {
		return fmt.Errorf("%w: got %s, current %s", ErrStaleContext, contextID, l.contextID)
	}
	return nil
}
