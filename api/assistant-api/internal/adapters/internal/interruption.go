package adapter_internal

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode"

	adapter_lifecycle "github.com/rapidaai/api/assistant-api/internal/adapters/lifecycle"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

const dispatchInterruptionEnabled = false
const interruptionDecisionWindow = 500 * time.Millisecond

type interruptionEvent struct {
	packet   internal_type.Packet
	complete bool
	reply    chan interruptionReply
}

type interruptionReply struct {
	turn     internal_type.TurnChangePacket
	accepted bool
}

type interruptionCandidate struct {
	contextID string
	state     string
	active    bool
	deciding  bool
	committed bool
	held      []internal_type.Packet
}

type interruptionOwner struct {
	cancel     context.CancelFunc
	requestor  *genericRequestor
	events     chan interruptionEvent
	work       chan internal_type.Packet
	done       chan struct{}
	workerDone chan struct{}

	candidate        *interruptionCandidate
	timer            *time.Timer
	deadline         <-chan time.Time
	queued           []internal_type.Packet
	committedContext string
	previousContext  string
	pendingVADEnd    string
}

func newInterruptionOwner(ctx context.Context, requestor *genericRequestor) *interruptionOwner {
	ctx, cancel := context.WithCancel(ctx)
	owner := &interruptionOwner{
		cancel: cancel, requestor: requestor,
		events: make(chan interruptionEvent), work: make(chan internal_type.Packet),
		done: make(chan struct{}), workerDone: make(chan struct{}),
	}
	go owner.run(ctx)
	go owner.runWork(ctx)
	return owner
}

func (r *genericRequestor) usesInterruptionOwner() bool {
	if r.interruption == nil || !r.GetMode().Audio() {
		return false
	}
	opts := r.GetOptions()
	trigger, err := opts.GetString(internal_options.MicrophoneOptionBargeInTrigger)
	if err != nil {
		trigger, _ = opts.GetString(internal_options.MicrophoneLegacyVADOptionBargeInTrigger)
	}
	return trigger != internal_options.BargeInTriggerWord
}

func (owner *interruptionOwner) submit(ctx context.Context, event interruptionEvent) interruptionReply {
	event.reply = make(chan interruptionReply, 1)
	select {
	case <-ctx.Done():
		return interruptionReply{}
	case <-owner.done:
		return interruptionReply{}
	case owner.events <- event:
	}
	select {
	case <-ctx.Done():
		return interruptionReply{}
	case <-owner.done:
		return interruptionReply{}
	case reply := <-event.reply:
		return reply
	}
}

func (owner *interruptionOwner) run(ctx context.Context) {
	defer close(owner.done)
	defer owner.cancel()
	defer func() {
		owner.stopTimer()
		if owner.candidate != nil && !owner.candidate.deciding {
			owner.output(ctx, internal_type.ContinueOutput{})
		}
		if owner.requestor.unclearInputWatchdog != nil {
			owner.requestor.unclearInputWatchdog.Cancel()
		}
		owner.candidate = nil
		owner.queued = nil
	}()
	for {
		var work chan internal_type.Packet
		var packet internal_type.Packet
		if len(owner.queued) > 0 {
			work = owner.work
			packet = owner.queued[0]
		}
		select {
		case <-ctx.Done():
			return
		case <-owner.deadline:
			if owner.candidate.active {
				owner.decide(ctx, "")
			} else {
				owner.output(ctx, internal_type.ContinueOutput{})
				owner.stopTimer()
				owner.candidate = nil
			}
		case work <- packet:
			owner.queued[0] = nil
			owner.queued = owner.queued[1:]
		case event := <-owner.events:
			event.reply <- owner.receive(ctx, event)
		}
	}
}

func (owner *interruptionOwner) receive(ctx context.Context, event interruptionEvent) interruptionReply {
	requestor := owner.requestor
	if turn, ok := event.packet.(internal_type.TurnChangePacket); ok {
		candidate := owner.candidate
		if candidate == nil || !candidate.deciding || candidate.contextID != turn.PreviousContextID {
			return interruptionReply{}
		}
		if event.complete {
			owner.candidate = nil
			if turn.ContextID == requestor.GetID() {
				owner.committedContext = turn.ContextID
				owner.previousContext = turn.PreviousContextID
				if candidate.active {
					owner.pendingVADEnd = turn.PreviousContextID
				}
				for _, held := range candidate.held {
					owner.replay(held, turn.ContextID)
				}
			}
			return interruptionReply{accepted: true}
		}
		if candidate.committed {
			return interruptionReply{}
		}
		if requestor.GetID() != candidate.contextID {
			owner.candidate = nil
			return interruptionReply{}
		}
		oldContextID, newContextID, err := requestor.messageLifecycle.RotateContext()
		if err != nil {
			owner.candidate = nil
			return interruptionReply{}
		}
		turn.ContextID = newContextID
		turn.PreviousContextID = oldContextID
		candidate.committed = true
		owner.stopUnclear()
		_ = requestor.messageLifecycle.UserListening(newContextID)
		return interruptionReply{turn: turn, accepted: true}
	}

	switch packet := event.packet.(type) {
	case internal_type.InterruptionDetectedPacket:
		contextID := requestor.GetID()
		if owner.candidate != nil {
			candidate := owner.candidate
			if packet.ContextID != "" && packet.ContextID != candidate.contextID && packet.ContextID != contextID {
				break
			}
			if packet.Event == internal_type.InterruptionEventStart {
				if !candidate.active {
					candidate.held = append(candidate.held, packet)
					owner.queued = append(owner.queued, internal_type.SpeechToTextStartPacket{ContextID: contextID})
				}
				candidate.active = true
			} else if packet.Event == internal_type.InterruptionEventEnd && candidate.active {
				candidate.active = false
				candidate.held = append(candidate.held, packet)
				owner.queued = append(owner.queued, internal_type.SpeechToTextEndPacket{ContextID: contextID})
			}
			break
		}
		if packet.ContextID != "" && packet.ContextID != contextID {
			if packet.Event != internal_type.InterruptionEventEnd || packet.ContextID != owner.pendingVADEnd {
				break
			}
			state := requestor.messageLifecycle.State()
			if state != adapter_lifecycle.MessageStateUserSpeaking && state != adapter_lifecycle.MessageStateUserListening && state != adapter_lifecycle.MessageStateUserIdle {
				break
			}
			owner.pendingVADEnd = ""
		}
		packet.ContextID = contextID
		state := requestor.messageLifecycle.State()
		outputActive := state == adapter_lifecycle.MessageStateAssistantGenerating ||
			state == adapter_lifecycle.MessageStateAssistantGenerated || state == adapter_lifecycle.MessageStateAssistantSpeaking
		if outputActive {
			if packet.Event != internal_type.InterruptionEventStart {
				break
			}
			owner.pendingVADEnd = ""
			owner.candidate = &interruptionCandidate{
				contextID: contextID, state: string(state), active: true,
				held: []internal_type.Packet{packet},
			}
			owner.timer = time.NewTimer(interruptionDecisionWindow)
			owner.deadline = owner.timer.C
			owner.queued = append(owner.queued, internal_type.SpeechToTextStartPacket{ContextID: contextID})
			if !owner.output(ctx, internal_type.PauseOutput{}) {
				owner.decide(ctx, "")
			}
			break
		}
		if packet.Event == internal_type.InterruptionEventStart {
			_ = requestor.messageLifecycle.UserSpeaking(contextID)
			owner.queued = append(owner.queued, internal_type.SpeechToTextStartPacket{ContextID: contextID})
		} else if packet.Event == internal_type.InterruptionEventEnd {
			_ = requestor.messageLifecycle.UserListening(contextID)
			owner.queued = append(owner.queued, internal_type.SpeechToTextEndPacket{ContextID: contextID})
		}
		owner.queued = append(owner.queued, packet)
	case internal_type.SpeechToTextPacket:
		if candidate := owner.candidate; candidate != nil {
			if packet.ContextID != "" && packet.ContextID != candidate.contextID && packet.ContextID != requestor.GetID() {
				break
			}
			if meaningfulInterruptionText(packet.Script) || (candidate.deciding && !packet.Interim && strings.TrimSpace(packet.Script) != "") {
				candidate.held = append(candidate.held, packet)
				if !candidate.deciding {
					owner.decide(ctx, packet.Script)
				}
			}
			break
		}
		if packet.ContextID != "" && packet.ContextID != requestor.GetID() && packet.ContextID != owner.previousContext {
			break
		}
		switch requestor.messageLifecycle.State() {
		case adapter_lifecycle.MessageStateUserIdle, adapter_lifecycle.MessageStateUserListening,
			adapter_lifecycle.MessageStateUserSpeaking, adapter_lifecycle.MessageStateUserThinking:
			owner.replay(packet, requestor.GetID())
		}
	case internal_type.EndOfSpeechPacket:
		if owner.candidate != nil {
			if packet.ContextID == owner.candidate.contextID || packet.ContextID == requestor.GetID() {
				if meaningfulInterruptionText(packet.Speech) {
					owner.candidate.held = append(owner.candidate.held, packet)
				}
			}
			break
		}
		if packet.ContextID == requestor.GetID() {
			owner.replay(packet, packet.ContextID)
		}
	case internal_type.UserInputPacket:
		if packet.ContextID == "" {
			packet.ContextID = requestor.GetID()
		}
		if owner.candidate != nil {
			if (packet.ContextID == owner.candidate.contextID || packet.ContextID == requestor.GetID()) && strings.TrimSpace(packet.Text) != "" {
				owner.candidate.held = append(owner.candidate.held, packet)
			}
			break
		}
		if packet.ContextID == requestor.GetID() && strings.TrimSpace(packet.Text) != "" {
			owner.stopUnclear()
			owner.queued = append(owner.queued, packet)
		}
	case internal_type.UserTextReceivedPacket:
		owner.stopUnclear()
		if owner.candidate != nil {
			if !owner.candidate.deciding {
				owner.output(ctx, internal_type.FlushOutput{})
			}
			owner.stopTimer()
			owner.candidate = nil
		}
	case internal_type.UnclearInputExpiredPacket:
		if event.complete {
			if packet.ContextID != owner.committedContext || packet.ContextID != requestor.GetID() ||
				requestor.unclearInputWatchdog == nil || !requestor.unclearInputWatchdog.AcceptExpiry(packet) {
				return interruptionReply{}
			}
			if err := requestor.messageLifecycle.UserPrompted(packet.ContextID); err != nil {
				return interruptionReply{}
			}
			previous, current, err := requestor.messageLifecycle.RotateContext()
			if err != nil {
				return interruptionReply{}
			}
			owner.stopUnclear()
			return interruptionReply{accepted: true, turn: internal_type.TurnChangePacket{ContextID: current, PreviousContextID: previous}}
		}
		if packet.ContextID == owner.committedContext && packet.ContextID == requestor.GetID() &&
			requestor.unclearInputWatchdog != nil && requestor.unclearInputWatchdog.AcceptExpiry(packet) {
			owner.queued = append(owner.queued, packet)
		}
	}
	return interruptionReply{accepted: true}
}

func (owner *interruptionOwner) decide(ctx context.Context, text string) {
	candidate := owner.candidate
	candidate.deciding = true
	owner.stopTimer()
	owner.output(ctx, internal_type.FlushOutput{})
	owner.queued = append(owner.queued, internal_type.TurnChangePacket{
		InterruptionDecision: true, PreviousContextID: candidate.contextID,
		Reason: "interrupted", Source: string(internal_type.InterruptionSourceVad),
		PreviousState: candidate.state, Trigger: string(internal_type.PacketNameInterruptionDetected),
		Text: text, Time: time.Now(),
	})
}

func (owner *interruptionOwner) replay(packet internal_type.Packet, contextID string) {
	switch held := packet.(type) {
	case internal_type.SpeechToTextPacket:
		held.ContextID = contextID
		if held.Interim && meaningfulInterruptionText(held.Script) {
			if owner.committedContext == contextID && owner.requestor.unclearInputWatchdog != nil {
				if behavior, err := owner.requestor.deploymentBehavior(); err == nil && behavior.UnclearInputTimeout != nil {
					timeout := time.Duration(*behavior.UnclearInputTimeout * float64(time.Second))
					if !owner.requestor.unclearInputWatchdog.Extend(contextID, timeout) {
						owner.requestor.unclearInputWatchdog.Start(contextID, timeout)
					}
				}
			}
		} else if !held.Interim && strings.TrimSpace(held.Script) != "" {
			owner.stopUnclear()
		}
		packet = held
	case internal_type.InterruptionDetectedPacket:
		held.ContextID = contextID
		packet = held
	case internal_type.EndOfSpeechPacket:
		held.ContextID = contextID
		held.Speechs = slices.Clone(held.Speechs)
		for index := range held.Speechs {
			held.Speechs[index].ContextID = contextID
		}
		if strings.TrimSpace(held.Speech) != "" {
			owner.stopUnclear()
		}
		packet = held
	case internal_type.UserInputPacket:
		held.ContextID = contextID
		owner.stopUnclear()
		packet = held
	}
	owner.queued = append(owner.queued, packet)
}

func (owner *interruptionOwner) stopUnclear() {
	owner.committedContext = ""
	owner.previousContext = ""
	if owner.requestor.unclearInputWatchdog != nil {
		owner.requestor.unclearInputWatchdog.Stop()
	}
}

func (owner *interruptionOwner) stopTimer() {
	if owner.timer != nil {
		owner.timer.Stop()
	}
	owner.timer = nil
	owner.deadline = nil
}

func (owner *interruptionOwner) output(ctx context.Context, control internal_type.Stream) bool {
	if err := owner.requestor.sendOutputControl(control); err != nil {
		owner.queued = append(owner.queued, internal_type.ObservabilityLogRecordPacket{
			ContextID: owner.requestor.GetID(), Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level: observability.LevelError, Message: "Interruption output control failed",
				Attributes: observability.Attributes{"component": observability.ComponentConversation.String(), "error": err.Error()},
			},
		})
		return false
	}
	return true
}

func (owner *interruptionOwner) runWork(ctx context.Context) {
	defer close(owner.workerDone)
	handler := requestorDispatchHandler{r: owner.requestor}
	for {
		select {
		case <-ctx.Done():
			return
		case packet := <-owner.work:
			switch value := packet.(type) {
			case internal_type.TurnChangePacket:
				handler.HandleTurnChange(ctx, value)
			case internal_type.SpeechToTextStartPacket:
				handler.HandleSpeechToTextStart(ctx, value)
			case internal_type.SpeechToTextEndPacket:
				handler.HandleSpeechToTextEnd(ctx, value)
			case internal_type.InterruptionDetectedPacket:
				if owner.requestor.endOfSpeechExecutor != nil {
					_ = owner.requestor.endOfSpeechExecutor.Execute(ctx, value)
				}
			case internal_type.SpeechToTextPacket:
				handler.handleSpeechToText(ctx, value)
			case internal_type.EndOfSpeechPacket:
				handler.handleEndOfSpeech(ctx, value)
			case internal_type.UserInputPacket:
				handler.handleUserInput(ctx, value)
			case internal_type.UnclearInputExpiredPacket:
				handler.handleUnclearInputExpired(ctx, value)
			case internal_type.ObservabilityLogRecordPacket:
				owner.requestor.OnPacket(ctx, value)
			}
		}
	}
}

func meaningfulInterruptionText(text string) bool {
	for _, token := range strings.FieldsFunc(text, func(char rune) bool { return unicode.IsSpace(char) || unicode.IsPunct(char) }) {
		switch strings.ToLower(token) {
		case "", "uh", "um", "hmm", "mm", "mhm", "ah", "oh":
		default:
			return true
		}
	}
	return false
}
