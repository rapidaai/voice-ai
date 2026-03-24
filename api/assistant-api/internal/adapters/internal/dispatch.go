// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	observe "github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// packetEnvelope carries a packet together with the context it was sent from.
type packetEnvelope struct {
	ctx context.Context
	pkt internal_type.Packet
}

// =============================================================================
// Shared helpers — called by multiple handlers
// =============================================================================

func (talking *genericRequestor) callEndOfSpeech(ctx context.Context, vl internal_type.Packet) error {
	if talking.endOfSpeech != nil {
		if err := talking.endOfSpeech.Analyze(ctx, vl); err != nil {
			talking.logger.Errorf("end of speech analyze error: %v", err)
		}
		return nil
	}
	return errors.New("end of speech analyzer not configured")
}

func (talking *genericRequestor) callTextAggregator(ctx context.Context, vl internal_type.Packet) error {
	if talking.textAggregator != nil {
		if err := talking.textAggregator.Aggregate(ctx, vl); err != nil {
			talking.logger.Debugf("unable to send packet to aggregator %v", err)
		}
		return nil
	}
	return errors.New("textAggregator not configured")
}

func (talking *genericRequestor) callSpeechToText(ctx context.Context, vl internal_type.UserAudioPacket) error {
	if talking.speechToTextTransformer != nil {
		utils.Go(ctx, func() {
			if err := talking.speechToTextTransformer.Transform(ctx, vl); err != nil {
				talking.logger.Tracef(ctx, "error while transforming input %s and error %s", talking.speechToTextTransformer.Name(), err.Error())
			}
		})
	}
	return nil
}

// =============================================================================
// OnPacket — enqueue into the priority channel
// =============================================================================

// OnPacket enqueues each packet into the appropriate priority channel.
// Each channel has a dedicated dispatcher goroutine so no tier can stall another.
//
// Priority:
//
//	critical — interrupts, directives                        (preempts everything)
//	input    — inbound audio, denoise, VAD, STT, EOS         (user → system)
//	output   — LLM generation, text aggregation, TTS          (system → user)
//	low      — recording, metrics, persistence, events         (background work)
func (r *genericRequestor) OnPacket(ctx context.Context, pkts ...internal_type.Packet) error {
	for _, p := range pkts {
		e := packetEnvelope{ctx: ctx, pkt: p}
		switch p.(type) {
		// Critical — interrupts and directives
		case internal_type.InterruptionPacket,
			internal_type.InterruptTTSPacket,
			internal_type.InterruptLLMPacket,
			internal_type.DirectivePacket:
			r.criticalCh <- e

		// Input — inbound audio pipeline, VAD, STT, EOS
		case internal_type.UserAudioPacket,
			internal_type.UserTextPacket,
			internal_type.DenoiseAudioPacket,
			internal_type.DenoisedAudioPacket,
			internal_type.VadAudioPacket,
			internal_type.VadSpeechActivityPacket,
			internal_type.SpeechToTextPacket,
			internal_type.EndOfSpeechPacket,
			internal_type.InterimEndOfSpeechPacket:
			r.inputCh <- e

		// Output — LLM generation, TTS, outbound pipeline
		case internal_type.ExecuteLLMPacket,
			internal_type.LLMResponseDeltaPacket,
			internal_type.LLMResponseDonePacket,
			internal_type.LLMErrorPacket,
			internal_type.SpeakTextPacket,
			internal_type.TextToSpeechAudioPacket,
			internal_type.TextToSpeechEndPacket,
			internal_type.StaticPacket:
			r.outputCh <- e

		// Low — recording, metrics, persistence, events, tools
		case internal_type.RecordUserAudioPacket,
			internal_type.RecordAssistantAudioPacket,
			internal_type.SaveMessagePacket,
			internal_type.ConversationMetricPacket,
			internal_type.ConversationMetadataPacket,
			internal_type.MessageMetricPacket,
			internal_type.MessageMetadataPacket,
			internal_type.LLMToolCallPacket,
			internal_type.LLMToolResultPacket,
			internal_type.ConversationEventPacket:
			r.lowCh <- e

		default:
			r.logger.Warnf("OnPacket: unrouted packet type %T, falling back to inputCh", p)
			r.inputCh <- e
		}
	}
	return nil
}

// =============================================================================
// runDispatcher — dedicated consumer goroutines per priority tier
// =============================================================================

// runCriticalDispatcher processes critical-priority packets (interrupts,
// directives) on a dedicated goroutine so they are never queued behind
// normal or low-priority work.
func (r *genericRequestor) runCriticalDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainCriticalChannel()
			return
		case e := <-r.criticalCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainCriticalChannel() {
	for {
		select {
		case e := <-r.criticalCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runInputDispatcher processes inbound pipeline packets (user audio, denoise,
// VAD, STT, EOS) on a dedicated goroutine. Kept separate from the output
// pipeline so a burst of inbound audio never delays LLM/TTS streaming.
func (r *genericRequestor) runInputDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainInputChannel()
			return
		case e := <-r.inputCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainInputChannel() {
	for {
		select {
		case e := <-r.inputCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runOutputDispatcher processes outbound pipeline packets (LLM generation,
// text aggregation, TTS audio) on a dedicated goroutine. Kept separate from
// the input pipeline so LLM response streaming is never queued behind audio.
func (r *genericRequestor) runOutputDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.drainOutputChannel()
			return
		case e := <-r.outputCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainOutputChannel() {
	for {
		select {
		case e := <-r.outputCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// runLowDispatcher processes low-priority packets (persistence, metrics,
// events, tools) on a dedicated goroutine so DB writes never stall audio
// or LLM processing.
func (r *genericRequestor) runLowDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining low packets so completion metrics (STATUS,
			// TIME_TAKEN) enqueued just before Disconnect() are not lost.
			r.drainLowChannel()
			return
		case e := <-r.lowCh:
			r.dispatch(e.ctx, e.pkt)
		}
	}
}

func (r *genericRequestor) drainLowChannel() {
	for {
		select {
		case e := <-r.lowCh:
			r.dispatch(e.ctx, e.pkt)
		default:
			return
		}
	}
}

// =============================================================================
// dispatch — routes a single packet to its handler
// =============================================================================

func (r *genericRequestor) dispatch(ctx context.Context, p internal_type.Packet) {
	switch vl := p.(type) {
	// Input
	case internal_type.UserTextPacket:
		r.handleUserText(ctx, vl)
	case internal_type.UserAudioPacket:
		r.handleUserAudio(ctx, vl)
	// Pre-processing
	case internal_type.DenoiseAudioPacket:
		r.handleDenoise(ctx, vl)
	case internal_type.DenoisedAudioPacket:
		r.handleDenoisedAudio(ctx, vl)
	case internal_type.VadAudioPacket:
		r.handleVadAudio(ctx, vl)
	case internal_type.VadSpeechActivityPacket:
		r.callEndOfSpeech(ctx, vl)
	// Recording
	case internal_type.RecordUserAudioPacket:
		r.handleRecordUserAudio(ctx, vl)
	case internal_type.RecordAssistantAudioPacket:
		r.handleRecordAssistantAudio(ctx, vl)
	// Speech detection
	case internal_type.SpeechToTextPacket:
		r.handleSpeechToText(ctx, vl)
	case internal_type.InterimEndOfSpeechPacket:
		r.handleInterimEndOfSpeech(ctx, vl)
	case internal_type.EndOfSpeechPacket:
		r.handleEndOfSpeech(ctx, vl)
	// Interruption
	case internal_type.InterruptionPacket:
		r.handleInterruption(ctx, vl)
	case internal_type.InterruptTTSPacket:
		r.handleInterruptTTS(ctx, vl)
	case internal_type.InterruptLLMPacket:
		r.handleInterruptLLM(ctx, vl)
	// LLM pipeline
	case internal_type.ExecuteLLMPacket:
		r.handleExecuteLLM(ctx, vl)
	case internal_type.LLMResponseDeltaPacket:
		r.handleLLMDelta(ctx, vl)
	case internal_type.LLMResponseDonePacket:
		r.handleLLMDone(ctx, vl)
	case internal_type.LLMErrorPacket:
		r.handleLLMError(ctx, vl)
	// Static / system-injected
	case internal_type.StaticPacket:
		r.handleStaticPacket(ctx, vl)
	// TTS pipeline
	case internal_type.SpeakTextPacket:
		r.handleSpeakText(ctx, vl)
	case internal_type.TextToSpeechAudioPacket:
		r.handleTTSAudio(ctx, vl)
	case internal_type.TextToSpeechEndPacket:
		r.handleTTSEnd(ctx, vl)
	// Persistence
	case internal_type.SaveMessagePacket:
		r.handleSaveMessage(ctx, vl)
	case internal_type.ConversationMetricPacket:
		r.handleConversationMetric(ctx, vl)
	case internal_type.ConversationMetadataPacket:
		r.handleConversationMetadata(ctx, vl)
	case internal_type.MessageMetricPacket:
		r.handleMessageMetric(ctx, vl)
	case internal_type.MessageMetadataPacket:
		r.handleMessageMetadata(ctx, vl)
	// Tools
	case internal_type.LLMToolCallPacket:
		r.handleToolCall(ctx, vl)
	case internal_type.LLMToolResultPacket:
		r.handleToolResult(ctx, vl)
	// Control
	case internal_type.DirectivePacket:
		r.handleDirective(ctx, vl)
	// Observability
	case internal_type.ConversationEventPacket:
		r.handleConversationEvent(ctx, vl)
	default:
		r.logger.Warnf("unknown packet type received in dispatcher %T", vl)
	}
}

// =============================================================================
// Input handlers
// =============================================================================

func (talking *genericRequestor) handleUserText(ctx context.Context, vl internal_type.UserTextPacket) {
	// Only fire a word-interrupt when the assistant is actively generating
	// (LLM streaming, TTS playing, or VAD-interrupted). When the assistant
	// is idle (e.g. greeting already delivered in text mode), skipping the
	// interrupt avoids an unnecessary contextID rotation that would put the
	// greeting and the user's response on different IDs.
	talking.handleInterruption(ctx, internal_type.InterruptionPacket{ContextID: talking.GetID(), Source: internal_type.InterruptionSourceWord})
	vl.ContextID = talking.GetID()
	if err := talking.callEndOfSpeech(ctx, vl); err != nil {
		talking.OnPacket(ctx, internal_type.EndOfSpeechPacket{ContextID: vl.ContextID, Speech: vl.Text})
	}
}

func (talking *genericRequestor) handleUserAudio(ctx context.Context, vl internal_type.UserAudioPacket) {
	if talking.denoiser != nil && !vl.NoiseReduced {
		talking.OnPacket(ctx, internal_type.DenoiseAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio})
		return
	}
	talking.OnPacket(ctx,
		internal_type.RecordUserAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
		internal_type.VadAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
	)
	talking.callSpeechToText(ctx, vl)
	// Route audio to EOS for audio-based turn detectors (e.g. Pipecat Smart Turn).
	// Text-based and silence-based EOS implementations ignore this packet type.
	talking.callEndOfSpeech(ctx, vl)
}

// =============================================================================
// Pre-processing handlers
// =============================================================================

func (talking *genericRequestor) handleDenoise(ctx context.Context, vl internal_type.DenoiseAudioPacket) {
	if talking.denoiser != nil {
		if err := talking.denoiser.Denoise(ctx, vl); err != nil {
			talking.logger.Warnf("denoiser returned unexpected error: %+v", err)
		}
	}
}

func (talking *genericRequestor) handleDenoisedAudio(ctx context.Context, vl internal_type.DenoisedAudioPacket) {
	talking.OnPacket(ctx,
		internal_type.RecordUserAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
		internal_type.VadAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio},
	)
	audioPkt := internal_type.UserAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio, NoiseReduced: vl.NoiseReduced}
	talking.callSpeechToText(ctx, audioPkt)
	// Route denoised audio to EOS for audio-based turn detectors
	talking.callEndOfSpeech(ctx, audioPkt)
}

func (talking *genericRequestor) handleVadAudio(ctx context.Context, vl internal_type.VadAudioPacket) {
	if talking.vad != nil {
		utils.Go(ctx, func() {
			if err := talking.vad.Process(ctx, internal_type.UserAudioPacket{ContextID: vl.ContextID, Audio: vl.Audio}); err != nil {
				talking.logger.Warnf("error while processing with vad %s", err.Error())
			}
		})
	}
}

// =============================================================================
// Recording handlers
// =============================================================================

func (talking *genericRequestor) handleRecordUserAudio(ctx context.Context, vl internal_type.RecordUserAudioPacket) {
	if talking.recorder != nil {
		if err := talking.recorder.Record(ctx, vl); err != nil {
			talking.logger.Errorf("recorder error: %v", err)
		}
	}
}

func (talking *genericRequestor) handleRecordAssistantAudio(ctx context.Context, vl internal_type.RecordAssistantAudioPacket) {
	if talking.recorder != nil {
		if err := talking.recorder.Record(ctx, vl); err != nil {
			talking.logger.Errorf("recorder error: %v", err)
		}
	}
}

// =============================================================================
// Speech detection handlers
// =============================================================================

func (talking *genericRequestor) handleSpeechToText(ctx context.Context, vl internal_type.SpeechToTextPacket) {
	vl.ContextID = talking.GetID()
	if err := talking.callEndOfSpeech(ctx, vl); err != nil {
		if !vl.Interim {
			talking.OnPacket(ctx, internal_type.EndOfSpeechPacket{ContextID: vl.ContextID, Speech: vl.Script, Language: vl.Language})
		}
	}
}

func (talking *genericRequestor) handleInterimEndOfSpeech(ctx context.Context, vl internal_type.InterimEndOfSpeechPacket) {
	talking.Notify(ctx, &protos.ConversationUserMessage{
		Id:        talking.GetID(),
		Message:   &protos.ConversationUserMessage_Text{Text: vl.Speech},
		Completed: false,
		Time:      timestamppb.New(time.Now()),
	})
}

func (talking *genericRequestor) handleEndOfSpeech(ctx context.Context, vl internal_type.EndOfSpeechPacket) {
	talking.stopIdleTimeoutTimer()
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	contextID := talking.GetID()
	if err := talking.Notify(ctx, &protos.ConversationUserMessage{
		Id:        contextID,
		Message:   &protos.ConversationUserMessage_Text{Text: vl.Speech},
		Completed: true,
		Time:      timestamppb.New(time.Now()),
	}); err != nil {
		talking.logger.Tracef(ctx, "might be returning processing the duplicate message so cut it out.")
		return
	}
	talking.OnPacket(ctx,
		internal_type.SaveMessagePacket{ContextID: contextID, MessageRole: "user", Text: vl.Speech, Language: vl.Language},
		internal_type.ExecuteLLMPacket{ContextID: contextID, Input: vl.Speech, Language: vl.Language})
}

// =============================================================================
// Interruption handlers
// =============================================================================

func (talking *genericRequestor) handleInterruption(ctx context.Context, vl internal_type.InterruptionPacket) {
	switch vl.Source {
	case internal_type.InterruptionSourceWord:
		talking.stopIdleTimeoutTimer()
		if err := talking.callEndOfSpeech(ctx, vl); err != nil {
			talking.logger.Errorf("end of speech error: %v", err)
		}

		if err := talking.Transition(Interrupted); err != nil {
			return
		}

		talking.OnPacket(ctx,
			internal_type.RecordAssistantAudioPacket{ContextID: vl.ContextID, Truncate: true},
			internal_type.InterruptTTSPacket{ContextID: vl.ContextID, StartAt: vl.StartAt, EndAt: vl.EndAt},
			internal_type.InterruptLLMPacket{ContextID: vl.ContextID},
		)

		utils.Go(ctx, func() {
			talking.Notify(ctx, &protos.ConversationInterruption{
				Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
				Time: timestamppb.Now(),
			})
		})

	default:
		if vl.StartAt < 5 {
			return
		}
		if err := talking.callEndOfSpeech(ctx, vl); err != nil {
			talking.logger.Errorf("end of speech error: %v", err)
		}

		if err := talking.Transition(Interrupt); err != nil {
			return
		}

		utils.Go(ctx, func() {
			talking.Notify(ctx, &protos.ConversationInterruption{
				Type: protos.ConversationInterruption_INTERRUPTION_TYPE_VAD,
				Time: timestamppb.Now(),
			})
		})
	}
}

func (talking *genericRequestor) handleInterruptTTS(ctx context.Context, vl internal_type.InterruptTTSPacket) {
	if talking.textToSpeechTransformer != nil {
		utils.Go(ctx, func() {
			if err := talking.textToSpeechTransformer.Transform(ctx, internal_type.InterruptionPacket{
				ContextID: vl.ContextID,
				StartAt:   vl.StartAt,
				EndAt:     vl.EndAt,
			}); err != nil {
				talking.logger.Errorf("speak: failed to send interruption to TTS: %v", err)
			}
		})
	}
}

func (talking *genericRequestor) handleInterruptLLM(ctx context.Context, vl internal_type.InterruptLLMPacket) {
	if talking.assistantExecutor != nil {
		utils.Go(ctx, func() {
			talking.assistantExecutor.Execute(ctx, talking, internal_type.InterruptionPacket{ContextID: vl.ContextID})
		})
	}
}

// =============================================================================
// LLM pipeline handlers
// =============================================================================

// handleExecuteLLM runs the LLM executor in a goroutine so the dispatcher
// is not blocked for the duration of the LLM response (which can be seconds).
func (talking *genericRequestor) handleExecuteLLM(ctx context.Context, vl internal_type.ExecuteLLMPacket) {
	if vl.Language != "" {
		talking.args["language"] = vl.Language
	}
	utils.Go(ctx, func() {
		if err := talking.assistantExecutor.Execute(ctx, talking, internal_type.UserTextPacket{ContextID: vl.ContextID, Text: vl.Input}); err != nil {
			talking.logger.Errorf("assistant executor error: %v", err)
			talking.OnPacket(ctx, internal_type.LLMErrorPacket{ContextID: vl.ContextID, Error: err})
		}
	})
}

func (talking *genericRequestor) handleLLMDelta(ctx context.Context, vl internal_type.LLMResponseDeltaPacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "llm",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "current_context": talking.GetID(), "text": vl.Text},
			Time:      time.Now(),
		})
		return
	}
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	if err := talking.callTextAggregator(ctx, vl); err != nil {
		talking.OnPacket(ctx, internal_type.SpeakTextPacket{ContextID: vl.ContextID, Text: vl.Text, IsFinal: false})
	}
}

func (talking *genericRequestor) handleLLMDone(ctx context.Context, vl internal_type.LLMResponseDonePacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "llm",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "done", "current_context": talking.GetID(), "text": vl.Text},
			Time:      time.Now(),
		})
		return
	}
	talking.startIdleTimeoutTimer(ctx)
	if err := talking.Transition(LLMGenerated); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	talking.OnPacket(ctx, internal_type.SaveMessagePacket{ContextID: vl.ContextID, MessageRole: "assistant", Text: vl.Text})
	if err := talking.callTextAggregator(ctx, vl); err != nil {
		talking.OnPacket(ctx, internal_type.SpeakTextPacket{ContextID: vl.ContextID, Text: vl.Text, IsFinal: true})
	}
}

func (talking *genericRequestor) handleLLMError(ctx context.Context, vl internal_type.LLMErrorPacket) {
	talking.logger.Errorf("LLM error for context %s: %v", vl.ContextID, vl.Error)
	_ = talking.Notify(ctx, &protos.ConversationError{
		AssistantConversationId: talking.Conversation().Id,
		Message:                 fmt.Sprintf("llm: %v", vl.Error),
	})
	talking.Transition(LLMGenerated)
}

// =============================================================================
// Static / system-injected handler
// =============================================================================

// handleStaticPacket speaks a pre-written message (greeting, error, idle timeout).
//
// Both modes follow the same state lifecycle: LLMGenerating → deliver →
// LLMGenerated → start idle timer. This ensures the state machine is in
// LLMGenerated after delivery so subsequent Transition(Interrupted) calls
// (from onIdleTimeout or user interrupt) always succeed and rotate the
// context ID.
//
// Text mode delivers synchronously; audio mode flows through the text
// aggregator and TTS pipeline in a goroutine.
func (talking *genericRequestor) handleStaticPacket(ctx context.Context, vl internal_type.StaticPacket) {
	talking.OnPacket(ctx, internal_type.SaveMessagePacket{ContextID: vl.ContextId(), MessageRole: vl.Role(), Text: vl.Content()})

	// Both text and audio modes follow the same state lifecycle:
	// LLMGenerating → deliver → LLMGenerated → start idle timer.
	// This ensures the state machine is in LLMGenerated after delivery,
	// so the next Transition(Interrupted) from onIdleTimeout or user
	// interrupt always succeeds and rotates the context ID.
	if err := talking.Transition(LLMGenerating); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}

	if err := talking.assistantExecutor.Execute(ctx, talking, vl); err != nil {
		talking.logger.Errorf("assistant executor error: %v", err)
	}

	if err := talking.callTextAggregator(ctx, internal_type.LLMResponseDeltaPacket{ContextID: vl.ContextId(), Text: vl.Text}); err != nil {
		talking.OnPacket(ctx, internal_type.SpeakTextPacket{ContextID: vl.ContextId(), Text: vl.Text, IsFinal: false})
	}

	if err := talking.callTextAggregator(ctx, internal_type.LLMResponseDonePacket{ContextID: vl.ContextId(), Text: vl.Text}); err != nil {
		talking.OnPacket(ctx, internal_type.SpeakTextPacket{ContextID: vl.ContextId(), Text: vl.Text, IsFinal: true})
	}

	if err := talking.Transition(LLMGenerated); err != nil {
		talking.logger.Errorf("messaging transition error: %v", err)
	}
	talking.startIdleTimeoutTimer(ctx)
}

// =============================================================================
// TTS pipeline handlers
// =============================================================================

func (talking *genericRequestor) handleSpeakText(ctx context.Context, vl internal_type.SpeakTextPacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "tts",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "speak_text", "current_context": talking.GetID()},
			Time:      time.Now(),
		})
		return
	}

	if talking.textToSpeechTransformer != nil && talking.GetMode().Audio() {
		if vl.IsFinal {
			if err := talking.textToSpeechTransformer.Transform(ctx, internal_type.LLMResponseDonePacket{ContextID: vl.ContextID, Text: vl.Text}); err != nil {
				talking.logger.Errorf("speak: failed to send to TTS: %v", err)
			}
		} else {
			if err := talking.textToSpeechTransformer.Transform(ctx, internal_type.LLMResponseDeltaPacket{ContextID: vl.ContextID, Text: vl.Text}); err != nil {
				talking.logger.Errorf("speak: failed to send to TTS: %v", err)
			}
		}

	}
	if err := talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        vl.ContextID,
		Completed: vl.IsFinal,
		Message:   &protos.ConversationAssistantMessage_Text{Text: vl.Text},
	}); err != nil {
		talking.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}
}

func (talking *genericRequestor) handleTTSAudio(ctx context.Context, vl internal_type.TextToSpeechAudioPacket) {
	if talking.GetMode().Audio() {
		audioInfo := internal_audio.GetAudioInfo(vl.AudioChunk, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG)
		talking.extendIdleTimeoutTimer(time.Duration(audioInfo.DurationMs) * time.Millisecond)
	}
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "tts",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "tts_audio", "current_context": talking.GetID()},
			Time:      time.Now(),
		})
		return
	}
	if err := talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        vl.ContextID,
		Message:   &protos.ConversationAssistantMessage_Audio{Audio: vl.AudioChunk},
		Completed: false,
	}); err != nil {
		talking.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}
	talking.OnPacket(ctx, internal_type.RecordAssistantAudioPacket{ContextID: vl.ContextID, Audio: vl.AudioChunk})
}

func (talking *genericRequestor) handleTTSEnd(ctx context.Context, vl internal_type.TextToSpeechEndPacket) {
	if vl.ContextID != talking.GetID() {
		talking.OnPacket(ctx, internal_type.ConversationEventPacket{
			ContextID: vl.ContextID,
			Name:      "tts",
			Data:      map[string]string{"type": "discarded", "reason": "stale_context", "packet": "tts_end", "current_context": talking.GetID()},
			Time:      time.Now(),
		})
		return
	}
	if err := talking.Notify(ctx, &protos.ConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        vl.ContextID,
		Completed: true,
	}); err != nil {
		talking.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}
}

// =============================================================================
// Persistence handlers
// =============================================================================

func (talking *genericRequestor) handleSaveMessage(ctx context.Context, vl internal_type.SaveMessagePacket) {
	if err := talking.onCreateMessage(ctx, vl); err != nil {
		talking.logger.Errorf("Error in onCreateMessage: %v", err)
	}
	if vl.Language != "" {
		talking.OnPacket(ctx, internal_type.MessageMetadataPacket{
			ContextID: vl.ContextID,
			Metadata:  []*protos.Metadata{{Key: "language", Value: vl.Language}},
		})
	}
}

func (talking *genericRequestor) handleConversationMetric(ctx context.Context, vl internal_type.ConversationMetricPacket) {
	if len(vl.Metrics) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetric{
			AssistantConversationId: vl.ContextID,
			Metrics:                 vl.Metrics,
		})
		if err := talking.onAddMetrics(ctx, vl.Metrics...); err != nil {
			talking.logger.Errorf("Error in onAddMetrics: %v", err)
		}
		talking.metrics.Collect(ctx, observe.ConversationMetricRecord{
			ConversationID: vl.ContextId(),
			Metrics:        vl.Metrics,
			Time:           time.Now(),
		})
	}
}

func (talking *genericRequestor) handleConversationMetadata(ctx context.Context, vl internal_type.ConversationMetadataPacket) {
	if len(vl.Metadata) > 0 {
		if err := talking.onAddMetadata(ctx, vl.Metadata...); err != nil {
			talking.logger.Errorf("Error in onAddMetadata: %v", err)
		}
	}
}

func (talking *genericRequestor) handleMessageMetric(ctx context.Context, vl internal_type.MessageMetricPacket) {
	if vl.ContextID == "" {
		vl.ContextID = talking.GetID()
	}
	if len(vl.Metrics) > 0 {
		_ = talking.Notify(ctx, &protos.ConversationMetric{
			AssistantConversationId: talking.Conversation().Id,
			Metrics:                 vl.Metrics,
		})
		if err := talking.onMessageMetric(ctx, vl.ContextID, vl.Metrics); err != nil {
			talking.logger.Errorf("Error in onMessageMetric: %v", err)
		}
		talking.metrics.Collect(ctx, observe.MessageMetricRecord{
			MessageID:      vl.ContextID,
			ConversationID: fmt.Sprintf("%d", talking.Conversation().Id),
			Metrics:        vl.Metrics,
			Time:           time.Now(),
		})
	}
}

func (talking *genericRequestor) handleMessageMetadata(ctx context.Context, vl internal_type.MessageMetadataPacket) {
	if len(vl.Metadata) == 0 {
		return
	}
	talking.logger.Debugf("message metadata received for context %s", vl.ContextID)
	metadata := make(map[string]interface{}, len(vl.Metadata))
	for _, m := range vl.Metadata {
		metadata[m.Key] = m.Value
	}
	dbCtx, cancel := context.WithTimeout(context.Background(), dbWriteTimeout)
	defer cancel()
	if _, err := talking.conversationService.ApplyMessageMetadata(
		dbCtx, talking.Auth(),
		talking.Conversation().Id,
		vl.ContextID,
		metadata,
	); err != nil {
		talking.logger.Errorf("Error in ApplyMessageMetadata: %v", err)
	}
}

// =============================================================================
// Tool handlers
// =============================================================================

func (talking *genericRequestor) handleToolCall(ctx context.Context, vl internal_type.LLMToolCallPacket) {
	req, _ := json.Marshal(map[string]interface{}{
		"id":        vl.ToolID,
		"name":      vl.Name,
		"arguments": vl.Arguments,
	})
	if err := talking.CreateToolLog(ctx, vl.ContextID, vl.ToolID, vl.Name, type_enums.RECORD_IN_PROGRESS, req); err != nil {
		talking.logger.Errorf("error logging tool call start: %v", err)
	}
}

func (talking *genericRequestor) handleToolResult(ctx context.Context, vl internal_type.LLMToolResultPacket) {
	res, _ := json.Marshal(vl.Result)
	if err := talking.UpdateToolLog(ctx, vl.ToolID, vl.TimeTaken, type_enums.RECORD_COMPLETE, res); err != nil {
		talking.logger.Errorf("error logging tool call result: %v", err)
	}
}

// =============================================================================
// Control handler
// =============================================================================

func (talking *genericRequestor) handleDirective(ctx context.Context, vl internal_type.DirectivePacket) {
	anyArgs, _ := utils.InterfaceMapToAnyMap(vl.Arguments)
	switch vl.Directive {
	case protos.ConversationDirective_END_CONVERSATION:
		if err := talking.Notify(ctx, &protos.ConversationDirective{
			Id:   vl.ContextID,
			Type: vl.Directive,
			Args: anyArgs,
			Time: timestamppb.Now(),
		}); err != nil {
			talking.logger.Errorf("error notifying end conversation action: %v", err)
		}
	default:
	}
}

// handleStageTransition handles the stage transition by updating the conversation's current_stage_id
func (talking *genericRequestor) handleStageTransition(ctx context.Context, contextID, stageName string) {
	if talking.assistantService == nil {
		talking.logger.Warn("assistant service not available for stage transition")
		return
	}

	conversation := talking.Conversation()
	if conversation == nil {
		talking.logger.Warn("no conversation available for stage transition")
		return
	}

	assistant := talking.assistant
	if assistant == nil || assistant.AssistantProviderModel == nil {
		talking.logger.Warn("no assistant available for stage transition")
		return
	}

	// Get all stages and find matching one
	stages, err := talking.assistantService.GetPromptStages(ctx, talking.auth, assistant.AssistantProviderModel.Id)
	if err != nil {
		talking.logger.Errorf("failed to get prompt stages: %v", err)
		return
	}

	var stageId uint64
	for _, stage := range stages {
		if stage.Name == stageName {
			stageId = stage.Id
			break
		}
	}

	if stageId == 0 {
		talking.logger.Warnf("stage not found: %s", stageName)
		return
	}

	// Update conversation stage
	err = talking.assistantService.UpdateConversationStage(ctx, talking.auth, conversation.Id, stageId)
	if err != nil {
		talking.logger.Errorf("failed to update conversation stage: %v", err)
		return
	}

	// Update in-memory conversation state so next LLM turn uses the new prompt
	conversation.CurrentStageId = stageId

	talking.logger.Infof("conversation %d transitioned to stage %s (id: %d)", conversation.Id, stageName, stageId)
}

// =============================================================================
// Observability handler
// =============================================================================

// handleConversationEvent forwards a ConversationEventPacket to the debugger
// via the gRPC stream. If the packet's ContextID is empty (e.g. emitted by an
// STT callback that doesn't hold the session context) the current interaction
// context ID from r.GetID() is used.
func (talking *genericRequestor) handleConversationEvent(ctx context.Context, vl internal_type.ConversationEventPacket) {
	contextID := vl.ContextID
	if contextID == "" {
		contextID = talking.GetID()
	}

	// Handle stage transition events
	if vl.Name == "stage_transition" {
		if stageName, ok := vl.Data["stage_name"]; ok {
			talking.handleStageTransition(ctx, contextID, stageName)
		}
	}

	if vl.Time.IsZero() {
		vl.Time = time.Now()
	}
	_ = talking.Notify(ctx, &protos.ConversationEvent{
		Id:   contextID,
		Name: vl.Name,
		Data: vl.Data,
		Time: timestamppb.New(vl.Time),
	})
	talking.events.Collect(ctx, observe.EventRecord{
		MessageID: contextID,
		Name:      vl.Name,
		Data:      vl.Data,
		Time:      vl.Time,
	})
}
