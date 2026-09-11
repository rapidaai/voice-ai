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

// AcceptUserTurn preserves a listening turn or atomically allocates its replacement.
func (l *messageLifecycle) AcceptUserTurn(contextID, trigger, source, text string) (internal_type.TurnChangePacket, error) {
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
	turn.ContextID = l.contextID
	return turn, nil
}

func (l *messageLifecycle) AcceptSpeechContext(contextID, text string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(contextID); err != nil {
		return "", err
	}
	if strings.TrimSpace(text) != "" {
		switch l.state {
		case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening:
			l.state = MessageStateUserListening
		}
		if !l.interruptionEnabled && l.unclearInputWatchdog != nil {
			l.unclearInputWatchdog.Stop()
		}
	}
	return l.contextID, nil
}

func (l *messageLifecycle) CompleteUserSpeech(p internal_type.EndOfSpeechPacket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(p.ContextID); err != nil {
		return err
	}
	if strings.TrimSpace(p.Speech) == "" {
		return nil
	}
	l.committedInterruptionContextID = ""
	l.previousInterruptionContextID = ""
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

func (l *messageLifecycle) CompleteAssistantSpeech(contextID string) error {
	l.mu.Lock()
	if err := l.validateContextLocked(contextID); err != nil {
		l.mu.Unlock()
		return err
	}
	if l.output.failed || l.output.completed || !l.output.generationClosed || !l.output.textDelivered || l.output.paused || l.output.terminalSending || l.interruptionContextID != "" {
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	if l.mode.Audio() && (l.output.hasText || l.output.hasAudio) && (!l.output.hasAudio || !l.output.terminalIssued || !l.output.receiptReceived) {
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	l.output.completed = true
	l.state = MessageStateAssistantIdle
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	onPacket := l.onPlaybackPacket
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

func (l *messageLifecycle) Prompt(packet internal_type.Packet) (internal_type.TurnChangePacket, internal_type.InjectMessagePacket, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	turn := internal_type.TurnChangePacket{}
	prompt := internal_type.InjectMessagePacket{}
	if l.interruptionContextID != "" {
		return turn, prompt, ErrInvalidTransition
	}
	switch p := packet.(type) {
	case internal_type.IdleTimeoutExpiredPacket:
		if p.ContextID != l.contextID || l.state != MessageStateAssistantIdle {
			return turn, prompt, ErrInvalidTransition
		}
		l.assistantPrompts++
	case internal_type.UnclearInputExpiredPacket:
		if p.ContextID != l.contextID {
			return turn, prompt, ErrStaleContext
		}
		if strings.TrimSpace(l.unclearInputPrompt) == "" || (l.interruptionEnabled &&
			(l.committedInterruptionContextID != p.ContextID || l.unclearInputWatchdog == nil || !l.unclearInputWatchdog.AcceptExpiry(p))) {
			return turn, prompt, ErrInvalidTransition
		}
		switch l.state {
		case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
		default:
			return turn, prompt, ErrInvalidTransition
		}
		l.userPrompts++
		prompt.Text = l.unclearInputPrompt
	default:
		return turn, prompt, ErrInvalidTransition
	}
	turn.PreviousContextID = l.contextID
	l.contextID = uuid.NewString()
	l.state = MessageStateAssistantIdle
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	l.output = assistantOutputState{}
	l.interruptionResumed = false
	l.interruptionSpeechActive = false
	l.interruptionHeldPackets = nil
	l.committedInterruptionContextID = ""
	l.previousInterruptionContextID = ""
	turn.ContextID = l.contextID
	turn.Reason = "interrupted"
	turn.Source = "requestor"
	turn.Time = time.Now()
	prompt.ContextID = l.contextID
	return turn, prompt, nil
}

func (l *messageLifecycle) ConfigureUnclearInput(ctx context.Context, behavior *internal_assistant_entity.AssistantDeploymentBehavior, onPacket func(context.Context, ...internal_type.Packet) error) {
	inputWatchdog := watchdog.NewUnclearInputWatchdog(watchdog.WithPacketContext(ctx), watchdog.WithOnPacket(onPacket))
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
}

func (l *messageLifecycle) StopUnclearInput() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Stop()
	}
}

func (l *messageLifecycle) AcceptUserInput(p internal_type.UserInputPacket) (internal_type.UserInputPacket, []internal_type.Packet) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.TrimSpace(p.Text) == "" {
		return internal_type.UserInputPacket{}, nil
	}
	if p.ContextID == "" {
		p.ContextID = l.contextID
	}
	if l.interruptionEnabled && l.interruptionContextID != "" {
		if p.ContextID == l.interruptionContextID || p.ContextID == l.contextID {
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
		}
		return internal_type.UserInputPacket{}, nil
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
	l.committedInterruptionContextID = ""
	l.previousInterruptionContextID = ""
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

func (l *messageLifecycle) AssistantTextCompleted(packet internal_type.Packet) []internal_type.Packet {
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
	if contextID != l.contextID || l.output.generationClosed || l.output.failed {
		return nil
	}
	l.output.generationClosed = true
	l.output.hasText = l.output.hasText || strings.TrimSpace(text) != ""
	return []internal_type.Packet{
		internal_type.MessageCreatePacket{ContextID: contextID, MessageRole: "assistant", Text: text},
	}
}
