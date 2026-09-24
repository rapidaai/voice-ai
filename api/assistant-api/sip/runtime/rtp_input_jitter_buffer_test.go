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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPInputJitterBuffer_InOrderPacketsEmitImmediately(t *testing.T) {
	buffer := newRTPInputJitterBuffer(rtpDefaultPacketizationTime)
	arrivedAt := time.Unix(1, 0)
	for sequence := uint16(1); sequence <= 3; sequence++ {
		packet := testRTPInputPacket(sequence, uint32(sequence-1)*160, byte(sequence))
		packets := buffer.push(packet, arrivedAt)
		require.Len(t, packets, 1)
		require.Equal(t, packet.Payload, packets[0].packet.Payload)
	}
	require.Zero(t, buffer.lostPackets())
	require.True(t, buffer.nextDeadline().IsZero())
}

func TestRTPInputJitterBuffer_ReordersUntilPacketAgeDeadline(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt))
	require.Equal(t, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	require.Empty(t, buffer.flushExpired(arrivedAt.Add(20*time.Millisecond)))

	packets := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(rtpInputReorderWindow-time.Nanosecond))
	require.Len(t, packets, 2)
	require.Equal(t, bytes.Repeat([]byte{2}, 160), packets[0].packet.Payload)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), packets[1].packet.Payload)
	require.Zero(t, buffer.lostPackets())
	require.True(t, buffer.nextDeadline().IsZero())
}

func TestRTPInputJitterBuffer_LossExpiresWithoutSynthesizingAudio(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(4, 480, 4), arrivedAt)

	packets := buffer.flushExpired(arrivedAt.Add(rtpInputReorderWindow))
	require.Len(t, packets, 1)
	require.Equal(t, bytes.Repeat([]byte{4}, 160), packets[0].packet.Payload)
	require.Equal(t, uint64(2), buffer.lostPackets())
	require.Empty(t, buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(time.Second)))
	require.Equal(t, uint64(1), buffer.lateOrDuplicatePackets())
}

func TestRTPInputJitterBuffer_FlushPendingEmitsBufferedTail(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	require.Len(t, buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt), 1)
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt))

	packets := buffer.flushPending()
	require.Len(t, packets, 1)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), packets[0].packet.Payload)
	require.Equal(t, uint64(1), buffer.lostPackets())
	require.Empty(t, buffer.bufferedPackets)
}

func TestRTPInputJitterBuffer_NewTrafficDoesNotPostponeExpiry(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	for sequence := uint16(3); sequence <= 10; sequence++ {
		require.Empty(t, buffer.push(testRTPInputPacket(sequence, uint32(sequence-1)*160, byte(sequence)), arrivedAt.Add(time.Duration(sequence-3)*10*time.Millisecond)))
		require.Equal(t, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	}
	packets := buffer.push(testRTPInputPacket(11, 1600, 11), arrivedAt.Add(rtpInputReorderWindow))
	require.Len(t, packets, 9)
	for index, packet := range packets {
		require.Equal(t, bytes.Repeat([]byte{byte(index + 3)}, 160), packet.packet.Payload)
	}
	require.Equal(t, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_DropsDuplicatesWithoutChangingDeadline(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt)
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 9), arrivedAt.Add(60*time.Millisecond)))
	require.Empty(t, buffer.push(testRTPInputPacket(1, 0, 9), arrivedAt.Add(60*time.Millisecond)))
	require.Equal(t, arrivedAt.Add(rtpInputReorderWindow), buffer.nextDeadline())
	packets := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt.Add(70*time.Millisecond))
	require.Equal(t, bytes.Repeat([]byte{3}, 160), packets[1].packet.Payload)
	require.Equal(t, uint64(2), buffer.droppedPackets())
	require.Equal(t, uint64(2), buffer.lateOrDuplicatePackets())
}

func TestRTPInputJitterBuffer_OrdersAudioAndTelephoneEvents(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Empty(t, buffer.push(testRTPInputPacket(4, 1600, 4), arrivedAt))
	second := &RTPPacket{SequenceNumber: 2, Timestamp: 160, PayloadType: CodecTelephoneEvent.PayloadType, Payload: []byte{1, 0x80, 0, 160}}
	third := &RTPPacket{SequenceNumber: 3, Timestamp: 160, PayloadType: CodecTelephoneEvent.PayloadType, Payload: []byte{1, 0x80, 0, 160}}
	require.Len(t, buffer.push(second, arrivedAt.Add(30*time.Millisecond)), 1)
	packets := buffer.push(third, arrivedAt.Add(30*time.Millisecond))
	require.Len(t, packets, 2)
	require.Equal(t, third.Payload, packets[0].packet.Payload)
	require.Equal(t, bytes.Repeat([]byte{4}, 160), packets[1].packet.Payload)
	require.Zero(t, buffer.lostPackets())
}

func TestRTPInputJitterBuffer_ResyncBoundsBuffer(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(3, 320, 3), arrivedAt)
	packet := testRTPInputPacket(1000, 999*160, 10)
	packets := buffer.push(packet, arrivedAt)
	require.Len(t, packets, 1)
	require.Equal(t, packet.Payload, packets[0].packet.Payload)
	require.Equal(t, uint64(1), buffer.resyncDroppedPackets())
	require.Zero(t, buffer.lostPackets())
	require.Empty(t, buffer.bufferedPackets)
}

func TestRTPInputJitterBuffer_ResetDiscardsPendingStream(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(100, 0, 1), arrivedAt)
	buffer.push(testRTPInputPacket(102, 320, 3), arrivedAt)
	buffer.reset(30 * time.Millisecond)
	packet := testRTPInputPacket(1, 200000, 4)
	packets := buffer.push(packet, arrivedAt)
	require.Len(t, packets, 1)
	require.Equal(t, packet.Payload, packets[0].packet.Payload)
	require.Equal(t, uint64(1), buffer.resyncDroppedPackets())
}

func TestRTPInputJitterBuffer_SequenceWrap(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	first := testRTPInputPacket(65535, 0, 1)
	require.Len(t, buffer.push(first, arrivedAt), 1)
	require.Empty(t, buffer.push(testRTPInputPacket(1, 320, 3), arrivedAt))
	packets := buffer.push(testRTPInputPacket(0, 160, 2), arrivedAt)
	require.Len(t, packets, 2)
	require.Equal(t, bytes.Repeat([]byte{2}, 160), packets[0].packet.Payload)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), packets[1].packet.Payload)
	require.Zero(t, buffer.lostPackets())
}

func TestRTPInputJitterBuffer_OwnsBufferedPayload(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(1, 0)
	buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	third := testRTPInputPacket(3, 320, 3)
	buffer.push(third, arrivedAt)
	third.Payload[0] = 99
	packets := buffer.push(testRTPInputPacket(2, 160, 2), arrivedAt)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), packets[1].packet.Payload)
}

func TestRTPInputJitterBuffer_PreservesPacketArrivalTime(t *testing.T) {
	buffer := newRTPInputJitterBuffer(20 * time.Millisecond)
	arrivedAt := time.Unix(123, 456)
	packets := buffer.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	require.Len(t, packets, 1)
	require.Equal(t, arrivedAt, packets[0].receivedAt)
}

func TestRTPInputJitterBuffer_InvalidSetupAndEmptyInput(t *testing.T) {
	for _, duration := range []time.Duration{0, time.Millisecond, 100 * time.Millisecond, 5500 * time.Microsecond} {
		buffer := newRTPInputJitterBuffer(duration)
		assert.Equal(t, rtpDefaultPacketizationTime, buffer.packetizationTime)
		assert.Empty(t, buffer.push(nil, time.Now()))
		assert.Empty(t, buffer.push(&RTPPacket{}, time.Now()))
		assert.Empty(t, buffer.flushExpired(time.Now()))
	}
}

func testRTPInputPacket(sequenceNumber uint16, timestamp uint32, payload byte) *RTPPacket {
	return &RTPPacket{Version: rtpVersion, PayloadType: CodecPCMU.PayloadType, SequenceNumber: sequenceNumber, Timestamp: timestamp, Payload: bytes.Repeat([]byte{payload}, 160)}
}
