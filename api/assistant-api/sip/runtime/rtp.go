// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rapidaai/pkg/utils"
)

// RTPPacket represents an RTP packet
type RTPPacket struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CSRCCount      uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	CSRC           []uint32
	Payload        []byte
}

// RTPHandler manages RTP streams for SIP calls
// No WebSocket needed - audio goes directly over RTP/UDP
type RTPHandler struct {
	mu      sync.RWMutex
	writeMu sync.Mutex
	running atomic.Bool
	closed  atomic.Bool

	conn      *net.UDPConn
	rtcpConn  *net.UDPConn
	localAddr RTPAddress

	remoteAddr         *net.UDPAddr
	remoteRTCPAddr     *net.UDPAddr
	remoteRTCPSignaled bool
	symmetricRTP       bool
	portStats          *RTPPortStats

	// RTP state
	ssrc                   uint32
	sequenceNumber         uint16
	timestamp              uint32
	codec                  *Codec
	inputPacketizationTime time.Duration
	remoteSSRC             atomic.Uint32

	// Audio delivery
	inboundAudioSink      func(InboundAudioFrame)
	inboundAudioSinkReady chan struct{}
	inboundAudioSinkOnce  sync.Once
	inputJitter           *rtpInputJitterBuffer

	ctx              context.Context
	cancel           context.CancelFunc
	loops            sync.WaitGroup
	timeoutCloseOnce sync.Once

	mediaTimeoutCh       chan struct{}
	mediaTimeoutKick     chan struct{}
	mediaTimeoutStart    atomic.Int64
	mediaTimeoutInitial  atomic.Int64
	mediaTimeoutGeneral  atomic.Int64
	lastRTPReceivedAt    atomic.Int64
	lastAudioDeliveredAt atomic.Int64

	// Statistics
	packetsSent                 atomic.Uint64
	packetsReceived             atomic.Uint64
	packetsDelivered            atomic.Uint64
	packetsDropped              atomic.Uint64
	invalidPackets              atomic.Uint64
	bytesReceived               atomic.Uint64
	bytesSent                   atomic.Uint64
	rtcpReception               rtpRTCPReceptionStats
	rtcpPacketsSent             atomic.Uint64
	rtcpPacketsReceived         atomic.Uint64
	rtcpReportsSent             atomic.Uint64
	rtcpSenderReportsSent       atomic.Uint64
	rtcpReceiverReportsSent     atomic.Uint64
	rtcpSenderReportsReceived   atomic.Uint64
	rtcpReceiverReportsReceived atomic.Uint64
	firstPacketSeen             atomic.Bool
	onFirstPacket               func()
}

type RTPPortStats struct {
	portsInUse       atomic.Int64
	bindAttempts     atomic.Uint64
	bindFailures     atomic.Uint64
	rangeExhaustions atomic.Uint64
}

type remoteMediaAddress struct {
	remoteRTPAddress  RTPAddress
	remoteRTCPAddress RTPAddress
}

// NewRTPHandler creates a new RTP handler for direct audio transport
func NewRTPHandler(ctx context.Context, config *RTPConfig) (*RTPHandler, error) {
	if config == nil {
		return nil, NewSIPError(rtpNewHandlerOperation, "", errRTPInvalidConfig.Error(), errRTPConfigRequired)
	}
	resolvedConfig := *config
	config = &resolvedConfig
	if err := config.Validate(); err != nil {
		return nil, NewSIPError(rtpNewHandlerOperation, "", errRTPInvalidConfig.Error(), err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	handlerCtx, cancel := context.WithCancel(ctx)
	ip := net.ParseIP(config.LocalAddress.IP)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: config.LocalAddress.Port,
	}

	// Use "udp4" explicitly for IPv4 addresses to prevent Go from creating an
	// IPv6 socket (::) that won't receive IPv4 RTP packets on macOS/BSD.
	// On Linux, dual-stack sockets receive both, but macOS disables IPV6_V6ONLY
	// by default, so an IPv6 socket never sees IPv4 traffic.
	network := rtpNetworkUDP4
	if ip != nil && ip.To4() == nil {
		network = rtpNetworkUDP6
	}

	var conn *net.UDPConn
	var err error
	if config.portStats != nil {
		config.portStats.bindAttempts.Add(1)
	}
	if config.LocalAddress.Port > 0 {
		conn, err = net.ListenUDP(network, addr)
	} else {
		portCount := config.RTPPortRangeEnd - config.RTPPortRangeStart + 1
		portCountUint32, convertErr := utils.IntToUint32(portCount)
		if convertErr != nil {
			cancel()
			return nil, NewSIPError(rtpNewHandlerOperation, "", errRTPInvalidConfig.Error(), convertErr)
		}
		firstPort := config.RTPPortRangeStart
		var portOffsetSeed [4]byte
		if _, randomErr := cryptorand.Read(portOffsetSeed[:]); randomErr == nil {
			randomOffset, convertErr := utils.Uint32ToInt(binary.BigEndian.Uint32(portOffsetSeed[:]) % portCountUint32)
			if convertErr == nil {
				firstPort += randomOffset
			}
		}
		port := firstPort
		portsInUse := 0
		for i := 0; i < portCount; i++ {
			addr.Port = port
			conn, err = net.ListenUDP(network, addr)
			if err == nil {
				break
			}
			if !errors.Is(err, syscall.EADDRINUSE) {
				break
			}
			portsInUse++
			port++
			if port > config.RTPPortRangeEnd {
				port = config.RTPPortRangeStart
			}
		}
		if conn == nil && portsInUse == portCount {
			err = fmt.Errorf(rtpErrorPortRangeFormat,
				ErrRTPPortRangeExhausted,
				config.RTPPortRangeStart,
				config.RTPPortRangeEnd,
				portCount)
		}
	}
	if err != nil {
		if config.portStats != nil {
			config.portStats.bindFailures.Add(1)
			if errors.Is(err, ErrRTPPortRangeExhausted) {
				config.portStats.rangeExhaustions.Add(1)
			}
		}
		cancel()
		return nil, NewSIPError(rtpNewHandlerOperation, "", errRTPCreateSocket.Error(), err)
	}

	// Set buffer sizes
	_ = conn.SetReadBuffer(rtpReadBufferSize)
	_ = conn.SetWriteBuffer(rtpWriteBufferSize)

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	var rtcpConn *net.UDPConn
	if localAddr.Port < rtpMaxPort {
		rtcpAddr := &net.UDPAddr{
			IP:   localAddr.IP,
			Port: localAddr.Port + rtcpPortOffset,
		}
		if candidate, rtcpErr := net.ListenUDP(network, rtcpAddr); rtcpErr == nil {
			rtcpConn = candidate
			_ = rtcpConn.SetReadBuffer(rtcpReadBufferSize)
			_ = rtcpConn.SetWriteBuffer(rtcpWriteBufferSize)
		}
	}

	// Get codec from payload type or use default
	codec := GetCodecByPayloadType(config.PayloadType)
	if codec == nil {
		codec = &CodecPCMU
	}

	var ssrcSeed [4]byte
	if _, randomErr := cryptorand.Read(ssrcSeed[:]); randomErr != nil {
		_ = conn.Close()
		if rtcpConn != nil {
			_ = rtcpConn.Close()
		}
		cancel()
		return nil, NewSIPError(rtpNewHandlerOperation, "", errRTPCreateSSRC.Error(), randomErr)
	}
	ssrc := binary.BigEndian.Uint32(ssrcSeed[:])

	handler := &RTPHandler{
		conn:                   conn,
		rtcpConn:               rtcpConn,
		localAddr:              RTPAddress{IP: localAddr.IP.String(), Port: localAddr.Port},
		symmetricRTP:           config.SymmetricRTP,
		portStats:              config.portStats,
		ssrc:                   ssrc,
		codec:                  codec,
		inputPacketizationTime: config.PacketizationTime,
		inboundAudioSinkReady:  make(chan struct{}),
		inputJitter:            newRTPInputJitterBuffer(codec, config.PacketizationTime),
		ctx:                    handlerCtx,
		cancel:                 cancel,
		mediaTimeoutCh:         make(chan struct{}),
		mediaTimeoutKick:       make(chan struct{}, 1),
	}
	handler.mediaTimeoutInitial.Store(int64(config.MediaTimeoutInitial))
	handler.mediaTimeoutGeneral.Store(int64(config.MediaTimeout))
	if handler.portStats != nil {
		handler.portStats.portsInUse.Add(1)
	}
	handler.loops.Add(1)
	go func() {
		defer handler.loops.Done()
		handler.mediaTimeoutLoop()
	}()

	return handler, nil
}

// Start begins RTP processing
func (h *RTPHandler) Start() {
	if h.closed.Load() {
		return
	}
	if !h.running.CompareAndSwap(false, true) {
		return // Already running
	}
	if h.conn == nil {
		h.running.Store(false)
		return
	}

	// Send an initial silence packet immediately to "punch" the RTP path.
	// Some PBXes (Asterisk with direct_media) expect to see RTP traffic very
	// quickly after the call is bridged. Waiting for the 20ms send cycle
	// may be too slow.
	h.sendInitialSilence()

	h.loops.Add(1)
	go func() {
		defer h.loops.Done()
		h.receiveLoop()
	}()
	if h.rtcpConn != nil {
		h.loops.Add(2)
		go func() {
			defer h.loops.Done()
			h.receiveRTCPLoop()
		}()
		go func() {
			defer h.loops.Done()
			h.sendRTCPReportsLoop()
		}()
	}
}

// sendInitialSilence sends one RTP packet synchronously to open the media path.
func (h *RTPHandler) sendInitialSilence() {
	h.mu.RLock()
	remoteAddr := cloneUDPAddr(h.remoteAddr)
	codec := h.codec
	h.mu.RUnlock()

	if remoteAddr == nil {
		return
	}

	if codec == nil {
		codec = &CodecPCMU
	}
	samplesPerPacket := int(codec.ClockRate * 20 / 1000)
	chunk := createSilenceChunk(samplesPerPacket, codec)

	_ = h.WriteAudio(chunk)
}

// Stop releases all RTP resources. It is safe before Start, after Start, and
// after earlier Stop calls because setup and teardown paths share this method.
func (h *RTPHandler) Stop() error {
	if !h.closed.CompareAndSwap(false, true) {
		return nil
	}
	h.running.Store(false)

	if h.cancel != nil {
		h.cancel()
	}

	var err error
	if h.conn != nil {
		err = h.conn.Close()
	}
	if h.rtcpConn != nil {
		if closeErr := h.rtcpConn.Close(); err == nil {
			err = closeErr
		}
	}
	if h.portStats != nil {
		h.portStats.portsInUse.Add(-1)
	}

	h.loops.Wait()
	h.flushPendingInboundAudio()

	return err
}

func (h *RTPHandler) flushPendingInboundAudio() {
	h.mu.RLock()
	inputJitter := h.inputJitter
	h.mu.RUnlock()
	if inputJitter == nil {
		return
	}

	frames := inputJitter.flushPending()
	h.deliverInboundAudio(frames)
}

func (h *RTPHandler) MediaTimeout() <-chan struct{} {
	if h == nil {
		return nil
	}
	return h.mediaTimeoutCh
}

func (h *RTPHandler) EnableMediaTimeout(enabled bool) {
	if h == nil {
		return
	}
	if !enabled {
		h.mediaTimeoutStart.Store(0)
		h.kickMediaTimeoutLoop()
		return
	}
	h.SetMediaTimeout(time.Duration(h.mediaTimeoutInitial.Load()), time.Duration(h.mediaTimeoutGeneral.Load()))
}

func (h *RTPHandler) SetMediaTimeout(initial, general time.Duration) {
	if h == nil || h.closed.Load() {
		return
	}
	if initial <= 0 {
		initial = rtpMediaTimeoutInitial
	}
	if general <= 0 {
		general = rtpMediaTimeout
	}
	h.mediaTimeoutInitial.Store(int64(initial))
	h.mediaTimeoutGeneral.Store(int64(general))
	h.mediaTimeoutStart.Store(time.Now().UnixNano())
	h.kickMediaTimeoutLoop()
}

func (h *RTPHandler) kickMediaTimeoutLoop() {
	if h.mediaTimeoutKick == nil {
		return
	}
	select {
	case h.mediaTimeoutKick <- struct{}{}:
	default:
	}
}

func (h *RTPHandler) mediaTimeoutLoop() {
	if h.mediaTimeoutCh == nil || h.ctx == nil {
		return
	}

	timer := time.NewTimer(rtpMediaTimeoutDisabledPark)
	defer timer.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-h.mediaTimeoutKick:
		case <-timer.C:
		}

		startNano := h.mediaTimeoutStart.Load()
		if startNano <= 0 {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(rtpMediaTimeoutDisabledPark)
			continue
		}

		lastPacketNano := h.lastRTPReceivedAt.Load()
		initial := time.Duration(h.mediaTimeoutInitial.Load())
		general := time.Duration(h.mediaTimeoutGeneral.Load())
		if initial <= 0 {
			initial = rtpMediaTimeoutInitial
		}
		if general <= 0 {
			general = rtpMediaTimeout
		}

		deadline := time.Unix(0, startNano).Add(initial)
		if lastPacketNano > 0 {
			deadline = time.Unix(0, lastPacketNano).Add(general)
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			h.timeoutCloseOnce.Do(func() {
				close(h.mediaTimeoutCh)
			})
			return
		}

		next := remaining
		if next > general {
			next = general
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(next)
	}
}

// IsRunning returns whether the RTP handler is running
func (h *RTPHandler) IsRunning() bool {
	return h.running.Load()
}

// SetRemoteAddress sets the remote RTP endpoint used by the handler's UDP socket.
func (h *RTPHandler) SetRemoteAddress(address RTPAddress) {
	h.setRemoteMediaAddress(remoteMediaAddress{
		remoteRTPAddress: address,
	})
}

func (h *RTPHandler) setRemoteMediaAddress(address remoteMediaAddress) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	parsedRemoteRTPIPAddress := net.ParseIP(address.remoteRTPAddress.IP)
	h.remoteAddr = &net.UDPAddr{
		IP:   parsedRemoteRTPIPAddress,
		Port: address.remoteRTPAddress.Port,
	}
	h.remoteRTCPSignaled = false
	h.remoteRTCPAddr = nil
	remoteRTCPAddress := address.remoteRTCPAddress
	if !remoteRTCPAddress.Validate() {
		remoteRTCPAddress.IP = address.remoteRTPAddress.IP
	}
	if remoteRTCPAddress.Validate() {
		h.remoteRTCPAddr = &net.UDPAddr{
			IP:   net.ParseIP(remoteRTCPAddress.IP),
			Port: remoteRTCPAddress.Port,
		}
		h.remoteRTCPSignaled = true
		return
	}
	if address.remoteRTPAddress.Port > 0 && address.remoteRTPAddress.Port < rtpMaxPort {
		h.remoteRTCPAddr = &net.UDPAddr{
			IP:   append(net.IP(nil), parsedRemoteRTPIPAddress...),
			Port: address.remoteRTPAddress.Port + rtcpPortOffset,
		}
	}
}

// SetRemoteRTCPAddress sets the remote RTCP endpoint advertised by SDP.
func (h *RTPHandler) SetRemoteRTCPAddress(address RTPAddress) {
	if h == nil || !address.Validate() {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	h.remoteRTCPAddr = &net.UDPAddr{
		IP:   net.ParseIP(address.IP),
		Port: address.Port,
	}
	h.remoteRTCPSignaled = true
}

func (h *RTPHandler) SetSymmetricRTP(enabled bool) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.symmetricRTP = enabled
	h.mu.Unlock()
}

// GetRemoteAddr returns the remote RTP address
func (h *RTPHandler) GetRemoteAddr() *net.UDPAddr {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return cloneUDPAddr(h.remoteAddr)
}

func cloneUDPAddr(address *net.UDPAddr) *net.UDPAddr {
	if address == nil {
		return nil
	}
	clone := *address
	clone.IP = append(net.IP(nil), address.IP...)
	return &clone
}

// LocalAddress returns the local RTP address.
func (h *RTPHandler) LocalAddress() RTPAddress {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.localAddr
}

// LocalRTCPPort returns the local RTCP port, or zero when RTCP is disabled.
func (h *RTPHandler) LocalRTCPPort() int {
	if h == nil || h.rtcpConn == nil {
		return 0
	}
	if addr, ok := h.rtcpConn.LocalAddr().(*net.UDPAddr); ok {
		return addr.Port
	}
	return 0
}

// SetInboundAudioSink sets the sole consumer for ordered inbound RTP audio.
// The receive loop waits for the first non-nil sink before reading packets.
func (h *RTPHandler) SetInboundAudioSink(sink func(InboundAudioFrame)) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.inboundAudioSink = sink
	ready := h.inboundAudioSinkReady
	if sink != nil && ready == nil {
		ready = make(chan struct{})
		h.inboundAudioSinkReady = ready
	}
	h.mu.Unlock()
	if sink != nil {
		h.inboundAudioSinkOnce.Do(func() {
			close(ready)
		})
	}
}

// WriteAudio writes one complete encoded audio frame to the RTP transport.
func (h *RTPHandler) WriteAudio(audio []byte) error {
	if h == nil {
		return ErrRTPNotInitialized
	}
	if len(audio) == 0 {
		return nil
	}
	if h.closed.Load() || !h.running.Load() {
		return ErrRTPHandlerStopped
	}
	if h.ctx != nil {
		select {
		default:
		case <-h.ctx.Done():
			return ErrRTPHandlerStopped
		}
	}
	h.writeMu.Lock()
	defer h.writeMu.Unlock()

	h.mu.RLock()
	remoteAddr := cloneUDPAddr(h.remoteAddr)
	conn := h.conn
	h.mu.RUnlock()
	if conn == nil || remoteAddr == nil {
		return ErrRTPNotInitialized
	}

	if _, err := conn.WriteToUDP(h.serializeRTPPacket(h.createRTPPacket(audio)), remoteAddr); err != nil {
		return err
	}
	h.packetsSent.Add(1)
	h.bytesSent.Add(uint64(len(audio)))
	return nil
}

// GetCodec returns the codec used by this handler
func (h *RTPHandler) GetCodec() *Codec {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.codec
}

// SetCodec updates the codec used by this RTP handler.
// This is needed when the remote side answers with a different codec than
// what was initially offered (e.g., PCMA instead of PCMU).
func (h *RTPHandler) SetCodec(codec *Codec) {
	if codec == nil {
		return
	}
	h.mu.RLock()
	packetizationTime := h.inputPacketizationTime
	h.mu.RUnlock()
	h.SetInboundMediaFormat(codec, packetizationTime)
}

func (h *RTPHandler) SetInboundMediaFormat(codec *Codec, packetizationTime time.Duration) {
	if codec == nil {
		codec = &CodecPCMU
	}
	if packetizationTime < rtpMinPacketizationTime ||
		packetizationTime > rtpMaxPacketizationTime ||
		packetizationTime%time.Millisecond != 0 {
		packetizationTime = rtpDefaultPacketizationTime
	}
	h.mu.Lock()
	h.codec = codec
	h.inputPacketizationTime = packetizationTime
	if h.inputJitter == nil {
		h.inputJitter = newRTPInputJitterBuffer(codec, packetizationTime)
	} else {
		h.inputJitter.reset(codec, packetizationTime)
	}
	h.mu.Unlock()

	h.rtcpReception.resetRTP(codec.ClockRate)
}

func (h *RTPHandler) receiveLoop() {
	select {
	case <-h.ctx.Done():
		return
	case <-h.inboundAudioSinkReady:
	}

	buf := make([]byte, rtpPacketMaxSize+1)

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		if !h.running.Load() {
			return
		}

		h.mu.RLock()
		inputJitter := h.inputJitter
		h.mu.RUnlock()
		now := time.Now()
		deadline := now.Add(rtpDefaultPacketizationTime)
		if inputJitter != nil {
			frames := inputJitter.flushExpired(now)
			h.deliverInboundAudio(frames)
			if pendingDeadline := inputJitter.nextDeadline(); !pendingDeadline.IsZero() {
				deadline = pendingDeadline
			}
		}
		if deadlineErr := h.conn.SetReadDeadline(deadline); deadlineErr != nil {
			return
		}
		n, remoteAddr, err := h.conn.ReadFromUDP(buf)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		if n > rtpPacketMaxSize {
			h.packetsDropped.Add(1)
			h.invalidPackets.Add(1)
			continue
		}

		if n < rtpHeaderSize {
			h.packetsDropped.Add(1)
			h.invalidPackets.Add(1)
			continue
		}

		packet, err := h.parseRTPPacket(buf[:n])
		if err != nil {
			h.packetsDropped.Add(1)
			h.invalidPackets.Add(1)
			continue
		}

		if remoteAddr != nil {
			h.mu.Lock()
			if h.remoteAddr == nil {
				h.remoteAddr = remoteAddr
			} else if h.symmetricRTP &&
				(!h.remoteAddr.IP.Equal(remoteAddr.IP) || h.remoteAddr.Port != remoteAddr.Port) {
				h.remoteAddr = remoteAddr
			}
			h.mu.Unlock()
		}

		h.packetsReceived.Add(1)
		h.bytesReceived.Add(uint64(len(packet.Payload)))
		if !h.running.Load() {
			return
		}

		h.mu.RLock()
		codec := h.codec
		inputJitter = h.inputJitter
		inputPacketizationTime := h.inputPacketizationTime
		h.mu.RUnlock()
		if codec == nil {
			codec = &CodecPCMU
		}
		isAudio := packet.PayloadType == codec.PayloadType
		isTelephoneEvent := packet.PayloadType == CodecTelephoneEvent.PayloadType && len(packet.Payload) >= 4 && len(packet.Payload)%4 == 0
		if (!isAudio && !isTelephoneEvent) || len(packet.Payload) == 0 {
			h.packetsDropped.Add(1)
			h.invalidPackets.Add(1)
			continue
		}

		arrivedAt := time.Now()
		hadRTP := h.lastRTPReceivedAt.Load() != 0
		if isTelephoneEvent && hadRTP && h.remoteSSRC.Load() != packet.SSRC {
			continue
		}
		h.markInboundRTPReceived(arrivedAt)
		previousSSRC := h.remoteSSRC.Swap(packet.SSRC)
		if hadRTP && previousSSRC != packet.SSRC && inputJitter != nil {
			inputJitter.reset(codec, inputPacketizationTime)
		}
		h.rtcpReception.recordRTP(packet, codec.ClockRate, arrivedAt)
		var frames []InboundAudioFrame
		if inputJitter != nil {
			frames = inputJitter.push(packet, arrivedAt)
		} else if isAudio {
			frames = []InboundAudioFrame{{Audio: packet.Payload, ReceivedAt: arrivedAt}}
		}
		h.deliverInboundAudio(frames)
	}
}

func (h *RTPHandler) deliverInboundAudio(frames []InboundAudioFrame) {
	for _, frame := range frames {
		h.mu.RLock()
		sink := h.inboundAudioSink
		h.mu.RUnlock()
		if sink == nil {
			h.packetsDropped.Add(1)
			continue
		}
		sink(frame)
		h.packetsDelivered.Add(1)
		deliveredAt := time.Now()
		h.markInboundAudioDelivered(deliveredAt)
	}
}

func (h *RTPHandler) SetOnFirstPacket(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onFirstPacket = fn
}

func (h *RTPHandler) notifyFirstPacketReceived() {
	if !h.firstPacketSeen.CompareAndSwap(false, true) {
		return
	}
	h.mu.RLock()
	fn := h.onFirstPacket
	h.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func (h *RTPHandler) markInboundRTPReceived(receivedAt time.Time) {
	h.lastRTPReceivedAt.Store(receivedAt.UnixNano())
	h.notifyFirstPacketReceived()
	h.kickMediaTimeoutLoop()
}

func (h *RTPHandler) markInboundAudioDelivered(deliveredAt time.Time) {
	h.lastAudioDeliveredAt.Store(deliveredAt.UnixNano())
}

// createSilenceChunk creates a silence chunk for the codec
func createSilenceChunk(size int, codec *Codec) []byte {
	chunk := make([]byte, size)
	silenceValue := byte(0xFF) // μ-law silence
	if codec != nil && codec.Name == "PCMA" {
		silenceValue = 0xD5 // A-law silence
	}
	for i := range chunk {
		chunk[i] = silenceValue
	}
	return chunk
}

func (h *RTPHandler) parseRTPPacket(data []byte) (*RTPPacket, error) {
	if len(data) < rtpHeaderSize {
		return nil, errRTPPacketTooSmall
	}

	packet := &RTPPacket{
		Version:        (data[0] >> 6) & 0x03,
		Padding:        (data[0] & 0x20) != 0,
		Extension:      (data[0] & 0x10) != 0,
		CSRCCount:      data[0] & 0x0F,
		Marker:         (data[1] & 0x80) != 0,
		PayloadType:    data[1] & 0x7F,
		SequenceNumber: binary.BigEndian.Uint16(data[2:4]),
		Timestamp:      binary.BigEndian.Uint32(data[4:8]),
		SSRC:           binary.BigEndian.Uint32(data[8:12]),
	}

	if packet.Version != rtpVersion {
		return nil, fmt.Errorf(rtpErrorIntFormat, errRTPUnsupportedVersion, packet.Version)
	}

	headerLen := rtpHeaderSize + int(packet.CSRCCount)*4
	if len(data) < headerLen {
		return nil, fmt.Errorf(rtpErrorSizeHeaderFormat, errRTPPacketShortCSRCHeader, len(data), headerLen)
	}
	if packet.CSRCCount > 0 {
		packet.CSRC = make([]uint32, packet.CSRCCount)
		for index := range packet.CSRC {
			offset := rtpHeaderSize + index*4
			packet.CSRC[index] = binary.BigEndian.Uint32(data[offset : offset+4])
		}
	}

	if packet.Extension {
		if len(data) < headerLen+4 {
			return nil, errRTPPacketShortExtension
		}
		extLen := binary.BigEndian.Uint16(data[headerLen+2 : headerLen+4])
		headerLen += 4 + int(extLen)*4
		if len(data) < headerLen {
			return nil, fmt.Errorf(rtpErrorSizeHeaderFormat, errRTPPacketShortExtensionPayload, len(data), headerLen)
		}
	}

	payloadLen := len(data) - headerLen
	if packet.Padding {
		if payloadLen <= 0 {
			return nil, errRTPPaddingNoPayload
		}
		paddingLen := int(data[len(data)-1])
		if paddingLen == 0 || paddingLen > payloadLen {
			return nil, fmt.Errorf(rtpErrorIntFormat, errRTPInvalidPaddingLength, paddingLen)
		}
		payloadLen -= paddingLen
	}

	if payloadLen < 0 || headerLen+payloadLen > len(data) {
		return nil, errRTPInvalidPacketLength
	}

	packet.Payload = make([]byte, payloadLen)
	copy(packet.Payload, data[headerLen:headerLen+payloadLen])

	return packet, nil
}

func (h *RTPHandler) createRTPPacket(payload []byte) *RTPPacket {
	h.mu.Lock()
	defer h.mu.Unlock()

	packet := &RTPPacket{
		Version:        rtpVersion,
		PayloadType:    h.codec.PayloadType,
		SequenceNumber: h.sequenceNumber,
		Timestamp:      h.timestamp,
		SSRC:           h.ssrc,
		Payload:        payload,
	}

	h.sequenceNumber++
	payloadSize, err := utils.IntToUint32(len(payload))
	if err == nil {
		h.timestamp += payloadSize
	}

	return packet
}

func (h *RTPHandler) serializeRTPPacket(packet *RTPPacket) []byte {
	headerLen := 12 + len(packet.CSRC)*4
	data := make([]byte, headerLen+len(packet.Payload))

	data[0] = (packet.Version << 6)
	if packet.Padding {
		data[0] |= 0x20
	}
	if packet.Extension {
		data[0] |= 0x10
	}
	data[0] |= packet.CSRCCount & 0x0F

	data[1] = packet.PayloadType & 0x7F
	if packet.Marker {
		data[1] |= 0x80
	}

	binary.BigEndian.PutUint16(data[2:4], packet.SequenceNumber)
	binary.BigEndian.PutUint32(data[4:8], packet.Timestamp)
	binary.BigEndian.PutUint32(data[8:12], packet.SSRC)

	for i, csrc := range packet.CSRC {
		binary.BigEndian.PutUint32(data[12+i*4:16+i*4], csrc)
	}

	copy(data[headerLen:], packet.Payload)

	return data
}

// GetStats returns RTP statistics
func (h *RTPHandler) GetStats() (sent, received uint64) {
	return h.packetsSent.Load(), h.packetsReceived.Load()
}

// GetDetailedStats returns detailed RTP statistics
func (h *RTPHandler) GetDetailedStats() RTPStats {
	var packetsLost uint64
	var jitterDropped uint64
	var lateOrDuplicatePackets uint64
	var resyncDroppedPackets uint64
	var silenceSuppressionFrames uint64
	h.mu.RLock()
	inputJitter := h.inputJitter
	remoteRTCPPort := 0
	if h.remoteRTCPAddr != nil {
		remoteRTCPPort = h.remoteRTCPAddr.Port
	}
	h.mu.RUnlock()
	if inputJitter != nil {
		packetsLost = inputJitter.lostPackets()
		jitterDropped = inputJitter.droppedPackets()
		lateOrDuplicatePackets = inputJitter.lateOrDuplicatePackets()
		resyncDroppedPackets = inputJitter.resyncDroppedPackets()
		silenceSuppressionFrames = inputJitter.silenceSuppressionFrameCount()
	}
	lastRTPReceivedAt := time.Time{}
	if receivedAt := h.lastRTPReceivedAt.Load(); receivedAt > 0 {
		lastRTPReceivedAt = time.Unix(0, receivedAt)
	}
	lastAudioDeliveredAt := time.Time{}
	if deliveredAt := h.lastAudioDeliveredAt.Load(); deliveredAt > 0 {
		lastAudioDeliveredAt = time.Unix(0, deliveredAt)
	}
	rtcpReception := h.rtcpReception.snapshot()
	return RTPStats{
		PacketsSent:                 h.packetsSent.Load(),
		PacketsReceived:             h.packetsReceived.Load(),
		PacketsDelivered:            h.packetsDelivered.Load(),
		BytesSent:                   h.bytesSent.Load(),
		BytesReceived:               h.bytesReceived.Load(),
		PacketsLost:                 packetsLost,
		PacketsDropped:              h.packetsDropped.Load() + jitterDropped,
		LateOrDuplicatePackets:      lateOrDuplicatePackets,
		InvalidPackets:              h.invalidPackets.Load(),
		JitterBufferResyncDropped:   resyncDroppedPackets,
		SilenceSuppressionFrames:    silenceSuppressionFrames,
		LastRTPReceivedAt:           lastRTPReceivedAt,
		LastAudioDeliveredAt:        lastAudioDeliveredAt,
		RTCPEnabled:                 h.rtcpConn != nil,
		LocalRTCPPort:               h.LocalRTCPPort(),
		RemoteRTCPPort:              remoteRTCPPort,
		RTCPPacketsSent:             h.rtcpPacketsSent.Load(),
		RTCPPacketsReceived:         h.rtcpPacketsReceived.Load(),
		RTCPReportsSent:             h.rtcpReportsSent.Load(),
		RTCPSenderReportsSent:       h.rtcpSenderReportsSent.Load(),
		RTCPReceiverReportsSent:     h.rtcpReceiverReportsSent.Load(),
		RTCPSenderReportsReceived:   h.rtcpSenderReportsReceived.Load(),
		RTCPReceiverReportsReceived: h.rtcpReceiverReportsReceived.Load(),
		RTCPFractionLost:            rtcpReception.FractionLost,
		RTCPPacketsLost:             rtcpReception.PacketsLost,
		RTCPJitter:                  rtcpReception.Jitter,
		RTCPRemoteFractionLost:      rtcpReception.RemoteLoss,
		RTCPRemotePacketsLost:       rtcpReception.RemotePacketsLost,
		RTCPRemoteJitter:            rtcpReception.RemoteJitter,
		RTCPRoundTripTime:           rtcpReception.RoundTripTime,
		Jitter:                      rtcpReception.JitterDuration,
	}
}
