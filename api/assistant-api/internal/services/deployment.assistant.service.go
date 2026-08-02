// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	workflow_api "github.com/rapidaai/protos"
)

type AssistantDeploymentService interface {
	CreateWhatsappDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		unclearInputTimeout *float64, unclearInputMessage *string,
		greetingInterruptible *bool,
		idealTimeout *uint64, idealTimeoutBackoff *uint64, idealTimeoutMessage *string, maxSessionDuration *uint64,
		whatsappProvider string,
		opts []*workflow_api.Metadata,
	) (*internal_assistant_entity.AssistantWhatsappDeployment, error)

	CreatePhoneDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		unclearInputTimeout *float64, unclearInputMessage *string,
		greetingInterruptible *bool,
		idealTimeout *uint64, idealTimeoutBackoff *uint64, idealTimeoutMessage *string, maxSessionDuration *uint64,
		phoneProvider string,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
		opts []*workflow_api.Metadata,
	) (*internal_assistant_entity.AssistantPhoneDeployment, error)

	CreateApiDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		unclearInputTimeout *float64, unclearInputMessage *string,
		greetingInterruptible *bool,
		idealTimeout *uint64, idealTimeoutBackoff *uint64, idealTimeoutMessage *string, maxSessionDuration *uint64,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_entity.AssistantApiDeployment, error)

	CreateDebuggerDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		unclearInputTimeout *float64, unclearInputMessage *string,
		greetingInterruptible *bool,
		idealTimeout *uint64, idealTimeoutBackoff *uint64, idealTimeoutMessage *string, maxSessionDuration *uint64,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_entity.AssistantDebuggerDeployment, error)

	CreateWebPluginDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		unclearInputTimeout *float64, unclearInputMessage *string,
		greetingInterruptible *bool,
		idealTimeout *uint64, idealTimeoutBackoff *uint64, idealTimeoutMessage *string, maxSessionDuration *uint64,
		suggestion []string,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_entity.AssistantWebPluginDeployment, error)

	GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error)
	GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error)
	GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error)
	GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error)
	GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error)

	GetAllAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantApiDeployment, error)
	GetAllAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantDebuggerDeployment, error)
	GetAllAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantPhoneDeployment, error)
	GetAllAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantWebPluginDeployment, error)
	GetAllAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantWhatsappDeployment, error)

	DisableAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantApiDeployment, error)
	DisableAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error)
	DisableAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error)
	DisableAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error)
	DisableAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error)
}
