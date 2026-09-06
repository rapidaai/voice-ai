// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	handler.SetRemoteAddr("127.0.0.1", 9000)

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
		LocalIP:           "127.0.0.1",
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
		LocalIP:           "127.0.0.1",
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
	assert.Equal(t, 30*time.Millisecond, inputJitter.playoutTimeout())
}

func TestRTPHandler_ReceiveLoopDropsNonAudioPayload(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
	handler.Start()

	sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer sender.Close()

	_, err = sender.WriteToUDP(handler.serializeRTPPacket(&RTPPacket{
		Version:        rtpVersion,
		PayloadType:    101,
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

	select {
	case audio := <-handler.AudioIn():
		t.Fatalf("unexpected audio payload: %v", audio)
	default:
	}
}

func TestRTPHandler_ReceiveLoopAcceptsNegotiatedAudioPayload(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMA.PayloadType,
		ClockRate:         CodecPCMA.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
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

	select {
	case audio := <-handler.AudioIn():
		assert.Equal(t, []byte{0xD5}, audio)
	case <-time.After(time.Second):
		t.Fatal("expected negotiated audio payload")
	}

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
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		PacketizationTime: 20 * time.Millisecond,
	})
	require.NoError(t, err)
	defer handler.Stop()
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

	select {
	case audio := <-handler.AudioIn():
		assert.Equal(t, []byte{0x01}, audio)
	case <-time.After(time.Second):
		t.Fatal("expected first SSRC audio")
	}

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

	select {
	case audio := <-handler.AudioIn():
		assert.Equal(t, []byte{0x02}, audio)
	case <-time.After(time.Second):
		t.Fatal("expected second SSRC audio")
	}

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.PacketsDelivered)
	assert.Zero(t, stats.LateOrDuplicatePackets)
}

func TestRTPHandler_EnqueueInboundAudioCountsInputQueueDrops(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMU
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)
	handler.audioInChan = make(chan []byte, 1)
	handler.audioInChan <- []byte{0x01}

	stopped, enqueued := handler.enqueueInboundAudio([][]byte{{0x02}})

	assert.False(t, stopped)
	assert.True(t, enqueued)
	assert.Equal(t, []byte{0x02}, <-handler.audioInChan)
	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.PacketsDelivered)
	assert.Equal(t, uint64(1), stats.PacketsDropped)
	assert.Equal(t, uint64(1), stats.AudioInputDropped)
	assert.Equal(t, uint64(1), stats.RTPIngressQueueDropped)
	assert.Zero(t, stats.PacketsLost)
	assert.Zero(t, stats.NetworkPacketsLost)
	assert.True(t, stats.LastRTPReceivedAt.IsZero())
	assert.False(t, stats.LastAudioDeliveredAt.IsZero())
}

func TestRTPHandler_EnqueueInboundAudioReportsEnqueueState(t *testing.T) {
	handler := newTestRTPHandler()
	handler.audioInChan = make(chan []byte, 1)
	handler.audioInChan <- []byte{0x01}

	stopped, enqueued := handler.enqueueInboundAudio([][]byte{{0x02}})

	assert.False(t, stopped)
	assert.True(t, enqueued)
	assert.Equal(t, []byte{0x02}, <-handler.audioInChan)
	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.PacketsDelivered)
	assert.Equal(t, uint64(1), stats.PacketsDropped)
	assert.Equal(t, uint64(1), stats.AudioInputDropped)
	assert.Equal(t, uint64(1), stats.RTPIngressQueueDropped)
	assert.Zero(t, stats.NetworkPacketsLost)

	stopped, enqueued = handler.enqueueInboundAudio([][]byte{{0x03}})

	assert.False(t, stopped)
	assert.True(t, enqueued)
	stats = handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.PacketsDelivered)
	assert.Equal(t, uint64(1), stats.PacketsDropped)
	assert.Equal(t, uint64(1), stats.AudioInputDropped)
	assert.Equal(t, uint64(1), stats.RTPIngressQueueDropped)
	assert.Zero(t, stats.NetworkPacketsLost)
}

func TestRTPHandler_DetailedStatsSeparateTransportAndDeliveryLiveness(t *testing.T) {
	handler := newTestRTPHandler()
	handler.audioInChan = make(chan []byte, 1)

	handler.markInboundRTPReceived(time.Now())

	stats := handler.GetDetailedStats()
	assert.False(t, stats.LastRTPReceivedAt.IsZero())
	assert.True(t, stats.LastAudioDeliveredAt.IsZero())

	stopped, enqueued := handler.enqueueInboundAudio([][]byte{{0x01}})

	require.False(t, stopped)
	require.True(t, enqueued)
	stats = handler.GetDetailedStats()
	assert.False(t, stats.LastRTPReceivedAt.IsZero())
	assert.False(t, stats.LastAudioDeliveredAt.IsZero())
}

func TestRTPHandler_EnqueueInboundAudioReportsPartialEnqueue(t *testing.T) {
	handler := newTestRTPHandler()
	handler.audioInChan = make(chan []byte, 1)
	handler.inboundQuality.recordReceived(time.Now(), 2)

	stopped, enqueued := handler.enqueueInboundAudio([][]byte{{0x01}, {0x02}})

	assert.False(t, stopped)
	assert.True(t, enqueued)
	assert.Equal(t, []byte{0x02}, <-handler.audioInChan)
	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.PacketsDelivered)
	assert.Equal(t, uint64(1), stats.PacketsDropped)
	assert.Equal(t, uint64(1), stats.AudioInputDropped)
	assert.Equal(t, uint64(1), stats.RTPIngressQueueDropped)
	assert.Zero(t, stats.NetworkPacketsLost)
	assert.Equal(t, rtpInboundQualityPoor, stats.InboundQuality)
	assert.Equal(t, uint64(2), stats.InboundWindowPacketsReceived)
	assert.Equal(t, uint64(2), stats.InboundWindowPacketsDelivered)
	assert.Equal(t, uint64(1), stats.InboundWindowAudioInputDropped)
	assert.InDelta(t, 0.5, stats.InboundLossRate, 0.0001)
	assert.InDelta(t, 1.0/3.0, stats.InboundDropRate, 0.0001)
	assert.InDelta(t, 2.0/3.0, stats.InboundDeliveryRate, 0.0001)
}

func TestRTPHandler_DetailedStatsSeparateInboundDropCategories(t *testing.T) {
	handler := newTestRTPHandler()
	buffer := newRTPInputJitterBuffer(&CodecPCMU, rtpDefaultPacketizationTime)
	handler.inputJitter = buffer
	handler.audioInChan = make(chan []byte, 1)
	handler.audioInChan <- []byte{0x01}

	require.Equal(t, [][]byte{{0x01}}, buffer.push(testRTPInputPacket(1, 0, 0x01)))
	require.Len(t, buffer.push(testRTPInputPacket(2, 640, 0x02)), 4)
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 960, 0x04)))
	require.Len(t, buffer.flushOnPlayoutTimeout(), 2)
	assert.Empty(t, buffer.push(testRTPInputPacket(4, 960, 0x09)))
	assert.Empty(t, buffer.push(testRTPInputPacket(6, 1120, 0x06)))
	require.Equal(t, [][]byte{{0x10}}, buffer.push(testRTPInputPacket(1000, 159840, 0x10)))

	handler.packetsDropped.Add(1)
	handler.invalidPackets.Add(1)
	stopped, enqueued := handler.enqueueInboundAudio([][]byte{{0x02}})
	require.False(t, stopped)
	require.True(t, enqueued)

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.PacketsLost)
	assert.Equal(t, uint64(1), stats.NetworkPacketsLost)
	assert.Equal(t, uint64(1), stats.LateOrDuplicatePackets)
	assert.Equal(t, uint64(1), stats.InvalidPackets)
	assert.Equal(t, uint64(1), stats.JitterBufferResyncDropped)
	assert.Equal(t, uint64(1), stats.RTPIngressQueueDropped)
	assert.Equal(t, uint64(3), stats.SilenceSuppressionFrames)
	assert.Equal(t, uint64(4), stats.PacketsDropped)
}

func TestRTPHandler_StopOwnsLoopShutdownBeforeClosingChannels(t *testing.T) {
	handler := newTestRTPHandler()
	handler.loops.Add(1)
	loopStarted := make(chan struct{})
	go func() {
		defer handler.loops.Done()
		close(loopStarted)
		<-handler.ctx.Done()
		_, _ = handler.enqueueInboundAudio([][]byte{{0xFF}})
	}()

	<-loopStarted
	require.NoError(t, handler.Stop())
	require.NoError(t, handler.Stop())

	require.Eventually(t, func() bool {
		select {
		case _, ok := <-handler.audioInChan:
			return !ok
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	select {
	case _, ok := <-handler.audioOutChan:
		require.True(t, ok, "RTP output queue must not be closed by Stop")
	default:
	}
}

func TestRTPHandler_StopClosesUnstartedSocket(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: 20000,
		RTPPortRangeEnd:   20999,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	_, port := handler.LocalAddr()

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
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)

	second, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:           "127.0.0.1",
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
		LocalIP:           "127.0.0.1",
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
		LocalIP:           "127.0.0.1",
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
		LocalIP:           "127.0.0.1",
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
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		SymmetricRTP:      true,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.SetRemoteAddr("127.0.0.1", 9)
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
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
		SymmetricRTP:      false,
	})
	require.NoError(t, err)
	defer handler.Stop()

	handler.SetRemoteAddr("127.0.0.1", 9)
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
		LocalIP:           "127.0.0.1",
		RTPPortRangeStart: port,
		RTPPortRangeEnd:   port,
		PayloadType:       CodecPCMU.PayloadType,
		ClockRate:         CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()
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

	select {
	case audio := <-handler.AudioIn():
		assert.Equal(t, []byte{0xFF}, audio)
	case <-time.After(time.Second):
		t.Fatal("expected valid RTP after oversized datagram")
	}
}

func TestRTPHandler_MediaTimeoutUsesInitialWindow(t *testing.T) {
	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:             "127.0.0.1",
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
		LocalIP:             "127.0.0.1",
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
		LocalIP:             "127.0.0.1",
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

func TestRTPHandler_EnqueueAudioReportsBackpressureAndStopped(t *testing.T) {
	handler := newTestRTPHandler()
	for i := 0; i < cap(handler.audioOutChan); i++ {
		require.NoError(t, handler.EnqueueAudio([]byte{byte(i)}))
	}

	err := handler.EnqueueAudio([]byte{0xff})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRTPOutputQueueFull))

	require.NoError(t, handler.Stop())
	err = handler.EnqueueAudio([]byte{0x01})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRTPHandlerStopped))
}
