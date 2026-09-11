package adapter_internal

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestInterruptionDeadlineChoosesExactlyOnce(t *testing.T) {
	for _, scenario := range []struct {
		name string
		end  bool
		text string
	}{
		{name: "active"},
		{name: "active fillers", text: "Um, HMM..."},
		{name: "ended empty", end: true},
		{name: "ended fillers", end: true, text: "Um, HMM..."},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please")
				t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
				requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
				requestor.messageLifecycle.ConfigureInterruption(true)
				streamer := requestor.streamer.(*streamTestStreamer)
				handler := requestorDispatchHandler{r: requestor}
				originalContext := requestor.GetID()
				start := internal_type.InterruptionDetectedPacket{ContextID: originalContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}
				handler.HandleInterruptionDetected(context.Background(), start)
				synctest.Wait()
				assert.Equal(t, originalContext, requestor.GetID())
				streamer.mu.Lock()
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: originalContext}}, streamer.sent)
				streamer.mu.Unlock()
				time.Sleep(200 * time.Millisecond)
				handler.HandleInterruptionDetected(context.Background(), start)
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: originalContext, Script: scenario.text, Interim: true})
				if scenario.end {
					handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: originalContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
				}
				time.Sleep(299 * time.Millisecond)
				synctest.Wait()
				streamer.mu.Lock()
				assert.Len(t, streamer.sent, 1)
				streamer.mu.Unlock()
				time.Sleep(time.Millisecond)
				synctest.Wait()
				assert.Equal(t, originalContext, requestor.GetID())
				streamer.mu.Lock()
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: originalContext}, &protos.ConversationPlaybackContinue{Id: originalContext}}, streamer.sent)
				streamer.mu.Unlock()
				contextAfterDecision := requestor.GetID()
				time.Sleep(time.Second)
				synctest.Wait()
				for _, packet := range drainEgressPackets(requestor) {
					_, expired := packet.(internal_type.UnclearInputExpiredPacket)
					assert.False(t, expired)
				}
				assert.Equal(t, contextAfterDecision, requestor.GetID())
				outputControlCount := 0
				streamer.mu.Lock()
				for _, sentPacket := range streamer.sent {
					switch sentPacket.(type) {
					case *protos.ConversationPlaybackPause, *protos.ConversationPlaybackContinue, *protos.ConversationPlaybackFlush:
						outputControlCount++
					}
				}
				streamer.mu.Unlock()
				assert.Equal(t, 2, outputControlCount)
			})
		})
	}
}

func TestInterruptionMeaningfulInterimCommitsAndReplaysInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please")
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		requestor.messageLifecycle.ConfigureInterruption(true)
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "um, wait", Interim: true})
		synctest.Wait()
		current := requestor.GetID()
		require.NotEqual(t, previous, current)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait please", Interim: true})
		synctest.Wait()
		assert.Equal(t, current, requestor.GetID())
		streamer.mu.Lock()
		require.GreaterOrEqual(t, len(streamer.sent), 2)
		assert.Equal(t, &protos.ConversationPlaybackPause{Id: previous}, streamer.sent[0])
		assert.Equal(t, &protos.ConversationPlaybackFlush{Id: previous}, streamer.sent[1])
		streamer.mu.Unlock()
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 4)
		assert.IsType(t, internal_type.EndOfSpeechInterruptionPacket{}, packets[0])
		assert.Equal(t, previous, packets[0].ContextId())
		assert.IsType(t, internal_type.InterruptionDetectedPacket{}, packets[1])
		assert.Equal(t, "um, wait", packets[2].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, "wait please", packets[3].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, current, packets[3].ContextId())
		time.Sleep(time.Second)
		synctest.Wait()
		var expired internal_type.UnclearInputExpiredPacket
		for _, packet := range drainEgressPackets(requestor) {
			if value, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
				expired = value
			}
		}
		assert.Equal(t, current, expired.ContextID)
		handler.HandleTurnChange(context.Background(), internal_type.TurnChangePacket{InterruptionDecision: true, PreviousContextID: previous})
		synctest.Wait()
		assert.Equal(t, current, requestor.GetID())
		outputControlCount := 0
		streamer.mu.Lock()
		for _, sentPacket := range streamer.sent {
			switch sentPacket.(type) {
			case *protos.ConversationPlaybackPause, *protos.ConversationPlaybackContinue, *protos.ConversationPlaybackFlush:
				outputControlCount++
			}
		}
		streamer.mu.Unlock()
		assert.Equal(t, 2, outputControlCount)
	})
}

func TestInterruptionLateConfirmationAfterContinuePreservesUnclearInput(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		endBeforeText bool
		finalReceived bool
		finalOnly     bool
	}{
		{name: "active speech interim"},
		{name: "ended speech interim", endBeforeText: true},
		{name: "ended speech final", endBeforeText: true, finalReceived: true},
		{name: "ended speech final only", endBeforeText: true, finalReceived: true, finalOnly: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please")
				t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
				requestor.messageLifecycle.ConfigureInterruption(true)
				eos := &recordingEOSExecutor{}
				requestor.endOfSpeechExecutor = eos
				streamer := requestor.streamer.(*streamTestStreamer)
				handler := requestorDispatchHandler{r: requestor}
				previousContext := requestor.GetID()
				start := internal_type.InterruptionDetectedPacket{
					ContextID: previousContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				}
				handler.HandleInterruptionDetected(context.Background(), start)
				time.Sleep(adapter_lifecycle.InterruptionDecisionWindow)
				synctest.Wait()
				handler.HandleInterruptionDetected(context.Background(), start)
				streamer.mu.Lock()
				assert.Equal(t, []proto.Message{
					&protos.ConversationPlaybackPause{Id: previousContext},
					&protos.ConversationPlaybackContinue{Id: previousContext},
				}, streamer.sent)
				streamer.mu.Unlock()
				if scenario.endBeforeText {
					handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
						ContextID: previousContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd,
					})
					assert.Contains(t, drainControlPackets(requestor), internal_type.SpeechToTextEndPacket{ContextID: previousContext})
				}
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
					ContextID: previousContext, Script: "wait please", Interim: !scenario.finalOnly,
				})
				synctest.Wait()
				currentContext := requestor.GetID()
				require.NotEqual(t, previousContext, currentContext)
				streamer.mu.Lock()
				require.GreaterOrEqual(t, len(streamer.sent), 3)
				assert.Equal(t, &protos.ConversationPlaybackFlush{Id: previousContext}, streamer.sent[2])
				streamer.mu.Unlock()
				packets := eos.snapshotExecuted()
				require.NotEmpty(t, packets)
				assert.Equal(t, internal_type.SpeechToTextPacket{
					ContextID: currentContext, Script: "wait please", Interim: !scenario.finalOnly,
				}, packets[len(packets)-1])
				if scenario.finalReceived && !scenario.finalOnly {
					handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
						ContextID: previousContext, Script: "Wait please.",
					})
				}
				time.Sleep(time.Second)
				synctest.Wait()
				var expiredContext string
				for _, packet := range drainEgressPackets(requestor) {
					if expired, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expiredContext = expired.ContextID
					}
				}
				if scenario.finalReceived {
					assert.Empty(t, expiredContext)
				} else {
					assert.Equal(t, currentContext, expiredContext)
				}
			})
		})
	}
}

func TestInterruptionPauseFailureCommitsWithoutWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.streamer = &failingOutputControlStreamer{err: errors.New("output unavailable")}
		requestor.messageLifecycle.ConfigureInterruption(true)
		previous := requestor.GetID()
		requestorDispatchHandler{r: requestor}.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		synctest.Wait()
		assert.NotEqual(t, previous, requestor.GetID())
		var failures int
		for _, packet := range drainBackgroundPackets(requestor) {
			if record, ok := packet.(internal_type.ObservabilityLogRecordPacket); ok && record.Record.Message == "Interruption output control failed" {
				failures++
			}
		}
		assert.Equal(t, 2, failures)
	})
}

func TestInterruptionPauseIOCompletesBeforeFlushCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle.ConfigureInterruption(true)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		pauseStarted, releasePause, pauseDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
		streamer := &terminalReceiptTestStreamer{onSend: func(packet proto.Message) error {
			if _, ok := packet.(*protos.ConversationPlaybackPause); ok {
				// Speech can confirm synchronously while transport application of Pause is delayed.
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
					ContextID: previous, Script: "wait", Interim: true,
				})
				close(pauseStarted)
				<-releasePause
			}
			return nil
		}}
		requestor.streamer = streamer
		go func() {
			defer close(pauseDone)
			handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
				ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
			})
		}()
		<-pauseStarted
		assert.Equal(t, previous, requestor.GetID(), "flush must not commit while Pause is still in flight")
		streamer.mu.Lock()
		assert.Empty(t, streamer.sent, "Flush must not reach playback before the delayed Pause")
		streamer.mu.Unlock()
		close(releasePause)
		<-pauseDone
		synctest.Wait()
		assert.NotEqual(t, previous, requestor.GetID())
		var controls []proto.Message
		streamer.mu.Lock()
		for _, packet := range streamer.sent {
			switch packet.(type) {
			case *protos.ConversationPlaybackPause, *protos.ConversationPlaybackFlush:
				controls = append(controls, packet)
			}
		}
		streamer.mu.Unlock()
		assert.Equal(t, []proto.Message{
			&protos.ConversationPlaybackPause{Id: previous},
			&protos.ConversationPlaybackFlush{Id: previous},
		}, controls)
	})
}

func TestInterruptionRejectsPauseSentAfterTurnCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle.ConfigureInterruption(true)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		_, _, pause := requestor.messageLifecycle.ObserveVAD(internal_type.InterruptionDetectedPacket{
			ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		require.NotNil(t, pause)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
			ContextID: previous, Script: "wait", Interim: true,
		})
		synctest.Wait()
		current := requestor.GetID()
		require.NotEqual(t, previous, current)
		require.ErrorIs(t, requestor.sendOutputControl(&protos.ConversationPlaybackPause{Id: pause.ContextID}), adapter_lifecycle.ErrStaleContext)
		assert.Equal(t, current, requestor.GetID())
		streamer := requestor.streamer.(*streamTestStreamer)
		streamer.mu.Lock()
		defer streamer.mu.Unlock()
		for _, packet := range streamer.sent {
			_, isPause := packet.(*protos.ConversationPlaybackPause)
			assert.False(t, isPause, "a Pause arriving after commit must not reach playback")
		}
	})
}

func TestInterruptionRejectsPauseBetweenFlushAndCommit(t *testing.T) {
	requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
	requestor.messageLifecycle.ConfigureInterruption(true)
	t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
	previous := requestor.GetID()
	_, _, pause := requestor.messageLifecycle.ObserveVAD(internal_type.InterruptionDetectedPacket{
		ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
	})
	require.NotNil(t, pause)
	decision, _, _ := requestor.messageLifecycle.ObserveSpeech(internal_type.SpeechToTextPacket{
		ContextID: previous, Script: "wait", Interim: true,
	}, true)
	require.NotNil(t, decision)
	require.True(t, requestor.messageLifecycle.BeginInterruptedTurn(*decision))
	require.NoError(t, requestor.sendOutputControl(&protos.ConversationPlaybackFlush{Id: previous}))
	require.ErrorIs(t, requestor.sendOutputControl(&protos.ConversationPlaybackPause{Id: pause.ContextID}), adapter_lifecycle.ErrStaleContext)
	streamer := requestor.streamer.(*streamTestStreamer)
	assert.Equal(t, []proto.Message{&protos.ConversationPlaybackFlush{Id: previous}}, streamer.sent,
		"a reserved candidate must not allow Pause after Flush while context rotation is pending")
	committed, ok := requestor.messageLifecycle.CommitInterruptedTurn(*decision)
	require.True(t, ok)
	assert.NotEqual(t, previous, committed.ContextID)
	assert.Len(t, requestor.messageLifecycle.FinishInterruptedTurn(committed), 2)
}

func TestInterruptionCancellationContinuesOnlyPendingOutput(t *testing.T) {
	for _, commit := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
			requestor.messageLifecycle.ConfigureInterruption(true)
			streamer := requestor.streamer.(*streamTestStreamer)
			handler := requestorDispatchHandler{r: requestor}
			previousContextID := requestor.GetID()
			handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
			if commit {
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
			}
			synctest.Wait()
			handler.HandleFinalizeBehavior(context.Background(), internal_type.FinalizeBehaviorPacket{})
			streamer.mu.Lock()
			if commit {
				require.GreaterOrEqual(t, len(streamer.sent), 2)
				assert.Equal(t, &protos.ConversationPlaybackPause{Id: previousContextID}, streamer.sent[0])
				assert.Equal(t, &protos.ConversationPlaybackFlush{Id: previousContextID}, streamer.sent[1])
			} else {
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: previousContextID}, &protos.ConversationPlaybackContinue{Id: previousContextID}}, streamer.sent)
			}
			streamer.mu.Unlock()
		})
	}
}

func TestInterruptionReleaseRemainsDisabled(t *testing.T) {
	assert.False(t, adapter_lifecycle.NewMessageLifecycle().InterruptionEnabled())
}

type blockedInterruptionTransformer struct {
	noopSpeechToTextTransformer
	blockOn internal_type.PacketName
	started chan struct{}
	release chan struct{}
}

func (transformer blockedInterruptionTransformer) Transform(ctx context.Context, packet internal_type.Packet) error {
	if packet.PacketName() == transformer.blockOn {
		close(transformer.started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-transformer.release:
		}
	}
	return nil
}

func TestInterruptionHoldsTranscriptsUntilCommitCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat")
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		started, release := make(chan struct{}), make(chan struct{})
		requestor.textToSpeechTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameTextToSpeechInterrupt, started: started, release: release,
		}
		requestor.messageLifecycle.ConfigureInterruption(true)
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait", Interim: true})
		<-started
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait please", Interim: true})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "Wait please"})
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
		assert.Len(t, eos.snapshotExecuted(), 1)
		streamer.mu.Lock()
		require.GreaterOrEqual(t, len(streamer.sent), 2)
		assert.IsType(t, &protos.ConversationPlaybackPause{}, streamer.sent[0])
		assert.IsType(t, &protos.ConversationPlaybackFlush{}, streamer.sent[1])
		streamer.mu.Unlock()
		close(release)
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired)
		}
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 5)
		assert.Equal(t, "wait", packets[2].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, "wait please", packets[3].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, "Wait please", packets[4].(internal_type.SpeechToTextPacket).Script)
	})
}

func TestInterruptionDeadlineDoesNotWaitForProviderWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		started, release := make(chan struct{}), make(chan struct{})
		requestor.speechToTextTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameSpeechToTextStart, started: started, release: release,
		}
		requestor.messageLifecycle.ConfigureInterruption(true)
		dispatcherContext, cancelDispatcher := context.WithCancel(context.Background())
		defer cancelDispatcher()
		go requestor.runCriticalDispatcher(dispatcherContext)
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		<-started
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		time.Sleep(adapter_lifecycle.InterruptionDecisionWindow)
		synctest.Wait()
		assert.Equal(t, previous, requestor.GetID())
		streamer.mu.Lock()
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: previous}, &protos.ConversationPlaybackContinue{Id: previous}}, streamer.sent)
		streamer.mu.Unlock()
		close(release)
	})
}

func TestInterruptionIgnoresStaleAndUngatedTranscripts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle.ConfigureInterruption(true)
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "late final"})
		assert.Equal(t, previous, requestor.GetID())
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: "obsolete", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		streamer.mu.Lock()
		assert.Empty(t, streamer.sent)
		streamer.mu.Unlock()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: "obsolete", Script: "late interim", Interim: true})
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		time.Sleep(adapter_lifecycle.InterruptionDecisionWindow)
		synctest.Wait()
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "late final"})
		synctest.Wait()
		assert.Equal(t, previous, requestor.GetID())
		streamer.mu.Lock()
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: previous}, &protos.ConversationPlaybackContinue{Id: previous}}, streamer.sent)
		streamer.mu.Unlock()
	})
}

func TestInterruptionIgnoresStaleVADEndOutsidePreviousContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle = adapter_lifecycle.NewMessageLifecycleWithContext("current", type_enums.AudioMode)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		requestor.messageLifecycle.ConfigureInterruption(true)
		streamer := requestor.streamer.(*streamTestStreamer)

		requestorDispatchHandler{r: requestor}.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: "obsolete",
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventEnd,
		})
		synctest.Wait()

		assert.Empty(t, eos.snapshotExecuted())
		streamer.mu.Lock()
		assert.Empty(t, streamer.sent)
		streamer.mu.Unlock()
		assert.Equal(t, "current", requestor.GetID())
	})
}

type recordingInterruptionLifecycle struct {
	adapter_lifecycle.MessageLifecycle
	pauses []internal_type.InterruptionDecisionExpiredPacket
}

func (l *recordingInterruptionLifecycle) ObserveInterruption(p internal_type.InterruptionDetectedPacket, bargeInTrigger string) adapter_lifecycle.InterruptionDecision {
	decision := l.MessageLifecycle.ObserveInterruption(p, bargeInTrigger)
	if decision.Pause != nil {
		l.pauses = append(l.pauses, *decision.Pause)
	}
	return decision
}

func TestInterruptionRejectsStaleDecisionPackets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle.ConfigureInterruption(true)
		lifecycle := &recordingInterruptionLifecycle{MessageLifecycle: requestor.messageLifecycle}
		requestor.messageLifecycle = lifecycle
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		contextID := requestor.GetID()

		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventStart,
		})
		require.Len(t, lifecycle.pauses, 1)
		firstPause := lifecycle.pauses[0]
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventEnd,
		})
		handler.HandleInterruptionDecisionExpired(context.Background(), firstPause)
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventStart,
		})
		require.Len(t, lifecycle.pauses, 2)
		secondPause := lifecycle.pauses[1]
		require.NotEqual(t, firstPause.Sequence, secondPause.Sequence)

		handler.HandleInterruptionDecisionExpired(context.Background(), firstPause)
		handler.HandleTurnChange(context.Background(), internal_type.TurnChangePacket{
			InterruptionDecision: true,
			InterruptionSequence: firstPause.Sequence,
			PreviousContextID:    contextID,
		})

		assert.Equal(t, contextID, requestor.GetID())
		assert.Equal(t, adapter_lifecycle.MessageStateAssistantSpeaking, requestor.messageLifecycle.State())
		streamer.mu.Lock()
		assert.Equal(t, []proto.Message{
			&protos.ConversationPlaybackPause{Id: contextID},
			&protos.ConversationPlaybackContinue{Id: contextID},
			&protos.ConversationPlaybackPause{Id: contextID},
		}, streamer.sent)
		streamer.mu.Unlock()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventEnd,
		})
		handler.HandleInterruptionDecisionExpired(context.Background(), secondPause)
		assert.Equal(t, contextID, requestor.GetID())
		streamer.mu.Lock()
		assert.Equal(t, []proto.Message{
			&protos.ConversationPlaybackPause{Id: contextID},
			&protos.ConversationPlaybackContinue{Id: contextID},
			&protos.ConversationPlaybackPause{Id: contextID},
			&protos.ConversationPlaybackContinue{Id: contextID},
		}, streamer.sent)
		streamer.mu.Unlock()
		handler.HandleFinalizeBehavior(context.Background(), internal_type.FinalizeBehaviorPacket{})
	})
}

func TestInterruptionFinalizationFencesDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle.ConfigureInterruption(true)
		lifecycle := &recordingInterruptionLifecycle{MessageLifecycle: requestor.messageLifecycle}
		requestor.messageLifecycle = lifecycle
		streamer := requestor.streamer.(*streamTestStreamer)
		handler := requestorDispatchHandler{r: requestor}
		contextID := requestor.GetID()

		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: contextID,
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventStart,
		})
		require.Len(t, lifecycle.pauses, 1)
		handler.HandleFinalizeBehavior(context.Background(), internal_type.FinalizeBehaviorPacket{ContextID: contextID})
		handler.HandleInterruptionDecisionExpired(context.Background(), lifecycle.pauses[0])

		assert.Equal(t, contextID, requestor.GetID())
		streamer.mu.Lock()
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackPause{Id: contextID}, &protos.ConversationPlaybackContinue{Id: contextID}}, streamer.sent)
		streamer.mu.Unlock()
	})
}
