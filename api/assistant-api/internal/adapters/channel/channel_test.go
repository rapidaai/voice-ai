package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	policychannel "github.com/rapidaai/pkg/channel"
	"github.com/stretchr/testify/require"
)

func TestRequestorChannels_OverflowEvictsOldestAudioPacket(tester *testing.T) {
	channels := NewRequestorChannels()
	capacity := channels.IngressChannel().Capacity()
	for index := 0; index < capacity; index++ {
		var packet internal_type.Packet = internal_type.UserAudioReceivedPacket{ContextID: "queued", Audio: []byte{1, 2}}
		if index%2 == 0 {
			packet = internal_type.SpeechToTextPacket{ContextID: "queued", Script: "retained"}
		}
		channels.OnIngress(Envelope{Ctx: context.Background(), Pkt: packet})
	}
	channels.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "latest", Audio: []byte{3, 4}}})
	if channels.IngressChannel().Len() != capacity {
		tester.Fatal("overflow must replace only the oldest packet")
	}
	for index := 0; index < capacity-1; index++ {
		packet := recvEnvelope(tester, channels.IngressChannel()).Pkt
		if packet.ContextId() != "queued" {
			tester.Fatal("unexpected retained packet")
		}
		if index == 0 && packet.PacketName() != internal_type.PacketNameSpeechToText {
			tester.Fatalf("oldest non-audio packet was evicted: %s", packet.PacketName())
		}
	}
	if packet := recvEnvelope(tester, channels.IngressChannel()).Pkt; packet.ContextId() != "latest" {
		tester.Fatal("latest packet was not retained")
	}
}

func TestRequestorChannels_CancelledIngressIsNotOverload(tester *testing.T) {
	channels := NewRequestorChannels()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	channels.OnIngress(Envelope{Ctx: ctx, Pkt: internal_type.UserAudioReceivedPacket{}})
	if channels.IngressChannel().Len() != 0 {
		tester.Fatal("cancelled input should not be admitted")
	}
}

func TestRequestorChannels_ControlOverflowRetainsNewestPacket(t *testing.T) {
	channels := newRequestorChannelsWithIngressCapacity(t, 1)
	channels.OnControl(Envelope{Ctx: context.Background(), Pkt: internal_type.TurnChangePacket{ContextID: "first"}})
	channels.OnControl(Envelope{Ctx: context.Background(), Pkt: internal_type.TurnChangePacket{ContextID: "second"}})

	if channels.ControlChannel().Len() != 1 {
		t.Fatalf("expected bounded control queue, got %d", channels.ControlChannel().Len())
	}
	if contextID := recvEnvelope(t, channels.ControlChannel()).Pkt.ContextId(); contextID != "second" {
		t.Fatalf("expected newest control packet to be retained, got %s", contextID)
	}
}

func recvEnvelope(t *testing.T, ch *policychannel.Channel[Envelope]) Envelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	envelope, err := ch.Receive(ctx)
	if err != nil {
		t.Fatalf("receive envelope: %v", err)
	}
	return envelope
}

func newTestChannel(t *testing.T, capacity int, policy policychannel.OverflowPolicy) *policychannel.Channel[Envelope] {
	t.Helper()
	ch, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(capacity),
		OverflowPolicy: policy,
	})
	if err != nil {
		t.Fatalf("create test channel: %v", err)
	}
	return ch
}

func newRequestorChannelsWithIngressCapacity(t *testing.T, capacity int) *RequestorChannels {
	t.Helper()
	ingressCh, err := policychannel.New[Envelope](policychannel.Config{
		CapacityPolicy: policychannel.FixedCapacity(capacity),
		OverflowPolicy: policychannel.ReplaceOldestWhenFull,
	}, policychannel.WithReplacementFilter(isRealtimeAudioEnvelope))
	if err != nil {
		t.Fatalf("create ingress test channel: %v", err)
	}
	return &RequestorChannels{
		controlChannel: newTestChannel(t, 1, policychannel.ReplaceOldestWhenFull),
		bootstrapCh:    newTestChannel(t, 1, policychannel.BlockWhenFull),
		ingressCh:      ingressCh,
		ingressPauseCh: make(chan struct{}),
		egressCh:       newTestChannel(t, 1, policychannel.BlockWhenFull),
		dataCh:         newTestChannel(t, 1, policychannel.BlockWhenFull),
		backgroundCh:   newTestChannel(t, 1, policychannel.BlockWhenFull),
	}
}

func TestRequestorChannels_OnRoutesToExpectedChannel(t *testing.T) {
	chs := NewRequestorChannels()
	ctx := context.Background()

	controlEnv := Envelope{Ctx: ctx, Pkt: internal_type.TurnChangePacket{ContextID: "ctrl"}}
	bootstrapEnv := Envelope{Ctx: ctx, Pkt: internal_type.InitializeAssistantPacket{ContextID: "boot"}}
	ingressEnv := Envelope{Ctx: ctx, Pkt: internal_type.UserTextReceivedPacket{ContextID: "in", Text: "hello"}}
	egressEnv := Envelope{Ctx: ctx, Pkt: internal_type.TextToSpeechTextPacket{ContextID: "out", Text: "hi"}}
	dataEnv := Envelope{Ctx: ctx, Pkt: internal_type.RecordUserAudioPacket{ContextID: "data"}}
	backgroundEnv := Envelope{Ctx: ctx, Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "bg"}}

	chs.OnControl(controlEnv)
	chs.OnBootstrap(bootstrapEnv)
	chs.OnIngress(ingressEnv)
	chs.OnEgress(egressEnv)
	chs.OnData(dataEnv)
	chs.OnBackground(backgroundEnv)

	gotControl := recvEnvelope(t, chs.ControlChannel())
	gotBootstrap := recvEnvelope(t, chs.BootstrapChannel())
	gotIngress := recvEnvelope(t, chs.IngressChannel())
	gotEgress := recvEnvelope(t, chs.EgressChannel())
	gotData := recvEnvelope(t, chs.DataChannel())
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
	if gotData.Pkt.ContextId() != "data" {
		t.Fatalf("unexpected data packet context: %s", gotData.Pkt.ContextId())
	}
	if gotBackground.Pkt.ContextId() != "bg" {
		t.Fatalf("unexpected background packet context: %s", gotBackground.Pkt.ContextId())
	}
}

func TestNewRequestorChannels_DefaultCapacities(t *testing.T) {
	chs := NewRequestorChannels()

	if chs.ControlChannel().Capacity() != 256 {
		t.Fatalf("unexpected controlChannel cap: %d", chs.ControlChannel().Capacity())
	}
	if chs.BootstrapChannel().Capacity() != 512 {
		t.Fatalf("unexpected bootstrapCh cap: %d", chs.BootstrapChannel().Capacity())
	}
	if chs.IngressChannel().Capacity() != 4096 {
		t.Fatalf("unexpected ingressCh cap: %d", chs.IngressChannel().Capacity())
	}
	if chs.EgressChannel().Capacity() != 2048 {
		t.Fatalf("unexpected egressCh cap: %d", chs.EgressChannel().Capacity())
	}
	if chs.DataChannel().Capacity() != 2048 {
		t.Fatalf("unexpected dataCh cap: %d", chs.DataChannel().Capacity())
	}
	if chs.BackgroundChannel().Capacity() != 2048 {
		t.Fatalf("unexpected backgroundCh cap: %d", chs.BackgroundChannel().Capacity())
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
	if chs.IngressChannel().Len() != 0 {
		t.Fatalf("expected ingress channel to be empty, got len=%d", chs.IngressChannel().Len())
	}
	if chs.EgressChannel().Len() != 2 {
		t.Fatalf("expected egress channel to be untouched, got len=%d", chs.EgressChannel().Len())
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
	chs.OnData(Envelope{Ctx: context.Background(), Pkt: internal_type.RecordUserAudioPacket{ContextID: "d1"}})

	chs.OnBackground(Envelope{Ctx: context.Background(), Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "g1"}})
	chs.OnBackground(Envelope{Ctx: context.Background(), Pkt: internal_type.ObservabilityEventRecordPacket{ContextID: "g2"}})

	dropped := chs.FlushAll()
	if dropped != 10 {
		t.Fatalf("expected flushAll to drop 10 packets, got %d", dropped)
	}

	if chs.ControlChannel().Len() != 0 || chs.BootstrapChannel().Len() != 0 || chs.IngressChannel().Len() != 0 || chs.EgressChannel().Len() != 0 || chs.DataChannel().Len() != 0 || chs.BackgroundChannel().Len() != 0 {
		t.Fatalf("expected all channels to be empty after flushAll")
	}

	if chs.FlushAll() != 0 {
		t.Fatalf("expected second flushAll to drop 0 packets")
	}
}

func TestRequestorChannels_OnIngressWhenFull_EvictsOnlyOldestAudioPacket(t *testing.T) {
	chs := newRequestorChannelsWithIngressCapacity(t, 3)

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.SpeechToTextPacket{ContextID: "transcript"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "old-audio"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.EndOfSpeechPacket{ContextID: "end-of-speech"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "new-audio"}})

	if got := chs.IngressChannel().Len(); got != 3 {
		t.Fatalf("expected bounded ingress queue, got %d", got)
	}

	for _, expected := range []string{"transcript", "end-of-speech", "new-audio"} {
		if got := recvEnvelope(t, chs.IngressChannel()).Pkt.ContextId(); got != expected {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	}
}

func TestRequestorChannels_OnIngressOverflow_ThenRefillsInOrder(t *testing.T) {
	chs := newRequestorChannelsWithIngressCapacity(t, 2)

	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "old-1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "old-2"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserAudioReceivedPacket{ContextID: "latest"}})
	_ = recvEnvelope(t, chs.IngressChannel())
	_ = recvEnvelope(t, chs.IngressChannel())
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "new-1"}})
	chs.OnIngress(Envelope{Ctx: context.Background(), Pkt: internal_type.UserTextReceivedPacket{ContextID: "new-2"}})

	if got := chs.IngressChannel().Len(); got != 2 {
		t.Fatalf("expected ingress len=2 after refill, got %d", got)
	}

	first := recvEnvelope(t, chs.IngressChannel())
	second := recvEnvelope(t, chs.IngressChannel())
	if first.Pkt.ContextId() != "new-1" || second.Pkt.ContextId() != "new-2" {
		t.Fatalf("expected new-1,new-2 order; got %s,%s", first.Pkt.ContextId(), second.Pkt.ContextId())
	}
}

func TestRequestorChannels_OnEgressWhenFull_BlocksUntilConsumerDrains(t *testing.T) {
	chs := newRequestorChannelsWithIngressCapacity(t, 1)

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
	chs := newRequestorChannelsWithIngressCapacity(t, 4)

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

	if got := chs.IngressChannel().Len(); got != chs.IngressChannel().Capacity() {
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

	if got := chs.IngressChannel().Len(); got != 0 {
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

	go func() {
		chs.PauseIngress(ctx, func() {
			close(paused)
		})
	}()

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
	if got := chs.IngressChannel().Len(); got != 0 {
		t.Fatalf("expected ingress to drop packets after pause, got len=%d", got)
	}
}

func TestRequestorChannels_RunIngressAfterPauseUsesNewPauseChannel(t *testing.T) {
	chs := NewRequestorChannels()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstHandled := make(chan struct{})
	firstPaused := make(chan struct{})
	go chs.RunIngress(ctx, func(Envelope) {
		close(firstHandled)
	})
	chs.OnIngress(Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserTextReceivedPacket{ContextID: "before-pause"},
	})
	select {
	case <-firstHandled:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first ingress packet")
	}
	chs.PauseIngress(ctx, func() {
		close(firstPaused)
	})
	select {
	case <-firstPaused:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first pause")
	}

	chs.ingressWriteMu.Lock()
	pausedIngressCh := chs.ingressPauseCh
	chs.ingressWriteMu.Unlock()
	handled := make(chan struct{})
	go chs.RunIngress(ctx, func(Envelope) {
		close(handled)
	})
	require.Eventually(t, func() bool {
		chs.ingressWriteMu.Lock()
		runningIngressCh := chs.ingressPauseCh
		chs.ingressWriteMu.Unlock()
		return runningIngressCh != pausedIngressCh
	}, time.Second, time.Millisecond)
	chs.OnIngress(Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserTextReceivedPacket{ContextID: "after-pause"},
	})

	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatalf("restarted ingress did not handle packet")
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
			t.Fatalf("unexpected control packet context: %s", got.Pkt.ContextId())
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for control packet to be handled")
	}

	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for control runner to stop")
	}
}
