package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type PromptMap map[string]interface{}

// Value Marshal
func (jsonField PromptMap) Value() (driver.Value, error) {
	return json.Marshal(jsonField)
}

// Scan Unmarshal
func (jsonField *PromptMap) Scan(value interface{}) error {
	if value == nil {
		*jsonField = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, jsonField)
	case string:
		return json.Unmarshal([]byte(v), jsonField)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// [{"name": "script", "type": "string"}]}
type PromptVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue"`
}

// PromptTemplate represents a single message in a conversation with a role and content.
// Role typically can be "system", "user", or "assistant".
type PromptTemplate struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AgentPromptTemplate defines the structure for an agent's prompt template.
// It includes the type of prompt, the actual prompt text, and any variables used in the prompt.
// type AgentPromptTemplate struct {
// 	Type      string            `json:"type"`
// 	Prompt    string            `json:"prompt"`
// 	Variables []*PromptVariable `json:"promptVariables"`
// }

func (pt *PromptTemplate) GetRole() string {
	return pt.Role
}
func (pt *PromptTemplate) GetContent() string {
	return pt.Content
}

type TextChatCompletePromptTemplate struct {
	Prompt    []*PromptTemplate `json:"prompt"`
	Variables []*PromptVariable `json:"promptVariables"`
}

func (jsonField *PromptMap) GetTextChatCompleteTemplate() (template *TextChatCompletePromptTemplate) {
	jsonData, err := json.Marshal(jsonField)
	if err != nil {
		return
	}
	if err = json.Unmarshal(jsonData, &template); err != nil {
		return
	}

	if len(template.Variables) > 0 {
		return
	}
	return
}

// func (jsonField *PromptMap) GetAgentPromptTemplate() (template *AgentPromptTemplate) {
// 	jsonData, err := json.Marshal(jsonField)
// 	if err != nil {
// 		return
// 	}
// 	if err = json.Unmarshal(jsonData, &template); err != nil {
// 		return
// 	}

// 	if len(template.Variables) > 0 {
// 		return
// 	}

// 	return
// }
