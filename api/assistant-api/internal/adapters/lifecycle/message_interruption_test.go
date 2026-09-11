package lifecycle

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessageInterruption_PlaybackControlOrdersDelayedPauseBeforeFlushCommit(t *testing.T) {
	l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
	l.ConfigureInterruption(true)
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.AssistantGenerating("assistant"))
	require.NoError(t, l.AssistantSpeaking("assistant"))
	_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	confirmed := make(chan *internal_type.TurnChangePacket, 1)
	pauseResult, flushResult := make(chan error, 1), make(chan error, 1)
	flushStarted, flushDone := make(chan struct{}), make(chan struct{})
	committedTurns := make(chan internal_type.TurnChangePacket, 1)
	applied := make(chan proto.Message, 2)
	releasePause := make(chan struct{}, 1)
	defer close(releasePause)
	go func() {
		pauseResult <- l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: pause.ContextID}, func(packet proto.Message) error {
			decision, _, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{
				ContextID: "assistant", Script: "wait", Interim: true,
			}, true)
			confirmed <- decision
			<-releasePause
			applied <- packet
			return nil
		})
	}()
	decision := <-confirmed
	require.NotNil(t, decision, "speech must be able to reenter lifecycle methods during Pause I/O")
	require.True(t, l.BeginInterruptedTurn(*decision))
	go func() {
		defer close(flushDone)
		close(flushStarted)
		if err := l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "assistant"}, func(packet proto.Message) error {
			applied <- packet
			return nil
		}); err != nil {
			flushResult <- err
			return
		}
		committed, ok := l.CommitInterruptedTurn(*decision)
		if !ok {
			flushResult <- errors.New("flush completion did not commit the confirmed turn")
			return
		}
		committedTurns <- committed
		flushResult <- nil
	}()
	<-flushStarted
	select {
	case <-flushDone:
		t.Error("flush completed while Pause transport application was still blocked")
	case <-time.After(20 * time.Millisecond):
	}
	assert.Empty(t, applied, "Flush must not apply before the pending Pause")
	assert.Empty(t, committedTurns)
	assert.Equal(t, "assistant", l.ContextID())
	releasePause <- struct{}{}
	require.NoError(t, <-pauseResult)
	require.NoError(t, <-flushResult)
	<-flushDone
	require.Len(t, applied, 2)
	assert.Equal(t, &protos.ConversationPlaybackPause{Id: "assistant"}, <-applied)
	assert.Equal(t, &protos.ConversationPlaybackFlush{Id: "assistant"}, <-applied)
	require.Len(t, committedTurns, 1)
	committed := <-committedTurns
	assert.Equal(t, committed.ContextID, l.ContextID())
	assert.NotEqual(t, "assistant", committed.ContextID)
	assert.Len(t, l.FinishInterruptedTurn(committed), 2)
	assert.Empty(t, l.FinishInterruptedTurn(committed))
}

func TestMessageInterruption_PlaybackControlRejectsPauseAfterFlush(t *testing.T) {
	for _, phase := range []string{"reserved", "committed", "finished"} {
		t.Run(phase, func(t *testing.T) {
			l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
			l.ConfigureInterruption(true)
			t.Cleanup(func() { l.CancelInterruption() })
			require.NoError(t, l.AssistantGenerating("assistant"))
			require.NoError(t, l.AssistantSpeaking("assistant"))
			_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			decision, _, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}, true)
			require.NotNil(t, decision)
			require.True(t, l.BeginInterruptedTurn(*decision))
			var applied []proto.Message
			require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "assistant"}, func(packet proto.Message) error {
				applied = append(applied, packet)
				return nil
			}))
			if phase != "reserved" {
				committed, ok := l.CommitInterruptedTurn(*decision)
				require.True(t, ok)
				if phase == "finished" {
					require.Len(t, l.FinishInterruptedTurn(committed), 2)
				}
			}
			current, state := l.ContextID(), l.State()
			err := l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: pause.ContextID}, func(packet proto.Message) error {
				applied = append(applied, packet)
				return nil
			})
			assert.ErrorIs(t, err, ErrStaleContext)
			assert.Equal(t, []proto.Message{&protos.ConversationPlaybackFlush{Id: "assistant"}}, applied)
			assert.Equal(t, current, l.ContextID())
			assert.Equal(t, state, l.State())
		})
	}
}

func TestMessageInterruption_MeaningfulSpeechCommitsAndReplaysOnce(t *testing.T) {
	l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
	assert.False(t, l.InterruptionEnabled())
	l.ConfigureInterruption(true)
	assert.True(t, l.InterruptionEnabled())
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.AssistantGenerating("assistant"))
	require.NoError(t, l.AssistantSpeaking("assistant"))
	start := internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}
	event, packets, pause := l.ObserveVAD(start)
	require.NotNil(t, pause)
	assert.Empty(t, event.ContextID)
	assert.Empty(t, packets)
	assert.Equal(t, "assistant", pause.ContextID)
	assert.Equal(t, "assistant", l.ContextID())
	assert.Equal(t, MessageStateAssistantSpeaking, l.State())

	interim := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait please", Interim: true}
	decision, admitted, startUnclear := l.ObserveSpeech(interim, true)
	require.NotNil(t, decision)
	assert.False(t, admitted)
	assert.False(t, startUnclear)
	assert.True(t, decision.InterruptionDecision)
	assert.Equal(t, pause.Sequence, decision.InterruptionSequence)
	assert.Equal(t, "assistant", decision.PreviousContextID)
	assert.Equal(t, string(MessageStateAssistantSpeaking), decision.PreviousState)
	assert.Equal(t, interim.Script, decision.Text)
	assert.False(t, l.IsCommittedInterruption("assistant"))

	require.True(t, l.BeginInterruptedTurn(*decision))
	assert.Equal(t, "assistant", l.ContextID(), "reserving flush must not rotate the turn")
	final := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait please stop"}
	nextDecision, admitted, startUnclear := l.ObserveSpeech(final, true)
	assert.Nil(t, nextDecision)
	assert.False(t, admitted, "speech arriving during flush must remain held")
	assert.False(t, startUnclear)
	eos := internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: final.Script, Speechs: []internal_type.SpeechToTextPacket{final}}
	assert.True(t, l.HoldInput(eos))
	assert.True(t, l.HoldInput(internal_type.UserInputPacket{ContextID: "obsolete", Text: "discard"}))

	committed, ok := l.CommitInterruptedTurn(*decision)
	require.True(t, ok)
	assert.Equal(t, "assistant", committed.PreviousContextID)
	require.NotEmpty(t, committed.ContextID)
	require.NotEqual(t, "assistant", committed.ContextID)
	assert.Equal(t, committed.ContextID, l.ContextID())
	assert.Equal(t, MessageStateUserListening, l.State())
	input := internal_type.UserInputPacket{ContextID: "assistant", Text: final.Script}
	assert.True(t, l.HoldInput(input), "input stays held until downstream turn setup finishes")

	start.ContextID = committed.ContextID
	interim.ContextID = committed.ContextID
	final.ContextID = committed.ContextID
	input.ContextID = committed.ContextID
	wantEOS := internal_type.EndOfSpeechPacket{ContextID: committed.ContextID, Speech: final.Script, Speechs: []internal_type.SpeechToTextPacket{final}}
	assert.Equal(t, []internal_type.Packet{start, interim, final, wantEOS, input}, l.FinishInterruptedTurn(committed))
	assert.Equal(t, "assistant", eos.Speechs[0].ContextID, "replay must not mutate the caller's transcript slice")
	assert.Empty(t, l.FinishInterruptedTurn(committed))
	assert.True(t, l.IsCommittedInterruption(committed.ContextID))
	assert.False(t, l.IsCommittedInterruption("assistant"))
	assert.False(t, l.HoldInput(input))

	nextDecision, admitted, startUnclear = l.ObserveSpeech(internal_type.SpeechToTextPacket{
		ContextID: "assistant", Script: "another word", Interim: true,
	}, true)
	assert.Nil(t, nextDecision)
	assert.True(t, admitted)
	assert.True(t, startUnclear)
	nextDecision, admitted, startUnclear = l.ObserveSpeech(final, true)
	assert.Nil(t, nextDecision)
	assert.True(t, admitted)
	assert.False(t, startUnclear)
	assert.False(t, l.IsCommittedInterruption(committed.ContextID))
}

func TestMessageInterruption_FillerRecovery(t *testing.T) {
	for _, scenario := range []struct {
		name string
		text string
	}{
		{name: "empty"},
		{name: "fillers", text: "Um, HMM... uh!"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
				l.ConfigureInterruption(true)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.AssistantGenerating("assistant"))
				require.NoError(t, l.AssistantSpeaking("assistant"))
				_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				l.ArmInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				for _, interim := range []bool{true, false} {
					decision, admitted, startUnclear := l.ObserveSpeech(internal_type.SpeechToTextPacket{
						ContextID: "assistant", Script: scenario.text, Interim: interim,
					}, true)
					assert.Nil(t, decision)
					assert.False(t, admitted)
					assert.False(t, startUnclear)
				}
				assert.True(t, l.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: scenario.text}))
				l.ObserveVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
				})
				time.Sleep(InterruptionDecisionWindow - time.Millisecond)
				synctest.Wait()
				assert.Empty(t, expired)
				time.Sleep(time.Millisecond)
				synctest.Wait()
				require.Len(t, expired, 1)
				packet := <-expired
				assert.Equal(t, *pause, packet)
				decision, continueContext := l.ExpireInterruption(packet)
				assert.Nil(t, decision)
				assert.Equal(t, "assistant", continueContext)
				assert.Equal(t, "assistant", l.ContextID())
				assert.Equal(t, MessageStateAssistantSpeaking, l.State())
				assert.False(t, l.IsCommittedInterruption("assistant"))
				decision, continueContext = l.ExpireInterruption(packet)
				assert.Nil(t, decision)
				assert.Empty(t, continueContext)
				assert.Empty(t, l.CancelInterruption(), "recovery must release the pending pause")
				time.Sleep(InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
			})
		})
	}
}

func TestMessageInterruption_StaleTimerAfterCancelAndNewCandidate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
		l.ConfigureInterruption(true)
		t.Cleanup(func() { l.CancelInterruption() })
		require.NoError(t, l.AssistantGenerating("assistant"))
		require.NoError(t, l.AssistantSpeaking("assistant"))
		start := internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}
		_, _, first := l.ObserveVAD(start)
		require.NotNil(t, first)
		expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
		l.ArmInterruption(first.ContextID, first.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		require.Len(t, expired, 1)
		stale := <-expired
		assert.Equal(t, "assistant", l.CancelInterruption())
		_, _, current := l.ObserveVAD(start)
		require.NotNil(t, current)
		require.NotEqual(t, stale.Sequence, current.Sequence)
		l.ArmInterruption(current.ContextID, current.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
		decision, continueContext := l.ExpireInterruption(stale)
		assert.Nil(t, decision)
		assert.Empty(t, continueContext)
		assert.Nil(t, l.FailInterruptionPause(stale))
		assert.False(t, l.BeginInterruptedTurn(internal_type.TurnChangePacket{
			InterruptionDecision: true, InterruptionSequence: stale.Sequence, PreviousContextID: stale.ContextID,
		}))
		assert.Equal(t, "assistant", l.ContextID())
		assert.Equal(t, MessageStateAssistantSpeaking, l.State())
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		require.Len(t, expired, 1)
		packet := <-expired
		assert.Equal(t, *current, packet)
		decision, continueContext = l.ExpireInterruption(packet)
		require.Nil(t, decision)
		assert.Equal(t, "assistant", continueContext)
		decision, admitted, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{
			ContextID: "assistant", Script: "stop", Interim: true,
		}, true)
		require.NotNil(t, decision)
		assert.False(t, admitted)
		assert.Greater(t, decision.InterruptionSequence, current.Sequence)
		require.True(t, l.BeginInterruptedTurn(*decision))
		committed, ok := l.CommitInterruptedTurn(*decision)
		require.True(t, ok)
		start.ContextID = committed.ContextID
		assert.Equal(t, []internal_type.Packet{start, internal_type.SpeechToTextPacket{
			ContextID: committed.ContextID, Script: "stop", Interim: true,
		}}, l.FinishInterruptedTurn(committed))
		assert.True(t, l.IsCommittedInterruption(committed.ContextID))
	})
}

func TestMessageInterruption_DuplicateCommitRejected(t *testing.T) {
	l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
	l.ConfigureInterruption(true)
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.AssistantGenerating("assistant"))
	require.NoError(t, l.AssistantSpeaking("assistant"))
	_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	decision, _, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
	require.NotNil(t, decision)
	require.True(t, l.BeginInterruptedTurn(*decision))
	assert.False(t, l.BeginInterruptedTurn(*decision), "only one caller may reserve flush")
	committed, ok := l.CommitInterruptedTurn(*decision)
	require.True(t, ok)
	_, duplicate := l.CommitInterruptedTurn(*decision)
	assert.False(t, duplicate, "a completed flush must not allocate another turn")
	assert.Equal(t, committed.ContextID, l.ContextID())
	assert.Len(t, l.FinishInterruptedTurn(committed), 2, "a duplicate commit must not invalidate the original replay")
	assert.Empty(t, l.FinishInterruptedTurn(committed))
	assert.False(t, l.BeginInterruptedTurn(*decision))
	_, duplicate = l.CommitInterruptedTurn(*decision)
	assert.False(t, duplicate)
	assert.Equal(t, committed.ContextID, l.ContextID())
}

func TestMessageInterruption_StaleFlushCommitAfterNewTurn(t *testing.T) {
	for _, operation := range []string{"RotateContext", "AcceptUserTurn"} {
		t.Run(operation, func(t *testing.T) {
			l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
			l.ConfigureInterruption(true)
			t.Cleanup(func() { l.CancelInterruption() })
			require.NoError(t, l.AssistantGenerating("assistant"))
			require.NoError(t, l.AssistantSpeaking("assistant"))
			_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			decision, _, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
			require.NotNil(t, decision)
			require.True(t, l.BeginInterruptedTurn(*decision))
			// A different turn wins while the reserved flush is in flight.
			switch operation {
			case "RotateContext":
				_, _, err := l.RotateContext()
				require.NoError(t, err)
			case "AcceptUserTurn":
				_, err := l.AcceptUserTurn("assistant", string(internal_type.PacketNameUserTextReceived), "user", "new request")
				require.NoError(t, err)
			}
			current := l.ContextID()
			require.NotEqual(t, "assistant", current)
			state := l.State()
			committed, ok := l.CommitInterruptedTurn(*decision)
			assert.False(t, ok, "a stale flush must not replace the winning turn")
			assert.Equal(t, current, l.ContextID())
			assert.Equal(t, state, l.State())
			assert.Empty(t, l.FinishInterruptedTurn(committed))
			assert.False(t, l.IsCommittedInterruption(current))
			assert.False(t, l.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "new request"}), "the new turn must not inherit held input")
		})
	}
}

func TestMessageInterruption_ShutdownCancelsPendingWork(t *testing.T) {
	for _, phase := range []string{"pause", "flush reserved", "flush committed"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
				l.ConfigureInterruption(true)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.AssistantGenerating("assistant"))
				require.NoError(t, l.AssistantSpeaking("assistant"))
				_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				l.ArmInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				decision := internal_type.TurnChangePacket{
					InterruptionDecision: true, InterruptionSequence: pause.Sequence, PreviousContextID: pause.ContextID,
				}
				committed := decision
				if phase != "pause" {
					confirmed, _, _ := l.ObserveSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
					require.NotNil(t, confirmed)
					decision = *confirmed
					require.True(t, l.BeginInterruptedTurn(decision))
					if phase == "flush committed" {
						var ok bool
						committed, ok = l.CommitInterruptedTurn(decision)
						require.True(t, ok)
					}
				}
				current, state := l.ContextID(), l.State()
				continueContext := l.CancelInterruption()
				if phase == "pause" {
					assert.Equal(t, "assistant", continueContext)
				} else {
					assert.Empty(t, continueContext, "shutdown must not continue output after flush was reserved")
				}
				assert.Empty(t, l.CancelInterruption())
				l.ArmInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				time.Sleep(2 * InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
				lateDecision, continueContext := l.ExpireInterruption(*pause)
				assert.Nil(t, lateDecision)
				assert.Empty(t, continueContext)
				assert.Nil(t, l.FailInterruptionPause(*pause))
				assert.False(t, l.BeginInterruptedTurn(decision))
				_, ok := l.CommitInterruptedTurn(decision)
				assert.False(t, ok)
				assert.Empty(t, l.FinishInterruptedTurn(committed))
				assert.Equal(t, current, l.ContextID())
				assert.Equal(t, state, l.State())
				assert.False(t, l.IsCommittedInterruption(current))
				assert.False(t, l.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "late"}))
			})
		})
	}
}

func TestMessageInterruption_SynchronousSpeechDuringPause(t *testing.T) {
	for _, scenario := range []struct {
		name              string
		pauseFailed       bool
		finishBeforePause bool
	}{
		{name: "confirmed before pause succeeds"},
		{name: "confirmed before pause fails", pauseFailed: true},
		{name: "committed before pause succeeds", finishBeforePause: true},
		{name: "committed before pause fails", pauseFailed: true, finishBeforePause: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
				l.ConfigureInterruption(true)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.AssistantGenerating("assistant"))
				require.NoError(t, l.AssistantSpeaking("assistant"))
				_, _, pause := l.ObserveVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				// Pause I/O reenters speech handling before its caller can arm the deadline.
				decision, admitted, startUnclear := l.ObserveSpeech(internal_type.SpeechToTextPacket{
					ContextID: "assistant", Script: "stop", Interim: true,
				}, true)
				require.NotNil(t, decision)
				assert.False(t, admitted)
				assert.False(t, startUnclear)
				var committed internal_type.TurnChangePacket
				if scenario.finishBeforePause {
					require.True(t, l.BeginInterruptedTurn(*decision))
					var ok bool
					committed, ok = l.CommitInterruptedTurn(*decision)
					require.True(t, ok)
					require.Len(t, l.FinishInterruptedTurn(committed), 2)
				}
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				if scenario.pauseFailed {
					assert.Nil(t, l.FailInterruptionPause(*pause), "late pause failure must not create a second turn decision")
				} else {
					l.ArmInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				}
				time.Sleep(2 * InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
				if !scenario.finishBeforePause {
					assert.Equal(t, "assistant", l.ContextID())
					require.True(t, l.BeginInterruptedTurn(*decision))
					var ok bool
					committed, ok = l.CommitInterruptedTurn(*decision)
					require.True(t, ok)
					require.Len(t, l.FinishInterruptedTurn(committed), 2)
				}
				assert.Equal(t, committed.ContextID, l.ContextID())
				assert.True(t, l.IsCommittedInterruption(committed.ContextID))
				assert.Empty(t, l.FinishInterruptedTurn(committed))
				assert.False(t, l.BeginInterruptedTurn(*decision))
			})
		})
	}
}

func TestMessageInterruption_ConfirmationAndExpiryOrdering(t *testing.T) {
	for _, confirmationFirst := range []bool{true, false} {
		name := "expiry before confirmation"
		if confirmationFirst {
			name = "confirmation before expiry"
		}
		t.Run(name, func(t *testing.T) {
			message := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
			message.ConfigureInterruption(true)
			t.Cleanup(func() { message.CancelInterruption() })
			require.NoError(t, message.AssistantGenerating("assistant"))
			_, _, pause := message.ObserveVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			if !confirmationFirst {
				decision, continueContext := message.ExpireInterruption(*pause)
				require.Nil(t, decision)
				assert.Equal(t, "assistant", continueContext)
			}
			decision, admitted, _ := message.ObserveSpeech(internal_type.SpeechToTextPacket{
				ContextID: "assistant", Script: "wait", Interim: true,
			}, true)
			require.NotNil(t, decision)
			assert.False(t, admitted)
			expiredDecision, continueContext := message.ExpireInterruption(*pause)
			assert.Nil(t, expiredDecision)
			assert.Empty(t, continueContext)
			assert.ErrorIs(t, message.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "assistant"}, func(proto.Message) error {
				t.Error("delayed continue must not resume a confirmed interruption")
				return nil
			}), ErrStaleContext)
			require.True(t, message.BeginInterruptedTurn(*decision))
			committed, ok := message.CommitInterruptedTurn(*decision)
			require.True(t, ok)
			assert.Len(t, message.FinishInterruptedTurn(committed), 2)
			assert.False(t, message.BeginInterruptedTurn(*decision))
			assert.NotEqual(t, "assistant", message.ContextID())
		})
	}
}

func TestMessageInterruption_ResumedSpeechKeepsVADEndAndRejectsStaleText(t *testing.T) {
	message := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
	message.ConfigureInterruption(true)
	t.Cleanup(func() { message.CancelInterruption() })
	require.NoError(t, message.AssistantGenerating("assistant"))
	start := internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}
	_, _, pause := message.ObserveVAD(start)
	require.NotNil(t, pause)
	decision, continueContext := message.ExpireInterruption(*pause)
	require.Nil(t, decision)
	require.Equal(t, "assistant", continueContext)
	_, packets, repeatedPause := message.ObserveVAD(start)
	assert.Nil(t, repeatedPause)
	assert.Empty(t, packets)
	end := start
	end.Event = internal_type.InterruptionEventEnd
	_, packets, repeatedPause = message.ObserveVAD(end)
	assert.Nil(t, repeatedPause)
	assert.Equal(t, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: "assistant"}}, packets)
	for _, transcript := range []internal_type.SpeechToTextPacket{
		{ContextID: "stale", Script: "wait"},
		{Script: "wait"},
		{ContextID: "assistant", Script: "um, hmm"},
	} {
		decision, admitted, startUnclear := message.ObserveSpeech(transcript, true)
		assert.Nil(t, decision)
		assert.False(t, admitted)
		assert.False(t, startUnclear)
	}
	decision, admitted, _ := message.ObserveSpeech(internal_type.SpeechToTextPacket{
		ContextID: "assistant", Script: "stop", Interim: true,
	}, true)
	require.NotNil(t, decision)
	assert.False(t, admitted)
	require.True(t, message.BeginInterruptedTurn(*decision))
	committed, ok := message.CommitInterruptedTurn(*decision)
	require.True(t, ok)
	start.ContextID = committed.ContextID
	end.ContextID = committed.ContextID
	assert.Equal(t, []internal_type.Packet{start, end, internal_type.SpeechToTextPacket{
		ContextID: committed.ContextID, Script: "stop", Interim: true,
	}}, message.FinishInterruptedTurn(committed))
}

func TestMessageInterruption_ResumedSpeechCannotRotateReplacementContext(t *testing.T) {
	for _, replacement := range []string{"rotation", "cancellation", "user turn", "prompt"} {
		t.Run(replacement, func(t *testing.T) {
			message := NewMessageLifecycleWithContext("assistant", type_enums.AudioMode)
			message.ConfigureInterruption(true)
			t.Cleanup(func() { message.CancelInterruption() })
			require.NoError(t, message.AssistantGenerating("assistant"))
			_, _, pause := message.ObserveVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			_, continueContext := message.ExpireInterruption(*pause)
			require.Equal(t, "assistant", continueContext)
			switch replacement {
			case "rotation":
				_, _, err := message.RotateContext()
				require.NoError(t, err)
			case "cancellation":
				assert.Empty(t, message.CancelInterruption())
			case "user turn":
				_, err := message.AcceptUserTurn("assistant", "test", "text", "new input")
				require.NoError(t, err)
			case "prompt":
				require.NoError(t, message.AssistantFinished("assistant"))
				require.NoError(t, message.AssistantIdle("assistant"))
				_, _, err := message.Prompt(internal_type.IdleTimeoutExpiredPacket{ContextID: "assistant"})
				require.NoError(t, err)
			}
			currentContext := message.ContextID()
			require.NoError(t, message.AssistantGenerating(currentContext))
			decision, admitted, _ := message.ObserveSpeech(internal_type.SpeechToTextPacket{
				ContextID: "assistant", Script: "stale stop", Interim: true,
			}, true)
			assert.Nil(t, decision)
			if replacement != "cancellation" {
				assert.False(t, admitted)
			}
			assert.Equal(t, currentContext, message.ContextID())
			decision, _, _ = message.ObserveSpeech(internal_type.SpeechToTextPacket{
				ContextID: currentContext, Script: "ungated", Interim: true,
			}, true)
			assert.Nil(t, decision)
		})
	}
}
