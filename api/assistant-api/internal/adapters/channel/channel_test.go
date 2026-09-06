package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func TestRequestorChannels_OverflowEvictsOldestPacket(tester *testing.T) {
	channels := NewRequestorChannels()
	capacity := cap(channels.IngressChannel())
	for index := 0; index < capacity; index++ {
		var packet internal_type.Packet = internal_type.UserAudioReceivedPacket{ContextID: "queued", Audio: []byte{1, 2}}
		if index%2 == 0 {
			packet = internal_type.SpeechToTextPacket{ContextID: "queued", Script: "retained"}
		}
		channels.OnIngress(Envelope{Ctx: context.Background(), Pkt: packet})
	}
	channels.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "latest", Audio: []byte{3, 4}}})
	if len(channels.IngressChannel()) != capacity {
		tester.Fatal("overflow must replace only the oldest packet")
	}
	for index := 0; index < capacity-1; index++ {
		if packet := (<-channels.IngressChannel()).Pkt; packet.ContextId() != "queued" {
			tester.Fatal("unexpected retained packet")
		}
	}
	if packet := (<-channels.IngressChannel()).Pkt; packet.ContextId() != "latest" {
		tester.Fatal("latest packet was not retained")
	}
}

func TestRequestorChannels_CancelledIngressIsNotOverload(tester *testing.T) {
	channels := NewRequestorChannels()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	channels.OnIngress(Envelope{Ctx: ctx, Pkt: internal_type.UserAudioReceivedPacket{}})
	if len(channels.IngressChannel()) != 0 {
		tester.Fatal("cancelled input should not be admitted")
	}
}

func recvEnvelope(t *testing.T, ch chan Envelope) Envelope {
	t.Helper()
	select {
	case env := <-ch:
		return env
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for envelope")
		return Envelope{}
	}
}

func TestRequestorChannels_OnRoutesToExpectedChannel(t *testing.T) {
	chs := NewRequestorChannels()
	ctx := context.Background()

	controlEnv := Envelope{Ctx: ctx, Pkt: internal_type.TurnChangePacket{ContextID: "ctrl"}}
	bootstrapEnv := Envelope{Ctx: ctx, Pkt: internal_type.InitializeAssistantPacket{ContextID: "boot"}}
	ingressEnv := Envelope{Ctx: ctx, Pkt: internal_type.UserTextReceivedPacket{ContextID: "in", Text: "hello"}}
	egressEnv := Envelope{Ctx: ctx, Pkt: internal_type.TextToSpeechTextPacket{ContextID: "out", Text: "hi"}}
	backgroundEnv := Envelope{Ctx: ctx, Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "bg"}}

	chs.OnControl(controlEnv)
	chs.OnBootstrap(bootstrapEnv)
	chs.OnIngress(ingressEnv)
	chs.OnEgress(egressEnv)
	chs.OnBackground(backgroundEnv)

	gotControl := recvEnvelope(t, chs.ControlChannel())
	gotBootstrap := recvEnvelope(t, chs.BootstrapChannel())
	gotIngress := recvEnvelope(t, chs.IngressChannel())
	gotEgress := recvEnvelope(t, chs.EgressChannel())
	gotBackground := recvEnvelope(t, chs.BackgroundChannel())

	if gotControl.Pkt.ContextId() != "ctrl" {
		t.Fatalf("unexpected control packet context: %s", gotControl.Pkt.ContextId())
	}
	if gotBootstrap.Pkt.ContextId() != "boot" {
		t.Fatalf("unexpected bootstrap packet context: %s", gotBootstrap.Pkt.ContextId())
	}
	if gotIngress.Pkt.ContextId() != "in" {
		t.Fatalf("unexpected ingress packet context: %s", gotIngress.Pkt.ContextId())
	}
	if gotEgress.Pkt.ContextId() != "out" {
		t.Fatalf("unexpected egress packet context: %s", gotEgress.Pkt.ContextId())
	}
	if gotBackground.Pkt.ContextId() != "bg" {
		t.Fatalf("unexpected background packet context: %s", gotBackground.Pkt.ContextId())
	}
}

func TestNewRequestorChannels_DefaultCapacities(t *testing.T) {
	chs := NewRequestorChannels()

	if cap(chs.ControlChannel()) != 256 {
		t.Fatalf("unexpected controlChannel cap: %d", cap(chs.ControlChannel()))
	}
	if cap(chs.BootstrapChannel()) != 512 {
		t.Fatalf("unexpected bootstrapCh cap: %d", cap(chs.BootstrapChannel()))
	}
	if cap(chs.IngressChannel()) != 4096 {
		t.Fatalf("unexpected ingressCh cap: %d", cap(chs.IngressChannel()))
	}
	if cap(chs.EgressChannel()) != 2048 {
		t.Fatalf("unexpected egressCh cap: %d", cap(chs.EgressChannel()))
	}
	if cap(chs.BackgroundChannel()) != 2048 {
		t.Fatalf("unexpected backgroundCh cap: %d", cap(chs.BackgroundChannel()))
	}
}

func TestRequestorChannels_FlushSpecificChannel(t *testing.T) {
	chs := NewRequestorChannels()

	for i := 0; i < 3; i++ {
		chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "ingress"}})
	}
	for i := 0; i < 2; i++ {
		chs.OnEgress(Envelope{Ctx: context.Background(), Pkt: internal_type.TextToSpeechTextPacket{ContextID: "egress"}})
	}

	droppedIngress := chs.FlushIngress()
	if droppedIngress != 3 {
		t.Fatalf("expected 3 dropped ingress packets, got %d", droppedIngress)
	}
	if len(chs.IngressChannel()) != 0 {
		t.Fatalf("expected ingress channel to be empty, got len=%d", len(chs.IngressChannel()))
	}
	if len(chs.EgressChannel()) != 2 {
		t.Fatalf("expected egress channel to be untouched, got len=%d", len(chs.EgressChannel()))
	}
	if chs.FlushIngress() != 0 {
		t.Fatalf("expected second ingress flush to drop 0 packets")
	}
}

func TestRequestorChannels_FlushAll(t *testing.T) {
	chs := NewRequestorChannels()

	chs.OnControl(Envelope{Ctx: context.Background(), Pkt: internal_type.TurnChangePacket{ContextID: "c1"}})
	chs.OnControl(Envelope{Ctx: context.Background(), Pkt: internal_type.TurnChangePacket{ContextID: "c2"}})

	chs.OnBootstrap(Envelope{Ctx: context.Background(), Pkt: internal_type.InitializeAssistantPacket{ContextID: "b1"}})

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "i1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "i2"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "i3"}})

	chs.OnEgress(Envelope{Ctx: context.Background(), Pkt: internal_type.TextToSpeechTextPacket{ContextID: "e1"}})

	chs.OnBackground(Envelope{Ctx: context.Background(), Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "g1"}})
	chs.OnBackground(Envelope{Ctx: context.Background(), Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "g2"}})

	dropped := chs.FlushAll()
	if dropped != 9 {
		t.Fatalf("expected flushAll to drop 9 packets, got %d", dropped)
	}

	if len(chs.ControlChannel()) != 0 || len(chs.BootstrapChannel()) != 0 || len(chs.IngressChannel()) != 0 || len(chs.EgressChannel()) != 0 || len(chs.BackgroundChannel()) != 0 {
		t.Fatalf("expected all channels to be empty after flushAll")
	}

	if chs.FlushAll() != 0 {
		t.Fatalf("expected second flushAll to drop 0 packets")
	}
}

func TestRequestorChannels_OnIngressWhenFull_EvictsOldestPacket(t *testing.T) {
	chs := &RequestorChannels{
		controlChannel: make(chan Envelope, 1),
		bootstrapCh:    make(chan Envelope, 1),
		ingressCh:      make(chan Envelope, 2),
		egressCh:       make(chan Envelope, 1),
		backgroundCh:   make(chan Envelope, 1),
	}

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "old-1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "old-2"}})

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "new"}})

	if got := len(chs.IngressChannel()); got != 2 {
		t.Fatalf("expected bounded ingress queue, got %d", got)
	}

	env := recvEnvelope(t, chs.IngressChannel())
	if env.Pkt.ContextId() != "old-2" {
		t.Fatalf("expected second-oldest packet, got %s", env.Pkt.ContextId())
	}
	if got := recvEnvelope(t, chs.IngressChannel()).Pkt.ContextId(); got != "new" {
		t.Fatalf("expected latest packet, got %s", got)
	}
}

func TestRequestorChannels_OnIngressOverflow_ThenRefillsInOrder(t *testing.T) {
	chs := &RequestorChannels{
		controlChannel: make(chan Envelope, 1),
		bootstrapCh:    make(chan Envelope, 1),
		ingressCh:      make(chan Envelope, 2),
		egressCh:       make(chan Envelope, 1),
		backgroundCh:   make(chan Envelope, 1),
	}

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "old-1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "old-2"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "latest"}})
	_ = recvEnvelope(t, chs.IngressChannel())
	_ = recvEnvelope(t, chs.IngressChannel())
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "new-1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "new-2"}})

	if got := len(chs.IngressChannel()); got != 2 {
		t.Fatalf("expected ingress len=2 after refill, got %d", got)
	}

	first := recvEnvelope(t, chs.IngressChannel())
	second := recvEnvelope(t, chs.IngressChannel())
	if first.Pkt.ContextId() != "new-1" || second.Pkt.ContextId() != "new-2" {
		t.Fatalf("expected new-1,new-2 order; got %s,%s", first.Pkt.ContextId(), second.Pkt.ContextId())
	}
}

func TestRequestorChannels_OnEgressWhenFull_BlocksUntilConsumerDrains(t *testing.T) {
	chs := &RequestorChannels{
		controlChannel: make(chan Envelope, 1),
		bootstrapCh:    make(chan Envelope, 1),
		ingressCh:      make(chan Envelope, 1),
		egressCh:       make(chan Envelope, 1),
		backgroundCh:   make(chan Envelope, 1),
	}

	chs.OnEgress(Envelope{Ctx: context.Background(), Pkt: internal_type.TextToSpeechTextPacket{ContextID: "first"}})

	done := make(chan struct{})
	go func() {
		chs.OnEgress(Envelope{Ctx: context.Background(), Pkt: internal_type.TextToSpeechTextPacket{ContextID: "second"}})
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("expected OnEgress to block while channel is full")
	case <-time.After(50 * time.Millisecond):
		// expected blocked
	}

	_ = recvEnvelope(t, chs.EgressChannel()) // drain first

	select {
	case <-done:
		// expected unblocked
	case <-time.After(time.Second):
		t.Fatalf("expected OnEgress to unblock after consumer drain")
	}

	got := recvEnvelope(t, chs.EgressChannel())
	if got.Pkt.ContextId() != "second" {
		t.Fatalf("expected second packet after unblock, got %s", got.Pkt.ContextId())
	}
}

func TestRequestorChannels_OnIngress_ConcurrentWritersNoDeadlock(t *testing.T) {
	chs := &RequestorChannels{
		controlChannel: make(chan Envelope, 1),
		bootstrapCh:    make(chan Envelope, 1),
		ingressCh:      make(chan Envelope, 4),
		egressCh:       make(chan Envelope, 1),
		backgroundCh:   make(chan Envelope, 1),
	}

	var wg sync.WaitGroup
	workers := 64
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			chs.OnIngress(Envelope{
				Ctx: context.Background(),
				Pkt: internal_type.UserTextReceivedPacket{ContextID: "c", Text: "payload"},
			})
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("concurrent OnIngress writers did not complete")
	}

	if got := len(chs.IngressChannel()); got != cap(chs.IngressChannel()) {
		t.Fatalf("expected full ingress queue after concurrent writes, got %d", got)
	}
}

func TestRequestorChannels_PauseIngressRejectsConcurrentWriters(t *testing.T) {
	chs := NewRequestorChannels()
	start := make(chan struct{})

	var writers sync.WaitGroup
	for index := 0; index < 256; index++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			chs.OnIngress(Envelope{
				Ctx: context.Background(),
				Pkt: internal_type.UserTextReceivedPacket{ContextID: "concurrent"},
			})
		}()
	}

	paused := make(chan struct{})
	go func() {
		<-start
		chs.PauseIngress(context.Background(), func() {
			close(paused)
		})
	}()

	close(start)
	writers.Wait()
	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ingress pause")
	}

	if got := len(chs.IngressChannel()); got != 0 {
		t.Fatalf("expected no ingress packets after pause, got %d", got)
	}
}

func TestRequestorChannels_PauseIngress_CallbackRunsAfterIngressHandlerReturns(t *testing.T) {
	chs := NewRequestorChannels()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handlerEntered := make(chan struct{})
	releaseHandler := make(chan struct{})
	paused := make(chan struct{})

	go chs.RunIngress(ctx, func(e Envelope) {
		close(handlerEntered)
		<-releaseHandler
	})

	chs.OnIngress(Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserTextReceivedPacket{ContextID: "active"},
	})

	select {
	case <-handlerEntered:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for ingress handler")
	}

	go chs.PauseIngress(ctx, func() {
		close(paused)
	})

	select {
	case <-paused:
		t.Fatalf("pause callback ran before active ingress handler returned")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseHandler)

	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for pause callback")
	}

	chs.OnIngress(Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserTextReceivedPacket{ContextID: "dropped"},
	})
	if got := len(chs.IngressChannel()); got != 0 {
		t.Fatalf("expected ingress to drop packets after pause, got len=%d", got)
	}
}

func TestRequestorChannels_RunControl_ProcessesAndStopsOnCancel(t *testing.T) {
	chs := NewRequestorChannels()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled := make(chan Envelope, 1)
	finished := make(chan struct{})

	go func() {
		chs.RunControl(ctx, func(e Envelope) {
			handled <- e
		})
		close(finished)
	}()

	chs.OnControl(Envelope{Ctx: context.Background(), Pkt: internal_type.TurnChangePacket{ContextID: "ctrl"}})

	select {
	case got := <-handled:
		if got.Pkt.ContextId() != "ctrl" {
			t.Fatalf("unexpected handled packet context: %s", got.Pkt.ContextId())
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for control packet to be handled")
	}

	cancel()

	select {
	case <-finished:
		// expected
	case <-time.After(time.Second):
		t.Fatalf("RunControl did not stop after context cancellation")
	}
}
