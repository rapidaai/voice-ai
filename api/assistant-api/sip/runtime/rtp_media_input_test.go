// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaf/g711"
)

func receiveInboundAudio(t testing.TB, audio <-chan InboundAudioFrame, timeout time.Duration) (InboundAudioFrame, error) {
	t.Helper()
	select {
	case frame := <-audio:
		return frame, nil
	case <-time.After(timeout):
		return InboundAudioFrame{}, context.DeadlineExceeded
	}
}

func captureInboundAudio(t testing.TB, handler *RTPHandler, capacity int) chan InboundAudioFrame {
	t.Helper()
	audio := make(chan InboundAudioFrame, capacity)
	handler.SetInboundAudioSink(func(frame InboundAudioFrame) {
		audio <- frame
	})
	return audio
}

func startRTPInputReceiver(tester *testing.T) (*RTPHandler, <-chan InboundAudioFrame, func(*RTPPacket)) {
	tester.Helper()
	connection, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(tester, err)
	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(tester, err)
	handler := newTestRTPHandler()
	handler.conn = connection
	handler.codec = &CodecPCMU
	handler.inputPacketizationTime = 20 * time.Millisecond
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	audio := captureInboundAudio(tester, handler, 32)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		handler.receiveLoop()
	}()
	tester.Cleanup(func() {
		sender.Close()
		handler.cancel()
		connection.Close()
		select {
		case <-finished:
		case <-time.After(time.Second):
			tester.Error("receive loop did not stop")
		}
	})
	return handler, audio, func(packet *RTPPacket) {
		tester.Helper()
		_, sendErr := sender.WriteToUDP(handler.serializeRTPPacket(packet), connection.LocalAddr().(*net.UDPAddr))
		require.NoError(tester, sendErr)
	}
}

func TestRTPHandler_ReordersAudioWithinDeadline(tester *testing.T) {
	handler, audioIn, send := startRTPInputReceiver(tester)
	sendPacket := func(sequence uint16, timestamp uint32, value byte) {
		send(&RTPPacket{Version: 2, SequenceNumber: sequence, Timestamp: timestamp, SSRC: 42, Payload: bytes.Repeat([]byte{value}, 160)})
	}
	sendPacket(1, 0, 1)
	audio, err := receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	require.Equal(tester, bytes.Repeat([]byte{1}, 160), audio.Audio)
	sendPacket(3, 320, 3)
	_, err = receiveInboundAudio(tester, audioIn, 35*time.Millisecond)
	require.ErrorIs(tester, err, context.DeadlineExceeded)
	sendPacket(2, 160, 2)
	for _, value := range []byte{2, 3} {
		audio, receiveErr := receiveInboundAudio(tester, audioIn, time.Second)
		require.NoError(tester, receiveErr)
		require.Equal(tester, bytes.Repeat([]byte{value}, 160), audio.Audio)
	}
	require.Zero(tester, handler.GetDetailedStats().PacketsLost)
	require.Zero(tester, handler.GetDetailedStats().LateOrDuplicatePackets)
}

func TestRTPHandler_TelephoneEventsDoNotBecomeAudioLoss(tester *testing.T) {
	for _, eventSSRC := range []uint32{42, 99} {
		tester.Run(fmt.Sprint(eventSSRC), func(tester *testing.T) {
			handler, audioIn, send := startRTPInputReceiver(tester)
			first := testRTPInputPacket(1, 0, 1)
			first.SSRC = 42
			send(first)
			require.Eventually(tester, func() bool { return len(audioIn) == 1 }, time.Second, time.Millisecond)
			_, err := receiveInboundAudio(tester, audioIn, time.Second)
			require.NoError(tester, err)
			send(&RTPPacket{Version: 2, SequenceNumber: 2, Timestamp: 160, SSRC: eventSSRC, PayloadType: 101, Payload: []byte{1, 0x80, 0, 160}})
			nextSequence := uint16(3)
			if eventSSRC != 42 {
				nextSequence = 2
			}
			next := testRTPInputPacket(nextSequence, 160, 3)
			next.SSRC = 42
			send(next)
			audio, err := receiveInboundAudio(tester, audioIn, time.Second)
			require.NoError(tester, err)
			require.Equal(tester, next.Payload, audio.Audio)
			require.Zero(tester, handler.GetDetailedStats().PacketsLost)
			require.Zero(tester, handler.GetDetailedStats().InvalidPackets)
		})
	}
}

func TestRTPHandler_InvalidTrafficDoesNotPostponeAudioDeadline(tester *testing.T) {
	handler, audioIn, send := startRTPInputReceiver(tester)
	send(testRTPInputPacket(1, 0, 1))
	require.Eventually(tester, func() bool { return len(audioIn) == 1 }, time.Second, time.Millisecond)
	_, err := receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	send(testRTPInputPacket(3, 320, 3))
	for index := 0; index < 15; index++ {
		send(&RTPPacket{Version: 2, PayloadType: 96, Payload: []byte{1, 2, 3, 4}})
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(tester, 2, len(audioIn))
	audio, err := receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	require.Equal(tester, bytes.Repeat([]byte{0xff}, 160), audio.Audio)
	audio, err = receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	require.Equal(tester, bytes.Repeat([]byte{3}, 160), audio.Audio)
	require.Equal(tester, uint64(1), handler.GetDetailedStats().PacketsLost)
}

func TestRTPHandler_ZeroSSRCStreamCanRestart(tester *testing.T) {
	_, audioIn, send := startRTPInputReceiver(tester)
	send(testRTPInputPacket(100, 0, 1))
	require.Eventually(tester, func() bool { return len(audioIn) == 1 }, time.Second, time.Millisecond)
	_, err := receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	next := testRTPInputPacket(1, 0, 2)
	next.SSRC = 42
	send(next)
	audio, err := receiveInboundAudio(tester, audioIn, time.Second)
	require.NoError(tester, err)
	require.Equal(tester, next.Payload, audio.Audio)
}

func TestRTPHandler_ParseRTPPacketRejectsMalformedHeaders(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "truncated CSRC list",
			data:    []byte{0x81, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1},
			wantErr: errRTPPacketShortCSRCHeader,
		},
		{
			name:    "missing extension header",
			data:    []byte{0x90, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1},
			wantErr: errRTPPacketShortExtension,
		},
		{
			name:    "truncated extension payload",
			data:    []byte{0x90, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1},
			wantErr: errRTPPacketShortExtensionPayload,
		},
		{
			name:    "zero padding length",
			data:    []byte{0xA0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0},
			wantErr: errRTPInvalidPaddingLength,
		},
		{
			name:    "padding exceeds payload",
			data:    []byte{0xA0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 2},
			wantErr: errRTPInvalidPaddingLength,
		},
	}

	handler := newTestRTPHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packet, err := handler.parseRTPPacket(test.data)
			require.Error(t, err)
			assert.ErrorIs(t, err, test.wantErr)
			assert.Nil(t, packet)
		})
	}
}

func TestRTPHandler_ParseRTPPacketPreservesCSRCValues(t *testing.T) {
	data := make([]byte, rtpHeaderSize+8+1)
	data[0] = 0x82
	data[1] = CodecPCMU.PayloadType
	binary.BigEndian.PutUint32(data[12:16], 1001)
	binary.BigEndian.PutUint32(data[16:20], 1002)
	data[20] = 0xFF

	packet, err := newTestRTPHandler().parseRTPPacket(data)

	require.NoError(t, err)
	assert.Equal(t, []uint32{1001, 1002}, packet.CSRC)
	assert.Equal(t, []byte{0xFF}, packet.Payload)
}

func TestRTPHandler_GetRemoteAddrReturnsCopy(t *testing.T) {
	handler := newTestRTPHandler()
	handler.SetRemoteAddress(RTPAddress{IP: "127.0.0.1", Port: 9000})

	address := handler.GetRemoteAddr()
	require.NotNil(t, address)
	address.IP[0] = 10
	address.Port = 10000

	stored := handler.GetRemoteAddr()
	require.NotNil(t, stored)
	assert.Equal(t, "127.0.0.1", stored.IP.String())
	assert.Equal(t, 9000, stored.Port)
}

func TestNewRTPHandlerDoesNotMutateConfig(t *testing.T) {
	config := &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: 20000,
		RTPPortRangeEnd:   20999,
		PayloadType:       CodecPCMU.PayloadType,
	}

	handler, err := NewRTPHandler(context.Background(), config)
	require.NoError(t, err)
	defer handler.Stop()

	assert.Zero(t, config.ClockRate)
	assert.Zero(t, config.MediaTimeoutInitial)
	assert.Zero(t, config.MediaTimeout)
}

func TestNewRTPHandlerInitializesInputPacketizationTime(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		PacketizationTime: 30 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.mu.RLock()
	inputJitter := handler.inputJitter
	handler.mu.RUnlock()
	require.NotNil(t, inputJitter)
	assert.Len(t, inputJitter.silencePayload, 240)
}

func TestRTPHandler_ReceiveLoopDropsNonAudioPayload(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
	audioIn := captureInboundAudio(t, handler, 1)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	_, err = sender.WriteToUDP(handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    96,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0x01},
	}), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		stats := handler.GetDetailedStats()
		return stats.PacketsReceived == 1 &&
			stats.PacketsDropped == 1 &&
			stats.InvalidPackets == 1 &&
			stats.PacketsLost == 0 &&
			stats.PacketsDelivered == 0
	}, time.Second, 10*time.Millisecond)

	assert.Empty(t, audioIn)
}

func TestRTPHandler_ReceiveLoopAcceptsNegotiatedAudioPayload(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMA.PayloadType,
		ClockRate:         CodecPCMA.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
	audioIn := captureInboundAudio(t, handler, 1)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	_, err = sender.WriteToUDP(handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMA.PayloadType,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0xD5},
	}), &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	audio, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xD5}, audio.Audio)

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.PacketsDelivered)
	assert.Zero(t, stats.PacketsDropped)
}

func TestRTPHandler_ReceiveLoopResetsInputJitterOnSSRCChange(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		PacketizationTime: 20 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()
	audioIn := captureInboundAudio(t, handler, 2)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	first := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 1,
		Timestamp:      0,
		SSRC:           1111,
		Payload:        []byte{0x01},
	})
	_, err = sender.WriteToUDP(first, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	audio, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x01}, audio.Audio)

	second := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 1,
		Timestamp:      0,
		SSRC:           2222,
		Payload:        []byte{0x02},
	})
	_, err = sender.WriteToUDP(second, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	audio, err = receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x02}, audio.Audio)

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.PacketsDelivered)
	assert.Zero(t, stats.LateOrDuplicatePackets)
}

func TestRTPHandler_DeliverInboundAudioForwardsFramesInOrder(t *testing.T) {
	handler := newTestRTPHandler()
	audioIn := captureInboundAudio(t, handler, 2)

	handler.deliverInboundAudio([]InboundAudioFrame{
		{Audio: []byte{0x01}},
		{Audio: []byte{0x02}},
	})

	first, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	second, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x01}, first.Audio)
	assert.Equal(t, []byte{0x02}, second.Audio)
	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.PacketsDelivered)
	assert.Zero(t, stats.PacketsDropped)
}

func TestRTPHandler_DetailedStatsSeparateTransportAndDeliveryLiveness(t *testing.T) {
	handler := newTestRTPHandler()
	audioIn := captureInboundAudio(t, handler, 1)

	handler.markInboundRTPReceived(time.Now())

	stats := handler.GetDetailedStats()
	assert.False(t, stats.LastRTPReceivedAt.IsZero())
	assert.True(t, stats.LastAudioDeliveredAt.IsZero())

	handler.deliverInboundAudio([]InboundAudioFrame{{Audio: []byte{0x01}}})
	_, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	stats = handler.GetDetailedStats()
	assert.False(t, stats.LastRTPReceivedAt.IsZero())
	assert.False(t, stats.LastAudioDeliveredAt.IsZero())
}

func TestRTPHandler_DeliverInboundAudioCountsMissingSink(t *testing.T) {
	handler := newTestRTPHandler()

	handler.deliverInboundAudio([]InboundAudioFrame{{Audio: []byte{0x01}}})

	stats := handler.GetDetailedStats()
	assert.Zero(t, stats.PacketsDelivered)
	assert.Equal(t, uint64(1), stats.PacketsDropped)
}

func TestRTPHandler_DetailedStatsSeparateInboundDropCategories(t *testing.T) {
	handler := newTestRTPHandler()
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)
	handler.inputJitter = buffer
	captureInboundAudio(t, handler, 1)

	arrivedAt := time.Now()
	require.Len(t, buffer.push(testRTPInputPacket(1, 0, 0x01), arrivedAt), 1)
	require.Len(t, buffer.push(testRTPInputPacket(2, 640, 0x02), arrivedAt), 4)
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 960, 0x04), arrivedAt))
	require.Len(t, buffer.flushExpired(arrivedAt.Add(rtpInputReorderWindow)), 2)
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 960, 0x09), arrivedAt))
	assert.Empty(t, buffer.push(testRTPInputPacket(6, 1120, 0x06), arrivedAt))
	require.Len(t, buffer.push(testRTPInputPacket(1000, 159840, 0x10), arrivedAt), 1)

	handler.packetsDropped.Add(1)
	handler.invalidPackets.Add(1)
	handler.deliverInboundAudio([]InboundAudioFrame{{Audio: []byte{0x02}}})

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.PacketsLost)
	assert.Equal(t, uint64(1), stats.LateOrDuplicatePackets)
	assert.Equal(t, uint64(1), stats.InvalidPackets)
	assert.Equal(t, uint64(1), stats.JitterBufferResyncDropped)
	assert.Equal(t, uint64(3), stats.SilenceSuppressionFrames)
	assert.Equal(t, uint64(3), stats.PacketsDropped)
}

func TestRTPHandler_StopWaitsForReceiveLoop(t *testing.T) {
	handler := newTestRTPHandler()
	audioIn := captureInboundAudio(t, handler, 1)
	handler.loops.Add(1)
	loopStarted := make(chan struct{})
	go func() {
		defer handler.loops.Done()
		close(loopStarted)
		<-handler.ctx.Done()
		handler.deliverInboundAudio([]InboundAudioFrame{{Audio: []byte{0xFF}}})
	}()

	<-loopStarted
	require.NoError(t, handler.Stop())
	require.NoError(t, handler.Stop())

	audio, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xFF}, audio.Audio)

}

func TestRTPHandler_StopFlushesPendingJitterAudio(t *testing.T) {
	handler := newTestRTPHandler()
	audioIn := captureInboundAudio(t, handler, 2)
	arrivedAt := time.Unix(1, 0)
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU, 20*time.Millisecond)
	handler.inputJitter.push(testRTPInputPacket(1, 0, 1), arrivedAt)
	handler.inputJitter.push(testRTPInputPacket(3, 320, 3), arrivedAt)

	require.NoError(t, handler.Stop())

	audio, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	require.Equal(t, bytes.Repeat([]byte{0xff}, 160), audio.Audio)
	audio, err = receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	require.Equal(t, bytes.Repeat([]byte{3}, 160), audio.Audio)
}

func TestRTPHandler_StopClosesUnstartedSocket(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: 20000,
		RTPPortRangeEnd:   20999,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	port := handler.LocalAddress().Port

	require.NoError(t, handler.Stop())
	require.NoError(t, handler.Stop())

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	})
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestRTPHandler_OwnsBoundPortUntilStop(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	first, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)

	second, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.Error(t, err)
	require.Nil(t, second)
	assert.ErrorIs(t, err, ErrRTPPortRangeExhausted)

	require.NoError(t, first.Stop())

	third, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	require.NoError(t, third.Stop())
}

func TestRTPHandler_UpdatesPortStatsForBindLifecycleAndExhaustion(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	stats := &RTPPortStats{}
	first, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		portStats:         stats,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.portsInUse.Load())
	assert.Equal(t, uint64(1), stats.bindAttempts.Load())
	assert.Equal(t, uint64(0), stats.bindFailures.Load())

	second, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		portStats:         stats,
	})
	require.Error(t, err)
	require.Nil(t, second)
	assert.ErrorIs(t, err, ErrRTPPortRangeExhausted)
	assert.Equal(t, int64(1), stats.portsInUse.Load())
	assert.Equal(t, uint64(2), stats.bindAttempts.Load())
	assert.Equal(t, uint64(1), stats.bindFailures.Load())
	assert.Equal(t, uint64(1), stats.rangeExhaustions.Load())

	require.NoError(t, first.Stop())
	assert.Equal(t, int64(0), stats.portsInUse.Load())
}

func TestRTPHandler_SymmetricRTPUpdatesRemoteAddressFromPacketSource(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		SymmetricRTP:      true,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.SetRemoteAddress(RTPAddress{IP: "127.0.0.1", Port: 9})
	captureInboundAudio(t, handler, 1)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	packet := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 1,
		Timestamp:      160,
		SSRC:           1234,
		Payload:        []byte{0xff},
	})
	_, err = sender.WriteToUDP(packet, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	senderPort := sender.LocalAddr().(*net.UDPAddr).Port
	require.Eventually(t, func() bool {
		remote := handler.GetRemoteAddr()
		return remote != nil && remote.IP.Equal(net.ParseIP("127.0.0.1")) && remote.Port == senderPort
	}, time.Second, 10*time.Millisecond)
}

func TestRTPHandler_RemoteAddressStaysFromSDPWhenSymmetricRTPDisabled(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		SymmetricRTP:      false,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.SetRemoteAddress(RTPAddress{IP: "127.0.0.1", Port: 9})
	captureInboundAudio(t, handler, 1)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	packet := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 1,
		Timestamp:      160,
		SSRC:           1234,
		Payload:        []byte{0xff},
	})
	_, err = sender.WriteToUDP(packet, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		received, _ := handler.GetStats()
		return received > 0
	}, time.Second, 10*time.Millisecond)

	remote := handler.GetRemoteAddr()
	require.NotNil(t, remote)
	assert.Equal(t, "127.0.0.1", remote.IP.String())
	assert.Equal(t, 9, remote.Port)
}

func TestRTPHandler_DropsOversizedDatagramAndContinuesReceiving(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:      RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
	audioIn := captureInboundAudio(t, handler, 1)
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()
	destination := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}

	oversized := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 1,
		Timestamp:      160,
		SSRC:           1234,
		Payload:        make([]byte, rtpPacketMaxSize-rtpHeaderSize+1),
	})
	_, err = sender.WriteToUDP(oversized, destination)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return handler.GetDetailedStats().PacketsDropped == 1
	}, time.Second, 10*time.Millisecond)

	valid := handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    CodecPCMU.PayloadType,
		SequenceNumber: 2,
		Timestamp:      320,
		SSRC:           1234,
		Payload:        []byte{0xFF},
	})
	_, err = sender.WriteToUDP(valid, destination)
	require.NoError(t, err)

	audio, err := receiveInboundAudio(t, audioIn, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xFF}, audio.Audio)
}

func TestRTPHandler_MediaTimeoutUsesInitialWindow(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:        RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart:   20000,
		RTPPortRangeEnd:     20999,
		PayloadType:         CodecPCMU.PayloadType,
		ClockRate:           CodecPCMU.ClockRate,
		MediaTimeoutInitial: 80 * time.Millisecond,
		MediaTimeout:        40 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.EnableMediaTimeout(true)

	select {
	case <-handler.MediaTimeout():
		t.Fatal("initial media timeout fired too early")
	case <-time.After(40 * time.Millisecond):
	}

	select {
	case <-handler.MediaTimeout():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("initial media timeout did not fire")
	}
}

func TestRTPHandler_MediaTimeoutUsesRegularWindowAfterAudio(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:        RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart:   20000,
		RTPPortRangeEnd:     20999,
		PayloadType:         CodecPCMU.PayloadType,
		ClockRate:           CodecPCMU.ClockRate,
		MediaTimeoutInitial: 200 * time.Millisecond,
		MediaTimeout:        50 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.EnableMediaTimeout(true)
	time.Sleep(30 * time.Millisecond)
	handler.markInboundRTPReceived(time.Now())

	select {
	case <-handler.MediaTimeout():
		t.Fatal("regular media timeout fired too early")
	case <-time.After(25 * time.Millisecond):
	}

	select {
	case <-handler.MediaTimeout():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("regular media timeout did not fire")
	}
}

func TestRTPHandler_MediaTimeoutStaysOpenWhileAudioFlows(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalAddress:        RTPAddress{IP: "127.0.0.1"},
		RTPPortRangeStart:   20000,
		RTPPortRangeEnd:     20999,
		PayloadType:         CodecPCMU.PayloadType,
		ClockRate:           CodecPCMU.ClockRate,
		MediaTimeoutInitial: 100 * time.Millisecond,
		MediaTimeout:        60 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.EnableMediaTimeout(true)

	for i := 0; i < 5; i++ {
		handler.markInboundRTPReceived(time.Now())
		select {
		case <-handler.MediaTimeout():
			t.Fatal("media timeout fired while audio was flowing")
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func TestRTPHandler_WriteAudioWritesFrameAndReportsStopped(t *testing.T) {
	handler := newTestRTPHandler()
	receiver := attachRTPOutputReceiver(t, handler)
	require.NoError(t, handler.WriteAudio([]byte{0x01, 0xff}))
	assert.Equal(t, []byte{0x01, 0xff}, readRTPPayload(t, handler, receiver))

	require.NoError(t, handler.Stop())
	err := handler.WriteAudio([]byte{0x01})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRTPHandlerStopped))
}

func TestRTPHandler_G711UDPContinuousWaveform(t *testing.T) {
	const (
		frameCount      = 10
		samplesPerFrame = 160
	)

	for _, codec := range []Codec{CodecPCMU, CodecPCMA} {
		t.Run(codec.Name, func(t *testing.T) {
			receiver := newBoundRTPHandler(t, codec)
			inboundAudio := captureInboundAudio(t, receiver, frameCount)
			receiver.Start()

			sender := newBoundRTPHandler(t, codec)
			sender.Start()
			sender.SetRemoteAddress(receiver.LocalAddress())

			pcm := make([]byte, frameCount*samplesPerFrame*2)
			for sampleIndex := 0; sampleIndex < frameCount*samplesPerFrame; sampleIndex++ {
				sample := int16(12000 * math.Sin(2*math.Pi*440*float64(sampleIndex)/8000))
				binary.LittleEndian.PutUint16(pcm[sampleIndex*2:], uint16(sample))
			}

			encodedFrames := make([][]byte, frameCount)
			for frameIndex := range frameCount {
				start := frameIndex * samplesPerFrame * 2
				end := start + samplesPerFrame*2
				encodedFrames[frameIndex] = g711.EncodeUlaw(pcm[start:end])
				if codec.Name == CodecPCMA.Name {
					encodedFrames[frameIndex] = g711.EncodeAlaw(pcm[start:end])
				}
				require.NoError(t, sender.WriteAudio(encodedFrames[frameIndex]))
			}

			decoded := make([]byte, 0, len(pcm))
			var previousReceivedAt time.Time
			for frameIndex, expected := range encodedFrames {
				frame, err := receiveInboundAudio(t, inboundAudio, time.Second)
				require.NoError(t, err)
				require.Equalf(t, expected, frame.Audio, "frame %d payload", frameIndex)
				require.False(t, frame.ReceivedAt.IsZero())
				if !previousReceivedAt.IsZero() {
					require.False(t, frame.ReceivedAt.Before(previousReceivedAt))
				}
				previousReceivedAt = frame.ReceivedAt

				decodedFrame := g711.DecodeUlaw(frame.Audio)
				if codec.Name == CodecPCMA.Name {
					decodedFrame = g711.DecodeAlaw(frame.Audio)
				}
				decoded = append(decoded, decodedFrame...)
			}
			require.Len(t, decoded, len(pcm))

			var totalError int64
			for sampleIndex := 0; sampleIndex < frameCount*samplesPerFrame; sampleIndex++ {
				original := int64(int16(binary.LittleEndian.Uint16(pcm[sampleIndex*2:])))
				roundTrip := int64(int16(binary.LittleEndian.Uint16(decoded[sampleIndex*2:])))
				sampleError := original - roundTrip
				if sampleError < 0 {
					sampleError = -sampleError
				}
				totalError += sampleError
			}
			assert.Less(t, totalError/(frameCount*samplesPerFrame), int64(500))

			senderStats := sender.GetDetailedStats()
			assert.Equal(t, uint64(frameCount), senderStats.PacketsSent)
			assert.Equal(t, uint64(frameCount*samplesPerFrame), senderStats.BytesSent)

			receiverStats := receiver.GetDetailedStats()
			assert.Equal(t, uint64(frameCount), receiverStats.PacketsReceived)
			assert.Equal(t, uint64(frameCount), receiverStats.PacketsDelivered)
			assert.Equal(t, uint64(frameCount*samplesPerFrame), receiverStats.BytesReceived)
			assert.Zero(t, receiverStats.PacketsLost)
			assert.Zero(t, receiverStats.PacketsDropped)
			assert.Zero(t, receiverStats.LateOrDuplicatePackets)
			assert.Zero(t, receiverStats.InvalidPackets)
		})
	}
}

func newBoundRTPHandler(t *testing.T, codec Codec) *RTPHandler {
	t.Helper()
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(t.Context(), &RTPConfig{
		LocalAddress: RTPAddress{
			IP:   "127.0.0.1",
			Port: port,
		},
		PayloadType:       codec.PayloadType,
		ClockRate:         codec.ClockRate,
		PacketizationTime: 20 * time.Millisecond,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, handler.Stop())
	})
	return handler
}
