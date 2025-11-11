package type_enums

import (
	"database/sql/driver"
	"encoding/json"
)

type AssistantProvider string

const (
	AGENTKIT  AssistantProvider = "AGENTKIT"
	WEBSOCKET AssistantProvider = "WEBSOCKET"
	MODEL     AssistantProvider = "MODEL"
)

func (m AssistantProvider) String() string {
	return string(m)
}

func (c AssistantProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c AssistantProvider) Value() (driver.Value, error) {
	return string(c), nil
}

func ToAssistantProvider(s string) AssistantProvider {
	switch s {
	case "AGENTKIT":
		return AGENTKIT
	case "WEBSOCKET":
		return WEBSOCKET
	default:
		return MODEL // or any other default status you prefer
	}
}
