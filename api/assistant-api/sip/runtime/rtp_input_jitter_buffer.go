// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"sync"
	"sync/atomic"
)

const (
	rtpInputReorderWindowFrames = 4
	rtpInputMaxLossGapFrames    = 25
	rtpInputMaxSilenceGapFrames = 25
)

type rtpInputJitterBuffer struct {
	mu sync.Mutex

	started            bool
	expectedSequence   uint16
	expectedTimestamp  uint32
	rtpSamplesPerFrame uint32
	silencePayload     []byte
	bufferedPackets    map[uint16]*RTPPacket

	packetsLost    atomic.Uint64
	packetsDropped atomic.Uint64
}

func newRTPInputJitterBuffer(codec *Codec) *rtpInputJitterBuffer {
	buffer := &rtpInputJitterBuffer{}
	buffer.reset(codec)
	return buffer
}

func (buffer *rtpInputJitterBuffer) reset(codec *Codec) {
	if codec == nil {
		codec = &CodecPCMU
	}
	samplesPerFrame := codec.ClockRate * 20 / 1000
	if samplesPerFrame == 0 {
		samplesPerFrame = CodecPCMU.ClockRate * 20 / 1000
	}
	silenceByte := byte(0xFF)
	if codec.Name == CodecPCMA.Name {
		silenceByte = 0xD5
	}
	silencePayload := make([]byte, int(samplesPerFrame))
	for i := range silencePayload {
		silencePayload[i] = silenceByte
	}

	buffer.mu.Lock()
	buffer.started = false
	buffer.expectedSequence = 0
	buffer.expectedTimestamp = 0
	buffer.rtpSamplesPerFrame = samplesPerFrame
	buffer.silencePayload = silencePayload
	buffer.bufferedPackets = make(map[uint16]*RTPPacket, rtpInputReorderWindowFrames+1)
	buffer.mu.Unlock()
}

func (buffer *rtpInputJitterBuffer) push(packet *RTPPacket) [][]byte {
	if packet == nil || len(packet.Payload) == 0 {
		return nil
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	if !buffer.started {
		buffer.started = true
		buffer.expectedSequence = packet.SequenceNumber
		buffer.expectedTimestamp = packet.Timestamp
		return buffer.emitPacket(packet)
	}

	sequenceDistance := rtpSequenceDistance(packet.SequenceNumber, buffer.expectedSequence)
	if sequenceDistance < 0 {
		buffer.packetsDropped.Add(1)
		return nil
	}
	if buffer.shouldResync(sequenceDistance) {
		return buffer.resyncToPacket(packet)
	}
	if sequenceDistance == 0 {
		out := buffer.emitTimestampGap(packet)
		out = append(out, buffer.emitPacket(packet)...)
		out = append(out, buffer.flushReadyPackets()...)
		return out
	}

	if _, exists := buffer.bufferedPackets[packet.SequenceNumber]; exists {
		buffer.packetsDropped.Add(1)
		return nil
	}
	buffer.bufferedPackets[packet.SequenceNumber] = cloneRTPPacket(packet)
	return buffer.flushReadyPackets()
}

func (buffer *rtpInputJitterBuffer) flushOnPlayoutTimeout() [][]byte {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	if !buffer.started || len(buffer.bufferedPackets) == 0 {
		return nil
	}
	out := [][]byte{buffer.emitMissingPacket()}
	out = append(out, buffer.flushReadyPackets()...)
	return out
}

func (buffer *rtpInputJitterBuffer) flushReadyPackets() [][]byte {
	var out [][]byte
	for {
		if packet, ok := buffer.bufferedPackets[buffer.expectedSequence]; ok {
			delete(buffer.bufferedPackets, buffer.expectedSequence)
			out = append(out, buffer.emitTimestampGap(packet)...)
			out = append(out, buffer.emitPacket(packet)...)
			continue
		}
		if !buffer.shouldFillMissingPacket() {
			return out
		}
		out = append(out, buffer.emitMissingPacket())
	}
}

func (buffer *rtpInputJitterBuffer) shouldFillMissingPacket() bool {
	for sequenceNumber := range buffer.bufferedPackets {
		if rtpSequenceDistance(sequenceNumber, buffer.expectedSequence) > rtpInputReorderWindowFrames {
			return true
		}
	}
	return false
}

func (buffer *rtpInputJitterBuffer) shouldResync(sequenceDistance int) bool {
	return sequenceDistance > rtpInputMaxLossGapFrames
}

func (buffer *rtpInputJitterBuffer) resyncToPacket(packet *RTPPacket) [][]byte {
	buffer.packetsDropped.Add(uint64(len(buffer.bufferedPackets)))
	buffer.bufferedPackets = make(map[uint16]*RTPPacket, rtpInputReorderWindowFrames+1)
	buffer.expectedSequence = packet.SequenceNumber
	buffer.expectedTimestamp = packet.Timestamp
	return buffer.emitPacket(packet)
}

func (buffer *rtpInputJitterBuffer) emitTimestampGap(packet *RTPPacket) [][]byte {
	if buffer.rtpSamplesPerFrame == 0 {
		return nil
	}
	timestampGap := packet.Timestamp - buffer.expectedTimestamp
	missingFrames := int(timestampGap / buffer.rtpSamplesPerFrame)
	if missingFrames <= 0 || missingFrames > rtpInputMaxSilenceGapFrames {
		return nil
	}
	out := make([][]byte, 0, missingFrames)
	for i := 0; i < missingFrames; i++ {
		out = append(out, cloneBytes(buffer.silencePayload))
		buffer.packetsLost.Add(1)
		buffer.expectedTimestamp += buffer.rtpSamplesPerFrame
	}
	return out
}

func (buffer *rtpInputJitterBuffer) emitPacket(packet *RTPPacket) [][]byte {
	buffer.expectedSequence = packet.SequenceNumber + 1
	buffer.expectedTimestamp = packet.Timestamp + buffer.rtpSamplesPerFrame
	return [][]byte{cloneBytes(packet.Payload)}
}

func (buffer *rtpInputJitterBuffer) emitMissingPacket() []byte {
	buffer.expectedSequence++
	buffer.expectedTimestamp += buffer.rtpSamplesPerFrame
	buffer.packetsLost.Add(1)
	return cloneBytes(buffer.silencePayload)
}

func (buffer *rtpInputJitterBuffer) lostPackets() uint64 {
	if buffer == nil {
		return 0
	}
	return buffer.packetsLost.Load()
}

func (buffer *rtpInputJitterBuffer) droppedPackets() uint64 {
	if buffer == nil {
		return 0
	}
	return buffer.packetsDropped.Load()
}

func rtpSequenceDistance(sequenceNumber uint16, expectedSequence uint16) int {
	// #nosec G115, RTP sequence distance intentionally uses signed 16-bit wrap.
	return int(int16(sequenceNumber - expectedSequence))
}

func cloneRTPPacket(packet *RTPPacket) *RTPPacket {
	if packet == nil {
		return nil
	}
	cloned := *packet
	cloned.Payload = cloneBytes(packet.Payload)
	return &cloned
}

func cloneBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out
}
