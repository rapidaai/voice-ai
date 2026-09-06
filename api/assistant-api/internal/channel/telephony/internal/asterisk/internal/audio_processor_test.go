// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"errors"
	"sync"
	"testing"
	"time"

	internal_channel_input "github.com/rapidaai/api/assistant-api/internal/channel/input"
	internal_channel_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
)

type mockResampler struct {
	err error
}

func (resampler *mockResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if resampler.err != nil {
		return nil, resampler.err
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func newTestProcessor(t *testing.T, silenceByte byte, frameSize int) *AudioProcessor {
	t.Helper()
	audioProcessor := &AudioProcessor{
		logger:             nil,
		resampler:          &mockResampler{},
		asteriskConfig:     &protos.AudioConfig{},
		downstreamConfig:   &protos.AudioConfig{},
		silenceByte:        silenceByte,
		optimalFrameSize:   frameSize,
		inputBuffer:        internal_channel_input.NewBytesInputBuffer(inputBufferThreshold * 2),
		outputBuffer:       internal_channel_output.NewBytesFrameBuffer(frameSize * 8),
		bridgeOutputBuffer: internal_channel_output.NewBytesFrameBuffer(bridgeOutputFrameSize * 8),
		outputHealth:       internal_channel_output.NewHealthStats(),
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame(frameSize, silenceByte)
	return audioProcessor
}

func TestNewAudioProcessor_HappyPath(t *testing.T) {
	audioProcessor, err := NewAudioProcessor(nil, AudioProcessorConfig{
		AsteriskConfig:   &protos.AudioConfig{},
		DownstreamConfig: &protos.AudioConfig{},
		FrameSize:        320,
	})
	if err != nil {
		t.Fatalf("NewAudioProcessor error: %v", err)
	}
	if audioProcessor.resampler == nil {
		t.Fatal("expected resampler")
	}
	if audioProcessor.GetOptimalFrameSize() != 320 {
		t.Errorf("expected frame size 320, got %d", audioProcessor.GetOptimalFrameSize())
	}
	if audioProcessor.GetDownstreamConfig() == nil {
		t.Fatal("expected non-nil downstream config")
	}
}

func TestSetOptimalFrameSize(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	audioProcessor.SetOptimalFrameSize(256)
	if audioProcessor.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256, got %d", audioProcessor.GetOptimalFrameSize())
	}
	if len(audioProcessor.silenceFrame) != 256 {
		t.Errorf("expected silence frame size 256, got %d", len(audioProcessor.silenceFrame))
	}

	audioProcessor.SetOptimalFrameSize(0)
	if audioProcessor.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256 after zero update, got %d", audioProcessor.GetOptimalFrameSize())
	}

	audioProcessor.SetOptimalFrameSize(-1)
	if audioProcessor.GetOptimalFrameSize() != 256 {
		t.Errorf("expected 256 after negative update, got %d", audioProcessor.GetOptimalFrameSize())
	}
}

func TestSetOptimalFrameSize_IgnoresOversizedValue(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	audioProcessor.SetOptimalFrameSize(maxOptimalFrameSize + 1)

	if audioProcessor.GetOptimalFrameSize() != 160 {
		t.Fatalf("expected oversized frame size to be ignored, got %d", audioProcessor.GetOptimalFrameSize())
	}
	if len(audioProcessor.silenceFrame) != 160 {
		t.Fatalf("silence frame size=%d want=160", len(audioProcessor.silenceFrame))
	}
}

func TestCreateSilenceFrame_BoundsOversizedAllocation(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	frame := audioProcessor.createSilenceFrame(maxOptimalFrameSize+1, 0xFF)

	if len(frame) != defaultFrameSize {
		t.Fatalf("silence frame size=%d want=%d", len(frame), defaultFrameSize)
	}
}

func TestProcessProviderAudioFrame_EmptyInput(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 320)
	receivedAt := time.Now()

	frame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		ReceivedAt: receivedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frame.BridgeAudio) != 0 {
		t.Fatalf("bridgeAudio length=%d want=0", len(frame.BridgeAudio))
	}
	if len(frame.PipelineAudio) != 0 {
		t.Fatalf("pipelineAudio length=%d want=0", len(frame.PipelineAudio))
	}
	if !frame.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("receivedAt=%s want=%s", frame.ReceivedAt, receivedAt)
	}
}

func TestProcessProviderAudioFrame_EmitsBridgeAndThresholdedPipelineAudio(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 320)

	smallChunk := make([]byte, inputBufferThreshold-1)
	for i := range smallChunk {
		smallChunk[i] = 0x42
	}
	firstFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: smallChunk})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(firstFrame.BridgeAudio) != inputBufferThreshold-1 {
		t.Fatalf("bridgeAudio length=%d want=%d", len(firstFrame.BridgeAudio), inputBufferThreshold-1)
	}
	if len(firstFrame.PipelineAudio) != 0 {
		t.Fatalf("pipelineAudio length=%d want=0", len(firstFrame.PipelineAudio))
	}

	secondFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{0x43}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secondFrame.PipelineAudio) != inputBufferThreshold {
		t.Fatalf("pipelineAudio length=%d want=%d", len(secondFrame.PipelineAudio), inputBufferThreshold)
	}
}

func TestProcessProviderAudioFrame_PropagatesConversionError(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 320)
	audioProcessor.resampler = &mockResampler{err: errors.New("resample failed")}

	_, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{1}})
	if err == nil {
		t.Fatal("expected conversion error")
	}
}

func TestProcessAssistantAudio_EmptyInput(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	if err := audioProcessor.ProcessAssistantAudio(nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no output frame for empty input")
	}
}

func TestProcessAssistantAudio_ProducesProviderAndBridgeOutputFrames(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)
	assistantAudio := make([]byte, bridgeOutputFrameSize)
	for i := range assistantAudio {
		assistantAudio[i] = 0xAA
	}

	if err := audioProcessor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := audioProcessor.NextOutputFrame()
	if !ok {
		t.Fatal("expected output frame")
	}
	if len(outputFrame.ProviderAudio) != 160 {
		t.Fatalf("providerAudio length=%d want=160", len(outputFrame.ProviderAudio))
	}
	if outputFrame.ProviderAudio[0] != 0xAA {
		t.Fatalf("providerAudio[0]=%x want=aa", outputFrame.ProviderAudio[0])
	}
	if len(outputFrame.BridgeAudio) != bridgeOutputFrameSize {
		t.Fatalf("bridgeAudio length=%d want=%d", len(outputFrame.BridgeAudio), bridgeOutputFrameSize)
	}
	if outputFrame.BridgeAudio[0] != 0xAA {
		t.Fatalf("bridgeAudio[0]=%x want=aa", outputFrame.BridgeAudio[0])
	}
}

func TestProcessAssistantAudio_PropagatesConversionError(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)
	audioProcessor.resampler = &mockResampler{err: errors.New("resample failed")}

	err := audioProcessor.ProcessAssistantAudio([]byte{1}, false)
	if err == nil {
		t.Fatal("expected conversion error")
	}
}

func TestProcessAssistantAudio_PadsProviderAndBridgeOnCompleted(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	audio := make([]byte, 100)
	for i := range audio {
		audio[i] = 0xBB
	}
	if err := audioProcessor.ProcessAssistantAudio(audio, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no provider frame before completion")
	}

	if err := audioProcessor.ProcessAssistantAudio(nil, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := audioProcessor.NextOutputFrame()
	if !ok {
		t.Fatal("expected padded output frame after completion")
	}
	if len(outputFrame.ProviderAudio) != 160 {
		t.Fatalf("providerAudio length=%d want=160", len(outputFrame.ProviderAudio))
	}
	for i := 100; i < 160; i++ {
		if outputFrame.ProviderAudio[i] != 0xFF {
			t.Fatalf("providerAudio[%d]=%x want=ff", i, outputFrame.ProviderAudio[i])
		}
	}
	if len(outputFrame.BridgeAudio) != bridgeOutputFrameSize {
		t.Fatalf("bridgeAudio length=%d want=%d", len(outputFrame.BridgeAudio), bridgeOutputFrameSize)
	}
}

func TestProcessAssistantAudio_UsesSLINSilencePadding(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 320)

	audio := make([]byte, 100)
	for i := range audio {
		audio[i] = 0xCC
	}
	if err := audioProcessor.ProcessAssistantAudio(audio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := audioProcessor.NextOutputFrame()
	if !ok {
		t.Fatal("expected output frame")
	}
	for i := 100; i < 320; i++ {
		if outputFrame.ProviderAudio[i] != 0x00 {
			t.Fatalf("providerAudio[%d]=%x want=00", i, outputFrame.ProviderAudio[i])
		}
	}
}

func TestClearOutputBuffer(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)
	if err := audioProcessor.ProcessAssistantAudio(make([]byte, 160), true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audioProcessor.ClearOutputBuffer()
	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no output frame after clearing output buffer")
	}
}

func TestIdleOutputFrame_UsesProviderSilence(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	outputFrame, ok := audioProcessor.IdleOutputFrame()
	if !ok {
		t.Fatal("expected idle output frame")
	}
	if len(outputFrame.ProviderAudio) != 160 {
		t.Fatalf("providerAudio length=%d want=160", len(outputFrame.ProviderAudio))
	}
	for i, b := range outputFrame.ProviderAudio {
		if b != 0xFF {
			t.Fatalf("providerAudio[%d]=%x want=ff", i, b)
		}
	}
}

func TestXOFFSuppressesOutput(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)
	if err := audioProcessor.ProcessAssistantAudio(make([]byte, 160), true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audioProcessor.SetXOFF()
	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no active frame while XOFF")
	}
	if _, ok := audioProcessor.IdleOutputFrame(); ok {
		t.Fatal("expected no idle frame while XOFF")
	}

	audioProcessor.SetXON()
	if _, ok := audioProcessor.NextOutputFrame(); !ok {
		t.Fatal("expected active frame after XON")
	}
}

func TestXOFF_XON(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 160)

	if audioProcessor.IsXOFF() {
		t.Error("should start with XOFF=false")
	}

	audioProcessor.SetXOFF()
	if !audioProcessor.IsXOFF() {
		t.Error("expected XOFF=true after SetXOFF")
	}

	audioProcessor.SetXON()
	if audioProcessor.IsXOFF() {
		t.Error("expected XOFF=false after SetXON")
	}
}

func TestXOFF_ConcurrentAccess(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0x00, 160)
	var waitGroup sync.WaitGroup

	for i := 0; i < 100; i++ {
		waitGroup.Add(3)
		go func() {
			defer waitGroup.Done()
			audioProcessor.SetXOFF()
		}()
		go func() {
			defer waitGroup.Done()
			audioProcessor.SetXON()
		}()
		go func() {
			defer waitGroup.Done()
			_ = audioProcessor.IsXOFF()
		}()
	}
	waitGroup.Wait()
}

func TestAudioProcessor_OutputHealthObserverRecordsTicks(t *testing.T) {
	audioProcessor := newTestProcessor(t, 0xFF, 160)

	audioProcessor.OnTickHealth(internal_channel_output.TickHealth{Active: true})
	audioProcessor.OnTickHealth(internal_channel_output.TickHealth{Idle: true, SendError: true})

	stats := audioProcessor.OutputHealthSnapshot()
	if stats.Ticks != 2 {
		t.Fatalf("ticks=%d want=2", stats.Ticks)
	}
	if stats.ActiveTicks != 1 {
		t.Fatalf("activeTicks=%d want=1", stats.ActiveTicks)
	}
	if stats.IdleTicks != 1 {
		t.Fatalf("idleTicks=%d want=1", stats.IdleTicks)
	}
	if stats.SendErrors != 1 {
		t.Fatalf("sendErrors=%d want=1", stats.SendErrors)
	}
}
