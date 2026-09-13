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
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please", adapter_lifecycle.WithInterruption(true))
				t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
				requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
				streamer := requestor.streamer.(*streamTestStreamer)
				handler := requestorDispatchHandler{r: requestor}
				originalContext := requestor.GetID()
				start := internal_type.InterruptionDetectedPacket{ContextID: originalContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}
				handler.HandleInterruptionDetected(context.Background(), start)
				synctest.Wait()
				assert.Equal(t, originalContext, requestor.GetID())
				streamer.mu.Lock()
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: originalContext, Kind: protos.ConversationPlaybackControl_PAUSE}}, streamer.sent)
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
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: originalContext, Kind: protos.ConversationPlaybackControl_PAUSE}, &protos.ConversationPlaybackControl{Id: originalContext, Kind: protos.ConversationPlaybackControl_CONTINUE}}, streamer.sent)
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
					if control, ok := sentPacket.(*protos.ConversationPlaybackControl); ok {
						switch control.GetKind() {
						case protos.ConversationPlaybackControl_PAUSE, protos.ConversationPlaybackControl_CONTINUE, protos.ConversationPlaybackControl_FLUSH:
							outputControlCount++
						}
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
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
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
		assert.Equal(t, &protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_PAUSE}, streamer.sent[0])
		assert.Equal(t, &protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_FLUSH}, streamer.sent[1])
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
			if control, ok := sentPacket.(*protos.ConversationPlaybackControl); ok {
				switch control.GetKind() {
				case protos.ConversationPlaybackControl_PAUSE, protos.ConversationPlaybackControl_CONTINUE, protos.ConversationPlaybackControl_FLUSH:
					outputControlCount++
				}
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
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please", adapter_lifecycle.WithInterruption(true))
				t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
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
					&protos.ConversationPlaybackControl{Id: previousContext, Kind: protos.ConversationPlaybackControl_PAUSE},
					&protos.ConversationPlaybackControl{Id: previousContext, Kind: protos.ConversationPlaybackControl_CONTINUE},
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
				assert.Equal(t, &protos.ConversationPlaybackControl{Id: previousContext, Kind: protos.ConversationPlaybackControl_FLUSH}, streamer.sent[2])
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
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
		requestor.streamer = &failingOutputControlStreamer{err: errors.New("output unavailable")}
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
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		pauseStarted, releasePause, pauseDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
		streamer := &terminalReceiptTestStreamer{onSend: func(packet proto.Message) error {
			if control, ok := packet.(*protos.ConversationPlaybackControl); ok && control.GetKind() == protos.ConversationPlaybackControl_PAUSE {
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
			if control, ok := packet.(*protos.ConversationPlaybackControl); ok {
				switch control.GetKind() {
				case protos.ConversationPlaybackControl_PAUSE, protos.ConversationPlaybackControl_FLUSH:
					controls = append(controls, packet)
				}
			}
		}
		streamer.mu.Unlock()
		assert.Equal(t, []proto.Message{
			&protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_PAUSE},
			&protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_FLUSH},
		}, controls)
	})
}

func TestInterruptionRejectsPauseSentAfterTurnCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		observation := requestor.messageLifecycle.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
			ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}, internal_options.BargeInTriggerVAD)
		pause := observation.Pause
		require.NotNil(t, pause)
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{
			ContextID: previous, Script: "wait", Interim: true,
		})
		synctest.Wait()
		current := requestor.GetID()
		require.NotEqual(t, previous, current)
		require.ErrorIs(t, requestor.sendOutputControl(&protos.ConversationPlaybackControl{Id: pause.ContextID, Kind: protos.ConversationPlaybackControl_PAUSE}), adapter_lifecycle.ErrStaleContext)
		assert.Equal(t, current, requestor.GetID())
		streamer := requestor.streamer.(*streamTestStreamer)
		streamer.mu.Lock()
		defer streamer.mu.Unlock()
		for _, packet := range streamer.sent {
			control, isControl := packet.(*protos.ConversationPlaybackControl)
			assert.False(t, isControl && control.GetKind() == protos.ConversationPlaybackControl_PAUSE, "a Pause arriving after commit must not reach playback")
		}
	})
}

func TestInterruptionRejectsPauseBetweenFlushAndCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var requestor *genericRequestor
		var packets []internal_type.Packet
		pauseResult := make(chan error, 1)
		requestor = newInterruptionTestRequestor(internal_options.BargeInTriggerVAD,
			adapter_lifecycle.WithInterruption(true),
			adapter_lifecycle.WithSend(func(control proto.Message) error {
				require.NoError(t, requestor.streamer.Send(control))
				assert.Equal(t, "ctx-active", requestor.GetID())
				pauseStarted := make(chan struct{})
				go func() {
					close(pauseStarted)
					pauseResult <- requestor.sendOutputControl(&protos.ConversationPlaybackControl{Id: "ctx-active", Kind: protos.ConversationPlaybackControl_PAUSE})
				}()
				<-pauseStarted
				require.Empty(t, pauseResult, "pause must wait for the in-flight flush")
				return nil
			}),
			adapter_lifecycle.WithDispatch(func(_ context.Context, packet internal_type.Packet) {
				packets = append(packets, packet)
			}),
		)
		t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
		previous := requestor.GetID()
		observation := requestor.messageLifecycle.OnInterruptionDetected(internal_type.InterruptionDetectedPacket{
			ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}, internal_options.BargeInTriggerVAD)
		pause := observation.Pause
		require.NotNil(t, pause)
		decision, _ := requestor.messageLifecycle.OnUserSpeech(internal_type.SpeechToTextPacket{
			ContextID: previous, Script: "wait", Interim: true,
		}, true)
		require.NotNil(t, decision)
		require.NoError(t, requestor.messageLifecycle.OnTurnChange(context.Background(), *decision))
		require.ErrorIs(t, <-pauseResult, adapter_lifecycle.ErrStaleContext)
		streamer := requestor.streamer.(*streamTestStreamer)
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_FLUSH}}, streamer.sent,
			"a reserved candidate must not allow Pause after Flush while context rotation is pending")
		require.Len(t, packets, 7)
		committed, ok := packets[4].(internal_type.TurnChangePacket)
		require.True(t, ok)
		assert.NotEqual(t, previous, committed.ContextID)
		assert.Equal(t, requestor.GetID(), committed.ContextID)
		assert.Equal(t, internal_type.InterruptionDetectedPacket{
			ContextID: committed.ContextID, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		}, packets[5])
		assert.Equal(t, internal_type.SpeechToTextPacket{
			ContextID: committed.ContextID, Script: "wait", Interim: true,
		}, packets[6])
	})
}

func TestInterruptionCancellationContinuesOnlyPendingOutput(t *testing.T) {
	for _, commit := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
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
				assert.Equal(t, &protos.ConversationPlaybackControl{Id: previousContextID, Kind: protos.ConversationPlaybackControl_PAUSE}, streamer.sent[0])
				assert.Equal(t, &protos.ConversationPlaybackControl{Id: previousContextID, Kind: protos.ConversationPlaybackControl_FLUSH}, streamer.sent[1])
			} else {
				assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: previousContextID, Kind: protos.ConversationPlaybackControl_PAUSE}, &protos.ConversationPlaybackControl{Id: previousContextID, Kind: protos.ConversationPlaybackControl_CONTINUE}}, streamer.sent)
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
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(requestor.messageLifecycle.StopUnclearInput)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		started, release := make(chan struct{}), make(chan struct{})
		requestor.textToSpeechTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameTextToSpeechInterrupt, started: started, release: release,
		}
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
		assert.Equal(t, protos.ConversationPlaybackControl_PAUSE, streamer.sent[0].(*protos.ConversationPlaybackControl).GetKind())
		assert.Equal(t, protos.ConversationPlaybackControl_FLUSH, streamer.sent[1].(*protos.ConversationPlaybackControl).GetKind())
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

func TestInterruptionHoldsTranscriptsUntilReplayCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
		t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
		started, release := make(chan struct{}), make(chan struct{})
		defer close(release)
		eos := &recordingEOSExecutor{onExecute: func(ctx context.Context, packet internal_type.Packet) error {
			if vad, ok := packet.(internal_type.InterruptionDetectedPacket); ok && vad.Event == internal_type.InterruptionEventStart {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}}
		requestor.endOfSpeechExecutor = eos
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(t.Context(), internal_type.InterruptionDetectedPacket{
			ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		handler.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait", Interim: true})
		<-started
		current := requestor.GetID()
		require.NotEqual(t, previous, current)
		handler.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: current, Script: "wait please"})
		handler.HandleEndOfSpeech(t.Context(), internal_type.EndOfSpeechPacket{ContextID: current, Speech: "wait please"})
		synctest.Wait()
		require.Len(t, eos.snapshotExecuted(), 2, "newer transcript must wait behind replayed VAD-start")
		release <- struct{}{}
		synctest.Wait()
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 4)
		assert.Equal(t, internal_type.SpeechToTextPacket{ContextID: current, Script: "wait", Interim: true}, packets[2])
		assert.Equal(t, internal_type.SpeechToTextPacket{ContextID: current, Script: "wait please"}, packets[3])
		assert.Equal(t, adapter_lifecycle.MessageStateUserFinished, requestor.messageLifecycle.State())
		assert.Equal(t, []internal_type.Packet{
			internal_type.UserInputPacket{ContextID: current, Text: "wait please"},
		}, drainIngressPackets(requestor), "derived input must pass ordinary admission")
		time.Sleep(2 * time.Second)
		synctest.Wait()
		for _, packet := range drainEgressPackets(requestor) {
			_, expired := packet.(internal_type.UnclearInputExpiredPacket)
			assert.False(t, expired, "replayed final must stop the unclear-input timer")
		}
	})
}

func TestInterruptionReplayRespectsInputPolicy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
		t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		started, release := make(chan struct{}), make(chan struct{})
		defer close(release)
		requestor.textToSpeechTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameTextToSpeechInterrupt, started: started, release: release,
		}
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(t.Context(), internal_type.InterruptionDetectedPacket{
			ContextID: previous, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
		})
		handler.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait", Interim: true})
		<-started
		requestor.dispatchRoute.ApplyPolicy(internal_type.DispatchPolicy{
			Target: internal_type.PacketNameSpeechToText, Action: internal_type.DispatchActionIgnore,
		})
		release <- struct{}{}
		synctest.Wait()
		for _, packet := range eos.snapshotExecuted() {
			assert.NotEqual(t, internal_type.PacketNameSpeechToText, packet.PacketName())
		}
		requestor.dispatchRoute.ApplyPolicy(internal_type.DispatchPolicy{
			Target: internal_type.PacketNameSpeechToText, Action: internal_type.DispatchActionPassthrough,
		})
		requestor.dispatch(t.Context(), internal_type.SpeechToTextPacket{ContextID: requestor.GetID(), Script: "wait please"})
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 3)
		assert.Equal(t, internal_type.SpeechToTextPacket{ContextID: requestor.GetID(), Script: "wait please"}, packets[2])
	})
}

type blockedSpeechAdmission struct {
	adapter_lifecycle.MessageLifecycle
	admitted chan struct{}
	release  chan struct{}
}

func (l *blockedSpeechAdmission) OnUserSpeech(packet internal_type.SpeechToTextPacket, adaptive bool) (*internal_type.TurnChangePacket, internal_type.SpeechToTextPacket) {
	turn, admitted := l.MessageLifecycle.OnUserSpeech(packet, adaptive)
	if admitted.ContextID != "" && !packet.Interim {
		close(l.admitted)
		<-l.release
	}
	return turn, admitted
}

func TestInterruptionAdmittedSpeechKeepsItsContext(t *testing.T) {
	for _, scenario := range []string{"current context", "previous context", "empty context"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Please repeat", adapter_lifecycle.WithInterruption(true))
				t.Cleanup(func() { requestor.messageLifecycle.CancelInterruption() })
				eos := &recordingEOSExecutor{}
				requestor.endOfSpeechExecutor = eos
				handler := requestorDispatchHandler{r: requestor}
				previous := requestor.GetID()
				handler.HandleInterruptionDetected(t.Context(), internal_type.InterruptionDetectedPacket{
					Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart,
				})
				handler.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
				synctest.Wait()
				current := requestor.GetID()
				time.Sleep(time.Second)
				synctest.Wait()
				var expired internal_type.UnclearInputExpiredPacket
				for _, packet := range drainEgressPackets(requestor) {
					if packet, ok := packet.(internal_type.UnclearInputExpiredPacket); ok {
						expired = packet
					}
				}
				require.Equal(t, current, expired.ContextID)
				admission := &blockedSpeechAdmission{
					MessageLifecycle: requestor.messageLifecycle, admitted: make(chan struct{}), release: make(chan struct{}),
				}
				defer close(admission.release)
				requestor.messageLifecycle = admission
				packet := internal_type.SpeechToTextPacket{ContextID: current, Script: "Wait please"}
				switch scenario {
				case "previous context":
					packet.ContextID = previous
				case "empty context":
					packet.ContextID = ""
				}
				finished := make(chan struct{})
				go func() {
					handler.HandleSpeechToText(t.Context(), packet)
					close(finished)
				}()
				<-admission.admitted
				handler.HandleUnclearInputExpired(t.Context(), expired)
				require.NotEqual(t, current, requestor.GetID())
				admission.release <- struct{}{}
				<-finished
				var transcripts []internal_type.SpeechToTextPacket
				for _, packet := range eos.snapshotExecuted() {
					if packet, ok := packet.(internal_type.SpeechToTextPacket); ok {
						transcripts = append(transcripts, packet)
					}
				}
				assert.Equal(t, []internal_type.SpeechToTextPacket{
					{ContextID: current, Script: "wait", Interim: true},
				}, transcripts, "a stale admitted final must not be moved into the prompt's context")
			})
		})
	}
}

func TestInterruptionDeadlineDoesNotWaitForProviderWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
		started, release := make(chan struct{}), make(chan struct{})
		requestor.speechToTextTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameSpeechToTextStart, started: started, release: release,
		}
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
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_PAUSE}, &protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_CONTINUE}}, streamer.sent)
		streamer.mu.Unlock()
		close(release)
	})
}

func TestInterruptionIgnoresStaleAndUngatedTranscripts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
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
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_PAUSE}, &protos.ConversationPlaybackControl{Id: previous, Kind: protos.ConversationPlaybackControl_CONTINUE}}, streamer.sent)
		streamer.mu.Unlock()
	})
}

func TestInterruptionIgnoresStaleVADEndOutsidePreviousContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle = adapter_lifecycle.NewMessageLifecycle(adapter_lifecycle.WithContextID("current"), adapter_lifecycle.WithMode(type_enums.AudioMode), adapter_lifecycle.WithInterruption(true))
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
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

func (l *recordingInterruptionLifecycle) OnInterruptionDetected(p internal_type.InterruptionDetectedPacket, bargeInTrigger string) adapter_lifecycle.InterruptionDecision {
	decision := l.MessageLifecycle.OnInterruptionDetected(p, bargeInTrigger)
	if decision.Pause != nil {
		l.pauses = append(l.pauses, *decision.Pause)
	}
	return decision
}

func TestInterruptionRejectsStaleDecisionPackets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
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
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_PAUSE},
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_CONTINUE},
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_PAUSE},
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
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_PAUSE},
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_CONTINUE},
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_PAUSE},
			&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_CONTINUE},
		}, streamer.sent)
		streamer.mu.Unlock()
		handler.HandleFinalizeBehavior(context.Background(), internal_type.FinalizeBehaviorPacket{})
	})
}

func TestInterruptionFinalizationFencesDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD, adapter_lifecycle.WithInterruption(true))
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
		assert.Equal(t, []proto.Message{&protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_PAUSE}, &protos.ConversationPlaybackControl{Id: contextID, Kind: protos.ConversationPlaybackControl_CONTINUE}}, streamer.sent)
		streamer.mu.Unlock()
	})
}
