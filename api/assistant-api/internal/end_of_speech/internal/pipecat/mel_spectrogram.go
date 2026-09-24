// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_pipecat

import (
	"math"

	"gonum.org/v1/gonum/dsp/fourier"
)

// whisperFeatures extracts Whisper-compatible mel spectrogram features from
// raw float32 PCM audio (16kHz mono, scaled to [-1, 1]).
//
// Pre-computes mel filterbank and Hann window at construction time.
type whisperFeatures struct {
	melFilters [whisperNMels][whisperNFreqBins]float64
	hannWindow [whisperNFFT]float64
}

type whisperFeatureScratch struct {
	// The transform mutates its workspace, so each scratch owns a separate instance.
	transform    *fourier.FFT
	windowed     [whisperNFFT]float64
	coefficients [whisperNFreqBins]complex128
	prepared     [whisperMaxSamples]float32
	padded       [whisperMaxSamples + whisperNFFT]float32
	power        [whisperNFreqBins]float64
	logMel       [whisperNMels * whisperMaxFrames]float64
	output       [whisperNMels * whisperMaxFrames]float32
}

func newWhisperFeatureScratch() *whisperFeatureScratch {
	return &whisperFeatureScratch{transform: fourier.NewFFT(whisperNFFT)}
}

func newWhisperFeatures() *whisperFeatures {
	wf := &whisperFeatures{}
	wf.initHannWindow()
	wf.initMelFilterbank()
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
		for sampleIndex := range scratch.windowed {
			scratch.windowed[sampleIndex] = float64(frameSamples[sampleIndex]) * wf.hannWindow[sampleIndex]
		}
		scratch.transform.Coefficients(scratch.coefficients[:], scratch.windowed[:])
		for frequencyBin, coefficient := range scratch.coefficients {
			scratch.power[frequencyBin] = real(coefficient)*real(coefficient) + imag(coefficient)*imag(coefficient)
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

// Match Pipecat's NumPy 1.26 buffered float32 sums and scalar epsilon arithmetic.
// The bounded tree retains pairwise order without recursion or waveform copies.
func normalize(samples []float32) {
	if len(samples) == 0 {
		return
	}

	var mean, variance float32
	var reductionNodes [whisperReductionTreeNodes]struct {
		offset int
		count  int
		sum    float32
	}
	for _, computeVariance := range [...]bool{false, true} {
		var sum float32
		for chunkOffset := 0; chunkOffset < len(samples); chunkOffset += whisperReductionChunkSamples {
			clear(reductionNodes[:])
			reductionNodes[1].offset = chunkOffset
			reductionNodes[1].count = min(whisperReductionChunkSamples, len(samples)-chunkOffset)
			for nodeIndex := 1; nodeIndex < len(reductionNodes); nodeIndex++ {
				node := &reductionNodes[nodeIndex]
				if node.count == 0 {
					continue
				}
				if node.count > whisperReductionLeafSamples {
					halfCount := node.count / 2
					halfCount -= halfCount % whisperReductionLanes
					reductionNodes[nodeIndex*2].offset = node.offset
					reductionNodes[nodeIndex*2].count = halfCount
					reductionNodes[nodeIndex*2+1].offset = node.offset + halfCount
					reductionNodes[nodeIndex*2+1].count = node.count - halfCount
					continue
				}
				var lanes [whisperReductionLanes]float32
				alignedCount := node.count - node.count%whisperReductionLanes
				for sampleOffset := 0; sampleOffset < alignedCount; sampleOffset++ {
					value := samples[node.offset+sampleOffset]
					if computeVariance {
						value -= mean
						value = float32(value * value)
					}
					if sampleOffset < whisperReductionLanes {
						lanes[sampleOffset] = value
					} else {
						lanes[sampleOffset%whisperReductionLanes] += value
					}
				}
				node.sum = ((lanes[0] + lanes[1]) + (lanes[2] + lanes[3])) + ((lanes[4] + lanes[5]) + (lanes[6] + lanes[7]))
				for sampleOffset := alignedCount; sampleOffset < node.count; sampleOffset++ {
					value := samples[node.offset+sampleOffset]
					if computeVariance {
						value -= mean
						value = float32(value * value)
					}
					node.sum += value
				}
			}
			for nodeIndex := len(reductionNodes)/2 - 1; nodeIndex > 0; nodeIndex-- {
				if reductionNodes[nodeIndex].count > whisperReductionLeafSamples {
					reductionNodes[nodeIndex].sum = reductionNodes[nodeIndex*2].sum + reductionNodes[nodeIndex*2+1].sum
				}
			}
			sum += reductionNodes[1].sum
		}
		if computeVariance {
			variance = float32(float64(sum) / float64(len(samples)))
		} else {
			mean = float32(float64(sum) / float64(len(samples)))
		}
	}

	divisor := float32(math.Sqrt(float64(variance) + whisperVarianceEpsilon))
	for index, sample := range samples {
		samples[index] = (sample - mean) / divisor
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

// initHannWindow pre-computes the Hann window of size nFFT.
// Matches numpy: hann(n+1)[:-1] i.e. periodic Hann window.
func (wf *whisperFeatures) initHannWindow() {
	for i := 0; i < whisperNFFT; i++ {
		wf.hannWindow[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(whisperNFFT)))
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
