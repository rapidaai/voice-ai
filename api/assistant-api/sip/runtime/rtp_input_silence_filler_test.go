// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRTPInputSilenceFiller_FillsOnlyContiguousTimestampGaps(t *testing.T) {
	filler := newRTPInputSilenceFiller(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	require.Len(t, filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(1, 0, 1))), 1)

	frames := filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(2, 480, 2)))
	require.Len(t, frames, 3)
	require.Equal(t, bytes.Repeat([]byte{0xff}, 160), frames[0].Audio)
	require.Equal(t, bytes.Repeat([]byte{0xff}, 160), frames[1].Audio)
	require.Equal(t, bytes.Repeat([]byte{2}, 160), frames[2].Audio)
	require.Equal(t, uint64(2), filler.frameCount())

	frames = filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(4, 800, 4)))
	require.Len(t, frames, 1)
	require.Equal(t, bytes.Repeat([]byte{4}, 160), frames[0].Audio)
	require.Equal(t, uint64(2), filler.frameCount())
}

func TestRTPInputSilenceFiller_UsesCodecSilenceAndPacketDuration(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		codec       Codec
		duration    time.Duration
		silenceByte byte
	}{
		{name: "PCMU", codec: CodecPCMU, duration: 20 * time.Millisecond, silenceByte: 0xff},
		{name: "PCMA", codec: CodecPCMA, duration: 30 * time.Millisecond, silenceByte: 0xd5},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			filler := newRTPInputSilenceFiller(&testCase.codec, testCase.duration)
			samples := int(testCase.codec.ClockRate) * int(testCase.duration/time.Millisecond) / 1000
			first := testRTPInputPacket(1, 0, 1)
			first.PayloadType = testCase.codec.PayloadType
			first.Payload = bytes.Repeat([]byte{1}, samples)
			second := testRTPInputPacket(2, uint32(samples*2), 2)
			second.PayloadType = testCase.codec.PayloadType
			second.Payload = bytes.Repeat([]byte{2}, samples)
			filler.process(bufferedInputPackets(time.Now(), first))

			frames := filler.process(bufferedInputPackets(time.Now(), second))
			require.Len(t, frames, 2)
			require.Equal(t, bytes.Repeat([]byte{testCase.silenceByte}, samples), frames[0].Audio)
			require.Equal(t, second.Payload, frames[1].Audio)
		})
	}
}

func TestRTPInputSilenceFiller_DoesNotTreatLossOrDTMFAsSilenceSuppression(t *testing.T) {
	filler := newRTPInputSilenceFiller(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Now()
	filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(1, 0, 1)))

	dtmf := &RTPPacket{SequenceNumber: 2, Timestamp: 160, PayloadType: CodecTelephoneEvent.PayloadType, Payload: []byte{1, 0x80, 0, 160}}
	require.Empty(t, filler.process(bufferedInputPackets(arrivedAt, dtmf)))
	frames := filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(3, 480, 3)))
	require.Len(t, frames, 1)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), frames[0].Audio)
	require.Zero(t, filler.frameCount())
}

func TestRTPInputSilenceFiller_HandlesSequenceAndTimestampWrap(t *testing.T) {
	filler := newRTPInputSilenceFiller(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Now()
	first := testRTPInputPacket(65535, ^uint32(0)-159, 1)
	second := testRTPInputPacket(0, 0, 2)
	require.Len(t, filler.process(bufferedInputPackets(arrivedAt, first)), 1)
	frames := filler.process(bufferedInputPackets(arrivedAt, second))
	require.Len(t, frames, 1)
	require.Equal(t, second.Payload, frames[0].Audio)
}

func TestRTPInputSilenceFiller_ResetClearsTimeline(t *testing.T) {
	filler := newRTPInputSilenceFiller(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Now()
	filler.process(bufferedInputPackets(arrivedAt, testRTPInputPacket(1, 0, 1)))
	filler.reset(&CodecPCMA, 30*time.Millisecond)
	packet := testRTPInputPacket(2, 10000, 2)
	packet.PayloadType = CodecPCMA.PayloadType
	packet.Payload = bytes.Repeat([]byte{2}, 240)
	frames := filler.process(bufferedInputPackets(arrivedAt, packet))
	require.Len(t, frames, 1)
	require.Equal(t, packet.Payload, frames[0].Audio)
}

func bufferedInputPackets(receivedAt time.Time, packets ...*RTPPacket) []rtpBufferedInputPacket {
	buffered := make([]rtpBufferedInputPacket, len(packets))
	for index, packet := range packets {
		buffered[index] = rtpBufferedInputPacket{packet: packet, receivedAt: receivedAt}
	}
	return buffered
}
