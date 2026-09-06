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

	"github.com/rapidaai/pkg/utils"
)

const (
	rtpInputReorderWindow              = 80 * time.Millisecond
	rtpInputMaxLossGap                 = 500 * time.Millisecond
	rtpInputMaxSilenceGap              = 500 * time.Millisecond
	rtpInputBufferedPacketMapCapacity  = 5
	rtpInputPacketizationStablePackets = 2
	rtpInputNanosecondsPerSecond       = 1000000000
)

type rtpInputJitterBuffer struct {
	mu sync.Mutex

	started                    bool
	expectedSequence           uint16
	expectedTimestamp          uint32
	rtpSamplesPerFrame         uint32
	minRTPSamplesPerFrame      uint32
	maxRTPSamplesPerFrame      uint32
	packetizationTime          time.Duration
	clockRate                  uint32
	silenceByte                byte
	silencePayload             []byte
	bufferedPackets            map[uint16]*RTPPacket
	hasLastPacket              bool
	lastPacketSequence         uint16
	lastPacketTimestamp        uint32
	candidateSamplesPerFrame   uint32
	candidateStablePacketCount int
	pendingPacketizationPacket *RTPPacket
	pendingSamplesPerFrame     uint32

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
	if codec == nil {
		codec = &CodecPCMU
	}
	if codec.ClockRate == 0 {
		codec = &CodecPCMU
	}
	if !validRTPPacketizationTime(packetizationTime) {
		packetizationTime = rtpDefaultPacketizationTime
	}
	ptimeMS, err := utils.Int64ToUint64(packetizationTime.Milliseconds())
	if err != nil {
		ptimeMS = sdpDefaultPTimeMS
	}
	samplesPerFrame, err := utils.Uint64ToUint32(uint64(codec.ClockRate) * ptimeMS / 1000)
	if err != nil || samplesPerFrame == 0 {
		samplesPerFrame = CodecPCMU.ClockRate * sdpDefaultPTimeMS / 1000
		packetizationTime = rtpDefaultPacketizationTime
	}
	minPTimeMS, minPTimeErr := utils.Int64ToUint64(rtpMinPacketizationTime.Milliseconds())
	maxPTimeMS, maxPTimeErr := utils.Int64ToUint64(rtpMaxPacketizationTime.Milliseconds())
	minSamplesPerFrame, minSampleErr := utils.Uint64ToUint32(uint64(codec.ClockRate) * minPTimeMS / 1000)
	maxSamplesPerFrame, maxSampleErr := utils.Uint64ToUint32(uint64(codec.ClockRate) * maxPTimeMS / 1000)
	if minPTimeErr != nil || maxPTimeErr != nil || minSampleErr != nil || maxSampleErr != nil {
		minSamplesPerFrame = CodecPCMU.ClockRate * sdpMinPTimeMS / 1000
		maxSamplesPerFrame = CodecPCMU.ClockRate * sdpMaxPTimeMS / 1000
	}
	silenceByte := byte(0xFF)
	if codec.Name == CodecPCMA.Name {
		silenceByte = 0xD5
	}
	silencePayload := rtpInputSilencePayload(samplesPerFrame, silenceByte)

	buffer.mu.Lock()
	buffer.started = false
	buffer.expectedSequence = 0
	buffer.expectedTimestamp = 0
	buffer.rtpSamplesPerFrame = samplesPerFrame
	buffer.minRTPSamplesPerFrame = minSamplesPerFrame
	buffer.maxRTPSamplesPerFrame = maxSamplesPerFrame
	buffer.packetizationTime = packetizationTime
	buffer.clockRate = codec.ClockRate
	buffer.silenceByte = silenceByte
	buffer.silencePayload = silencePayload
	buffer.bufferedPackets = make(map[uint16]*RTPPacket, rtpInputBufferedPacketMapCapacity)
	buffer.hasLastPacket = false
	buffer.lastPacketSequence = 0
	buffer.lastPacketTimestamp = 0
	buffer.candidateSamplesPerFrame = 0
	buffer.candidateStablePacketCount = 0
	buffer.pendingPacketizationPacket = nil
	buffer.pendingSamplesPerFrame = 0
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
		buffer.observePacketization(packet)
		return buffer.emitPacket(packet)
	}

	out := buffer.resolvePendingPacketization(packet)
	sequenceDistance := rtpSequenceDistance(packet.SequenceNumber, buffer.expectedSequence)
	if sequenceDistance < 0 {
		buffer.packetsDropped.Add(1)
		buffer.lateOrDuplicatePacketsCount.Add(1)
		return out
	}
	if buffer.shouldResync(sequenceDistance) {
		return append(out, buffer.resyncToPacket(packet)...)
	}
	if sequenceDistance == 0 {
		isPacketizationChanging := buffer.observePacketization(packet)
		if isPacketizationChanging &&
			buffer.candidateStablePacketCount == 1 &&
			buffer.pendingPacketizationPacket == nil {
			pendingPacket := *packet
			pendingPacket.Payload = cloneBytes(packet.Payload)
			buffer.pendingPacketizationPacket = &pendingPacket
			buffer.pendingSamplesPerFrame = buffer.candidateSamplesPerFrame
			return out
		}
		if !isPacketizationChanging || buffer.candidateStablePacketCount == 0 {
			out = append(out, buffer.emitTimestampGap(packet)...)
		}
		out = append(out, buffer.emitPacket(packet)...)
		out = append(out, buffer.flushReadyPackets()...)
		return out
	}

	if _, exists := buffer.bufferedPackets[packet.SequenceNumber]; exists {
		buffer.packetsDropped.Add(1)
		buffer.lateOrDuplicatePacketsCount.Add(1)
		return out
	}
	bufferedPacket := *packet
	bufferedPacket.Payload = cloneBytes(packet.Payload)
	buffer.bufferedPackets[packet.SequenceNumber] = &bufferedPacket
	return append(out, buffer.flushReadyPackets()...)
}

func (buffer *rtpInputJitterBuffer) flushOnPlayoutTimeout() [][]byte {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	out := buffer.resolvePendingPacketization(nil)
	if !buffer.started || len(buffer.bufferedPackets) == 0 {
		return out
	}
	out = append(out, buffer.emitMissingPacket())
	out = append(out, buffer.flushReadyPackets()...)
	return out
}

func (buffer *rtpInputJitterBuffer) resolvePendingPacketization(packet *RTPPacket) [][]byte {
	pendingPacket := buffer.pendingPacketizationPacket
	if pendingPacket == nil {
		return nil
	}
	pendingSamplesPerFrame := buffer.pendingSamplesPerFrame
	buffer.pendingPacketizationPacket = nil
	buffer.pendingSamplesPerFrame = 0

	isStablePacketization := false
	if packet != nil && packet.SequenceNumber == pendingPacket.SequenceNumber+1 {
		delta := packet.Timestamp - pendingPacket.Timestamp
		isStablePacketization = delta == pendingSamplesPerFrame &&
			buffer.commitSamplesPerFrame(pendingSamplesPerFrame)
	}
	if !isStablePacketization {
		buffer.candidateSamplesPerFrame = 0
		buffer.candidateStablePacketCount = 0
		out := buffer.emitTimestampGap(pendingPacket)
		out = append(out, buffer.emitPacket(pendingPacket)...)
		return out
	}
	buffer.candidateSamplesPerFrame = 0
	buffer.candidateStablePacketCount = 0
	out := buffer.emitPacket(pendingPacket)
	return out
}

func (buffer *rtpInputJitterBuffer) flushReadyPackets() [][]byte {
	var out [][]byte
	for {
		if packet, ok := buffer.bufferedPackets[buffer.expectedSequence]; ok {
			delete(buffer.bufferedPackets, buffer.expectedSequence)
			isPacketizationChanging := buffer.observePacketization(packet)
			if !isPacketizationChanging {
				out = append(out, buffer.emitTimestampGap(packet)...)
			}
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
	reorderWindowFrames := buffer.framesForDuration(rtpInputReorderWindow)
	for sequenceNumber := range buffer.bufferedPackets {
		if rtpSequenceDistance(sequenceNumber, buffer.expectedSequence) > reorderWindowFrames {
			return true
		}
	}
	return false
}

func (buffer *rtpInputJitterBuffer) shouldResync(sequenceDistance int) bool {
	return sequenceDistance > buffer.framesForDuration(rtpInputMaxLossGap)
}

func (buffer *rtpInputJitterBuffer) resyncToPacket(packet *RTPPacket) [][]byte {
	bufferedPackets, err := utils.IntToUint64(len(buffer.bufferedPackets))
	if err == nil {
		buffer.packetsDropped.Add(bufferedPackets)
		buffer.resyncDroppedPacketsCount.Add(bufferedPackets)
	}
	buffer.bufferedPackets = make(map[uint16]*RTPPacket, rtpInputBufferedPacketMapCapacity)
	buffer.expectedSequence = packet.SequenceNumber
	buffer.expectedTimestamp = packet.Timestamp
	buffer.hasLastPacket = false
	buffer.candidateSamplesPerFrame = 0
	buffer.candidateStablePacketCount = 0
	buffer.pendingPacketizationPacket = nil
	buffer.pendingSamplesPerFrame = 0
	buffer.observePacketization(packet)
	return buffer.emitPacket(packet)
}

func (buffer *rtpInputJitterBuffer) emitTimestampGap(packet *RTPPacket) [][]byte {
	if buffer.rtpSamplesPerFrame == 0 {
		return nil
	}
	timestampGap := packet.Timestamp - buffer.expectedTimestamp
	missingFrames, err := utils.Uint32ToInt(timestampGap / buffer.rtpSamplesPerFrame)
	if err != nil {
		return nil
	}
	if missingFrames <= 0 || missingFrames > buffer.framesForDuration(rtpInputMaxSilenceGap) {
		return nil
	}
	out := make([][]byte, 0, missingFrames)
	for i := 0; i < missingFrames; i++ {
		out = append(out, cloneBytes(buffer.silencePayload))
		buffer.expectedTimestamp += buffer.rtpSamplesPerFrame
	}
	if frames, err := utils.IntToUint64(missingFrames); err == nil {
		buffer.silenceSuppressionFrames.Add(frames)
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

func (buffer *rtpInputJitterBuffer) lateOrDuplicatePackets() uint64 {
	if buffer == nil {
		return 0
	}
	return buffer.lateOrDuplicatePacketsCount.Load()
}

func (buffer *rtpInputJitterBuffer) resyncDroppedPackets() uint64 {
	if buffer == nil {
		return 0
	}
	return buffer.resyncDroppedPacketsCount.Load()
}

func (buffer *rtpInputJitterBuffer) silenceSuppressionFrameCount() uint64 {
	if buffer == nil {
		return 0
	}
	return buffer.silenceSuppressionFrames.Load()
}

func (buffer *rtpInputJitterBuffer) playoutTimeout() time.Duration {
	if buffer == nil {
		return rtpDefaultPacketizationTime
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	if !validRTPPacketizationTime(buffer.packetizationTime) {
		return rtpDefaultPacketizationTime
	}
	return buffer.packetizationTime
}

func (buffer *rtpInputJitterBuffer) framesForDuration(limit time.Duration) int {
	packetizationTime := buffer.packetizationTime
	if !validRTPPacketizationTime(packetizationTime) {
		packetizationTime = rtpDefaultPacketizationTime
	}
	frames64 := int64(limit / packetizationTime)
	if limit%packetizationTime != 0 {
		frames64++
	}
	frames, err := utils.Int64ToInt(frames64)
	if err != nil || frames < 1 {
		return 1
	}
	return frames
}

func (buffer *rtpInputJitterBuffer) observePacketization(packet *RTPPacket) bool {
	if packet == nil {
		return false
	}
	if !buffer.hasLastPacket {
		buffer.hasLastPacket = true
		buffer.lastPacketSequence = packet.SequenceNumber
		buffer.lastPacketTimestamp = packet.Timestamp
		return false
	}

	expectedSequence := buffer.lastPacketSequence + 1
	if packet.SequenceNumber != expectedSequence {
		buffer.candidateSamplesPerFrame = 0
		buffer.candidateStablePacketCount = 0
		buffer.lastPacketSequence = packet.SequenceNumber
		buffer.lastPacketTimestamp = packet.Timestamp
		return false
	}

	delta := packet.Timestamp - buffer.lastPacketTimestamp
	buffer.lastPacketSequence = packet.SequenceNumber
	buffer.lastPacketTimestamp = packet.Timestamp

	hasValidSamplesPerFrame := delta >= buffer.minRTPSamplesPerFrame &&
		delta <= buffer.maxRTPSamplesPerFrame
	if delta == 0 || !hasValidSamplesPerFrame {
		buffer.candidateSamplesPerFrame = 0
		buffer.candidateStablePacketCount = 0
		return false
	}
	if delta == buffer.rtpSamplesPerFrame {
		buffer.candidateSamplesPerFrame = 0
		buffer.candidateStablePacketCount = 0
		return false
	}
	if delta != buffer.candidateSamplesPerFrame {
		buffer.candidateSamplesPerFrame = delta
		buffer.candidateStablePacketCount = 1
		return true
	}

	buffer.candidateStablePacketCount++
	if buffer.candidateStablePacketCount < rtpInputPacketizationStablePackets {
		return true
	}
	buffer.commitSamplesPerFrame(delta)
	buffer.candidateSamplesPerFrame = 0
	buffer.candidateStablePacketCount = 0
	return true
}

func (buffer *rtpInputJitterBuffer) commitSamplesPerFrame(samplesPerFrame uint32) bool {
	if buffer.clockRate == 0 || samplesPerFrame == 0 {
		return false
	}
	packetizationNanos := uint64(samplesPerFrame) * rtpInputNanosecondsPerSecond / uint64(buffer.clockRate)
	packetizationNanosInt64, err := utils.Uint64ToInt64(packetizationNanos)
	if err != nil {
		return false
	}
	buffer.rtpSamplesPerFrame = samplesPerFrame
	buffer.packetizationTime = time.Duration(packetizationNanosInt64)
	buffer.silencePayload = rtpInputSilencePayload(samplesPerFrame, buffer.silenceByte)
	return true
}

func rtpInputSilencePayload(samplesPerFrame uint32, silenceByte byte) []byte {
	payloadSize, err := utils.Uint32ToInt(samplesPerFrame)
	if err != nil || payloadSize <= 0 {
		return nil
	}
	silencePayload := make([]byte, payloadSize)
	for i := range silencePayload {
		silencePayload[i] = silenceByte
	}
	return silencePayload
}

func rtpSequenceDistance(sequenceNumber uint16, expectedSequence uint16) int {
	// #nosec G115, RTP sequence distance intentionally uses signed 16-bit wrap.
	return int(int16(sequenceNumber - expectedSequence))
}

func cloneBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out
}
