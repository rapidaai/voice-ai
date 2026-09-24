package lifecycle

import (
	"errors"
	"strings"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

type generationPhase uint8
type playbackPhase uint8

func (l *messageLifecycle) OnMessageInjected(packet internal_type.InjectMessagePacket) (internal_type.InjectMessagePacket, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validateContextLocked(packet.ContextID); err != nil {
		return packet, err
	}
	if l.output.generation == generationCompleted || l.output.playback == playbackFailed || l.output.playback == playbackCompleted {
		return packet, ErrInvalidTransition
	}
	packet.Interim = l.output.generation != generationIdle
	if l.output.generation == generationIdle {
		l.output.generation = generationStarted
		l.state = MessageStateAssistantGenerating
	}
	l.output.hasText = l.output.hasText || strings.TrimSpace(packet.Text) != ""
	return packet, nil
}

// SendAssistantMessage serializes output with controls and records only successful delivery.
func (l *messageLifecycle) SendAssistantMessage(message *protos.ConversationAssistantMessage) error {
	if l.sendOutput == nil {
		return ErrSenderNotConfigured
	}
	l.playbackControlMu.Lock()
	defer l.playbackControlMu.Unlock()
	l.mu.Lock()
	if err := l.validateContextLocked(message.GetId()); err != nil {
		l.mu.Unlock()
		return err
	}
	if l.output.playback == playbackFailed || l.output.playback == playbackCompleted {
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	switch content := message.Message.(type) {
	case *protos.ConversationAssistantMessage_Audio:
		if !l.mode.Audio() || l.output.playback != playbackOpen || (message.Completed && l.output.generation != generationCompleted) {
			l.mu.Unlock()
			return ErrInvalidTransition
		}
		if message.Completed && !l.output.hasAudio && len(content.Audio) == 0 && l.output.hasText {
			l.output.playback = playbackFailed
			l.mu.Unlock()
			return errors.New("synthesis completed without audio for nonempty text")
		}
		if message.Completed {
			l.output.playback = playbackClosing
		}
	case *protos.ConversationAssistantMessage_Text:
		l.output.hasText = l.output.hasText || strings.TrimSpace(content.Text) != ""
	default:
		l.mu.Unlock()
		return ErrInvalidTransition
	}
	l.mu.Unlock()
	err := l.sendOutput(message)
	l.mu.Lock()
	if message.Id != l.contextID {
		l.mu.Unlock()
		return err
	}
	if err != nil {
		l.output.playback = playbackFailed
		l.output.receiptReceived = false
		if l.output.receiptTimer != nil {
			l.output.receiptTimer.Stop()
		}
		l.mu.Unlock()
		return err
	}
	switch content := message.Message.(type) {
	case *protos.ConversationAssistantMessage_Audio:
		l.output.hasAudio = l.output.hasAudio || len(content.Audio) > 0
		l.output.audioDuration += time.Duration(internal_audio.GetAudioInfo(content.Audio, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG).DurationMs) * time.Millisecond
		if message.Completed {
			if l.output.playback == playbackClosing {
				l.output.playback = playbackAwaitingReceipt
			}
			l.output.receiptRemaining = l.output.audioDuration + 5*time.Second
		}
	case *protos.ConversationAssistantMessage_Text:
		l.output.textDelivered = l.output.textDelivered || message.Completed
	}
	l.mu.Unlock()
	l.awaitPlayback(message.Id)
	_ = l.completeAssistantMessage(message.Id)
	return nil
}

// awaitPlayback bounds missing receipts in active playback time, excluding pauses.
func (l *messageLifecycle) awaitPlayback(contextID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if contextID != l.contextID || l.output.playback != playbackAwaitingReceipt || l.output.receiptReceived || l.output.paused || l.output.receiptTimer != nil {
		return
	}
	l.output.receiptDeadline = time.Now().Add(l.output.receiptRemaining)
	deadline := l.output.receiptDeadline
	l.output.receiptTimer = time.AfterFunc(l.output.receiptRemaining, func() {
		l.mu.Lock()
		if contextID != l.contextID || l.output.playback != playbackAwaitingReceipt || l.output.receiptReceived || l.output.paused || !l.output.receiptDeadline.Equal(deadline) {
			l.mu.Unlock()
			return
		}
		l.output.playback = playbackFailed
		l.output.receiptTimer = nil
		onPacket := l.onPacket
		l.mu.Unlock()
		if onPacket != nil {
			_ = onPacket(internal_type.TextToSpeechErrorPacket{ContextID: contextID, Error: errors.New("playback completion receipt timed out"), Type: internal_type.TTSPlaybackTimeout})
		}
	})
}

func (l *messageLifecycle) OnMessageFailed(contextID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if contextID != l.contextID || l.output.playback == playbackCompleted {
		return
	}
	l.output.playback = playbackFailed
	l.output.receiptReceived = false
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
		l.output.receiptTimer = nil
	}
}
