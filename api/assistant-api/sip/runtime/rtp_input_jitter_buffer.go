// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"sync"
	"sync/atomic"
	"time"
)

type rtpBufferedInputPacket struct {
	packet     *RTPPacket
	receivedAt time.Time
}

// InboundAudioFrame carries audio and the time it entered the RTP receive path.
type InboundAudioFrame struct {
	Audio      []byte
	ReceivedAt time.Time
}

// rtpInputJitterBuffer owns packet ordering and loss detection only.
type rtpInputJitterBuffer struct {
	mu sync.Mutex

	started           bool
	expectedSequence  uint16
	packetizationTime time.Duration
	bufferedPackets   map[uint16]rtpBufferedInputPacket

	packetsLost                 atomic.Uint64
	packetsDropped              atomic.Uint64
	lateOrDuplicatePacketsCount atomic.Uint64
	resyncDroppedPacketsCount   atomic.Uint64
}

func newRTPInputJitterBuffer(packetizationTime time.Duration) *rtpInputJitterBuffer {
	buffer := &rtpInputJitterBuffer{}
	buffer.reset(packetizationTime)
	return buffer
}

func (buffer *rtpInputJitterBuffer) reset(packetizationTime time.Duration) {
	if packetizationTime < rtpMinPacketizationTime ||
		packetizationTime > rtpMaxPacketizationTime ||
		packetizationTime%time.Millisecond != 0 {
		packetizationTime = rtpDefaultPacketizationTime
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	discarded := uint64(len(buffer.bufferedPackets))
	buffer.packetsDropped.Add(discarded)
	buffer.resyncDroppedPacketsCount.Add(discarded)
	buffer.started = false
	buffer.expectedSequence = 0
	buffer.packetizationTime = packetizationTime
	buffer.bufferedPackets = make(map[uint16]rtpBufferedInputPacket, rtpInputBufferedPacketMapCapacity)
}

func (buffer *rtpInputJitterBuffer) push(packet *RTPPacket, receivedAt time.Time) []rtpBufferedInputPacket {
	if packet == nil || len(packet.Payload) == 0 {
		return nil
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	out := buffer.flushReadyPackets(receivedAt, false)
	if !buffer.started {
		buffer.started = true
		buffer.expectedSequence = packet.SequenceNumber
	}
	distance := rtpSequenceDistance(packet.SequenceNumber, buffer.expectedSequence)
	if _, duplicate := buffer.bufferedPackets[packet.SequenceNumber]; distance < 0 || duplicate {
		buffer.packetsDropped.Add(1)
		buffer.lateOrDuplicatePacketsCount.Add(1)
		return out
	}
	if distance > buffer.framesForDuration(rtpInputMaxLossGap) {
		discarded := uint64(len(buffer.bufferedPackets))
		buffer.packetsDropped.Add(discarded)
		buffer.resyncDroppedPacketsCount.Add(discarded)
		clear(buffer.bufferedPackets)
		buffer.expectedSequence = packet.SequenceNumber
	}
	bufferedPacket := *packet
	bufferedPacket.Payload = cloneBytes(packet.Payload)
	buffer.bufferedPackets[packet.SequenceNumber] = rtpBufferedInputPacket{packet: &bufferedPacket, receivedAt: receivedAt}
	return append(out, buffer.flushReadyPackets(receivedAt, false)...)
}

func (buffer *rtpInputJitterBuffer) flushExpired(now time.Time) []rtpBufferedInputPacket {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.flushReadyPackets(now, false)
}

func (buffer *rtpInputJitterBuffer) flushPending() []rtpBufferedInputPacket {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.flushReadyPackets(time.Time{}, true)
}

func (buffer *rtpInputJitterBuffer) nextDeadline() time.Time {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	var deadline time.Time
	for _, packet := range buffer.bufferedPackets {
		candidate := packet.receivedAt.Add(rtpInputReorderWindow)
		if deadline.IsZero() || candidate.Before(deadline) {
			deadline = candidate
		}
	}
	return deadline
}

func (buffer *rtpInputJitterBuffer) flushReadyPackets(now time.Time, force bool) []rtpBufferedInputPacket {
	var out []rtpBufferedInputPacket
	for len(buffer.bufferedPackets) > 0 {
		if buffered, ok := buffer.bufferedPackets[buffer.expectedSequence]; ok {
			delete(buffer.bufferedPackets, buffer.expectedSequence)
			buffer.expectedSequence++
			out = append(out, buffered)
			continue
		}
		var deadline time.Time
		var nextSequence uint16
		nextDistance := 1 << 15
		for sequence, packet := range buffer.bufferedPackets {
			candidate := packet.receivedAt.Add(rtpInputReorderWindow)
			if deadline.IsZero() || candidate.Before(deadline) {
				deadline = candidate
			}
			if distance := rtpSequenceDistance(sequence, buffer.expectedSequence); distance < nextDistance {
				nextSequence = sequence
				nextDistance = distance
			}
		}
		if !force && now.Before(deadline) {
			break
		}
		// #nosec G115, buffered sequence distances are non-negative and bounded by the RTP sequence space.
		buffer.packetsLost.Add(uint64(nextDistance))
		buffer.expectedSequence = nextSequence
	}
	return out
}

func (buffer *rtpInputJitterBuffer) framesForDuration(limit time.Duration) int {
	frames := limit / buffer.packetizationTime
	if limit%buffer.packetizationTime != 0 {
		frames++
	}
	count := int(frames)
	if count < 1 {
		return 1
	}
	return count
}

func (buffer *rtpInputJitterBuffer) lostPackets() uint64 {
	return buffer.packetsLost.Load()
}

func (buffer *rtpInputJitterBuffer) droppedPackets() uint64 {
	return buffer.packetsDropped.Load()
}

func (buffer *rtpInputJitterBuffer) lateOrDuplicatePackets() uint64 {
	return buffer.lateOrDuplicatePacketsCount.Load()
}

func (buffer *rtpInputJitterBuffer) resyncDroppedPackets() uint64 {
	return buffer.resyncDroppedPacketsCount.Load()
}

func rtpSequenceDistance(sequenceNumber uint16, expectedSequence uint16) int {
	// #nosec G115, RTP sequence distance intentionally uses signed 16-bit wrap.
	return int(int16(sequenceNumber - expectedSequence))
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}
