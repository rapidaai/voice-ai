package lifecycle

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMessageInterruption_PlaybackControlOrdersDelayedPauseBeforeFlushCommit(t *testing.T) {
	confirmed := make(chan *internal_type.TurnChangePacket, 1)
	applied := make(chan proto.Message, 2)
	releasePause := make(chan struct{}, 1)
	defer close(releasePause)
	var l *messageLifecycle
	l = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
		WithSend(func(packet proto.Message) error {
			if _, pause := packet.(*protos.ConversationPlaybackPause); pause {
				decision, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{
					ContextID: "assistant", Script: "wait", Interim: true,
				}, true)
				confirmed <- decision
				<-releasePause
			}
			applied <- packet
			return nil
		})).(*messageLifecycle)
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.OnGenerationStarted("assistant"))
	require.NoError(t, l.OnSpeechStarted("assistant"))
	_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	pauseResult, flushResult := make(chan error, 1), make(chan error, 1)
	flushStarted, flushDone := make(chan struct{}), make(chan struct{})
	committedTurns := make(chan internal_type.TurnChangePacket, 1)
	go func() {
		pauseResult <- l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: pause.ContextID})
	}()
	decision := <-confirmed
	require.NotNil(t, decision, "speech must be able to reenter lifecycle methods during Pause I/O")
	require.True(t, l.beginInterruptedTurn(*decision))
	go func() {
		defer close(flushDone)
		close(flushStarted)
		if err := l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "assistant"}); err != nil {
			flushResult <- err
			return
		}
		committed, ok := l.commitInterruptedTurn(*decision)
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
	assert.Len(t, l.finishInterruptedTurn(committed), 2)
	assert.Empty(t, l.finishInterruptedTurn(committed))
}

func TestMessageInterruption_PlaybackControlRejectsPauseAfterFlush(t *testing.T) {
	for _, phase := range []string{"reserved", "committed", "finished"} {
		t.Run(phase, func(t *testing.T) {
			var applied []proto.Message
			l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
				WithSend(func(packet proto.Message) error {
					applied = append(applied, packet)
					return nil
				})).(*messageLifecycle)
			t.Cleanup(func() { l.CancelInterruption() })
			require.NoError(t, l.OnGenerationStarted("assistant"))
			require.NoError(t, l.OnSpeechStarted("assistant"))
			_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			decision, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}, true)
			require.NotNil(t, decision)
			require.True(t, l.beginInterruptedTurn(*decision))
			require.NoError(t, l.SendPlaybackControl(&protos.ConversationPlaybackFlush{Id: "assistant"}))
			if phase != "reserved" {
				committed, ok := l.commitInterruptedTurn(*decision)
				require.True(t, ok)
				if phase == "finished" {
					require.Len(t, l.finishInterruptedTurn(committed), 2)
				}
			}
			current, state := l.ContextID(), l.State()
			err := l.SendPlaybackControl(&protos.ConversationPlaybackPause{Id: pause.ContextID})
			assert.ErrorIs(t, err, ErrStaleContext)
			assert.Equal(t, []proto.Message{&protos.ConversationPlaybackFlush{Id: "assistant"}}, applied)
			assert.Equal(t, current, l.ContextID())
			assert.Equal(t, state, l.State())
		})
	}
}

func TestMessageInterruption_MeaningfulSpeechCommitsAndReplaysOnce(t *testing.T) {
	assert.False(t, NewMessageLifecycle().InterruptionEnabled())
	l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
	assert.True(t, l.InterruptionEnabled())
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.OnGenerationStarted("assistant"))
	require.NoError(t, l.OnSpeechStarted("assistant"))
	start := internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}
	event, packets, pause := l.observeVAD(start)
	require.NotNil(t, pause)
	assert.Empty(t, event.ContextID)
	assert.Empty(t, packets)
	assert.Equal(t, "assistant", pause.ContextID)
	assert.Equal(t, "assistant", l.ContextID())
	assert.Equal(t, MessageStateAssistantSpeaking, l.State())

	interim := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait please", Interim: true}
	decision, admitted := l.OnUserSpeech(interim, true)
	require.NotNil(t, decision)
	assert.Empty(t, admitted.ContextID)
	assert.True(t, decision.InterruptionDecision)
	assert.Equal(t, pause.Sequence, decision.InterruptionSequence)
	assert.Equal(t, "assistant", decision.PreviousContextID)
	assert.Equal(t, string(MessageStateAssistantSpeaking), decision.PreviousState)
	assert.Equal(t, interim.Script, decision.Text)

	require.True(t, l.beginInterruptedTurn(*decision))
	assert.Equal(t, "assistant", l.ContextID(), "reserving flush must not rotate the turn")
	final := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait please stop"}
	nextDecision, admitted := l.OnUserSpeech(final, true)
	assert.Nil(t, nextDecision)
	assert.Empty(t, admitted.ContextID, "speech arriving during flush must remain held")
	eos := internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: final.Script, Speechs: []internal_type.SpeechToTextPacket{final}}
	assert.True(t, l.HoldInput(eos))
	assert.True(t, l.HoldInput(internal_type.UserInputPacket{ContextID: "obsolete", Text: "discard"}))

	committed, ok := l.commitInterruptedTurn(*decision)
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
	assert.Equal(t, []internal_type.Packet{start, interim, final, wantEOS, input}, l.finishInterruptedTurn(committed))
	assert.Equal(t, "assistant", eos.Speechs[0].ContextID, "replay must not mutate the caller's transcript slice")
	assert.Empty(t, l.finishInterruptedTurn(committed))
	assert.False(t, l.HoldInput(input))

	nextDecision, admitted = l.OnUserSpeech(internal_type.SpeechToTextPacket{
		ContextID: "assistant", Script: "another word", Interim: true,
	}, true)
	assert.Nil(t, nextDecision)
	assert.Equal(t, committed.ContextID, admitted.ContextID)
	nextDecision, admitted = l.OnUserSpeech(final, true)
	assert.Nil(t, nextDecision)
	assert.Equal(t, committed.ContextID, admitted.ContextID)
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
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.OnGenerationStarted("assistant"))
				require.NoError(t, l.OnSpeechStarted("assistant"))
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				l.armInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				for _, interim := range []bool{true, false} {
					decision, admitted := l.OnUserSpeech(internal_type.SpeechToTextPacket{
						ContextID: "assistant", Script: scenario.text, Interim: interim,
					}, true)
					assert.Nil(t, decision)
					assert.Empty(t, admitted.ContextID)
				}
				assert.True(t, l.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: scenario.text}))
				l.observeVAD(internal_type.InterruptionDetectedPacket{
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
				continueContext := l.OnInterruptionExpired(packet)
				assert.Equal(t, "assistant", continueContext)
				assert.Equal(t, "assistant", l.ContextID())
				assert.Equal(t, MessageStateAssistantSpeaking, l.State())
				continueContext = l.OnInterruptionExpired(packet)
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
		l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
		t.Cleanup(func() { l.CancelInterruption() })
		require.NoError(t, l.OnGenerationStarted("assistant"))
		require.NoError(t, l.OnSpeechStarted("assistant"))
		start := internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}
		_, _, first := l.observeVAD(start)
		require.NotNil(t, first)
		expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
		l.armInterruption(first.ContextID, first.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		require.Len(t, expired, 1)
		stale := <-expired
		assert.Equal(t, "assistant", l.CancelInterruption())
		_, _, current := l.observeVAD(start)
		require.NotNil(t, current)
		require.NotEqual(t, stale.Sequence, current.Sequence)
		l.armInterruption(current.ContextID, current.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
		continueContext := l.OnInterruptionExpired(stale)
		assert.Empty(t, continueContext)
		assert.Nil(t, l.failInterruptionPause(stale))
		assert.False(t, l.beginInterruptedTurn(internal_type.TurnChangePacket{
			InterruptionDecision: true, InterruptionSequence: stale.Sequence, PreviousContextID: stale.ContextID,
		}))
		assert.Equal(t, "assistant", l.ContextID())
		assert.Equal(t, MessageStateAssistantSpeaking, l.State())
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		require.Len(t, expired, 1)
		packet := <-expired
		assert.Equal(t, *current, packet)
		continueContext = l.OnInterruptionExpired(packet)
		assert.Equal(t, "assistant", continueContext)
		decision, admitted := l.OnUserSpeech(internal_type.SpeechToTextPacket{
			ContextID: "assistant", Script: "stop", Interim: true,
		}, true)
		require.NotNil(t, decision)
		assert.Empty(t, admitted.ContextID)
		assert.Greater(t, decision.InterruptionSequence, current.Sequence)
		require.True(t, l.beginInterruptedTurn(*decision))
		committed, ok := l.commitInterruptedTurn(*decision)
		require.True(t, ok)
		start.ContextID = committed.ContextID
		assert.Equal(t, []internal_type.Packet{start, internal_type.SpeechToTextPacket{
			ContextID: committed.ContextID, Script: "stop", Interim: true,
		}}, l.finishInterruptedTurn(committed))
	})
}

func TestMessageInterruption_DuplicateCommitRejected(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.OnGenerationStarted("assistant"))
	require.NoError(t, l.OnSpeechStarted("assistant"))
	_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	decision, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
	require.NotNil(t, decision)
	require.True(t, l.beginInterruptedTurn(*decision))
	assert.False(t, l.beginInterruptedTurn(*decision), "only one caller may reserve flush")
	committed, ok := l.commitInterruptedTurn(*decision)
	require.True(t, ok)
	_, duplicate := l.commitInterruptedTurn(*decision)
	assert.False(t, duplicate, "a completed flush must not allocate another turn")
	assert.Equal(t, committed.ContextID, l.ContextID())
	assert.Len(t, l.finishInterruptedTurn(committed), 2, "a duplicate commit must not invalidate the original replay")
	assert.Empty(t, l.finishInterruptedTurn(committed))
	assert.False(t, l.beginInterruptedTurn(*decision))
	_, duplicate = l.commitInterruptedTurn(*decision)
	assert.False(t, duplicate)
	assert.Equal(t, committed.ContextID, l.ContextID())
}

func TestMessageInterruption_StaleFlushCommitAfterNewTurn(t *testing.T) {
	l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
	t.Cleanup(func() { l.CancelInterruption() })
	require.NoError(t, l.OnGenerationStarted("assistant"))
	require.NoError(t, l.OnSpeechStarted("assistant"))
	_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	decision, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
	require.NotNil(t, decision)
	require.True(t, l.beginInterruptedTurn(*decision))
	// A different turn wins while the reserved flush is in flight.
	_, err := l.OnUserTurnStarted("assistant", string(internal_type.PacketNameUserTextReceived), "user", "new request")
	require.NoError(t, err)
	current := l.ContextID()
	require.NotEqual(t, "assistant", current)
	state := l.State()
	committed, ok := l.commitInterruptedTurn(*decision)
	assert.False(t, ok, "a stale flush must not replace the winning turn")
	assert.Equal(t, current, l.ContextID())
	assert.Equal(t, state, l.State())
	assert.Empty(t, l.finishInterruptedTurn(committed))
	assert.False(t, l.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "new request"}), "the new turn must not inherit held input")
}

func TestMessageInterruption_ShutdownCancelsPendingWork(t *testing.T) {
	for _, phase := range []string{"pause", "flush reserved", "flush committed"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.OnGenerationStarted("assistant"))
				require.NoError(t, l.OnSpeechStarted("assistant"))
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				l.armInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				decision := internal_type.TurnChangePacket{
					InterruptionDecision: true, InterruptionSequence: pause.Sequence, PreviousContextID: pause.ContextID,
				}
				committed := decision
				if phase != "pause" {
					confirmed, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "stop", Interim: true}, true)
					require.NotNil(t, confirmed)
					decision = *confirmed
					require.True(t, l.beginInterruptedTurn(decision))
					if phase == "flush committed" {
						var ok bool
						committed, ok = l.commitInterruptedTurn(decision)
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
				l.armInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				time.Sleep(2 * InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
				continueContext = l.OnInterruptionExpired(*pause)
				assert.Empty(t, continueContext)
				assert.Nil(t, l.failInterruptionPause(*pause))
				assert.False(t, l.beginInterruptedTurn(decision))
				_, ok := l.commitInterruptedTurn(decision)
				assert.False(t, ok)
				assert.Empty(t, l.finishInterruptedTurn(committed))
				assert.Equal(t, current, l.ContextID())
				assert.Equal(t, state, l.State())
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
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
				t.Cleanup(func() { l.CancelInterruption() })
				require.NoError(t, l.OnGenerationStarted("assistant"))
				require.NoError(t, l.OnSpeechStarted("assistant"))
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				// Pause I/O reenters speech handling before its caller can arm the deadline.
				decision, admitted := l.OnUserSpeech(internal_type.SpeechToTextPacket{
					ContextID: "assistant", Script: "stop", Interim: true,
				}, true)
				require.NotNil(t, decision)
				assert.Empty(t, admitted.ContextID)
				var committed internal_type.TurnChangePacket
				if scenario.finishBeforePause {
					require.True(t, l.beginInterruptedTurn(*decision))
					var ok bool
					committed, ok = l.commitInterruptedTurn(*decision)
					require.True(t, ok)
					require.Len(t, l.finishInterruptedTurn(committed), 2)
				}
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 2)
				if scenario.pauseFailed {
					assert.Nil(t, l.failInterruptionPause(*pause), "late pause failure must not create a second turn decision")
				} else {
					l.armInterruption(pause.ContextID, pause.Sequence, func(p internal_type.InterruptionDecisionExpiredPacket) { expired <- p })
				}
				time.Sleep(2 * InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
				if !scenario.finishBeforePause {
					assert.Equal(t, "assistant", l.ContextID())
					require.True(t, l.beginInterruptedTurn(*decision))
					var ok bool
					committed, ok = l.commitInterruptedTurn(*decision)
					require.True(t, ok)
					require.Len(t, l.finishInterruptedTurn(committed), 2)
				}
				assert.Equal(t, committed.ContextID, l.ContextID())
				assert.Empty(t, l.finishInterruptedTurn(committed))
				assert.False(t, l.beginInterruptedTurn(*decision))
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
			message := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
				WithSend(func(proto.Message) error {
					t.Error("delayed continue must not resume a confirmed interruption")
					return nil
				})).(*messageLifecycle)
			t.Cleanup(func() { message.CancelInterruption() })
			require.NoError(t, message.OnGenerationStarted("assistant"))
			_, _, pause := message.observeVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			if !confirmationFirst {
				continueContext := message.OnInterruptionExpired(*pause)
				assert.Equal(t, "assistant", continueContext)
			}
			decision, admitted := message.OnUserSpeech(internal_type.SpeechToTextPacket{
				ContextID: "assistant", Script: "wait", Interim: true,
			}, true)
			require.NotNil(t, decision)
			assert.Empty(t, admitted.ContextID)
			continueContext := message.OnInterruptionExpired(*pause)
			assert.Empty(t, continueContext)
			assert.ErrorIs(t, message.SendPlaybackControl(&protos.ConversationPlaybackContinue{Id: "assistant"}), ErrStaleContext)
			require.True(t, message.beginInterruptedTurn(*decision))
			committed, ok := message.commitInterruptedTurn(*decision)
			require.True(t, ok)
			assert.Len(t, message.finishInterruptedTurn(committed), 2)
			assert.False(t, message.beginInterruptedTurn(*decision))
			assert.NotEqual(t, "assistant", message.ContextID())
		})
	}
}

func TestMessageInterruption_ResumedSpeechKeepsVADEndAndRejectsStaleText(t *testing.T) {
	message := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
	t.Cleanup(func() { message.CancelInterruption() })
	require.NoError(t, message.OnGenerationStarted("assistant"))
	start := internal_type.InterruptionDetectedPacket{
		ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	}
	_, _, pause := message.observeVAD(start)
	require.NotNil(t, pause)
	continueContext := message.OnInterruptionExpired(*pause)
	require.Equal(t, "assistant", continueContext)
	_, packets, repeatedPause := message.observeVAD(start)
	assert.Nil(t, repeatedPause)
	assert.Empty(t, packets)
	end := start
	end.Event = internal_type.InterruptionEventEnd
	_, packets, repeatedPause = message.observeVAD(end)
	assert.Nil(t, repeatedPause)
	assert.Equal(t, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: "assistant"}}, packets)
	for _, transcript := range []internal_type.SpeechToTextPacket{
		{ContextID: "stale", Script: "wait"},
		{Script: "wait"},
		{ContextID: "assistant", Script: "um, hmm"},
	} {
		decision, admitted := message.OnUserSpeech(transcript, true)
		assert.Nil(t, decision)
		assert.Empty(t, admitted.ContextID)
	}
	decision, admitted := message.OnUserSpeech(internal_type.SpeechToTextPacket{
		ContextID: "assistant", Script: "stop", Interim: true,
	}, true)
	require.NotNil(t, decision)
	assert.Empty(t, admitted.ContextID)
	require.True(t, message.beginInterruptedTurn(*decision))
	committed, ok := message.commitInterruptedTurn(*decision)
	require.True(t, ok)
	start.ContextID = committed.ContextID
	end.ContextID = committed.ContextID
	assert.Equal(t, []internal_type.Packet{start, end, internal_type.SpeechToTextPacket{
		ContextID: committed.ContextID, Script: "stop", Interim: true,
	}}, message.finishInterruptedTurn(committed))
}

func TestMessageInterruption_ResumedSpeechCannotRotateReplacementContext(t *testing.T) {
	for _, replacement := range []string{"cancellation", "user turn", "prompt"} {
		t.Run(replacement, func(t *testing.T) {
			message := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true), WithSend(func(proto.Message) error { return nil })).(*messageLifecycle)
			t.Cleanup(func() { message.CancelInterruption() })
			require.NoError(t, message.OnGenerationStarted("assistant"))
			_, _, pause := message.observeVAD(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
			require.NotNil(t, pause)
			continueContext := message.OnInterruptionExpired(*pause)
			require.Equal(t, "assistant", continueContext)
			switch replacement {
			case "cancellation":
				assert.Empty(t, message.CancelInterruption())
			case "user turn":
				_, err := message.OnUserTurnStarted("assistant", "test", "text", "new input")
				require.NoError(t, err)
			case "prompt":
				require.Len(t, message.OnGenerationCompleted(internal_type.LLMResponseDonePacket{
					ContextID: "assistant", Text: "ready",
				}), 1)
				require.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "assistant", Completed: true,
					Message: &protos.ConversationAssistantMessage_Text{Text: "ready"},
				}))
				require.NoError(t, message.OnSpeechStarted("assistant"))
				require.NoError(t, message.SendAssistantMessage(&protos.ConversationAssistantMessage{
					Id: "assistant", Completed: true,
					Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0, 0}},
				}))
				require.NoError(t, message.OnPlaybackCompleted("assistant"))
				require.Equal(t, MessageStateAssistantIdle, message.State())
				_, _, err := message.OnPrompt(internal_type.IdleTimeoutExpiredPacket{ContextID: "assistant"})
				require.NoError(t, err)
			}
			currentContext := message.ContextID()
			require.NoError(t, message.OnGenerationStarted(currentContext))
			decision, admitted := message.OnUserSpeech(internal_type.SpeechToTextPacket{
				ContextID: "assistant", Script: "stale stop", Interim: true,
			}, true)
			assert.Nil(t, decision)
			if replacement != "cancellation" {
				assert.Empty(t, admitted.ContextID)
			}
			assert.Equal(t, currentContext, message.ContextID())
			decision, _ = message.OnUserSpeech(internal_type.SpeechToTextPacket{
				ContextID: currentContext, Script: "ungated", Interim: true,
			}, true)
			assert.Nil(t, decision)
		})
	}
}

func TestMessageInterruption_OnPlaybackPausedSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 1)
		resumed := make(chan string, 1)
		var l *messageLifecycle
		l = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
			WithInterruptionExpiry(func(packet internal_type.InterruptionDecisionExpiredPacket) {
				resumed <- l.OnInterruptionExpired(packet)
				expired <- packet
			})).(*messageLifecycle)
		defer l.CancelInterruption()
		require.NoError(t, l.OnGenerationStarted("assistant"))
		_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		require.NotNil(t, pause)
		assert.Nil(t, l.OnPlaybackPaused(*pause, nil))
		time.Sleep(InterruptionDecisionWindow - time.Millisecond)
		synctest.Wait()
		assert.Empty(t, expired)
		time.Sleep(time.Millisecond)
		synctest.Wait()
		require.Len(t, expired, 1)
		assert.Equal(t, *pause, <-expired)
		assert.Equal(t, "assistant", <-resumed)
		assert.Equal(t, "assistant", l.ContextID())
		assert.Equal(t, MessageStateAssistantGenerating, l.State())
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		assert.Empty(t, expired)
	})
}

func TestMessageInterruption_OnPlaybackPausedFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 1)
		l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
			WithInterruptionExpiry(func(packet internal_type.InterruptionDecisionExpiredPacket) { expired <- packet })).(*messageLifecycle)
		defer l.CancelInterruption()
		require.NoError(t, l.OnGenerationStarted("assistant"))
		_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		require.NotNil(t, pause)
		assert.Nil(t, l.OnPlaybackPaused(*pause, nil))
		pauseError := errors.New("pause failed")
		decision := l.OnPlaybackPaused(*pause, pauseError)
		require.NotNil(t, decision)
		assert.True(t, decision.InterruptionDecision)
		assert.Equal(t, pause.Sequence, decision.InterruptionSequence)
		assert.Equal(t, "assistant", decision.PreviousContextID)
		assert.Equal(t, string(MessageStateAssistantGenerating), decision.PreviousState)
		assert.Equal(t, "interrupted", decision.Reason)
		assert.Equal(t, string(internal_type.InterruptionSourceVad), decision.Source)
		assert.False(t, decision.Time.IsZero())
		assert.Equal(t, "assistant", l.ContextID(), "pause failure decides the turn without committing it")
		assert.Nil(t, l.OnPlaybackPaused(*pause, pauseError))
		assert.Nil(t, l.OnPlaybackPaused(*pause, nil))
		time.Sleep(InterruptionDecisionWindow)
		synctest.Wait()
		assert.Empty(t, expired, "failure must cancel the armed pause deadline")
		assert.Empty(t, l.OnInterruptionExpired(*pause))
	})
}

func TestMessageInterruption_OnPlaybackPausedStale(t *testing.T) {
	for _, scenario := range []string{"context", "sequence", "cancelled", "confirmed"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				expired := make(chan internal_type.InterruptionDecisionExpiredPacket, 1)
				l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
					WithInterruptionExpiry(func(packet internal_type.InterruptionDecisionExpiredPacket) { expired <- packet })).(*messageLifecycle)
				defer l.CancelInterruption()
				require.NoError(t, l.OnGenerationStarted("assistant"))
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				switch scenario {
				case "context":
					pause.ContextID = "stale"
				case "sequence":
					pause.Sequence++
				case "cancelled":
					l.CancelInterruption()
				case "confirmed":
					decision, _ := l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait"}, true)
					require.NotNil(t, decision)
				}
				assert.Nil(t, l.OnPlaybackPaused(*pause, nil))
				assert.Nil(t, l.OnPlaybackPaused(*pause, errors.New("late pause failure")))
				time.Sleep(InterruptionDecisionWindow)
				synctest.Wait()
				assert.Empty(t, expired)
				assert.Equal(t, "assistant", l.ContextID())
				assert.Equal(t, MessageStateAssistantGenerating, l.State())
			})
		})
	}
}

func TestMessageInterruption_OnTurnChangeOrdersUpdatesBeforeReentrantInput(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		flushError error
	}{
		{name: "success"},
		{name: "failed flush", flushError: errors.New("flush failed")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			start := internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}
			interim := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}
			end := start
			end.Event = internal_type.InterruptionEventEnd
			final := internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait please"}
			eos := internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: final.Script, Speechs: []internal_type.SpeechToTextPacket{final}}
			input := internal_type.UserInputPacket{ContextID: "assistant", Text: final.Script}
			var events []any
			var committed internal_type.TurnChangePacket
			turnReturned := false
			var decision *internal_type.TurnChangePacket
			var l *messageLifecycle
			l = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
				WithSend(func(control proto.Message) error {
					events = append(events, control)
					assert.Equal(t, "assistant", l.ContextID(), "flush precedes context rotation")
					require.NoError(t, l.OnTurnChange(t.Context(), *decision), "a reentrant duplicate must be rejected")
					_, _, repeatedPause := l.observeVAD(end)
					assert.Nil(t, repeatedPause)
					next, admitted := l.OnUserSpeech(final, true)
					assert.Nil(t, next)
					assert.Empty(t, admitted.ContextID, "speech during flush stays held")
					return scenario.flushError
				}),
				WithDispatch(func(_ context.Context, packet internal_type.Packet) {
					events = append(events, packet)
					assert.NotEqual(t, "assistant", l.ContextID(), "provider updates follow commit")
					switch packet := packet.(type) {
					case internal_type.EndOfSpeechInterruptionPacket:
						assert.False(t, l.unclearInputWatchdog.Stop(), "unclear input must stop before provider updates")
					case internal_type.TurnChangePacket:
						committed = packet
						assert.True(t, l.HoldInput(eos))
						assert.True(t, l.HoldInput(input))
						assert.Equal(t, MessageStateUserListening, l.State())
						turnReturned = true
					case internal_type.InterruptionDetectedPacket:
						assert.True(t, turnReturned, "held packets replay after the turn callback")
					case internal_type.SpeechToTextPacket:
						assert.True(t, turnReturned, "held packets replay after the turn callback")
						contextID, err := l.OnTranscriptReceived(packet)
						assert.NoError(t, err)
						assert.Equal(t, packet.ContextID, contextID)
					case internal_type.EndOfSpeechPacket:
						assert.True(t, turnReturned, "held packets replay after the turn callback")
						assert.NoError(t, l.OnUserSpeechCompleted(packet))
					case internal_type.UserInputPacket:
						assert.True(t, turnReturned, "held packets replay after the turn callback")
						input, packets := l.OnUserInput(packet)
						assert.Equal(t, packet, input)
						assert.NotEmpty(t, packets)
					}
				})).(*messageLifecycle)
			defer l.CancelInterruption()
			require.NoError(t, l.Initialize(t.Context()))
			require.True(t, l.unclearInputWatchdog.Start("assistant", time.Minute))
			require.NoError(t, l.OnGenerationStarted("assistant"))
			require.NoError(t, l.OnSpeechStarted("assistant"))
			_, _, pause := l.observeVAD(start)
			require.NotNil(t, pause)
			decision, _ = l.OnUserSpeech(interim, true)
			require.NotNil(t, decision)
			err := l.OnTurnChange(t.Context(), *decision)
			assert.ErrorIs(t, err, scenario.flushError)
			require.NotEmpty(t, committed.ContextID)
			assert.Equal(t, l.ContextID(), committed.ContextID)
			assert.Equal(t, decision.Time, committed.Time)
			start.ContextID = committed.ContextID
			interim.ContextID = committed.ContextID
			end.ContextID = committed.ContextID
			final.ContextID = committed.ContextID
			input.ContextID = committed.ContextID
			assert.Equal(t, []any{
				&protos.ConversationPlaybackFlush{Id: "assistant"},
				internal_type.EndOfSpeechInterruptionPacket{ContextID: "assistant", Source: internal_type.InterruptionSourceVad},
				internal_type.TextToSpeechInterruptPacket{ContextID: "assistant"},
				internal_type.LLMInterruptPacket{ContextID: "assistant"},
				internal_type.StopIdleTimeoutPacket{ContextID: "assistant"},
				committed, start, interim, end, final,
				internal_type.EndOfSpeechPacket{ContextID: committed.ContextID, Speech: final.Script,
					Speechs: []internal_type.SpeechToTextPacket{{ContextID: committed.ContextID, Script: final.Script}}},
				input,
			}, events)
			assert.Equal(t, "assistant", eos.Speechs[0].ContextID)
			require.NoError(t, l.OnTurnChange(t.Context(), *decision), "a completed duplicate must be rejected")
			assert.Equal(t, committed.ContextID, l.ContextID())
		})
	}
}

func TestMessageInterruption_OnTurnChangeOrdinary(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		packet internal_type.TurnChangePacket
	}{
		{name: "defaults"},
		{name: "explicit context", packet: internal_type.TurnChangePacket{ContextID: "provided"}},
		{name: "explicit time", packet: internal_type.TurnChangePacket{Time: time.Unix(123, 0)}},
		{name: "explicit context and time", packet: internal_type.TurnChangePacket{ContextID: "provided", Time: time.Unix(123, 0)}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var packets []internal_type.Packet
			var l *messageLifecycle
			l = NewMessageLifecycle(WithContextID("current"), WithMode(type_enums.AudioMode),
				WithDispatch(func(_ context.Context, packet internal_type.Packet) {
					assert.Equal(t, "current", l.ContextID())
					packets = append(packets, packet)
				})).(*messageLifecycle)
			assert.False(t, l.InterruptionEnabled())
			packet := scenario.packet
			packet.PreviousContextID, packet.Reason = "previous", "user_input"
			before := time.Now()
			require.NoError(t, l.OnTurnChange(t.Context(), packet))
			require.Len(t, packets, 1)
			turn := packets[0].(internal_type.TurnChangePacket)
			if packet.ContextID == "" {
				packet.ContextID = "current"
			}
			if packet.Time.IsZero() {
				assert.False(t, turn.Time.Before(before))
				assert.False(t, turn.Time.After(time.Now()))
				packet.Time = turn.Time
			}
			assert.Equal(t, packet, turn)
		})
	}
}

func TestMessageInterruption_HoldsLiveInputUntilReplayDrains(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		replayStarted := make(chan struct{})
		releaseReplay := make(chan struct{})
		defer close(releaseReplay)
		completed := make(chan error, 1)
		observed := make(chan internal_type.Packet, 8)
		var message MessageLifecycle
		message = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
			WithSend(func(proto.Message) error { return nil }),
			WithDispatch(func(_ context.Context, packet internal_type.Packet) {
				switch packet := packet.(type) {
				case internal_type.InterruptionDetectedPacket:
					if packet.Event == internal_type.InterruptionEventStart {
						close(replayStarted)
						<-releaseReplay
					}
					observed <- packet
				case internal_type.SpeechToTextPacket:
					contextID, err := message.OnTranscriptReceived(packet)
					assert.NoError(t, err)
					assert.Equal(t, packet.ContextID, contextID)
					observed <- packet
				case internal_type.EndOfSpeechPacket:
					assert.NoError(t, message.OnUserSpeechCompleted(packet))
					observed <- packet
				case internal_type.UserInputPacket:
					admitted, _ := message.OnUserInput(packet)
					assert.Equal(t, packet.ContextID, admitted.ContextID)
					assert.Equal(t, packet, admitted)
					observed <- packet
				}
			}))
		defer message.CancelInterruption()
		require.NoError(t, message.Initialize(t.Context()))
		require.NoError(t, message.OnGenerationStarted("assistant"))
		message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
			ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}, "")
		turn, _ := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}, true)
		require.NotNil(t, turn)
		go func() { completed <- message.OnTurnChange(t.Context(), *turn) }()
		<-replayStarted
		current := message.ContextID()
		next, admitted := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: current, Script: "wait please"}, true)
		assert.Nil(t, next)
		assert.Empty(t, admitted.ContextID, "the final must stay behind the held interim")
		decision := message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
			ContextID: current, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
		}, "")
		assert.Nil(t, decision.EndOfSpeech, "live VAD-end must stay behind replayed VAD-start")
		assert.True(t, message.HoldInput(internal_type.EndOfSpeechPacket{ContextID: current, Speech: "wait please"}))
		assert.True(t, message.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "wait please"}))
		synctest.Wait()
		assert.Empty(t, observed)
		releaseReplay <- struct{}{}
		require.NoError(t, <-completed)
		require.Len(t, observed, 6)
		assert.Equal(t, internal_type.PacketNameInterruptionDetected, (<-observed).PacketName())
		assert.Equal(t, "wait", (<-observed).(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, "wait please", (<-observed).(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, internal_type.InterruptionEventEnd, (<-observed).(internal_type.InterruptionDetectedPacket).Event)
		assert.Equal(t, internal_type.PacketNameEndOfSpeech, (<-observed).PacketName())
		assert.Equal(t, internal_type.PacketNameUserInput, (<-observed).PacketName())
		assert.False(t, message.HoldInput(internal_type.UserInputPacket{ContextID: current, Text: "next"}))
		assert.False(t, message.(*messageLifecycle).unclearInputWatchdog.Stop(), "replayed final must stop the unclear-input timer")
	})
}

func TestMessageInterruption_CancelDuringReplayDropsRemainingInput(t *testing.T) {
	for _, scenario := range []string{"cancel", "replace turn"} {
		t.Run(scenario, func(t *testing.T) {
			var message MessageLifecycle
			var observed []internal_type.Packet
			message = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
				WithSend(func(proto.Message) error { return nil }),
				WithDispatch(func(_ context.Context, packet internal_type.Packet) {
					switch packet.(type) {
					case internal_type.InterruptionDetectedPacket:
						message.CancelInterruption()
						if scenario == "replace turn" {
							assert.NoError(t, message.OnUserSpeechCompleted(internal_type.EndOfSpeechPacket{ContextID: message.ContextID(), Speech: "replacement"}))
							assert.NoError(t, message.OnGenerationStarted(message.ContextID()))
							_, err := message.OnUserTurnStarted(message.ContextID(), "test", "stt", "replacement")
							assert.NoError(t, err)
						}
					case internal_type.SpeechToTextPacket, internal_type.EndOfSpeechPacket, internal_type.UserInputPacket:
						observed = append(observed, packet)
					}
				}))
			defer message.CancelInterruption()
			require.NoError(t, message.OnGenerationStarted("assistant"))
			message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}, "")
			turn, _ := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}, true)
			require.NotNil(t, turn)
			require.True(t, message.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: "wait"}))
			require.True(t, message.HoldInput(internal_type.UserInputPacket{ContextID: "assistant", Text: "wait"}))
			require.NoError(t, message.OnTurnChange(t.Context(), *turn))
			assert.Empty(t, observed, "cancellation must discard every remaining admitted input type")
			assert.False(t, message.HoldInput(internal_type.UserInputPacket{ContextID: message.ContextID(), Text: "new input"}))
		})
	}
}

func TestMessageInterruption_HoldsEOSAndUserInputWithoutTranscript(t *testing.T) {
	for _, inputContext := range []string{"assistant", ""} {
		t.Run("context="+inputContext, func(t *testing.T) {
			l := NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true)).(*messageLifecycle)
			t.Cleanup(func() { l.CancelInterruption() })
			require.NoError(t, l.OnGenerationStarted("assistant"))
			start := internal_type.InterruptionDetectedPacket{
				ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			}
			_, _, pause := l.observeVAD(start)
			require.NotNil(t, pause)
			require.True(t, l.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: "wait please"}))
			require.True(t, l.HoldInput(internal_type.UserInputPacket{ContextID: inputContext, Text: "wait please"}))
			require.True(t, l.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "assistant", Speech: "um, hmm"}))
			require.True(t, l.HoldInput(internal_type.EndOfSpeechPacket{ContextID: "stale", Speech: "discard"}))
			require.True(t, l.HoldInput(internal_type.UserInputPacket{ContextID: "stale", Text: "discard"}))
			decision := l.OnPlaybackPaused(*pause, errors.New("pause failed"))
			require.NotNil(t, decision)
			require.True(t, l.beginInterruptedTurn(*decision))
			committed, ok := l.commitInterruptedTurn(*decision)
			require.True(t, ok)
			start.ContextID = committed.ContextID
			assert.Equal(t, []internal_type.Packet{
				start,
				internal_type.EndOfSpeechPacket{ContextID: committed.ContextID, Speech: "wait please"},
				internal_type.UserInputPacket{ContextID: committed.ContextID, Text: "wait please"},
			}, l.finishInterruptedTurn(committed))
			assert.Empty(t, l.finishInterruptedTurn(committed))
		})
	}
}

func TestMessageInterruption_UnclearExpiryWaitsForReplay(t *testing.T) {
	for _, scenario := range []string{"interim only", "final received", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started, release := make(chan struct{}), make(chan struct{})
				defer close(release)
				finished := make(chan error, 1)
				prompts := make(chan internal_type.InjectMessagePacket, 2)
				var message MessageLifecycle
				timeout, prompt := 1.0, "Please repeat"
				message = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(true),
					WithBehavior(func() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
						return &internal_assistant_entity.AssistantDeploymentBehavior{UnclearInputTimeout: &timeout, UnclearInputMessage: &prompt}, nil
					}),
					WithSend(func(proto.Message) error { return nil }),
					WithOnPacket(func(packets ...internal_type.Packet) error {
						for _, packet := range packets {
							if expired, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
								_, _, err := message.OnPrompt(expired)
								assert.ErrorIs(t, err, ErrInvalidTransition, "expiry must wait for replay")
							}
						}
						return nil
					}),
					WithDispatch(func(_ context.Context, packet internal_type.Packet) {
						switch packet := packet.(type) {
						case internal_type.SpeechToTextPacket:
							contextID, err := message.OnTranscriptReceived(packet)
							assert.NoError(t, err)
							assert.Equal(t, packet.ContextID, contextID)
							if packet.Interim {
								close(started)
								<-release
							}
						case internal_type.UnclearInputExpiredPacket:
							_, prompt, err := message.OnPrompt(packet)
							if err == nil {
								prompts <- prompt
							}
						}
					}))
				defer message.CancelInterruption()
				require.NoError(t, message.Initialize(t.Context()))
				require.NoError(t, message.OnGenerationStarted("assistant"))
				message.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}, "")
				turn, _ := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait", Interim: true}, true)
				require.NotNil(t, turn)
				go func() { finished <- message.OnTurnChange(t.Context(), *turn) }()
				<-started
				current := message.ContextID()
				time.Sleep(2 * time.Second)
				synctest.Wait()
				assert.Empty(t, prompts)
				switch scenario {
				case "final received":
					_, accepted := message.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: current, Script: "wait please"}, true)
					assert.Empty(t, accepted.ContextID)
				case "cancelled":
					message.CancelInterruption()
				}
				release <- struct{}{}
				require.NoError(t, <-finished)
				if scenario == "interim only" {
					require.Len(t, prompts, 1)
					assert.Equal(t, "Please repeat", (<-prompts).Text)
					assert.NotEqual(t, current, message.ContextID())
				} else {
					assert.Empty(t, prompts)
					assert.Equal(t, current, message.ContextID())
				}
			})
		})
	}
}

func TestMessageInterruption_OnTurnChangeStale(t *testing.T) {
	for _, scenario := range []string{"context", "sequence", "missing sequence", "cancelled", "disabled", "replaced during flush"} {
		t.Run(scenario, func(t *testing.T) {
			currentContext := "assistant"
			flushError := errors.New("flush failed after turn replacement")
			flushCount := 0
			var l *messageLifecycle
			l = NewMessageLifecycle(WithContextID("assistant"), WithMode(type_enums.AudioMode), WithInterruption(scenario != "disabled"),
				WithSend(func(proto.Message) error {
					flushCount++
					require.Equal(t, "replaced during flush", scenario)
					_, err := l.OnUserTurnStarted("assistant", "test", "text", "new input")
					require.NoError(t, err)
					currentContext = l.ContextID()
					return flushError
				}),
				WithDispatch(func(context.Context, internal_type.Packet) {
					t.Error("stale turn must not update providers or replay held input")
				})).(*messageLifecycle)
			defer l.CancelInterruption()
			require.NoError(t, l.OnGenerationStarted("assistant"))
			decision := &internal_type.TurnChangePacket{
				PreviousContextID: "assistant", InterruptionDecision: true, InterruptionSequence: 1,
			}
			if scenario != "disabled" {
				_, _, pause := l.observeVAD(internal_type.InterruptionDetectedPacket{
					ContextID: "assistant", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				require.NotNil(t, pause)
				decision, _ = l.OnUserSpeech(internal_type.SpeechToTextPacket{ContextID: "assistant", Script: "wait"}, true)
				require.NotNil(t, decision)
			}
			switch scenario {
			case "context":
				decision.PreviousContextID = "stale"
			case "sequence":
				decision.InterruptionSequence++
			case "missing sequence":
				decision.InterruptionSequence = 0
			case "cancelled":
				l.CancelInterruption()
			}
			err := l.OnTurnChange(t.Context(), *decision)
			if scenario == "replaced during flush" {
				assert.ErrorIs(t, err, flushError)
				assert.Equal(t, 1, flushCount)
				assert.NotEqual(t, "assistant", currentContext)
			} else {
				assert.NoError(t, err)
				assert.Zero(t, flushCount)
			}
			assert.Equal(t, currentContext, l.ContextID())
		})
	}
}
