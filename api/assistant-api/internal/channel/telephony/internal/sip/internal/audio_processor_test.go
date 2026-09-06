// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
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

func (m *mockResampler) Resample(data []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.out != nil {
		return append([]byte(nil), m.out...), nil
	}
	// passthrough — just return the same data
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

// pushRecorder captures all streams pushed via the pushInput callback.
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
	audioIn      chan []byte
	audioOut     chan []byte
	fallback     sip_runtime.RTPFallbackAudioSource
	localAddress sip_runtime.RTPAddress
}

func newTestRTPHandler(codec *sip_runtime.Codec) *fakeRTPHandler {
	if codec == nil {
		codec = &sip_runtime.CodecPCMU
	}
	return &fakeRTPHandler{
		codec:    codec,
		audioIn:  make(chan []byte, 100),
		audioOut: make(chan []byte, 100),
	}
}

func (h *fakeRTPHandler) AudioIn() <-chan []byte       { return h.audioIn }
func (h *fakeRTPHandler) GetCodec() *sip_runtime.Codec { return h.codec }
func (h *fakeRTPHandler) LocalAddress() sip_runtime.RTPAddress {
	return h.localAddress
}
func (h *fakeRTPHandler) SetFallbackAudioSource(source sip_runtime.RTPFallbackAudioSource) {
	h.fallback = source
}
func (h *fakeRTPHandler) ClearFallbackAudioSource() { h.fallback = nil }
func (h *fakeRTPHandler) FlushAudioOut() {
	for {
		select {
		case <-h.audioOut:
		default:
			return
		}
	}
}
func (h *fakeRTPHandler) EnqueueAudio(audio []byte) error {
	select {
	case h.audioOut <- audio:
		return nil
	default:
		return sip_runtime.ErrRTPOutputQueueFull
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

func newTestAudioProcessor(t *testing.T, codec *sip_runtime.Codec, resampler internal_type.AudioResampler, recorder *pushRecorder) *AudioProcessor {
	t.Helper()
	rtp := testRTPHandler(t, codec)
	return NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  resampler,
		PushInput:  recorder.push,
	})
}

// ---------------------------------------------------------------------------
// Tests: ProcessProviderAudioFrame
// ---------------------------------------------------------------------------

func TestProcessProviderAudioFrame_EmitsBridgeImmediatelyAndBuffersPipelineAudio(t *testing.T) {
	rec := &pushRecorder{}
	resampledAudio := make([]byte, BridgeOutputFrameSize)
	for i := range resampledAudio {
		resampledAudio[i] = byte(i)
	}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: resampledAudio}, rec)
	receivedAt := time.Now()

	firstFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      make([]byte, MulawFrameSize),
		ReceivedAt: receivedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, resampledAudio, firstFrame.BridgeAudio)
	assert.Empty(t, firstFrame.PipelineAudio)
	assert.Equal(t, receivedAt, firstFrame.ReceivedAt)

	secondFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      make([]byte, MulawFrameSize),
		ReceivedAt: receivedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, resampledAudio, secondFrame.BridgeAudio)
	assert.Len(t, secondFrame.PipelineAudio, InputBufferThreshold)
}

func TestProcessProviderAudioFrame_PCMUDecodesToLinearBeforeResample(t *testing.T) {
	rec := &pushRecorder{}
	resampler := &captureResampler{out: []byte{0x01, 0x02, 0x03}}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, resampler, rec)

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
	rec := &pushRecorder{}
	resampler := &captureResampler{out: []byte{0x04, 0x05, 0x06}}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, resampler, rec)

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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("resample failed")}, rec)

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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.ambientMixer = nil

	audio := make([]byte, BridgeOutputFrameSize*2)
	for i := range audio {
		audio[i] = byte(i)
	}

	require.NoError(t, proc.ProcessAssistantAudio(audio, false))

	firstFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	assert.Equal(t, audio[:BridgeOutputFrameSize], firstFrame.ProviderAudio)
	assert.Empty(t, firstFrame.BridgeAudio)

	secondFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	assert.Equal(t, audio[BridgeOutputFrameSize:], secondFrame.ProviderAudio)
	assert.Empty(t, secondFrame.BridgeAudio)
}

func TestProcessAssistantAudio_BridgeActiveDoesNotRecordNormalOutput(t *testing.T) {
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestProcessAssistantAudio_TransferActiveDoesNotRecordNormalOutput(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.SetTransferActive(true)

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestEncodeAssistantOutputFrame_PCMAConvertsToAlaw(t *testing.T) {
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMA)
	resampledAudio := []byte{0xFF, 0x7F}
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{out: resampledAudio},
		PushInput:  rec.push,
	})

	providerAudio, err := proc.encodeAssistantOutputFrame(make([]byte, BridgeOutputFrameSize))
	require.NoError(t, err)
	assert.Equal(t, internal_audio.UlawToAlaw(resampledAudio), providerAudio)
}

func TestConvertOutputAudio_PCMAConvertsResampledMulaw(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, &mockResampler{out: []byte{0xFF, 0x7F}}, rec)

	convertedAudio, err := proc.convertOutputAudio([]byte{1, 2})
	require.NoError(t, err)
	assert.Equal(t, internal_audio.UlawToAlaw([]byte{0xFF, 0x7F}), convertedAudio)
}

func TestEncodeAssistantOutputFrame_ResamplerError(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("fail")}, rec)

	_, err := proc.encodeAssistantOutputFrame(make([]byte, BridgeOutputFrameSize))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAssistantAudioConversionFailed))
}

func TestProcessAssistantAudio_NextOutputFrameKeepsPipelineAudio(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.ambientMixer = nil

	assistantAudio := make([]byte, BridgeOutputFrameSize)
	for i := range assistantAudio {
		assistantAudio[i] = byte(255 - i%256)
	}

	require.NoError(t, proc.ProcessAssistantAudio(assistantAudio, false))
	outputFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)

	assert.Equal(t, assistantAudio, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
	assert.False(t, outputFrame.Idle)
}

func TestIdleOutputFrame_SilentDuringTransfer(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.SetTransferActive(true)

	outputFrame, ok := proc.IdleOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
}

func TestComplete_PadsPartialFrameWithLinearSilence(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.ambientMixer = nil

	require.NoError(t, proc.ProcessAssistantAudio([]byte{0x01}, true))

	outputFrame, ok := proc.NextOutputFrame()
	require.True(t, ok)
	require.Len(t, outputFrame.ProviderAudio, BridgeOutputFrameSize)
	assert.Equal(t, byte(0x01), outputFrame.ProviderAudio[0])
	assert.Equal(t, byte(0x00), outputFrame.ProviderAudio[1])
	assert.Equal(t, byte(0x00), outputFrame.ProviderAudio[len(outputFrame.ProviderAudio)-1])
}

// ---------------------------------------------------------------------------
// Tests: ClearOutputBuffer
// ---------------------------------------------------------------------------

func TestClearOutputBuffer_ResetsBuffer(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	require.NoError(t, proc.ProcessAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	proc.ClearOutputBuffer()

	outputFrame, ok := proc.NextOutputFrame()
	assert.False(t, ok)
	assert.Empty(t, outputFrame.ProviderAudio)
	assert.Empty(t, outputFrame.BridgeAudio)
}

func TestClearOutputBuffer_PreservesInputBuffer(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: make([]byte, BridgeOutputFrameSize)}, rec)

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Empty(t, inputFrame.PipelineAudio)
	proc.ClearOutputBuffer()

	inputFrame, err = proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Len(t, inputFrame.PipelineAudio, InputBufferThreshold)
}

func TestClearInputBuffer_ResetsInputBuffer(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{out: make([]byte, BridgeOutputFrameSize)}, rec)

	inputFrame, err := proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Empty(t, inputFrame.PipelineAudio)
	proc.ClearInputBuffer()

	inputFrame, err = proc.ProcessProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{Audio: make([]byte, MulawFrameSize)})
	require.NoError(t, err)
	assert.Empty(t, inputFrame.PipelineAudio)
}

func TestRTPOutputQueueFullError_ReturnsStructuredError(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	err := proc.rtpOutputQueueFullError()

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRTPOutputQueueFull))
}

// ---------------------------------------------------------------------------
// Tests: ForwardUserAudio
// ---------------------------------------------------------------------------

func TestForwardUserAudio_NoBridge_ReturnsFalse(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	audio := []byte{0x01, 0x02, 0x03}
	result := proc.ForwardUserAudio(audio)
	assert.False(t, result, "should return false when no bridge is active")
}

func TestForwardUserAudio_BridgeActive_ReturnsTrue(t *testing.T) {
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

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
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

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
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMA)
	// User has PCMU, bridge target has PCMA — need transcode
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
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	// Fill bridgeUserCh to capacity
	for i := 0; i < AudioChannelSize; i++ {
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
		// OK — did not hang
	case <-time.After(time.Second):
		t.Fatal("ForwardUserAudio hung when bridgeUserCh was full")
	}
}

func TestForwardUserAudio_DoesNotRecordWhenBridgeRTPQueueFull(t *testing.T) {
	rec := &pushRecorder{}
	records := make(chan observability.Record, 2)
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
		Record: func(record ...observability.Record) error {
			for _, item := range record {
				records <- item
			}
			return nil
		},
	})
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	for i := 0; i < 100; i++ {
		require.NoError(t, bridgeRTP.EnqueueAudio([]byte{byte(i)}))
	}
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	assert.True(t, proc.ForwardUserAudio([]byte{0xaa}))

	select {
	case recorded := <-proc.bridgeUserCh:
		t.Fatalf("dropped bridge RTP frame was queued for recording: %v", recorded)
	default:
	}
	require.Empty(t, rec.get())
	select {
	case record := <-records:
		log, ok := record.(observability.RecordLog)
		require.True(t, ok)
		assert.Equal(t, "bridge_audio_out_full", log.Attributes["reason"])
	case <-time.After(time.Second):
		t.Fatal("expected observability record")
	}
}

// ---------------------------------------------------------------------------
// Tests: RecordTransferOperatorAudio
// ---------------------------------------------------------------------------

func TestRecordTransferOperatorAudio_QueuesAudio(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	// Fill channel
	for i := 0; i < AudioChannelSize; i++ {
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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	assert.False(t, proc.IsBridgeActive())

	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)

	assert.True(t, proc.IsBridgeActive())
}

func TestConnectTransferMedia_NilRTP_DoesNotActivate(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	proc.ConnectTransferMedia(nil, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMU.Name)
	assert.False(t, proc.IsBridgeActive())
}

func TestDisconnectTransferMedia_DeactivatesBridge(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

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
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

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
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMA)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

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

func TestRingback_UsesRTPFallbackSourceWithoutQueueProducer(t *testing.T) {
	rec := &pushRecorder{}
	rtp := testRTPHandler(t, &sip_runtime.CodecPCMU)
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: rtp,
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

	proc.StartRingback()
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, 0, rtpAudioOutLen(t, rtp))

	proc.StopRingback()
}

// ---------------------------------------------------------------------------
// Tests: RunBridgeRecorder
// ---------------------------------------------------------------------------

func TestRunBridgeRecorder_ExitsOnContextCancel(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		proc.RunBridgeRecorder(ctx)
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
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx)

	// Send user audio
	audio := []byte{0x01, 0x02, 0x03}
	proc.bridgeUserCh <- bridgeRecordingFrame{audio: audio, codecName: sip_runtime.CodecPCMU.Name}

	// Wait for pushInput to be called
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
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMA, &mockResampler{}, rec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx)

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
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx)

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
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	bridgeRTP := testRTPHandler(t, &sip_runtime.CodecPCMA)
	proc.ConnectTransferMedia(bridgeRTP, &sip_runtime.CodecPCMU, sip_runtime.CodecPCMA.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx)

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
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{err: errors.New("fail")}, rec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go proc.RunBridgeRecorder(ctx)

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
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	assert.NotNil(t, proc.inputBuffer)
	assert.NotNil(t, proc.bridgeUserCh)
	assert.NotNil(t, proc.bridgeOperatorCh)
	assert.Equal(t, AudioChannelSize, cap(proc.bridgeUserCh))
	assert.Equal(t, AudioChannelSize, cap(proc.bridgeOperatorCh))
	assert.False(t, proc.IsBridgeActive())
}

func TestApplyAmbient_NoPrimary_WithAmbientConfig_ProducesFrame(t *testing.T) {
	rec := &pushRecorder{}
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: testRTPHandler(t, &sip_runtime.CodecPCMU),
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
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
	rec := &pushRecorder{}
	proc := NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: testRTPHandler(t, &sip_runtime.CodecPCMU),
		Resampler:  &mockResampler{},
		PushInput:  rec.push,
	})

	out := proc.applyAmbient(nil)
	assert.Nil(t, out)
}

func TestApplyAmbient_PrimaryWithNoAmbient_ReturnsInput(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)

	frame := []byte{1, 2, 3}
	assert.Equal(t, frame, proc.applyAmbient(frame))
}

func TestApplyAmbient_MixErrorReturnsInput(t *testing.T) {
	rec := &pushRecorder{}
	proc := newTestAudioProcessor(t, &sip_runtime.CodecPCMU, &mockResampler{}, rec)
	proc.ambientMixer = &mockAmbientMixer{err: errors.New("mix failed")}

	frame := []byte{1, 2, 3}
	assert.Equal(t, frame, proc.applyAmbient(frame))
}

// ---------------------------------------------------------------------------
// Benchmarks — hot-path audio processing (called every 20ms per RTP packet)
// ---------------------------------------------------------------------------

func benchAudioProcessor(b *testing.B, codec *sip_runtime.Codec) *AudioProcessor {
	b.Helper()
	return NewAudioProcessor(AudioProcessorConfig{
		RTPHandler: newTestRTPHandler(codec),
		Resampler:  &mockResampler{},
		PushInput:  func(internal_type.Stream) {},
	})
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
		Resampler:  &mockResampler{},
		PushInput:  func(internal_type.Stream) {},
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
