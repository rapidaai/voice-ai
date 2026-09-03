// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	observability_collector_conversationmetadata "github.com/rapidaai/api/assistant-api/internal/observability/collectors/conversationmetadata"
	observability_collector_conversationmetric "github.com/rapidaai/api/assistant-api/internal/observability/collectors/conversationmetric"
	observability_collector_requestlog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/requestlog"
	observability_collector_toollog "github.com/rapidaai/api/assistant-api/internal/observability/collectors/toollog"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

func (d *Dispatcher) createConversation(ctx context.Context, stage SessionEstablishedPipeline) (uint64, error) {
	dirEnum := type_enums.DIRECTION_INBOUND
	if stage.Direction == sip_runtime.CallDirectionOutbound {
		dirEnum = type_enums.DIRECTION_OUTBOUND
	}

	conversationIdentifier := stage.CallAddress.From
	if stage.Direction == sip_runtime.CallDirectionOutbound {
		conversationIdentifier = stage.CallAddress.To
	}

	assistant := stage.Session.GetAssistant()
	if assistant == nil && d.assistantService != nil {
		var err error
		assistant, err = d.assistantService.Get(ctx, stage.Auth, stage.AssistantID, utils.GetVersionDefinition("latest"),
			&internal_services.GetAssistantOption{InjectPhoneDeployment: true})
		if err != nil {
			return 0, fmt.Errorf("failed to load assistant %d: %w", stage.AssistantID, err)
		}
	}

	assistantID := stage.AssistantID
	var assistantProviderID uint64
	if assistant != nil {
		assistantID = assistant.Id
		assistantProviderID = assistant.AssistantProviderId
	}
	if d.assistantConversationService == nil {
		return 0, fmt.Errorf("assistant conversation service not configured")
	}
	conversation, err := d.assistantConversationService.CreateConversation(
		ctx, stage.Auth, conversationIdentifier, assistantID, assistantProviderID, dirEnum, utils.PhoneCall,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %w", err)
	}
	return conversation.Id, nil
}

func (d *Dispatcher) ensureCallContext(ctx context.Context, stage SessionEstablishedPipeline, conversationID uint64) (*callcontext.CallContext, error) {
	if stage.Direction == sip_runtime.CallDirectionOutbound {
		contextID := stage.Session.GetContextID()
		if contextID != "" {
			if claimed, err := d.callContextStore.Claim(ctx, contextID); err == nil && claimed != nil {
				claimed.Direction = string(stage.Direction)
				claimed.CallerNumber = stage.CallAddress.To
				claimed.FromNumber = stage.CallAddress.From
				return claimed, nil
			}
			if loaded, err := d.callContextStore.Get(ctx, contextID); err == nil && loaded != nil {
				loaded.Direction = string(stage.Direction)
				loaded.CallerNumber = stage.CallAddress.To
				loaded.FromNumber = stage.CallAddress.From
				return loaded, nil
			}
		}

		callContext := &callcontext.CallContext{
			AssistantID:    stage.AssistantID,
			ConversationID: conversationID,
			Direction:      string(stage.Direction),
			Provider:       "sip",
			CallerNumber:   stage.CallAddress.To,
			FromNumber:     stage.CallAddress.From,
			ChannelUUID:    stage.Session.GetCallID(),
			ContextID:      contextID,
		}
		if err := callContext.SetAuthentication(stage.Auth); err != nil {
			return nil, err
		}
		return callContext, nil
	}

	callContext := &callcontext.CallContext{
		AssistantID:    stage.AssistantID,
		ConversationID: conversationID,
		Direction:      string(stage.Direction),
		Provider:       "sip",
		CallerNumber:   stage.CallAddress.From,
		FromNumber:     stage.CallAddress.To,
		ChannelUUID:    stage.Session.GetCallID(),
	}
	if err := callContext.SetAuthentication(stage.Auth); err != nil {
		return nil, err
	}
	if assistant := stage.Session.GetAssistant(); assistant != nil {
		callContext.AssistantProviderId = assistant.AssistantProviderId
	}
	if _, err := d.callContextStore.Save(ctx, callContext); err != nil {
		d.logger.Warnw("Failed to persist inbound SIP call context", "call_id", stage.Session.GetCallID(), "error", err)
		return callContext, nil
	}
	_, _ = d.callContextStore.Claim(ctx, callContext.ContextID)
	return callContext, nil
}

func (d *Dispatcher) setupCall(ctx context.Context, stage SessionEstablishedPipeline, conversationID uint64, cc *callcontext.CallContext) (*CallSetupResult, error) {
	assistant := stage.Session.GetAssistant()
	if assistant == nil && d.assistantService != nil {
		var err error
		assistant, err = d.assistantService.Get(ctx, stage.Auth, stage.AssistantID, utils.GetVersionDefinition("latest"),
			&internal_services.GetAssistantOption{InjectPhoneDeployment: true})
		if err != nil {
			return nil, fmt.Errorf("failed to load assistant %d: %w", stage.AssistantID, err)
		}
	}

	result := &CallSetupResult{
		AssistantID:    stage.AssistantID,
		ConversationID: conversationID,
		CallContext:    cc,
		Auth:           stage.Auth,
	}
	if assistant != nil {
		result.AssistantID = assistant.Id
		result.AssistantProviderId = assistant.AssistantProviderId
	}
	if projectContext, err := stage.Auth.ProjectContext(); err == nil {
		result.OrganizationID = projectContext.OrganizationID
		result.ProjectID = projectContext.ProjectID
	} else if organizationContext, err := stage.Auth.OrganizationContext(); err == nil {
		result.OrganizationID = organizationContext.OrganizationID
	}

	return result, nil
}

func (d *Dispatcher) createObserver(ctx context.Context, scope *CallSetupResult, auth *types.Authentication) observability.Recorder {
	return observability.New(
		observability.WithLogger(d.logger),
		observability.WithAuth(auth),
		observability.WithGlobalScope(observability.GlobalScope{
			ProjectID:      scope.ProjectID,
			OrganizationID: scope.OrganizationID,
		}),
		observability.WithContext(ctx),
		observability.WithGracePeriod(),
		observability.WithCollectors(
			observability_collector_conversationmetric.New(observability_collector_conversationmetric.Config{
				Logger:              d.logger,
				ConversationService: d.assistantConversationService,
			}),
			observability_collector_conversationmetadata.New(observability_collector_conversationmetadata.Config{
				Logger:              d.logger,
				ConversationService: d.assistantConversationService,
			}),
			observability_collector_requestlog.New(observability_collector_requestlog.Config{
				Logger:         d.logger,
				HTTPLogService: d.httpLogService,
			}),
			observability_collector_toollog.New(observability_collector_toollog.Config{
				Logger:      d.logger,
				ToolService: d.assistantToolService,
			}),
			collectors.NewWithWebhookConfiguration(ctx, d.logger, auth, scope.AssistantID, d.configurationService, d.httpLogService),
			collectors.NewWithEnv(ctx, d.logger, d.assistantConfig),
		),
	)
}
