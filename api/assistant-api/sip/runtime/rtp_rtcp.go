package sip_runtime

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/rapidaai/pkg/utils"
)

type rtpRTCPReceptionStats struct {
	mu sync.Mutex

	ssrc             uint32
	clockRate        uint32
	started          bool
	baseSequence     uint32
	highestSequence  uint32
	receivedPackets  uint32
	reportExpected   uint32
	reportReceived   uint32
	previousSequence uint16
	sequenceCycles   uint32

	previousTransit int64
	jitter          float64

	lastSenderReport           uint32
	lastSenderReportReceivedAt time.Time
	lastRemoteFractionLost     uint8
	lastRemotePacketsLost      uint32
	lastRemoteJitter           uint32
	roundTripTime              time.Duration
}

type rtpRTCPReceptionSnapshot struct {
	FractionLost      uint8
	PacketsLost       uint32
	Jitter            uint32
	JitterDuration    time.Duration
	RemoteLoss        uint8
	RemotePacketsLost uint32
	RemoteJitter      uint32
	RoundTripTime     time.Duration
}

func (h *RTPHandler) receiveRTCPLoop() {
	buf := make([]byte, rtcpReadBufferSize)
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}
		if !h.running.Load() {
			return
		}
		if err := h.rtcpConn.SetReadDeadline(time.Now().Add(rtcpReadTimeout)); err != nil {
			continue
		}
		n, remoteAddr, err := h.rtcpConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				continue
			}
			continue
		}
		if remoteAddr != nil {
			h.mu.Lock()
			if h.remoteRTCPAddr == nil || (h.symmetricRTP && !h.remoteRTCPSignaled) {
				h.remoteRTCPAddr = cloneUDPAddr(remoteAddr)
			}
			h.mu.Unlock()
		}
		packets, err := rtcp.Unmarshal(buf[:n])
		if err != nil {
			continue
		}
		now := time.Now()
		receivedRTCPPackets := 0
		for _, packet := range packets {
			receivedRTCPPackets += h.recordRTCPPacket(packet, now)
		}
		if packetCount, convertErr := utils.IntToUint64(receivedRTCPPackets); convertErr == nil {
			h.rtcpPacketsReceived.Add(packetCount)
		}
	}
}

func (h *RTPHandler) sendRTCPReportsLoop() {
	ticker := time.NewTicker(rtcpReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case now := <-ticker.C:
			_ = h.sendRTCPReport(now)
		}
	}
}

func (h *RTPHandler) sendRTCPReport(now time.Time) error {
	if h == nil || h.rtcpConn == nil {
		return nil
	}

	h.mu.RLock()
	remoteRTCPAddr := cloneUDPAddr(h.remoteRTCPAddr)
	localAddress := h.localAddr
	ssrc := h.ssrc
	timestamp := h.timestamp
	h.mu.RUnlock()
	if remoteRTCPAddr == nil {
		return nil
	}

	reports := make([]rtcp.ReceptionReport, 0, 1)
	if remoteSSRC := h.remoteSSRC.Load(); remoteSSRC != 0 {
		reports = append(reports, h.rtcpReception.receptionReport(remoteSSRC, now))
	}

	var compoundPacket rtcp.CompoundPacket
	packetsSent := h.packetsSent.Load()
	if packetsSent > 0 {
		packetCount, _ := utils.Uint64ToUint32(min(packetsSent, rtcpMaxUint32))
		octetCount, _ := utils.Uint64ToUint32(min(h.bytesSent.Load(), rtcpMaxUint32))
		compoundPacket = rtcp.CompoundPacket{
			&rtcp.SenderReport{
				SSRC:        ssrc,
				NTPTime:     rtcpNTP(now),
				RTPTime:     timestamp,
				PacketCount: packetCount,
				OctetCount:  octetCount,
				Reports:     reports,
			},
			rtcp.NewCNAMESourceDescription(ssrc, fmt.Sprintf(rtcpCNAMEFormat, ssrc, localAddress.IP)),
		}
	} else {
		compoundPacket = rtcp.CompoundPacket{
			&rtcp.ReceiverReport{
				SSRC:    ssrc,
				Reports: reports,
			},
			rtcp.NewCNAMESourceDescription(ssrc, fmt.Sprintf(rtcpCNAMEFormat, ssrc, localAddress.IP)),
		}
	}

	data, err := compoundPacket.Marshal()
	if err != nil {
		return err
	}
	if _, err := h.rtcpConn.WriteToUDP(data, remoteRTCPAddr); err != nil {
		return err
	}
	if packetCount, convertErr := utils.IntToUint64(len(compoundPacket)); convertErr == nil {
		h.rtcpPacketsSent.Add(packetCount)
	}
	h.rtcpReportsSent.Add(1)
	if packetsSent > 0 {
		h.rtcpSenderReportsSent.Add(1)
	} else {
		h.rtcpReceiverReportsSent.Add(1)
	}
	return nil
}

func (h *RTPHandler) recordRTCPPacket(packet rtcp.Packet, receivedAt time.Time) int {
	switch typedPacket := packet.(type) {
	case *rtcp.CompoundPacket:
		receivedRTCPPackets := 0
		for _, innerPacket := range *typedPacket {
			receivedRTCPPackets += h.recordRTCPPacket(innerPacket, receivedAt)
		}
		return receivedRTCPPackets
	case *rtcp.SenderReport:
		h.rtcpSenderReportsReceived.Add(1)
		h.rtcpReception.recordSenderReport(typedPacket, receivedAt)
		return 1
	case *rtcp.ReceiverReport:
		h.rtcpReceiverReportsReceived.Add(1)
		h.rtcpReception.recordReceiverReport(typedPacket, h.ssrc, receivedAt)
		return 1
	}
	return 1
}

func (s *rtpRTCPReceptionStats) recordRTP(packet *RTPPacket, clockRate uint32, arrivedAt time.Time) {
	if packet == nil {
		return
	}
	if clockRate == 0 {
		clockRate = rtpDefaultClockRate
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.ssrc != packet.SSRC {
		s.resetRTPLocked(packet.SSRC, clockRate)
		sequence := uint32(packet.SequenceNumber)
		s.started = true
		s.baseSequence = sequence
		s.highestSequence = sequence
		s.previousSequence = packet.SequenceNumber
		s.receivedPackets = 1
		s.previousTransit = rtpTransit(packet, clockRate, arrivedAt)
		return
	}

	if packet.SequenceNumber < s.previousSequence && s.previousSequence-packet.SequenceNumber > rtpSequenceRolloverThreshold {
		s.sequenceCycles += rtpSequenceCycle
	}
	extendedSequence := s.sequenceCycles | uint32(packet.SequenceNumber)
	isLatePacket := extendedSequence <= s.highestSequence
	if extendedSequence > s.highestSequence {
		s.highestSequence = extendedSequence
	}
	s.previousSequence = packet.SequenceNumber
	s.receivedPackets++
	if isLatePacket {
		return
	}

	transit := rtpTransit(packet, clockRate, arrivedAt)
	transitDelta := transit - s.previousTransit
	if transitDelta < 0 {
		transitDelta = -transitDelta
	}
	s.previousTransit = transit
	s.jitter += (float64(transitDelta) - s.jitter) / 16
}

func (s *rtpRTCPReceptionStats) resetRTP(clockRate uint32) {
	if clockRate == 0 {
		clockRate = rtpDefaultClockRate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetRTPLocked(0, clockRate)
}

func (s *rtpRTCPReceptionStats) resetRTPLocked(ssrc uint32, clockRate uint32) {
	s.ssrc = ssrc
	s.clockRate = clockRate
	s.started = false
	s.baseSequence = 0
	s.highestSequence = 0
	s.receivedPackets = 0
	s.reportExpected = 0
	s.reportReceived = 0
	s.previousSequence = 0
	s.sequenceCycles = 0
	s.previousTransit = 0
	s.jitter = 0
	s.lastSenderReport = 0
	s.lastSenderReportReceivedAt = time.Time{}
	s.lastRemoteFractionLost = 0
	s.lastRemotePacketsLost = 0
	s.lastRemoteJitter = 0
	s.roundTripTime = 0
}

func rtpTransit(packet *RTPPacket, clockRate uint32, arrivedAt time.Time) int64 {
	clockRateInt := utils.Uint32ToInt64(clockRate)
	arrivalSeconds := arrivedAt.Unix() * clockRateInt
	arrivalNanoseconds, err := utils.IntToUint64(arrivedAt.Nanosecond())
	if err != nil {
		return 0
	}
	arrivalFraction, err := utils.Uint64ToInt64(arrivalNanoseconds * uint64(clockRate) / rtcpNanosecondsUint64)
	if err != nil {
		return 0
	}
	packetTimestamp := utils.Uint32ToInt64(packet.Timestamp)
	return arrivalSeconds + arrivalFraction - packetTimestamp
}

func (s *rtpRTCPReceptionStats) recordSenderReport(report *rtcp.SenderReport, receivedAt time.Time) {
	if report == nil || report.NTPTime == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var ntpBytes [8]byte
	binary.BigEndian.PutUint64(ntpBytes[:], report.NTPTime)
	s.lastSenderReport = binary.BigEndian.Uint32(ntpBytes[2:6])
	s.lastSenderReportReceivedAt = receivedAt
}

func (s *rtpRTCPReceptionStats) recordReceiverReport(report *rtcp.ReceiverReport, localSSRC uint32, receivedAt time.Time) {
	if report == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, receptionReport := range report.Reports {
		if receptionReport.SSRC != localSSRC {
			continue
		}
		s.lastRemoteFractionLost = receptionReport.FractionLost
		s.lastRemotePacketsLost = receptionReport.TotalLost
		s.lastRemoteJitter = receptionReport.Jitter
		s.roundTripTime = 0
		if receptionReport.LastSenderReport == 0 || receptionReport.Delay == 0 {
			continue
		}
		rttUnits := rtcpCompactNTP(receivedAt) - receptionReport.LastSenderReport - receptionReport.Delay
		durationUnits := utils.Uint32ToInt64(rttUnits)
		s.roundTripTime = time.Duration(durationUnits) * time.Second / rtcpCompactUnit
	}
}

func (s *rtpRTCPReceptionStats) receptionReport(ssrc uint32, now time.Time) rtcp.ReceptionReport {
	s.mu.Lock()
	defer s.mu.Unlock()

	expected := uint32(0)
	if s.started && s.highestSequence >= s.baseSequence {
		expected = s.highestSequence - s.baseSequence + 1
	}
	packetsLost := uint32(0)
	if expected > s.receivedPackets {
		packetsLost = expected - s.receivedPackets
	}

	expectedInterval := expected - s.reportExpected
	receivedInterval := s.receivedPackets - s.reportReceived
	lostInterval := uint32(0)
	if expectedInterval > receivedInterval {
		lostInterval = expectedInterval - receivedInterval
	}
	fractionLost := uint8(0)
	if expectedInterval > 0 && lostInterval > 0 {
		fractionLost, _ = utils.Uint32ToUint8(min((lostInterval<<8)/expectedInterval, rtcpMaxUint8))
	}
	s.reportExpected = expected
	s.reportReceived = s.receivedPackets

	delay := uint32(0)
	if !s.lastSenderReportReceivedAt.IsZero() {
		delayUnits := now.Sub(s.lastSenderReportReceivedAt).Nanoseconds() * rtcpCompactUnit / rtcpNanoseconds
		if delayUnits > rtcpMaxUint32Int64 {
			delay = ^uint32(0)
		} else if delayUnits > 0 {
			delay, _ = utils.Int64ToUint32(delayUnits)
		}
	}

	return rtcp.ReceptionReport{
		SSRC:               ssrc,
		FractionLost:       fractionLost,
		TotalLost:          packetsLost,
		LastSequenceNumber: s.highestSequence,
		Jitter:             uint32(s.jitter),
		LastSenderReport:   s.lastSenderReport,
		Delay:              delay,
	}
}

func (s *rtpRTCPReceptionStats) snapshot() rtpRTCPReceptionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	expected := uint32(0)
	if s.started && s.highestSequence >= s.baseSequence {
		expected = s.highestSequence - s.baseSequence + 1
	}
	packetsLost := uint32(0)
	if expected > s.receivedPackets {
		packetsLost = expected - s.receivedPackets
	}
	fractionLost := uint8(0)
	if expected > 0 && packetsLost > 0 {
		fractionLost, _ = utils.Uint32ToUint8(min((packetsLost<<8)/expected, rtcpMaxUint8))
	}

	jitter := uint32(s.jitter)
	return rtpRTCPReceptionSnapshot{
		FractionLost:      fractionLost,
		PacketsLost:       packetsLost,
		Jitter:            jitter,
		JitterDuration:    rtpJitterDuration(jitter, s.clockRate),
		RemoteLoss:        s.lastRemoteFractionLost,
		RemotePacketsLost: s.lastRemotePacketsLost,
		RemoteJitter:      s.lastRemoteJitter,
		RoundTripTime:     s.roundTripTime,
	}
}

func rtpJitterDuration(jitter uint32, clockRate uint32) time.Duration {
	if jitter == 0 {
		return 0
	}
	if clockRate == 0 {
		clockRate = rtpDefaultClockRate
	}
	nanoseconds := uint64(jitter) * rtcpNanosecondsUint64 / uint64(clockRate)
	duration, err := utils.Uint64ToInt64(nanoseconds)
	if err != nil {
		return 0
	}
	return time.Duration(duration)
}

func rtcpNTP(t time.Time) uint64 {
	if t.Before(time.Unix(0, 0)) {
		return 0
	}
	seconds, err := utils.Int64ToUint64(t.Unix())
	if err != nil {
		return 0
	}
	fraction, err := utils.IntToUint64(t.Nanosecond())
	if err != nil {
		return 0
	}
	seconds += rtcpNTPUnixOffset
	fraction = fraction * rtcpNTPFractionUnit / rtcpNanosecondsUint64
	return seconds<<32 | fraction
}

func rtcpCompactNTP(t time.Time) uint32 {
	var ntpBytes [8]byte
	binary.BigEndian.PutUint64(ntpBytes[:], rtcpNTP(t))
	return binary.BigEndian.Uint32(ntpBytes[2:6])
}
