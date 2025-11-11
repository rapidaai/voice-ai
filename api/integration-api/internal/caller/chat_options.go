package internal_callers

import (
	"fmt"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type ChatCompletionOptions struct {
	AIOptions

	// The available tool definitions that the chat completions request can use, including caller-defined functions.
	ToolDefinitions []*ToolDefinition `json:"tool_definitions"`
}

type ToolDefinition struct {
	// type of tool
	Type string `json:"type"`

	// if type is function then definitions
	Function *FunctionDefinition `json:"functionDefinition"`
}

type FunctionDefinition struct {
	// REQUIRED; The name of the function to be called.
	Name string `json:"name"`
	// A description of what the function does. The model will use this
	// description when selecting the function and interpreting its parameters.
	Description string `json:"description"`

	// REQUIRED; The function definition details for the function tool.
	Parameters *FunctionParameter `json:"parameters"`
}

type FunctionParameter struct {
	// Required is a list of required parameter names
	Required []string `json:"required"`

	// Type specifies the data type of the parameter (e.g., "object", "array", etc.)
	Type string `json:"type"`

	// Properties is a map of parameter properties, where the key is the property name
	// and the value is a FunctionParameterProperty struct
	Properties map[string]FunctionParameterProperty `json:"properties"`
}

func (fp *FunctionParameter) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	if fp.Required != nil {
		result["required"] = fp.Required
	}

	if fp.Type != "" {
		result["type"] = fp.Type
	}

	if fp.Properties != nil {
		properties := make(map[string]interface{})
		for key, prop := range fp.Properties {
			properties[key] = prop.ToMap()
		}
		result["properties"] = properties
	}
	return result
}

type FunctionParameterProperty struct {
	// Type specifies the data type of the property (e.g., "string", "number", etc.)
	Type string `json:"type"`

	// Description provides a human-readable explanation of the property
	Description string `json:"description"`

	// Enum is an optional list of allowed values for the property
	Enum []*string `json:"enum,omitempty"`

	// Items is used for array types to describe the structure of array elements
	Items map[string]interface{} `json:"items,omitempty"`
}

func (fpp *FunctionParameterProperty) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if fpp.Type != "" {
		result["type"] = fpp.Type
	}

	if fpp.Description != "" {
		result["description"] = fpp.Description
	}

	if fpp.Enum != nil {
		result["enum"] = fpp.Enum
	}

	if fpp.Items != nil {
		result["items"] = fpp.Items
	}

	return result
}

func NewChatOptions(
	requestId uint64,
	irRequest *lexatic_backend.ChatRequest,
	preHook func(rst map[string]interface{}),
	postHook func(rst map[string]interface{}, metrics types.Metrics),
) *ChatCompletionOptions {
	cc := &ChatCompletionOptions{
		AIOptions: AIOptions{
			RequestId:      requestId,
			PreHook:        preHook,
			PostHook:       postHook,
			ModelParameter: irRequest.GetModelParameters(),
		},
	}
	err := utils.Cast(irRequest.GetToolDefinitions(), &cc.ToolDefinitions)
	if err != nil {
		fmt.Printf("initializing function definition with err %+v", err)
	}
	return cc
}
