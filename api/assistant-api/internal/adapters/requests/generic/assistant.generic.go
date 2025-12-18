// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software.
// Unauthorized copying, modification, or redistribution is strictly prohibited.
package internal_adapter_request_generic

import (
	"errors"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

func (dm *GenericRequestor) Assistant() *internal_assistant_entity.Assistant {
	return dm.assistant
}

func (gr *GenericRequestor) Conversation() *internal_conversation_gorm.AssistantConversation {
	return gr.assistantConversation
}

func (gr *GenericRequestor) GetSpeechToTextTransformer() (
	*internal_assistant_entity.AssistantDeploymentAudio,
	error,
) {
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

func (gr *GenericRequestor) GetTextToSpeechTransformer() (*internal_assistant_entity.AssistantDeploymentAudio, error) {
	switch gr.source {
	case utils.PhoneCall:
		if a := gr.assistant; a != nil && a.AssistantPhoneDeployment != nil && a.AssistantPhoneDeployment.OuputAudio != nil {
			return a.AssistantPhoneDeployment.OuputAudio, nil
		}

	case utils.SDK:
		if a := gr.assistant; a != nil && a.AssistantApiDeployment != nil && a.AssistantApiDeployment.OuputAudio != nil {
			return a.AssistantApiDeployment.OuputAudio, nil
		}

	case utils.WebPlugin:
		if a := gr.assistant; a != nil && a.AssistantWebPluginDeployment != nil && a.AssistantWebPluginDeployment.OuputAudio != nil {
			return a.AssistantWebPluginDeployment.OuputAudio, nil
		}

	case utils.Debugger:
		if a := gr.assistant; a != nil && a.AssistantDebuggerDeployment != nil && a.AssistantDebuggerDeployment.OuputAudio != nil {
			return a.AssistantDebuggerDeployment.OuputAudio, nil
		}
	}
	return nil, errors.New("audio is not enabled for the source")
}

func (gr *GenericRequestor) GetAssistant(
	auth types.SimplePrinciple,
	assistantId uint64,
	version string) (*internal_assistant_entity.Assistant, error) {
	versionId := utils.GetVersionDefinition(version)
	assistantOpts := &internal_services.GetAssistantOption{
		InjectTag: false,
		//
		InjectAssistantProvider:      true,
		InjectKnowledgeConfiguration: true,

		// these are very specific for deployment

		InjectTool:          true,
		InjectAnalysis:      true,
		InjectWebhook:       true,
		InjectConversations: false,
	}
	switch gr.source {
	case utils.PhoneCall:
		assistantOpts.InjectPhoneDeployment = true
	case utils.Whatsapp:
		assistantOpts.InjectWhatsappDeployment = true
	case utils.SDK:
		assistantOpts.InjectApiDeployment = true
	case utils.WebPlugin:
		assistantOpts.InjectWebpluginDeployment = true
	case utils.Debugger:
		assistantOpts.InjectDebuggerDeployment = true
	}
	return gr.assistantService.Get(gr.ctx, auth, assistantId, versionId, assistantOpts)
}
