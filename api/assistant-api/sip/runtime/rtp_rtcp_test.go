package sip_runtime

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pion/rtcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRTPHandlerBindsRTCPCompanionPort(t *testing.T) {
	rtpPort, rtcpPort := reserveUDPPortPair(t)

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:     "127.0.0.1",
		LocalPort:   rtpPort,
		PayloadType: CodecPCMU.PayloadType,
		ClockRate:   CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()

	assert.Equal(t, rtcpPort, handler.LocalRTCPPort())
	stats := handler.GetDetailedStats()
	assert.True(t, stats.RTCPEnabled)
	assert.Equal(t, rtcpPort, stats.LocalRTCPPort)
}

func TestRTPHandlerContinuesWhenRTCPCompanionPortUnavailable(t *testing.T) {
	rtpPort, rtcpPort := reserveUDPPortPair(t)
	blocker, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: rtcpPort,
	})
	require.NoError(t, err)
	defer blocker.Close()

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:     "127.0.0.1",
		LocalPort:   rtpPort,
		PayloadType: CodecPCMU.PayloadType,
		ClockRate:   CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()

	assert.Zero(t, handler.LocalRTCPPort())
	assert.False(t, handler.GetDetailedStats().RTCPEnabled)
}

func TestRTPHandlerSendsRTCPReportWithInboundReceptionStats(t *testing.T) {
	rtpPort, _ := reserveUDPPortPair(t)
	remoteRTCP, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	require.NoError(t, err)
	defer remoteRTCP.Close()

	handler, err := NewRTPHandler(context.Background(), &RTPConfig{
		LocalIP:     "127.0.0.1",
		LocalPort:   rtpPort,
		PayloadType: CodecPCMU.PayloadType,
		ClockRate:   CodecPCMU.ClockRate,
	})
	require.NoError(t, err)
	defer handler.Stop()

	remoteRTCPPort := remoteRTCP.LocalAddr().(*net.UDPAddr).Port
	handler.SetRemoteAddr("127.0.0.1", 9)
	handler.SetRemoteRTCPAddr("127.0.0.1", remoteRTCPPort)
	handler.remoteSSRC.Store(1234)

	now := time.Unix(1_700_000_000, 0)
	handler.rtcpReception.recordRTP(&RTPPacket{
		SequenceNumber: 1,
		Timestamp:      160,
	}, CodecPCMU.ClockRate, now)
	handler.rtcpReception.recordRTP(&RTPPacket{
		SequenceNumber: 3,
		Timestamp:      480,
	}, CodecPCMU.ClockRate, now.Add(40*time.Millisecond))
	handler.packetsSent.Store(1)
	handler.bytesSent.Store(160)

	require.NoError(t, handler.sendRTCPReport(now.Add(100*time.Millisecond)))

	buf := make([]byte, rtcpReadBufferSize)
	require.NoError(t, remoteRTCP.SetReadDeadline(time.Now().Add(time.Second)))
	n, _, err := remoteRTCP.ReadFromUDP(buf)
	require.NoError(t, err)
	packets, err := rtcp.Unmarshal(buf[:n])
	require.NoError(t, err)

	var senderReport *rtcp.SenderReport
	for _, packet := range packets {
		switch typedPacket := packet.(type) {
		case *rtcp.CompoundPacket:
			compound := typedPacket
			for _, innerPacket := range *compound {
				if report, ok := innerPacket.(*rtcp.SenderReport); ok {
					senderReport = report
				}
			}
		case *rtcp.SenderReport:
			senderReport = typedPacket
		}
	}
	require.NotNil(t, senderReport)
	assert.Equal(t, handler.ssrc, senderReport.SSRC)
	require.Len(t, senderReport.Reports, 1)
	assert.Equal(t, uint32(1234), senderReport.Reports[0].SSRC)
	assert.Equal(t, uint32(1), senderReport.Reports[0].TotalLost)
	assert.NotZero(t, senderReport.Reports[0].FractionLost)

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(2), stats.RTCPPacketsSent)
	assert.Equal(t, uint64(1), stats.RTCPReportsSent)
	assert.Equal(t, uint64(1), stats.RTCPSenderReportsSent)
	assert.Equal(t, uint8(85), stats.RTCPFractionLost)
	assert.Equal(t, uint32(1), stats.RTCPPacketsLost)
}

func TestRTPHandlerRecordsRTCPReceiverReportFeedback(t *testing.T) {
	handler := newTestRTPHandler()
	handler.ssrc = 5678

	receivedAt := time.Unix(1_700_000_000, 0)
	delay := uint32((20 * time.Millisecond).Nanoseconds() * rtcpCompactUnit / rtcpNanoseconds)
	handler.recordRTCPPacket(&rtcp.ReceiverReport{
		Reports: []rtcp.ReceptionReport{
			{
				SSRC:             handler.ssrc,
				FractionLost:     32,
				TotalLost:        2,
				Jitter:           7,
				LastSenderReport: rtcpCompactNTP(receivedAt.Add(-100 * time.Millisecond)),
				Delay:            delay,
			},
		},
	}, receivedAt)

	stats := handler.GetDetailedStats()
	assert.Equal(t, uint64(1), stats.RTCPReceiverReportsReceived)
	assert.Equal(t, uint8(32), stats.RTCPRemoteFractionLost)
	assert.Equal(t, uint32(2), stats.RTCPRemotePacketsLost)
	assert.Equal(t, uint32(7), stats.RTCPRemoteJitter)
	assert.InDelta(t,
		(80 * time.Millisecond).Nanoseconds(),
		stats.RTCPRoundTripTime.Nanoseconds(),
		float64((2 * time.Millisecond).Nanoseconds()),
	)
}

func TestRTPHandlerClearsRTTWhenReceiverReportHasNoDelay(t *testing.T) {
	handler := newTestRTPHandler()
	handler.ssrc = 5678
	receivedAt := time.Unix(1_700_000_000, 0)
	delay := uint32((20 * time.Millisecond).Nanoseconds() * rtcpCompactUnit / rtcpNanoseconds)

	handler.recordRTCPPacket(&rtcp.ReceiverReport{
		Reports: []rtcp.ReceptionReport{
			{
				SSRC:             handler.ssrc,
				LastSenderReport: rtcpCompactNTP(receivedAt.Add(-100 * time.Millisecond)),
				Delay:            delay,
			},
		},
	}, receivedAt)
	require.NotZero(t, handler.GetDetailedStats().RTCPRoundTripTime)

	handler.recordRTCPPacket(&rtcp.ReceiverReport{
		Reports: []rtcp.ReceptionReport{
			{
				SSRC:             handler.ssrc,
				LastSenderReport: rtcpCompactNTP(receivedAt),
			},
		},
	}, receivedAt.Add(time.Second))

	assert.Zero(t, handler.GetDetailedStats().RTCPRoundTripTime)
}

func TestRTPHandlerRecordsCompoundRTCPPacketCount(t *testing.T) {
	handler := newTestRTPHandler()
	compound := rtcp.CompoundPacket{
		&rtcp.ReceiverReport{SSRC: 1},
		rtcp.NewCNAMESourceDescription(1, "test@example.com"),
	}

	assert.Equal(t, 2, handler.recordRTCPPacket(&compound, time.Now()))
}

func reserveUDPPortPair(t *testing.T) (int, int) {
	t.Helper()

	for attempt := 0; attempt < 100; attempt++ {
		rtpConn, err := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		})
		require.NoError(t, err)

		rtpPort := rtpConn.LocalAddr().(*net.UDPAddr).Port
		if rtpPort >= rtpMaxPort {
			require.NoError(t, rtpConn.Close())
			continue
		}
		rtcpConn, err := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: rtpPort + rtcpPortOffset,
		})
		if err == nil {
			require.NoError(t, rtcpConn.Close())
			require.NoError(t, rtpConn.Close())
			return rtpPort, rtpPort + rtcpPortOffset
		}
		require.NoError(t, rtpConn.Close())
	}

	t.Fatal("failed to reserve udp rtp/rtcp port pair")
	return 0, 0
}
