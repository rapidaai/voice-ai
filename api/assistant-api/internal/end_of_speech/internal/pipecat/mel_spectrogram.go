// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"math"
	"math/cmplx"
)

// whisperFeatures extracts Whisper-compatible mel spectrogram features from
// raw float32 PCM audio (16kHz mono, normalized to [-1, 1]).
//
// Pre-computes mel filterbank and Hann window at construction time.
type whisperFeatures struct {
	melFilters [whisperNMels][whisperNFreqBins]float64
	hannWindow [whisperNFFT]float64
	dftCos     [whisperNFreqBins][whisperNFFT]float64
	dftSin     [whisperNFreqBins][whisperNFFT]float64
}

type whisperFeatureScratch struct {
	prepared [whisperMaxSamples]float32
	padded   [whisperMaxSamples + whisperNFFT]float32
	power    [whisperNFreqBins]float64
	logMel   [whisperNMels * whisperMaxFrames]float64
	output   [whisperNMels * whisperMaxFrames]float32
}

func newWhisperFeatureScratch() *whisperFeatureScratch {
	return &whisperFeatureScratch{}
}

func newWhisperFeatures() *whisperFeatures {
	wf := &whisperFeatures{}
	wf.initHannWindow()
	wf.initMelFilterbank()
	wf.initDFT()
	return wf
}

// Extract computes mel spectrogram features from float32 PCM samples (16kHz).
// Returns a flat float32 slice of shape [whisperNMels * whisperMaxFrames] = [80*800].
// Audio is truncated to last 8 seconds or zero-padded at the beginning.
func (wf *whisperFeatures) Extract(audio []float32) []float32 {
	scratch := newWhisperFeatureScratch()
	output := make([]float32, whisperNMels*whisperMaxFrames)
	return wf.extractInto(audio, output, scratch)
}

func (wf *whisperFeatures) extractInto(audio []float32, output []float32, scratch *whisperFeatureScratch) []float32 {
	samples := prepareAudioInto(audio, scratch.prepared[:])
	normalize(samples)
	padded := reflectPadInto(samples, whisperNFFT/2, scratch.padded[:])
	logMel := scratch.logMel[:]
	output = output[:whisperNMels*whisperMaxFrames]

	globalMax := -math.MaxFloat64
	for frame := 0; frame < whisperMaxFrames; frame++ {
		frameStartSample := frame * whisperHopLength
		frameSamples := padded[frameStartSample : frameStartSample+whisperNFFT]
		for frequencyBin := 0; frequencyBin < whisperNFreqBins; frequencyBin++ {
			var realPart, imaginaryPart float64
			for sampleIndex := 0; sampleIndex < whisperNFFT; sampleIndex++ {
				windowedSample := float64(frameSamples[sampleIndex]) * wf.hannWindow[sampleIndex]
				realPart += windowedSample * wf.dftCos[frequencyBin][sampleIndex]
				imaginaryPart -= windowedSample * wf.dftSin[frequencyBin][sampleIndex]
			}
			scratch.power[frequencyBin] = realPart*realPart + imaginaryPart*imaginaryPart
		}

		for mel := 0; mel < whisperNMels; mel++ {
			var melValue float64
			for bin := 0; bin < whisperNFreqBins; bin++ {
				if wf.melFilters[mel][bin] == 0 {
					continue
				}
				melValue += wf.melFilters[mel][bin] * scratch.power[bin]
			}
			if melValue < 1e-10 {
				melValue = 1e-10
			}
			logValue := math.Log10(melValue)
			logMel[mel*whisperMaxFrames+frame] = logValue
			if logValue > globalMax {
				globalMax = logValue
			}
		}
	}

	clampMin := globalMax - 8.0
	for mel := 0; mel < whisperNMels; mel++ {
		offset := mel * whisperMaxFrames
		for frame := 0; frame < whisperMaxFrames; frame++ {
			value := logMel[offset+frame]
			if value < clampMin {
				value = clampMin
			}
			output[offset+frame] = float32((value + 4.0) / 4.0)
		}
	}

	return output
}

// prepareAudio truncates to last 8 seconds or zero-pads at the beginning.
func prepareAudio(audio []float32) []float32 {
	padded := make([]float32, whisperMaxSamples)
	return prepareAudioInto(audio, padded)
}

func prepareAudioInto(audio []float32, padded []float32) []float32 {
	samples := padded[:whisperMaxSamples]
	if len(audio) >= whisperMaxSamples {
		copy(samples, audio[len(audio)-whisperMaxSamples:])
		return samples
	}
	offset := whisperMaxSamples - len(audio)
	clear(samples[:offset])
	copy(samples[offset:], audio)
	return samples
}

// normalize applies zero-mean unit-variance normalization in-place.
func normalize(samples []float32) {
	n := float64(len(samples))
	if n == 0 {
		return
	}

	var sum float64
	for _, s := range samples {
		sum += float64(s)
	}
	mean := sum / n

	var variance float64
	for _, s := range samples {
		d := float64(s) - mean
		variance += d * d
	}
	variance /= n

	stddev := math.Sqrt(variance + 1e-7)
	for i, s := range samples {
		samples[i] = float32((float64(s) - mean) / stddev)
	}
}

// reflectPad applies reflect padding on both sides of the signal.
func reflectPad(signal []float32, padSize int) []float32 {
	padded := make([]float32, padSize+len(signal)+padSize)
	return reflectPadInto(signal, padSize, padded)
}

func reflectPadInto(signal []float32, padSize int, padded []float32) []float32 {
	n := len(signal)
	output := padded[:padSize+n+padSize]

	for i := 0; i < padSize; i++ {
		idx := padSize - i
		if idx >= n {
			idx = n - 1
		}
		output[i] = signal[idx]
	}

	copy(output[padSize:], signal)

	for i := 0; i < padSize; i++ {
		idx := n - 2 - i
		if idx < 0 {
			idx = 0
		}
		output[padSize+n+i] = signal[idx]
	}

	return output
}

// fft performs in-place radix-2 Cooley-Tukey FFT.
// Input length must be a power of 2.
func fft(x []complex128) {
	n := len(x)
	if n <= 1 {
		return
	}

	// Bit-reversal permutation
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}

	// Butterfly stages
	for size := 2; size <= n; size <<= 1 {
		halfSize := size >> 1
		wBase := -2.0 * math.Pi / float64(size)
		for start := 0; start < n; start += size {
			wn := complex(1, 0)
			wStep := cmplx.Exp(complex(0, wBase))
			for k := 0; k < halfSize; k++ {
				t := wn * x[start+k+halfSize]
				x[start+k+halfSize] = x[start+k] - t
				x[start+k] = x[start+k] + t
				wn *= wStep
			}
		}
	}
}

// initHannWindow pre-computes the Hann window of size nFFT.
// Matches numpy: hann(n+1)[:-1] i.e. periodic Hann window.
func (wf *whisperFeatures) initHannWindow() {
	for i := 0; i < whisperNFFT; i++ {
		wf.hannWindow[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(whisperNFFT)))
	}
}

func (wf *whisperFeatures) initDFT() {
	for bin := 0; bin < whisperNFreqBins; bin++ {
		for sample := 0; sample < whisperNFFT; sample++ {
			angle := 2.0 * math.Pi * float64(bin) * float64(sample) / float64(whisperNFFT)
			wf.dftCos[bin][sample] = math.Cos(angle)
			wf.dftSin[bin][sample] = math.Sin(angle)
		}
	}
}

// initMelFilterbank computes the mel filterbank matrix using the Slaney mel
// scale and Slaney normalization (area = 1 per filter).
func (wf *whisperFeatures) initMelFilterbank() {
	fMax := float64(whisperSampleRate) / 2.0

	// n_mels + 2 linearly spaced points in mel domain
	melMin := hzToMel(0)
	melMax := hzToMel(fMax)
	nPoints := whisperNMels + 2
	melPoints := make([]float64, nPoints)
	for i := range melPoints {
		melPoints[i] = melMin + float64(i)*(melMax-melMin)/float64(nPoints-1)
	}

	// Convert back to Hz
	hzPoints := make([]float64, nPoints)
	for i, m := range melPoints {
		hzPoints[i] = melToHz(m)
	}

	// FFT bin frequencies
	fftFreqs := make([]float64, whisperNFreqBins)
	for i := range fftFreqs {
		fftFreqs[i] = float64(i) * float64(whisperSampleRate) / float64(whisperNFFT)
	}

	// Build triangular filters with Slaney normalization
	for i := 0; i < whisperNMels; i++ {
		lower := hzPoints[i]
		center := hzPoints[i+1]
		upper := hzPoints[i+2]

		enorm := 2.0 / (upper - lower) // Slaney normalization

		for j := 0; j < whisperNFreqBins; j++ {
			f := fftFreqs[j]
			if f >= lower && f < center && center > lower {
				wf.melFilters[i][j] = enorm * (f - lower) / (center - lower)
			} else if f >= center && f <= upper && upper > center {
				wf.melFilters[i][j] = enorm * (upper - f) / (upper - center)
			}
		}
	}
}

func hzToMel(hz float64) float64 {
	if hz < melMinLogHz {
		return hz / melFSP
	}
	return melMinLogM + math.Log(hz/melMinLogHz)/melLogStep
}

func melToHz(mel float64) float64 {
	if mel < melMinLogM {
		return melFSP * mel
	}
	return melMinLogHz * math.Exp(melLogStep*(mel-melMinLogM))
}
