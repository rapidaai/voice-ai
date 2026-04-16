// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPCMFromStreamChunk_RawPCM(t *testing.T) {
	raw := []byte{0x01, 0x00, 0xff, 0x7f} // 2 samples LE
	out := pcmFromStreamChunk(raw)
	assert.Equal(t, raw, out)
}

func TestPCMFromStreamChunk_OddRawTrims(t *testing.T) {
	raw := []byte{0x01, 0x00, 0x02}
	out := pcmFromStreamChunk(raw)
	assert.Equal(t, []byte{0x01, 0x00}, out)
}

func TestPCMFromStreamChunk_EmptyPasses(t *testing.T) {
	assert.Empty(t, pcmFromStreamChunk(nil))
	assert.Empty(t, pcmFromStreamChunk([]byte{}))
}

func TestPCMFromStreamChunk_WAVExtractsDataOnly(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0x02, 0x00, 0x03, 0x00}
	wav := buildMinimalWAV(t, pcm)
	out := pcmFromStreamChunk(wav)
	assert.Equal(t, pcm, out)
}

func TestPCMFromStreamChunk_WAVWithOddDataTrims(t *testing.T) {
	// Odd-length data should be trimmed to the nearest whole sample.
	pcm := []byte{0x01, 0x00, 0x02, 0x00, 0x03}
	wav := buildMinimalWAV(t, pcm)
	out := pcmFromStreamChunk(wav)
	assert.Equal(t, []byte{0x01, 0x00, 0x02, 0x00}, out)
}

func TestPCMFromStreamChunk_WAVNoDataSubchunkReturnsNil(t *testing.T) {
	// RIFF/WAVE header with only a JUNK chunk — no `data` subchunk.
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WAVE")
	buf.WriteString("JUNK")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4))
	buf.WriteString("xxxx")
	b := buf.Bytes()
	binary.LittleEndian.PutUint32(b[4:8], uint32(len(b)-8))
	assert.Nil(t, pcmFromStreamChunk(b))
}

func TestWAVDataSubchunk_IgnoresJUNKBeforeData(t *testing.T) {
	pcm := []byte{0xab, 0xcd}
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	// Placeholder size — fixed below
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WAVE")
	// JUNK chunk (4 bytes payload)
	buf.WriteString("JUNK")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4))
	buf.WriteString("xxxx")
	// data chunk
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	b := buf.Bytes()
	binary.LittleEndian.PutUint32(b[4:8], uint32(len(b)-8))

	got := wavDataSubchunk(b)
	require.Equal(t, pcm, got)
}

func TestWAVDataSubchunk_TruncatedSizeReturnsNil(t *testing.T) {
	// RIFF/WAVE header claims a data chunk larger than the buffer.
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WAVE")
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(999))
	buf.Write([]byte{0x01, 0x02}) // only 2 bytes of promised 999
	assert.Nil(t, wavDataSubchunk(buf.Bytes()))
}

func TestIsWAV(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want bool
	}{
		{"empty", nil, false},
		{"too short", []byte("RIFF\x00\x00\x00\x00WAV"), false},
		{"riff wave", []byte("RIFF\x00\x00\x00\x00WAVE"), true},
		{"riff rmp3", []byte("RIFF\x00\x00\x00\x00RMP3"), false},
		{"not riff", []byte("fLaC\x00\x00\x00\x00WAVE"), false},
		{"bare pcm", []byte{0x01, 0x00, 0x02, 0x00}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isWAV(tt.b))
		})
	}
}

// buildMinimalWAV returns RIFF/WAVE with fmt + data only (mono PCM16 16 kHz).
func buildMinimalWAV(t *testing.T, pcm []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0}) // patch later
	buf.WriteString("WAVE")
	// fmt
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16)) // PCM fmt size
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // mono
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(32000)) // byte rate
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))     // block align
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))    // bits
	// data
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	out := buf.Bytes()
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}
