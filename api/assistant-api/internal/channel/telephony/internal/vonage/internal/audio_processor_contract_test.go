package internal_vonage

import (
	"testing"
	"time"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/protos"
)

func newTestAudioProcessor() *AudioProcessor {
	audioProcessor := &AudioProcessor{
		audioConfig:        &protos.AudioConfig{},
		inputBuffer:        newInputBufferForTest(),
		outputBuffer:       newOutputBufferForTest(OutputChunkSize * 8),
		bridgeOutputBuffer: newOutputBufferForTest(OutputChunkSize * 8),
	}
	audioProcessor.silenceFrame = audioProcessor.createSilenceFrame()
	return audioProcessor
}

func TestOutputDrainedRequiresAllProviderBytes(t *testing.T) {
	processor := newTestAudioProcessor()
	if err := processor.ProcessAssistantAudio([]byte{1, 2}, false); err != nil {
		t.Fatal(err)
	}
	if _, available := processor.NextOutputFrame(); available {
		t.Fatal("partial frame is available")
	}
	if processor.OutputDrained() {
		t.Fatal("partial frame was treated as drained")
	}
	if err := processor.ProcessAssistantAudio(nil, true); err != nil {
		t.Fatal(err)
	}
	if _, available := processor.NextOutputFrame(); !available {
		t.Fatal("terminal frame is unavailable")
	}
	if !processor.OutputDrained() {
		t.Fatal("sent terminal frame remains queued")
	}
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
	providerAudio := make([]byte, OutputChunkSize)
	providerAudio[0] = 7
	audioProcessor := newTestAudioProcessor()
	receivedAt := time.Now()

	firstFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      providerAudio,
		ReceivedAt: receivedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(firstFrame.BridgeAudio) != OutputChunkSize {
		t.Fatalf("bridgeAudio length=%d want=%d", len(firstFrame.BridgeAudio), OutputChunkSize)
	}
	if len(firstFrame.PipelineAudio) != 0 {
		t.Fatalf("pipelineAudio length=%d want=0", len(firstFrame.PipelineAudio))
	}

	secondFrame, err := audioProcessor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: providerAudio})
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

func TestAudioProcessor_ProcessAssistantAudio_ProducesProviderAndBridgeOutputFrames(t *testing.T) {
	assistantAudio := make([]byte, OutputChunkSize)
	assistantAudio[0] = 4
	audioProcessor := newTestAudioProcessor()

	if err := audioProcessor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFrame, ok := audioProcessor.NextOutputFrame()
	if !ok {
		t.Fatal("expected output frame")
	}
	if len(outputFrame.ProviderAudio) != OutputChunkSize || outputFrame.ProviderAudio[0] != 4 {
		t.Fatalf("unexpected provider audio length=%d", len(outputFrame.ProviderAudio))
	}
	if len(outputFrame.BridgeAudio) != OutputChunkSize || outputFrame.BridgeAudio[0] != 4 {
		t.Fatalf("unexpected bridge audio length=%d", len(outputFrame.BridgeAudio))
	}
}

func TestAudioProcessor_ClearOutputBuffer_ClearsProviderAndBridgeBuffers(t *testing.T) {
	assistantAudio := make([]byte, OutputChunkSize)
	audioProcessor := newTestAudioProcessor()

	if err := audioProcessor.ProcessAssistantAudio(assistantAudio, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	audioProcessor.ClearOutputBuffer()

	if _, ok := audioProcessor.NextOutputFrame(); ok {
		t.Fatal("expected no output frame after clear")
	}
}

func TestAudioProcessor_IdleOutputFrame_UsesProviderSilence(t *testing.T) {
	audioProcessor := newTestAudioProcessor()

	outputFrame, ok := audioProcessor.IdleOutputFrame()
	if !ok {
		t.Fatal("expected idle output frame")
	}
	if len(outputFrame.ProviderAudio) != OutputChunkSize {
		t.Fatalf("providerAudio length=%d want=%d", len(outputFrame.ProviderAudio), OutputChunkSize)
	}
	if outputFrame.ProviderAudio[0] != 0 {
		t.Fatalf("providerAudio[0]=%x want=0", outputFrame.ProviderAudio[0])
	}
	if len(outputFrame.BridgeAudio) != 0 {
		t.Fatalf("bridgeAudio length=%d want=0", len(outputFrame.BridgeAudio))
	}
}
