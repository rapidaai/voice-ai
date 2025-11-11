package type_enums

import (
	"database/sql/driver"
	"encoding/json"
)

type ConversationDirection string

const (
	DIRECTION_INBOUND  ConversationDirection = "inbound"
	DIRECTION_OUTBOUND ConversationDirection = "outbound"
)

func (m ConversationDirection) String() string {
	return string(m)
}

func (c ConversationDirection) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c ConversationDirection) Value() (driver.Value, error) {
	return string(c), nil
}

func ToConversationDirection(s string) ConversationDirection {
	switch s {
	case "inbound":
		return DIRECTION_INBOUND
	case "outbound":
		return DIRECTION_OUTBOUND
	default:
		return DIRECTION_INBOUND // or any other default status you prefer
	}
}

type MessageActor string
type MessageMode string

var (
	UserActor      MessageActor = "user"
	AssistantActor MessageActor = "assistant"

	AudioMode MessageMode = "audio"
	TextMode  MessageMode = "text"
)

// String returns the string representation of MessageMode
func (m MessageMode) String() string {
	return string(m)
}
func (a MessageActor) ActingAssistant() bool {
	return a == AssistantActor
}

func (a MessageActor) ActingUser() bool {
	return a == UserActor
}

func (a MessageMode) Audio() bool {
	return a == AudioMode
}
func (a MessageMode) Text() bool {
	return a == TextMode
}

type MessageAction string

const (
	ACTION_TOOL_CALL MessageAction = "tool-call"
	ACTION_LLM_CALL  MessageAction = "llm-call"
)

func (m MessageAction) String() string {
	return string(m)
}

func (c MessageAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c MessageAction) Value() (driver.Value, error) {
	return string(c), nil
}

func ToMessageAction(s string) MessageAction {
	switch s {
	case "tool-call":
		return ACTION_TOOL_CALL
	default:
		return ACTION_LLM_CALL // or any other default status you prefer
	}
}
