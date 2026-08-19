// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"context"
	"encoding/binary"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// ============================================================================
// Test helpers
// ============================================================================

func userInput(msg string) internal_type.UserTextReceivedPacket {
	return internal_type.UserTextReceivedPacket{Text: msg}
}

func sttInput(msg string, complete bool) internal_type.SpeechToTextPacket {
	return internal_type.SpeechToTextPacket{Script: msg, Interim: !complete}
}

func interruptInput() internal_type.EndOfSpeechInterruptionPacket {
	return internal_type.EndOfSpeechInterruptionPacket{Source: "vad"}
}

func audioInput(nSamples int) internal_type.EndOfSpeechAudioPacket {
	pcm := make([]byte, nSamples*2)
	for i := 0; i < nSamples; i++ {
		// 440 Hz sine wave as PCM16
		v := int16(16000.0 * math.Sin(2.0*math.Pi*440.0*float64(i)/16000.0))
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(v))
	}
	return internal_type.EndOfSpeechAudioPacket{Audio: pcm}
}

func newTestOpts(m map[string]any) utils.Option {
	return utils.Option(m)
}

type testPredictor struct {
	predict func([]float32) (float64, error)
}

func (predictor testPredictor) Predict(audio []float32) (float64, error) {
	return predictor.predict(audio)
}

func newTestEOS(onPacket func(context.Context, ...internal_type.Packet) error, opts utils.Option) *pipecatEndOfSpeech {
	fallbackTimeout := time.Duration(defaultPctFallbackTimeout) * time.Millisecond
	if v, err := opts.GetFloat64("microphone.eos.fallback_timeout"); err == nil {
		fallbackTimeout = time.Duration(v) * time.Millisecond
	} else if v, err := opts.GetFloat64("microphone.eos.timeout"); err == nil {
		fallbackTimeout = time.Duration(v) * time.Millisecond
	}
	extendedTimeout := time.Duration(defaultPctExtendedTimeout) * time.Millisecond
	if v, err := opts.GetFloat64("microphone.eos.extended_timeout"); err == nil {
		extendedTimeout = time.Duration(v) * time.Millisecond
	} else if v, err := opts.GetFloat64("microphone.eos.silence_timeout"); err == nil {
		extendedTimeout = time.Duration(v) * time.Millisecond
	}
	quickTimeout := time.Duration(defaultPctQuickTimeout) * time.Millisecond
	if v, err := opts.GetFloat64("microphone.eos.quick_timeout"); err == nil {
		quickTimeout = time.Duration(v) * time.Millisecond
	}
	threshold := defaultPctThreshold
	if v, err := opts.GetFloat64("microphone.eos.threshold"); err == nil {
		threshold = v
	}

	eos := &pipecatEndOfSpeech{
		onPacket:        onPacket,
		opts:            opts,
		threshold:       threshold,
		quickTimeout:    quickTimeout,
		extendedTimeout: extendedTimeout,
		fallbackTimeout: fallbackTimeout,
		audioBuffer:     make([]float32, 0, maxAudioSamples),
		commandCh:       make(chan workerCommand, 32),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}
	go eos.worker()
	return eos
}

func newTestEOSWithPredictor(
	callback func(context.Context, ...internal_type.Packet) error,
	opts utils.Option,
	predictor func([]float32) (float64, error),
) *pipecatEndOfSpeech {
	eos := newTestEOS(callback, opts)
	eos.predictor = testPredictor{predict: predictor}
	return eos
}

func closeTestEndOfSpeech(eos *pipecatEndOfSpeech) {
	_ = eos.Close(context.Background())
}

// ============================================================================
// MEL SPECTROGRAM TESTS
// ============================================================================

func TestHzToMel_LinearRegion(t *testing.T) {
	assert.InDelta(t, 0.0, hzToMel(0), 1e-6)
	assert.InDelta(t, 500.0/melFSP, hzToMel(500), 1e-6)
	assert.InDelta(t, 1000.0/melFSP, hzToMel(1000), 1e-6)
}

func TestHzToMel_LogRegion(t *testing.T) {
	mel4000 := hzToMel(4000)
	assert.True(t, mel4000 > hzToMel(1000))
	mel8000 := hzToMel(8000)
	assert.True(t, mel8000 > mel4000)
}

func TestMelToHz_Roundtrip(t *testing.T) {
	freqs := []float64{0, 50, 100, 250, 500, 750, 1000, 1500, 2000, 4000, 6000, 8000}
	for _, f := range freqs {
		got := melToHz(hzToMel(f))
		assert.InDelta(t, f, got, 1e-6, "roundtrip failed for %f Hz", f)
	}
}

func TestPrepareAudio_ExactLength(t *testing.T) {
	audio := make([]float32, whisperMaxSamples)
	for i := range audio {
		audio[i] = 0.5
	}
	result := prepareAudio(audio)
	assert.Len(t, result, whisperMaxSamples)
	assert.Equal(t, float32(0.5), result[0])
}

func TestPrepareAudio_Truncation(t *testing.T) {
	audio := make([]float32, whisperMaxSamples+1000)
	for i := range audio {
		audio[i] = float32(i)
	}
	result := prepareAudio(audio)
	assert.Len(t, result, whisperMaxSamples)
	assert.Equal(t, float32(1000), result[0])
	assert.Equal(t, float32(whisperMaxSamples+999), result[whisperMaxSamples-1])
}

func TestPrepareAudio_Padding(t *testing.T) {
	audio := make([]float32, 1000)
	for i := range audio {
		audio[i] = 1.0
	}
	result := prepareAudio(audio)
	assert.Len(t, result, whisperMaxSamples)
	// Left side should be zeros
	assert.Equal(t, float32(0), result[0])
	assert.Equal(t, float32(0), result[whisperMaxSamples-1001])
	// Right side should be the audio
	assert.Equal(t, float32(1.0), result[whisperMaxSamples-1000])
	assert.Equal(t, float32(1.0), result[whisperMaxSamples-1])
}

func TestNormalize_ZeroMean(t *testing.T) {
	samples := []float32{1, 2, 3, 4, 5}
	normalize(samples)

	var sum float64
	for _, s := range samples {
		sum += float64(s)
	}
	assert.InDelta(t, 0.0, sum/float64(len(samples)), 1e-5)
}

func TestNormalize_UnitVariance(t *testing.T) {
	samples := []float32{-2, -1, 0, 1, 2}
	normalize(samples)

	var variance float64
	for _, s := range samples {
		variance += float64(s) * float64(s)
	}
	variance /= float64(len(samples))
	assert.InDelta(t, 1.0, variance, 0.05)
}

func TestNormalize_AllSame(t *testing.T) {
	samples := []float32{5, 5, 5, 5}
	normalize(samples)
	// All same value → stddev ≈ 0 → all outputs should be ≈ 0
	for _, s := range samples {
		assert.InDelta(t, 0.0, s, 1e-2)
	}
}

func TestNormalize_Empty(t *testing.T) {
	var samples []float32
	normalize(samples) // should not panic
}

func TestReflectPad(t *testing.T) {
	signal := []float32{1, 2, 3, 4, 5}
	padded := reflectPad(signal, 2)

	assert.Len(t, padded, 9)
	assert.Equal(t, float32(3), padded[0])
	assert.Equal(t, float32(2), padded[1])
	assert.Equal(t, float32(1), padded[2])
	assert.Equal(t, float32(5), padded[6])
	assert.Equal(t, float32(4), padded[7])
	assert.Equal(t, float32(3), padded[8])
}

func TestReflectPad_SingleElement(t *testing.T) {
	signal := []float32{42}
	padded := reflectPad(signal, 3)
	assert.Len(t, padded, 7)
	// With single element, reflect is just the element itself
	assert.Equal(t, float32(42), padded[3])
}

func TestReflectPad_ZeroPad(t *testing.T) {
	signal := []float32{1, 2, 3}
	padded := reflectPad(signal, 0)
	assert.Equal(t, signal, padded)
}

// ============================================================================
// FFT TESTS
// ============================================================================

func TestFFT_Impulse(t *testing.T) {
	// FFT of [1, 0, 0, 0] = [1, 1, 1, 1] (flat spectrum)
	x := []complex128{1, 0, 0, 0}
	fft(x)
	for i := range x {
		assert.InDelta(t, 1.0, real(x[i]), 1e-10)
		assert.InDelta(t, 0.0, imag(x[i]), 1e-10)
	}
}

func TestFFT_DC(t *testing.T) {
	// FFT of [1, 1, 1, 1] = [4, 0, 0, 0] (all energy at DC)
	x := []complex128{1, 1, 1, 1}
	fft(x)
	assert.InDelta(t, 4.0, real(x[0]), 1e-10)
	for i := 1; i < 4; i++ {
		assert.InDelta(t, 0.0, real(x[i]), 1e-10)
		assert.InDelta(t, 0.0, imag(x[i]), 1e-10)
	}
}

func TestFFT_Parseval(t *testing.T) {
	n := 512
	x := make([]complex128, n)
	var energyTime float64
	for i := range x {
		v := math.Sin(2.0 * math.Pi * float64(i) / float64(n))
		x[i] = complex(v, 0)
		energyTime += v * v
	}

	fft(x)

	var energyFreq float64
	for _, v := range x {
		r := real(v)
		im := imag(v)
		energyFreq += r*r + im*im
	}
	energyFreq /= float64(n)

	assert.InDelta(t, energyTime, energyFreq, 1e-6)
}

func TestFFT_Linearity(t *testing.T) {
	// FFT(a*x + b*y) = a*FFT(x) + b*FFT(y)
	n := 256
	x := make([]complex128, n)
	y := make([]complex128, n)
	z := make([]complex128, n)
	for i := range x {
		x[i] = complex(math.Sin(2*math.Pi*float64(i)/float64(n)), 0)
		y[i] = complex(math.Cos(2*math.Pi*3*float64(i)/float64(n)), 0)
		z[i] = 2*x[i] + 3*y[i]
	}

	fft(x)
	fft(y)
	fft(z)

	for i := range z {
		expected := 2*x[i] + 3*y[i]
		assert.InDelta(t, real(expected), real(z[i]), 1e-6)
		assert.InDelta(t, imag(expected), imag(z[i]), 1e-6)
	}
}

func TestFFT_SingleTone(t *testing.T) {
	// A single-frequency sine should have energy at that bin
	n := 512
	k := 10 // frequency bin 10
	x := make([]complex128, n)
	for i := range x {
		x[i] = complex(math.Sin(2.0*math.Pi*float64(k)*float64(i)/float64(n)), 0)
	}

	fft(x)

	// Bin k should have the highest magnitude
	maxMag := 0.0
	maxBin := 0
	for i := range x {
		mag := math.Sqrt(real(x[i])*real(x[i]) + imag(x[i])*imag(x[i]))
		if mag > maxMag {
			maxMag = mag
			maxBin = i
		}
	}
	// Energy should be at bin k or n-k (negative frequency)
	assert.True(t, maxBin == k || maxBin == n-k)
}

// ============================================================================
// WHISPER FEATURE EXTRACTION TESTS
// ============================================================================

func TestWhisperFeatures_Init(t *testing.T) {
	wf := newWhisperFeatures()

	// Hann window: 0 at start, peaks at center
	assert.InDelta(t, 0.0, wf.hannWindow[0], 1e-10)
	assert.InDelta(t, 1.0, wf.hannWindow[whisperNFFT/2], 1e-3)
	// Symmetric: hannWindow[i] ≈ hannWindow[nFFT - i]
	for i := 1; i < whisperNFFT/2; i++ {
		assert.InDelta(t, wf.hannWindow[i], wf.hannWindow[whisperNFFT-i], 1e-10)
	}

	// Mel filters: each filter should have a peak of 1 or less
	for i := 0; i < whisperNMels; i++ {
		hasNonZero := false
		for j := 0; j < whisperNFreqBins; j++ {
			if wf.melFilters[i][j] > 0 {
				hasNonZero = true
			}
		}
		assert.True(t, hasNonZero, "mel filter %d has no non-zero entries", i)
	}
}

func TestWhisperFeatures_MelFilterbankCoverage(t *testing.T) {
	wf := newWhisperFeatures()

	// Every freq bin (except maybe the very edges) should be covered by at least one mel filter
	covered := make([]bool, whisperNFreqBins)
	for i := 0; i < whisperNMels; i++ {
		for j := 0; j < whisperNFreqBins; j++ {
			if wf.melFilters[i][j] > 0 {
				covered[j] = true
			}
		}
	}
	// Count uncovered bins (may be at the very edges)
	uncovered := 0
	for _, c := range covered {
		if !c {
			uncovered++
		}
	}
	// At most a few edge bins should be uncovered
	assert.Less(t, uncovered, 5, "too many uncovered frequency bins")
}

func TestWhisperFeatures_OutputShape(t *testing.T) {
	wf := newWhisperFeatures()

	testCases := []struct {
		name     string
		nSamples int
	}{
		{"1_second", 16000},
		{"500ms", 8000},
		{"8_seconds", 128000},
		{"10_seconds", 160000},
		{"100ms", 1600},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			audio := make([]float32, tc.nSamples)
			features := wf.Extract(audio)
			require.Len(t, features, whisperNMels*whisperMaxFrames)
		})
	}
}

func TestWhisperFeatures_DifferentAudio(t *testing.T) {
	wf := newWhisperFeatures()

	silence := make([]float32, 16000)
	silenceFeats := wf.Extract(silence)

	sine := make([]float32, 16000)
	for i := range sine {
		sine[i] = float32(math.Sin(2.0 * math.Pi * 440.0 * float64(i) / 16000.0))
	}
	sineFeats := wf.Extract(sine)

	different := false
	for i := range silenceFeats {
		if silenceFeats[i] != sineFeats[i] {
			different = true
			break
		}
	}
	assert.True(t, different)
}

func TestWhisperFeatures_Deterministic(t *testing.T) {
	wf := newWhisperFeatures()

	audio := make([]float32, 16000)
	for i := range audio {
		audio[i] = float32(math.Sin(2.0 * math.Pi * 440.0 * float64(i) / 16000.0))
	}

	f1 := wf.Extract(audio)
	f2 := wf.Extract(audio)

	for i := range f1 {
		assert.Equal(t, f1[i], f2[i], "features not deterministic at index %d", i)
	}
}

func TestWhisperFeatures_OutputRange(t *testing.T) {
	wf := newWhisperFeatures()

	audio := make([]float32, 32000) // 2 seconds
	for i := range audio {
		audio[i] = float32(math.Sin(2.0 * math.Pi * 1000.0 * float64(i) / 16000.0))
	}
	features := wf.Extract(audio)

	// After normalization: (log_mel + 4.0) / 4.0, typical range [-1, 1+]
	for _, v := range features {
		assert.False(t, math.IsNaN(float64(v)), "NaN in features")
		assert.False(t, math.IsInf(float64(v), 0), "Inf in features")
	}
}

func TestWhisperFeatures_FrequencySelectivity(t *testing.T) {
	wf := newWhisperFeatures()

	// Low frequency tone (200 Hz) should excite lower mel bins
	low := make([]float32, 128000)
	for i := range low {
		low[i] = float32(math.Sin(2.0 * math.Pi * 200.0 * float64(i) / 16000.0))
	}
	lowFeats := wf.Extract(low)

	// High frequency tone (4000 Hz) should excite higher mel bins
	high := make([]float32, 128000)
	for i := range high {
		high[i] = float32(math.Sin(2.0 * math.Pi * 4000.0 * float64(i) / 16000.0))
	}
	highFeats := wf.Extract(high)

	// Sum energy in low mel bins (0-19) vs high mel bins (60-79)
	var lowLowEnergy, lowHighEnergy float64
	var highLowEnergy, highHighEnergy float64
	for m := 0; m < 20; m++ {
		for f := 0; f < whisperMaxFrames; f++ {
			lowLowEnergy += float64(lowFeats[m*whisperMaxFrames+f])
			highLowEnergy += float64(highFeats[m*whisperMaxFrames+f])
		}
	}
	for m := 60; m < 80; m++ {
		for f := 0; f < whisperMaxFrames; f++ {
			lowHighEnergy += float64(lowFeats[m*whisperMaxFrames+f])
			highHighEnergy += float64(highFeats[m*whisperMaxFrames+f])
		}
	}

	// Low tone should have relatively more energy in low mel bins
	assert.Greater(t, lowLowEnergy, lowHighEnergy, "200Hz tone should excite lower mel bins more")
	// High tone should have relatively more energy in high mel bins
	assert.Greater(t, highHighEnergy, highLowEnergy, "4000Hz tone should excite higher mel bins more")
}

// ============================================================================
// AUDIO BUFFER TESTS
// ============================================================================

func TestAppendAudio_PCM16Conversion(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	// 100 samples of int16 value 1000
	pcm := make([]byte, 200)
	for i := 0; i < 100; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(1000))
	}
	eos.appendAudio(pcm)

	assert.Len(t, eos.audioBuffer, 100)
	assert.InDelta(t, 1000.0/32768.0, eos.audioBuffer[0], 1e-5)
}

func TestAppendAudio_NegativeValues(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	pcm := make([]byte, 2)
	v := int16(-16384)
	binary.LittleEndian.PutUint16(pcm, uint16(v))
	eos.appendAudio(pcm)

	assert.Len(t, eos.audioBuffer, 1)
	assert.InDelta(t, -16384.0/32768.0, eos.audioBuffer[0], 1e-5)
}

func TestAppendAudio_RollingBuffer(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	// Fill buffer to near capacity
	eos.audioBuffer = make([]float32, maxAudioSamples-10)

	// Append 100 more samples — should evict oldest
	pcm := make([]byte, 200)
	eos.appendAudio(pcm)

	assert.Len(t, eos.audioBuffer, maxAudioSamples)
}

func TestAppendAudio_EmptyInput(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.appendAudio(nil)
	assert.Len(t, eos.audioBuffer, 0)

	eos.appendAudio([]byte{0}) // single byte, can't form a sample
	assert.Len(t, eos.audioBuffer, 0)
}

func TestAppendAudio_ConcurrentSafety(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pcm := make([]byte, 200)
			for i := 0; i < 100; i++ {
				eos.appendAudio(pcm)
			}
		}()
	}
	wg.Wait()

	assert.LessOrEqual(t, len(eos.audioBuffer), maxAudioSamples)
}

func TestPredictEOU_EmptyAudioBufferReturnsMinusOne(t *testing.T) {
	eos := &pipecatEndOfSpeech{}
	assert.Equal(t, -1.0, eos.predictEOU())
}

func TestPredictEOU_DetectorUnavailableReturnsMinusOne(t *testing.T) {
	eos := &pipecatEndOfSpeech{
		audioBuffer: []float32{0.1, -0.1, 0.05},
	}
	assert.Equal(t, -1.0, eos.predictEOU())
}

func TestPredictEOU_ReusesCachedProbabilityForSameAudioGeneration(t *testing.T) {
	var predictorCalls int32
	eos := newTestEOSWithPredictor(
		func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}),
		func([]float32) (float64, error) {
			atomic.AddInt32(&predictorCalls, 1)
			return 0.75, nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))

	assert.Equal(t, 0.75, eos.predictEOU())
	assert.Equal(t, 0.75, eos.predictEOU())
	assert.Equal(t, int32(1), atomic.LoadInt32(&predictorCalls))
}

func TestPredictEOU_InvalidatesCacheWhenAudioChanges(t *testing.T) {
	var predictorCalls int32
	eos := newTestEOSWithPredictor(
		func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}),
		func([]float32) (float64, error) {
			return float64(atomic.AddInt32(&predictorCalls, 1)), nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	assert.Equal(t, 1.0, eos.predictEOU())
	assert.Equal(t, 1.0, eos.predictEOU())

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	assert.Equal(t, 2.0, eos.predictEOU())
	assert.Equal(t, int32(2), atomic.LoadInt32(&predictorCalls))
}

// ============================================================================
// EOS INTEGRATION TESTS (without ONNX model — fallback timeout path)
// ============================================================================

func TestEOS_UserTextImmediateFire(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	err := eos.Execute(context.Background(), userInput("hello"))
	require.NoError(t, err)

	select {
	case p := <-called:
		assert.Equal(t, "hello", p.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

func TestEOS_STTAccumulatesText(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0, "microphone.eos.extended_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	// First final STT
	eos.Execute(ctx, sttInput("hello", true))
	// Second final STT — text should accumulate
	eos.Execute(ctx, sttInput("world", true))

	select {
	case p := <-called:
		assert.Contains(t, p.Speech, "hello")
		assert.Contains(t, p.Speech, "world")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

func TestEOS_STTWithoutAudioFallsBackToFallbackTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 80.0,
		"microphone.eos.extended_timeout": 1000.0,
	}))
	defer closeTestEndOfSpeech(eos)

	start := time.Now()
	// No audio sent before final STT -> predictEOU returns -1 via empty audio buffer.
	err := eos.Execute(context.Background(), sttInput("hello", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.Less(t, elapsed, 350*time.Millisecond)
	case <-time.After(700 * time.Millisecond):
		t.Fatal("expected fallback timeout path when no audio is buffered")
	}
}

func TestEOS_STTWithAudioAndNoDetectorFallsBack(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 90.0,
		"microphone.eos.extended_timeout": 1000.0,
	}))
	defer closeTestEndOfSpeech(eos)

	// Ensure audio is buffered, but detector is intentionally nil in newTestEOS.
	err := eos.Execute(context.Background(), audioInput(1600))
	require.NoError(t, err)

	start := time.Now()
	err = eos.Execute(context.Background(), sttInput("hello", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.Less(t, elapsed, 350*time.Millisecond)
	case <-time.After(700 * time.Millisecond):
		t.Fatal("expected fallback timeout path when detector is unavailable")
	}
}

func TestEOS_FinalSTTModelPredictsDone_UsesQuickTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOSWithPredictor(
		callback,
		newTestOpts(map[string]any{
			"microphone.eos.threshold":        0.5,
			"microphone.eos.quick_timeout":    80.0,
			"microphone.eos.extended_timeout": 1000.0,
			"microphone.eos.fallback_timeout": 500.0,
		}),
		func(_ []float32) (float64, error) {
			return 0.92, nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	err := eos.Execute(context.Background(), audioInput(1600))
	require.NoError(t, err)

	start := time.Now()
	err = eos.Execute(context.Background(), sttInput("ideal quick", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.GreaterOrEqual(t, elapsed, 45*time.Millisecond)
		assert.Less(t, elapsed, 300*time.Millisecond)
	case <-time.After(700 * time.Millisecond):
		t.Fatal("expected quick timeout EOS callback")
	}
}

func TestEOS_FinalSTTModelPredictsNotDone_UsesExtendedTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOSWithPredictor(
		callback,
		newTestOpts(map[string]any{
			"microphone.eos.threshold":        0.5,
			"microphone.eos.quick_timeout":    60.0,
			"microphone.eos.extended_timeout": 260.0,
			"microphone.eos.fallback_timeout": 80.0,
		}),
		func(_ []float32) (float64, error) {
			return 0.10, nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	err := eos.Execute(context.Background(), audioInput(1600))
	require.NoError(t, err)

	start := time.Now()
	err = eos.Execute(context.Background(), sttInput("ideal extended", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.GreaterOrEqual(t, elapsed, 180*time.Millisecond)
		assert.Less(t, elapsed, 700*time.Millisecond)
	case <-time.After(1 * time.Second):
		t.Fatal("expected extended timeout EOS callback")
	}
}

func TestEOS_AudioPacketAccumulates(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	err := eos.Execute(ctx, audioInput(1600))
	require.NoError(t, err)

	assert.Len(t, eos.audioBuffer, 1600)

	// Execute more audio
	err = eos.Execute(ctx, audioInput(3200))
	require.NoError(t, err)

	assert.Len(t, eos.audioBuffer, 4800)
}

func TestEOS_EmptyUserTextIgnored(t *testing.T) {
	callCount := int64(0)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				atomic.AddInt64(&callCount, 1)
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.Execute(context.Background(), userInput(""))
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(0), atomic.LoadInt64(&callCount))
}

func TestEOS_InterruptionWithNoTextIgnored(t *testing.T) {
	callCount := int64(0)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				atomic.AddInt64(&callCount, 1)
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.Execute(context.Background(), interruptInput())
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int64(0), atomic.LoadInt64(&callCount))
}

func TestEOS_VADStartDefersFinalUntilRecoveryTimeout(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 60.0,
		"microphone.eos.extended_timeout": 120.0,
	}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("hello", true)))

	select {
	case <-called:
		t.Fatal("callback fired before stuck VAD recovery elapsed")
	case <-time.After(90 * time.Millisecond):
	}

	select {
	case p := <-called:
		assert.Equal(t, "hello", p.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback after stuck VAD recovery")
	}

	select {
	case p := <-called:
		t.Fatalf("unexpected duplicate callback after stuck VAD recovery: %+v", p)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestEOS_VADEndFlushesPendingFinal(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 100.0,
		"microphone.eos.quick_timeout":    40.0,
	}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("hello", true)))

	select {
	case <-called:
		t.Fatal("callback fired before VAD end")
	case <-time.After(60 * time.Millisecond):
	}

	deadline := time.After(300 * time.Millisecond)
	for {
		eos.mu.RLock()
		pending := eos.state.pending != nil
		eos.mu.RUnlock()
		if pending {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for pending final")
		case <-time.After(10 * time.Millisecond):
		}
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case p := <-called:
		assert.Equal(t, "hello", p.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD end")
	}

	select {
	case p := <-called:
		t.Fatalf("unexpected duplicate callback after VAD end: %+v", p)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestEOS_VADEndFlushWindowUsesLateFinalSTT(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 200.0,
		"microphone.eos.quick_timeout":    40.0,
	}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-late-final",
		Script:    "Rightly?",
		Interim:   false,
	}))

	select {
	case p := <-called:
		assert.Equal(t, "me an idea about the how how do I do things? Rightly?", p.Speech)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for late final callback")
	}
}

func TestEOS_VADEndCompleteClearsPendingInterimWhenFinalIsShorter(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 70.0,
		"microphone.eos.quick_timeout":    20.0,
	}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	require.NoError(t, eos.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things? Rightly?",
		Interim:   true,
	}))
	require.NoError(t, eos.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-short-final",
		Script:    "me an idea about the how how do I do things?",
		Interim:   false,
	}))
	require.NoError(t, eos.Execute(ctx, internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case p := <-called:
		assert.Equal(t, "me an idea about the how how do I do things?", p.Speech)
		require.Len(t, p.Speechs, 2)
		assert.True(t, p.Speechs[0].Interim)
		assert.False(t, p.Speechs[1].Interim)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for final transcript")
	}
}

func TestEOS_StaleTimerCompletionDoesNotShrinkLatestSegment(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 2)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				called <- p
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.mu.Lock()
	eos.state.segment = speechSegment{
		Revision:  1,
		ContextID: "ctx-stale",
		Text:      "me an idea about the how how do I do things?",
		Timestamp: time.Now(),
	}
	oldSegment := eos.state.segment
	eos.mu.Unlock()

	eos.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: oldSegment,
		timeout: 30 * time.Millisecond,
	})

	time.Sleep(10 * time.Millisecond)
	latestSegment := speechSegment{
		Revision:  2,
		ContextID: "ctx-stale",
		Text:      "me an idea about the how how do I do things? Rightly?",
		Timestamp: time.Now(),
	}
	eos.mu.Lock()
	eos.state.segment = latestSegment
	eos.mu.Unlock()

	select {
	case p := <-called:
		t.Fatalf("stale completion fired: %q", p.Speech)
	case <-time.After(80 * time.Millisecond):
	}

	eos.enqueueCommand(workerCommand{
		ctx:     context.Background(),
		segment: latestSegment,
		timeout: 10 * time.Millisecond,
	})

	select {
	case p := <-called:
		assert.Equal(t, latestSegment.Text, p.Speech)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for latest completion")
	}
}

func TestEOS_OnlyInterimWithVADDoesNotComplete(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 60.0,
	}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("interim only", false)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case p := <-called:
		t.Fatalf("callback should not fire for interim-only VAD input: %+v", p)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEOS_ResetClearsAudioBuffer(t *testing.T) {
	called := make(chan struct{}, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- struct{}{}:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	// Accumulate audio
	eos.Execute(ctx, audioInput(16000))
	assert.Greater(t, len(eos.audioBuffer), 0)

	// Fire EOS (which triggers reset)
	eos.Execute(ctx, userInput("test"))

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout")
	}

	// After reset, audio buffer should be cleared
	time.Sleep(50 * time.Millisecond)
	eos.mu.RLock()
	bufLen := len(eos.audioBuffer)
	eos.mu.RUnlock()
	assert.Equal(t, 0, bufLen)
}

func TestEOS_Name(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)
	assert.Equal(t, "pipecatSmartTurnEndOfSpeech", eos.Name())
}

func TestEOS_ObservabilityEvent_Initialized(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 1)
	logs := make(chan internal_type.ObservabilityLogRecordPacket, 1)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
			if log, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok {
				select {
				case logs <- log:
				default:
				}
			}
		}
		return nil
	}

	eos, err := newPipecatEndOfSpeechForTest(context.Background(), logger, callback, utils.Option{})
	require.NoError(t, err)
	defer func() { _ = eos.Close(context.Background()) }()

	timeout := time.After(500 * time.Millisecond)
	var sawInitMetric, sawInitLog bool
	for !sawInitMetric || !sawInitLog {
		select {
		case metric := <-metrics:
			assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, metric.Scope)
			require.NotEmpty(t, metric.Record.Metrics)
			assert.Equal(t, observability.MetricEOSInitLatencyMs, metric.Record.Metrics[0].Name)
			assert.Equal(t, pipecatEndOfSpeechName, metric.Record.Attributes["provider"])
			_, parseErr := strconv.Atoi(metric.Record.Metrics[0].Value)
			assert.NoError(t, parseErr)
			sawInitMetric = true
		case log := <-logs:
			assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, log.Scope)
			assert.Equal(t, observability.LevelInfo, log.Record.Level)
			assert.Equal(t, pipecatEndOfSpeechName, log.Record.Attributes["provider"])
			assert.NotEmpty(t, log.Record.Attributes["options"])
			sawInitLog = true
		case <-timeout:
			t.Fatal("timeout waiting for initialized observability records")
		}
	}
}

func TestEOS_ObservabilityEvent_UserTextDetected(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok && event.Record.Event == observability.EOSCompleted {
				select {
				case events <- event:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-interim",
		Text:      "hello",
	}))

	select {
	case event := <-events:
		assert.Equal(t, "ctx-interim", event.ContextID)
		assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, event.Scope)
		assert.Equal(t, observability.ComponentEOS, event.Record.Component)
		assert.Equal(t, observability.EOSCompleted, event.Record.Event)
		assert.Equal(t, pipecatEndOfSpeechName, event.Record.Attributes["provider"])
		assert.Equal(t, "ctx-interim", event.Record.Attributes["context_id"])
		assert.Equal(t, "hello", event.Record.Attributes["speech"])
		assert.False(t, event.Record.OccurredAt.IsZero())
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for detected observability event")
	}
}

func TestEOS_ObservabilityEvent_Detected(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 4)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 2)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok && event.Record.Event == observability.EOSCompleted {
				select {
				case events <- event:
				default:
				}
			}
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-detected",
		Text:      "hello world",
	}))

	timeout := time.After(500 * time.Millisecond)
	var sawDetected, sawMetric bool
	for !sawDetected || !sawMetric {
		select {
		case event := <-events:
			assert.Equal(t, "ctx-detected", event.ContextID)
			assert.Equal(t, pipecatEndOfSpeechName, event.Record.Attributes["provider"])
			assert.Equal(t, "ctx-detected", event.Record.Attributes["context_id"])
			assert.Equal(t, "hello world", event.Record.Attributes["speech"])
			assert.Equal(t, "0.0000", event.Record.Attributes["confidence"])
			assert.Equal(t, "2", event.Record.Attributes["word_count"])
			assert.Equal(t, "11", event.Record.Attributes["char_count"])
			assert.NotEmpty(t, event.Record.Attributes["text_to_trigger_ms"])
			assert.NotEmpty(t, event.Record.Attributes["wait_to_trigger_ms"])
			assert.False(t, event.Record.OccurredAt.IsZero())
			_, parseErr := strconv.Atoi(event.Record.Attributes["text_to_trigger_ms"])
			assert.NoError(t, parseErr)
			_, parseErr = strconv.Atoi(event.Record.Attributes["wait_to_trigger_ms"])
			assert.NoError(t, parseErr)
			sawDetected = true
		case metric := <-metrics:
			require.NotEmpty(t, metric.Record.Metrics)
			if metric.Record.Metrics[0].Name != observability.MetricEOSLatencyMs {
				continue
			}
			assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, metric.Scope)
			assert.Equal(t, pipecatEndOfSpeechName, metric.Record.Attributes["provider"])
			_, parseErr := strconv.Atoi(metric.Record.Metrics[0].Value)
			assert.NoError(t, parseErr)
			sawMetric = true
		case <-timeout:
			t.Fatal("timeout waiting for detected observability event")
		}
	}
}

func TestEOS_ObservabilityEvent_Lifecycle(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	events := make(chan internal_type.ObservabilityEventRecordPacket, 8)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 8)
	logs := make(chan internal_type.ObservabilityLogRecordPacket, 8)
	usages := make(chan internal_type.ObservabilityUsageRecordPacket, 2)
	callback := func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok {
				select {
				case events <- event:
				default:
				}
			}
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
			if log, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok {
				select {
				case logs <- log:
				default:
				}
			}
			if usage, ok := packet.(internal_type.ObservabilityUsageRecordPacket); ok {
				select {
				case usages <- usage:
				default:
				}
			}
		}
		return nil
	}

	eos, err := newPipecatEndOfSpeechForTest(context.Background(), logger, callback, utils.Option{})
	require.NoError(t, err)

	require.NoError(t, eos.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-debug",
		Text:      "hello",
	}))

	sawInitMetric := false
	sawInitLog := false
	sawStarted := false
	sawDetected := false
	timeout := time.After(500 * time.Millisecond)
	for !sawInitMetric || !sawInitLog || !sawStarted || !sawDetected {
		select {
		case event := <-events:
			if event.Record.Event == observability.EOSStarted {
				sawStarted = true
				continue
			}
			if event.Record.Event == observability.EOSCompleted {
				sawDetected = true
				continue
			}
			t.Fatalf("unexpected eos event: %+v", event)
		case metric := <-metrics:
			if len(metric.Record.Metrics) > 0 && metric.Record.Metrics[0].Name == observability.MetricEOSInitLatencyMs {
				sawInitMetric = true
			}
		case log := <-logs:
			if log.Record.Level == observability.LevelInfo &&
				log.Record.Attributes["provider"] == pipecatEndOfSpeechName &&
				log.Record.Attributes["options"] != "" {
				sawInitLog = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for eos lifecycle events")
		}
	}

	require.NoError(t, eos.Close(context.Background()))

	timeout = time.After(500 * time.Millisecond)
	sawClosed := false
	sawUsage := false
	for !sawClosed || !sawUsage {
		select {
		case event := <-events:
			if event.Record.Event == observability.EOSClosed {
				sawClosed = true
			}
		case usage := <-usages:
			if usage.Record.Component == observability.ComponentName(observability.UsageConversationEOSDuration) {
				assert.Equal(t, pipecatEndOfSpeechName, usage.Record.Attributes["provider"])
				sawUsage = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for closed eos event")
		}
	}
}

func TestEOS_ObservabilityStartedForSpeechToText(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			event, ok := packet.(internal_type.ObservabilityEventRecordPacket)
			if !ok || event.Record.Event != observability.EOSStarted {
				continue
			}
			select {
			case events <- event:
			default:
			}
		}
		return nil
	}, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	require.NoError(t, eos.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello",
		Interim:   true,
	}))

	select {
	case event := <-events:
		assert.Equal(t, "ctx-started", event.ContextID)
		assert.Equal(t, internal_type.ObservabilityRecordScopeUserMessage, event.Scope)
		assert.Equal(t, observability.ComponentEOS, event.Record.Component)
		assert.Equal(t, pipecatEndOfSpeechName, event.Record.Attributes["provider"])
		assert.Equal(t, "ctx-started", event.Record.Attributes["context_id"])
		assert.Equal(t, "hello", event.Record.Attributes["speech"])
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for eos started event")
	}

	require.NoError(t, eos.Execute(ctx, internal_type.SpeechToTextPacket{
		ContextID: "ctx-started",
		Script:    "hello again",
		Interim:   true,
	}))

	select {
	case event := <-events:
		t.Fatalf("unexpected duplicate started event: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEOS_KeepsMetrics(t *testing.T) {
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 2)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if metric, ok := packet.(internal_type.ObservabilityMetricRecordPacket); ok {
				select {
				case metrics <- metric:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), internal_type.UserTextReceivedPacket{
		ContextID: "ctx-off",
		Text:      "hello",
	}))

	select {
	case metric := <-metrics:
		require.NotEmpty(t, metric.Record.Metrics)
		assert.Equal(t, observability.MetricEOSLatencyMs, metric.Record.Metrics[0].Name)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for eos metric")
	}
}

func TestEOS_MetricUsesLastTimerArm(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 2)
	metrics := make(chan internal_type.ObservabilityMetricRecordPacket, 1)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			switch typed := packet.(type) {
			case internal_type.ObservabilityEventRecordPacket:
				if typed.Record.Event != observability.EOSCompleted {
					continue
				}
				select {
				case events <- typed:
				default:
				}
			case internal_type.ObservabilityMetricRecordPacket:
				select {
				case metrics <- typed:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 120.0,
		"microphone.eos.extended_timeout": 900.0,
	}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	require.NoError(t, eos.Execute(ctx, sttInput("hello", true)))
	time.Sleep(80 * time.Millisecond)
	require.NoError(t, eos.Execute(ctx, sttInput("...", false)))

	timeout := time.After(800 * time.Millisecond)
	var detected internal_type.ObservabilityEventRecordPacket
	var metric internal_type.ObservabilityMetricRecordPacket
	for detected.Record.Event == "" || len(metric.Record.Metrics) == 0 {
		select {
		case detected = <-events:
		case metric = <-metrics:
		case <-timeout:
			t.Fatal("timeout waiting for detected eos packets")
		}
	}

	textMs, err := strconv.Atoi(detected.Record.Attributes["text_to_trigger_ms"])
	require.NoError(t, err)
	waitMs, err := strconv.Atoi(detected.Record.Attributes["wait_to_trigger_ms"])
	require.NoError(t, err)
	require.NotEmpty(t, metric.Record.Metrics)
	assert.Equal(t, observability.MetricEOSLatencyMs, metric.Record.Metrics[0].Name)
	metricMs, err := strconv.Atoi(metric.Record.Metrics[0].Value)
	require.NoError(t, err)

	assert.InDelta(t, waitMs, metricMs, 30)
	assert.Greater(t, textMs, waitMs+40)
}

func TestEOS_RespectsExplicitEmptyConcat(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	eos := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if endOfSpeech, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- endOfSpeech:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 80.0,
	}))
	defer closeTestEndOfSpeech(eos)

	empty := ""
	packets := []internal_type.SpeechToTextPacket{
		{ContextID: "ctx-concat", Script: "I", Interim: false},
		{ContextID: "ctx-concat", Script: "'m", Concat: &empty, Interim: false},
		{ContextID: "ctx-concat", Script: "thinking", Interim: false},
		{ContextID: "ctx-concat", Script: ".", Concat: &empty, Interim: false},
	}
	for _, packet := range packets {
		require.NoError(t, eos.Execute(context.Background(), packet))
	}

	select {
	case result := <-called:
		assert.Equal(t, "I'm thinking.", result.Speech)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for end of speech")
	}
}

func TestEOS_ObservabilityEvent_DetectedConfidence(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 4)
	eos := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok && event.Record.Event == observability.EOSCompleted {
				select {
				case events <- event:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.threshold":     0.5,
		"microphone.eos.quick_timeout": 10.0,
	}), func([]float32) (float64, error) {
		return 0.7345, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-confidence",
		Script:    "hello there",
		Interim:   false,
	}))

	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-events:
			assert.Equal(t, "ctx-confidence", event.ContextID)
			assert.Equal(t, "0.7345", event.Record.Attributes["confidence"])
			return
		case <-timeout:
			t.Fatal("timeout waiting for detected confidence observability event")
		}
	}
}

func TestEOS_ObservabilityEvent_DetectedConfidenceAfterExtend(t *testing.T) {
	events := make(chan internal_type.ObservabilityEventRecordPacket, 4)
	eos := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok && event.Record.Event == observability.EOSCompleted {
				select {
				case events <- event:
				default:
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.threshold":        0.5,
		"microphone.eos.extended_timeout": 100.0,
	}), func([]float32) (float64, error) {
		return 0.3125, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-extend-confidence",
		Script:    "continue",
		Interim:   false,
	}))
	require.NoError(t, eos.Execute(context.Background(), interruptInput()))

	timeout := time.After(750 * time.Millisecond)
	for {
		select {
		case event := <-events:
			assert.Equal(t, "ctx-extend-confidence", event.ContextID)
			assert.Equal(t, "0.3125", event.Record.Attributes["confidence"])
			return
		case <-timeout:
			t.Fatal("timeout waiting for extended detected confidence observability event")
		}
	}
}

func TestEOS_CloseStopsWorker(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	err := eos.Close(context.Background())
	assert.NoError(t, err)
	err = eos.Close(context.Background())
	assert.NoError(t, err)
}

func TestEOS_SendAfterClose_DoesNotEnqueueCommand(t *testing.T) {
	eos := &pipecatEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	close(eos.stopCh)

	eos.enqueueCommand(workerCommand{fireImmediately: true})

	assert.Equal(t, 0, len(eos.commandCh))
}

func TestEOS_EnqueueCommandBlocksUntilChannelHasSpace(t *testing.T) {
	eos := &pipecatEndOfSpeech{
		commandCh: make(chan workerCommand, 1),
		stopCh:    make(chan struct{}),
		state:     &endOfSpeechState{segment: speechSegment{}},
	}
	eos.commandCh <- workerCommand{segment: speechSegment{Text: "first"}}

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		eos.enqueueCommand(workerCommand{segment: speechSegment{Text: "second"}})
		close(done)
	}()

	<-started
	select {
	case <-done:
		t.Fatal("enqueueCommand should wait while channel is full")
	case <-time.After(50 * time.Millisecond):
	}

	first := <-eos.commandCh
	assert.Equal(t, "first", first.segment.Text)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("enqueueCommand should resume after channel space is available")
	}

	second := <-eos.commandCh
	assert.Equal(t, "second", second.segment.Text)
}

func TestEOS_ConcurrentExecute(t *testing.T) {
	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0, "microphone.eos.extended_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	var wg sync.WaitGroup
	ctx := context.Background()
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				switch i % 4 {
				case 0:
					eos.Execute(ctx, audioInput(160))
				case 1:
					eos.Execute(ctx, sttInput("text", true))
				case 2:
					eos.Execute(ctx, interruptInput())
				case 3:
					eos.Execute(ctx, userInput("msg"))
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestEOS_ContextCancelStillFires(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	ctxErr := make(chan error, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case ctxErr <- ctx.Err():
				default:
				}
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 100.0, "microphone.eos.extended_timeout": 100.0}))
	defer closeTestEndOfSpeech(eos)

	ctx, cancel := context.WithCancel(context.Background())
	eos.Execute(ctx, sttInput("hello", true))
	cancel() // cancel before timer fires

	select {
	case p := <-called:
		assert.Equal(t, "hello", p.Speech)
		assert.NoError(t, <-ctxErr)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: callback should fire even after context cancel")
	}
}

func TestEOS_PredictorSerializedUnderConcurrentExecute(t *testing.T) {
	var inFlight int32
	var maxInFlight int32

	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	predictor := func([]float32) (float64, error) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			maximum := atomic.LoadInt32(&maxInFlight)
			if current <= maximum || atomic.CompareAndSwapInt32(&maxInFlight, maximum, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		return 0.0, nil
	}

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.extended_timeout": 500.0,
	}), predictor)
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = eos.Execute(context.Background(), sttInput("serialized", true))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&maxInFlight))
}

func TestEOS_CallbackFiresOnlyOnce(t *testing.T) {
	callCount := int64(0)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				atomic.AddInt64(&callCount, 1)
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 50.0, "microphone.eos.extended_timeout": 50.0}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	eos.Execute(ctx, sttInput("hello", true))
	// Send more inputs that should be ignored after callback fires
	time.Sleep(20 * time.Millisecond)
	eos.Execute(ctx, sttInput("world", true))
	eos.Execute(ctx, interruptInput())

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, int64(1), atomic.LoadInt64(&callCount))
}

func TestEOS_InterimSTTExtendsTimer(t *testing.T) {
	called := make(chan internal_type.EndOfSpeechPacket, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if p, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- p:
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 150.0, "microphone.eos.extended_timeout": 150.0}))
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	// Send final STT to start text accumulation
	eos.Execute(ctx, sttInput("hello", true))

	// Send interim STTs to extend timer
	for i := 0; i < 3; i++ {
		time.Sleep(80 * time.Millisecond)
		eos.Execute(ctx, sttInput("...", false))
	}

	// Should eventually fire
	select {
	case p := <-called:
		assert.Contains(t, p.Speech, "hello")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestEOS_FinalSTTInferenceFailure_UsesConfiguredFallbackTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	fallbackMs := 60.0
	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": fallbackMs,
		"microphone.eos.extended_timeout": 900.0,
		"microphone.eos.quick_timeout":    20.0,
	}))
	defer closeTestEndOfSpeech(eos)

	start := time.Now()
	err := eos.Execute(context.Background(), sttInput("fallback path", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.GreaterOrEqual(t, elapsed, 35*time.Millisecond, "should not fire immediately")
		assert.Less(t, elapsed, 300*time.Millisecond, "should use microphone.eos.fallback_timeout, not extended timeout")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for EOS callback")
	}
}

func TestEOS_FinalSTTInferenceFailure_UsesDefaultFallbackTimeoutWhenUnset(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		// No microphone.eos.fallback_timeout -> should use default fallback (500ms)
		"microphone.eos.extended_timeout": 1500.0,
	}))
	defer closeTestEndOfSpeech(eos)

	start := time.Now()
	err := eos.Execute(context.Background(), sttInput("default fallback", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.GreaterOrEqual(t, elapsed, 420*time.Millisecond)
		assert.Less(t, elapsed, 1200*time.Millisecond)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for EOS callback")
	}
}

func TestEOS_FinalSTTInferenceFailure_DoesNotUseSilenceTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	callback := func(ctx context.Context, res ...internal_type.Packet) error {
		for _, r := range res {
			if _, ok := r.(internal_type.EndOfSpeechPacket); ok {
				select {
				case called <- time.Now():
				default:
				}
			}
		}
		return nil
	}

	eos := newTestEOS(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 70.0,
		"microphone.eos.extended_timeout": 1400.0,
	}))
	defer closeTestEndOfSpeech(eos)

	start := time.Now()
	err := eos.Execute(context.Background(), sttInput("no silence timeout", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.Less(t, elapsed, 400*time.Millisecond)
	case <-time.After(700 * time.Millisecond):
		t.Fatal("EOS should have fired via fallback timeout before silence_timeout")
	}
}

// ============================================================================
// FACTORY TEST
// ============================================================================

func TestEOS_FactoryCreationFails_NoModel(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	// With a bogus model path, should fail
	_, err := newPipecatEndOfSpeechForTest(context.Background(), logger,
		func(context.Context, ...internal_type.Packet) error { return nil },
		utils.Option{"microphone.eos.pipecat.model_path": "/nonexistent/model.onnx"})
	assert.Error(t, err)
}
