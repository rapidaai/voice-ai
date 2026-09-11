// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

type MessageState string

const (
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
	ErrEmptyContextID              = errors.New("empty context id")
	ErrStaleContext                = errors.New("stale context")
	ErrInvalidTransition           = errors.New("invalid message lifecycle transition")
	ErrPlaybackTerminalNotIssued   = errors.New("playback terminal not issued")
	ErrDuplicatePlaybackCompletion = errors.New("duplicate playback completion")
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
	ObservePlaybackCompletion(string) error
	CanStartIdleTimeout(string) bool
	ConfigureInterruption(bool)
	InterruptionEnabled() bool
	CancelInterruption() string
	ObserveSpeech(internal_type.SpeechToTextPacket, bool) (*internal_type.TurnChangePacket, bool, bool)
	HoldInput(internal_type.Packet) bool
	ObserveVAD(internal_type.InterruptionDetectedPacket) (internal_type.InterruptionDetectedPacket, []internal_type.Packet, *internal_type.InterruptionDecisionExpiredPacket)
	ObserveInterruption(internal_type.InterruptionDetectedPacket, string) InterruptionDecision
	ArmInterruption(contextID string, sequence uint64, onExpired func(internal_type.InterruptionDecisionExpiredPacket))
	FailInterruptionPause(internal_type.InterruptionDecisionExpiredPacket) *internal_type.TurnChangePacket
	ExpireInterruption(internal_type.InterruptionDecisionExpiredPacket) (*internal_type.TurnChangePacket, string)
	BeginInterruptedTurn(internal_type.TurnChangePacket) bool
	CommitInterruptedTurn(internal_type.TurnChangePacket) (internal_type.TurnChangePacket, bool)
	FinishInterruptedTurn(internal_type.TurnChangePacket) []internal_type.Packet
	IsCommittedInterruption(string) bool
	AcceptUserTurn(string, string, string, string) (internal_type.TurnChangePacket, error)
	AcceptSpeechContext(string, string) (string, error)
	CompleteUserSpeech(internal_type.EndOfSpeechPacket) error
	CompleteAssistantSpeech(string) error
	Prompt(internal_type.Packet) (internal_type.TurnChangePacket, internal_type.InjectMessagePacket, error)
	ConfigureUnclearInput(context.Context, *internal_assistant_entity.AssistantDeploymentBehavior, func(context.Context, ...internal_type.Packet) error)
	StopUnclearInput()
	AcceptUserInput(internal_type.UserInputPacket) (internal_type.UserInputPacket, []internal_type.Packet)
	AssistantTextCompleted(internal_type.Packet) []internal_type.Packet
	SendPlaybackControl(proto.Message, func(proto.Message) error) error
	ConfigurePlaybackCompletion(func(...internal_type.Packet) error)
	SendAssistantMessage(*protos.ConversationAssistantMessage, func(proto.Message) error) error
	AcceptInjectedMessage(internal_type.InjectMessagePacket) (internal_type.InjectMessagePacket, error)
	FailAssistantMessage(string)
}

type messageLifecycle struct {
	playbackControlMu                  sync.Mutex
	mu                                 sync.RWMutex
	contextID                          string
	mode                               type_enums.MessageMode
	state                              MessageState
	userPrompts                        uint64
	assistantPrompts                   uint64
	output                             assistantOutputState
	onPlaybackPacket                   func(...internal_type.Packet) error
	interruptionEnabled                bool
	interruptionContextID              string
	interruptionPreviousState          MessageState
	interruptionSequence               uint64
	interruptionSpeechActive           bool
	interruptionResumed                bool
	interruptionDecisionPending        bool
	interruptionTurnCommitted          bool
	interruptionHeldPackets            []internal_type.Packet
	interruptionDecisionTimer          *time.Timer
	committedInterruptionContextID     string
	previousInterruptionContextID      string
	pendingInterruptionVADEndContextID string
	unclearInputWatchdog               *watchdog.UnclearInputWatchdog
	unclearInputTimeout                time.Duration
	unclearInputPrompt                 string
}

// Output state is reset with its owning message, never shared across message IDs.
type assistantOutputState struct {
	started          bool
	generationClosed bool
	textDelivered    bool
	hasText          bool
	hasAudio         bool
	terminalSending  bool
	terminalIssued   bool
	receiptReceived  bool
	completed        bool
	failed           bool
	paused           bool
	receiptDeadline  time.Time
	receiptRemaining time.Duration
	audioDuration    time.Duration
	receiptTimer     *time.Timer
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
		contextID:           initialContextID,
		mode:                initialMode,
		state:               MessageStateAssistantIdle,
		interruptionEnabled: InterruptionEnabledByDefault,
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
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	l.output = assistantOutputState{}
	if l.interruptionDecisionTimer != nil {
		l.interruptionDecisionTimer.Stop()
		l.interruptionDecisionTimer = nil
	}
	l.interruptionSequence++
	l.interruptionContextID = ""
	l.interruptionPreviousState = ""
	l.interruptionSpeechActive = false
	l.interruptionResumed = false
	l.interruptionDecisionPending = false
	l.interruptionTurnCommitted = false
	l.interruptionHeldPackets = nil
	l.committedInterruptionContextID = ""
	l.previousInterruptionContextID = ""
	l.pendingInterruptionVADEndContextID = ""
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
	return oldContextID, newContextID, nil
}

// ObservePlaybackCompletion holds receipts until final delivery and any pause resolve.
func (l *messageLifecycle) ObservePlaybackCompletion(contextID string) error {
	l.mu.Lock()
	if err := l.validateContextLocked(contextID); err != nil {
		l.mu.Unlock()
		return err
	}
	if l.output.failed || (!l.output.terminalIssued && !l.output.terminalSending) {
		l.mu.Unlock()
		return ErrPlaybackTerminalNotIssued
	}
	if l.output.receiptReceived {
		l.mu.Unlock()
		return ErrDuplicatePlaybackCompletion
	}
	l.output.receiptReceived = true
	l.mu.Unlock()
	_ = l.CompleteAssistantSpeech(contextID)
	return nil
}

func (l *messageLifecycle) Mode() type_enums.MessageMode {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.mode
}

func (l *messageLifecycle) SetMode(mode type_enums.MessageMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if mode != l.mode && l.output.started {
		l.output.failed = true
		if l.output.receiptTimer != nil {
			l.output.receiptTimer.Stop()
		}
	}
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
	case MessageStateAssistantIdle, MessageStateUserIdle:
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
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
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
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
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
	case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
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
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserThinking, MessageStateUserFinished:
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
	case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
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
	if l.output.generationClosed || l.output.failed || l.output.completed {
		return ErrInvalidTransition
	}
	if l.state == MessageStateAssistantSpeaking {
		l.output.started = true
		return nil
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateAssistantFinished, MessageStateAssistantPrompted, MessageStateUserFinished, MessageStateUserPrompted, MessageStateAssistantGenerating:
		l.output.started = true
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
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking:
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

func (l *messageLifecycle) validateContextLocked(contextID string) error {
	if contextID == "" {
		return ErrEmptyContextID
	}
	if contextID != l.contextID {
		return fmt.Errorf("%w: got %s, current %s", ErrStaleContext, contextID, l.contextID)
	}
	return nil
}
