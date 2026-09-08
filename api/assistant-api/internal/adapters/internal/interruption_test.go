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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startInterruptionTestOwner(t *testing.T, requestor *genericRequestor) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	requestor.interruption = newInterruptionOwner(ctx, requestor)
	return func() {
		cancel()
		<-requestor.interruption.done
		<-requestor.interruption.workerDone
	}
}

func interruptionTestControls(requestor *genericRequestor) []internal_type.Stream {
	streamer := requestor.streamer.(*streamTestStreamer)
	streamer.mu.Lock()
	defer streamer.mu.Unlock()
	var controls []internal_type.Stream
	for _, packet := range streamer.sent {
		switch packet.(type) {
		case internal_type.PauseOutput, internal_type.ContinueOutput, internal_type.FlushOutput:
			controls = append(controls, packet)
		}
	}
	return controls
}

func TestMeaningfulInterruptionText(t *testing.T) {
	for _, text := range []string{"", " \t", "...", " UH! ", "Um, hmm...", "um,hmm", "mm MHM ah OH", "“hmm”"} {
		assert.False(t, meaningfulInterruptionText(text), "%q", text)
	}
	for _, text := range []string{"hello", "um, wait!", "hmm yes", "uh-huh", "0", "你好"} {
		assert.True(t, meaningfulInterruptionText(text), "%q", text)
	}
}

func TestInterruptionDeadlineChoosesExactlyOnce(t *testing.T) {
	for _, scenario := range []struct {
		name string
		end  bool
		text string
	}{
		{name: "active"},
		{name: "ended empty", end: true},
		{name: "ended fillers", end: true, text: "Um, HMM..."},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please")
				requestor.endOfSpeechExecutor = &recordingEOSExecutor{}
				defer startInterruptionTestOwner(t, requestor)()
				handler := requestorDispatchHandler{r: requestor}
				originalContext := requestor.GetID()
				start := internal_type.InterruptionDetectedPacket{ContextID: originalContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart}
				handler.HandleInterruptionDetected(context.Background(), start)
				synctest.Wait()
				assert.Equal(t, originalContext, requestor.GetID())
				assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}}, interruptionTestControls(requestor))
				time.Sleep(200 * time.Millisecond)
				handler.HandleInterruptionDetected(context.Background(), start)
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: originalContext, Script: scenario.text, Interim: true})
				if scenario.end {
					handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: originalContext, Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
				}
				time.Sleep(299 * time.Millisecond)
				synctest.Wait()
				assert.Len(t, interruptionTestControls(requestor), 1)
				time.Sleep(time.Millisecond)
				synctest.Wait()
				if scenario.end {
					assert.Equal(t, originalContext, requestor.GetID())
					assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.ContinueOutput{}}, interruptionTestControls(requestor))
				} else {
					assert.NotEqual(t, originalContext, requestor.GetID())
					assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.FlushOutput{}}, interruptionTestControls(requestor))
				}
				assert.False(t, requestor.unclearInputWatchdog.Stop())
				contextAfterDecision := requestor.GetID()
				time.Sleep(time.Second)
				synctest.Wait()
				assert.Equal(t, contextAfterDecision, requestor.GetID())
				assert.Len(t, interruptionTestControls(requestor), 2)
			})
		})
	}
}

func TestInterruptionMeaningfulInterimCommitsAndReplaysInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newUnclearInputTestRequestor(internal_options.BargeInTriggerVAD, 1, "Repeat please")
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		defer startInterruptionTestOwner(t, requestor)()
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
		assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.FlushOutput{}}, interruptionTestControls(requestor))
		packets := eos.snapshotExecuted()
		require.Len(t, packets, 4)
		assert.IsType(t, internal_type.EndOfSpeechInterruptionPacket{}, packets[0])
		assert.Equal(t, previous, packets[0].ContextId())
		assert.IsType(t, internal_type.InterruptionDetectedPacket{}, packets[1])
		assert.Equal(t, "um, wait", packets[2].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, "wait please", packets[3].(internal_type.SpeechToTextPacket).Script)
		assert.Equal(t, current, packets[3].ContextId())
		assert.True(t, requestor.unclearInputWatchdog.Stop())
		handler.HandleTurnChange(context.Background(), internal_type.TurnChangePacket{InterruptionDecision: true, PreviousContextID: previous})
		synctest.Wait()
		assert.Equal(t, current, requestor.GetID())
		assert.Len(t, interruptionTestControls(requestor), 2)
	})
}

func TestInterruptionPauseFailureCommitsWithoutWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.streamer = &failingOutputControlStreamer{err: errors.New("output unavailable")}
		defer startInterruptionTestOwner(t, requestor)()
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

func TestInterruptionCancellationContinuesOnlyPendingOutput(t *testing.T) {
	for _, commit := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
			stop := startInterruptionTestOwner(t, requestor)
			handler := requestorDispatchHandler{r: requestor}
			handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
			if commit {
				handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "wait", Interim: true})
			}
			synctest.Wait()
			stop()
			if commit {
				assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.FlushOutput{}}, interruptionTestControls(requestor))
			} else {
				assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.ContinueOutput{}}, interruptionTestControls(requestor))
			}
		})
	}
}

func TestInterruptionReleaseRemainsDisabled(t *testing.T) {
	assert.False(t, dispatchInterruptionEnabled)
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
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		started, release := make(chan struct{}), make(chan struct{})
		requestor.textToSpeechTransformer = blockedInterruptionTransformer{
			blockOn: internal_type.PacketNameTextToSpeechInterrupt, started: started, release: release,
		}
		defer startInterruptionTestOwner(t, requestor)()
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait", Interim: true})
		<-started
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait please", Interim: true})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "Wait please"})
		synctest.Wait()
		assert.False(t, requestor.unclearInputWatchdog.Stop())
		assert.Len(t, eos.snapshotExecuted(), 1)
		assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.FlushOutput{}}, interruptionTestControls(requestor))
		close(release)
		synctest.Wait()
		assert.False(t, requestor.unclearInputWatchdog.Stop())
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
		defer startInterruptionTestOwner(t, requestor)()
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		<-started
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		time.Sleep(interruptionDecisionWindow)
		synctest.Wait()
		assert.Equal(t, previous, requestor.GetID())
		assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.ContinueOutput{}}, interruptionTestControls(requestor))
		close(release)
	})
}

func TestInterruptionIgnoresStaleAndUngatedTranscripts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		defer startInterruptionTestOwner(t, requestor)()
		handler := requestorDispatchHandler{r: requestor}
		previous := requestor.GetID()
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "late final"})
		assert.Equal(t, previous, requestor.GetID())
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{ContextID: "obsolete", Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		assert.Empty(t, interruptionTestControls(requestor))
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventStart})
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{ContextID: "obsolete", Script: "late interim", Interim: true})
		handler.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{Source: internal_type.InterruptionSourceVad, Event: internal_type.InterruptionEventEnd})
		time.Sleep(interruptionDecisionWindow)
		synctest.Wait()
		handler.HandleSpeechToText(context.Background(), internal_type.SpeechToTextPacket{Script: "late final"})
		synctest.Wait()
		assert.Equal(t, previous, requestor.GetID())
		assert.Equal(t, []internal_type.Stream{internal_type.PauseOutput{}, internal_type.ContinueOutput{}}, interruptionTestControls(requestor))
	})
}

func TestInterruptionIgnoresStaleVADEndOutsidePreviousContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		requestor.messageLifecycle = adapter_lifecycle.NewMessageLifecycleWithContext("current", type_enums.AudioMode)
		eos := &recordingEOSExecutor{}
		requestor.endOfSpeechExecutor = eos
		defer startInterruptionTestOwner(t, requestor)()

		requestorDispatchHandler{r: requestor}.HandleInterruptionDetected(context.Background(), internal_type.InterruptionDetectedPacket{
			ContextID: "obsolete",
			Source:    internal_type.InterruptionSourceVad,
			Event:     internal_type.InterruptionEventEnd,
		})
		synctest.Wait()

		assert.Empty(t, eos.snapshotExecuted())
		assert.Empty(t, interruptionTestControls(requestor))
		assert.Equal(t, "current", requestor.GetID())
	})
}
