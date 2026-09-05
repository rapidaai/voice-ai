// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPInputJitterBuffer_InOrderPacketsEmitImmediately(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	assert.Equal(t, [][]byte{{0x03}}, buffer.push(testRTPInputPacket(3, 320, 0x03)))
}

func TestRTPInputJitterBuffer_ReordersPacketsWithinWindow(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.push(testRTPInputPacket(2, 160, 0x02))

	assert.Equal(t, [][]byte{{0x02}, {0x03}}, out)
}

func TestRTPInputJitterBuffer_DropsDuplicatePackets(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	assert.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.push(testRTPInputPacket(1, 0, 0x09)))
	assert.Equal(t, [][]byte{{0x02}}, buffer.push(testRTPInputPacket(2, 160, 0x02)))
	assert.Equal(t, uint64(1), buffer.droppedPackets())
}

func TestRTPInputJitterBuffer_FillsMissingPacketAfterWindow(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

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

func TestRTPInputJitterBuffer_ResyncsLargeSequenceJump(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.push(testRTPInputPacket(1000, 999*160, 0x10))

	assert.Equal(t, [][]byte{{0x10}}, out)
	assert.Equal(t, uint64(0), buffer.lostPackets())
	assert.Equal(t, uint64(1), buffer.droppedPackets())
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 480, 0x04)))
}

func TestRTPInputJitterBuffer_FlushOnPlayoutTimeoutReleasesBufferedPacket(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Empty(t, buffer.push(testRTPInputPacket(3, 320, 0x03)))

	out := buffer.flushOnPlayoutTimeout()

	require.Len(t, out, 2)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, []byte{0x03}, out[1])
	assert.Equal(t, uint64(1), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_FlushOnPlayoutTimeoutWithoutBufferedPackets(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	assert.Empty(t, buffer.flushOnPlayoutTimeout())
	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	assert.Empty(t, buffer.flushOnPlayoutTimeout())
}

func TestRTPInputJitterBuffer_FillsTimestampGapForSilenceSuppression(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))

	out := buffer.push(testRTPInputPacket(2, 480, 0x02))

	require.Len(t, out, 3)
	assert.Equal(t, byte(0xFF), out[0][0])
	assert.Equal(t, byte(0xFF), out[1][0])
	assert.Equal(t, []byte{0x02}, out[2])
	assert.Equal(t, uint64(2), buffer.lostPackets())
}

func TestRTPInputJitterBuffer_UsesPCMASilence(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMA)

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	out := buffer.push(testRTPInputPacket(2, 320, 0x02))

	require.Len(t, out, 2)
	assert.Equal(t, byte(0xD5), out[0][0])
	assert.Equal(t, []byte{0x02}, out[1])
}

func TestRTPInputJitterBuffer_HandlesSequenceWrap(t *testing.T) {
	buffer := newRTPInputJitterBuffer(&CodecPCMU)

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
