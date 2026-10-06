package internal_twilio

import (
	"errors"
	"testing"
	"time"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
)

func newTestAudioProcessor(resamplerOutput []byte, resamplerErr error) *AudioProcessor {
	resampler := &twilioFakeResampler{out: resamplerOutput, err: resamplerErr}
	processor := &AudioProcessor{
		resampler:          resampler,
		twilioConfig:       &protos.AudioConfig{},
		downstreamConfig:   &protos.AudioConfig{},
		inputBuffer:        newInputBufferForTest(),
		outputBuffer:       newOutputBufferForTest(OutputChunkSize * 8),
		bridgeOutputBuffer: newOutputBufferForTest(BridgeOutputFrameSize * 8),
	}
	processor.silenceFrame = processor.createSilenceFrame()
	return processor
}

func newInputBufferForTest() *inputBufferForTest {
	return &inputBufferForTest{data: make([]byte, 0, InputBufferThreshold*2)}
}

func newOutputBufferForTest(capacity int) *outputBufferForTest {
	return &outputBufferForTest{data: make([]byte, 0, capacity)}
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
	processor := newTestAudioProcessor(convertedAudio, nil)
	receivedAt := time.Now()

	firstFrame, err := processor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
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

	secondFrame, err := processor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{2}})
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
	processor := newTestAudioProcessor(nil, errors.New("resample failed"))

	_, err := processor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: []byte{1}})
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !errors.Is(err, ErrProviderAudioConversionFailed) {
		t.Fatalf("expected provider audio conversion error, got %v", err)
	}
}

func TestAudioProcessor_ProcessAssistantAudio_ProducesProviderAndBridgeOutputFrames(t *testing.T) {
	providerAudio := make([]byte, OutputChunkSize)
	providerAudio[0] = 9
	assistantAudio := make([]byte, BridgeOutputFrameSize)
	assistantAudio[0] = 4
	processor := newTestAudioProcessor(providerAudio, nil)

	if err := processor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := processor.NextOutputFrame()
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
	processor := newTestAudioProcessor(nil, errors.New("resample failed"))

	err := processor.ProcessAssistantAudio([]byte{1}, false)
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !errors.Is(err, ErrAssistantAudioConversionFailed) {
		t.Fatalf("expected assistant audio conversion error, got %v", err)
	}
}

func TestAudioProcessor_ClearOutputBuffer_ClearsProviderAndBridgeBuffers(t *testing.T) {
	providerAudio := make([]byte, OutputChunkSize)
	assistantAudio := make([]byte, BridgeOutputFrameSize)
	processor := newTestAudioProcessor(providerAudio, nil)

	if err := processor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	processor.ClearOutputBuffer()

	if _, ok := processor.NextOutputFrame(); ok {
		t.Fatal("expected no output frame after clear")
	}
}

func TestAudioProcessor_IdleOutputFrame_UsesProviderSilence(t *testing.T) {
	processor := newTestAudioProcessor(nil, nil)

	outputFrame, ok := processor.IdleOutputFrame()
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
