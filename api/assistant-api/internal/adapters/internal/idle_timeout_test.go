package adapter_internal

import (
	"testing"
	"testing/synctest"
	"time"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func TestIdleTimeout_PlaybackReceiptsRearmPromptsThroughDisconnect(t *testing.T) {
	for _, response := range []string{"greeting", "normal"} {
		t.Run(response, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
				r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
				defer r.sessionLifecycle.CloseTimeouts()
				defer r.messageLifecycle.CancelInterruption()
				h := requestorDispatchHandler{r: r}
				timeout, limit := uint64(1), uint64(2)
				require.NoError(t, r.sessionLifecycle.ConfigureTimeouts(t.Context(), r.GetID(), &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &timeout, IdleTimeoutBackoff: &limit,
				}, r.OnPacket))
				if response == "greeting" {
					_, err := r.messageLifecycle.OnUserTurnStarted("", "test", "test", "")
					require.NoError(t, err)
					h.HandleInjectMessage(t.Context(), internal_type.InjectMessagePacket{ContextID: r.GetID(), Text: "Hello"})
				}
				for count := uint64(0); count <= limit; count++ {
					contextID := r.GetID()
					r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: contextID, Text: "Spoken response"})
					require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "Spoken response"},
					}))
					require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
						Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
					}))
					time.Sleep(2 * time.Second)
					synctest.Wait()
					for _, packet := range drainEgressPackets(r) {
						switch packet.(type) {
						case internal_type.StartIdleTimeoutPacket, internal_type.IdleTimeoutExpiredPacket:
							t.Fatalf("idle timer advanced before playback receipt: %#v", packet)
						}
					}
					require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(contextID))
					starts := 0
					for _, packet := range drainEgressPackets(r) {
						if start, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
							starts++
							require.Equal(t, contextID, start.ContextID)
							h.HandleStartIdleTimeout(t.Context(), start)
						}
					}
					require.Equal(t, 1, starts)
					time.Sleep(time.Second)
					synctest.Wait()
					var expiries []internal_type.IdleTimeoutExpiredPacket
					for _, packet := range drainEgressPackets(r) {
						if expiry, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
							expiries = append(expiries, expiry)
						}
					}
					require.Len(t, expiries, 1)
					h.HandleIdleTimeoutExpired(t.Context(), expiries[0])
					var prompts []internal_type.InjectMessagePacket
					for _, packet := range drainEgressPackets(r) {
						if prompt, ok := packet.(internal_type.InjectMessagePacket); ok {
							prompts = append(prompts, prompt)
						}
					}
					streamer := r.streamer.(*streamTestStreamer)
					streamer.mu.Lock()
					var disconnects []*protos.ConversationDisconnection
					for _, message := range streamer.sent {
						if disconnect, ok := message.(*protos.ConversationDisconnection); ok {
							disconnects = append(disconnects, disconnect)
						}
					}
					streamer.mu.Unlock()
					if count < limit {
						require.Empty(t, disconnects)
						require.Len(t, prompts, 1)
						require.Equal(t, count+1, r.sessionLifecycle.IdleTimeoutCount())
						require.Equal(t, "Are you still there?", prompts[0].Text)
						require.NotEqual(t, contextID, prompts[0].ContextID)
						require.Equal(t, adapter_lifecycle.MessageStateAssistantPrompted, r.messageLifecycle.State())
						h.HandleInjectMessage(t.Context(), prompts[0])
					} else {
						require.Empty(t, prompts)
						require.Len(t, disconnects, 1)
						require.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_IDLE_TIMEOUT, disconnects[0].Type)
					}
				}
			})
		})
	}
}

func TestIdleTimeout_RawSpeechResetsPromptCount(t *testing.T) {
	for _, tc := range []struct {
		name    string
		script  string
		interim bool
		stale   bool
		vadOnly bool
		reset   bool
	}{
		{name: "um interim", script: "um", interim: true, reset: true},
		{name: "um final", script: "um", reset: true},
		{name: "hmm interim", script: "hmm", interim: true, reset: true},
		{name: "hmm final", script: "hmm", reset: true},
		{name: "meaningful interim", script: "wait please", interim: true, reset: true},
		{name: "meaningful final", script: "wait please", reset: true},
		{name: "blank interim", script: " \t ", interim: true},
		{name: "blank final", script: " \t "},
		{name: "stale interim", script: "wait please", interim: true, stale: true},
		{name: "stale final", script: "wait please", stale: true},
		{name: "VAD only", vadOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
				r.endOfSpeechExecutor = &recordingEOSExecutor{}
				r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
				defer r.sessionLifecycle.CloseTimeouts()
				defer r.messageLifecycle.CancelInterruption()
				h := requestorDispatchHandler{r: r}
				timeout, limit := uint64(1), uint64(2)
				require.NoError(t, r.sessionLifecycle.ConfigureTimeouts(t.Context(), r.GetID(), &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &timeout, IdleTimeoutBackoff: &limit,
				}, r.OnPacket))
				previous := r.GetID()
				r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: previous, Text: "Spoken response"})
				require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: previous, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "Spoken response"},
				}))
				require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: previous, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
				}))
				require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(previous))
				for _, packet := range drainEgressPackets(r) {
					if start, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
						h.HandleStartIdleTimeout(t.Context(), start)
					}
				}
				time.Sleep(time.Second)
				synctest.Wait()
				for _, packet := range drainEgressPackets(r) {
					if expiry, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
						h.HandleIdleTimeoutExpired(t.Context(), expiry)
					}
				}
				require.Equal(t, adapter_lifecycle.MessageStateAssistantPrompted, r.messageLifecycle.State())
				for _, packet := range drainEgressPackets(r) {
					if prompt, ok := packet.(internal_type.InjectMessagePacket); ok {
						h.HandleInjectMessage(t.Context(), prompt)
					}
				}
				require.NoError(t, r.messageLifecycle.OnSpeechStarted(r.GetID()))
				require.Equal(t, uint64(1), r.sessionLifecycle.IdleTimeoutCount())
				drainEgressPackets(r)
				if tc.vadOnly {
					h.HandleInterruptionDetected(t.Context(), internal_type.InterruptionDetectedPacket{
						ContextID: r.GetID(), Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
					})
				} else {
					contextID := r.GetID()
					if tc.stale {
						contextID = previous
					}
					h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{
						ContextID: contextID, Script: tc.script, Interim: tc.interim,
					})
				}
				synctest.Wait()
				resetPackets := 0
				for _, packet := range drainEgressPackets(r) {
					if stop, ok := packet.(internal_type.StopIdleTimeoutPacket); ok {
						if stop.ResetCount {
							resetPackets++
						}
						h.HandleStopIdleTimeout(t.Context(), stop)
					}
				}
				if tc.reset {
					require.Positive(t, resetPackets)
					require.Zero(t, r.sessionLifecycle.IdleTimeoutCount())
				} else {
					require.Zero(t, resetPackets)
					require.Equal(t, uint64(1), r.sessionLifecycle.IdleTimeoutCount())
				}
			})
		})
	}
}

func TestIdleTimeout_FillerResetPreservesCompletedPlaybackTimer(t *testing.T) {
	for _, order := range []string{"reset before completion", "completion before reset"} {
		t.Run(order, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				pending, release := make(chan struct{}), make(chan struct{})
				defer close(release)
				var r *genericRequestor
				r = newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithOnPacket(func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if stop, ok := packet.(internal_type.StopIdleTimeoutPacket); ok && stop.ResetCount {
							close(pending)
							<-release
						}
					}
					return r.OnPacket(t.Context(), packets...)
				}))
				r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
				defer r.sessionLifecycle.CloseTimeouts()
				defer r.messageLifecycle.CancelInterruption()
				h := requestorDispatchHandler{r: r}
				timeout, limit := uint64(1), uint64(2)
				require.NoError(t, r.sessionLifecycle.ConfigureTimeouts(t.Context(), r.GetID(), &internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &timeout, IdleTimeoutBackoff: &limit,
				}, r.OnPacket))
				current := r.GetID()
				r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: current, Text: "Spoken response"})
				require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: current, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "Spoken response"},
				}))
				require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: current, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
				}))
				h.HandleInterruptionDetected(t.Context(), internal_type.InterruptionDetectedPacket{
					ContextID: current, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				synctest.Wait()
				drainEgressPackets(r)
				finished := make(chan struct{})
				go func() {
					h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: current, Script: "um", Interim: true})
					close(finished)
				}()
				<-pending
				if order == "reset before completion" {
					release <- struct{}{}
					<-finished
				}
				require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(current))
				time.Sleep(adapter_lifecycle.InterruptionDecisionWindow + time.Millisecond)
				synctest.Wait()
				require.Equal(t, adapter_lifecycle.MessageStateAssistantIdle, r.messageLifecycle.State())
				require.Equal(t, current, r.GetID(), "filler must not start another turn")
				if order == "completion before reset" {
					release <- struct{}{}
					<-finished
				}
				var operations []string
				for _, packet := range drainEgressPackets(r) {
					switch p := packet.(type) {
					case internal_type.StartIdleTimeoutPacket:
						operations = append(operations, "start")
						h.HandleStartIdleTimeout(t.Context(), p)
					case internal_type.StopIdleTimeoutPacket:
						operations = append(operations, "reset")
						require.True(t, p.ResetCount)
						h.HandleStopIdleTimeout(t.Context(), p)
					}
				}
				if order == "reset before completion" {
					require.Equal(t, []string{"reset", "start"}, operations)
				} else {
					require.Equal(t, []string{"start", "reset", "start"}, operations)
				}
				require.Zero(t, r.sessionLifecycle.IdleTimeoutCount())
				time.Sleep(2 * time.Second)
				synctest.Wait()
				var expiries []internal_type.IdleTimeoutExpiredPacket
				for _, packet := range drainEgressPackets(r) {
					if expiry, ok := packet.(internal_type.IdleTimeoutExpiredPacket); ok {
						expiries = append(expiries, expiry)
					}
				}
				require.Len(t, expiries, 1, "completed response must retain an idle timer after filler reset")
				h.HandleIdleTimeoutExpired(t.Context(), expiries[0])
				require.Equal(t, uint64(1), r.sessionLifecycle.IdleTimeoutCount())
			})
		})
	}
}

func TestIdleTimeout_QueuedPromptCannotReplaceAdmittedSpeech(t *testing.T) {
	for _, promptKind := range []string{"idle", "unclear"} {
		t.Run(promptKind, func(t *testing.T) {
			for _, transcript := range []string{"interim", "final"} {
				t.Run(transcript, func(t *testing.T) {
					for _, stage := range []string{"raw admission", "listening", "finished", "generating"} {
						if stage == "raw admission" && transcript == "interim" {
							continue // The existing admission fixture blocks final transcripts only.
						}
						t.Run(stage, func(t *testing.T) {
							synctest.Test(t, func(t *testing.T) {
								r := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat")
								r.sessionLifecycle = adapter_lifecycle.NewSessionLifecycle()
								defer r.sessionLifecycle.CloseTimeouts()
								defer r.messageLifecycle.CancelInterruption()
								defer r.messageLifecycle.StopUnclearInput()
								eos := &recordingEOSExecutor{}
								r.endOfSpeechExecutor = eos
								h := requestorDispatchHandler{r: r}
								if promptKind == "idle" {
									timeout, limit := uint64(1), uint64(2)
									require.NoError(t, r.sessionLifecycle.ConfigureTimeouts(t.Context(), r.GetID(), &internal_assistant_entity.AssistantDeploymentBehavior{
										IdleTimeout: &timeout, IdleTimeoutBackoff: &limit,
									}, r.OnPacket))
									contextID := r.GetID()
									r.messageLifecycle.OnGenerationCompleted(internal_type.LLMResponseDonePacket{ContextID: contextID, Text: "Spoken response"})
									require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
										Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Text{Text: "Spoken response"},
									}))
									require.NoError(t, r.messageLifecycle.SendAssistantMessage(&protos.ConversationAssistantMessage{
										Id: contextID, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
									}))
									require.NoError(t, r.messageLifecycle.OnPlaybackCompleted(contextID))
									for _, packet := range drainEgressPackets(r) {
										if start, ok := packet.(internal_type.StartIdleTimeoutPacket); ok {
											h.HandleStartIdleTimeout(t.Context(), start)
										}
									}
								} else {
									_, err := r.messageLifecycle.OnUserTurnStarted(r.GetID(), "test", "stt", "wait")
									require.NoError(t, err)
									h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: r.GetID(), Script: "wait", Interim: true})
								}
								time.Sleep(time.Second)
								synctest.Wait()
								for _, packet := range drainEgressPackets(r) {
									switch expiry := packet.(type) {
									case internal_type.IdleTimeoutExpiredPacket:
										h.HandleIdleTimeoutExpired(t.Context(), expiry)
									case internal_type.UnclearInputExpiredPacket:
										h.HandleUnclearInputExpired(t.Context(), expiry)
									}
								}
								var prompts []internal_type.InjectMessagePacket
								for _, packet := range drainEgressPackets(r) {
									if prompt, ok := packet.(internal_type.InjectMessagePacket); ok {
										prompts = append(prompts, prompt)
									}
								}
								require.Len(t, prompts, 1)
								require.Equal(t, adapter_lifecycle.MessageStateAssistantPrompted, r.messageLifecycle.State())
								assistant := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 4)}
								r.assistantExecutor = assistant
								if stage == "raw admission" {
									admission := &blockedSpeechAdmission{
										MessageLifecycle: r.messageLifecycle, admitted: make(chan struct{}), release: make(chan struct{}),
									}
									defer close(admission.release)
									r.messageLifecycle = admission
									finished := make(chan struct{})
									go func() {
										h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: prompts[0].ContextID, Script: "I am here"})
										close(finished)
									}()
									<-admission.admitted
									current := r.GetID()
									require.NotEqual(t, prompts[0].ContextID, current)
									h.HandleInjectMessage(t.Context(), prompts[0])
									synctest.Wait()
									require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
									require.Empty(t, assistant.packets)
									admission.release <- struct{}{}
									<-finished
									executed := eos.snapshotExecuted()
									require.NotEmpty(t, executed)
									require.Equal(t, internal_type.SpeechToTextPacket{ContextID: current, Script: "I am here"}, executed[len(executed)-1])
									return
								}
								h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: r.GetID(), Script: "I am here", Interim: transcript == "interim"})
								current := r.GetID()
								require.NotEqual(t, prompts[0].ContextID, current)
								require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
								executed := eos.snapshotExecuted()
								require.NotEmpty(t, executed)
								require.Equal(t, internal_type.SpeechToTextPacket{ContextID: current, Script: "I am here", Interim: transcript == "interim"}, executed[len(executed)-1])
								if stage != "listening" {
									h.HandleEndOfSpeech(t.Context(), internal_type.EndOfSpeechPacket{ContextID: current, Speech: "I am here"})
									require.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
								}
								if stage == "generating" {
									h.HandleUserInput(t.Context(), internal_type.UserInputPacket{ContextID: current, Text: "I am here"})
									synctest.Wait()
									require.Equal(t, adapter_lifecycle.MessageStateAssistantGenerating, r.messageLifecycle.State())
									require.Len(t, assistant.packets, 1)
									require.Equal(t, internal_type.UserInputPacket{ContextID: current, Text: "I am here"}, <-assistant.packets)
								}
								state := r.messageLifecycle.State()
								h.HandleInjectMessage(t.Context(), prompts[0])
								synctest.Wait()
								require.Equal(t, current, r.GetID())
								require.Equal(t, state, r.messageLifecycle.State())
								require.Empty(t, assistant.packets, "superseded prompt must not reach the assistant executor")
							})
						})
					}
				})
			}
		})
	}
}
