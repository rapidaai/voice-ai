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

type rtpInputJitterBuffer struct {
	mu sync.Mutex

	started            bool
	expectedSequence   uint16
	expectedTimestamp  uint32
	hasAudioTimestamp  bool
	rtpSamplesPerFrame uint32
	packetizationTime  time.Duration
	clockRate          uint32
	audioPayloadType   uint8
	silenceByte        byte
	silencePayload     []byte
	bufferedPackets    map[uint16]rtpBufferedInputPacket

	packetsLost                 atomic.Uint64
	packetsDropped              atomic.Uint64
	lateOrDuplicatePacketsCount atomic.Uint64
	resyncDroppedPacketsCount   atomic.Uint64
	silenceSuppressionFrames    atomic.Uint64
}

func newRTPInputJitterBuffer(codec *Codec, packetizationTime time.Duration) *rtpInputJitterBuffer {
	buffer := &rtpInputJitterBuffer{}
	buffer.reset(codec, packetizationTime)
	return buffer
}

func (buffer *rtpInputJitterBuffer) reset(codec *Codec, packetizationTime time.Duration) {
	if codec == nil || codec.ClockRate == 0 {
		codec = &CodecPCMU
	}
	if packetizationTime < rtpMinPacketizationTime ||
		packetizationTime > rtpMaxPacketizationTime ||
		packetizationTime%time.Millisecond != 0 {
		packetizationTime = rtpDefaultPacketizationTime
	}
	ptimeMS := uint64(packetizationTime / time.Millisecond)
	samplesPerFrame := uint32(uint64(codec.ClockRate) * ptimeMS / 1000)
	if samplesPerFrame == 0 {
		codec = &CodecPCMU
		samplesPerFrame = CodecPCMU.ClockRate * sdpDefaultPTimeMS / 1000
		packetizationTime = rtpDefaultPacketizationTime
	}
	silenceByte := byte(0xFF)
	if codec.Name == CodecPCMA.Name {
		silenceByte = 0xD5
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	discarded := uint64(len(buffer.bufferedPackets))
	buffer.packetsDropped.Add(discarded)
	buffer.resyncDroppedPacketsCount.Add(discarded)
	buffer.started = false
	buffer.expectedSequence = 0
	buffer.expectedTimestamp = 0
	buffer.hasAudioTimestamp = false
	buffer.rtpSamplesPerFrame = samplesPerFrame
	buffer.packetizationTime = packetizationTime
	buffer.clockRate = codec.ClockRate
	buffer.audioPayloadType = codec.PayloadType
	buffer.silenceByte = silenceByte
	buffer.silencePayload = rtpInputSilencePayload(samplesPerFrame, silenceByte)
	buffer.bufferedPackets = make(map[uint16]rtpBufferedInputPacket, rtpInputBufferedPacketMapCapacity)
}

func (buffer *rtpInputJitterBuffer) push(packet *RTPPacket, receivedAt time.Time) []InboundAudioFrame {
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
		buffer.hasAudioTimestamp = false
	}
	bufferedPacket := *packet
	bufferedPacket.Payload = cloneBytes(packet.Payload)
	buffer.bufferedPackets[packet.SequenceNumber] = rtpBufferedInputPacket{packet: &bufferedPacket, receivedAt: receivedAt}
	return append(out, buffer.flushReadyPackets(receivedAt, false)...)
}

func (buffer *rtpInputJitterBuffer) flushExpired(now time.Time) []InboundAudioFrame {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.flushReadyPackets(now, false)
}

func (buffer *rtpInputJitterBuffer) flushPending() []InboundAudioFrame {
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

func (buffer *rtpInputJitterBuffer) flushReadyPackets(now time.Time, force bool) []InboundAudioFrame {
	var out []InboundAudioFrame
	hadLoss := false
	for len(buffer.bufferedPackets) > 0 {
		if buffered, ok := buffer.bufferedPackets[buffer.expectedSequence]; ok {
			delete(buffer.bufferedPackets, buffer.expectedSequence)
			out = append(out, buffer.emitPacket(buffered, hadLoss)...)
			hadLoss = false
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
		buffer.packetsLost.Add(uint64(nextDistance))
		buffer.expectedSequence = nextSequence
		hadLoss = true
	}
	return out
}

func (buffer *rtpInputJitterBuffer) emitPacket(buffered rtpBufferedInputPacket, hadLoss bool) []InboundAudioFrame {
	packet := buffered.packet
	buffer.expectedSequence = packet.SequenceNumber + 1
	if packet.PayloadType != buffer.audioPayloadType {
		buffer.hasAudioTimestamp = false
		return nil
	}
	var out []InboundAudioFrame
	timestampGap := packet.Timestamp - buffer.expectedTimestamp
	maxGap := uint64(buffer.clockRate) * uint64(rtpInputMaxSilenceGap/time.Millisecond) / 1000
	if buffer.hasAudioTimestamp && uint64(timestampGap) <= maxGap {
		for timestampGap > 0 {
			samples := min(timestampGap, buffer.rtpSamplesPerFrame)
			out = append(out, InboundAudioFrame{
				Audio:      cloneBytes(buffer.silencePayload[:samples]),
				ReceivedAt: buffered.receivedAt,
			})
			timestampGap -= samples
			if !hadLoss {
				buffer.silenceSuppressionFrames.Add(1)
			}
		}
	}
	samples := uint32(len(packet.Payload))
	buffer.expectedTimestamp = packet.Timestamp + samples
	buffer.hasAudioTimestamp = true
	packetDuration := time.Duration(samples) * time.Second / time.Duration(buffer.clockRate)
	if packetDuration >= rtpMinPacketizationTime && packetDuration <= rtpMaxPacketizationTime {
		buffer.rtpSamplesPerFrame = samples
		buffer.packetizationTime = packetDuration
		if len(buffer.silencePayload) != len(packet.Payload) {
			buffer.silencePayload = rtpInputSilencePayload(samples, buffer.silenceByte)
		}
	}
	return append(out, InboundAudioFrame{Audio: packet.Payload, ReceivedAt: buffered.receivedAt})
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

func (buffer *rtpInputJitterBuffer) silenceSuppressionFrameCount() uint64 {
	return buffer.silenceSuppressionFrames.Load()
}

func rtpInputSilencePayload(samplesPerFrame uint32, silenceByte byte) []byte {
	payloadSize := int(samplesPerFrame)
	if payloadSize <= 0 {
		return nil
	}
	silencePayload := make([]byte, payloadSize)
	for index := range silencePayload {
		silencePayload[index] = silenceByte
	}
	return silencePayload
}

func rtpSequenceDistance(sequenceNumber uint16, expectedSequence uint16) int {
	// #nosec G115, RTP sequence distance intentionally uses signed 16-bit wrap.
	return int(int16(sequenceNumber - expectedSequence))
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}
