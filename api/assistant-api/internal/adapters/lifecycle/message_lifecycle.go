// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"context"
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

// MessageLifecycle owns message admission, interruption decisions, and delivery completion.
type MessageLifecycle interface {
	ContextID() string
	Mode() type_enums.MessageMode
	SetMode(mode type_enums.MessageMode)
	State() MessageState
	CanStartIdleTimeout(contextID string) bool

	Initialize(ctx context.Context) error
	StopUnclearInput()

	OnUserTurnStarted(contextID, trigger, source, text string) (internal_type.TurnChangePacket, error)
	OnTranscriptReceived(packet internal_type.SpeechToTextPacket) (internal_type.TurnChangePacket, error)
	OnUserInput(packet internal_type.UserInputPacket) (internal_type.UserInputPacket, []internal_type.Packet)
	OnUserSpeechCompleted(packet internal_type.EndOfSpeechPacket) error
	OnPrompt(packet internal_type.Packet) (internal_type.TurnChangePacket, internal_type.InjectMessagePacket, error)

	OnGenerationStarted(contextID string) error
	OnSpeechStarted(contextID string) error
	OnGenerationCompleted(packet internal_type.Packet) []internal_type.Packet
	OnMessageInjected(packet internal_type.InjectMessagePacket) (internal_type.InjectMessagePacket, error)
	SendAssistantMessage(message *protos.ConversationAssistantMessage) error
	SendPlaybackControl(control proto.Message) error
	OnPlaybackCompleted(contextID string) error
	OnMessageFailed(contextID string)
	Close(contextID string) *protos.ConversationPlaybackControl

	CancelInterruption() string
	OnUserSpeech(packet internal_type.SpeechToTextPacket) (*internal_type.TurnChangePacket, internal_type.SpeechToTextPacket)
	HoldInput(packet internal_type.Packet) bool
	OnInterruptionDetected(packet internal_type.InterruptionDetectedPacket, bargeInTrigger string) InterruptionDecision
	OnPlaybackPaused(packet internal_type.InterruptionDecisionExpiredPacket, pauseError error) *internal_type.TurnChangePacket
	OnInterruptionExpired(packet internal_type.InterruptionDecisionExpiredPacket) string
	OnTurnChange(ctx context.Context, packet internal_type.TurnChangePacket) error
}

type messageLifecycle struct {
	playbackControlMu      sync.Mutex
	mu                     sync.RWMutex
	contextID              string
	mode                   type_enums.MessageMode
	state                  MessageState
	output                 assistantOutputState
	onPacket               func(...internal_type.Packet) error
	sendOutput             func(proto.Message) error
	dispatchPacket         func(context.Context, internal_type.Packet)
	onInterruptionExpired  func(internal_type.InterruptionDecisionExpiredPacket)
	loadBehavior           func() (*internal_assistant_entity.AssistantDeploymentBehavior, error)
	interruption           interruptionState
	previousContextID      string
	pendingVADEndContextID string
	unclearInputWatchdog   *watchdog.UnclearInputWatchdog
	unclearInputTimeout    time.Duration
	unclearInputPrompt     string
}

// Output state is reset with its owning message, never shared across message IDs.
type assistantOutputState struct {
	generation       generationPhase
	playback         playbackPhase
	textDelivered    bool
	hasText          bool
	hasAudio         bool
	receiptReceived  bool
	paused           bool
	receiptDeadline  time.Time
	receiptRemaining time.Duration
	audioDuration    time.Duration
	receiptTimer     *time.Timer
}

// NewMessageLifecycle defaults to text mode and confirms audio interruptions with speech.
func NewMessageLifecycle(options ...MessageOption) MessageLifecycle {
	message := &messageLifecycle{
		mode:  type_enums.TextMode,
		state: MessageStateAssistantIdle,
	}
	for _, option := range options {
		if option != nil {
			option(message)
		}
	}
	if message.contextID == "" {
		message.contextID = uuid.NewString()
	}
	if message.mode == "" {
		message.mode = type_enums.TextMode
	}
	return message
}

func (l *messageLifecycle) ContextID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.contextID
}

// OnPlaybackCompleted holds receipts until final delivery and any pause resolve.
func (l *messageLifecycle) OnPlaybackCompleted(contextID string) error {
	l.mu.Lock()
	if err := l.validateContextLocked(contextID); err != nil {
		l.mu.Unlock()
		return err
	}
	if l.output.receiptReceived && l.output.playback != playbackFailed {
		l.mu.Unlock()
		return ErrDuplicatePlaybackCompletion
	}
	if !l.mode.Audio() || (l.output.playback != playbackClosing && l.output.playback != playbackAwaitingReceipt) {
		l.mu.Unlock()
		return ErrPlaybackTerminalNotIssued
	}
	l.output.receiptReceived = true
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
		l.output.receiptTimer = nil
	}
	l.mu.Unlock()
	_ = l.completeAssistantMessage(contextID)
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
	if mode != l.mode && l.output.generation != generationIdle {
		l.output.playback = playbackFailed
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

// OnGenerationStarted admits generation only while the response remains open.
func (l *messageLifecycle) OnGenerationStarted(contextID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return err
	}
	if l.output.generation == generationCompleted || l.output.playback == playbackFailed || l.output.playback == playbackCompleted {
		return ErrInvalidTransition
	}
	if l.state == MessageStateAssistantSpeaking {
		l.output.generation = generationStarted
		return nil
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateAssistantFinished, MessageStateAssistantPrompted, MessageStateUserFinished, MessageStateUserPrompted, MessageStateAssistantGenerating:
		l.output.generation = generationStarted
		l.state = MessageStateAssistantGenerating
		return nil
	default:
		return fmt.Errorf("%w: assistant_generating from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) OnSpeechStarted(contextID string) error {
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

// Close cancels message work; the caller applies any returned playback control.
func (l *messageLifecycle) Close(contextID string) *protos.ConversationPlaybackControl {
	l.OnMessageFailed(contextID)
	defer l.StopUnclearInput()
	if interruptedContextID := l.CancelInterruption(); interruptedContextID != "" {
		return &protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE, Id: interruptedContextID}
	}
	return nil
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
