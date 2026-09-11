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
	"os"
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
	predict        func([]float32) (float64, error)
	predictContext func(context.Context, []float32) (float64, error)
}

func (predictor testPredictor) Predict(audio []float32) (float64, error) {
	return predictor.predict(audio)
}

func (predictor testPredictor) PredictContext(ctx context.Context, audio []float32) (float64, error) {
	if predictor.predictContext != nil {
		return predictor.predictContext(ctx, audio)
	}
	return predictor.Predict(audio)
}

func newTestEOS(onPacket func(context.Context, ...internal_type.Packet) error, opts utils.Option) *pipecatEndOfSpeech {
	fallbackTimeout := time.Duration(defaultPctFallbackTimeout) * time.Millisecond
	if v, err := opts.GetFloat64("microphone.eos.fallback_timeout"); err == nil {
		fallbackTimeout = time.Duration(v) * time.Millisecond
	}
	extendedTimeout := time.Duration(defaultPctExtendedTimeout) * time.Millisecond
	if v, err := opts.GetFloat64("microphone.eos.extended_timeout"); err == nil {
		extendedTimeout = time.Duration(v) * time.Millisecond
	}
	threshold := defaultPctThreshold
	if v, err := opts.GetFloat64("microphone.eos.threshold"); err == nil {
		threshold = v
	}

	eos := &pipecatEndOfSpeech{
		onPacket:        onPacket,
		opts:            opts,
		threshold:       threshold,
		extendedTimeout: extendedTimeout,
		fallbackTimeout: fallbackTimeout,
		turnStopTimeout: defaultPctTurnStopTimeout,
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

func TestWhisperFFT_MatchesDirectTransform(t *testing.T) {
	for _, signal := range []string{"silence", "impulse", "constant", "tones"} {
		t.Run(signal, func(t *testing.T) {
			scratch := newWhisperFeatureScratch()
			for sampleIndex := range scratch.windowed {
				switch signal {
				case "impulse":
					if sampleIndex == 0 {
						scratch.windowed[sampleIndex] = 1
					}
				case "constant":
					scratch.windowed[sampleIndex] = 1
				case "tones":
					scratch.windowed[sampleIndex] = math.Sin(2*math.Pi*37*float64(sampleIndex)/float64(whisperNFFT)) +
						0.3*math.Cos(2*math.Pi*12*float64(sampleIndex)/float64(whisperNFFT))
				}
			}
			scratch.transform.Coefficients(scratch.coefficients[:], scratch.windowed[:])
			for frequencyBin, coefficient := range scratch.coefficients {
				var expectedReal, expectedImaginary float64
				for sampleIndex, sample := range scratch.windowed {
					angle := 2 * math.Pi * float64(frequencyBin) * float64(sampleIndex) / float64(whisperNFFT)
					expectedReal += sample * math.Cos(angle)
					expectedImaginary -= sample * math.Sin(angle)
				}
				assert.InDelta(t, expectedReal, real(coefficient), 1e-8, "bin %d real", frequencyBin)
				assert.InDelta(t, expectedImaginary, imag(coefficient), 1e-8, "bin %d imaginary", frequencyBin)
			}
		})
	}
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

func TestWhisperFeatures_PowerUsesWhisperWindowLength(t *testing.T) {
	wf := newWhisperFeatures()
	frame := make([]float32, whisperNFFT)
	for sampleIndex := range frame {
		frame[sampleIndex] = float32(
			math.Sin(2.0*math.Pi*37.0*float64(sampleIndex)/float64(whisperNFFT)) +
				0.3*math.Cos(2.0*math.Pi*12.0*float64(sampleIndex)/float64(whisperNFFT)),
		)
	}

	scratch := newWhisperFeatureScratch()
	for sampleIndex := range scratch.windowed {
		scratch.windowed[sampleIndex] = float64(frame[sampleIndex]) * wf.hannWindow[sampleIndex]
	}
	scratch.transform.Coefficients(scratch.coefficients[:], scratch.windowed[:])
	got := real(scratch.coefficients[37])*real(scratch.coefficients[37]) + imag(scratch.coefficients[37])*imag(scratch.coefficients[37])

	var expected400RealPart, expected400ImaginaryPart float64
	for sampleIndex := range frame {
		windowedSample := float64(frame[sampleIndex]) * wf.hannWindow[sampleIndex]
		angle := 2.0 * math.Pi * 37.0 * float64(sampleIndex) / float64(whisperNFFT)
		expected400RealPart += windowedSample * math.Cos(angle)
		expected400ImaginaryPart -= windowedSample * math.Sin(angle)
	}
	expected400 := expected400RealPart*expected400RealPart + expected400ImaginaryPart*expected400ImaginaryPart

	var zeroPadded512RealPart, zeroPadded512ImaginaryPart float64
	for sampleIndex := range frame {
		windowedSample := float64(frame[sampleIndex]) * wf.hannWindow[sampleIndex]
		angle := 2.0 * math.Pi * 37.0 * float64(sampleIndex) / 512.0
		zeroPadded512RealPart += windowedSample * math.Cos(angle)
		zeroPadded512ImaginaryPart -= windowedSample * math.Sin(angle)
	}
	zeroPadded512 := zeroPadded512RealPart*zeroPadded512RealPart + zeroPadded512ImaginaryPart*zeroPadded512ImaginaryPart

	assert.InDelta(t, expected400, got, 1e-8)
	assert.Greater(t, math.Abs(got-zeroPadded512), got*0.10)
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

func TestAppendAudio_PCM16Boundaries(t *testing.T) {
	endOfSpeech := &pipecatEndOfSpeech{}
	endOfSpeech.appendAudio([]byte{0, 128, 255, 255, 0, 0, 255, 127, 0})
	require.Equal(t, []float32{-1, -1.0 / 32768, 0, 32767.0 / 32768}, endOfSpeech.audioBuffer)
	require.Equal(t, uint64(4), endOfSpeech.audioNextSample)
}

func TestExecuteAudio_SilenceSampleConversion(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		silenceSamples  uint64
		extendedTimeout time.Duration
		audio           []byte
		expectedState   turnState
	}{
		{name: "below limit", extendedTimeout: 2 * pipecatAudioSampleDuration, audio: []byte{0, 0}, expectedState: turnStateIncomplete},
		{name: "at limit", silenceSamples: 1, extendedTimeout: 2 * pipecatAudioSampleDuration, audio: []byte{0, 0}, expectedState: turnStateComplete},
		{name: "fractional sample budget", silenceSamples: 1, extendedTimeout: 2*pipecatAudioSampleDuration + 1, audio: []byte{0, 0}, expectedState: turnStateIncomplete},
		{name: "intermediate product overflow", silenceSamples: 10 * 86400 * pipecatAudioSampleRate, extendedTimeout: time.Second, expectedState: turnStateComplete},
		{name: "largest representable sample duration", silenceSamples: maxAudioDurationSamples, extendedTimeout: time.Duration(math.MaxInt64), expectedState: turnStateIncomplete},
		{name: "duration overflow", silenceSamples: maxAudioDurationSamples + 1, extendedTimeout: time.Duration(math.MaxInt64), expectedState: turnStateComplete},
		{name: "signed count overflow", silenceSamples: math.MaxUint64, extendedTimeout: time.Duration(math.MaxInt64), expectedState: turnStateComplete},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			endOfSpeech := &pipecatEndOfSpeech{
				extendedTimeout: testCase.extendedTimeout,
				commandCh:       make(chan workerCommand, 1),
				stopCh:          make(chan struct{}),
				state: &endOfSpeechState{
					vadState:       vadStateEnded,
					turnState:      turnStateIncomplete,
					silenceSamples: testCase.silenceSamples,
					segment:        speechSegment{FinalText: "finished"},
				},
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: testCase.audio}))
			require.Equal(t, testCase.expectedState, endOfSpeech.state.turnState)
			if testCase.expectedState == turnStateComplete {
				require.Len(t, endOfSpeech.commandCh, 1)
				require.Equal(t, "finished", (<-endOfSpeech.commandCh).segment.Text)
			} else {
				require.Empty(t, endOfSpeech.commandCh)
			}
		})
	}
}

func TestExecuteVADEnd_SampleDurationBounds(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		elapsedSamples uint64
		expectedDelay  time.Duration
	}{
		{name: "one sample", elapsedSamples: 1, expectedDelay: pipecatAudioSampleDuration},
		{name: "normal delay", elapsedSamples: pipecatAudioSampleRate / 2, expectedDelay: 500 * time.Millisecond},
		{name: "intermediate product overflow", elapsedSamples: 10 * 86400 * pipecatAudioSampleRate, expectedDelay: 240 * time.Hour},
		{name: "duration overflow", elapsedSamples: maxAudioDurationSamples + 1, expectedDelay: time.Duration(maxAudioDurationSamples) * pipecatAudioSampleDuration},
		{name: "signed count overflow", elapsedSamples: math.MaxUint64 - pipecatAudioSampleRate, expectedDelay: time.Duration(maxAudioDurationSamples) * pipecatAudioSampleDuration},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			endOfSpeech := &pipecatEndOfSpeech{
				threshold:       0.5,
				fallbackTimeout: 500 * time.Millisecond,
				extendedTimeout: time.Second,
				turnStopTimeout: defaultPctTurnStopTimeout,
				audioNextSample: testCase.elapsedSamples + pipecatAudioSampleRate,
				state:           &endOfSpeechState{},
			}
			earliestDeadline := time.Now().Add(endOfSpeech.fallbackTimeout - testCase.expectedDelay)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd, EndAt: 1,
			}))
			latestDeadline := time.Now().Add(endOfSpeech.fallbackTimeout - testCase.expectedDelay)
			require.False(t, endOfSpeech.state.transcriptDeadline.Before(earliestDeadline))
			require.False(t, endOfSpeech.state.transcriptDeadline.After(latestDeadline))
		})
	}
}

func TestAppendAudio_RollingBuffer(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	// Fill buffer to near capacity
	eos.audioBuffer = make([]float32, maxAudioSamples-10)

	// Append 100 more samples. Oldest samples should be evicted.
	pcm := make([]byte, 200)
	eos.appendAudio(pcm)

	assert.Len(t, eos.audioBuffer, maxAudioSamples)
}

func TestAppendAudio_TracksAbsoluteSampleWindow(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.audioStartSample = 50
	eos.audioBuffer = make([]float32, maxAudioSamples-10)
	eos.audioNextSample = eos.audioStartSample + uint64(len(eos.audioBuffer))

	pcm := make([]byte, 200)
	eos.appendAudio(pcm)

	eos.mu.RLock()
	defer eos.mu.RUnlock()
	assert.Len(t, eos.audioBuffer, maxAudioSamples)
	assert.Equal(t, uint64(140), eos.audioStartSample)
	assert.Equal(t, eos.audioStartSample+uint64(len(eos.audioBuffer)), eos.audioNextSample)
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

func TestPredictEOU_EmptyAudioBufferRemainsIncomplete(t *testing.T) {
	eos := &pipecatEndOfSpeech{}
	probability, err := eos.predictEOU(t.Context())
	require.NoError(t, err)
	assert.Zero(t, probability)
}

func TestPredictEOU_DetectorUnavailableReturnsError(t *testing.T) {
	eos := &pipecatEndOfSpeech{
		audioBuffer: []float32{0.1, -0.1, 0.05},
	}
	probability, err := eos.predictEOU(t.Context())
	require.ErrorIs(t, err, errPipecatDetectorNil)
	assert.Zero(t, probability)
}

func TestPredictEOU_SpeechOffsetBeyondBufferSkipsPrediction(t *testing.T) {
	for _, speechStartSample := range []uint64{preSpeechAudioSamples + 4, math.MaxInt64, math.MaxUint64} {
		endOfSpeech := &pipecatEndOfSpeech{
			audioBuffer:       []float32{0.1, -0.1, 0.05},
			hasSpeechStart:    true,
			speechStartSample: speechStartSample,
		}
		probability, err := endOfSpeech.predictEOU(t.Context())
		require.NoError(t, err, "speech start: %d", speechStartSample)
		require.Zero(t, probability)
	}
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

	for range 2 {
		probability, err := eos.predictEOU(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 0.75, probability)
	}
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
	for range 2 {
		probability, err := eos.predictEOU(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1.0, probability)
	}

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	probability, err := eos.predictEOU(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2.0, probability)
	assert.Equal(t, int32(2), atomic.LoadInt32(&predictorCalls))
}

func TestPredictEOU_UsesSpeechScopedAudio(t *testing.T) {
	captured := make(chan []float32, 1)
	eos := newTestEOSWithPredictor(
		func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}),
		func(audio []float32) (float64, error) {
			captured <- audio
			return 0.7, nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	eos.mu.Lock()
	eos.audioStartSample = 1000
	eos.audioBuffer = make([]float32, 20000)
	for i := range eos.audioBuffer {
		eos.audioBuffer[i] = float32(int(eos.audioStartSample) + i)
	}
	eos.audioNextSample = eos.audioStartSample + uint64(len(eos.audioBuffer))
	eos.hasSpeechStart = true
	eos.speechStartSample = 12000
	eos.audioGeneration = 1
	eos.mu.Unlock()

	probability, err := eos.predictEOU(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 0.7, probability)

	select {
	case audio := <-captured:
		require.Len(t, audio, 17000)
		assert.Equal(t, float32(4000), audio[0])
		assert.Equal(t, float32(20999), audio[len(audio)-1])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for predictor input")
	}
}

func TestPredictEOU_FailedPredictionIsNotCached(t *testing.T) {
	var predictionCalls int
	endOfSpeech := newTestEOSWithPredictor(
		func(context.Context, ...internal_type.Packet) error { return nil }, nil,
		func([]float32) (float64, error) {
			predictionCalls++
			if predictionCalls == 1 {
				return 0, errPipecatDetectorCreateInputTensor
			}
			return 0.9, nil
		},
	)
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	probability, err := endOfSpeech.predictEOU(t.Context())
	require.ErrorIs(t, err, errPipecatDetectorRunInference)
	require.ErrorIs(t, err, errPipecatDetectorCreateInputTensor)
	assert.Zero(t, probability)
	for range 2 {
		probability, err = endOfSpeech.predictEOU(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 0.9, probability)
	}
	assert.Equal(t, 2, predictionCalls)
}

func TestExecuteVADStartMarksSpeechStartFromPacketTimestamp(t *testing.T) {
	eos := newTestEOS(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}))
	defer closeTestEndOfSpeech(eos)

	eos.mu.Lock()
	eos.audioNextSample = 16000
	eos.mu.Unlock()

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source:  internal_type.InterruptionSourceVad,
		Event:   internal_type.InterruptionEventStart,
		StartAt: 0.5,
	}))

	eos.mu.RLock()
	defer eos.mu.RUnlock()
	assert.True(t, eos.hasSpeechStart)
	assert.Equal(t, uint64(8000), eos.speechStartSample)
}

func TestPipecatEndOfSpeech_IgnoresInterruptionForDifferentContext(t *testing.T) {
	eos := &pipecatEndOfSpeech{
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		extendedTimeout: 30 * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
		state: &endOfSpeechState{segment: speechSegment{
			Revision:  1,
			ContextID: "ctx-new",
			FinalText: "new turn",
			Text:      "new turn",
			Timestamp: time.Now(),
		}},
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.EndOfSpeechInterruptionPacket{
		ContextID: "ctx-old",
		Source:    internal_type.InterruptionSourceVad,
	}))

	select {
	case command := <-eos.commandCh:
		t.Fatalf("unexpected command for old context: %+v", command)
	default:
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.EndOfSpeechInterruptionPacket{
		ContextID: "ctx-new",
		Source:    internal_type.InterruptionSourceVad,
	}))

	select {
	case command := <-eos.commandCh:
		assert.Equal(t, "ctx-new", command.segment.ContextID)
	default:
		t.Fatal("expected command for active context")
	}
}

// ============================================================================
// EOS INTEGRATION TESTS (without ONNX model, fallback timeout path)
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
	// Second final STT should accumulate text.
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
	// Without VAD, committed text uses the transcript inactivity budget.
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

func TestEOS_FinalSTTWithoutVADUsesFallbackTimeout(t *testing.T) {
	called := make(chan time.Time, 1)
	var predictorCalls int32
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
			"microphone.eos.extended_timeout": 1000.0,
			"microphone.eos.fallback_timeout": 500.0,
		}),
		func(_ []float32) (float64, error) {
			atomic.AddInt32(&predictorCalls, 1)
			return 0.92, nil
		},
	)
	defer closeTestEndOfSpeech(eos)

	err := eos.Execute(context.Background(), audioInput(1600))
	require.NoError(t, err)

	start := time.Now()
	err = eos.Execute(context.Background(), sttInput("transcript without vad", true))
	require.NoError(t, err)

	select {
	case firedAt := <-called:
		elapsed := firedAt.Sub(start)
		assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond)
		assert.Less(t, elapsed, 800*time.Millisecond)
	case <-time.After(900 * time.Millisecond):
		t.Fatal("expected fallback timeout EOS callback")
	}
	assert.Equal(t, int32(0), atomic.LoadInt32(&predictorCalls))
}

func TestEOS_FinalSTTDoesNotRunSmartTurnPrediction(t *testing.T) {
	var predictorCalls int32
	endOfSpeech := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func([]float32) (float64, error) {
				atomic.AddInt32(&predictorCalls, 1)
				return 0.92, nil
			},
		},
		threshold:       0.5,
		extendedTimeout: 260 * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
		fallbackTimeout: 80 * time.Millisecond,
		audioBuffer:     []float32{0.1, 0.2, 0.3},
		audioGeneration: 1,
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-continuation",
		Script:    "I I just wanted to talk to you and then.",
		Interim:   false,
	}))

	select {
	case command := <-endOfSpeech.commandCh:
		assert.WithinDuration(t, time.Now().Add(80*time.Millisecond), command.deadline, 30*time.Millisecond)
		assert.Equal(t, "I I just wanted to talk to you and then.", command.segment.Text)
		assert.InDelta(t, 0.0, command.confidence, 0.0001)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for transcript safety command")
	}
	assert.Equal(t, int32(0), atomic.LoadInt32(&predictorCalls))
}

func TestEOS_FinalSTTWithoutVADIgnoresIncompletePrediction(t *testing.T) {
	called := make(chan time.Time, 1)
	var predictorCalls int32
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
			"microphone.eos.extended_timeout": 260.0,
			"microphone.eos.fallback_timeout": 80.0,
		}),
		func(_ []float32) (float64, error) {
			atomic.AddInt32(&predictorCalls, 1)
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
		assert.GreaterOrEqual(t, elapsed, 45*time.Millisecond)
		assert.Less(t, elapsed, 250*time.Millisecond)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected fallback timeout EOS callback")
	}
	assert.Equal(t, int32(0), atomic.LoadInt32(&predictorCalls))
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

func TestEOS_VADStartCancelsPendingFinalUntilVADEnd(t *testing.T) {
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.threshold":        0.5,
		"microphone.eos.fallback_timeout": 60.0,
		"microphone.eos.extended_timeout": 120.0,
	}), func([]float32) (float64, error) {
		return 0.9, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), sttInput("hello", true)))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))

	select {
	case p := <-called:
		t.Fatalf("callback fired after speech restarted: %+v", p)
	case <-time.After(120 * time.Millisecond):
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

func TestEOS_FinalSTTWhileVADSpeakingWaitsForVADEnd(t *testing.T) {
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.threshold":        0.5,
		"microphone.eos.fallback_timeout": 50.0,
		"microphone.eos.extended_timeout": 120.0,
	}), func([]float32) (float64, error) {
		return 0.9, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("hello", true)))

	select {
	case p := <-called:
		t.Fatalf("unexpected callback while VAD is still speaking: %+v", p)
	case <-time.After(150 * time.Millisecond):
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case p := <-called:
		assert.Equal(t, "hello", p.Speech)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for callback after VAD ended")
	}
}

func TestEOS_FinalSTTAfterIncompleteVADEndSchedulesFallback(t *testing.T) {
	var predictorCalls int32
	eos := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func([]float32) (float64, error) {
				atomic.AddInt32(&predictorCalls, 1)
				return 0.1, nil
			},
		},
		threshold:       0.5,
		extendedTimeout: 250 * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
		fallbackTimeout: 60 * time.Millisecond,
		audioBuffer:     []float32{0.1, 0.2, 0.3},
		audioGeneration: 1,
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("not done yet", true)))
	select {
	case command := <-eos.commandCh:
		assert.Equal(t, "not done yet", command.segment.Text)
		assert.False(t, command.fireImmediately)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("incomplete turn did not schedule its fallback")
	}
	require.NoError(t, eos.Execute(context.Background(), audioInput(4000)))

	select {
	case command := <-eos.commandCh:
		assert.False(t, command.fireImmediately)
		assert.WithinDuration(t, time.Now().Add(60*time.Millisecond), command.deadline, 30*time.Millisecond)
		assert.Equal(t, "not done yet", command.segment.Text)
		assert.InDelta(t, 0.1, command.confidence, 0.0001)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for model-backed command")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&predictorCalls))
}

func TestEOS_FinalSTTAfterVADEndCompletePredictionWaitsForTranscriptDeadline(t *testing.T) {
	var predictorCalls int32
	eos := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func([]float32) (float64, error) {
				atomic.AddInt32(&predictorCalls, 1)
				return 0.9, nil
			},
		},
		threshold:       0.5,
		extendedTimeout: 250 * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
		fallbackTimeout: 60 * time.Millisecond,
		audioBuffer:     []float32{0.1, 0.2, 0.3},
		audioGeneration: 1,
		commandCh:       make(chan workerCommand, 1),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{segment: speechSegment{}},
	}

	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("done now", true)))

	select {
	case command := <-eos.commandCh:
		assert.False(t, command.fireImmediately)
		assert.WithinDuration(t, time.Now().Add(60*time.Millisecond), command.deadline, 30*time.Millisecond)
		assert.Equal(t, "done now", command.segment.Text)
		assert.InDelta(t, 0.9, command.confidence, 0.0001)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for model-backed transcript deadline")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&predictorCalls))
}

func TestEOS_VADEndCompletePredictionWaitsForTranscriptDeadline(t *testing.T) {
	var predictorCalls int32
	endOfSpeech := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{
			predict: func([]float32) (float64, error) {
				atomic.AddInt32(&predictorCalls, 1)
				return 0.88, nil
			},
		},
		threshold:       0.5,
		extendedTimeout: 300 * time.Millisecond,
		turnStopTimeout: defaultPctTurnStopTimeout,
		fallbackTimeout: 60 * time.Millisecond,
		audioBuffer:     []float32{0.1, 0.2, 0.3},
		audioGeneration: 1,
		commandCh:       make(chan workerCommand, 2),
		stopCh:          make(chan struct{}),
		state: &endOfSpeechState{
			segment: speechSegment{
				Revision:  7,
				ContextID: "ctx-vad-continuation",
				FinalText: "I just wanted to talk with you and then",
				Text:      "I just wanted to talk with you and then",
				Timestamp: time.Now(),
			},
			transcript: transcriptStateFinalized,
		},
	}

	require.NoError(t, endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		ContextID: "ctx-vad-continuation",
		Source:    internal_type.InterruptionSourceVad,
		Event:     internal_type.InterruptionEventEnd,
	}))

	select {
	case command := <-endOfSpeech.commandCh:
		assert.Equal(t, 0.0, command.confidence)
		assert.Equal(t, "I just wanted to talk with you and then", command.segment.Text)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("VAD end did not arm completion before inference")
	}
	select {
	case command := <-endOfSpeech.commandCh:
		assert.False(t, command.fireImmediately)
		assert.WithinDuration(t, time.Now().Add(60*time.Millisecond), command.deadline, 30*time.Millisecond)
		assert.Equal(t, "I just wanted to talk with you and then", command.segment.Text)
		assert.InDelta(t, 0.88, command.confidence, 0.0001)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for VAD transcript deadline")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&predictorCalls))
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.threshold":        0.5,
		"microphone.eos.fallback_timeout": 100.0,
	}), func([]float32) (float64, error) {
		return 0.9, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 200.0,
	}), func([]float32) (float64, error) { return 0.9, nil })
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 70.0,
	}), func([]float32) (float64, error) { return 0.9, nil })
	defer closeTestEndOfSpeech(eos)

	ctx := context.Background()
	require.NoError(t, eos.Execute(ctx, audioInput(1600)))
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

func TestEOS_InterimWithVADCompletePredictionWaitsForCommittedText(t *testing.T) {
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

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.threshold":        0.5,
		"microphone.eos.fallback_timeout": 50.0,
	}), func([]float32) (float64, error) {
		return 0.91, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("interim complete", false)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))

	select {
	case p := <-called:
		t.Fatalf("interim-only transcript completed: %+v", p)
	case <-time.After(100 * time.Millisecond):
	}
	require.NoError(t, eos.Execute(context.Background(), sttInput("committed text", true)))
	select {
	case p := <-called:
		assert.Equal(t, "committed text", p.Speech)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for committed text after the transcript deadline")
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
	if _, err := os.Stat(resolvePctModelPath("")); err != nil {
		t.Skipf("pipecat model asset unavailable: %v", err)
	}

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

	eos, err := New(
		WithContext(context.Background()),
		WithLogger(logger),
		WithOnPacket(callback),
		WithOptions(utils.Option{}),
	)
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
	if _, err := os.Stat(resolvePctModelPath("")); err != nil {
		t.Skipf("pipecat model asset unavailable: %v", err)
	}

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

	eos, err := New(
		WithContext(context.Background()),
		WithLogger(logger),
		WithOnPacket(callback),
		WithOptions(utils.Option{}),
	)
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
		"microphone.eos.threshold":        0.5,
		"microphone.eos.fallback_timeout": 10.0,
	}), func([]float32) (float64, error) {
		return 0.7345, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-confidence",
		Script:    "hello there",
		Interim:   false,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
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
		"microphone.eos.fallback_timeout": 20.0,
	}), func([]float32) (float64, error) {
		return 0.3125, nil
	})
	defer closeTestEndOfSpeech(eos)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.SpeechToTextPacket{
		ContextID: "ctx-extend-confidence",
		Script:    "continue",
		Interim:   false,
	}))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventEnd,
	}))
	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))

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
	var predictionCalls int32
	predictionStarted := make(chan struct{}, 2)
	releasePrediction := make(chan struct{}, 2)
	stopped := make(chan error, 2)

	callback := func(context.Context, ...internal_type.Packet) error { return nil }
	predictor := func([]float32) (float64, error) {
		current := atomic.AddInt32(&inFlight, 1)
		for {
			maximum := atomic.LoadInt32(&maxInFlight)
			if current <= maximum || atomic.CompareAndSwapInt32(&maxInFlight, maximum, current) {
				break
			}
		}
		atomic.AddInt32(&predictionCalls, 1)
		predictionStarted <- struct{}{}
		<-releasePrediction
		atomic.AddInt32(&inFlight, -1)
		return 0.0, nil
	}

	eos := newTestEOSWithPredictor(callback, newTestOpts(map[string]any{
		"microphone.eos.extended_timeout": 500.0,
	}), predictor)
	defer closeTestEndOfSpeech(eos)
	defer close(releasePrediction)

	require.NoError(t, eos.Execute(context.Background(), audioInput(1600)))
	require.NoError(t, eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad,
		Event:  internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(context.Background(), sttInput("serialized", true)))

	go func() {
		stopped <- eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		})
	}()
	select {
	case <-predictionStarted:
	case <-time.After(time.Second):
		t.Fatal("first prediction did not start")
	}
	require.NoError(t, eos.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, eos.Execute(t.Context(), audioInput(1600)))
	go func() {
		stopped <- eos.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		})
	}()
	require.Eventually(t, func() bool {
		eos.mu.RLock()
		defer eos.mu.RUnlock()
		return eos.state.vadState == vadStateEnded
	}, time.Second, time.Millisecond)
	select {
	case <-predictionStarted:
		t.Error("second prediction entered while the first was still running")
	case <-time.After(30 * time.Millisecond):
	}
	releasePrediction <- struct{}{}
	releasePrediction <- struct{}{}
	for range 2 {
		select {
		case err := <-stopped:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("prediction did not finish after release")
		}
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&maxInFlight))
	assert.Equal(t, int32(2), atomic.LoadInt32(&predictionCalls))
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

func TestEOS_IncompleteTurnAudioCanCompleteBeforeWallDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		textBeforeSilence bool
	}{
		{name: "committed text with newer interim", textBeforeSilence: true},
		{name: "committed text after silence", textBeforeSilence: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 2)
			endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- result
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 10.0,
				"microphone.eos.extended_timeout": 1000.0,
			}), func([]float32) (float64, error) { return 0.5, nil })
			defer closeTestEndOfSpeech(endOfSpeech)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			}))
			if testCase.textBeforeSilence {
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed", true)))
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("unconfirmed tail", false)))
			}
			select {
			case packet := <-completed:
				t.Fatalf("incomplete turn released by transcript timeout: %+v", packet)
			case <-time.After(120 * time.Millisecond):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(15999)))
			select {
			case packet := <-completed:
				t.Fatalf("turn released before silent audio limit: %+v", packet)
			default:
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1)))
			if !testCase.textBeforeSilence {
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed", true)))
			}
			select {
			case packet := <-completed:
				assert.Equal(t, "committed", packet.Speech)
			case <-time.After(200 * time.Millisecond):
				t.Fatal("silence-complete turn did not release committed transcript")
			}
		})
	}
}

func TestEOS_TranscriptDuringInferenceKeepsOriginalDeadline(t *testing.T) {
	predictionStarted := make(chan struct{})
	finishPrediction := make(chan struct{})
	vadStopped := make(chan error, 1)
	endOfSpeech := &pipecatEndOfSpeech{
		onPacket: func(context.Context, ...internal_type.Packet) error { return nil },
		predictor: testPredictor{predict: func([]float32) (float64, error) {
			close(predictionStarted)
			<-finishPrediction
			return 0.9, nil
		}},
		threshold:       0.5,
		fallbackTimeout: 20 * time.Millisecond,
		extendedTimeout: time.Second,
		turnStopTimeout: defaultPctTurnStopTimeout,
		commandCh:       make(chan workerCommand, 4),
		stopCh:          make(chan struct{}),
		state:           &endOfSpeechState{},
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	go func() {
		vadStopped <- endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		})
	}()
	<-predictionStarted
	endOfSpeech.mu.RLock()
	deadline := endOfSpeech.state.transcriptDeadline
	endOfSpeech.mu.RUnlock()
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("first", true)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("second", true)))
	for _, transcript := range []string{"first", "first second"} {
		select {
		case command := <-endOfSpeech.commandCh:
			assert.Equal(t, transcript, command.segment.Text)
			assert.Equal(t, deadline, command.deadline)
			assert.Equal(t, 0.0, command.confidence)
		case <-time.After(time.Second):
			t.Fatal("transcript did not arm completion during inference")
		}
	}
	time.Sleep(30 * time.Millisecond)
	close(finishPrediction)
	require.NoError(t, <-vadStopped)
	select {
	case command := <-endOfSpeech.commandCh:
		assert.Equal(t, "first second", command.segment.Text)
		assert.Equal(t, deadline, command.deadline)
		assert.True(t, command.deadline.Before(time.Now()))
		assert.Equal(t, 0.9, command.confidence)
	case <-time.After(time.Second):
		t.Fatal("transcript update invalidated the current VAD prediction")
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("uncertain tail", false)))
	select {
	case command := <-endOfSpeech.commandCh:
		assert.Equal(t, deadline, command.deadline)
		assert.Equal(t, "first second", command.segment.Text)
	case <-time.After(time.Second):
		t.Fatal("interim update lost the committed transcript deadline")
	}
}

func TestEOS_ResumedIncompleteSpeechRetainsFirstAudioOnset(t *testing.T) {
	var analyzedSampleCounts []int
	endOfSpeech := newTestEOSWithPredictor(func(context.Context, ...internal_type.Packet) error { return nil },
		newTestOpts(map[string]any{}), func(audio []float32) (float64, error) {
			analyzedSampleCounts = append(analyzedSampleCounts, len(audio))
			return 0.1, nil
		})
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart, StartAt: 0.1,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(16000)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart, StartAt: 1.2,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(16000)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	assert.Equal(t, []int{17600, 35200}, analyzedSampleCounts)
}

func TestEOS_VADRestartDiscardsInFlightPrediction(t *testing.T) {
	predictionStarted := make(chan struct{})
	finishPrediction := make(chan struct{})
	vadStopped := make(chan error, 1)
	completed := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				completed <- result
			}
		}
		return nil
	}, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 10.0}), func([]float32) (float64, error) {
		close(predictionStarted)
		<-finishPrediction
		return 0.9, nil
	})
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("first burst", true)))
	go func() {
		vadStopped <- endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		})
	}()
	<-predictionStarted
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("continuation", true)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), interruptInput()))
	close(finishPrediction)
	require.NoError(t, <-vadStopped)
	select {
	case packet := <-completed:
		t.Fatalf("stale prediction or interruption completed resumed speech: %+v", packet)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEOS_CompletedCallbackPreservesNextTurn(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 2)
	var endOfSpeech *pipecatEndOfSpeech
	endOfSpeech = newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				completed <- result
				if result.Speech == "first turn" {
					return endOfSpeech.Execute(ctx, sttInput("next turn", true))
				}
			}
		}
		return nil
	}, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 10.0}))
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), userInput("first turn")))
	for _, speech := range []string{"first turn", "next turn"} {
		select {
		case packet := <-completed:
			assert.Equal(t, speech, packet.Speech)
		case <-time.After(time.Second):
			t.Fatalf("missing completion for %q", speech)
		}
	}
}

func TestEOS_VADDeadlineDiscountsBufferedStopSilence(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		endAt           float64
		remainingBudget time.Duration
	}{
		{name: "known audio timestamp", endAt: 0.8, remainingBudget: 300 * time.Millisecond},
		{name: "missing timestamp", remainingBudget: 500 * time.Millisecond},
		{name: "timestamp ahead of audio", endAt: 2, remainingBudget: 500 * time.Millisecond},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			endOfSpeech := &pipecatEndOfSpeech{
				onPacket:        func(context.Context, ...internal_type.Packet) error { return nil },
				predictor:       testPredictor{predict: func([]float32) (float64, error) { return 0.9, nil }},
				threshold:       0.5,
				fallbackTimeout: 500 * time.Millisecond,
				turnStopTimeout: defaultPctTurnStopTimeout,
				commandCh:       make(chan workerCommand, 2),
				stopCh:          make(chan struct{}),
				state:           &endOfSpeechState{},
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(16000)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed", true)))
			stoppedAt := time.Now()
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				EndAt: testCase.endAt,
			}))
			for commandIndex := 0; commandIndex < 2; commandIndex++ {
				select {
				case command := <-endOfSpeech.commandCh:
					assert.WithinDuration(t, stoppedAt.Add(testCase.remainingBudget), command.deadline, 20*time.Millisecond)
				case <-time.After(time.Second):
					t.Fatal("missing transcript safety deadline")
				}
			}
		})
	}
}

func TestEOS_VADWithoutPredictionAudioCanCompleteBeforeWallDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		audioSamples  int
		predictor     turnPredictor
		expectedError error
	}{
		{name: "empty audio"},
		{name: "unavailable predictor", audioSamples: 1600, expectedError: errPipecatDetectorNil},
		{
			name: "failed inference", audioSamples: 1600,
			predictor: testPredictor{predict: func([]float32) (float64, error) {
				return 0, errPipecatDetectorCreateInputTensor
			}},
			expectedError: errPipecatDetectorCreateInputTensor,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			inferenceErrors := make(chan internal_type.ObservabilityLogRecordPacket, 1)
			endOfSpeech := newTestEOS(func(ctx context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						completed <- result
					}
					if record, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok && record.Record.Level == observability.LevelError {
						inferenceErrors <- record
					}
				}
				return nil
			}, newTestOpts(map[string]any{
				"microphone.eos.fallback_timeout": 10.0,
				"microphone.eos.extended_timeout": 1000.0,
			}))
			endOfSpeech.predictor = testCase.predictor
			defer closeTestEndOfSpeech(endOfSpeech)
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}))
			require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(testCase.audioSamples)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed", true)))
			err := endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
				Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
			})
			require.ErrorIs(t, err, testCase.expectedError)
			if testCase.expectedError != nil {
				select {
				case record := <-inferenceErrors:
					assert.Contains(t, record.Record.Message, testCase.expectedError.Error())
				default:
					t.Fatal("inference failure was not reported")
				}
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("tail", true)))
			require.NoError(t, endOfSpeech.Execute(t.Context(), interruptInput()))
			select {
			case packet := <-completed:
				t.Fatalf("missing prediction released text at the transcript deadline: %+v", packet)
			case <-time.After(80 * time.Millisecond):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 16000)}))
			select {
			case packet := <-completed:
				t.Fatalf("turn completed before the received-silence limit: %+v", packet)
			case <-time.After(20 * time.Millisecond):
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 16000)}))
			select {
			case packet := <-completed:
				assert.Equal(t, "committed tail", packet.Speech)
			case <-time.After(200 * time.Millisecond):
				t.Fatal("received-silence limit did not release committed text")
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 1920)}))
			select {
			case packet := <-completed:
				t.Fatalf("turn completed twice: %+v", packet)
			case <-time.After(20 * time.Millisecond):
			}
		})
	}
}

func TestEOS_ResumedSpeechRecoversAfterInferenceFailure(t *testing.T) {
	completed := make(chan internal_type.EndOfSpeechPacket, 4)
	var predictionSamples []int
	endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				completed <- result
			}
		}
		return nil
	}, newTestOpts(map[string]any{"microphone.eos.fallback_timeout": 10.0}), func(audio []float32) (float64, error) {
		predictionSamples = append(predictionSamples, len(audio))
		if len(predictionSamples) == 1 {
			return 0, errPipecatDetectorRunInference
		}
		return 0.9, nil
	})
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("first part", true)))
	require.ErrorIs(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}), errPipecatDetectorRunInference)
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}))
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("second part", true)))
	select {
	case packet := <-completed:
		t.Fatalf("failed inference released speech before the next stop: %+v", packet)
	case <-time.After(30 * time.Millisecond):
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
		Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}))
	select {
	case packet := <-completed:
		assert.Equal(t, "first part second part", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("successful inference did not release the resumed turn")
	}
	assert.Equal(t, []int{1600, 3200}, predictionSamples)
}

func TestEOS_QueuedUserTextPreservesNewerInput(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		secondInput string
	}{
		{name: "two explicit messages", secondInput: "text"},
		{name: "speech starts before delivery", secondInput: "vad"},
		{name: "transcript before delivery", secondInput: "stt"},
		{name: "speech stops before delivery", secondInput: "stop"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			completed := make(chan internal_type.EndOfSpeechPacket, 4)
			endOfSpeech := &pipecatEndOfSpeech{
				predictor: testPredictor{predict: func([]float32) (float64, error) { return 0.9, nil }},
				onPacket: func(ctx context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
							completed <- result
						}
					}
					return nil
				},
				fallbackTimeout: 10 * time.Millisecond,
				turnStopTimeout: defaultPctTurnStopTimeout,
				commandCh:       make(chan workerCommand, 4),
				stopCh:          make(chan struct{}),
				state:           &endOfSpeechState{},
			}
			defer closeTestEndOfSpeech(endOfSpeech)
			if testCase.secondInput == "stop" {
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
			}
			require.NoError(t, endOfSpeech.Execute(t.Context(), userInput("first")))
			switch testCase.secondInput {
			case "text":
				require.NoError(t, endOfSpeech.Execute(t.Context(), userInput("second")))
			case "stt":
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("second", true)))
			case "vad":
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}))
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("second", true)))
			case "stop":
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				}))
				require.NoError(t, endOfSpeech.Execute(t.Context(), interruptInput()))
				require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
			}
			go endOfSpeech.worker()
			select {
			case packet := <-completed:
				assert.Equal(t, "first", packet.Speech)
			case <-time.After(time.Second):
				t.Fatal("accepted explicit text was discarded")
			}
			if testCase.secondInput == "vad" {
				require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
				require.NoError(t, endOfSpeech.Execute(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				}))
			}
			if testCase.secondInput == "stop" {
				select {
				case packet := <-completed:
					t.Fatalf("VAD re-emitted explicit text: %+v", packet)
				case <-time.After(30 * time.Millisecond):
				}
				require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("second", true)))
			}
			select {
			case packet := <-completed:
				assert.Equal(t, "second", packet.Speech)
			case <-time.After(time.Second):
				t.Fatal("older explicit text cleared newer input")
			}
		})
	}
}

func TestEOS_OrphanVADStopWaitsForSilentAudio(t *testing.T) {
	predictionStarted := make(chan struct{})
	finishPrediction := make(chan struct{})
	vadStopped := make(chan error, 1)
	completed := make(chan internal_type.EndOfSpeechPacket, 2)
	endOfSpeech := newTestEOSWithPredictor(func(ctx context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if result, ok := packet.(internal_type.EndOfSpeechPacket); ok {
				completed <- result
			}
		}
		return nil
	}, newTestOpts(map[string]any{
		"microphone.eos.fallback_timeout": 20.0,
		"microphone.eos.extended_timeout": 100.0,
	}), func([]float32) (float64, error) {
		close(predictionStarted)
		<-finishPrediction
		return 0.1, nil
	})
	defer closeTestEndOfSpeech(endOfSpeech)
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	require.NoError(t, endOfSpeech.Execute(t.Context(), sttInput("committed", true)))
	go func() {
		vadStopped <- endOfSpeech.Execute(context.Background(), internal_type.InterruptionDetectedPacket{
			Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		})
	}()
	<-predictionStarted
	time.Sleep(40 * time.Millisecond)
	select {
	case packet := <-completed:
		t.Errorf("old transcript timer completed pending inference: %+v", packet)
	default:
	}
	close(finishPrediction)
	require.NoError(t, <-vadStopped)
	select {
	case packet := <-completed:
		t.Fatalf("old transcript timer completed an incomplete turn: %+v", packet)
	case <-time.After(40 * time.Millisecond):
	}
	require.NoError(t, endOfSpeech.Execute(t.Context(), audioInput(1600)))
	select {
	case packet := <-completed:
		assert.Equal(t, "committed", packet.Speech)
	case <-time.After(time.Second):
		t.Fatal("orphan VAD stop stranded text after the silent audio limit")
	}
}

func TestEOS_FactoryCreationFails_NoModel(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	_, err := New(
		WithContext(context.Background()),
		WithLogger(logger),
		WithOnPacket(func(context.Context, ...internal_type.Packet) error { return nil }),
		WithOptions(utils.Option{"microphone.eos.pipecat.model_path": "/nonexistent/model.onnx"}),
	)
	assert.Error(t, err)
}
