package lifecycle

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestMessageLifecycle_ObserveInterruptionRotatesLegacyTurn(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mode    type_enums.MessageMode
		source  internal_type.InterruptionSource
		trigger string
		want    MessageState
	}{
		{name: "vad", mode: type_enums.AudioMode, source: internal_type.InterruptionSourceVad, trigger: internal_options.BargeInTriggerVAD, want: MessageStateUserSpeaking},
		{name: "word", mode: type_enums.AudioMode, source: internal_type.InterruptionSourceWord, trigger: internal_options.BargeInTriggerWord, want: MessageStateUserListening},
		{name: "text", mode: type_enums.TextMode, source: internal_type.InterruptionSourceWord, trigger: internal_options.BargeInTriggerVAD, want: MessageStateUserListening},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := &messageLifecycle{contextID: "old", mode: tt.mode, state: MessageStateAssistantSpeaking, output: assistantOutputState{terminalIssued: true, receiptReceived: true}}
			decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
				ContextID: "old", Source: tt.source, Event: internal_type.InterruptionEventStart,
			}, tt.trigger)
			current := l.ContextID()
			if current == "" || current == "old" || l.State() != tt.want {
				t.Fatalf("turn was not rotated: context=%q state=%s", current, l.State())
			}
			wantPackets := []internal_type.Packet{
				internal_type.StopIdleTimeoutPacket{}, internal_type.EndOfSpeechInterruptionPacket{},
				internal_type.ObservabilityEventRecordPacket{}, internal_type.TurnChangePacket{},
				internal_type.ObservabilityEventRecordPacket{}, internal_type.TextToSpeechInterruptPacket{}, internal_type.LLMInterruptPacket{},
			}
			if len(decision.Packets) != len(wantPackets) {
				t.Fatalf("unexpected packets: %+v", decision.Packets)
			}
			for index, want := range wantPackets {
				if got := decision.Packets[index].PacketName(); got != want.PacketName() {
					t.Fatalf("packet %d: got=%s want=%s", index, got, want.PacketName())
				}
				wantContext := "old"
				if index == 3 || index == 4 {
					wantContext = current
				}
				if got := decision.Packets[index].ContextId(); got != wantContext {
					t.Fatalf("packet %d: context=%q want=%q", index, got, wantContext)
				}
			}
			turn := decision.Packets[3].(internal_type.TurnChangePacket)
			if turn.PreviousContextID != "old" || turn.PreviousState != string(MessageStateAssistantSpeaking) || turn.Source != string(tt.source) || turn.Time.IsZero() {
				t.Fatalf("invalid turn command: %+v", turn)
			}
			if decision.Flush == nil || decision.Flush.Id != "old" || decision.Notification == nil {
				t.Fatalf("missing playback commands: %+v", decision)
			}
			if tt.source == internal_type.InterruptionSourceVad {
				if decision.EndOfSpeech == nil || decision.EndOfSpeech.ContextID != current || decision.SpeechToTextStart == nil || decision.SpeechToTextStart.ContextID != current {
					t.Fatalf("missing VAD boundary commands: %+v", decision)
				}
				if decision.Notification.Type != protos.ConversationInterruption_INTERRUPTION_TYPE_VAD {
					t.Fatalf("wrong notification: %+v", decision.Notification)
				}
			} else if decision.EndOfSpeech != nil || decision.SpeechToTextStart != nil || decision.Notification.Type != protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
				t.Fatalf("unexpected word commands: %+v", decision)
			}
			if err := l.OnPlaybackCompleted(current); err != ErrPlaybackTerminalNotIssued {
				t.Fatalf("rotated turn retained playback eligibility: %v", err)
			}
		})
	}
}

func TestMessageLifecycle_ObserveInterruptionPreservesLegacyStates(t *testing.T) {
	for _, tt := range []struct {
		name      string
		mode      type_enums.MessageMode
		state     MessageState
		source    internal_type.InterruptionSource
		event     internal_type.InterruptionEvent
		trigger   string
		wantState MessageState
		wantEOS   bool
		wantStart bool
		wantEnd   bool
	}{
		{name: "word trigger vad start", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventStart, trigger: internal_options.BargeInTriggerWord, wantState: MessageStateAssistantSpeaking, wantEOS: true, wantStart: true},
		{name: "word trigger vad end", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, trigger: internal_options.BargeInTriggerWord, wantState: MessageStateAssistantSpeaking, wantEOS: true, wantEnd: true},
		{name: "vad ignores word", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceWord, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateAssistantSpeaking},
		{name: "idle vad start", mode: type_enums.AudioMode, state: MessageStateUserIdle, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventStart, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserSpeaking, wantEOS: true, wantStart: true},
		{name: "listening vad start", mode: type_enums.AudioMode, state: MessageStateUserListening, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventStart, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserListening, wantEOS: true, wantStart: true},
		{name: "thinking vad start", mode: type_enums.AudioMode, state: MessageStateUserThinking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventStart, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserThinking, wantEOS: true, wantStart: true},
		{name: "speaking vad end", mode: type_enums.AudioMode, state: MessageStateUserSpeaking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserListening, wantEOS: true, wantEnd: true},
		{name: "listening vad end", mode: type_enums.AudioMode, state: MessageStateUserListening, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserListening, wantEOS: true, wantEnd: true},
		{name: "thinking vad end", mode: type_enums.AudioMode, state: MessageStateUserThinking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, trigger: internal_options.BargeInTriggerVAD, wantState: MessageStateUserThinking, wantEnd: true},
		{name: "speaking text word", mode: type_enums.TextMode, state: MessageStateUserSpeaking, source: internal_type.InterruptionSourceWord, wantState: MessageStateUserListening},
		{name: "thinking text word", mode: type_enums.TextMode, state: MessageStateUserThinking, source: internal_type.InterruptionSourceWord, wantState: MessageStateUserThinking},
		{name: "idle audio word", mode: type_enums.AudioMode, state: MessageStateUserIdle, source: internal_type.InterruptionSourceWord, trigger: internal_options.BargeInTriggerWord, wantState: MessageStateUserIdle},
		{name: "unknown source", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: "unknown", wantState: MessageStateAssistantSpeaking},
		{name: "unknown vad event", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceVad, event: "unknown", wantState: MessageStateAssistantSpeaking},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := &messageLifecycle{contextID: "ctx", mode: tt.mode, state: tt.state}
			decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{Source: tt.source, Event: tt.event}, tt.trigger)
			if l.ContextID() != "ctx" || l.State() != tt.wantState || decision.Flush != nil || decision.Notification != nil || decision.Pause != nil {
				t.Fatalf("unexpected transition: context=%q state=%s decision=%+v", l.ContextID(), l.State(), decision)
			}
			if (decision.EndOfSpeech != nil) != tt.wantEOS || (decision.SpeechToTextStart != nil) != tt.wantStart {
				t.Fatalf("unexpected boundaries: %+v", decision)
			}
			if tt.wantEnd {
				if len(decision.Packets) != 1 || decision.Packets[0] != (internal_type.SpeechToTextEndPacket{ContextID: "ctx"}) {
					t.Fatalf("unexpected end boundary: %+v", decision.Packets)
				}
			} else if len(decision.Packets) != 0 {
				t.Fatalf("unexpected packets: %+v", decision.Packets)
			}
		})
	}
}

func TestMessageLifecycle_ObserveInterruptionStaleAdmission(t *testing.T) {
	for _, tt := range []struct {
		name   string
		state  MessageState
		source internal_type.InterruptionSource
		event  internal_type.InterruptionEvent
		accept bool
	}{
		{name: "stale word while speaking", state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceWord},
		{name: "stale word while listening", state: MessageStateUserListening, source: internal_type.InterruptionSourceWord, accept: true},
		{name: "stale word while thinking", state: MessageStateUserThinking, source: internal_type.InterruptionSourceWord, accept: true},
		{name: "stale vad start", state: MessageStateUserListening, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventStart},
		{name: "stale vad end while thinking", state: MessageStateUserThinking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd},
		{name: "stale vad end while listening", state: MessageStateUserListening, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, accept: true},
		{name: "stale vad end while user speaking", state: MessageStateUserSpeaking, source: internal_type.InterruptionSourceVad, event: internal_type.InterruptionEventEnd, accept: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := &messageLifecycle{contextID: "ctx", mode: type_enums.TextMode, state: tt.state}
			decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "old", Source: tt.source, Event: tt.event}, internal_options.BargeInTriggerVAD)
			if l.ContextID() != "ctx" || decision.Flush != nil {
				t.Fatalf("stale event rotated turn: %+v", decision)
			}
			if !tt.accept && (!reflect.DeepEqual(decision, InterruptionDecision{}) || l.State() != tt.state) {
				t.Fatalf("stale event accepted: state=%s decision=%+v", l.State(), decision)
			}
			if tt.accept && tt.source == internal_type.InterruptionSourceVad && (decision.EndOfSpeech == nil || decision.EndOfSpeech.ContextID != "ctx") {
				t.Fatalf("admitted VAD end lost active context: %+v", decision)
			}
		})
	}
}

func TestMessageLifecycle_ObserveInterruptionAdaptiveVAD(t *testing.T) {
	l := &messageLifecycle{contextID: "ctx", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, interruptionEnabled: true}
	t.Cleanup(func() { l.CancelInterruption() })
	decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, internal_options.BargeInTriggerVAD)
	if decision.Pause == nil || decision.Pause.ContextID != "ctx" || decision.SpeechToTextStart == nil || decision.SpeechToTextStart.ContextID != "ctx" {
		t.Fatalf("adaptive pause missing: %+v", decision)
	}
	if l.ContextID() != "ctx" || l.State() != MessageStateAssistantSpeaking || decision.Flush != nil || decision.EndOfSpeech != nil || len(decision.Packets) != 0 {
		t.Fatalf("adaptive start used legacy rotation: %+v", decision)
	}
	decision = l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
	}, internal_options.BargeInTriggerVAD)
	if decision.Pause != nil || decision.EndOfSpeech != nil || len(decision.Packets) != 1 || decision.Packets[0] != (internal_type.SpeechToTextEndPacket{ContextID: "ctx"}) {
		t.Fatalf("adaptive end was not held: %+v", decision)
	}
}

func TestMessageLifecycle_ObserveInterruptionConcurrentVADStarts(t *testing.T) {
	l := &messageLifecycle{contextID: "ctx", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking}
	decisions := make(chan InterruptionDecision, 32)
	var calls sync.WaitGroup
	for range 32 {
		calls.Add(1)
		go func() {
			defer calls.Done()
			decisions <- l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
				ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}, internal_options.BargeInTriggerVAD)
		}()
	}
	calls.Wait()
	close(decisions)
	rotations := 0
	for decision := range decisions {
		if decision.Flush != nil {
			rotations++
		}
	}
	if rotations != 1 || l.ContextID() == "ctx" || l.State() != MessageStateUserSpeaking {
		t.Fatalf("concurrent starts produced %d rotations, state=%s", rotations, l.State())
	}
}

func TestMessageLifecycle_ObserveInterruptionLegacyTurnInvalidatesAdaptivePause(t *testing.T) {
	l := &messageLifecycle{contextID: "ctx", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, interruptionEnabled: true}
	t.Cleanup(func() { l.CancelInterruption() })
	pause := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}, internal_options.BargeInTriggerVAD).Pause
	if pause == nil {
		t.Fatal("adaptive interruption did not pause")
	}
	turn, _, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "ctx", Script: "stop", Interim: true}, true)
	if turn == nil || !l.beginInterruptedTurn(*turn) {
		t.Fatal("adaptive interruption did not reserve flush")
	}
	decision := l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
		ContextID: "ctx", Source: internal_type.InterruptionSourceWord,
	}, internal_options.BargeInTriggerWord)
	current := l.ContextID()
	if current == "ctx" || decision.Flush == nil || l.State() != MessageStateUserListening {
		t.Fatalf("word trigger did not replace paused turn: %+v", decision)
	}
	if _, accepted := l.commitInterruptedTurn(*turn); accepted {
		t.Fatal("stale adaptive flush replaced legacy turn")
	}
	if l.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "new request"}) || l.CanStartIdleTimeout(current) {
		t.Fatal("legacy turn retained adaptive held input or lost listening state")
	}
	if resumeContextID := l.OnInterruptionExpired(*pause); resumeContextID != "" || l.ContextID() != current {
		t.Fatal("stale adaptive pause changed legacy turn")
	}
}

func TestMessageLifecycle_ObserveInterruptionVADStartsUnclearAfterEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timeout := 0.02
		expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
		l := &messageLifecycle{
			contextID: "ctx", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking,
			loadBehavior: func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
				return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout}, nil
			},
			onPacket: func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expired <- packet
					}
				}
				return nil
			},
		}
		t.Cleanup(l.StopUnclearInput)
		if err := l.Initialize(context.Background()); err != nil {
			t.Fatal(err)
		}
		l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}, internal_options.BargeInTriggerVAD)
		select {
		case packet := <-expired:
			t.Fatalf("VAD start began unclear countdown: %+v", packet)
		case <-time.After(50 * time.Millisecond):
		}
		l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd}, internal_options.BargeInTriggerVAD)
		select {
		case packet := <-expired:
			if packet.ContextID != l.ContextID() {
				t.Fatalf("unclear countdown retained previous turn: %+v", packet)
			}
		case <-time.After(time.Second):
			t.Fatal("VAD end did not start unclear countdown")
		}
	})
}

func TestMessageLifecycle_ObserveInterruptionWordExtendsUnclear(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timeout := 0.3
		expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
		l := &messageLifecycle{
			contextID: "ctx", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking,
			loadBehavior: func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
				return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout}, nil
			},
			onPacket: func(packets ...internal_type.Packet) error {
				for _, packet := range packets {
					if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expired <- packet
					}
				}
				return nil
			},
		}
		t.Cleanup(l.StopUnclearInput)
		if err := l.Initialize(context.Background()); err != nil {
			t.Fatal(err)
		}
		l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: internal_type.InterruptionSourceWord}, internal_options.BargeInTriggerWord)
		time.Sleep(150 * time.Millisecond)
		l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: internal_type.InterruptionSourceWord}, internal_options.BargeInTriggerWord)
		select {
		case packet := <-expired:
			t.Fatalf("duplicate word did not extend countdown: %+v", packet)
		case <-time.After(200 * time.Millisecond):
		}
		select {
		case packet := <-expired:
			if packet.ContextID != l.ContextID() {
				t.Fatalf("unclear countdown retained previous turn: %+v", packet)
			}
		case <-time.After(time.Second):
			t.Fatal("extended unclear countdown did not expire")
		}
		l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: internal_type.InterruptionSourceWord}, internal_options.BargeInTriggerWord)
		select {
		case packet := <-expired:
			t.Fatalf("duplicate word restarted expired countdown: %+v", packet)
		case <-time.After(350 * time.Millisecond):
		}
	})
}

func TestMessageLifecycle_ObserveInterruptionDisabledUnclear(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mode    type_enums.MessageMode
		state   MessageState
		timeout float64
		source  internal_type.InterruptionSource
	}{
		{name: "zero timeout", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, source: internal_type.InterruptionSourceWord},
		{name: "negative timeout", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, timeout: -1, source: internal_type.InterruptionSourceWord},
		{name: "text word", mode: type_enums.TextMode, state: MessageStateAssistantSpeaking, timeout: 0.01, source: internal_type.InterruptionSourceWord},
		{name: "word trigger vad", mode: type_enums.AudioMode, state: MessageStateAssistantSpeaking, timeout: 0.01, source: internal_type.InterruptionSourceVad},
		{name: "idle word", mode: type_enums.AudioMode, state: MessageStateUserIdle, timeout: 0.01, source: internal_type.InterruptionSourceWord},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				expired := make(chan internal_type.UnclearInputExpiredPacket, 2)
				l := &messageLifecycle{
					contextID: "ctx", mode: tt.mode, state: tt.state,
					loadBehavior: func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
						return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &tt.timeout}, nil
					},
					onPacket: func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
								expired <- packet
							}
						}
						return nil
					},
				}
				t.Cleanup(l.StopUnclearInput)
				if err := l.Initialize(context.Background()); err != nil {
					t.Fatal(err)
				}
				l.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{ContextID: "ctx", Source: tt.source, Event: internal_type.InterruptionEventStart}, internal_options.BargeInTriggerWord)
				select {
				case packet := <-expired:
					t.Fatalf("ineligible unclear countdown started: %+v", packet)
				case <-time.After(50 * time.Millisecond):
				}
			})
		})
	}
}
