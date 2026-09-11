package lifecycle

import (
	"context"
	"math"
	"testing"
	"testing/synctest"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestSessionLifecycle_HappyPath(t *testing.T) {
	l := NewSessionLifecycle()

	if err := l.Transition(EventConnectRequested); err != nil {
		t.Fatalf("connect transition failed: %v", err)
	}
	if got := l.Current(); got != StateInitializing {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateInitializing)
	}

	if err := l.Transition(EventInitializationCompleted); err != nil {
		t.Fatalf("init complete transition failed: %v", err)
	}
	if got := l.Current(); got != StateReady {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateReady)
	}

	if err := l.Transition(EventSwitchRequested); err != nil {
		t.Fatalf("switch requested transition failed: %v", err)
	}
	if got := l.Current(); got != StateSwitching {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateSwitching)
	}

	if err := l.Transition(EventSwitchCompleted); err != nil {
		t.Fatalf("switch completed transition failed: %v", err)
	}
	if got := l.Current(); got != StateReady {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateReady)
	}

	if err := l.Transition(EventDisconnectRequested); err != nil {
		t.Fatalf("disconnect requested transition failed: %v", err)
	}
	if got := l.Current(); got != StateDisconnecting {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateDisconnecting)
	}

	if err := l.Transition(EventDisconnectCompleted); err != nil {
		t.Fatalf("disconnect completed transition failed: %v", err)
	}
	if got := l.Current(); got != StateDisconnected {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateDisconnected)
	}
}

func TestSessionLifecycle_TimeoutDurationBounds(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		seconds uint64
	}{
		{name: "disabled", seconds: 0},
		{name: "one second", seconds: 1},
		{name: "maximum seconds", seconds: uint64(math.MaxInt64 / int64(time.Second))},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				lifecycle := NewSessionLifecycle()
				defer lifecycle.CloseTimeouts()
				expired := make(chan internal_type.Packet, 2)
				err := lifecycle.ConfigureTimeouts(context.Background(), "current", &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &testCase.seconds, MaxSessionDuration: &testCase.seconds,
				}, func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						switch packet.(type) {
						case internal_type.IdleTimeoutExpiredPacket, internal_type.MaxSessionExpiredPacket:
							expired <- packet
						}
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				lifecycle.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "current"},
					NewMessageLifecycleWithContext("current", type_enums.TextMode))
				if testCase.seconds == 0 {
					time.Sleep(time.Second)
					synctest.Wait()
					if len(expired) != 0 {
						t.Fatal("disabled timeouts emitted expiry")
					}
					return
				}
				if testCase.seconds == uint64(math.MaxInt64/int64(time.Second)) {
					time.Sleep(time.Second)
					synctest.Wait()
					if len(expired) != 0 {
						t.Fatal("maximum valid timeout expired immediately")
					}
					if lifecycle.MaxSessionExpired(internal_type.MaxSessionExpiredPacket{ContextID: "current"}) == nil {
						t.Fatal("maximum valid session timeout was not armed")
					}
					return
				}
				time.Sleep(time.Duration(testCase.seconds)*time.Second - time.Nanosecond)
				synctest.Wait()
				if len(expired) != 0 {
					t.Fatal("timeout expired before the configured duration")
				}
				time.Sleep(time.Nanosecond)
				synctest.Wait()
				if len(expired) != 2 {
					t.Fatalf("expected both timeouts to expire, got %d", len(expired))
				}
			})
		})
	}
}

func TestSessionLifecycle_InvalidTimeoutPreservesActiveTimers(t *testing.T) {
	for _, field := range []string{"idle timeout", "maximum session duration"} {
		t.Run(field, func(t *testing.T) {
			for _, testCase := range []struct {
				name    string
				seconds uint64
			}{
				{name: "multiplication overflow", seconds: uint64(math.MaxInt64/int64(time.Second)) + 1},
				{name: "signed conversion overflow", seconds: uint64(math.MaxInt64) + 1},
				{name: "maximum uint64", seconds: math.MaxUint64},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						lifecycle := NewSessionLifecycle()
						defer lifecycle.CloseTimeouts()
						seconds := uint64(1)
						expired := make(chan internal_type.Packet, 2)
						err := lifecycle.ConfigureTimeouts(context.Background(), "current", &internal_assistant_entity.AssistantDeploymentBehavior{
							IdleTimeout: &seconds, MaxSessionDuration: &seconds,
						}, func(_ context.Context, packets ...internal_type.Packet) error {
							for _, packet := range packets {
								switch packet.(type) {
								case internal_type.IdleTimeoutExpiredPacket, internal_type.MaxSessionExpiredPacket:
									expired <- packet
								}
							}
							return nil
						})
						if err != nil {
							t.Fatal(err)
						}
						lifecycle.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "current"},
							NewMessageLifecycleWithContext("current", type_enums.TextMode))
						behavior := &internal_assistant_entity.AssistantDeploymentBehavior{}
						if field == "idle timeout" {
							behavior.IdleTimeout = &testCase.seconds
						} else {
							behavior.MaxSessionDuration = &testCase.seconds
						}
						if err := lifecycle.ConfigureTimeouts(context.Background(), "invalid", behavior, nil); err == nil {
							t.Fatal("expected invalid timeout configuration to fail")
						}
						time.Sleep(time.Second)
						synctest.Wait()
						if len(expired) != 2 {
							t.Fatalf("invalid configuration changed active timers: received %d expiries", len(expired))
						}
						for range 2 {
							if packet := <-expired; packet.ContextId() != "current" {
								t.Fatalf("expiry belongs to unexpected context: %s", packet.ContextId())
							}
						}
					})
				})
			}
		})
	}
}

func TestSessionLifecycle_MaxExpiryAdmission(t *testing.T) {
	for _, action := range []protos.ToolCallAction{
		protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED,
		protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
		protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
	} {
		t.Run(action.String(), func(t *testing.T) {
			l := NewSessionLifecycle()
			defer l.CloseTimeouts()
			duration := uint64(60)
			l.ConfigureTimeouts(context.Background(), "session", &internal_assistant_entity.AssistantDeploymentBehavior{MaxSessionDuration: &duration}, nil)
			if got := l.MaxSessionExpired(internal_type.MaxSessionExpiredPacket{ContextID: "stale"}); got != nil {
				t.Fatal("stale maximum-session expiry was accepted")
			}
			packets := l.ToolCall(internal_type.LLMToolCallPacket{ContextID: "message", Action: action})
			got := l.MaxSessionExpired(internal_type.MaxSessionExpiredPacket{ContextID: "session"})
			if action == protos.ToolCallAction_TOOL_CALL_ACTION_UNSPECIFIED {
				if len(packets) != 0 || got == nil || got.Type != protos.ConversationDisconnection_DISCONNECTION_TYPE_MAX_DURATION {
					t.Fatal("ordinary tool call changed the maximum-session deadline")
				}
			} else if len(packets) != 1 || got != nil {
				t.Fatal("terminal tool action did not cancel timeout eligibility")
			}
			if l.MaxSessionExpired(internal_type.MaxSessionExpiredPacket{ContextID: "session"}) != nil {
				t.Fatal("duplicate maximum-session expiry was accepted")
			}
		})
	}
}

func TestSessionLifecycle_StaleStopPreservesCurrentCountdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := NewSessionLifecycle()
		defer l.CloseTimeouts()
		timeout := uint64(1)
		expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
		l.ConfigureTimeouts(context.Background(), "current", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
					expired <- p
				}
			}
			return nil
		})
		message := NewMessageLifecycleWithContext("current", type_enums.AudioMode)
		l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "current"}, message)
		l.StopIdleTimeout(internal_type.StopIdleTimeoutPacket{ContextID: "previous", ResetCount: true})
		time.Sleep(time.Second)
		synctest.Wait()
		prompt, disconnect := l.IdleTimeoutExpired(<-expired)
		if prompt.ContextID != "current" || disconnect != nil {
			t.Fatal("stale stop canceled the active message countdown")
		}
	})
}

func TestSessionLifecycle_FatalSwitchFailure(t *testing.T) {
	l := NewSessionLifecycleWithState(StateReady)
	if err := l.Transition(EventSwitchRequested); err != nil {
		t.Fatalf("switch requested transition failed: %v", err)
	}
	if err := l.Transition(EventSwitchFailedFatal); err != nil {
		t.Fatalf("fatal switch failure transition failed: %v", err)
	}
	if got := l.Current(); got != StateFailed {
		t.Fatalf("state mismatch: got=%v want=%v", got, StateFailed)
	}
}

func TestSessionLifecycle_InvalidTransition(t *testing.T) {
	l := NewSessionLifecycleWithState(StateReady)
	if err := l.Transition(EventInitializationCompleted); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}

func TestSessionLifecycle_IdleTimeoutPrompt(t *testing.T) {
	timeout := uint64(1)
	blank, custom := " \t\n", "Please respond."
	for _, tt := range []struct {
		name   string
		prompt *string
		want   string
	}{
		{name: "default", want: "Are you still there?"},
		{name: "blank", prompt: &blank, want: "Are you still there?"},
		{name: "custom", prompt: &custom, want: "Please respond."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewSessionLifecycle()
				defer l.CloseTimeouts()
				expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
				l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &timeout, IdleTimeoutMessage: tt.prompt,
				}, func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
							expired <- p
						}
					}
					return nil
				})
				l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
				time.Sleep(time.Second)
				synctest.Wait()
				expiry := <-expired
				packet, disconnect := l.IdleTimeoutExpired(expiry)
				if packet.ContextID != "ctx" || packet.Text != tt.want || disconnect != nil || l.IdleTimeoutCount() != 1 {
					t.Fatalf("expiry: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
				}
				packet, disconnect = l.IdleTimeoutExpired(expiry)
				if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 1 {
					t.Fatalf("duplicate expiry accepted: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
				}
			})
		})
	}
}

func TestSessionLifecycle_IdleTimeoutDisabled(t *testing.T) {
	zero := uint64(0)
	for _, tt := range []struct {
		name     string
		behavior *internal_assistant_entity.AssistantDeploymentBehavior
	}{
		{name: "missing behavior"},
		{name: "missing duration", behavior: &internal_assistant_entity.AssistantDeploymentBehavior{}},
		{name: "zero duration", behavior: &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &zero}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := NewSessionLifecycle()
			t.Cleanup(l.CloseTimeouts)
			l.ConfigureTimeouts(context.Background(), "ctx", tt.behavior, nil)
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
			packet, disconnect := l.IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
			if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
				t.Fatalf("disabled expiry accepted: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
			}
		})
	}
}

func TestSessionLifecycle_IdleTimeoutBackoff(t *testing.T) {
	timeout := uint64(1)
	for _, tt := range []struct {
		name    string
		backoff uint64
	}{{name: "unlimited"}, {name: "two prompts", backoff: 2}} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				backoff := tt.backoff
				l := NewSessionLifecycle()
				defer l.CloseTimeouts()
				expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
				l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &timeout, IdleTimeoutBackoff: &backoff,
				}, func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
							expired <- p
						}
					}
					return nil
				})
				message := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
				for count := uint64(0); count < 3; count++ {
					l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
					time.Sleep(time.Second)
					synctest.Wait()
					packet, disconnect := l.IdleTimeoutExpired(<-expired)
					if backoff > 0 && count >= backoff {
						if packet.Text != "" || disconnect == nil || disconnect.Type != protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT || l.IdleTimeoutCount() != backoff {
							t.Fatalf("backoff: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
						}
						continue
					}
					if packet.Text == "" || disconnect != nil || l.IdleTimeoutCount() != count+1 {
						t.Fatalf("prompt %d: packet=%+v disconnect=%v count=%d", count, packet, disconnect, l.IdleTimeoutCount())
					}
				}
			})
		})
	}
}

func TestSessionLifecycle_IdleTimeoutEligibility(t *testing.T) {
	timeout := uint64(1)
	for _, tt := range []struct {
		name string
		act  func(*testing.T, SessionLifecycle, MessageLifecycle)
	}{
		{name: "missing owner", act: func(_ *testing.T, l SessionLifecycle, _ MessageLifecycle) {
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, nil)
		}},
		{name: "stale start", act: func(_ *testing.T, l SessionLifecycle, message MessageLifecycle) {
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "old"}, message)
		}},
		{name: "speaking at start", act: func(t *testing.T, l SessionLifecycle, message MessageLifecycle) {
			if err := message.UserSpeaking("ctx"); err != nil {
				t.Fatal(err)
			}
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
		}},
		{name: "speaking at expiry", act: func(t *testing.T, l SessionLifecycle, message MessageLifecycle) {
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			if err := message.UserSpeaking("ctx"); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "rotated before expiry", act: func(t *testing.T, l SessionLifecycle, message MessageLifecycle) {
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			if _, _, err := message.RotateContext(); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewSessionLifecycle()
				defer l.CloseTimeouts()
				expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
				l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
							expired <- p
						}
					}
					return nil
				})
				message := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
				tt.act(t, l, message)
				time.Sleep(time.Second)
				synctest.Wait()
				expiry := internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"}
				if tt.name == "speaking at expiry" || tt.name == "rotated before expiry" {
					expiry = <-expired
				} else if len(expired) != 0 {
					t.Fatal("ineligible start produced an expiry")
				}
				packet, disconnect := l.IdleTimeoutExpired(expiry)
				if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
					t.Fatalf("ineligible expiry accepted: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
				}
			})
		})
	}
}

func TestSessionLifecycle_StopIdleTimeout(t *testing.T) {
	timeout := uint64(1)
	for _, resetCount := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			l := NewSessionLifecycle()
			defer l.CloseTimeouts()
			expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
			l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
						expired <- p
					}
				}
				return nil
			})
			message := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			time.Sleep(time.Second)
			synctest.Wait()
			l.IdleTimeoutExpired(<-expired)
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			time.Sleep(time.Second)
			synctest.Wait()
			expiry := <-expired
			l.StopIdleTimeout(internal_type.StopIdleTimeoutPacket{ContextID: "ctx", ResetCount: resetCount})
			wantCount := uint64(1)
			if resetCount {
				wantCount = 0
			}
			packet, disconnect := l.IdleTimeoutExpired(expiry)
			if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != wantCount {
				t.Fatalf("stopped expiry accepted: reset=%v packet=%+v disconnect=%v count=%d", resetCount, packet, disconnect, l.IdleTimeoutCount())
			}
		})
	}
}

func TestSessionLifecycle_TimeoutDurations(t *testing.T) {
	t.Parallel()
	timeout := uint64(1)
	l := NewSessionLifecycle()
	t.Cleanup(l.CloseTimeouts)
	packets := make(chan internal_type.Packet, 4)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "session")
	l.ConfigureTimeouts(ctx, "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{
		IdleTimeout: &timeout, MaxSessionDuration: &timeout,
	}, func(packetCtx context.Context, batch ...internal_type.Packet) error {
		if packetCtx.Value(contextKey{}) != "session" {
			t.Error("callback lost session context")
		}
		for _, packet := range batch {
			switch packet.(type) {
			case internal_type.IdleTimeoutExpiredPacket, internal_type.MaxSessionExpiredPacket:
				packets <- packet
			}
		}
		return nil
	})
	l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
	select {
	case packet := <-packets:
		t.Fatalf("timeout fired before configured seconds: %+v", packet)
	case <-time.After(200 * time.Millisecond):
	}
	idleExpired, maxExpired := false, false
	deadline := time.After(3 * time.Second)
	for !idleExpired || !maxExpired {
		select {
		case packet := <-packets:
			if packet.ContextId() != "ctx" {
				t.Fatalf("unexpected timeout context: %+v", packet)
			}
			switch packet := packet.(type) {
			case internal_type.IdleTimeoutExpiredPacket:
				idleExpired = true
				prompt, disconnect := l.IdleTimeoutExpired(packet)
				if prompt.Text == "" || disconnect != nil || l.IdleTimeoutCount() != 1 {
					t.Fatalf("timer expiry rejected: %+v %v", prompt, disconnect)
				}
			case internal_type.MaxSessionExpiredPacket:
				maxExpired = true
			}
		case <-deadline:
			t.Fatalf("missing timeout: idle=%v max=%v", idleExpired, maxExpired)
		}
	}
}

func TestSessionLifecycle_ExtendIdleTimeout(t *testing.T) {
	t.Parallel()
	timeout := uint64(1)
	l := NewSessionLifecycle()
	t.Cleanup(l.CloseTimeouts)
	expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
	l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if packet, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
				expired <- packet
			}
		}
		return nil
	})
	l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
	l.ExtendIdleTimeout("old", time.Hour)
	l.ExtendIdleTimeout("ctx", -time.Hour)
	l.ExtendIdleTimeout("ctx", 500*time.Millisecond)
	select {
	case packet := <-expired:
		t.Fatalf("expiry before extended duration: %+v", packet)
	case <-time.After(1200 * time.Millisecond):
	}
	select {
	case packet := <-expired:
		if packet.ContextID != "ctx" {
			t.Fatalf("unexpected expiry: %+v", packet)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("extended timeout did not expire")
	}
}

func TestSessionLifecycle_CloseTimeouts(t *testing.T) {
	for _, action := range []string{"close", "cancel", "reconfigure"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			timeout := uint64(1)
			l := NewSessionLifecycle()
			t.Cleanup(l.CloseTimeouts)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			expired := make(chan internal_type.Packet, 4)
			l.ConfigureTimeouts(ctx, "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{
				IdleTimeout: &timeout, MaxSessionDuration: &timeout,
			}, func(_ context.Context, packets ...internal_type.Packet) error {
				for _, packet := range packets {
					switch packet.(type) {
					case internal_type.IdleTimeoutExpiredPacket, internal_type.MaxSessionExpiredPacket:
						expired <- packet
					}
				}
				return nil
			})
			message := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			switch action {
			case "close":
				l.CloseTimeouts()
				l.CloseTimeouts()
			case "cancel":
				cancel()
			case "reconfigure":
				l.ConfigureTimeouts(context.Background(), "new", nil, nil)
			}
			select {
			case packet := <-expired:
				t.Fatalf("timeout after %s: %+v", action, packet)
			case <-time.After(1200 * time.Millisecond):
			}
			l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
			packet, disconnect := l.IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
			if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
				t.Fatalf("expiry accepted after %s: %+v %v", action, packet, disconnect)
			}
		})
	}
}

func TestSessionLifecycle_ConfigureTimeoutsCallbackCanClose(t *testing.T) {
	timeout := uint64(1)
	l := NewSessionLifecycle()
	t.Cleanup(l.CloseTimeouts)
	l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(context.Context, ...internal_type.Packet) error {
		l.CloseTimeouts()
		return nil
	})
	packet, disconnect := l.IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
	if packet.Text != "" || disconnect != nil {
		t.Fatalf("configuration survived callback close: %+v %v", packet, disconnect)
	}
}

func TestSessionLifecycle_IdleTimeoutRejectsStalePackets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timeout := uint64(1)
		l := NewSessionLifecycle()
		defer l.CloseTimeouts()
		expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
		l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
					expired <- p
				}
			}
			return nil
		})
		packet, disconnect := l.IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
		if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
			t.Fatal("pre-start expiry accepted")
		}
		l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
		time.Sleep(time.Second)
		synctest.Wait()
		expiry := <-expired
		for _, expired := range []internal_type.IdleTimeoutExpiredPacket{
			{ContextID: "old"},
			{ContextID: "ctx", Count: 1},
			{ContextID: "ctx"},
			{ContextID: "old", Generation: expiry.Generation, Deadline: expiry.Deadline},
			{ContextID: "ctx", Count: 1, Generation: expiry.Generation, Deadline: expiry.Deadline},
			{ContextID: "ctx", Generation: expiry.Generation, Deadline: expiry.Deadline.Add(time.Second)},
			{},
		} {
			packet, disconnect := l.IdleTimeoutExpired(expired)
			if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
				t.Fatalf("stale expiry accepted: %+v %v", packet, disconnect)
			}
		}
		packet, disconnect = l.IdleTimeoutExpired(expiry)
		if packet.Text == "" || disconnect != nil || l.IdleTimeoutCount() != 1 {
			t.Fatalf("valid expiry rejected after stale packets: %+v %v", packet, disconnect)
		}
	})
}

func TestSessionLifecycle_CanceledContextRejectsExpiryImmediately(t *testing.T) {
	for _, cancelBeforeConfigure := range []bool{false, true} {
		timeout := uint64(1)
		l := NewSessionLifecycle()
		t.Cleanup(l.CloseTimeouts)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if cancelBeforeConfigure {
			cancel()
		}
		l.ConfigureTimeouts(ctx, "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, nil)
		cancel()
		l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
		packet, disconnect := l.IdleTimeoutExpired(internal_type.IdleTimeoutExpiredPacket{ContextID: "ctx"})
		if packet.Text != "" || disconnect != nil {
			t.Fatalf("canceled expiry accepted: beforeConfigure=%v packet=%+v disconnect=%v", cancelBeforeConfigure, packet, disconnect)
		}
	}
}

func TestSessionLifecycle_ReconfigureTimeoutsDetachesOldContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timeout, backoff := uint64(1), uint64(2)
		prompt := "Please respond."
		l := NewSessionLifecycle()
		defer l.CloseTimeouts()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		l.ConfigureTimeouts(ctx, "old", &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}, nil)
		expired := make(chan internal_type.IdleTimeoutExpiredPacket, 1)
		l.ConfigureTimeouts(context.Background(), "ctx", &internal_assistant_entity.AssistantDeploymentBehavior{
			IdleTimeout: &timeout, IdleTimeoutBackoff: &backoff, IdleTimeoutMessage: &prompt,
		}, func(_ context.Context, packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
					expired <- p
				}
			}
			return nil
		})
		cancel()
		timeout, backoff, prompt = 0, 0, "Changed."
		l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, NewMessageLifecycleWithContext("ctx", type_enums.TextMode))
		time.Sleep(time.Second)
		synctest.Wait()
		packet, disconnect := l.IdleTimeoutExpired(<-expired)
		if packet.Text != "Please respond." || disconnect != nil || l.IdleTimeoutCount() != 1 {
			t.Fatalf("new configuration changed: packet=%+v disconnect=%v count=%d", packet, disconnect, l.IdleTimeoutCount())
		}
	})
}

func TestSessionLifecycle_IdleTimeoutQueuedExpiryAdmission(t *testing.T) {
	for _, action := range []string{"restart", "replace", "extend", "reconfigure", "cancel", "close"} {
		t.Run(action, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewSessionLifecycle()
				defer l.CloseTimeouts()
				timeout := uint64(1)
				behavior := &internal_assistant_entity.AssistantDeploymentBehavior{IdleTimeout: &timeout}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				expired := make(chan internal_type.IdleTimeoutExpiredPacket, 2)
				onPacket := func(_ context.Context, packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if p, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
							expired <- p
						}
					}
					return nil
				}
				l.ConfigureTimeouts(ctx, "ctx", behavior, onPacket)
				message := NewMessageLifecycleWithContext("ctx", type_enums.TextMode)
				l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
				time.Sleep(time.Second)
				synctest.Wait()
				stale := <-expired
				switch action {
				case "restart":
					l.StopIdleTimeout(internal_type.StopIdleTimeoutPacket{ContextID: "ctx"})
					l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
				case "replace":
					l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
				case "extend":
					l.ExtendIdleTimeout("ctx", time.Second)
				case "reconfigure":
					l.ConfigureTimeouts(ctx, "ctx", behavior, onPacket)
					l.StartIdleTimeout(internal_type.StartIdleTimeoutPacket{ContextID: "ctx"}, message)
				case "cancel":
					cancel()
				case "close":
					l.CloseTimeouts()
				}
				packet, disconnect := l.IdleTimeoutExpired(stale)
				if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
					t.Fatalf("queued expiry accepted after %s: packet=%+v disconnect=%v", action, packet, disconnect)
				}
				time.Sleep(time.Second)
				synctest.Wait()
				if action == "cancel" || action == "close" {
					if len(expired) != 0 {
						t.Fatal("canceled countdown emitted another expiry")
					}
					return
				}
				current := <-expired
				if current.ContextID != stale.ContextID || current.Count != stale.Count {
					t.Fatalf("test did not preserve context and count: stale=%+v current=%+v", stale, current)
				}
				packet, disconnect = l.IdleTimeoutExpired(stale)
				if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 0 {
					t.Fatal("old expiry matched the replacement countdown's expiry")
				}
				packet, disconnect = l.IdleTimeoutExpired(current)
				if packet.Text == "" || disconnect != nil || l.IdleTimeoutCount() != 1 {
					t.Fatalf("current expiry rejected: packet=%+v disconnect=%v", packet, disconnect)
				}
				packet, disconnect = l.IdleTimeoutExpired(current)
				if packet.Text != "" || disconnect != nil || l.IdleTimeoutCount() != 1 {
					t.Fatal("duplicate expiry accepted")
				}
			})
		})
	}
}
