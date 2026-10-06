package lifecycle

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

type interruptionPhase uint8

type interruptionState struct {
	phase         interruptionPhase
	contextID     string
	previousState MessageState
	sequence      uint64
	speechActive  bool
	heldPackets   []internal_type.Packet
	timer         *time.Timer
}

// Playback controls are ordered independently of the state lock so I/O cannot block admission.
func (l *messageLifecycle) SendPlaybackControl(control proto.Message) error {
	playbackControl, ok := control.(*protos.ConversationPlaybackControl)
	if !ok {
		return ErrInvalidPlaybackControl
	}
	switch playbackControl.GetKind() {
	case protos.ConversationPlaybackControl_PAUSE, protos.ConversationPlaybackControl_CONTINUE, protos.ConversationPlaybackControl_FLUSH:
	default:
		return ErrInvalidPlaybackControl
	}
	if l.sendOutput == nil {
		return ErrSenderNotConfigured
	}
	l.playbackControlMu.Lock()
	defer l.playbackControlMu.Unlock()
	l.mu.Lock()
	switch playbackControl.GetKind() {
	case protos.ConversationPlaybackControl_PAUSE:
		if playbackControl.Id != l.contextID || playbackControl.Id != l.interruption.contextID || l.interruption.phase == interruptionDraining {
			l.mu.Unlock()
			return ErrStaleContext
		}
	case protos.ConversationPlaybackControl_CONTINUE:
		if playbackControl.Id != l.contextID || l.interruption.contextID != "" {
			l.mu.Unlock()
			return ErrStaleContext
		}
	}
	contextID := playbackControl.Id
	switch playbackControl.GetKind() {
	case protos.ConversationPlaybackControl_PAUSE:
		if contextID == l.contextID {
			l.output.paused = true
			if l.output.receiptTimer != nil {
				l.output.receiptRemaining = max(0, time.Until(l.output.receiptDeadline))
				l.output.receiptTimer.Stop()
				l.output.receiptTimer = nil
			}
		}
	case protos.ConversationPlaybackControl_FLUSH:
		if contextID == l.contextID {
			l.output.playback = playbackFailed
			l.output.receiptReceived = false
			if l.output.receiptTimer != nil {
				l.output.receiptTimer.Stop()
				l.output.receiptTimer = nil
			}
		}
	}
	l.mu.Unlock()
	if err := l.sendOutput(control); err != nil {
		l.OnMessageFailed(contextID)
		return err
	}
	if playbackControl.GetKind() == protos.ConversationPlaybackControl_CONTINUE {
		l.mu.Lock()
		if contextID == l.contextID {
			l.output.paused = false
		}
		l.mu.Unlock()
		l.awaitPlayback(contextID)
		_ = l.completeAssistantMessage(contextID)
	}
	return nil
}

func (l *messageLifecycle) CanStartIdleTimeout(contextID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return contextID == l.contextID && l.state == MessageStateAssistantIdle && l.interruption.contextID == ""
}

// CancelInterruption invalidates callbacks before releasing the held input.
func (l *messageLifecycle) CancelInterruption() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	continueContextID := ""
	if l.interruption.phase != interruptionDraining {
		continueContextID = l.interruption.contextID
	}
	if l.interruption.timer != nil {
		l.interruption.timer.Stop()
	}
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Cancel()
	}
	l.interruption = interruptionState{sequence: l.interruption.sequence + 1}
	l.previousContextID = ""
	l.pendingVADEndContextID = ""
	return continueContextID
}

// OnUserSpeech admits input or retains it until the interrupted turn finishes draining.
// Admitted speech carries the destination context captured under the lifecycle lock.
func (l *messageLifecycle) OnUserSpeech(p internal_type.SpeechToTextPacket) (*internal_type.TurnChangePacket, internal_type.SpeechToTextPacket) {
	l.mu.Lock()
	var speechContextID string
	defer func() {
		l.mu.Unlock()
		if speechContextID != "" && l.onPacket != nil {
			_ = l.onPacket(internal_type.StopIdleTimeoutPacket{ContextID: speechContextID, ResetCount: true})
			// A receipt can finish playback while the activity reset is being queued.
			l.mu.RLock()
			hasCompletedPlayback := speechContextID == l.contextID && l.output.playback == playbackCompleted
			l.mu.RUnlock()
			if hasCompletedPlayback {
				_ = l.onPacket(internal_type.StartIdleTimeoutPacket{ContextID: speechContextID})
			}
		}
	}()
	// Speech activity resets idle retries even when it does not interrupt playback.
	if strings.TrimSpace(p.Script) != "" && (p.ContextID == "" || p.ContextID == l.contextID || p.ContextID == l.interruption.contextID) {
		speechContextID = l.contextID
	}
	isMeaningful := false
	for _, token := range strings.FieldsFunc(p.Script, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsPunct(character)
	}) {
		if !slices.Contains(interruptionFillerWords[:], strings.ToLower(token)) {
			isMeaningful = true
			break
		}
	}
	if l.interruption.contextID == "" && (l.output.playback == playbackCompleted || l.state == MessageStateUserFinished || l.state == MessageStateAssistantPrompted) && strings.TrimSpace(p.Script) != "" && (p.ContextID == "" || p.ContextID == l.contextID) {
		turn := l.startSpeechTurnLocked()
		p.ContextID = turn.ContextID
		return &turn, p
	}
	if l.interruption.phase == interruptionResumed {
		// Resuming playback does not discard a delayed transcript from the same message.
		if p.ContextID != l.contextID || !isMeaningful {
			return nil, internal_type.SpeechToTextPacket{}
		}
		l.interruption.phase = interruptionIdle
	}
	if l.interruption.contextID == "" && isMeaningful && (p.ContextID == "" || p.ContextID == l.contextID) {
		switch l.state {
		case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking:
			l.interruption.sequence++
			l.interruption.phase = interruptionWaiting
			l.interruption.contextID = l.contextID
			l.interruption.previousState = l.state
		default:
			l.interruption.heldPackets = nil
		}
	}
	if l.interruption.contextID != "" {
		if p.ContextID != "" && p.ContextID != l.interruption.contextID && p.ContextID != l.contextID {
			return nil, internal_type.SpeechToTextPacket{}
		}
		if isMeaningful || (l.interruption.phase != interruptionWaiting && !p.Interim && strings.TrimSpace(p.Script) != "") {
			l.interruption.heldPackets = append(l.interruption.heldPackets, p)
			if l.interruption.phase == interruptionWaiting {
				l.interruption.phase = interruptionConfirmed
				if l.interruption.timer != nil {
					l.interruption.timer.Stop()
					l.interruption.timer = nil
				}
				return &internal_type.TurnChangePacket{
					InterruptionDecision: true, InterruptionSequence: l.interruption.sequence,
					PreviousContextID: l.interruption.contextID, PreviousState: string(l.interruption.previousState),
					Reason: "interrupted", Source: string(internal_type.InterruptionSourceVad),
					Trigger: string(internal_type.PacketNameInterruptionDetected), Text: p.Script, Time: time.Now(),
				}, internal_type.SpeechToTextPacket{}
			}
		}
		return nil, internal_type.SpeechToTextPacket{}
	}
	if p.ContextID != "" && p.ContextID != l.contextID && p.ContextID != l.previousContextID {
		return nil, internal_type.SpeechToTextPacket{}
	}
	p.ContextID = l.contextID
	return nil, p
}

func (l *messageLifecycle) HoldInput(packet internal_type.Packet) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.contextID == "" {
		return false
	}
	switch p := packet.(type) {
	case internal_type.EndOfSpeechPacket:
		if p.ContextID != l.interruption.contextID && p.ContextID != l.contextID {
			return true
		}
		for _, token := range strings.FieldsFunc(p.Speech, func(character rune) bool {
			return unicode.IsSpace(character) || unicode.IsPunct(character)
		}) {
			if !slices.Contains(interruptionFillerWords[:], strings.ToLower(token)) {
				l.interruption.heldPackets = append(l.interruption.heldPackets, p)
				return true
			}
		}
	case internal_type.UserInputPacket:
		if p.ContextID == "" {
			p.ContextID = l.contextID
		}
		if p.ContextID == l.interruption.contextID || p.ContextID == l.contextID {
			l.interruption.heldPackets = append(l.interruption.heldPackets, p)
		}
	}
	return true
}

// observeVAD returns provider packets, an admitted EOS event, or a pause candidate.
func (l *messageLifecycle) observeVAD(p internal_type.InterruptionDetectedPacket) (internal_type.InterruptionDetectedPacket, []internal_type.Packet, *internal_type.InterruptionDecisionExpiredPacket) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.phase == interruptionResumed && (p.ContextID == "" || p.ContextID == l.contextID) {
		if p.Event == internal_type.InterruptionEventStart && l.interruption.speechActive {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		if p.Event == internal_type.InterruptionEventEnd && l.interruption.speechActive {
			p.ContextID = l.contextID
			l.interruption.heldPackets = append(l.interruption.heldPackets, p)
			l.interruption.speechActive = false
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
		}
	}
	if l.interruption.contextID != "" {
		if p.ContextID != "" && p.ContextID != l.interruption.contextID && p.ContextID != l.contextID {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		if p.Event == internal_type.InterruptionEventStart && !l.interruption.speechActive {
			l.interruption.heldPackets = append(l.interruption.heldPackets, p)
			l.interruption.speechActive = true
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextStartPacket{ContextID: l.contextID}}, nil
		}
		if p.Event == internal_type.InterruptionEventEnd && l.interruption.speechActive {
			l.interruption.heldPackets = append(l.interruption.heldPackets, p)
			l.interruption.speechActive = false
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
		}
		return internal_type.InterruptionDetectedPacket{}, nil, nil
	}
	isCommittedEnd := false
	if p.ContextID != "" && p.ContextID != l.contextID {
		if p.Event != internal_type.InterruptionEventEnd || p.ContextID != l.pendingVADEndContextID {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		l.pendingVADEndContextID = ""
		isCommittedEnd = true
	}
	p.ContextID = l.contextID
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking:
		if p.Event != internal_type.InterruptionEventStart {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		l.interruption.sequence++
		l.interruption.contextID = l.contextID
		l.interruption.previousState = l.state
		l.interruption.speechActive = true
		l.interruption.phase = interruptionWaiting
		l.interruption.heldPackets = []internal_type.Packet{p}
		l.pendingVADEndContextID = ""
		return internal_type.InterruptionDetectedPacket{}, nil, &internal_type.InterruptionDecisionExpiredPacket{ContextID: l.contextID, Sequence: l.interruption.sequence}
	}
	if p.Event == internal_type.InterruptionEventStart {
		switch l.state {
		case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
			l.state = MessageStateUserSpeaking
		}
		return p, []internal_type.Packet{internal_type.SpeechToTextStartPacket{ContextID: l.contextID}}, nil
	}
	if p.Event == internal_type.InterruptionEventEnd {
		if !isCommittedEnd {
			switch l.state {
			case MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
				l.state = MessageStateUserListening
			}
		}
		return p, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
	}
	return p, nil, nil
}

// OnPlaybackPaused starts confirmation after a successful pause or handles its failure.
func (l *messageLifecycle) OnPlaybackPaused(packet internal_type.InterruptionDecisionExpiredPacket, pauseError error) *internal_type.TurnChangePacket {
	if pauseError != nil || l.onInterruptionExpired == nil {
		return l.failInterruptionPause(packet)
	}
	l.armInterruption(packet.ContextID, packet.Sequence, l.onInterruptionExpired)
	return nil
}

func (l *messageLifecycle) armInterruption(contextID string, sequence uint64, onExpired func(internal_type.InterruptionDecisionExpiredPacket)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.contextID != contextID || l.interruption.sequence != sequence || l.interruption.phase != interruptionWaiting {
		return
	}
	l.interruption.timer = time.AfterFunc(InterruptionDecisionWindow, func() {
		onExpired(internal_type.InterruptionDecisionExpiredPacket{ContextID: contextID, Sequence: sequence})
	})
}

func (l *messageLifecycle) failInterruptionPause(p internal_type.InterruptionDecisionExpiredPacket) *internal_type.TurnChangePacket {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.contextID != p.ContextID || l.interruption.sequence != p.Sequence || l.interruption.phase != interruptionWaiting {
		return nil
	}
	l.interruption.phase = interruptionConfirmed
	if l.interruption.timer != nil {
		l.interruption.timer.Stop()
		l.interruption.timer = nil
	}
	return &internal_type.TurnChangePacket{
		InterruptionDecision: true, InterruptionSequence: p.Sequence, PreviousContextID: p.ContextID,
		PreviousState: string(l.interruption.previousState), Reason: "interrupted", Source: string(internal_type.InterruptionSourceVad),
		Trigger: string(internal_type.PacketNameInterruptionDetected), Time: time.Now(),
	}
}

func (l *messageLifecycle) OnInterruptionExpired(p internal_type.InterruptionDecisionExpiredPacket) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.contextID != p.ContextID || l.interruption.sequence != p.Sequence || l.interruption.phase != interruptionWaiting {
		return ""
	}
	if l.interruption.timer != nil {
		l.interruption.timer.Stop()
		l.interruption.timer = nil
	}
	l.interruption.sequence++
	l.interruption.contextID = ""
	l.interruption.previousState = ""
	l.interruption.phase = interruptionResumed
	return p.ContextID
}

// OnTurnChange keeps held input behind downstream interruption and turn updates.
// Callbacks run synchronously without holding the lifecycle state lock.
func (l *messageLifecycle) OnTurnChange(ctx context.Context, packet internal_type.TurnChangePacket) error {
	if l.dispatchPacket == nil {
		return ErrDispatcherNotConfigured
	}
	var flushError error
	if packet.InterruptionDecision {
		if !l.beginInterruptedTurn(packet) {
			return nil
		}
		flushError = l.SendPlaybackControl(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH, Id: packet.PreviousContextID})
		var committed bool
		packet, committed = l.commitInterruptedTurn(packet)
		if !committed {
			return flushError
		}
		l.StopUnclearInput()
		l.dispatchPacket(ctx, internal_type.EndOfSpeechInterruptionPacket{ContextID: packet.PreviousContextID, Source: internal_type.InterruptionSourceVad})
		l.dispatchPacket(ctx, internal_type.TextToSpeechInterruptPacket{ContextID: packet.PreviousContextID})
		l.dispatchPacket(ctx, internal_type.LLMInterruptPacket{ContextID: packet.PreviousContextID})
		l.dispatchPacket(ctx, internal_type.StopIdleTimeoutPacket{ContextID: packet.PreviousContextID})
	}
	if packet.ContextID == "" {
		packet.ContextID = l.ContextID()
	}
	if packet.Time.IsZero() {
		packet.Time = time.Now()
	}
	l.dispatchPacket(ctx, packet)
	if !packet.InterruptionDecision {
		return flushError
	}
	for {
		heldPackets := l.finishInterruptedTurn(packet)
		if len(heldPackets) == 0 {
			_ = l.completeAssistantMessage(l.ContextID())
			return flushError
		}
		for _, heldPacket := range heldPackets {
			l.mu.RLock()
			isCurrent := l.interruption.sequence == packet.InterruptionSequence
			switch held := heldPacket.(type) {
			case internal_type.SpeechToTextPacket:
				held.ContextID = l.contextID
				heldPacket = held
			case internal_type.InterruptionDetectedPacket:
				held.ContextID = l.contextID
				heldPacket = held
			}
			l.mu.RUnlock()
			if !isCurrent {
				return flushError
			}
			l.dispatchPacket(ctx, heldPacket)
		}
	}
}

func (l *messageLifecycle) beginInterruptedTurn(p internal_type.TurnChangePacket) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p.InterruptionSequence == 0 || l.contextID != p.PreviousContextID || l.interruption.contextID != p.PreviousContextID ||
		l.interruption.sequence != p.InterruptionSequence || l.interruption.phase != interruptionConfirmed {
		return false
	}
	l.interruption.phase = interruptionDraining
	if l.interruption.timer != nil {
		l.interruption.timer.Stop()
		l.interruption.timer = nil
	}
	return true
}

func (l *messageLifecycle) commitInterruptedTurn(p internal_type.TurnChangePacket) (internal_type.TurnChangePacket, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.contextID != p.PreviousContextID || l.interruption.contextID != p.PreviousContextID || l.interruption.sequence != p.InterruptionSequence || l.interruption.phase != interruptionDraining {
		return p, false
	}
	p.PreviousContextID = l.contextID
	l.contextID = uuid.NewString()
	p.ContextID = l.contextID
	l.state = MessageStateUserListening
	if l.output.receiptTimer != nil {
		l.output.receiptTimer.Stop()
	}
	l.output = assistantOutputState{}
	l.previousContextID = p.PreviousContextID
	return p, true
}

func (l *messageLifecycle) finishInterruptedTurn(p internal_type.TurnChangePacket) []internal_type.Packet {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruption.sequence != p.InterruptionSequence || l.interruption.contextID != p.PreviousContextID {
		return nil
	}
	heldPackets := l.interruption.heldPackets
	l.interruption.heldPackets = nil
	// Derived input and expiry wait until all held speech has reached its destination turn.
	inputPackets := heldPackets[:0]
	for _, packet := range heldPackets {
		switch held := packet.(type) {
		case internal_type.SpeechToTextPacket:
			held.ContextID = l.contextID
			packet = held
		case internal_type.InterruptionDetectedPacket:
			held.ContextID = l.contextID
			packet = held
		case internal_type.EndOfSpeechPacket:
			if held.ContextID == p.PreviousContextID {
				held.ContextID = p.ContextID
				held.Speechs = slices.Clone(held.Speechs)
				for speechIndex := range held.Speechs {
					held.Speechs[speechIndex].ContextID = p.ContextID
				}
			}
			packet = held
		case internal_type.UserInputPacket:
			if held.ContextID == p.PreviousContextID {
				held.ContextID = p.ContextID
			}
			l.interruption.heldPackets = append(l.interruption.heldPackets, held)
			continue
		case internal_type.UnclearInputExpiredPacket:
			l.interruption.heldPackets = append(l.interruption.heldPackets, packet)
			continue
		}
		inputPackets = append(inputPackets, packet)
	}
	heldPackets = inputPackets
	// Live input joins the next batch until every replay callback has returned.
	if len(heldPackets) == 0 {
		heldPackets = l.interruption.heldPackets
		l.interruption.heldPackets = nil
		if l.interruption.speechActive && l.pendingVADEndContextID == "" {
			l.pendingVADEndContextID = p.PreviousContextID
		}
		l.interruption = interruptionState{sequence: l.interruption.sequence}
		return heldPackets
	}
	return heldPackets
}
