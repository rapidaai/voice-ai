package adapter_internal

import (
	"context"
	"testing"

	adapter_channel "github.com/rapidaai/api/assistant-api/internal/adapters/channel"
	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	adapter_router "github.com/rapidaai/api/assistant-api/internal/adapters/router"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func newDispatchPolicyTestRequestor() *genericRequestor {
	channels := adapter_channel.NewRequestorChannels()
	return &genericRequestor{
		channels:                channels,
		dispatchRoute:           adapter_router.NewDispatchRoute(adapter_router.NewRoutePolicy(), channels),
		sessionLifecycle:        adapter_lifecycle.NewSessionLifecycleWithState(adapter_lifecycle.StateReady),
		speechToTextTransformer: noopSpeechToTextTransformer{},
		endOfSpeechExecutor:     &recordingEOSExecutor{},
	}
}

func applyDispatchPolicy(r *genericRequestor, policy internal_type.DispatchPolicy) {
	requestorDispatchHandler{r: r}.HandleDispatchPolicy(context.Background(), internal_type.DispatchPolicyPacket{
		Policy: policy,
	})
}

type noopSpeechToTextTransformer struct{}

func (noopSpeechToTextTransformer) Name() string { return "noop" }
func (noopSpeechToTextTransformer) Initialize() error {
	return nil
}
func (noopSpeechToTextTransformer) Transform(context.Context, internal_type.Packet) error {
	return nil
}
func (noopSpeechToTextTransformer) Close(context.Context) error {
	return nil
}

func TestDispatchPolicy_IgnoreDropsMatchingPacketAndPassthroughRestoresDispatch(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})
	r.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "audio-dropped",
		Audio:     []byte("audio"),
	})
	if got := len(r.channels.IngressChannel()); got != 0 {
		t.Fatalf("expected ignored user audio to skip handler execution, ingress len=%d", got)
	}

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionPassthrough,
	})
	r.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "audio-allowed",
		Audio:     []byte("audio"),
	})
	if got := len(r.channels.IngressChannel()); got != 2 {
		t.Fatalf("expected passthrough user audio handler to enqueue downstream packets, ingress len=%d", got)
	}
}

func TestDispatchPolicy_IgnoreDoesNotDropUnrelatedPacket(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})
	r.dispatch(context.Background(), internal_type.LLMToolResultPacket{
		ContextID: "tool-result",
		ToolID:    "tool-1",
		Name:      "transfer",
		Result:    map[string]string{"ok": "true"},
	})

	if got := len(r.channels.DataChannel()); got != 1 {
		t.Fatalf("expected unrelated tool result to dispatch normally, data len=%d", got)
	}
}

func TestDispatchPolicy_MultiplePoliciesCanBeApplied(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})
	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameInterruptionDetected,
		Action: internal_type.DispatchActionIgnore,
	})

	r.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{ContextID: "audio-dropped"})
	r.dispatch(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: "interrupt-dropped"})

	if got := len(r.channels.IngressChannel()); got != 0 {
		t.Fatalf("expected ignored user audio to skip downstream ingress packets, got %d", got)
	}
	if got := len(r.channels.ControlChannel()); got != 0 {
		t.Fatalf("expected ignored interruption to skip control packets, got %d", got)
	}
	if got := len(r.channels.EgressChannel()); got != 0 {
		t.Fatalf("expected ignored interruption to skip egress packets, got %d", got)
	}
}

func TestDispatchPolicy_IgnoreDrainsQueuedMatchingPackets(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	if err := r.OnPacket(context.Background(),
		internal_type.UserAudioReceivedPacket{ContextID: "audio-queued"},
		internal_type.LLMToolResultPacket{ContextID: "tool-result"},
	); err != nil {
		t.Fatalf("initial packet batch returned error: %v", err)
	}
	if got := len(r.channels.IngressChannel()); got != 2 {
		t.Fatalf("expected two queued ingress packets before policy, got %d", got)
	}

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})
	if got := len(r.channels.IngressChannel()); got != 1 {
		t.Fatalf("expected matching queued audio to be drained, ingress len=%d", got)
	}

	env := <-r.channels.IngressChannel()
	if got := env.Pkt.ContextId(); got != "tool-result" {
		t.Fatalf("expected unrelated queued tool result to remain, got %s", got)
	}
}

func TestDispatchPolicy_IgnoreDrainsOnlyTargetRoute(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	r.channels.OnIngress(adapter_channel.Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserAudioReceivedPacket{ContextID: "audio-ingress"},
	})
	r.channels.OnEgress(adapter_channel.Envelope{
		Ctx: context.Background(),
		Pkt: internal_type.UserAudioReceivedPacket{ContextID: "audio-egress"},
	})

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})

	if got := len(r.channels.IngressChannel()); got != 0 {
		t.Fatalf("expected matching ingress packet to be drained, ingress len=%d", got)
	}
	if got := len(r.channels.EgressChannel()); got != 1 {
		t.Fatalf("expected non-target-route packet to remain, egress len=%d", got)
	}
}

func TestDispatchPolicy_PointerTargetMatchesConcretePacket(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchActionIgnore,
	})
	r.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "audio-dropped",
	})
	if got := len(r.channels.IngressChannel()); got != 0 {
		t.Fatalf("expected pointer target to ignore concrete user audio, ingress len=%d", got)
	}
}

func TestDispatchPolicy_UnsupportedActionDoesNotInstallPolicy(t *testing.T) {
	r := newDispatchPolicyTestRequestor()

	applyDispatchPolicy(r, internal_type.DispatchPolicy{
		Target: internal_type.PacketNameUserAudioReceived,
		Action: internal_type.DispatchAction("unsupported"),
	})
	r.dispatch(context.Background(), internal_type.UserAudioReceivedPacket{
		ContextID: "audio-allowed",
	})
	if got := len(r.channels.IngressChannel()); got != 2 {
		t.Fatalf("expected unsupported action not to install policy, ingress len=%d", got)
	}
}
