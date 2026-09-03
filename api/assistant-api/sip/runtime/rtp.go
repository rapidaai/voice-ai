// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rapidaai/pkg/commons"
)

// RTP constants
const (
	rtpVersion             = 2
	rtpHeaderSize          = 12
	rtpReadBufferSize      = 65536
	rtpWriteBufferSize     = 65536
	rtpPacketMaxSize       = 1500
	rtpPacketInterval      = 20 * time.Millisecond
	rtpInputPlayoutTimeout = rtpPacketInterval
	rtpMediaTimeoutInitial = 30 * time.Second
	rtpMediaTimeout        = 15 * time.Second

	// Audio channel buffer sizes
	rtpAudioInBufferSize  = 100
	rtpAudioOutBufferSize = 100
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
	logger  commons.Logger
	running atomic.Bool
	closed  atomic.Bool

	conn      *net.UDPConn
	localIP   string
	localPort int

	remoteAddr   *net.UDPAddr
	symmetricRTP bool
	portStats    *RTPPortStats

	// RTP state
	ssrc           uint32
	sequenceNumber uint16
	timestamp      uint32
	codec          *Codec

	// Audio channels
	audioInChan  chan []byte
	audioOutChan chan []byte
	inputJitter  *rtpInputJitterBuffer
	outputSource RTPFallbackAudioSource

	// flushAudioCh signals the sendLoop to discard all pending audio
	// (used on user interruption to silence stale frames immediately).
	flushAudioCh chan struct{}

	// codecVersion is bumped by SetCodec so the sendLoop can detect mid-call
	// codec changes and regenerate its pre-computed silence chunk.
	codecVersion uint32

	ctx              context.Context
	cancel           context.CancelFunc
	loops            sync.WaitGroup
	inputCloseOnce   sync.Once
	timeoutCloseOnce sync.Once

	mediaTimeoutCh      chan struct{}
	mediaTimeoutKick    chan struct{}
	mediaTimeoutStart   atomic.Int64
	mediaTimeoutInitial atomic.Int64
	mediaTimeoutGeneral atomic.Int64
	lastPacketTime      atomic.Int64

	// Statistics
	packetsSent     atomic.Uint64
	packetsReceived atomic.Uint64
	packetsDropped  atomic.Uint64
	bytesReceived   atomic.Uint64
	bytesSent       atomic.Uint64
	firstPacketSeen atomic.Bool
	onFirstPacket   func()
}

type RTPFallbackAudioSource func(frameSize int) []byte

type RTPPortStats struct {
	portsInUse       atomic.Int64
	bindAttempts     atomic.Uint64
	bindFailures     atomic.Uint64
	rangeExhaustions atomic.Uint64
}

// RTPConfig holds configuration for RTP handler
type RTPConfig struct {
	LocalIP     string
	LocalPort   int
	PayloadType uint8  // 0 = PCMU, 8 = PCMA
	ClockRate   uint32 // 8000 for G.711
	Logger      commons.Logger

	RTPPortRangeStart int
	RTPPortRangeEnd   int
	SymmetricRTP      bool

	MediaTimeoutInitial time.Duration
	MediaTimeout        time.Duration

	portStats *RTPPortStats
}

// Validate validates the RTP configuration
func (c *RTPConfig) Validate() error {
	if c.LocalIP == "" {
		return fmt.Errorf("local_ip is required")
	}
	if c.LocalPort < 0 || c.LocalPort > 65535 {
		return fmt.Errorf("invalid local_port: %d", c.LocalPort)
	}
	if c.LocalPort == 0 {
		if c.RTPPortRangeStart <= 0 || c.RTPPortRangeEnd <= 0 {
			return fmt.Errorf("rtp port range is required when local_port is not specified")
		}
		if c.RTPPortRangeStart > c.RTPPortRangeEnd {
			return fmt.Errorf("rtp_port_range_start must be less than or equal to rtp_port_range_end")
		}
		if c.RTPPortRangeStart > 65535 || c.RTPPortRangeEnd > 65535 {
			return fmt.Errorf("rtp port range must be between 1 and 65535")
		}
	}
	if c.ClockRate == 0 {
		c.ClockRate = 8000 // Default to 8kHz
	}
	if c.MediaTimeoutInitial <= 0 {
		c.MediaTimeoutInitial = rtpMediaTimeoutInitial
	}
	if c.MediaTimeout <= 0 {
		c.MediaTimeout = rtpMediaTimeout
	}
	return nil
}

// NewRTPHandler creates a new RTP handler for direct audio transport
func NewRTPHandler(ctx context.Context, config *RTPConfig) (*RTPHandler, error) {
	if config == nil {
		return nil, NewSIPError("NewRTPHandler", "", "invalid configuration", fmt.Errorf("rtp config is required"))
	}
	if err := config.Validate(); err != nil {
		return nil, NewSIPError("NewRTPHandler", "", "invalid configuration", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	handlerCtx, cancel := context.WithCancel(ctx)

	ip := net.ParseIP(config.LocalIP)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: config.LocalPort,
	}

	// Use "udp4" explicitly for IPv4 addresses to prevent Go from creating an
	// IPv6 socket (::) that won't receive IPv4 RTP packets on macOS/BSD.
	// On Linux, dual-stack sockets receive both, but macOS disables IPV6_V6ONLY
	// by default, so an IPv6 socket never sees IPv4 traffic.
	network := "udp4"
	if ip != nil && ip.To4() == nil {
		network = "udp6"
	}

	var conn *net.UDPConn
	var err error
	if config.portStats != nil {
		config.portStats.bindAttempts.Add(1)
	}
	if config.LocalPort > 0 {
		conn, err = net.ListenUDP(network, addr)
	} else {
		portCount := config.RTPPortRangeEnd - config.RTPPortRangeStart + 1
		firstPort := rand.Intn(portCount) + config.RTPPortRangeStart
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
			err = fmt.Errorf("%w in range %d-%d (%d ports tried)",
				ErrRTPPortRangeExhausted,
				config.RTPPortRangeStart,
				config.RTPPortRangeEnd,
				portCount)
			if config.Logger != nil {
				config.Logger.Warnw("RTP port range exhausted",
					"range_start", config.RTPPortRangeStart,
					"range_end", config.RTPPortRangeEnd,
					"ports_tried", portCount)
			}
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
		return nil, NewSIPError("NewRTPHandler", "", "failed to create RTP socket", err)
	}

	// Set buffer sizes
	if err := conn.SetReadBuffer(rtpReadBufferSize); err != nil && config.Logger != nil {
		config.Logger.Warnw("Failed to set RTP read buffer size", "error", err)
	}
	if err := conn.SetWriteBuffer(rtpWriteBufferSize); err != nil && config.Logger != nil {
		config.Logger.Warnw("Failed to set RTP write buffer size", "error", err)
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Get codec from payload type or use default
	codec := GetCodecByPayloadType(config.PayloadType)
	if codec == nil {
		codec = &CodecPCMU
	}

	handler := &RTPHandler{
		logger:           config.Logger,
		conn:             conn,
		localIP:          localAddr.IP.String(),
		localPort:        localAddr.Port,
		symmetricRTP:     config.SymmetricRTP,
		portStats:        config.portStats,
		ssrc:             rand.Uint32(),
		codec:            codec,
		audioInChan:      make(chan []byte, rtpAudioInBufferSize),
		audioOutChan:     make(chan []byte, rtpAudioOutBufferSize),
		inputJitter:      newRTPInputJitterBuffer(codec),
		flushAudioCh:     make(chan struct{}, 1),
		ctx:              handlerCtx,
		cancel:           cancel,
		mediaTimeoutCh:   make(chan struct{}),
		mediaTimeoutKick: make(chan struct{}, 1),
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

	if config.Logger != nil {
		config.Logger.Debugw("RTP socket bound",
			"local_addr", localAddr.String(),
			"range_start", config.RTPPortRangeStart,
			"range_end", config.RTPPortRangeEnd,
			"symmetric_rtp", config.SymmetricRTP)
	}

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

	// Log the actual socket address the OS assigned (important: 0.0.0.0 vs specific IP)
	if h.logger != nil {
		h.logger.Infow("RTP Start() called",
			"conn_local_addr", h.conn.LocalAddr().String(),
			"handler_local_ip", h.localIP,
			"handler_local_port", h.localPort,
			"remote_addr", fmt.Sprintf("%v", h.remoteAddr),
			"codec", h.codec.Name,
			"ssrc", h.ssrc)
	}

	// Send an initial silence packet immediately to "punch" the RTP path.
	// Some PBXes (Asterisk with direct_media) expect to see RTP traffic very
	// quickly after the call is bridged. Waiting for the 20ms send cycle
	// may be too slow.
	h.sendInitialSilence()

	h.loops.Add(2)
	go func() {
		defer h.loops.Done()
		h.receiveLoop()
	}()
	go func() {
		defer h.loops.Done()
		h.sendLoop()
	}()

	if h.logger != nil {
		h.logger.Infow("RTP handler started — sendLoop and receiveLoop launched",
			"local_addr", fmt.Sprintf("%s:%d", h.localIP, h.localPort),
			"codec", h.codec.Name)
	}
}

// sendInitialSilence sends the first silence RTP packet synchronously to
// "punch" the RTP path immediately, then returns. The sendLoop goroutine
// will take over and keep sending silence every 20ms until real audio arrives.
func (h *RTPHandler) sendInitialSilence() {
	h.mu.RLock()
	remoteAddr := h.remoteAddr
	h.mu.RUnlock()

	if remoteAddr == nil {
		if h.logger != nil {
			h.logger.Warnw("sendInitialSilence: remoteAddr is nil — no RTP will be sent until remote address is set")
		}
		return
	}

	samplesPerPacket := int(h.codec.ClockRate * 20 / 1000)
	chunk := h.createSilenceChunk(samplesPerPacket)
	packet := h.createRTPPacket(chunk)
	data := h.serializeRTPPacket(packet)

	if h.logger != nil {
		h.logger.Debugw("sendInitialSilence: sending RTP",
			"conn_local_addr", h.conn.LocalAddr().String(),
			"dest_addr", remoteAddr.String(),
			"dest_ip", remoteAddr.IP.String(),
			"dest_port", remoteAddr.Port,
			"packet_bytes", len(data),
			"payload_bytes", len(chunk),
			"seq", packet.SequenceNumber,
			"ssrc", packet.SSRC)
	}

	n, err := h.sendPacket(data, remoteAddr)
	if err != nil {
		if h.logger != nil {
			h.logger.Warnw("sendInitialSilence: send FAILED",
				"error", err,
				"dest", remoteAddr.String(),
				"conn_local", h.conn.LocalAddr().String())
		}
		return
	}

	h.packetsSent.Add(1)
	h.bytesSent.Add(uint64(len(chunk)))
	if h.logger != nil {
		h.logger.Debugw("sendInitialSilence: sent",
			"bytes_written", n,
			"dest", remoteAddr.String(),
			"conn_local", h.conn.LocalAddr().String(),
			"seq", packet.SequenceNumber,
			"payload_size", len(chunk))
	}
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
	if h.portStats != nil {
		h.portStats.portsInUse.Add(-1)
	}

	h.loops.Wait()
	h.closeInboundChannel()

	if h.logger != nil {
		sent, received := h.GetStats()
		h.logger.Debugw("RTP handler stopped",
			"packets_sent", sent,
			"packets_received", received)
	}

	return err
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

	const disabledPark = time.Hour
	timer := time.NewTimer(disabledPark)
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
			timer.Reset(disabledPark)
			continue
		}

		lastPacketNano := h.lastPacketTime.Load()
		initial := time.Duration(h.mediaTimeoutInitial.Load())
		general := time.Duration(h.mediaTimeoutGeneral.Load())
		if initial <= 0 {
			initial = rtpMediaTimeoutInitial
		}
		if general <= 0 {
			general = rtpMediaTimeout
		}

		deadline := time.Unix(0, startNano).Add(initial)
		timeout := initial
		if lastPacketNano > 0 {
			deadline = time.Unix(0, lastPacketNano).Add(general)
			timeout = general
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			if h.logger != nil {
				h.logger.Infow("RTP media timeout",
					"packets_received", h.packetsReceived.Load(),
					"timeout", timeout)
			}
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

func (h *RTPHandler) closeInboundChannel() {
	h.inputCloseOnce.Do(func() {
		if h.audioInChan != nil {
			close(h.audioInChan)
		}
	})
}

// IsRunning returns whether the RTP handler is running
func (h *RTPHandler) IsRunning() bool {
	return h.running.Load()
}

// SetRemoteAddr sets the remote RTP endpoint used by the handler's UDP socket.
func (h *RTPHandler) SetRemoteAddr(ip string, port int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	parsedIP := net.ParseIP(ip)
	h.remoteAddr = &net.UDPAddr{
		IP:   parsedIP,
		Port: port,
	}

	if h.logger != nil {
		h.logger.Infow("RTP remote address set",
			"input_ip", ip,
			"parsed_ip", fmt.Sprintf("%v", parsedIP),
			"port", port,
			"resolved_addr", h.remoteAddr.String(),
			"is_ipv4", parsedIP != nil && parsedIP.To4() != nil)
	}
}

func (h *RTPHandler) SetSymmetricRTP(enabled bool) {
	if h == nil {
		return
	}
	h.mu.Lock()
	changed := h.symmetricRTP != enabled
	h.symmetricRTP = enabled
	h.mu.Unlock()
	if changed && h.logger != nil {
		h.logger.Infow("RTP symmetric mode updated", "enabled", enabled)
	}
}

// GetRemoteAddr returns the remote RTP address
func (h *RTPHandler) GetRemoteAddr() *net.UDPAddr {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.remoteAddr
}

// LocalAddr returns the local RTP address
func (h *RTPHandler) LocalAddr() (string, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.localIP, h.localPort
}

// AudioIn returns the channel for received audio
func (h *RTPHandler) AudioIn() <-chan []byte {
	return h.audioInChan
}

// EnqueueAudio queues outbound audio without exposing RTP channel lifecycle.
// Producers never own channel close; stopped or full queues are returned as errors.
func (h *RTPHandler) EnqueueAudio(audio []byte) error {
	if h == nil {
		return ErrRTPNotInitialized
	}
	if len(audio) == 0 {
		return nil
	}
	if h.closed.Load() || !h.running.Load() {
		return ErrRTPHandlerStopped
	}
	if h.audioOutChan == nil {
		return ErrRTPNotInitialized
	}
	if h.ctx != nil {
		select {
		case <-h.ctx.Done():
			return ErrRTPHandlerStopped
		default:
		}
	}
	if h.ctx != nil {
		select {
		case h.audioOutChan <- audio:
			return nil
		case <-h.ctx.Done():
			return ErrRTPHandlerStopped
		default:
			return ErrRTPOutputQueueFull
		}
	}
	select {
	case h.audioOutChan <- audio:
		return nil
	default:
		return ErrRTPOutputQueueFull
	}
}

// FlushAudioOut signals the sendLoop to discard all pending audio.
// Used on user interruption to silence stale frames immediately.
func (h *RTPHandler) FlushAudioOut() {
	select {
	case h.flushAudioCh <- struct{}{}:
	default:
	}
}

func (h *RTPHandler) SetFallbackAudioSource(source RTPFallbackAudioSource) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.outputSource = source
	h.mu.Unlock()
}

func (h *RTPHandler) ClearFallbackAudioSource() {
	h.SetFallbackAudioSource(nil)
}

// GetCodec returns the codec used by this handler
func (h *RTPHandler) GetCodec() *Codec {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.codec
}

// SetCodec updates the codec used by this RTP handler.
// This is needed when the remote side answers with a different codec than
// what was initially offered (e.g., PCMA instead of PCMU). The payload type
// and clock rate of outgoing packets are updated immediately; the silence
// pattern is also adjusted (0xFF for PCMU, 0xD5 for PCMA).
// The codecVersion counter is bumped so the sendLoop regenerates its
// pre-computed silence chunk on the next iteration.
func (h *RTPHandler) SetCodec(codec *Codec) {
	if codec == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	old := h.codec
	h.codec = codec
	h.codecVersion++
	if h.inputJitter == nil {
		h.inputJitter = newRTPInputJitterBuffer(codec)
	} else {
		h.inputJitter.reset(codec)
	}
	if h.logger != nil {
		h.logger.Infow("RTP codec updated",
			"old_codec", old.Name,
			"new_codec", codec.Name,
			"payload_type", codec.PayloadType,
			"clock_rate", codec.ClockRate,
			"codec_version", h.codecVersion)
	}
}

func (h *RTPHandler) receiveLoop() {
	if h.logger != nil {
		h.logger.Infow("RTP receiveLoop started",
			"local_addr", h.conn.LocalAddr().String(),
			"codec", h.codec.Name)
	}

	buf := make([]byte, rtpPacketMaxSize)

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		// Exit early if handler is stopped — avoids sending to closed channels
		if !h.running.Load() {
			return
		}

		var n int
		var remoteAddr *net.UDPAddr
		var err error

		if deadlineErr := h.conn.SetReadDeadline(time.Now().Add(rtpInputPlayoutTimeout)); deadlineErr != nil {
			continue
		}
		n, remoteAddr, err = h.conn.ReadFromUDP(buf)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if h.writeInboundAudioPayloads(h.flushInboundRTPOnTimeout(), 0) {
					return
				}
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				continue
			}
			// Connection closed or other error
			if h.running.Load() && h.logger != nil {
				// h.logger.Warnw("RTP receive error", "error", err)
			}
			continue
		}

		if n < rtpHeaderSize {
			if h.logger != nil {
				h.logger.Warnw("RTP packet too small", "size", n)
			}
			continue
		}

		packet, err := h.parseRTPPacket(buf[:n])
		if err != nil {
			if h.logger != nil {
				h.logger.Warnw("Failed to parse RTP packet", "error", err)
			}
			continue
		}

		if remoteAddr != nil {
			h.mu.Lock()
			if h.remoteAddr == nil {
				h.remoteAddr = remoteAddr
				if h.logger != nil {
					h.logger.Info("RTP: Auto-detected remote address", "addr", remoteAddr.String())
				}
			} else if h.symmetricRTP &&
				(!h.remoteAddr.IP.Equal(remoteAddr.IP) || h.remoteAddr.Port != remoteAddr.Port) {
				previousAddr := h.remoteAddr
				h.remoteAddr = remoteAddr
				if h.logger != nil {
					h.logger.Infow("RTP symmetric remote address updated",
						"previous_addr", previousAddr.String(),
						"packet_addr", remoteAddr.String())
				}
			}
			h.mu.Unlock()
		}

		// Update statistics
		h.packetsReceived.Add(1)
		h.bytesReceived.Add(uint64(len(packet.Payload)))
		// running state and context together with the send.
		if !h.running.Load() {
			return
		}
		audioPayloads, acceptedAudio := h.processInboundRTPPacket(packet)
		if acceptedAudio {
			h.markInboundAudioPacketReceived()
		}
		if h.writeInboundAudioPayloads(audioPayloads, packet.SequenceNumber) {
			return
		}
	}
}

func (h *RTPHandler) processInboundRTPPacket(packet *RTPPacket) ([][]byte, bool) {
	if !h.isInboundAudioPayload(packet) {
		h.recordDroppedInboundRTPPacket()
		return nil, false
	}
	h.mu.RLock()
	inputJitter := h.inputJitter
	h.mu.RUnlock()
	if inputJitter == nil {
		return [][]byte{packet.Payload}, true
	}
	return inputJitter.push(packet), true
}

func (h *RTPHandler) isInboundAudioPayload(packet *RTPPacket) bool {
	h.mu.RLock()
	codec := h.codec
	h.mu.RUnlock()
	if codec == nil {
		codec = &CodecPCMU
	}
	return packet.PayloadType == codec.PayloadType
}

func (h *RTPHandler) recordDroppedInboundRTPPacket() {
	h.packetsDropped.Add(1)
}

func (h *RTPHandler) flushInboundRTPOnTimeout() [][]byte {
	h.mu.RLock()
	inputJitter := h.inputJitter
	h.mu.RUnlock()
	if inputJitter == nil {
		return nil
	}
	return inputJitter.flushOnPlayoutTimeout()
}

func (h *RTPHandler) writeInboundAudioPayloads(audioPayloads [][]byte, sequenceNumber uint16) bool {
	for _, payload := range audioPayloads {
		select {
		case <-h.ctx.Done():
			return true
		case h.audioInChan <- payload:
		default:
			h.recordDroppedInboundRTPPacket()
			if h.logger != nil {
				h.logger.Warnw("RTP: Audio input channel full, dropping packet", "seq", sequenceNumber)
			}
		}
	}
	return false
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

func (h *RTPHandler) markInboundAudioPacketReceived() {
	h.lastPacketTime.Store(time.Now().UnixNano())
	h.notifyFirstPacketReceived()
	h.kickMediaTimeoutLoop()
}

func (h *RTPHandler) sendLoop() {
	// Calculate samples per packet based on codec (20ms packets)
	h.mu.RLock()
	samplesPerPacket := int(h.codec.ClockRate * 20 / 1000) // e.g., 160 bytes for PCMU at 8kHz
	lastCodecVersion := h.codecVersion
	h.mu.RUnlock()

	// Pre-create silence chunk (μ-law silence is 0xFF, PCMA silence is 0xD5)
	silenceChunk := h.createSilenceChunk(samplesPerPacket)

	var pendingAudio []byte
	// First sendLoop packet should go out immediately (sendInitialSilence
	// already sent packet #1, this will send packet #2 without delay).
	nextSendTime := time.Now()

	for {
		// Check for context cancellation
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		// If the codec changed (e.g., via re-INVITE), regenerate the
		// silence chunk so it uses the correct silence byte pattern.
		h.mu.RLock()
		cv := h.codecVersion
		h.mu.RUnlock()
		if cv != lastCodecVersion {
			lastCodecVersion = cv
			h.mu.RLock()
			samplesPerPacket = int(h.codec.ClockRate * 20 / 1000)
			h.mu.RUnlock()
			silenceChunk = h.createSilenceChunk(samplesPerPacket)
		}

		// Collect pending audio (non-blocking) or handle flush signal
		select {
		case <-h.flushAudioCh:
			// Interruption: discard all queued audio immediately
			pendingAudio = nil
			// Drain any remaining audio in the channel
			for {
				select {
				case <-h.audioOutChan:
				default:
					goto collectDone
				}
			}
		case audio, ok := <-h.audioOutChan:
			if ok {
				pendingAudio = append(pendingAudio, audio...)
			}
		default:
		}
	collectDone:

		// Wait until next send time with precision
		now := time.Now()
		if sleepDuration := nextSendTime.Sub(now); sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}

		// Schedule next send immediately to minimize drift
		nextSendTime = nextSendTime.Add(rtpPacketInterval)

		// If we've fallen behind, reset timing (don't try to catch up)
		if time.Now().After(nextSendTime) {
			nextSendTime = time.Now().Add(rtpPacketInterval)
		}

		h.mu.RLock()
		remoteAddr := h.remoteAddr
		h.mu.RUnlock()

		if remoteAddr == nil {
			continue
		}

		// Get exactly ONE chunk: queued audio, fallback media, or silence.
		chunk := h.nextOutputChunk(&pendingAudio, samplesPerPacket, silenceChunk)

		packet := h.createRTPPacket(chunk)
		data := h.serializeRTPPacket(packet)

		_, err := h.sendPacket(data, remoteAddr)
		if err != nil {
			if h.running.Load() && h.logger != nil {
				h.logger.Warnw("RTP sendLoop: send FAILED",
					"error", err,
					"dest", remoteAddr.String(),
					"conn_local", h.conn.LocalAddr().String())
			}
			continue
		}

		// Update statistics
		h.packetsSent.Add(1)
		h.bytesSent.Add(uint64(len(chunk)))

	}
}

// createSilenceChunk creates a silence chunk for the codec
func (h *RTPHandler) createSilenceChunk(size int) []byte {
	chunk := make([]byte, size)
	silenceValue := byte(0xFF) // μ-law silence
	if h.codec.Name == "PCMA" {
		silenceValue = 0xD5 // A-law silence
	}
	for i := range chunk {
		chunk[i] = silenceValue
	}
	return chunk
}

// getAudioChunk extracts or creates an audio chunk for sending
func (h *RTPHandler) getAudioChunk(pendingAudio *[]byte, size int, silenceChunk []byte) []byte {
	if len(*pendingAudio) >= size {
		chunk := (*pendingAudio)[:size]
		*pendingAudio = (*pendingAudio)[size:]
		return chunk
	}

	if len(*pendingAudio) > 0 {
		// Partial audio - pad with silence
		silenceValue := silenceChunk[0]
		chunk := make([]byte, size)
		copy(chunk, *pendingAudio)
		for i := len(*pendingAudio); i < size; i++ {
			chunk[i] = silenceValue
		}
		*pendingAudio = nil
		return chunk
	}

	// No audio - return silence
	return silenceChunk
}

func (h *RTPHandler) nextOutputChunk(pendingAudio *[]byte, size int, silenceChunk []byte) []byte {
	if len(*pendingAudio) > 0 {
		return h.getAudioChunk(pendingAudio, size, silenceChunk)
	}
	h.mu.RLock()
	outputSource := h.outputSource
	h.mu.RUnlock()
	if outputSource != nil {
		if chunk := outputSource(size); len(chunk) > 0 {
			return h.fitAudioChunk(chunk, size, silenceChunk)
		}
	}
	return silenceChunk
}

func (h *RTPHandler) fitAudioChunk(chunk []byte, size int, silenceChunk []byte) []byte {
	if len(chunk) == size {
		return chunk
	}
	if len(chunk) > size {
		return chunk[:size]
	}
	fitted := make([]byte, size)
	copy(fitted, chunk)
	copy(fitted[len(chunk):], silenceChunk[len(chunk):])
	return fitted
}

func (h *RTPHandler) parseRTPPacket(data []byte) (*RTPPacket, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("packet too small")
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

	if packet.Version != 2 {
		return nil, fmt.Errorf("unsupported RTP version: %d", packet.Version)
	}

	headerLen := 12 + int(packet.CSRCCount)*4

	if packet.Extension && len(data) >= headerLen+4 {
		extLen := binary.BigEndian.Uint16(data[headerLen+2 : headerLen+4])
		headerLen += 4 + int(extLen)*4
	}

	payloadLen := len(data) - headerLen
	if packet.Padding && payloadLen > 0 {
		paddingLen := int(data[len(data)-1])
		payloadLen -= paddingLen
	}

	if payloadLen < 0 || headerLen+payloadLen > len(data) {
		return nil, fmt.Errorf("invalid packet length")
	}

	packet.Payload = make([]byte, payloadLen)
	copy(packet.Payload, data[headerLen:headerLen+payloadLen])

	return packet, nil
}

func (h *RTPHandler) sendPacket(data []byte, remoteAddr *net.UDPAddr) (int, error) {
	return h.conn.WriteToUDP(data, remoteAddr)
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
	h.timestamp += uint32(len(payload))

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
	h.mu.RLock()
	inputJitter := h.inputJitter
	h.mu.RUnlock()
	if inputJitter != nil {
		packetsLost = inputJitter.lostPackets()
		jitterDropped = inputJitter.droppedPackets()
	}
	return RTPStats{
		PacketsSent:     h.packetsSent.Load(),
		PacketsReceived: h.packetsReceived.Load(),
		BytesSent:       h.bytesSent.Load(),
		BytesReceived:   h.bytesReceived.Load(),
		PacketsLost:     packetsLost,
		PacketsDropped:  h.packetsDropped.Load() + jitterDropped,
	}
}
