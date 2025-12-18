// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software.
// Unauthorized copying, modification, or redistribution is strictly prohibited.
package internal_adapter_request_generic

import (
	"context"
	"errors"
	"strings"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

func (gr *GenericRequestor) GetBehavior() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
	switch gr.source {
	case utils.PhoneCall:
		if a := gr.assistant; a != nil && a.AssistantPhoneDeployment != nil {
			return &a.AssistantPhoneDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Whatsapp:
		if a := gr.assistant; a != nil && a.AssistantWhatsappDeployment != nil {
			return &a.AssistantWhatsappDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.SDK:
		if a := gr.assistant; a != nil && a.AssistantApiDeployment != nil {
			return &a.AssistantApiDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.WebPlugin:
		if a := gr.assistant; a != nil && a.AssistantWebPluginDeployment != nil {
			return &a.AssistantWebPluginDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Debugger:
		if a := gr.assistant; a != nil && a.AssistantDebuggerDeployment != nil {
			return &a.AssistantDebuggerDeployment.AssistantDeploymentBehavior, nil
		}
	}
	return nil, errors.New("deployment is not enabled for source")
}

func (communication *GenericRequestor) OnGreet(ctx context.Context) error {
	message := communication.messaging.Create(type_enums.UserActor, "")
	utils.Go(ctx, func() {
		if err := communication.OnCreateMessage(ctx, message.GetId(), message); err != nil {
			communication.logger.Errorf("Error in OnCreateMessage: %v", err)
		}
	})
	behavior, err := communication.GetBehavior()
	if err != nil {
		communication.logger.Errorf("error while fetching deployment behavior: %v", err)
		return nil
	}

	if behavior.Greeting == nil {
		communication.logger.Errorf("error while fetching deployment behavior: %v", err)
		return nil
	}
	greetingCnt := communication.templateParser.Parse(*behavior.Greeting, communication.GetArgs())
	if strings.TrimSpace(greetingCnt) == "" {
		communication.logger.Warnf("empty greeting message, could be space in the table or argument contains space")
		return nil
	}
	greetings := types.NewMessage(
		"assistant", &types.Content{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(greetingCnt),
		},
	)

	// sending greeting
	if err := communication.OnGeneration(ctx, message.GetId(), greetings); err != nil {
		communication.logger.Errorf("error while sending greeting message: %v", err)
		return nil
	}

	// mark complete of greeting
	if err := communication.OnGenerationComplete(ctx, message.GetId(), greetings, nil); err != nil {
		communication.logger.Errorf("error while completing greeting message: %v", err)
	}
	return nil
}

func (communication *GenericRequestor) OnError(ctx context.Context, messageId string) error {
	behavior, err := communication.GetBehavior()
	if err != nil {
		communication.logger.Warnf("no on error message setup for assistant.")
		return nil
	}
	mistakeContent := communication.templateParser.Parse(*behavior.Mistake, communication.GetArgs())
	if strings.TrimSpace(mistakeContent) == "" {
		communication.logger.Warnf("empty on error message, could be space in the table or argument contains space")
		mistakeContent = "Oops! It looks like something went wrong. Let me look into that for you right away. I really appreciate your patience—hang tight while I get this sorted!"
	}
	msg := types.NewMessage(
		"assistant", &types.Content{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(mistakeContent),
		})
	if err := communication.OnGeneration(ctx, messageId, msg); err != nil {
		communication.logger.Errorf("error while sending on error message: %v", err)
		return nil
	}
	if err := communication.OnGenerationComplete(ctx, messageId, msg, nil); err != nil {
		communication.logger.Errorf("error while completing on error message: %v", err)
	}
	return nil
}
