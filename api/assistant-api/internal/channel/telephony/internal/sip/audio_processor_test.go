// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	internal_telephony_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaf/g711"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockResampler struct {
	out []byte
	err error
}

func TestNewAudioProcessor_CreatesIndependentStreamResamplers(tester *testing.T) {
	processor := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: testRTPHandler(tester, &sip_runtime.CodecPCMU),
	})

	resamplers := map[internal_type.AudioResampler]struct{}{
		processor.resamplers.provider:       {},
		processor.resamplers.assistant:      {},
		processor.resamplers.bridgeUser:     {},
		processor.resamplers.bridgeOperator: {},
		processor.resamplers.ambient:        {},
	}
	require.Len(tester, resamplers, 5)
}

func (m *mockResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return append([]byte(nil), m.out...), nil
	}
	// Pass through the same data.
	return data, nil
}

type captureResampler struct {
	out   []byte
	err   error
	input []byte
	from  *protos.AudioConfig
	to    *protos.AudioConfig
}

func (r *captureResampler) Resample(data []byte, from, to *protos.AudioConfig) ([]byte, error) {
	r.input = append([]byte(nil), data...)
	r.from = from
	r.to = to
	if r.err != nil {
		return nil, r.err
	}
	if r.out != nil {
		return append([]byte(nil), r.out...), nil
	}
	return append([]byte(nil), data...), nil
}

type mockAmbientMixer struct {
	err error
}

type blockingFrameBuffer struct {
	buffer  internal_telephony_output.FrameBuffer
	entered chan struct{}
	release chan struct{}
}

func (b *blockingFrameBuffer) Write(data []byte) {
	close(b.entered)
	<-b.release
	b.buffer.Write(data)
}

func (b *blockingFrameBuffer) Next(frameSize int) ([]byte, bool) {
	return b.buffer.Next(frameSize)
}

func (b *blockingFrameBuffer) Complete(frameSize int, padByte byte) {
	b.buffer.Complete(frameSize, padByte)
}

func (b *blockingFrameBuffer) Clear() {
	b.buffer.Clear()
}

func (b *blockingFrameBuffer) Len() int {
	return b.buffer.Len()
}

func (m *mockAmbientMixer) Configure(internal_ambient.Config) error { return nil }
func (m *mockAmbientMixer) Mix(primary []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return primary, nil
}
func (m *mockAmbientMixer) Reset() {}
func (m *mockAmbientMixer) CurrentConfig() internal_ambient.Config {
	return internal_ambient.Config{}
}

// pushRecorder captures all streams pushed to the recording sink.
type pushRecorder struct {
	mu      sync.Mutex
	streams []internal_type.Stream
}

func (r *pushRecorder) push(s internal_type.Stream) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams = append(r.streams, s)
}

func (r *pushRecorder) get() []internal_type.Stream {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]internal_type.Stream, len(r.streams))
	copy(cp, r.streams)
	return cp
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type fakeRTPHandler struct {
	codec        *sip_runtime.Codec
	inboundSink  func(sip_runtime.InboundAudioFrame)
	audioOut     chan []byte
	localAddress sip_runtime.RTPAddress
}

func newTestRTPHandler(codec *sip_runtime.Codec) *fakeRTPHandler {
	if codec == nil {
		codec = &sip_runtime.CodecPCMU
	}
	return &fakeRTPHandler{
		codec:    codec,
		audioOut: make(chan []byte, 100),
	}
}

func (h *fakeRTPHandler) SetInboundAudioSink(sink func(sip_runtime.InboundAudioFrame)) {
	h.inboundSink = sink
}

func (h *fakeRTPHandler) emitInboundAudio(frame sip_runtime.InboundAudioFrame) {
	if h.inboundSink != nil {
		h.inboundSink(frame)
	}
}
func (h *fakeRTPHandler) GetCodec() *sip_runtime.Codec { return h.codec }
func (h *fakeRTPHandler) LocalAddress() sip_runtime.RTPAddress {
	return h.localAddress
}
func (h *fakeRTPHandler) WriteAudio(audio []byte) error {
	select {
	case h.audioOut <- audio:
		return nil
	default:
		return errors.New("test RTP audio sink full")
	}
}

func testRTPHandler(t *testing.T, codec *sip_runtime.Codec) *fakeRTPHandler {
	t.Helper()
	return newTestRTPHandler(codec)
}

func rtpAudioOutLen(t *testing.T, handler *fakeRTPHandler) int {
	t.Helper()
	return len(handler.audioOut)
}

func newTestAudioProcessor(t *testing.T, codec *sip_runtime.Codec, resampler internal_type.AudioResampler) *AudioProcessor {
	t.Helper()
	rtp := testRTPHandler(t, codec)
	processor := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
	})
	processor.resamplers.provider = resampler
	processor.resamplers.assistant = resampler
	processor.resamplers.bridgeUser = resampler
	processor.resamplers.bridgeOperator = resampler
	return processor
}

// ---------------------------------------------------------------------------
// Tests: ProcessProviderAudioFrame
// ---------------------------------------------------------------------------

func TestProcessProviderAudioFrame_RealG711StreamingMatchesContinuousConversion(tester *testing.T) {
	for _, codec := range []sip_runtime.Codec{sip_runtime.CodecPCMU, sip_runtime.CodecPCMA} {
		for _, duration := range []int{5, 10, 20, 30, 40, 60, 0} {
			tester.Run(fmt.Sprintf("%s/%dms", codec.Name, duration), func(tester *testing.T) {
				durations := []int{5, 10, 20, 30, 40, 60}
				if duration > 0 {
					durations = []int{duration, duration, duration, duration}
				}
				var totalSamples int
				for _, frameDuration := range durations {
					totalSamples += frameDuration * 8
				}
				pcm := make([]byte, totalSamples*2)
				for sampleIndex := 0; sampleIndex < totalSamples; sampleIndex++ {
					sample := int16(12000 * math.Sin(2*math.Pi*440*float64(sampleIndex)/8000))
					binary.LittleEndian.PutUint16(pcm[sampleIndex*2:], uint16(sample))
				}
				encoded := g711.EncodeUlaw(pcm)
				if codec.Name == sip_runtime.CodecPCMA.Name {
					encoded = g711.EncodeAlaw(pcm)
				}
				processor := NewAudioProcessor(AudioProcessorConfig{
					RTPHandler: testRTPHandler(tester, &codec),
				})
				reference := resampler_soxr.New(resampler_soxr.WithQuickQuality())
				expected, err := reference.Resample(decodeG711ToLinear8k(encoded, codec.Name), Linear8kConfig, Rapida16kConfig)
				require.NoError(tester, err)
				var pipeline, recording []byte
				var offset int
				for _, frameDuration := range durations {
					frame, err := processor.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
						Audio: encoded[offset : offset+frameDuration*8], ReceivedAt: time.Now(),
					})
					require.NoError(tester, err)
					require.Equal(tester, frame.BridgeAudio, frame.PipelineAudio)
					require.Zero(tester, len(frame.PipelineAudio)%2)
					pipeline = append(pipeline, frame.PipelineAudio...)
					recording = append(recording, frame.BridgeAudio...)
					offset += frameDuration * 8
				}
				require.Equal(tester, len(expected), len(pipeline))
				require.Equal(tester, totalSamples*4, len(pipeline))
				require.True(tester, bytes.Equal(expected, pipeline), "chunking changed the PCM stream")
				require.Equal(tester, expected, recording)
			})
		}
	}
}

func TestProcessProviderAudioFrame_EmitsBridgeAndPipelineImmediately(t *testing.T) {
	resampledAudio := make([]byte, BridgeOutputFrameSize)
	for i := range resampledAudio {
		resampledAudio[i] = byte(i)
	}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: resampledAudio})
	receivedAt := time.Now()

	firstFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      make([]byte, MulawFrameSize),
		ReceivedAt: receivedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, resampledAudio, firstFrame.BridgeAudio)
	assert.Equal(t, resampledAudio, firstFrame.PipelineAudio)
	assert.Equal(t, receivedAt, firstFrame.ReceivedAt)

	secondFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      make([]byte, MulawFrameSize),
		ReceivedAt: receivedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, resampledAudio, secondFrame.BridgeAudio)
	assert.Equal(t, resampledAudio, secondFrame.PipelineAudio)
	secondFrame.PipelineAudio[0] ^= 0xff
	assert.Equal(t, resampledAudio, secondFrame.BridgeAudio)
	assert.Equal(t, resampledAudio, firstFrame.PipelineAudio)
}

func TestProcessProviderAudioFrame_PCMUDecodesToLinearBeforeResample(t *testing.T) {
	resampler := &captureResampler{out: []byte{0x01, 0x02, 0x03}}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, resampler)

	input := make([]byte, MulawFrameSize)
	for i := range input {
		input[i] = byte(i)
	}

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio: input,
	})

	require.NoError(t, err)
	assert.Equal(t, g711.DecodeUlaw(input), resampler.input)
	assert.Same(t, Linear8kConfig, resampler.from)
	assert.Same(t, Rapida16kConfig, resampler.to)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, inputFrame.BridgeAudio)
}

func TestProcessProviderAudioFrame_PCMADecodesToLinearBeforeResample(t *testing.T) {
	resampler := &captureResampler{out: []byte{0x04, 0x05, 0x06}}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, resampler)

	input := make([]byte, MulawFrameSize)
	for i := range input {
		input[i] = 0xD5
	}

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio: input,
	})

	require.NoError(t, err)
	assert.Equal(t, g711.DecodeAlaw(input), resampler.input)
	assert.Same(t, Linear8kConfig, resampler.from)
	assert.Same(t, Rapida16kConfig, resampler.to)
	assert.Equal(t, []byte{0x04, 0x05, 0x06}, inputFrame.BridgeAudio)
}

func TestProcessProviderAudioFrame_ResamplerError(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("resample failed")})

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio: make([]byte, MulawFrameSize),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderAudioConversionFailed))
	assert.Empty(t, inputFrame.BridgeAudio)
	assert.Empty(t, inputFrame.PipelineAudio)
}

// ---------------------------------------------------------------------------
// Tests: ProcessAssistantAudio
// ---------------------------------------------------------------------------

func TestProcessAssistantAudio_BuffersAssistantPCM16kFrames(t *testing.T) {
	providerAudio := make([]byte, MulawFrameSize*2)
	for index := range providerAudio {
		providerAudio[index] = byte(index)
	}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: providerAudio})
	proc.ambientMixer = nil

	audio := make([]byte, BridgeOutputFrameSize*2)
	for i := range audio {
		audio[i] = byte(i)
	}

	require.NoError(t, proc.ProcessAssistantAudio(audio, false))

	firstFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	assert.Equal(t, providerAudio[:MulawFrameSize], firstFrame.ProviderAudio)
	assert.Equal(t, audio[:BridgeOutputFrameSize], firstFrame.BridgeAudio)

	secondFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	assert.Equal(t, providerAudio[MulawFrameSize:], secondFrame.ProviderAudio)
	assert.Equal(t, audio[BridgeOutputFrameSize:], secondFrame.BridgeAudio)
}

func TestProcessAssistantAudio_BridgeActiveDoesNotRecordNormalOutput(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestProcessAssistantAudio_TransferActiveDoesNotRecordNormalOutput(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	proc.SetTransferActive(true)

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestConvertOutputAudio_PCMAConvertsResampledMulaw(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, &mockResampler{out: []byte{0xFF, 0x7F}})

	convertedAudio, err := proc.convertOutputAudio([]byte{1, 2})
	require.NoError(t, err)
	assert.Equal(t, internal_audio.UlawToAlaw([]byte{0xFF, 0x7F}), convertedAudio)
}

func TestProcessAssistantAudio_ResamplerError(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("fail")})

	err := proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAssistantAudioConversionFailed))
}

func TestProcessAssistantAudio_NextOutputFrameKeepsPipelineAudio(t *testing.T) {
	providerAudio := make([]byte, MulawFrameSize)
	for index := range providerAudio {
		providerAudio[index] = byte(index)
	}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: providerAudio})
	proc.ambientMixer = nil

	assistantAudio := make([]byte, BridgeOutputFrameSize)
	for i := range assistantAudio {
		assistantAudio[i] = byte(255 - i%256)
	}

	require.NoError(t, proc.ProcessAssistantAudio(assistantAudio, false))
	outputFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)

	assert.Equal(t, providerAudio, outputFrame.ProviderAudio)
	assert.Equal(t, assistantAudio, outputFrame.BridgeAudio)
	assert.False(t, outputFrame.Idle)
}

func TestProcessAssistantAudio_NextOutputFrameWaitsForPairedBuffers(t *testing.T) {
	providerAudio := bytes.Repeat([]byte{0x7F}, MulawFrameSize)
	assistantAudio := bytes.Repeat([]byte{0x35}, BridgeOutputFrameSize)
	processor := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: providerAudio})
	processor.ambientMixer = nil
	blockedBridge := &blockingFrameBuffer{
		buffer:  internal_telephony_output.NewBytesFrameBuffer(BridgeOutputFrameSize),
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	processor.bridgeOutputBuffer = blockedBridge

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- processor.ProcessAssistantAudio(assistantAudio, false)
	}()
	<-blockedBridge.entered

	type outputResult struct {
		frame internal_telephony_media.AssistantOutputFrame
		ok    bool
	}
	outputDone := make(chan outputResult, 1)
	go func() {
		frame, ok := processor.NextOutputFrame()
		outputDone <- outputResult{frame: frame, ok: ok}
	}()

	select {
	case result := <-outputDone:
		t.Fatalf("output returned before paired bridge write completed: %+v", result)
	case <-time.After(25 * time.Millisecond):
	}

	close(blockedBridge.release)
	require.NoError(t, <-writeDone)
	result := <-outputDone
	require.True(t, result.ok)
	assert.Equal(t, providerAudio, result.frame.ProviderAudio)
	assert.Equal(t, assistantAudio, result.frame.BridgeAudio)
}

func TestIdleOutputFrame_SilentDuringTransfer(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	proc.SetTransferActive(true)

	outputFrame, ok := proc.IdleOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
}

func TestComplete_PadsPartialFrameWithLinearSilence(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	proc.ambientMixer = nil

	require.NoError(t, proc.ProcessAssistantAudio([]byte{0x01}, true))

	outputFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	require.Len(t, outputFrame.ProviderAudio, MulawFrameSize)
	assert.Equal(t, byte(0x01), outputFrame.ProviderAudio[0])
	assert.Equal(t, byte(MulawSilenceByte), outputFrame.ProviderAudio[1])
	assert.Equal(t, byte(MulawSilenceByte), outputFrame.ProviderAudio[len(outputFrame.ProviderAudio)-1])
	require.Len(t, outputFrame.BridgeAudio, BridgeOutputFrameSize)
	assert.Equal(t, byte(0x01), outputFrame.BridgeAudio[0])
	assert.Equal(t, byte(0x00), outputFrame.BridgeAudio[len(outputFrame.BridgeAudio)-1])
}

// ---------------------------------------------------------------------------
// Tests: ClearOutputBuffer
// ---------------------------------------------------------------------------

func TestClearOutputBuffer_ResetsBuffer(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	proc.ClearOutputBuffer()

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestClearOutputBuffer_DoesNotInterruptInputAudio(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: make([]byte, BridgeOutputFrameSize)})

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Len(t, inputFrame.PipelineAudio, BridgeOutputFrameSize)
	proc.ClearOutputBuffer()

	inputFrame, err = proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Len(t, inputFrame.PipelineAudio, BridgeOutputFrameSize)
}

// ---------------------------------------------------------------------------
// Tests: ForwardUserAudio
// ---------------------------------------------------------------------------

func TestForwardUserAudio_NoBridge_ReturnsFalse(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	audio := []byte{0x01, 0x02, 0x03}
	result := proc.ForwardUserAudio(audio)
	assert.False(t, result, "should return false when no bridge is active")
}

func TestForwardUserAudio_BridgeActive_ReturnsTrue(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	audio := []byte{0x01, 0x02, 0x03}
	result := proc.ForwardUserAudio(audio)
	assert.True(t, result)

	// Check that audio was queued to bridgeUserCh
	select {
	case queued := <-proc.bridgeUserCh:
		assert.Equal(t, audio, queued.audio)
		assert.Equal(t, sip_runtime.CodecPCMU.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected audio on bridgeUserCh")
	}
}

func TestForwardUserAudio_BridgeActive_SuccessPath(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	audio := []byte{0xAA, 0xBB, 0xCC}
	// ForwardUserAudio enqueues bridge RTP and records only after enqueue succeeds.
	result := proc.ForwardUserAudio(audio)
	assert.True(t, result)

	select {
	case queued := <-proc.bridgeUserCh:
		assert.Equal(t, audio, queued.audio, "raw audio should be queued to bridgeUserCh")
		assert.Equal(t, sip_runtime.CodecPCMU.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected audio on bridgeUserCh")
	}
}

func TestForwardUserAudio_WithTranscode_PCMU_to_PCMA(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMA)
	// User has PCMU and the bridge target has PCMA, so transcode.
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMA.Name)

	audio := make([]byte, 160)
	for i := range audio {
		audio[i] = 0xFF // µ-law silence
	}
	result := proc.ForwardUserAudio(audio)
	assert.True(t, result, "should return true when bridge is active")

	// Raw (untranscoded) audio should still go to bridgeUserCh.
	// The transcode only applies to the bridge RTP enqueue. We verify the contract by confirming
	// that bridgeUserCh gets the original raw audio, proving the transcode path
	// is separate.
	select {
	case queued := <-proc.bridgeUserCh:
		assert.Equal(t, audio, queued.audio, "raw audio should go to bridgeUserCh without transcoding")
		assert.Equal(t, sip_runtime.CodecPCMU.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected raw audio on bridgeUserCh")
	}
}

func TestForwardUserAudio_Backpressure_DropsAudio(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	// Fill bridgeUserCh to capacity
	for i := 0; i < BridgeRecordingChannelCapacity; i++ {
		proc.bridgeUserCh <- bridgeRecordingFrame{audio: []byte{byte(i)}, codecName: sip_runtime.CodecPCMU.Name}
	}

	// Should not block even though channel is full
	done := make(chan struct{})
	go func() {
		proc.ForwardUserAudio([]byte{0xFF})
		close(done)
	}()

	select {
	case <-done:
		// The call completed without blocking.
	case <-time.After(time.Second):
		t.Fatal("ForwardUserAudio hung when bridgeUserCh was full")
	}
}

func TestForwardUserAudio_DoesNotRecordWhenBridgeWriteFails(t *testing.T) {
	records := make(chan observability.Record, 2)
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Record: func(record ...observability.Record) error {
			for _, item := range record {
				records <- item
			}
			return nil
		},
	})
	resampler := &mockResampler{}
	proc.resamplers.provider = resampler
	proc.resamplers.assistant = resampler
	proc.resamplers.bridgeUser = resampler
	proc.resamplers.bridgeOperator = resampler
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	for i := 0; i < 100; i++ {
		require.NoError(t, bridgeRTP.WriteAudio([]byte{byte(i)}))
	}
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	assert.True(t, proc.ForwardUserAudio([]byte{0xaa}))

	select {
	case recorded := <-proc.bridgeUserCh:
		t.Fatalf("dropped bridge RTP frame was queued for recording: %v", recorded)
	default:
	}
	select {
	case record := <-records:
		log, ok := record.(observability.RecordLog)
		require.True(t, ok)
		assert.Equal(t, "bridge_audio_write_failed", log.Attributes["reason"])
	case <-time.After(time.Second):
		t.Fatal("expected observability record")
	}
}

// ---------------------------------------------------------------------------
// Tests: RecordTransferOperatorAudio
// ---------------------------------------------------------------------------

func TestRecordTransferOperatorAudio_QueuesAudio(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	audio := []byte{0x10, 0x20, 0x30}
	proc.RecordTransferOperatorAudio(audio)

	select {
	case queued := <-proc.bridgeOperatorCh:
		assert.Equal(t, audio, queued.audio)
		assert.Equal(t, sip_runtime.CodecPCMU.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected audio on bridgeOperatorCh")
	}
}

func TestRecordTransferOperatorAudio_Backpressure_DropsAudio(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	// Fill channel
	for i := 0; i < BridgeRecordingChannelCapacity; i++ {
		proc.bridgeOperatorCh <- bridgeRecordingFrame{audio: []byte{byte(i)}, codecName: sip_runtime.CodecPCMU.Name}
	}

	// Should not block
	done := make(chan struct{})
	go func() {
		proc.RecordTransferOperatorAudio([]byte{0xFF})
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("RecordTransferOperatorAudio hung when bridgeOperatorCh was full")
	}
}

// ---------------------------------------------------------------------------
// Tests: ConnectTransferMedia / DisconnectTransferMedia / IsBridgeActive
// ---------------------------------------------------------------------------

func TestConnectTransferMedia_ActivatesBridge(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	assert.False(t, proc.IsBridgeActive())

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	assert.True(t, proc.IsBridgeActive())
}

func TestConnectTransferMedia_NilRTP_DoesNotActivate(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	proc.ConnectTransferMedia(nil, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)
	assert.False(t, proc.IsBridgeActive())
}

func TestDisconnectTransferMedia_DeactivatesBridge(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)
	assert.True(t, proc.IsBridgeActive())

	proc.DisconnectTransferMedia()
	assert.False(t, proc.IsBridgeActive())

	// ForwardUserAudio should now return false
	assert.False(t, proc.ForwardUserAudio([]byte{0x01}))
}

// TestForwardUserAudio_ConcurrentClear verifies the bridgeMu invariant: while
// any ForwardUserAudio call is in flight, DisconnectTransferMedia must NOT return.
// This guarantees a caller can safely close the outbound RTP channel after
// DisconnectTransferMedia returns without racing into a "send on closed channel"
// panic. Run with `-race` for full coverage.
func TestForwardUserAudio_ConcurrentClear(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	const writers = 8
	const iters = 200

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				select {
				case <-stop:
					return
				default:
				}
				proc.ForwardUserAudio([]byte{0x01, 0x02})
			}
		}()
	}

	// Race: while writers are spinning, clear the bridge target. Clear must
	// block until any in-flight ForwardUserAudio releases the mutex; no
	// writer should observe a panic. Race detector catches any unsynchronized
	// access to p.bridge.
	time.Sleep(2 * time.Millisecond)
	proc.DisconnectTransferMedia()
	assert.False(t, proc.IsBridgeActive(), "DisconnectTransferMedia should leave bridge inactive")

	close(stop)
	wg.Wait()

	// Subsequent ForwardUserAudio returns false; no panic.
	assert.False(t, proc.ForwardUserAudio([]byte{0x03}))
}

func TestConnectTransferMedia_MatchingCodecs_NoTranscode(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	// With matching codecs, bridgeUserCh should receive the original audio unchanged
	// (the raw audio IS the same as what goes to outRTP when no transcode is needed).
	audio := []byte{0x01, 0x02, 0x03, 0x04}
	result := proc.ForwardUserAudio(audio)
	assert.True(t, result)

	select {
	case queued := <-proc.bridgeUserCh:
		assert.Equal(t, audio, queued.audio, "matching codecs should not alter raw audio")
		assert.Equal(t, sip_runtime.CodecPCMU.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected audio on bridgeUserCh")
	}
}

func TestConnectTransferMedia_PCMA_to_PCMU_Transcode(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, &mockResampler{})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	// inCodec=PCMA, outCodec=PCMU means A-law → µ-law transcode for outRTP
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMA, sip_runtime.CodecPCMU.Name)

	audio := make([]byte, 160)
	for i := range audio {
		audio[i] = 0xD5 // A-law silence
	}
	result := proc.ForwardUserAudio(audio)
	assert.True(t, result)

	// bridgeUserCh always gets the raw audio; transcode is limited to bridge RTP enqueue.
	select {
	case queued := <-proc.bridgeUserCh:
		assert.Equal(t, audio, queued.audio, "bridgeUserCh should get raw untranscoded audio")
		assert.Equal(t, sip_runtime.CodecPCMA.Name, queued.codecName)
	case <-time.After(time.Second):
		t.Fatal("expected raw audio on bridgeUserCh")
	}
}

// ---------------------------------------------------------------------------
// Tests: Ringback
// ---------------------------------------------------------------------------

func TestRingback_IsProducedByOutputPacer(t *testing.T) {
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
	})

	proc.StartRingback()
	proc.SetTransferActive(true)
	frame, ok := proc.IdleOutputFrame()
	require.True(t, ok)
	assert.Len(t, frame.ProviderAudio, MulawFrameSize)
	assert.True(t, frame.Idle)

	proc.StopRingback()
	_, ok = proc.IdleOutputFrame()
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Tests: RunBridgeRecorder
// ---------------------------------------------------------------------------

func TestRunBridgeRecorder_ExitsOnContextCancel(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		proc.RunBridgeRecorder(ctx, rec.push)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("RunBridgeRecorder did not exit after context cancellation")
	}
}

func TestRunBridgeRecorder_UserAudio_PushesConversationBridgeUserAudio(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx, rec.push)

	// Send user audio
	audio := []byte{0x01, 0x02, 0x03}
	proc.bridgeUserCh <- bridgeRecordingFrame{audio: audio, codecName: sip_runtime.CodecPCMU.Name}

	// Wait for the recording sink to be called.
	require.Eventually(t, func() bool {
		return len(rec.get()) >= 1
	}, time.Second, 10*time.Millisecond)

	streams := rec.get()
	require.Len(t, streams, 1)

	msg, ok := streams[0].(*protos.ConversationBridgeUserAudio)
	require.True(t, ok, "expected ConversationBridgeUserAudio, got %T", streams[0])
	assert.Equal(t, g711.DecodeUlaw(audio), msg.Audio)
	assert.False(t, msg.Time.AsTime().IsZero())
}

func TestRunBridgeRecorder_UserAudio_DecodesPCMARecording(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, &mockResampler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx, rec.push)

	audio := []byte{0xD5, 0xD4, 0xD3}
	proc.bridgeUserCh <- bridgeRecordingFrame{audio: audio, codecName: sip_runtime.CodecPCMA.Name}

	require.Eventually(t, func() bool {
		return len(rec.get()) >= 1
	}, time.Second, 10*time.Millisecond)

	streams := rec.get()
	require.Len(t, streams, 1)

	msg, ok := streams[0].(*protos.ConversationBridgeUserAudio)
	require.True(t, ok, "expected ConversationBridgeUserAudio, got %T", streams[0])
	assert.Equal(t, g711.DecodeAlaw(audio), msg.Audio)
}

func TestRunBridgeRecorder_OperatorAudio_PushesConversationBridgeOperatorAudio(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx, rec.push)

	// Send operator audio
	audio := []byte{0x10, 0x20, 0x30}
	proc.bridgeOperatorCh <- bridgeRecordingFrame{audio: audio, codecName: sip_runtime.CodecPCMU.Name}

	require.Eventually(t, func() bool {
		return len(rec.get()) >= 1
	}, time.Second, 10*time.Millisecond)

	streams := rec.get()
	require.Len(t, streams, 1)

	msg, ok := streams[0].(*protos.ConversationBridgeOperatorAudio)
	require.True(t, ok, "expected ConversationBridgeOperatorAudio, got %T", streams[0])
	assert.Equal(t, g711.DecodeUlaw(audio), msg.Audio)
	assert.False(t, msg.Time.AsTime().IsZero())
}

func TestRunBridgeRecorder_OperatorAudio_DecodesConnectedBridgeCodec(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMA)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMA.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx, rec.push)

	audio := []byte{0xD5, 0xD4, 0xD3}
	proc.RecordTransferOperatorAudio(audio)

	require.Eventually(t, func() bool {
		return len(rec.get()) >= 1
	}, time.Second, 10*time.Millisecond)

	streams := rec.get()
	require.Len(t, streams, 1)

	msg, ok := streams[0].(*protos.ConversationBridgeOperatorAudio)
	require.True(t, ok, "expected ConversationBridgeOperatorAudio, got %T", streams[0])
	assert.Equal(t, g711.DecodeAlaw(audio), msg.Audio)
}

func TestRunBridgeRecorder_ResamplerError_DoesNotPush(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("fail")})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx, rec.push)

	proc.bridgeUserCh <- bridgeRecordingFrame{audio: []byte{0x01}, codecName: sip_runtime.CodecPCMU.Name}
	proc.bridgeOperatorCh <- bridgeRecordingFrame{audio: []byte{0x02}, codecName: sip_runtime.CodecPCMU.Name}

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	streams := rec.get()
	assert.Empty(t, streams, "should not push when resampler fails")
}

// ---------------------------------------------------------------------------
// Tests: NewAudioProcessor contract
// ---------------------------------------------------------------------------

func TestNewAudioProcessor_InitializesChannels(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	assert.NotNil(t, proc.bridgeUserCh)
	assert.NotNil(t, proc.bridgeOperatorCh)
	assert.Equal(t, BridgeRecordingChannelCapacity, cap(proc.bridgeUserCh))
	assert.Equal(t, BridgeRecordingChannelCapacity, cap(proc.bridgeOperatorCh))
	assert.Equal(t, ChunkDuration, proc.OutputFrameDuration())
	assert.False(t, proc.IsBridgeActive())
}

func TestApplyAmbient_NoPrimary_WithAmbientConfig_ProducesFrame(t *testing.T) {
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: testRTPHandler(t, &sip_runtime.CodecPCMU),
		Ambient: &internal_ambient.Config{
			Profile: "office",
			Volume:  20,
			Enabled: true,
		},
	})

	out := proc.applyAmbient(nil)
	require.NotNil(t, out)
	assert.Len(t, out, MulawFrameSize)
}

func TestApplyAmbient_NoPrimary_NoAmbient_ReturnsNil(t *testing.T) {
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: testRTPHandler(t, &sip_runtime.CodecPCMU),
	})

	out := proc.applyAmbient(nil)
	assert.Nil(t, out)
}

func TestApplyAmbient_PrimaryWithNoAmbient_ReturnsInput(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})

	frame := []byte{1, 2, 3}
	assert.Equal(t, frame, proc.applyAmbient(frame))
}

func TestApplyAmbient_MixErrorReturnsInput(t *testing.T) {
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{})
	proc.ambientMixer = &mockAmbientMixer{err: errors.New("mix failed")}

	frame := []byte{1, 2, 3}
	assert.Equal(t, frame, proc.applyAmbient(frame))
}

// ---------------------------------------------------------------------------
// Benchmarks for hot-path audio processing called every 20ms per RTP packet.
// ---------------------------------------------------------------------------

func benchAudioProcessor(b *testing.B, codec *sip_runtime.Codec) *AudioProcessor {
	b.Helper()
	processor := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: newTestRTPHandler(codec),
	})
	resampler := &mockResampler{out: make([]byte, BridgeOutputFrameSize)}
	processor.resamplers.provider = resampler
	processor.resamplers.assistant = resampler
	processor.resamplers.bridgeUser = resampler
	processor.resamplers.bridgeOperator = resampler
	return processor
}

// BenchmarkProcessProviderAudioFrame_PCMU measures the per-packet input processing for PCMU.
func BenchmarkProcessProviderAudioFrame_PCMU(b *testing.B) {
	proc := benchAudioProcessor(b, &sip_runtime.CodecPCMU)
	frame := make([]byte, MulawFrameSize)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: frame})
	}
}

// BenchmarkProcessProviderAudioFrame_PCMA measures the per-packet input processing for PCMA.
func BenchmarkProcessProviderAudioFrame_PCMA(b *testing.B) {
	proc := benchAudioProcessor(b, &sip_runtime.CodecPCMA)
	frame := make([]byte, MulawFrameSize)
	for i := range frame {
		frame[i] = 0xD5
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: frame})
	}
}

// BenchmarkProcessAssistantAudio measures output buffering throughput.
func BenchmarkProcessAssistantAudio(b *testing.B) {
	proc := benchAudioProcessor(b, &sip_runtime.CodecPCMU)
	frame := make([]byte, BridgeOutputFrameSize)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proc.ProcessAssistantAudio(frame, false)
		if i%1000 == 0 {
			proc.ClearOutputBuffer()
		}
	}
}

// BenchmarkForwardUserAudio_NoBridge measures the no-bridge fast-exit path.
func BenchmarkForwardUserAudio_NoBridge(b *testing.B) {
	proc := benchAudioProcessor(b, &sip_runtime.CodecPCMU)
	frame := make([]byte, MulawFrameSize)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proc.ForwardUserAudio(frame)
	}
}

// BenchmarkForwardUserAudio_BridgeActive measures the bridge forwarding hot path.
func BenchmarkForwardUserAudio_BridgeActive(b *testing.B) {
	rtp := newTestRTPHandler(&sip_runtime.CodecPCMU)
	bridgeRTP := newTestRTPHandler(&sip_runtime.CodecPCMU)

	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
	})
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	frame := make([]byte, MulawFrameSize)

	// Drain channels in background to prevent backpressure blocking
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-proc.bridgeUserCh:
			}
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proc.ForwardUserAudio(frame)
	}
}
