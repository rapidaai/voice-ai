package adapter_internal

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (h requestorDispatchHandler) HandleInterruptionDetected(ctx context.Context, p internal_type.InterruptionDetectedPacket) {
	bargeInTrigger := internal_options.BargeInTriggerVAD
	if opts := h.r.GetOptions(); len(opts) > 0 {
		if value, err := opts.GetString(internal_options.MicrophoneOptionBargeInTrigger); err == nil {
			bargeInTrigger = value
		} else if value, err := opts.GetString(internal_options.MicrophoneLegacyVADOptionBargeInTrigger); err == nil {
			bargeInTrigger = value
		}
	}
	decision := h.r.messageLifecycle.OnInterruptionDetected(p, bargeInTrigger)
	if pause := decision.Pause; pause != nil {
		pauseError := h.r.sendOutputControl(&protos.ConversationPlaybackPause{Id: pause.ContextID})
		if pauseError != nil {
			h.r.OnPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: pause.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Interruption output control failed",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"error":     pauseError.Error(),
					},
				},
			})
		}
		if turnChange := h.r.messageLifecycle.OnPlaybackPaused(*pause, pauseError); turnChange != nil {
			utils.Go(ctx, func() { h.r.dispatch(ctx, *turnChange) })
		}
		if pauseError != nil {
			return
		}
	}
	h.r.OnPacket(ctx, decision.Packets...)
	if decision.Flush != nil {
		if outputControlError := h.r.sendOutputControl(decision.Flush); outputControlError != nil && h.r.logger != nil {
			h.r.logger.Errorf("error while flushing interrupted output %v", outputControlError)
		}
		h.r.Notify(ctx, decision.Notification)
	}
	if decision.EndOfSpeech != nil && h.r.endOfSpeechExecutor != nil {
		_ = h.r.endOfSpeechExecutor.Execute(ctx, *decision.EndOfSpeech)
	}
	if decision.SpeechToTextStart != nil {
		h.r.OnPacket(ctx, *decision.SpeechToTextStart)
	}
}
