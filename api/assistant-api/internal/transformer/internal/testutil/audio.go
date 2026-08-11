// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transformer_testutil

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

const (
	// SampleRate is the standard audio sample rate for voice AI (16 kHz).
	SampleRate = 16000

	// BytesPerSample is the number of bytes per sample for LINEAR16 encoding.
	BytesPerSample = 2

	// FrameDuration is the standard frame duration in milliseconds.
	FrameDuration = 20

	// FrameSize is the number of bytes per 20ms frame at 16 kHz mono LINEAR16.
	FrameSize = SampleRate * BytesPerSample * FrameDuration / 1000 // 640
)

type speechToTextAudioTransformer interface {
	Transform(ctx context.Context, in internal_type.Packet) error
}

// SilentPCM generates silence (all zeros) of the specified duration in seconds.
// Returns raw LINEAR16, 16 kHz, mono PCM bytes.
func SilentPCM(durationSec float64) []byte {
	numSamples := int(float64(SampleRate) * durationSec)
	return make([]byte, numSamples*BytesPerSample)
}

// SineTonePCM generates a pure sine wave at the given frequency and duration.
// The amplitude is set to ~50% of int16 max to avoid clipping.
// Returns raw LINEAR16, 16 kHz, mono PCM bytes.
func SineTonePCM(freqHz float64, durationSec float64) []byte {
	numSamples := int(float64(SampleRate) * durationSec)
	buf := make([]byte, numSamples*BytesPerSample)
	amplitude := float64(math.MaxInt16) * 0.5

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(SampleRate)
		sample := int16(amplitude * math.Sin(2*math.Pi*freqHz*t))
		binary.LittleEndian.PutUint16(buf[i*BytesPerSample:], uint16(sample))
	}
	return buf
}

// LoadSpeechPCM loads a raw PCM file from testdata/.
// Skips the test if the file doesn't exist.
func LoadSpeechPCM(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join(testdataDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("speech fixture %q not found: %v (generate with: say \"Hello world\" -o /tmp/hello_world.aiff && ffmpeg -i /tmp/hello_world.aiff -ar 16000 -ac 1 -f s16le testdata/hello_world.pcm)", filename, err)
	}
	return data
}

// ChunkAudio splits raw PCM data into chunks of the given size in bytes.
// The last chunk may be smaller than chunkSize.
func ChunkAudio(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}

// FeedAudio sends chunked PCM audio into the STT transformer at real-time pace.
// After all speech audio is sent, it appends 1 second of silence so the STT
// provider's endpointing logic detects the utterance boundary and emits a final
// transcript.
func FeedAudio(ctx context.Context, t *testing.T, stt speechToTextAudioTransformer, audio []byte) {
	t.Helper()
	// Append trailing silence so the provider can finalize the utterance.
	audioWithSilence := make([]byte, len(audio)+SampleRate*BytesPerSample) // +1s silence
	copy(audioWithSilence, audio)

	chunks := ChunkAudio(audioWithSilence, FrameSize)
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := stt.Transform(ctx, internal_type.SpeechToTextAudioPacket{
			ContextID: "test-stt",
			Audio:     chunk,
		})
		if err != nil {
			t.Logf("Transform error (may be expected after close): %v", err)
			return
		}
		time.Sleep(time.Duration(FrameDuration) * time.Millisecond)
	}
}
