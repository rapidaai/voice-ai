package lifecycle

import (
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
)

// InterruptionDecisionWindow bounds the pause while speech is being confirmed.
const InterruptionDecisionWindow = 500 * time.Millisecond

// InterruptionEnabledByDefault preserves the staged interruption rollout.
const InterruptionEnabledByDefault = false

// Playback controls are ordered independently of the state lock so I/O cannot block admission.
func (l *messageLifecycle) SendPlaybackControl(control proto.Message, send func(proto.Message) error) error {
	l.playbackControlMu.Lock()
	defer l.playbackControlMu.Unlock()
	l.mu.Lock()
	if l.interruptionEnabled {
		switch message := control.(type) {
		case *protos.ConversationPlaybackPause:
			if message.Id != l.contextID || message.Id != l.interruptionContextID || l.interruptionTurnCommitted {
				l.mu.Unlock()
				return ErrStaleContext
			}
		case *protos.ConversationPlaybackContinue:
			if message.Id != l.contextID || l.interruptionContextID != "" {
				l.mu.Unlock()
				return ErrStaleContext
			}
		}
	}
	contextID := ""
	switch message := control.(type) {
	case *protos.ConversationPlaybackPause:
		contextID = message.Id
		if contextID == l.contextID {
			l.output.paused = true
			if l.output.receiptTimer != nil {
				l.output.receiptRemaining = max(0, time.Until(l.output.receiptDeadline))
				l.output.receiptTimer.Stop()
				l.output.receiptTimer = nil
			}
		}
	case *protos.ConversationPlaybackContinue:
		contextID = message.Id
	case *protos.ConversationPlaybackFlush:
		contextID = message.Id
		if contextID == l.contextID {
			l.output.failed = true
			l.output.receiptReceived = false
			if l.output.receiptTimer != nil {
				l.output.receiptTimer.Stop()
				l.output.receiptTimer = nil
			}
		}
	}
	l.mu.Unlock()
	if err := send(control); err != nil {
		l.FailAssistantMessage(contextID)
		return err
	}
	if _, ok := control.(*protos.ConversationPlaybackContinue); ok {
		l.mu.Lock()
		if contextID == l.contextID {
			l.output.paused = false
		}
		l.mu.Unlock()
		l.AwaitPlayback(contextID)
		_ = l.CompleteAssistantSpeech(contextID)
	}
	return nil
}

func (l *messageLifecycle) ConfigureInterruption(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.interruptionEnabled = enabled
}

func (l *messageLifecycle) InterruptionEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.interruptionEnabled
}

func (l *messageLifecycle) CanStartIdleTimeout(contextID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return contextID == l.contextID && l.state == MessageStateAssistantIdle && l.interruptionContextID == ""
}

// CancelInterruption invalidates callbacks before releasing the held input.
func (l *messageLifecycle) CancelInterruption() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	continueContextID := ""
	if !l.interruptionTurnCommitted {
		continueContextID = l.interruptionContextID
	}
	if l.interruptionDecisionTimer != nil {
		l.interruptionDecisionTimer.Stop()
	}
	if l.unclearInputWatchdog != nil {
		l.unclearInputWatchdog.Cancel()
	}
	l.interruptionSequence++
	l.interruptionContextID = ""
	l.interruptionPreviousState = ""
	l.interruptionSpeechActive = false
	l.interruptionResumed = false
	l.interruptionDecisionPending = false
	l.interruptionTurnCommitted = false
	l.interruptionHeldPackets = nil
	l.interruptionDecisionTimer = nil
	l.committedInterruptionContextID = ""
	l.previousInterruptionContextID = ""
	l.pendingInterruptionVADEndContextID = ""
	return continueContextID
}

// ObserveSpeech admits input or retains it until the interrupted turn is committed.
func (l *messageLifecycle) ObserveSpeech(p internal_type.SpeechToTextPacket, adaptive bool) (*internal_type.TurnChangePacket, bool, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.interruptionEnabled || !adaptive {
		return nil, true, false
	}
	isMeaningful := false
	for _, token := range strings.FieldsFunc(p.Script, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsPunct(character)
	}) {
		switch strings.ToLower(token) {
		case "", "uh", "um", "hmm", "mm", "mhm", "ah", "oh":
		default:
			isMeaningful = true
		}
		if isMeaningful {
			break
		}
	}
	if l.output.completed && isMeaningful && (p.ContextID == "" || p.ContextID == l.contextID) {
		turn := &internal_type.TurnChangePacket{PreviousContextID: l.contextID, PreviousState: string(l.state), Reason: "user_input", Source: "stt", Time: time.Now()}
		l.contextID = uuid.NewString()
		l.output = assistantOutputState{}
		l.state = MessageStateUserListening
		l.interruptionResumed = false
		l.interruptionHeldPackets = nil
		turn.ContextID = l.contextID
		return turn, true, false
	}
	if l.interruptionResumed {
		// Resuming playback does not discard a delayed transcript from the same message.
		if p.ContextID != l.contextID || !isMeaningful {
			return nil, false, false
		}
		l.interruptionResumed = false
		switch l.state {
		case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking:
			l.interruptionSequence++
			l.interruptionContextID = l.contextID
			l.interruptionPreviousState = l.state
		default:
			l.interruptionHeldPackets = nil
		}
	}
	if l.interruptionContextID != "" {
		if p.ContextID != "" && p.ContextID != l.interruptionContextID && p.ContextID != l.contextID {
			return nil, false, false
		}
		if isMeaningful || (l.interruptionDecisionPending && !p.Interim && strings.TrimSpace(p.Script) != "") {
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
			if !l.interruptionDecisionPending {
				l.interruptionDecisionPending = true
				if l.interruptionDecisionTimer != nil {
					l.interruptionDecisionTimer.Stop()
					l.interruptionDecisionTimer = nil
				}
				return &internal_type.TurnChangePacket{
					InterruptionDecision: true, InterruptionSequence: l.interruptionSequence,
					PreviousContextID: l.interruptionContextID, PreviousState: string(l.interruptionPreviousState),
					Reason: "interrupted", Source: string(internal_type.InterruptionSourceVad),
					Trigger: string(internal_type.PacketNameInterruptionDetected), Text: p.Script, Time: time.Now(),
				}, false, false
			}
		}
		return nil, false, false
	}
	if p.ContextID != "" && p.ContextID != l.contextID && p.ContextID != l.previousInterruptionContextID {
		return nil, false, false
	}
	startUnclear := p.Interim && isMeaningful && l.committedInterruptionContextID == l.contextID
	if startUnclear && l.unclearInputWatchdog != nil {
		if !l.unclearInputWatchdog.Extend(l.contextID, l.unclearInputTimeout) {
			l.unclearInputWatchdog.Start(l.contextID, l.unclearInputTimeout)
		}
	}
	if !p.Interim && strings.TrimSpace(p.Script) != "" {
		l.committedInterruptionContextID = ""
		l.previousInterruptionContextID = ""
		if l.unclearInputWatchdog != nil {
			l.unclearInputWatchdog.Stop()
		}
	}
	return nil, true, startUnclear
}

func (l *messageLifecycle) HoldInput(packet internal_type.Packet) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.interruptionEnabled || l.interruptionContextID == "" {
		return false
	}
	switch p := packet.(type) {
	case internal_type.EndOfSpeechPacket:
		if p.ContextID != l.interruptionContextID && p.ContextID != l.contextID {
			return true
		}
		for _, token := range strings.FieldsFunc(p.Speech, func(character rune) bool {
			return unicode.IsSpace(character) || unicode.IsPunct(character)
		}) {
			switch strings.ToLower(token) {
			case "", "uh", "um", "hmm", "mm", "mhm", "ah", "oh":
			default:
				l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
				return true
			}
		}
	case internal_type.UserInputPacket:
		if p.ContextID == l.interruptionContextID || p.ContextID == l.contextID {
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
		}
	}
	return true
}

// ObserveVAD returns provider packets, an admitted EOS event, or a pause candidate.
func (l *messageLifecycle) ObserveVAD(p internal_type.InterruptionDetectedPacket) (internal_type.InterruptionDetectedPacket, []internal_type.Packet, *internal_type.InterruptionDecisionExpiredPacket) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.output.completed && p.Event == internal_type.InterruptionEventStart && (p.ContextID == "" || p.ContextID == l.contextID) {
		previousContextID := l.contextID
		l.contextID = uuid.NewString()
		l.output = assistantOutputState{}
		l.state = MessageStateUserSpeaking
		l.interruptionResumed = false
		l.interruptionHeldPackets = nil
		l.pendingInterruptionVADEndContextID = previousContextID
		p.ContextID = l.contextID
		return p, []internal_type.Packet{
			internal_type.StopIdleTimeoutPacket{ContextID: previousContextID},
			internal_type.TurnChangePacket{ContextID: l.contextID, PreviousContextID: previousContextID, Reason: "user_input", Source: "vad", Time: time.Now()},
			internal_type.SpeechToTextStartPacket{ContextID: l.contextID},
		}, nil
	}
	if l.interruptionResumed && (p.ContextID == "" || p.ContextID == l.contextID) {
		if p.Event == internal_type.InterruptionEventStart && l.interruptionSpeechActive {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		if p.Event == internal_type.InterruptionEventEnd && l.interruptionSpeechActive {
			p.ContextID = l.contextID
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
			l.interruptionSpeechActive = false
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
		}
	}
	if l.interruptionContextID != "" {
		if p.ContextID != "" && p.ContextID != l.interruptionContextID && p.ContextID != l.contextID {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		if p.Event == internal_type.InterruptionEventStart && !l.interruptionSpeechActive {
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
			l.interruptionSpeechActive = true
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextStartPacket{ContextID: l.contextID}}, nil
		}
		if p.Event == internal_type.InterruptionEventEnd && l.interruptionSpeechActive {
			l.interruptionHeldPackets = append(l.interruptionHeldPackets, p)
			l.interruptionSpeechActive = false
			return internal_type.InterruptionDetectedPacket{}, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
		}
		return internal_type.InterruptionDetectedPacket{}, nil, nil
	}
	isCommittedEnd := false
	if p.ContextID != "" && p.ContextID != l.contextID {
		if p.Event != internal_type.InterruptionEventEnd || p.ContextID != l.pendingInterruptionVADEndContextID {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		l.pendingInterruptionVADEndContextID = ""
		isCommittedEnd = true
	}
	p.ContextID = l.contextID
	switch l.state {
	case MessageStateAssistantGenerating, MessageStateAssistantGenerated, MessageStateAssistantSpeaking:
		if p.Event != internal_type.InterruptionEventStart {
			return internal_type.InterruptionDetectedPacket{}, nil, nil
		}
		l.interruptionSequence++
		l.interruptionContextID = l.contextID
		l.interruptionPreviousState = l.state
		l.interruptionSpeechActive = true
		l.interruptionResumed = false
		l.interruptionDecisionPending = false
		l.interruptionTurnCommitted = false
		l.interruptionHeldPackets = []internal_type.Packet{p}
		l.pendingInterruptionVADEndContextID = ""
		return internal_type.InterruptionDetectedPacket{}, nil, &internal_type.InterruptionDecisionExpiredPacket{ContextID: l.contextID, Sequence: l.interruptionSequence}
	}
	if p.Event == internal_type.InterruptionEventStart {
		switch l.state {
		case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
			l.state = MessageStateUserSpeaking
		}
		return p, []internal_type.Packet{internal_type.SpeechToTextStartPacket{ContextID: l.contextID}}, nil
	}
	if p.Event == internal_type.InterruptionEventEnd {
		if !isCommittedEnd {
			switch l.state {
			case MessageStateAssistantIdle, MessageStateUserIdle, MessageStateUserListening, MessageStateUserSpeaking:
				l.state = MessageStateUserListening
			}
		}
		return p, []internal_type.Packet{internal_type.SpeechToTextEndPacket{ContextID: l.contextID}}, nil
	}
	return p, nil, nil
}

func (l *messageLifecycle) ArmInterruption(contextID string, sequence uint64, onExpired func(internal_type.InterruptionDecisionExpiredPacket)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruptionContextID != contextID || l.interruptionSequence != sequence || l.interruptionDecisionPending {
		return
	}
	l.interruptionDecisionTimer = time.AfterFunc(InterruptionDecisionWindow, func() {
		onExpired(internal_type.InterruptionDecisionExpiredPacket{ContextID: contextID, Sequence: sequence})
	})
}

func (l *messageLifecycle) FailInterruptionPause(p internal_type.InterruptionDecisionExpiredPacket) *internal_type.TurnChangePacket {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruptionContextID != p.ContextID || l.interruptionSequence != p.Sequence || l.interruptionDecisionPending {
		return nil
	}
	l.interruptionDecisionPending = true
	if l.interruptionDecisionTimer != nil {
		l.interruptionDecisionTimer.Stop()
		l.interruptionDecisionTimer = nil
	}
	return &internal_type.TurnChangePacket{
		InterruptionDecision: true, InterruptionSequence: p.Sequence, PreviousContextID: p.ContextID,
		PreviousState: string(l.interruptionPreviousState), Reason: "interrupted", Source: string(internal_type.InterruptionSourceVad),
		Trigger: string(internal_type.PacketNameInterruptionDetected), Time: time.Now(),
	}
}

func (l *messageLifecycle) ExpireInterruption(p internal_type.InterruptionDecisionExpiredPacket) (*internal_type.TurnChangePacket, string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.interruptionEnabled || l.interruptionContextID != p.ContextID || l.interruptionSequence != p.Sequence || l.interruptionDecisionPending {
		return nil, ""
	}
	if l.interruptionDecisionTimer != nil {
		l.interruptionDecisionTimer.Stop()
		l.interruptionDecisionTimer = nil
	}
	l.interruptionSequence++
	l.interruptionContextID = ""
	l.interruptionPreviousState = ""
	l.interruptionResumed = true
	return nil, p.ContextID
}

func (l *messageLifecycle) BeginInterruptedTurn(p internal_type.TurnChangePacket) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.interruptionEnabled || p.InterruptionSequence == 0 || l.contextID != p.PreviousContextID || l.interruptionContextID != p.PreviousContextID ||
		l.interruptionSequence != p.InterruptionSequence || !l.interruptionDecisionPending || l.interruptionTurnCommitted {
		return false
	}
	l.interruptionTurnCommitted = true
	if l.interruptionDecisionTimer != nil {
		l.interruptionDecisionTimer.Stop()
		l.interruptionDecisionTimer = nil
	}
	return true
}

func (l *messageLifecycle) CommitInterruptedTurn(p internal_type.TurnChangePacket) (internal_type.TurnChangePacket, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.contextID != p.PreviousContextID || l.interruptionContextID != p.PreviousContextID || l.interruptionSequence != p.InterruptionSequence || !l.interruptionTurnCommitted {
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
	return p, true
}

func (l *messageLifecycle) FinishInterruptedTurn(p internal_type.TurnChangePacket) []internal_type.Packet {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interruptionSequence != p.InterruptionSequence || l.interruptionContextID != p.PreviousContextID || l.contextID != p.ContextID {
		return nil
	}
	heldPackets := l.interruptionHeldPackets
	if l.interruptionSpeechActive {
		l.pendingInterruptionVADEndContextID = p.PreviousContextID
	}
	l.committedInterruptionContextID = p.ContextID
	l.previousInterruptionContextID = p.PreviousContextID
	l.interruptionContextID = ""
	l.interruptionPreviousState = ""
	l.interruptionSpeechActive = false
	l.interruptionResumed = false
	l.interruptionDecisionPending = false
	l.interruptionTurnCommitted = false
	l.interruptionHeldPackets = nil
	for index, packet := range heldPackets {
		switch held := packet.(type) {
		case internal_type.InterruptionDetectedPacket:
			held.ContextID = p.ContextID
			heldPackets[index] = held
		case internal_type.SpeechToTextPacket:
			held.ContextID = p.ContextID
			heldPackets[index] = held
		case internal_type.EndOfSpeechPacket:
			held.ContextID = p.ContextID
			held.Speechs = slices.Clone(held.Speechs)
			for speechIndex := range held.Speechs {
				held.Speechs[speechIndex].ContextID = p.ContextID
			}
			heldPackets[index] = held
		case internal_type.UserInputPacket:
			held.ContextID = p.ContextID
			heldPackets[index] = held
		}
	}
	return heldPackets
}

func (l *messageLifecycle) IsCommittedInterruption(contextID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.interruptionEnabled && contextID == l.contextID && contextID == l.committedInterruptionContextID
}
