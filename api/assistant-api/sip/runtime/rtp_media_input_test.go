// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPHandler_ProcessInboundRTPPacketDropsNonAudioPayload(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMU
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU)

	audioPayloads, acceptedAudio := handler.processInboundRTPPacket(&RTPPacket{
		PayloadType:    101,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0x01},
	})

	assert.False(t, acceptedAudio)
	assert.Empty(t, audioPayloads)
	assert.Equal(t, uint64(1), handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_ProcessInboundRTPPacketAcceptsNegotiatedAudioPayload(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMA
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMA)

	audioPayloads, acceptedAudio := handler.processInboundRTPPacket(&RTPPacket{
		PayloadType:    CodecPCMA.PayloadType,
		SequenceNumber: 1,
		Timestamp:      0,
		Payload:        []byte{0xD5},
	})

	require.True(t, acceptedAudio)
	assert.Equal(t, [][]byte{{0xD5}}, audioPayloads)
	assert.Zero(t, handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_WriteInboundAudioPayloadsCountsInputQueueDrops(t *testing.T) {
	handler := newTestRTPHandler()
	handler.codec = &CodecPCMU
	handler.inputJitter = newRTPInputJitterBuffer(&CodecPCMU)
	handler.audioInChan = make(chan []byte, 1)
	handler.audioInChan <- []byte{0x01}

	stopped := handler.writeInboundAudioPayloads([][]byte{{0x02}}, 2)

	assert.False(t, stopped)
	assert.Equal(t, uint64(1), handler.GetDetailedStats().PacketsDropped)
}

func TestRTPHandler_StopOwnsLoopShutdownBeforeClosingChannels(t *testing.T) {
	handler := newTestRTPHandler()
	handler.loops.Add(1)
	loopStarted := make(chan struct{})
	go func() {
		defer handler.loops.Done()
		close(loopStarted)
		<-handler.ctx.Done()
		_ = handler.writeInboundAudioPayloads([][]byte{{0xFF}}, 1)
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
	handler.markInboundAudioPacketReceived()

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
		handler.markInboundAudioPacketReceived()
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

func TestRTPHandler_NextOutputChunkUsesFallbackOnlyWhenQueueIsEmpty(t *testing.T) {
	handler := newTestRTPHandler()
	handler.SetFallbackAudioSource(func(frameSize int) []byte {
		return []byte{0x11, 0x22}
	})
	silence := []byte{0xff, 0xff}

	pendingAudio := []byte{0x33, 0x44}
	assert.Equal(t, []byte{0x33, 0x44}, handler.nextOutputChunk(&pendingAudio, 2, silence))

	assert.Equal(t, []byte{0x11, 0x22}, handler.nextOutputChunk(&pendingAudio, 2, silence))

	handler.ClearFallbackAudioSource()
	assert.Equal(t, silence, handler.nextOutputChunk(&pendingAudio, 2, silence))
}
