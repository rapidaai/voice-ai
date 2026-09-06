// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPInputJitterBuffer_InOrderPacketsEmitImmediately(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	assert.Equal(t, [][]byte{{0x03}}, buffer.push(testRTPInputPacket(3, 320, 0x03)))
}

func TestRTPInputJitterBuffer_ReordersPacketsWithinWindow(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.push(testRTPInputPacket(2, 160, 0x02))

	assert.Equal(t, [][]byte{{0x02}, {0x03}}, out)
}

func TestRTPInputJitterBuffer_DropsDuplicatePackets(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(1, 0, 0x09)))
	assert.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	assert.Equal(t, uint64(1), buffer.droppedPackets())
	assert.Equal(t, uint64(1), buffer.lateOrDuplicatePackets())
	assert.Zero(t, buffer.resyncDroppedPackets())
}

func TestRTPInputJitterBuffer_FillsMissingPacketAfterWindow(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	for sequenceNumber := uint16(3); sequenceNumber <= 6; sequenceNumber++ {
		assert.Empty(t, buffer.push(testRTPInputPacket(sequenceNumber, uint32(sequenceNumber-1)*160, byte(sequenceNumber))))
	}

	out := buffer.push(testRTPInputPacket(7, 960, 0x07))

	require.Len(t, out, 6)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, []byte{0x03}, out[1])
	assert.Equal(t, []byte{0x04}, out[2])
	assert.Equal(t, []byte{0x05}, out[3])
	assert.Equal(t, []byte{0x06}, out[4])
	assert.Equal(t, []byte{0x07}, out[5])
	assert.Equal(t, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_ReorderWindowUsesDuration(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 5*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(18, 17*40, 0x12)))

	out := buffer.push(testRTPInputPacket(19, 18*40, 0x13))

	require.Len(t, out, 1)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_ResyncsLargeSequenceJump(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.push(testRTPInputPacket(1000, 999*160, 0x10))

	assert.Equal(t, [][]byte{{0x10}}, out)
	assert.Equal(t, uint64(0), buffer.lostPackets())
	assert.Equal(t, uint64(1), buffer.droppedPackets())
	assert.Zero(t, buffer.lateOrDuplicatePackets())
	assert.Equal(t, uint64(1), buffer.resyncDroppedPackets())
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 480, 0x04)))
}

func TestRTPInputJitterBuffer_ResyncLimitUsesDuration(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 60*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	out := buffer.push(testRTPInputPacket(12, 11*480, 0x12))

	assert.Equal(t, [][]byte{{0x12}}, out)
	assert.Zero(t, buffer.lostPackets())
	assert.Zero(t, buffer.droppedPackets())
	assert.Zero(t, buffer.resyncDroppedPackets())
}

func TestRTPInputJitterBuffer_FlushOnPlayoutTimeoutReleasesBufferedPacket(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, []byte{0x03}, out[1])
	assert.Equal(t, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_FlushOnPlayoutTimeoutWithoutBufferedPackets(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	assert.Empty(t, buffer.flushOnPlayoutTimeout())
	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.flushOnPlayoutTimeout())
}

func TestRTPInputJitterBuffer_FillsTimestampGapForSilenceSuppression(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))

	out := buffer.push(testRTPInputPacket(2, 640, 0x02))

	require.Len(t, out, 4)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, byte(0xFF), out[1][0])
	assert.Equal(t, byte(0xFF), out[2][0])
	assert.Equal(t, []byte{0x02}, out[3])
	assert.Zero(t, buffer.lostPackets())
	assert.Equal(t, uint64(3), buffer.silenceSuppressionFrameCount())
}

func TestRTPInputJitterBuffer_UsesPCMASilence(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMA, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	out := buffer.push(testRTPInputPacket(2, 640, 0x02))

	require.Len(t, out, 4)
	assert.Equal(t, byte(0xD5), out[0][0])
	assert.Equal(t, []byte{0x02}, out[3])
}

func TestRTPInputJitterBuffer_UsesConfiguredPacketizationForLossConcealment(t *testing.T) {
	tests := []struct {
		name              string
		packetizationTime time.Duration
		timestampDelta    uint32
		silenceLength     int
		silenceByte       byte
		codec             *Codec
	}{
		{
			name:              "10 ms pcmu",
			packetizationTime: 10 * time.Millisecond,
			timestampDelta:    80,
			silenceLength:     80,
			silenceByte:       0xFF,
			codec:             &CodecPCMU,
		},
		{
			name:              "20 ms pcmu",
			packetizationTime: 20 * time.Millisecond,
			timestampDelta:    160,
			silenceLength:     160,
			silenceByte:       0xFF,
			codec:             &CodecPCMU,
		},
		{
			name:              "30 ms pcmu",
			packetizationTime: 30 * time.Millisecond,
			timestampDelta:    240,
			silenceLength:     240,
			silenceByte:       0xFF,
			codec:             &CodecPCMU,
		},
		{
			name:              "10 ms pcma",
			packetizationTime: 10 * time.Millisecond,
			timestampDelta:    80,
			silenceLength:     80,
			silenceByte:       0xD5,
			codec:             &CodecPCMA,
		},
		{
			name:              "20 ms pcma",
			packetizationTime: 20 * time.Millisecond,
			timestampDelta:    160,
			silenceLength:     160,
			silenceByte:       0xD5,
			codec:             &CodecPCMA,
		},
		{
			name:              "30 ms pcma",
			packetizationTime: 30 * time.Millisecond,
			timestampDelta:    240,
			silenceLength:     240,
			silenceByte:       0xD5,
			codec:             &CodecPCMA,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buffer := newRTPInputJitterBuffer(test.codec, test.packetizationTime)

			require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
			assert.Empty(t, buffer.push(testRTPInputPacket(3, test.timestampDelta*2, 0x03)))

			out := buffer.flushOnPlayoutTimeout()

			require.Len(t, out, 2)
			require.Len(t, out[0], test.silenceLength)
			assert.Equal(t, test.silenceByte, out[0][0])
			assert.Equal(t, []byte{0x03}, out[1])
			assert.Equal(t, uint64(1), buffer.lostPackets())
		})
	}
}

func TestRTPInputJitterBuffer_LearnsStablePacketizationFromTimestamps(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(2, 80, 0x02)))
	require.Equal(t, [][]byte{{0x02}, {0x03}}, buffer.push(testRTPInputPacket(3, 160, 0x03)))
	assert.Equal(t, 10*time.Millisecond, buffer.playoutTimeout())

	assert.Empty(t, buffer.push(testRTPInputPacket(5, 320, 0x05)))
	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	require.Len(t, out[0], 80)
	assert.Equal(t, []byte{0x05}, out[1])
}

func TestRTPInputJitterBuffer_UpdatesPacketizationAfterStableChange(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	require.Equal(t, [][]byte{{0x03}}, buffer.push(testRTPInputPacket(3, 320, 0x03)))
	require.Empty(t, buffer.push(testRTPInputPacket(4, 560, 0x04)))
	require.Equal(t, [][]byte{{0x04}, {0x05}}, buffer.push(testRTPInputPacket(5, 800, 0x05)))
	assert.Equal(t, 30*time.Millisecond, buffer.playoutTimeout())

	assert.Empty(t, buffer.push(testRTPInputPacket(7, 1280, 0x07)))
	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	require.Len(t, out[0], 240)
	assert.Equal(t, []byte{0x07}, out[1])
}

func TestRTPInputJitterBuffer_PacketizationChangeDoesNotCreateLoss(t *testing.T) {
	tests := []struct {
		name               string
		nextTimestamp      uint32
		changedTimestamp   uint32
		expectedPacketTime time.Duration
	}{
		{
			name:               "20 ms to 10 ms",
			nextTimestamp:      400,
			changedTimestamp:   480,
			expectedPacketTime: 10 * time.Millisecond,
		},
		{
			name:               "20 ms to 30 ms",
			nextTimestamp:      560,
			changedTimestamp:   800,
			expectedPacketTime: 30 * time.Millisecond,
		},
		{
			name:               "20 ms to 40 ms",
			nextTimestamp:      640,
			changedTimestamp:   960,
			expectedPacketTime: 40 * time.Millisecond,
		},
		{
			name:               "20 ms to 60 ms",
			nextTimestamp:      800,
			changedTimestamp:   1280,
			expectedPacketTime: 60 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)

			require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
			require.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
			require.Equal(t, [][]byte{{0x03}}, buffer.push(testRTPInputPacket(3, 320, 0x03)))
			require.Empty(t, buffer.push(testRTPInputPacket(4, test.nextTimestamp, 0x04)))
			require.Equal(t, [][]byte{{0x04}, {0x05}}, buffer.push(testRTPInputPacket(5, test.changedTimestamp, 0x05)))

			assert.Equal(t, test.expectedPacketTime, buffer.playoutTimeout())
			assert.Zero(t, buffer.lostPackets())
			assert.Zero(t, buffer.silenceSuppressionFrameCount())
		})
	}
}

func TestRTPInputJitterBuffer_OneTimeTimestampJumpIsSilenceSuppression(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(2, 480, 0x02)))
	out := buffer.push(testRTPInputPacket(3, 640, 0x03))

	require.Len(t, out, 4)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, byte(0xFF), out[1][0])
	assert.Equal(t, []byte{0x02}, out[2])
	assert.Equal(t, []byte{0x03}, out[3])
	assert.Equal(t, 20*time.Millisecond, buffer.playoutTimeout())
	assert.Zero(t, buffer.lostPackets())
	assert.Equal(t, uint64(2), buffer.silenceSuppressionFrameCount())
}

func TestRTPInputJitterBuffer_ReorderedPacketsDoNotCorruptPacketizationLearning(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))
	require.Equal(t, [][]byte{{0x02}, {0x03}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	assert.Equal(t, 20*time.Millisecond, buffer.playoutTimeout())

	assert.Empty(t, buffer.push(testRTPInputPacket(5, 640, 0x05)))
	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	require.Len(t, out[0], 160)
	assert.Equal(t, []byte{0x05}, out[1])
}

func TestRTPInputJitterBuffer_ResyncPreservesActivePacketization(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, 30*time.Millisecond)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(3, 480, 0x03)))
	require.Equal(t, [][]byte{{0x10}}, buffer.push(testRTPInputPacket(1000, 240000, 0x10)))

	assert.Empty(t, buffer.push(testRTPInputPacket(1002, 240480, 0x12)))
	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	require.Len(t, out[0], 240)
	assert.Equal(t, []byte{0x12}, out[1])
}

func TestRTPInputJitterBuffer_HandlesSequenceWrap(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(65535, 0, 0x01)))
	assert.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(0, 160, 0x02)))
	assert.Equal(t, [][]byte{{0x03}}, buffer.push(testRTPInputPacket(1, 320, 0x03)))
}

func testRTPInputPacket(sequenceNumber uint16, timestamp uint32, payload byte) *RTPPacket {
	return &RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: sequenceNumber,
		Timestamp:      timestamp,
		Payload:        []byte{payload},
	}
}
