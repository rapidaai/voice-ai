// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"errors"

	"github.com/google/uuid"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

var (
	ErrAssistantNotInitialized    = errors.New("assistant is not initialized")
	ErrConversationNotInitialized = errors.New("conversation is not initialized")
)

func (dm *genericRequestor) Assistant() (*internal_assistant_entity.Assistant, error) {
	if dm.assistant == nil {
		return nil, ErrAssistantNotInitialized
	}
	return dm.assistant, nil
}

func (gr *genericRequestor) Conversation() (*internal_conversation_entity.AssistantConversation, error) {
	if gr.assistantConversation == nil {
		return nil, ErrConversationNotInitialized
	}
	return gr.assistantConversation, nil
}

func (gr *genericRequestor) speechToTextTransformerConfig() (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	switch gr.source {
	case utils.PhoneCall:
		if a := gr.assistant; a != nil && a.AssistantPhoneDeployment != nil && a.AssistantPhoneDeployment.InputAudio != nil {
			return a.AssistantPhoneDeployment.InputAudio, nil
		}

	case utils.SDK:
		if a := gr.assistant; a != nil && a.AssistantApiDeployment != nil && a.AssistantApiDeployment.InputAudio != nil {
			return a.AssistantApiDeployment.InputAudio, nil
		}

	case utils.WebPlugin:
		if a := gr.assistant; a != nil && a.AssistantWebPluginDeployment != nil && a.AssistantWebPluginDeployment.InputAudio != nil {
			return a.AssistantWebPluginDeployment.InputAudio, nil
		}

	case utils.Debugger:
		if a := gr.assistant; a != nil && a.AssistantDebuggerDeployment != nil && a.AssistantDebuggerDeployment.InputAudio != nil {
			return a.AssistantDebuggerDeployment.InputAudio, nil
		}
	}
	return nil, errors.New("audio is not enabled for the source")
}

func (gr *genericRequestor) textToSpeechTransformerConfig() (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	switch gr.source {
	case utils.PhoneCall:
		if a := gr.assistant; a != nil && a.AssistantPhoneDeployment != nil && a.AssistantPhoneDeployment.OutputAudio != nil {
			return a.AssistantPhoneDeployment.OutputAudio, nil
		}

	case utils.SDK:
		if a := gr.assistant; a != nil && a.AssistantApiDeployment != nil && a.AssistantApiDeployment.OutputAudio != nil {
			return a.AssistantApiDeployment.OutputAudio, nil
		}

	case utils.WebPlugin:
		if a := gr.assistant; a != nil && a.AssistantWebPluginDeployment != nil && a.AssistantWebPluginDeployment.OutputAudio != nil {
			return a.AssistantWebPluginDeployment.OutputAudio, nil
		}

	case utils.Debugger:
		if a := gr.assistant; a != nil && a.AssistantDebuggerDeployment != nil && a.AssistantDebuggerDeployment.OutputAudio != nil {
			return a.AssistantDebuggerDeployment.OutputAudio, nil
		}
	}
	return nil, errors.New("audio is not enabled for the source")
}

func (dm *genericRequestor) Auth() *types.Authentication {
	return dm.auth
}

func (tc *genericRequestor) Metadata() map[string]interface{} {
	return tc.metadata
}

func (r *genericRequestor) identifier(config *protos.ConversationInitialization) string {
	switch identity := config.GetUserIdentity().(type) {
	case *protos.ConversationInitialization_Phone:
		return identity.Phone.GetPhoneNumber()
	case *protos.ConversationInitialization_Web:
		return identity.Web.GetUserId()
	default:
		return uuid.NewString()
	}
}
