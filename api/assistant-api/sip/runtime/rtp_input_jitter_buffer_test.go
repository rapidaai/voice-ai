// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPInputJitterBuffer_InOrderPacketsEmitImmediately(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)
	arrivedAt := time.Unix(1, 0)
	for sequence := uint16(1); sequence <= 3; sequence++ {
		packet := testRTPInputPacket(sequence, uint32(sequence-1)*160, byte(sequence))
		frames := buffer.push(packet, arrivedAt)
		require.Len(tester, frames, 1)
		require.Equal(tester, packet.Payload, frames[0].Audio)
	}
	require.Zero(tester, buffer.lostPackets())
	require.True(tester, buffer.nextDeadline().IsZero())
	require.Empty(tester, buffer.flushExpired(arrivedAt.Add(time.Hour)))
}

func TestRTPInputJitterBuffer_ReordersUntilPacketAgeDeadline(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Empty(tester, buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt))
	require.Equal(tester, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	require.Empty(tester, buffer.flushExpired(arrivedAt.Add(20*time.Millisecond)))
	out := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(rtpInputReorderWindow-time.Nanosecond))
	require.Len(tester, out, 2)
	require.Equal(tester, bytes.Repeat([]byte{2}, 160), out[0].Audio)
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), out[1].Audio)
	require.Zero(tester, buffer.lostPackets())
	require.True(tester, buffer.nextDeadline().IsZero())
}

func TestRTPInputJitterBuffer_LossExpiresWithoutFurtherPackets(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(4, 480, 4), arrivedAt)
	out := buffer.flushExpired(arrivedAt.Add(rtpInputReorderWindow))
	require.Len(tester, out, 3)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 160), out[0].Audio)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 160), out[1].Audio)
	require.Equal(tester, bytes.Repeat([]byte{4}, 160), out[2].Audio)
	require.Equal(tester, uint64(2), buffer.lostPackets())
	require.Zero(tester, buffer.silenceSuppressionFrameCount())
	require.Empty(tester, buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(time.Second)))
	require.Equal(tester, uint64(1), buffer.lateOrDuplicatePackets())
}

func TestRTPInputJitterBuffer_FlushPendingEmitsBufferedTail(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	frames := buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Len(tester, frames, 1)
	require.Equal(tester, bytes.Repeat([]byte{1}, 160), frames[0].Audio)
	require.Empty(tester, buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt))

	out := buffer.flushPending()

	require.Len(tester, out, 2)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 160), out[0].Audio)
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), out[1].Audio)
	require.Equal(tester, uint64(1), buffer.lostPackets())
	require.Empty(tester, buffer.bufferedPackets)
}

func TestRTPInputJitterBuffer_NewTrafficDoesNotPostponeExpiry(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	for sequence := uint16(3); sequence <= 10; sequence++ {
		require.Empty(tester, buffer.push(testRTPInputPacket(sequence, uint32(sequence-1)*160, byte(sequence)), arrivedAt.Add(time.Duration(sequence-3)*10*time.Millisecond)))
		require.Equal(tester, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	}
	out := buffer.push(testRTPInputPacket(11, 1600, 11), arrivedAt.Add(rtpInputReorderWindow))
	require.Len(tester, out, 10)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 160), out[0].Audio)
	for index, payload := range out[1:] {
		require.Equal(tester, bytes.Repeat([]byte{byte(index + 3)}, 160), payload.Audio)
	}
	require.Equal(tester, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_DropsDuplicatesWithoutChangingDeadline(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt)
	require.Empty(tester, buffer.push(testRTPInputPacket(3, 320, 9), arrivedAt.Add(60*time.Millisecond)))
	require.Empty(tester, buffer.push(testRTPInputPacket(1, 0, 9), arrivedAt.Add(60*time.Millisecond)))
	require.Equal(tester, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	out := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(70*time.Millisecond))
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), out[1].Audio)
	require.Equal(tester, uint64(2), buffer.droppedPackets())
	require.Equal(tester, uint64(2), buffer.lateOrDuplicatePackets())
}

func TestRTPInputJitterBuffer_G711DurationsAndCodecSilence(tester *testing.T) {
	for _, codec := range []Codec{CodecPCMU, CodecPCMA} {
		for _, duration := range []int{5, 10, 20, 30, 40, 60} {
			tester.Run(fmt.Sprintf("%s/%dms", codec.Name, duration), func(tester *testing.T) {
				buffer := newRTPInputJitterBuffer(&codec, time.Duration(duration)*time.Millisecond)
				arrivedAt := time.Unix(1, 0)
				samples := duration * 8
				first := testRTPInputPacket(1, 0, 1)
				first.PayloadType = codec.PayloadType
				first.Payload = bytes.Repeat([]byte{1}, samples)
				frames := buffer.push(first, arrivedAt)
				require.Len(tester, frames, 1)
				require.Equal(tester, first.Payload, frames[0].Audio)
				third := testRTPInputPacket(3, uint32(samples*2), 3)
				third.PayloadType = codec.PayloadType
				third.Payload = bytes.Repeat([]byte{3}, samples)
				require.Empty(tester, buffer.push(third, arrivedAt))
				out := buffer.flushExpired(arrivedAt.Add(rtpInputReorderWindow))
				require.Len(tester, out, 2)
				silence := byte(0xff)
				if codec.Name == CodecPCMA.Name {
					silence = 0xd5
				}
				require.Equal(tester, bytes.Repeat([]byte{silence}, samples), out[0].Audio)
				require.Equal(tester, third.Payload, out[1].Audio)
				require.Equal(tester, uint64(1), buffer.lostPackets())
			})
		}
	}
}

func TestRTPInputJitterBuffer_PayloadDurationChangesDoNotInsertSilence(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	var timestamp uint32
	for index, duration := range []int{20, 30, 5, 60, 10, 40, 20} {
		packet := testRTPInputPacket(uint16(index+1), timestamp, byte(index+1))
		packet.Payload = bytes.Repeat([]byte{byte(index + 1)}, duration*8)
		frames := buffer.push(packet, arrivedAt)
		require.Len(tester, frames, 1)
		require.Equal(tester, packet.Payload, frames[0].Audio)
		timestamp += uint32(len(packet.Payload))
	}
	require.Zero(tester, buffer.silenceSuppressionFrameCount())
	require.Zero(tester, buffer.lostPackets())
}

func TestRTPInputJitterBuffer_TimestampGapsPreserveExactDuration(tester *testing.T) {
	for _, gap := range []uint32{80, 160, 240, 480, 4000, 8000} {
		tester.Run(fmt.Sprint(gap), func(tester *testing.T) {
			buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
			arrivedAt := time.Unix(1, 0)
			buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
			out := buffer.push(testRTPInputPacket(2, 160+gap, 2), arrivedAt)
			joined := make([]byte, 0)
			for _, frame := range out {
				joined = append(joined, frame.Audio...)
			}
			expected := bytes.Repeat([]byte{2}, 160)
			if gap <= 4000 {
				expected = append(bytes.Repeat([]byte{0xff}, int(gap)), expected...)
			}
			require.Equal(tester, expected, joined)
			require.Zero(tester, buffer.lostPackets())
		})
	}
}

func TestRTPInputJitterBuffer_OrdersTelephoneEventsBeforeAudioRouting(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Empty(tester, buffer.push(testRTPInputPacket(4, 1600, 4), arrivedAt))
	for sequence := uint16(2); sequence <= 3; sequence++ {
		packet := &RTPPacket{SequenceNumber: sequence, Timestamp: 160, PayloadType: CodecTelephoneEvent.PayloadType, Payload: []byte{1, 0x80, 0, 160}}
		out := buffer.push(packet, arrivedAt.Add(30*time.Millisecond))
		if sequence == 3 {
			require.Len(tester, out, 1)
			require.Equal(tester, bytes.Repeat([]byte{4}, 160), out[0].Audio)
		} else {
			require.Empty(tester, out)
		}
	}
	require.Zero(tester, buffer.lostPackets())
	require.Zero(tester, buffer.silenceSuppressionFrameCount())
}

func TestRTPInputJitterBuffer_LossUsesTimestampGapRatherThanPacketCount(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(5, 240, 5), arrivedAt)
	out := buffer.flushExpired(arrivedAt.Add(rtpInputReorderWindow))
	require.Len(tester, out, 2)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 80), out[0].Audio)
	require.Equal(tester, bytes.Repeat([]byte{5}, 160), out[1].Audio)
	require.Equal(tester, uint64(3), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_ResyncBoundsBufferAndSilence(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt)
	packet := testRTPInputPacket(1000, 999*160, 10)
	frames := buffer.push(packet, arrivedAt)
	require.Len(tester, frames, 1)
	require.Equal(tester, packet.Payload, frames[0].Audio)
	require.Equal(tester, uint64(1), buffer.resyncDroppedPackets())
	require.Zero(tester, buffer.lostPackets())
	require.Empty(tester, buffer.bufferedPackets)
	require.Empty(tester, buffer.push(testRTPInputPacket(4, 480, 4), arrivedAt))
}

func TestRTPInputJitterBuffer_ResetDiscardsPendingStream(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(100, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(102, 320, 3), arrivedAt)
	buffer.reset(&CodecPCMA, 30*time.Millisecond)
	packet := testRTPInputPacket(1, 200000, 4)
	packet.PayloadType = CodecPCMA.PayloadType
	packet.Payload = bytes.Repeat([]byte{4}, 240)
	frames := buffer.push(packet, arrivedAt)
	require.Len(tester, frames, 1)
	require.Equal(tester, packet.Payload, frames[0].Audio)
	require.Equal(tester, uint64(1), buffer.resyncDroppedPackets())
	require.Empty(tester, buffer.flushExpired(arrivedAt.Add(time.Second)))
}

func TestRTPInputJitterBuffer_SequenceAndTimestampWrap(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	first := testRTPInputPacket(65535, ^uint32(0)-159, 1)
	frames := buffer.push(first, arrivedAt)
	require.Len(tester, frames, 1)
	require.Equal(tester, first.Payload, frames[0].Audio)
	require.Empty(tester, buffer.push(testRTPInputPacket(1, 160, 3), arrivedAt))
	out := buffer.push(testRTPInputPacket(0, 0, 2), arrivedAt)
	require.Len(tester, out, 2)
	require.Equal(tester, bytes.Repeat([]byte{2}, 160), out[0].Audio)
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), out[1].Audio)
	require.Zero(tester, buffer.lostPackets())
	require.Zero(tester, buffer.silenceSuppressionFrameCount())
}

func TestRTPInputJitterBuffer_OwnsBufferedPayload(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	third := testRTPInputPacket(3, 320, 3)
	buffer.push(third, arrivedAt)
	third.Payload[0] = 99
	out := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt)
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), out[1].Audio)
}

func TestRTPInputJitterBuffer_PreservesPacketArrivalTime(tester *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	arrivedAt := time.Unix(123, 456)

	frames := buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)

	require.Len(tester, frames, 1)
	require.Equal(tester, arrivedAt, frames[0].ReceivedAt)
}

func TestRTPInputJitterBuffer_InvalidSetupAndEmptyInput(tester *testing.T) {
	for _, duration := range []time.Duration{0, time.Millisecond, 100 * time.Millisecond, 5500 * time.Microsecond} {
		buffer := newRTPInputJitterBuffer(nil, duration)
		assert.Len(tester, buffer.silencePayload, 160)
		assert.Empty(tester, buffer.push(nil, time.Now()))
		assert.Empty(tester, buffer.push(&RTPPacket{}, time.Now()))
		assert.Empty(tester, buffer.flushExpired(time.Now()))
	}
	buffer := newRTPInputJitterBuffer(&Codec{}, 20*time.Millisecond)
	assert.Len(tester, buffer.silencePayload, 160)
}

func testRTPInputPacket(sequenceNumber uint16, timestamp uint32, payload byte) *RTPPacket {
	return &RTPPacket{Version: rtpVersion, PayloadType: CodecPCMU.PayloadType, SequenceNumber: sequenceNumber, Timestamp: timestamp, Payload: bytes.Repeat([]byte{payload}, 160)}
}
