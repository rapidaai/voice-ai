package lifecycle

import (
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InterruptionDecision orders packets before EOS and a speech start after EOS.
// Playback controls are executed by the dispatcher.
type InterruptionDecision struct {
	Packets           []internal_type.Packet
	EndOfSpeech       *internal_type.InterruptionDetectedPacket
	SpeechToTextStart *internal_type.SpeechToTextStartPacket
	Pause             *internal_type.InterruptionDecisionExpiredPacket
	Flush             *protos.ConversationPlaybackFlush
	Notification      *protos.ConversationInterruption
}

// OnInterruptionDetected owns VAD and word admission, turn transitions, and countdown eligibility.
func (l *messageLifecycle) OnInterruptionDetected(
	p internal_type.InterruptionDetectedPacket,
	bargeInTrigger string,
) InterruptionDecision {
	l.mu.Lock()
	if l.interruptionEnabled && l.mode.Audio() && p.Source == internal_type.InterruptionSourceVad && bargeInTrigger != internal_options.BargeInTriggerWord {
		l.mu.Unlock()
		admitted, packets, pause := l.observeVAD(p)
		decision := InterruptionDecision{Packets: packets, Pause: pause}
		if pause != nil {
			decision.SpeechToTextStart = &internal_type.SpeechToTextStartPacket{ContextID: pause.ContextID}
		}
		if admitted.ContextID != "" {
			decision.EndOfSpeech = &admitted
		}
		return decision
	}
	defer l.mu.Unlock()

	decision := InterruptionDecision{}
	if p.ContextID == "" {
		p.ContextID = l.contextID
	}
	if p.ContextID != l.contextID {
		switch p.Source {
		case internal_type.InterruptionSourceWord:
			switch l.state {
			case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
				p.ContextID = l.contextID
			default:
				return decision
			}
		case internal_type.InterruptionSourceVad:
			if p.Event != internal_type.InterruptionEventEnd {
				return decision
			}
			switch l.state {
			case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
				p.ContextID = l.contextID
			default:
				return decision
			}
		default:
			return decision
		}
	}

	previousState := l.state
	shouldRotate := false
	shouldStartUnclear := false
	shouldExtendUnclear := false
	switch p.Source {
	case internal_type.InterruptionSourceVad:
		switch p.Event {
		case internal_type.InterruptionEventStart:
			if bargeInTrigger != internal_options.BargeInTriggerWord {
				switch l.state {
				case MessageStateUserIdle:
					l.state = MessageStateUserSpeaking
				case MessageStateUserListening, MessageStateUserSpeaking, MessageStateUserThinking:
				default:
					shouldRotate = true
				}
			}
			decision.EndOfSpeech = &p
		case internal_type.InterruptionEventEnd:
			decision.Packets = append(decision.Packets, internal_type.SpeechToTextEndPacket{ContextID: p.ContextID})
			if bargeInTrigger == internal_options.BargeInTriggerWord {
				decision.EndOfSpeech = &p
				return decision
			}
			if l.state == MessageStateUserSpeaking {
				l.state = MessageStateUserListening
			}
			if l.state == MessageStateUserListening {
				decision.EndOfSpeech = &p
				shouldStartUnclear = true
			}
		default:
			return decision
		}
	case internal_type.InterruptionSourceWord:
		if !l.mode.Text() && bargeInTrigger != internal_options.BargeInTriggerWord {
			return decision
		}
		switch l.state {
		case MessageStateUserIdle:
			if l.mode.Text() {
				l.state = MessageStateUserListening
			}
		case MessageStateUserListening, MessageStateUserSpeaking:
			if l.mode.Text() {
				l.state = MessageStateUserListening
			} else {
				shouldExtendUnclear = true
			}
		case MessageStateUserThinking:
			shouldExtendUnclear = !l.mode.Text()
		default:
			shouldRotate = true
			shouldStartUnclear = !l.mode.Text()
		}
	default:
		return decision
	}

	if shouldRotate {
		oldContextID := l.contextID
		l.contextID = uuid.NewString()
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
		l.state = MessageStateUserListening
		interruptionType := protos.ConversationInterruption_INTERRUPTION_TYPE_WORD
		if p.Source == internal_type.InterruptionSourceVad {
			l.state = MessageStateUserSpeaking
			interruptionType = protos.ConversationInterruption_INTERRUPTION_TYPE_VAD
		}
		p.ContextID = l.contextID
		now := time.Now()
		decision.Packets = append(decision.Packets,
			internal_type.StopIdleTimeoutPacket{ContextID: oldContextID},
			internal_type.EndOfSpeechInterruptionPacket{ContextID: oldContextID, Source: p.Source},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: oldContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component: observability.ComponentTurn, Event: observability.TurnInterrupted, OccurredAt: now,
					Attributes: observability.Attributes{
						"old_context_id": oldContextID, "new_context_id": l.contextID,
						"reason": "interrupted", "source": string(p.Source), "mode": l.mode.String(),
						"state": string(previousState), "trigger": string(p.PacketName()),
					},
				},
			},
			internal_type.TurnChangePacket{
				ContextID: l.contextID, PreviousContextID: oldContextID,
				Reason: "interrupted", Source: string(p.Source), PreviousState: string(previousState),
				Trigger: string(p.PacketName()), Time: now,
			},
			internal_type.ObservabilityEventRecordPacket{
				ContextID: l.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component: observability.ComponentTurn, Event: observability.TurnStarted, OccurredAt: now,
					Attributes: observability.Attributes{
						"context_id": l.contextID, "previous_context_id": oldContextID,
						"reason": "interruption", "source": string(p.Source), "mode": l.mode.String(),
						"previous_state": string(previousState), "trigger": string(p.PacketName()),
					},
				},
			},
			internal_type.TextToSpeechInterruptPacket{ContextID: oldContextID},
			internal_type.LLMInterruptPacket{ContextID: oldContextID},
		)
		decision.Flush = &protos.ConversationPlaybackFlush{Id: oldContextID}
		decision.Notification = &protos.ConversationInterruption{Type: interruptionType, Time: timestamppb.New(now)}
	}
	if p.Source == internal_type.InterruptionSourceVad && p.Event == internal_type.InterruptionEventStart {
		decision.SpeechToTextStart = &internal_type.SpeechToTextStartPacket{ContextID: p.ContextID}
	}
	if l.unclearInputWatchdog != nil && l.unclearInputTimeout > 0 {
		if shouldStartUnclear {
			l.unclearInputWatchdog.Start(p.ContextID, l.unclearInputTimeout)
		} else if shouldExtendUnclear {
			l.unclearInputWatchdog.Extend(p.ContextID, l.unclearInputTimeout)
		}
	}
	return decision
}
