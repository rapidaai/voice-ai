package internal_adapter_request_customizers

import (
	"fmt"
	"sync"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type Messaging interface {
	Create(ma type_enums.MessageActor, msg string) *types.Message
	GetActor() type_enums.MessageActor
	GetMessage(actor type_enums.MessageActor) (*types.Message, error)
	GetMode() type_enums.MessageMode
	SwitchMode(mm type_enums.MessageMode)
	Transition(state InteractionState) error
}

type InteractionState int

const (
	Unknown        InteractionState = 1
	UserSpeaking   InteractionState = 2
	UserCompleted  InteractionState = 3
	AgentSpeaking  InteractionState = 4
	AgentCompleted InteractionState = 5
	Interrupt      InteractionState = 6
	Interrupted    InteractionState = 7
	LLMGenerating  InteractionState = 8
)

func (s InteractionState) String() string {
	switch s {
	case Unknown:
		return "Unknown"
	case UserSpeaking:
		return "UserSpeaking"
	case UserCompleted:
		return "UserCompleted"
	case AgentSpeaking:
		return "AgentSpeaking"
	case AgentCompleted:
		return "AgentCompleted"
	case Interrupt:
		return "Interrupt"
	case Interrupted:
		return "Interrupted"
	case LLMGenerating:
		return "LLMGenerating"
	default:
		return "InvalidState"
	}
}

type messaging struct {
	logger commons.Logger
	in     *types.Message
	out    *types.Message
	actor  type_enums.MessageActor
	mode   type_enums.MessageMode
	state  InteractionState
	mutex  sync.Mutex
}

func NewMessaging(logger commons.Logger) Messaging {
	return &messaging{
		logger: logger,
		actor:  type_enums.UserActor,
		mode:   type_enums.TextMode,
		state:  Unknown,
	}
}

func (ms *messaging) GetActor() type_enums.MessageActor {
	return ms.actor
}

func (ms *messaging) GetMode() type_enums.MessageMode {
	return ms.mode
}

func (ms *messaging) SwitchMode(mm type_enums.MessageMode) {
	ms.mode = mm
}

func (ms *messaging) Create(actor type_enums.MessageActor, msg string) *types.Message {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	if actor.ActingUser() {
		if ms.in != nil {
			ms.in.MergeContent(&types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(msg),
			})
		} else {
			ms.in = types.NewMessage("user", &types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(msg),
			})
			ms.in.AddMetadata("mode", ms.mode.String())
		}
		return ms.in
	} else {
		if ms.out != nil {
			ms.out.MergeContent(&types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(msg),
			})
		} else {
			ms.out = types.NewMessage("assistant", &types.Content{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte(msg),
			})
		}
		return ms.out
	}
}

func (ms *messaging) GetMessage(actor type_enums.MessageActor) (*types.Message, error) {
	if actor.ActingAssistant() {
		if ms.out == nil {
			return nil, fmt.Errorf("invalid message for acting assistant")
		}
		return ms.out, nil
	}
	if ms.in != nil {
		return ms.in, nil
	}
	return nil, fmt.Errorf("invalid message for acting assistant")
}

func (ms *messaging) Transition(newState InteractionState) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	switch newState {
	case Unknown:
		ms.logger.Debugf("Transition error: invalid transition: cannot transition to Unknown state")
		return fmt.Errorf("Transition: invalid transition: cannot transition to Unknown state")

	case UserSpeaking:
		if ms.state == AgentSpeaking {
			ms.logger.Debugf("Transition error: invalid transition: user can't speak when agent is speaking")
			return fmt.Errorf("Transition: invalid transition: user can't speak when agent is speaking")
		}

	case AgentSpeaking:
		if ms.state == UserSpeaking || ms.state == Interrupted || ms.state == Interrupt {
			ms.logger.Debugf("Transition error: invalid transition: agent can't speak when user is speaking")
			return fmt.Errorf("Transition: invalid transition: agent can't speak when user is speaking")
		}

	case UserCompleted:
		if ms.state == UserCompleted {
			ms.logger.Debugf("Transition error: invalid transition: already completed by the user")
			return fmt.Errorf("Transition: invalid transition: already completed by the user")
		}
	case LLMGenerating:

	case AgentCompleted:
		// flushing old
		ms.out = nil
	// case AgentCompleted:
	//

	case Interrupt:
		if ms.state == Interrupted || ms.state == Interrupt {
			ms.logger.Debugf("Transition error: invalid transition: user can't interrupt multiple times")
			return fmt.Errorf("Transition: invalid transition: agent can't interrupt multiple times")
		}

		if ms.state == AgentCompleted || ms.state == AgentSpeaking {
			ms.in = nil
		}

	case Interrupted:
		if ms.state == Interrupted {
			ms.logger.Debugf("Transition error: invalid transition: user can't interrupted multiple times")
			return fmt.Errorf("Transition: invalid transition: agent can't interrupted multiple times")
		}
		if ms.state == AgentCompleted || ms.state == AgentSpeaking {
			ms.in = nil
		}

	default:
		ms.logger.Debugf("Transition error: invalid transition: unknown state %v", newState)
		return fmt.Errorf("Transition: invalid transition: unknown state %v", newState)
	}

	ms.state = newState
	return nil
}
