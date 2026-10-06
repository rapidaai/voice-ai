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

// rtpInputSilenceFiller owns silence-suppression handling after packet ordering.
// It fills timestamp gaps only when RTP sequence numbers are contiguous.
type rtpInputSilenceFiller struct {
	mu sync.Mutex

	hasAudioTimestamp  bool
	lastAudioSequence  uint16
	expectedTimestamp  uint32
	rtpSamplesPerFrame uint32
	clockRate          uint32
	audioPayloadType   uint8
	silencePayload     []byte

	framesInserted atomic.Uint64
}

func newRTPInputSilenceFiller(codec *Codec, packetizationTime time.Duration) *rtpInputSilenceFiller {
	filler := &rtpInputSilenceFiller{}
	filler.reset(codec, packetizationTime)
	return filler
}

func (filler *rtpInputSilenceFiller) reset(codec *Codec, packetizationTime time.Duration) {
	if codec == nil || codec.ClockRate == 0 {
		codec = &CodecPCMU
	}
	if packetizationTime < rtpMinPacketizationTime ||
		packetizationTime > rtpMaxPacketizationTime ||
		packetizationTime%time.Millisecond != 0 {
		packetizationTime = rtpDefaultPacketizationTime
	}
	// #nosec G115, packetizationTime is validated as a positive bounded duration above.
	samplesPerFrame := uint32(uint64(codec.ClockRate) * uint64(packetizationTime/time.Millisecond) / 1000)
	if samplesPerFrame == 0 {
		codec = &CodecPCMU
		samplesPerFrame = CodecPCMU.ClockRate * sdpDefaultPTimeMS / 1000
	}
	silenceByte := byte(0xFF)
	if codec.Name == CodecPCMA.Name {
		silenceByte = 0xD5
	}

	filler.mu.Lock()
	filler.hasAudioTimestamp = false
	filler.lastAudioSequence = 0
	filler.expectedTimestamp = 0
	filler.rtpSamplesPerFrame = samplesPerFrame
	filler.clockRate = codec.ClockRate
	filler.audioPayloadType = codec.PayloadType
	filler.silencePayload = make([]byte, samplesPerFrame)
	for index := range filler.silencePayload {
		filler.silencePayload[index] = silenceByte
	}
	filler.mu.Unlock()
}

func (filler *rtpInputSilenceFiller) process(packets []rtpBufferedInputPacket) []InboundAudioFrame {
	if len(packets) == 0 {
		return nil
	}

	filler.mu.Lock()
	defer filler.mu.Unlock()

	frames := make([]InboundAudioFrame, 0, len(packets))
	for _, buffered := range packets {
		packet := buffered.packet
		if packet == nil || packet.PayloadType != filler.audioPayloadType || len(packet.Payload) == 0 {
			continue
		}

		if filler.hasAudioTimestamp && packet.SequenceNumber == filler.lastAudioSequence+1 {
			timestampGap := packet.Timestamp - filler.expectedTimestamp
			maxGapSamples := uint64(filler.clockRate) * uint64(rtpInputMaxSilenceGap/time.Millisecond) / 1000
			if timestampGap > 0 && uint64(timestampGap) <= maxGapSamples {
				for timestampGap > 0 {
					samples := min(timestampGap, filler.rtpSamplesPerFrame)
					frames = append(frames, InboundAudioFrame{
						Audio:      cloneBytes(filler.silencePayload[:samples]),
						ReceivedAt: buffered.receivedAt,
					})
					timestampGap -= samples
					filler.framesInserted.Add(1)
				}
			}
		}

		// #nosec G115, RTP payloads are bounded by rtpPacketMaxSize before this stage.
		samples := uint32(len(packet.Payload))
		filler.lastAudioSequence = packet.SequenceNumber
		filler.expectedTimestamp = packet.Timestamp + samples
		filler.hasAudioTimestamp = true
		packetDuration := time.Duration(samples) * time.Second / time.Duration(filler.clockRate)
		if packetDuration >= rtpMinPacketizationTime && packetDuration <= rtpMaxPacketizationTime {
			filler.rtpSamplesPerFrame = samples
			if len(filler.silencePayload) != len(packet.Payload) {
				silenceByte := filler.silencePayload[0]
				filler.silencePayload = make([]byte, samples)
				for index := range filler.silencePayload {
					filler.silencePayload[index] = silenceByte
				}
			}
		}
		frames = append(frames, InboundAudioFrame{Audio: packet.Payload, ReceivedAt: buffered.receivedAt})
	}
	return frames
}

func (filler *rtpInputSilenceFiller) frameCount() uint64 {
	return filler.framesInserted.Load()
}
