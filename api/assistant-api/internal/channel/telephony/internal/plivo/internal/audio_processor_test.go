// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo

import (
	"errors"
	"testing"
	"time"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
)

type plivoFakeResampler struct {
	out []byte
	err error
}

func (resampler *plivoFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if resampler.err != nil {
		return nil, resampler.err
	}
	return append([]byte(nil), resampler.out...), nil
}

func newTestAudioProcessor(resamplerOutput []byte, resamplerErr error) *AudioProcessor {
	resampler := &plivoFakeResampler{out: resamplerOutput, err: resamplerErr}
	audioProcessor := &AudioProcessor{
		resampler:          resampler,
		plivoConfig:        &protos.AudioConfig{},
		downstreamConfig:   &protos.AudioConfig{},
		inputBuffer:        &inputBufferForTest{data: make([]byte, 0, InputBufferThreshold*2)},
		outputBuffer:       &outputBufferForTest{data: make([]byte, 0, OutputChunkSize*8)},
		bridgeOutputBuffer: &outputBufferForTest{data: make([]byte, 0, BridgeOutputFrameSize*8)},
		outputHealth:       nil,
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame()
	return audioProcessor
}

type inputBufferForTest struct {
	data []byte
}

func (buffer *inputBufferForTest) Write(data []byte) {
	buffer.data = append(buffer.data, data...)
}

func (buffer *inputBufferForTest) DrainIfReady(threshold int) ([]byte, bool) {
	if len(buffer.data) < threshold {
		return nil, false
	}
	out := append([]byte(nil), buffer.data...)
	buffer.data = buffer.data[:0]
	return out, true
}

func (buffer *inputBufferForTest) Clear() {
	buffer.data = buffer.data[:0]
}

func (buffer *inputBufferForTest) Len() int {
	return len(buffer.data)
}

type outputBufferForTest struct {
	data []byte
}

func (buffer *outputBufferForTest) Write(data []byte) {
	buffer.data = append(buffer.data, data...)
}

func (buffer *outputBufferForTest) Next(frameSize int) ([]byte, bool) {
	if len(buffer.data) < frameSize {
		return nil, false
	}
	frame := append([]byte(nil), buffer.data[:frameSize]...)
	buffer.data = buffer.data[frameSize:]
	return frame, true
}

func (buffer *outputBufferForTest) Complete(frameSize int, padByte byte) {
	remainder := len(buffer.data) % frameSize
	if remainder == 0 {
		return
	}
	padding := make([]byte, frameSize-remainder)
	for i := range padding {
		padding[i] = padByte
	}
	buffer.data = append(buffer.data, padding...)
}

func (buffer *outputBufferForTest) Clear() {
	buffer.data = buffer.data[:0]
}

func (buffer *outputBufferForTest) Len() int {
	return len(buffer.data)
}

func TestAudioProcessor_ProcessProviderAudioFrame_EmitsBridgeAndThresholdedPipelineAudio(t *testing.T) {
	convertedAudio := make([]byte, BridgeOutputFrameSize)
	convertedAudio[0] = 7
	audioProcessor := newTestAudioProcessor(convertedAudio, nil)
	receivedAt := time.Now()

	firstFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      []byte{1},
		ReceivedAt: receivedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(firstFrame.BridgeAudio) != BridgeOutputFrameSize {
		t.Fatalf("bridgeAudio length=%d want=%d", len(firstFrame.BridgeAudio), BridgeOutputFrameSize)
	}
	if len(firstFrame.PipelineAudio) != 0 {
		t.Fatalf("pipelineAudio length=%d want=0", len(firstFrame.PipelineAudio))
	}

	secondFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secondFrame.PipelineAudio) != InputBufferThreshold {
		t.Fatalf("pipelineAudio length=%d want=%d", len(secondFrame.PipelineAudio), InputBufferThreshold)
	}
	if !firstFrame.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("receivedAt=%s want=%s", firstFrame.ReceivedAt, receivedAt)
	}
}

func TestAudioProcessor_ProcessProviderAudioFrame_PropagatesConversionError(t *testing.T) {
	audioProcessor := newTestAudioProcessor(nil, errors.New("resample failed"))

	_, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{1}})
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !errors.Is(err, ErrProviderAudioConversionFailed) {
		t.Fatalf("expected ErrProviderAudioConversionFailed, got %v", err)
	}
}

func TestAudioProcessor_ProcessAssistantAudio_ProducesProviderAndBridgeOutputFrames(t *testing.T) {
	providerAudio := make([]byte, OutputChunkSize)
	providerAudio[0] = 9
	assistantAudio := make([]byte, BridgeOutputFrameSize)
	assistantAudio[0] = 4
	audioProcessor := newTestAudioProcessor(providerAudio, nil)

	if err := audioProcessor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := audioProcessor.NextOutputFrame()
	if !ok {
		t.Fatal("expected output frame")
	}
	if len(outputFrame.ProviderAudio) != OutputChunkSize || outputFrame.ProviderAudio[0] != 9 {
		t.Fatalf("unexpected provider audio: %v", outputFrame.ProviderAudio[:1])
	}
	if len(outputFrame.BridgeAudio) != BridgeOutputFrameSize || outputFrame.BridgeAudio[0] != 4 {
		t.Fatalf("unexpected bridge audio length=%d", len(outputFrame.BridgeAudio))
	}
}

func TestAudioProcessor_ProcessAssistantAudio_PropagatesConversionError(t *testing.T) {
	audioProcessor := newTestAudioProcessor(nil, errors.New("resample failed"))

	err := audioProcessor.ProcessAssistantAudio([]byte{1}, false)
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !errors.Is(err, ErrAssistantAudioConversionFailed) {
		t.Fatalf("expected ErrAssistantAudioConversionFailed, got %v", err)
	}
}

func TestAudioProcessor_ClearOutputBuffer_ClearsProviderAndBridgeBuffers(t *testing.T) {
	providerAudio := make([]byte, OutputChunkSize)
	assistantAudio := make([]byte, BridgeOutputFrameSize)
	audioProcessor := newTestAudioProcessor(providerAudio, nil)

	if err := audioProcessor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	audioProcessor.ClearOutputBuffer()

	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no output frame after clear")
	}
}

func TestAudioProcessor_IdleOutputFrame_UsesProviderSilence(t *testing.T) {
	audioProcessor := newTestAudioProcessor(nil, nil)

	outputFrame, ok := audioProcessor.IdleOutputFrame()
	if !ok {
		t.Fatal("expected idle output frame")
	}
	if len(outputFrame.ProviderAudio) != OutputChunkSize {
		t.Fatalf("providerAudio length=%d want=%d", len(outputFrame.ProviderAudio), OutputChunkSize)
	}
	if outputFrame.ProviderAudio[0] != MulawSilence {
		t.Fatalf("providerAudio[0]=%x want=%x", outputFrame.ProviderAudio[0], MulawSilence)
	}
	if len(outputFrame.BridgeAudio) != 0 {
		t.Fatalf("bridgeAudio length=%d want=0", len(outputFrame.BridgeAudio))
	}
}
