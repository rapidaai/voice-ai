package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/watchdog"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

// OnUserTurnStarted preserves a listening turn or atomically allocates its replacement.
func (l *messageLifecycle) OnUserTurnStarted(contextID, trigger, source, text string) (internal_type.TurnChangePacket, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	turn := internal_type.TurnChangePacket{ContextID: l.contextID}
	switch l.state {
	case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		return turn, nil
	}
	if contextID != "" && contextID != l.contextID {
		return turn, ErrStaleContext
	}
	turn.PreviousContextID = l.contextID
	turn.PreviousState = string(l.state)
	turn.Reason = "interrupted"
	turn.Source = source
	turn.Trigger = trigger
	turn.Text = text
	turn.Time = time.Now()
	l.contextID = uuid.NewString()
	l.state = MessageStateAssistantIdle
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	l.output = assistantOutputState{}
	if l.interruption.timer != nil {
		l.interruption.timer.Stop()
		l.interruption.timer = nil
	}
	l.interruption = interruptionState{sequence: l.interruption.sequence + 1}
	l.previousContextID = ""
	l.pendingVADEndContextID = ""
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
	turn.ContextID = l.contextID
	return turn, nil
}

// OnTranscriptReceived admits speech into the listening turn or starts its successor.
func (l *messageLifecycle) OnTranscriptReceived(packet internal_type.SpeechToTextPacket) (internal_type.TurnChangePacket, error) {
	l.mu.Lock()
	turn := internal_type.TurnChangePacket{ContextID: l.contextID}
	if err := l.validateContextLocked(packet.ContextID); err != nil {
		l.mu.Unlock()
		return internal_type.TurnChangePacket{}, err
	}
	if strings.TrimSpace(packet.Script) == "" {
		l.mu.Unlock()
		return turn, nil
	}
	if l.output.playback == playbackCompleted || l.state == MessageStateUserFinished || l.state == MessageStateAssistantPrompted {
		turn = l.startSpeechTurnLocked()
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening:
		l.state = MessageStateUserListening
	case MessageStateUserSpeaking, MessageStateUserThinking:
	default:
		l.mu.Unlock()
		return internal_type.TurnChangePacket{}, ErrInvalidTransition
	}
	if !packet.Interim {
		l.previousContextID = ""
		if l.unclearInputWatchdog != nil {
			l.unclearInputWatchdog.Stop()
		}
	} else if l.mode.Audio() && l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Start(turn.ContextID, l.unclearInputTimeout)
	}
	onPacket := l.onPacket
	l.mu.Unlock()
	if onPacket != nil {
		return turn, onPacket(internal_type.StopIdleTimeoutPacket{ContextID: turn.ContextID, ResetCount: true})
	}
	return turn, nil
}

// Speech admission and replay share this transition while holding the message lock.
func (l *messageLifecycle) startSpeechTurnLocked() internal_type.TurnChangePacket {
	turn := internal_type.TurnChangePacket{
		PreviousContextID: l.contextID, PreviousState: string(l.state),
		Reason: "user_input", Source: "stt", Time: time.Now(),
	}
	l.pendingVADEndContextID = l.contextID
	l.previousContextID = l.contextID
	l.contextID = uuid.NewString()
	l.output = assistantOutputState{}
	l.state = MessageStateUserListening
	if l.interruption.contextID == "" {
		l.interruption = interruptionState{sequence: l.interruption.sequence}
	}
	turn.ContextID = l.contextID
	return turn
}

func (l *messageLifecycle) OnUserSpeechCompleted(p internal_type.EndOfSpeechPacket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(p.ContextID); err != nil {
		return err
	}
	if strings.TrimSpace(p.Speech) == "" {
		return nil
	}
	l.previousContextID = ""
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking,
		MessageStateUserThinking, MessageStateUserFinished:
		l.state = MessageStateUserFinished
		return nil
	default:
		return fmt.Errorf("%w: user_finished from %s", ErrInvalidTransition, l.state)
	}
}

func (l *messageLifecycle) completeAssistantMessage(contextID string) error {
	l.mu.Lock()
	if err := l.validateContextLocked(contextID); err != nil {
		l.mu.Unlock()
		return err
	}
	switch l.output.playback {
	case playbackFailed, playbackCompleted, playbackClosing:
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	if l.output.generation != generationCompleted || !l.output.textDelivered || l.output.paused || l.interruption.contextID != "" {
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	if l.mode.Audio() && (l.output.hasText || l.output.hasAudio) && (!l.output.hasAudio || l.output.playback != playbackAwaitingReceipt || !l.output.receiptReceived) {
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	l.output.playback = playbackCompleted
	l.state = MessageStateAssistantIdle
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	onPacket := l.onPacket
	l.mu.Unlock()
	if onPacket == nil {
		return nil
	}
	return onPacket(
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: contextID, Scope: internal_type.ObservabilityRecordScopeAssistantMessage,
			Record: observability.NewMessageMetricRecord(contextID, observability.MessageRoleAssistant, []*protos.Metric{{
				Name: "assistant_turn", Value: type_enums.CONVERSATION_COMPLETE.String(), Description: "Assistant message delivery completed",
			}}),
		},
		internal_type.StartIdleTimeoutPacket{ContextID: contextID},
	)
}

func (l *messageLifecycle) OnPrompt(packet internal_type.Packet) (internal_type.TurnChangePacket, internal_type.InjectMessagePacket, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	turn := internal_type.TurnChangePacket{}
	prompt := internal_type.InjectMessagePacket{}
	if l.interruption.contextID != "" {
		if expired, ok := packet.(internal_type.UnclearInputExpiredPacket); ok && l.interruption.phase == interruptionDraining &&
			expired.ContextID == l.contextID && l.unclearInputWatchdog != nil && l.unclearInputWatchdog.AcceptExpiry(expired) {
			l.interruption.heldPackets = append(l.interruption.heldPackets, expired)
		}
		return turn, prompt, ErrInvalidTransition
	}
	switch p := packet.(type) {
	case internal_type.IdleTimeoutExpiredPacket:
		if p.ContextID != l.contextID || l.state != MessageStateAssistantIdle {
			return turn, prompt, ErrInvalidTransition
		}
	case internal_type.UnclearInputExpiredPacket:
		if p.ContextID != l.contextID {
			return turn, prompt, ErrStaleContext
		}
		if strings.TrimSpace(l.unclearInputPrompt) == "" || l.unclearInputWatchdog == nil || !l.unclearInputWatchdog.AcceptExpiry(p) {
			return turn, prompt, ErrInvalidTransition
		}
		switch l.state {
		case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		default:
			return turn, prompt, ErrInvalidTransition
		}
		prompt.Text = l.unclearInputPrompt
	default:
		return turn, prompt, ErrInvalidTransition
	}
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
	turn.PreviousContextID = l.contextID
	l.contextID = uuid.NewString()
	l.state = MessageStateAssistantPrompted
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	l.output = assistantOutputState{}
	l.interruption = interruptionState{sequence: l.interruption.sequence}
	l.previousContextID = ""
	turn.ContextID = l.contextID
	turn.Reason = "interrupted"
	turn.Source = "requestor"
	turn.Time = time.Now()
	prompt.ContextID = l.contextID
	return turn, prompt, nil
}

// Initialize loads behavior after deployment configuration is available.
func (l *messageLifecycle) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var behavior *internal_assistant_entity.AssistantDeploymentBehavior
	if l.loadBehavior != nil {
		var err error
		behavior, err = l.loadBehavior()
		if err != nil {
			return fmt.Errorf("load message lifecycle behavior: %w", err)
		}
	}
	inputWatchdog := watchdog.NewUnclearInputWatchdog(watchdog.WithPacketContext(ctx), watchdog.WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		if l.onPacket == nil {
			return nil
		}
		return l.onPacket(packets...)
	}))
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Cancel()
	}
	l.unclearInputWatchdog = inputWatchdog
	l.unclearInputTimeout = 0
	l.unclearInputPrompt = ""
	if behavior != nil {
		if behavior.UnclearInputTimeout != nil {
			l.unclearInputTimeout = time.Duration(*behavior.UnclearInputTimeout * float64(time.Second))
		}
		if behavior.UnclearInputMessage != nil {
			l.unclearInputPrompt = *behavior.UnclearInputMessage
		}
	}
	return nil
}

func (l *messageLifecycle) StopUnclearInput() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
}

// OnUserInput records processed user input after it passes admission.
func (l *messageLifecycle) OnUserInput(p internal_type.UserInputPacket) (internal_type.UserInputPacket, []internal_type.Packet) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.TrimSpace(p.Text) == "" {
		return internal_type.UserInputPacket{}, nil
	}
	if p.ContextID == "" {
		p.ContextID = l.contextID
	}
	if p.ContextID != l.contextID {
		return internal_type.UserInputPacket{}, nil
	}
	switch l.state {
	case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserThinking, MessageStateUserFinished:
	default:
		return internal_type.UserInputPacket{}, nil
	}
	l.state = MessageStateUserFinished
	l.previousContextID = ""
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
	return p, []internal_type.Packet{
		internal_type.StopIdleTimeoutPacket{ContextID: p.ContextID, ResetCount: true},
		internal_type.MessageCreatePacket{ContextID: p.ContextID, MessageRole: "user", Text: p.Text},
		internal_type.ObservabilityMetadataRecordPacket{
			ContextID: p.ContextID, Scope: internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.NewMessageMetadataRecord(p.ContextID, observability.MessageRoleUser, []*protos.Metadata{
				{Key: "language", Value: p.Language.Name}, {Key: "language_code", Value: p.Language.ISO639_1},
			}),
		},
		internal_type.ObservabilityMetricRecordPacket{
			ContextID: p.ContextID, Scope: internal_type.ObservabilityRecordScopeUserMessage,
			Record: observability.NewMessageMetricRecord(p.ContextID, observability.MessageRoleUser, []*protos.Metric{{
				Name: "user_turn", Value: type_enums.CONVERSATION_COMPLETE.String(), Description: "User turn completed and ready for assistant response generation",
			}}),
		},
	}
}

// OnGenerationCompleted closes generation without completing outstanding delivery.
func (l *messageLifecycle) OnGenerationCompleted(packet internal_type.Packet) []internal_type.Packet {
	l.mu.Lock()
	defer l.mu.Unlock()
	contextID, text := "", ""
	switch p := packet.(type) {
	case internal_type.LLMResponseDonePacket:
		contextID, text = p.ContextID, p.Text
	case internal_type.InjectMessagePacket:
		if p.Interim {
			return nil
		}
		contextID, text = p.ContextID, p.Text
	default:
		return nil
	}
	if contextID != l.contextID || l.output.generation == generationCompleted || l.output.playback == playbackFailed {
		return nil
	}
	l.output.generation = generationCompleted
	l.output.hasText = l.output.hasText || strings.TrimSpace(text) != ""
	if l.state == MessageStateAssistantGenerating {
		l.state = MessageStateAssistantGenerated
	}
	return []internal_type.Packet{
		internal_type.MessageCreatePacket{ContextID: contextID, MessageRole: "assistant", Text: text},
	}
}
