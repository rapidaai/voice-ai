// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_inworld

import "encoding/binary"

// pcmFromStreamChunk returns raw PCM for one decoded base64 payload.
// Inworld's voice:stream endpoint returns LINEAR16 audio wrapped in a
// minimal RIFF/WAVE container on every NDJSON line — concatenating those
// chunks verbatim embeds a WAV header every few milliseconds and
// produces audible clicks downstream. When a RIFF wrapper is detected
// we return only the `data` subchunk payload; bare PCM bytes pass
// through. Either path is trimmed to a whole sample pair so Rapida's
// LINEAR16 @ 16 kHz consumers never see half a sample.
//
// The fix mirrors what Inworld's own examples do (e.g. tts_cli.py
// strips `chunk[:4]==b'RIFF'` → `chunk[44:]` per chunk), but we walk
// the RIFF chunk list properly instead of assuming a fixed 44-byte
// header — `JUNK`/`LIST` padding chunks exist in the wild.
func pcmFromStreamChunk(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	if isWAV(b) {
		pcm := wavDataSubchunk(b)
		if len(pcm) == 0 {
			return nil
		}
		return alignPCM16(pcm)
	}
	return alignPCM16(b)
}

// isWAV reports whether b starts with a RIFF/WAVE signature. Minimum
// 12 bytes: "RIFF" + 4-byte size + "WAVE".
func isWAV(b []byte) bool {
	return len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WAVE"
}

// wavDataSubchunk walks RIFF chunks after the WAVE header and returns
// the payload of the first `data` chunk. Unknown chunks (`fmt `,
// `JUNK`, `LIST`, etc.) are skipped per the RIFF spec. Returns nil if
// a chunk size is truncated or no `data` chunk is present.
func wavDataSubchunk(wav []byte) []byte {
	off := 12
	for off+8 <= len(wav) {
		id := string(wav[off : off+4])
		sz := int(binary.LittleEndian.Uint32(wav[off+4 : off+8]))
		off += 8
		if sz < 0 || off+sz > len(wav) {
			return nil
		}
		payload := wav[off : off+sz]
		off += sz
		if sz%2 == 1 {
			off++ // RIFF chunks are word-aligned
		}
		if id == "data" {
			return payload
		}
	}
	return nil
}

// alignPCM16 drops a trailing odd byte so concatenated chunks never
// split a 16-bit sample across the boundary. No-op when len is even.
func alignPCM16(b []byte) []byte {
	if len(b)%2 != 0 {
		return b[:len(b)-1]
	}
	return b
}
