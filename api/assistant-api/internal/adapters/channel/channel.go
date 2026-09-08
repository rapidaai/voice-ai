// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package channel

import (
	"context"
	"sync"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	policychannel "github.com/rapidaai/pkg/channel"
)

// Envelope carries a packet together with the context it was sent from.
type Envelope struct {
	Ctx context.Context
	Pkt internal_type.Packet
}

type ChannelWriter interface {
	OnControl(Envelope)
	OnBootstrap(Envelope)
	OnIngress(Envelope)
	OnEgress(Envelope)
	OnData(Envelope)
	OnBackground(Envelope)
}

type ChannelReader interface {
	ControlChannel() *policychannel.Channel[Envelope]
	BootstrapChannel() *policychannel.Channel[Envelope]
	IngressChannel() *policychannel.Channel[Envelope]
	EgressChannel() *policychannel.Channel[Envelope]
	DataChannel() *policychannel.Channel[Envelope]
	BackgroundChannel() *policychannel.Channel[Envelope]
}

type ChannelFlusher interface {
	FlushControl() int
	FlushBootstrap() int
	FlushIngress() int
	FlushEgress() int
	FlushData() int
	FlushBackground() int
	FlushControlMatching(func(internal_type.Packet) bool) int
	FlushBootstrapMatching(func(internal_type.Packet) bool) int
	FlushIngressMatching(func(internal_type.Packet) bool) int
	FlushEgressMatching(func(internal_type.Packet) bool) int
	FlushDataMatching(func(internal_type.Packet) bool) int
	FlushBackgroundMatching(func(internal_type.Packet) bool) int
	FlushAll() int
}

type ChannelRunner interface {
	RunControl(context.Context, func(Envelope))
	RunBootstrap(context.Context, func(Envelope))
	RunIngress(context.Context, func(Envelope))
	PauseIngress(context.Context, func())
	RunEgress(context.Context, func(Envelope))
	RunData(context.Context, func(Envelope))
	RunBackground(context.Context, func(Envelope))
}

// RequestorChannelBus is the unified channel interface used by the requestor.
type RequestorChannelBus interface {
	ChannelWriter
	ChannelReader
	ChannelFlusher
	ChannelRunner
}

// RequestorChannels groups all dispatcher channels used by one requestor.
type RequestorChannels struct {
	// controlChannel is for urgent runtime control packets:
	// interruptions, turn-change, and other immediate control directives.
	controlChannel *policychannel.Channel[Envelope]

	// bootstrapCh is reserved for session initialization/bootstrap packets.
	// Use this channel only for connect-time setup flow.
	bootstrapCh *policychannel.Channel[Envelope]

	// ingressCh carries inbound user-side packets:
	// user audio/text and upstream processing packets (VAD/STT/EOS/tool result).
	ingressCh       *policychannel.Channel[Envelope]
	ingressPauseCh  chan struct{}
	ingressPausedCh chan struct{}
	ingressWriteMu  sync.Mutex

	// egressCh carries outbound assistant-side packets:
	// LLM deltas/done, TTS text/audio/end, and output error/control events.
	egressCh *policychannel.Channel[Envelope]

	// dataCh carries DB writes, recording, and lifecycle orchestration that does
	// not require the observer. Drained by the data dispatcher started at
	// NewGenericRequestor and runs for the entire session.
	dataCh *policychannel.Channel[Envelope]

	// backgroundCh is for observer-touching telemetry (events, metrics).
	// Drained by the dispatcher started after telemetry init completes.
	backgroundCh *policychannel.Channel[Envelope]
}

func NewRequestorChannels() *RequestorChannels {
	controlChannel, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(256),
		OverflowPolicy: policychannel.ReplaceOldestWhenFull,
	})
	if err != nil {
		panic(err)
	}
	bootstrapCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(512),
		OverflowPolicy: policychannel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	ingressCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(4096),
		OverflowPolicy: policychannel.ReplaceOldestWhenFull,
	}, policychannel.WithReplacementFilter(isRealtimeAudioEnvelope))
	if err != nil {
		panic(err)
	}
	egressCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(2048),
		OverflowPolicy: policychannel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	dataCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(2048),
		OverflowPolicy: policychannel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	backgroundCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(2048),
		OverflowPolicy: policychannel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	channels := &RequestorChannels{
		controlChannel:  controlChannel,
		bootstrapCh:     bootstrapCh,
		ingressCh:       ingressCh,
		ingressPauseCh:  make(chan struct{}),
		ingressPausedCh: make(chan struct{}),
		egressCh:        egressCh,
		dataCh:          dataCh,
		backgroundCh:    backgroundCh,
	}
	close(channels.ingressPausedCh)
	return channels
}

func isRealtimeAudioEnvelope(envelope Envelope) bool {
	if envelope.Pkt == nil {
		return false
	}
	switch envelope.Pkt.PacketName() {
	case internal_type.PacketNameUserAudioReceived,
		internal_type.PacketNameSpeechToTextAudio,
		internal_type.PacketNameDenoiseAudio,
		internal_type.PacketNameDenoisedAudio,
		internal_type.PacketNameVadAudio,
		internal_type.PacketNameEndOfSpeechAudio:
		return true
	default:
		return false
	}
}

func (c *RequestorChannels) ControlChannel() *policychannel.Channel[Envelope] {
	return c.controlChannel
}
func (c *RequestorChannels) BootstrapChannel() *policychannel.Channel[Envelope] {
	return c.bootstrapCh
}
func (c *RequestorChannels) IngressChannel() *policychannel.Channel[Envelope] {
	return c.ingressCh
}
func (c *RequestorChannels) EgressChannel() *policychannel.Channel[Envelope] {
	return c.egressCh
}
func (c *RequestorChannels) DataChannel() *policychannel.Channel[Envelope] {
	return c.dataCh
}
func (c *RequestorChannels) BackgroundChannel() *policychannel.Channel[Envelope] {
	return c.backgroundCh
}

// OnControl routes an envelope to the control channel.
// Keep enqueue policy in this layer (block/drop/timeout) so it can evolve
// without touching dispatch routing code.
func (c *RequestorChannels) OnControl(e Envelope) {
	_, _ = c.controlChannel.Send(context.Background(), e)
}

// OnBootstrap routes an envelope to the bootstrap channel.
func (c *RequestorChannels) OnBootstrap(e Envelope) {
	_, _ = c.bootstrapCh.Send(context.Background(), e)
}

// OnIngress routes an envelope to the ingress channel.
func (c *RequestorChannels) OnIngress(e Envelope) {
	if e.Ctx != nil && e.Ctx.Err() != nil {
		return
	}

	c.ingressWriteMu.Lock()
	defer c.ingressWriteMu.Unlock()

	select {
	case <-c.ingressPauseCh:
		return
	default:
	}

	_, _ = c.ingressCh.Send(context.Background(), e)
}

// OnEgress routes an envelope to the egress channel.
func (c *RequestorChannels) OnEgress(e Envelope) {
	_, _ = c.egressCh.Send(context.Background(), e)
}

// OnData routes an envelope to the data channel (DB writes, recording, lifecycle).
func (c *RequestorChannels) OnData(e Envelope) {
	_, _ = c.dataCh.Send(context.Background(), e)
}

// OnBackground routes an envelope to the background channel.
func (c *RequestorChannels) OnBackground(e Envelope) {
	_, _ = c.backgroundCh.Send(context.Background(), e)
}

func run(ctx context.Context, ch *policychannel.Channel[Envelope], onEnvelope func(Envelope)) {
	for {
		e, err := ch.Receive(ctx)
		if err != nil {
			return
		}
		onEnvelope(e)
	}
}

func (c *RequestorChannels) RunControl(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.controlChannel, onEnvelope)
}

func (c *RequestorChannels) RunBootstrap(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.bootstrapCh, onEnvelope)
}

func (c *RequestorChannels) RunIngress(ctx context.Context, onEnvelope func(Envelope)) {
	if c.ingressPausedCh != nil {
		c.ingressPausedCh = make(chan struct{})
		defer close(c.ingressPausedCh)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.ingressPauseCh:
			return
		case <-c.ingressCh.Ready():
			e, err := c.ingressCh.TryReceive()
			if err == nil {
				onEnvelope(e)
			}
		}
	}
}

func (c *RequestorChannels) PauseIngress(ctx context.Context, onPaused func()) {
	c.ingressWriteMu.Lock()
	if c.ingressPauseCh != nil {
		select {
		case <-c.ingressPauseCh:
		default:
			close(c.ingressPauseCh)
		}
	}
	c.FlushIngress()
	c.ingressWriteMu.Unlock()

	if c.ingressPausedCh != nil {
		select {
		case <-ctx.Done():
			return
		case <-c.ingressPausedCh:
		}
	}
	c.FlushIngress()
	if onPaused != nil {
		onPaused()
	}
}

func (c *RequestorChannels) RunEgress(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.egressCh, onEnvelope)
}

func (c *RequestorChannels) RunData(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.dataCh, onEnvelope)
}

func (c *RequestorChannels) RunBackground(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.backgroundCh, onEnvelope)
}

// FlushControl drains queued control packets and returns dropped count.
func (c *RequestorChannels) FlushControl() int {
	return c.controlChannel.Drain()
}

// FlushBootstrap drains queued bootstrap packets and returns dropped count.
func (c *RequestorChannels) FlushBootstrap() int {
	return c.bootstrapCh.Drain()
}

// FlushIngress drains queued ingress packets and returns dropped count.
func (c *RequestorChannels) FlushIngress() int {
	return c.ingressCh.Drain()
}

// FlushEgress drains queued egress packets and returns dropped count.
func (c *RequestorChannels) FlushEgress() int {
	return c.egressCh.Drain()
}

// FlushData drains queued data packets and returns dropped count.
func (c *RequestorChannels) FlushData() int {
	return c.dataCh.Drain()
}

// FlushBackground drains queued background packets and returns dropped count.
func (c *RequestorChannels) FlushBackground() int {
	return c.backgroundCh.Drain()
}

// FlushControlMatching drains matching queued control packets.
func (c *RequestorChannels) FlushControlMatching(match func(internal_type.Packet) bool) int {
	return c.controlChannel.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushBootstrapMatching drains matching queued bootstrap packets.
func (c *RequestorChannels) FlushBootstrapMatching(match func(internal_type.Packet) bool) int {
	return c.bootstrapCh.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushIngressMatching drains matching queued ingress packets.
func (c *RequestorChannels) FlushIngressMatching(match func(internal_type.Packet) bool) int {
	return c.ingressCh.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushEgressMatching drains matching queued egress packets.
func (c *RequestorChannels) FlushEgressMatching(match func(internal_type.Packet) bool) int {
	return c.egressCh.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushDataMatching drains matching queued data packets.
func (c *RequestorChannels) FlushDataMatching(match func(internal_type.Packet) bool) int {
	return c.dataCh.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushBackgroundMatching drains matching queued background packets.
func (c *RequestorChannels) FlushBackgroundMatching(match func(internal_type.Packet) bool) int {
	return c.backgroundCh.RemoveIf(func(e Envelope) bool {
		return match != nil && match(e.Pkt)
	})
}

// FlushAll drains all channels and returns total dropped packets.
func (c *RequestorChannels) FlushAll() int {
	return c.FlushControl() +
		c.FlushBootstrap() +
		c.FlushIngress() +
		c.FlushEgress() +
		c.FlushData() +
		c.FlushBackground()
}
