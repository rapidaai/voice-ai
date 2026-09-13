package adapter_internal

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	internal_end_of_speech "github.com/rapidaai/api/assistant-api/internal/end_of_speech"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type blockedUserInputLifecycle struct {
	adapter_lifecycle.MessageLifecycle
	accepted chan struct{}
	release  chan struct{}
}

func (l *blockedUserInputLifecycle) OnUserInput(packet internal_type.UserInputPacket) (internal_type.UserInputPacket, []internal_type.Packet) {
	input, packets := l.MessageLifecycle.OnUserInput(packet)
	close(l.accepted)
	<-l.release
	return input, packets
}

func TestReplaySupersededInputDoesNotStartGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requestor := newInterruptionTestRequestor(internal_options.BargeInTriggerVAD)
		handler := requestorDispatchHandler{r: requestor}
		lifecycle := &blockedUserInputLifecycle{
			MessageLifecycle: adapter_lifecycle.NewMessageLifecycle(
				adapter_lifecycle.WithContextID("first"), adapter_lifecycle.WithMode(type_enums.AudioMode),
				adapter_lifecycle.WithDispatch(handler.HandleMessageLifecyclePacket)),
			accepted: make(chan struct{}), release: make(chan struct{}),
		}
		defer close(lifecycle.release)
		requestor.messageLifecycle = lifecycle
		assistant := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 1)}
		requestor.assistantExecutor = assistant
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			handler.HandleAdmittedInput(t.Context(), internal_type.UserInputPacket{ContextID: "first", Text: "wait"})
		}()
		<-lifecycle.accepted
		handler.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: "first", Script: "and one more thing"})
		require.NotEqual(t, "first", requestor.GetID())
		lifecycle.release <- struct{}{}
		<-finished
		synctest.Wait()
		require.Empty(t, assistant.packets)
		require.Equal(t, adapter_lifecycle.MessageStateUserListening, lifecycle.State())
	})
}

func TestReplayPostEOSSpeechAdmitsSuccessor(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		withEOS      bool
		duringReplay bool
		interim      bool
	}{
		{name: "silence EOS/final during replay", withEOS: true, duringReplay: true},
		{name: "silence EOS/interim during replay", withEOS: true, duringReplay: true, interim: true},
		{name: "silence EOS/final after replay", withEOS: true},
		{name: "silence EOS/interim after replay", withEOS: true, interim: true},
		{name: "no EOS/final during replay", duringReplay: true},
		{name: "no EOS/interim during replay", duringReplay: true, interim: true},
		{name: "no EOS/final after replay"},
		{name: "no EOS/interim after replay", interim: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started, release := make(chan struct{}), make(chan struct{})
				defer close(release)
				var r *genericRequestor
				r = newInterruptionTestRequestor(internal_options.BargeInTriggerVAD,
					adapter_lifecycle.WithDispatch(func(ctx context.Context, packet internal_type.Packet) {
						requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket(ctx, packet)
						if speech, ok := packet.(internal_type.SpeechToTextPacket); ok && speech.Script == "wait" {
							close(started)
							<-release
						}
					}),
				)
				defer r.messageLifecycle.CancelInterruption()
				h := requestorDispatchHandler{r: r}
				if scenario.withEOS {
					eos, err := internal_end_of_speech.New(
						internal_end_of_speech.WithContext(t.Context()),
						internal_end_of_speech.WithOptions(utils.Option{
							internal_end_of_speech.EndOfSpeechOptionsKeyProvider: string(internal_end_of_speech.SilenceBasedEndOfSpeech),
						}),
						internal_end_of_speech.WithOnPacket(r.OnPacket),
					)
					require.NoError(t, err)
					defer eos.Close(context.Background())
					r.endOfSpeechExecutor = &recordingEOSExecutor{onExecute: eos.Execute}
				}
				assistant := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 8)}
				r.assistantExecutor = assistant
				previous := r.GetID()
				h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: previous, Script: "wait"})
				<-started
				firstTurn := r.GetID()
				require.NotEqual(t, previous, firstTurn)
				if !scenario.duringReplay {
					release <- struct{}{}
					synctest.Wait()
				}

				// Advance the real silence timer while the first replay callback can remain blocked.
				time.Sleep(2 * time.Second)
				synctest.Wait()
				var firstEOS []internal_type.EndOfSpeechPacket
				for _, packet := range drainIngressPackets(r) {
					if eos, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						firstEOS = append(firstEOS, eos)
					}
					r.dispatch(t.Context(), packet)
				}
				require.Len(t, firstEOS, 1)
				require.Equal(t, firstTurn, firstEOS[0].ContextID)
				require.Equal(t, "wait", firstEOS[0].Speech)
				if scenario.duringReplay {
					require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
				} else {
					require.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
				}

				const nextText = "and one more thing"
				h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{
					ContextID: firstTurn, Script: nextText, Interim: scenario.interim,
				})
				if scenario.duringReplay {
					require.Equal(t, firstTurn, r.GetID(), "live speech must wait behind replayed EOS")
					release <- struct{}{}
				}
				synctest.Wait()
				successor := r.GetID()
				require.NotEqual(t, firstTurn, successor, "post-EOS speech starts a separate user turn")
				if scenario.interim {
					require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())
					h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: successor, Script: nextText})
					require.Equal(t, successor, r.GetID(), "the final completes the interim's successor turn")
				}
				time.Sleep(2 * time.Second)
				synctest.Wait()

				// The earlier input has not started generation, so ordinary admission may supersede it.
				var derived []internal_type.UserInputPacket
				for packets := drainIngressPackets(r); len(packets) > 0; packets = drainIngressPackets(r) {
					for _, packet := range packets {
						if input, ok := packet.(internal_type.UserInputPacket); ok {
							derived = append(derived, input)
						}
						r.dispatch(t.Context(), packet)
					}
					synctest.Wait()
				}
				require.Equal(t, []internal_type.UserInputPacket{
					{ContextID: firstTurn, Text: "wait"},
					{ContextID: successor, Text: nextText},
				}, derived)
				var accepted []internal_type.UserInputPacket
				for len(assistant.packets) > 0 {
					if input, ok := (<-assistant.packets).(internal_type.UserInputPacket); ok {
						accepted = append(accepted, input)
					}
				}
				require.Equal(t, []internal_type.UserInputPacket{{ContextID: successor, Text: nextText}}, accepted)
				var history []internal_type.MessageCreatePacket
				for r.channels.DataChannel().Len() > 0 {
					if message, ok := receiveEnvelope(t, r.channels.DataChannel()).Pkt.(internal_type.MessageCreatePacket); ok {
						history = append(history, message)
					}
				}
				require.Equal(t, []internal_type.MessageCreatePacket{
					{ContextID: successor, MessageRole: "user", Text: nextText},
				}, history, "separate utterances must not be merged into successor history")
			})
		})
	}
}

func TestReplayDerivedInputWaitsForSpeechAcrossContextRotation(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		withEOS bool
	}{
		{name: "silence EOS", withEOS: true},
		{name: "no EOS"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				firstSpeech, releaseFirst := make(chan struct{}), make(chan struct{})
				firstEOS, releaseEOS := make(chan struct{}), make(chan struct{})
				nextSpeech, releaseNext := make(chan struct{}), make(chan struct{})
				defer close(releaseFirst)
				defer close(releaseEOS)
				defer close(releaseNext)
				const nextText = "and one more thing"
				var r *genericRequestor
				r = newInterruptionTestRequestor(internal_options.BargeInTriggerVAD,
					adapter_lifecycle.WithDispatch(func(ctx context.Context, packet internal_type.Packet) {
						requestorDispatchHandler{r: r}.HandleMessageLifecyclePacket(ctx, packet)
						switch p := packet.(type) {
						case internal_type.SpeechToTextPacket:
							if p.Script == "wait" {
								close(firstSpeech)
								<-releaseFirst
							} else if p.Script == nextText {
								close(nextSpeech)
								<-releaseNext
							}
						case internal_type.EndOfSpeechPacket:
							if p.Speech == "wait" {
								close(firstEOS)
								<-releaseEOS
							}
						}
					}),
				)
				defer r.messageLifecycle.CancelInterruption()
				h := requestorDispatchHandler{r: r}
				if scenario.withEOS {
					eos, err := internal_end_of_speech.New(
						internal_end_of_speech.WithContext(t.Context()),
						internal_end_of_speech.WithOptions(utils.Option{
							internal_end_of_speech.EndOfSpeechOptionsKeyProvider: string(internal_end_of_speech.SilenceBasedEndOfSpeech),
						}),
						internal_end_of_speech.WithOnPacket(r.OnPacket),
					)
					require.NoError(t, err)
					defer eos.Close(context.Background())
					r.endOfSpeechExecutor = &recordingEOSExecutor{onExecute: eos.Execute}
				}
				assistant := &toolDispatchTestExecutor{packets: make(chan internal_type.Packet, 8)}
				r.assistantExecutor = assistant
				h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: r.GetID(), Script: "wait"})
				<-firstSpeech
				firstTurn := r.GetID()
				time.Sleep(2 * time.Second)
				synctest.Wait()
				for _, packet := range drainIngressPackets(r) {
					r.dispatch(t.Context(), packet)
				}
				releaseFirst <- struct{}{}
				<-firstEOS
				require.Equal(t, adapter_lifecycle.MessageStateUserFinished, r.messageLifecycle.State())
				inputs := drainIngressPackets(r)
				oldInput := internal_type.UserInputPacket{ContextID: firstTurn, Text: "wait"}
				require.Equal(t, []internal_type.Packet{oldInput}, inputs)

				// EOS has produced its input before the next transcript joins the replay queue.
				r.dispatch(t.Context(), oldInput)
				h.HandleSpeechToText(t.Context(), internal_type.SpeechToTextPacket{ContextID: firstTurn, Script: nextText})
				releaseEOS <- struct{}{}
				<-nextSpeech
				successor := r.GetID()
				require.NotEqual(t, firstTurn, successor)
				require.Equal(t, adapter_lifecycle.MessageStateUserListening, r.messageLifecycle.State())

				// An old derived input stays stale even while the original replay owns a rotated context.
				r.dispatch(t.Context(), oldInput)
				time.Sleep(2 * time.Second)
				synctest.Wait()
				var successorEOS []internal_type.EndOfSpeechPacket
				for _, packet := range drainIngressPackets(r) {
					if eos, ok := packet.(internal_type.EndOfSpeechPacket); ok {
						successorEOS = append(successorEOS, eos)
					}
					r.dispatch(t.Context(), packet)
				}
				require.Len(t, successorEOS, 1)
				require.Equal(t, successor, successorEOS[0].ContextID)
				require.Equal(t, nextText, successorEOS[0].Speech)
				releaseNext <- struct{}{}
				synctest.Wait()
				for packets := drainIngressPackets(r); len(packets) > 0; packets = drainIngressPackets(r) {
					for _, packet := range packets {
						r.dispatch(t.Context(), packet)
					}
					synctest.Wait()
				}
				r.dispatch(t.Context(), oldInput)
				synctest.Wait()
				var accepted []internal_type.UserInputPacket
				for len(assistant.packets) > 0 {
					if input, ok := (<-assistant.packets).(internal_type.UserInputPacket); ok {
						accepted = append(accepted, input)
					}
				}
				require.Equal(t, []internal_type.UserInputPacket{{ContextID: successor, Text: nextText}}, accepted)
				require.Equal(t, successor, r.GetID())
				var history []internal_type.MessageCreatePacket
				for r.channels.DataChannel().Len() > 0 {
					if message, ok := receiveEnvelope(t, r.channels.DataChannel()).Pkt.(internal_type.MessageCreatePacket); ok {
						history = append(history, message)
					}
				}
				require.Equal(t, []internal_type.MessageCreatePacket{
					{ContextID: successor, MessageRole: "user", Text: nextText},
				}, history, "superseded derived input must neither generate nor enter successor history")
			})
		})
	}
}
