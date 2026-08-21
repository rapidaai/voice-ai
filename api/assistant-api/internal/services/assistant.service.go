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
	type_enums "github.com/rapidaai/pkg/types/enums"
	workflow_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GetAssistantOption struct {
	InjectTag                    bool
	InjectAssistantProvider      bool
	InjectKnowledgeConfiguration bool
	InjectWebpluginDeployment    bool
	InjectApiDeployment          bool
	InjectDebuggerDeployment     bool
	InjectPhoneDeployment        bool
	InjectWhatsappDeployment     bool
	InjectTool                   bool

	InjectAnalysis       bool
	InjectAuthentication bool
	InjectStorage        bool
}

func NewDefaultGetAssistantOption() *GetAssistantOption {
	return &GetAssistantOption{
		InjectTag:                    true,
		InjectAssistantProvider:      true,
		InjectKnowledgeConfiguration: true,
		InjectWebpluginDeployment:    true,
		InjectApiDeployment:          true,
		InjectDebuggerDeployment:     true,
		InjectPhoneDeployment:        true,
		InjectWhatsappDeployment:     true,
		InjectTool:                   true,
		InjectAuthentication:         true,
		InjectStorage:                true,
	}
}

type AssistantService interface {
	Get(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantProviderId *uint64,
		opts *GetAssistantOption) (*internal_assistant_entity.Assistant, error)

	GetAll(ctx context.Context,
		auth *types.Authentication,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		opts *GetAssistantOption) (int64, []*internal_assistant_entity.Assistant, error)

	GetAssistantDashboard(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		fromDate *timestamppb.Timestamp,
		toDate *timestamppb.Timestamp,
	) (*workflow_api.AssistantDashboard, error)

	GetAllAssistantProviderModel(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantProviderModel, error)

	GetAllAssistantProviderWebsocket(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantProviderWebsocket, error)
	GetAllAssistantProviderAgentkit(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantProviderAgentkit, error)
	GetAllAssistantProviderAgentflow(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_entity.AssistantProviderAgentflow, error)

	UpdateAssistantVersion(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantProvider type_enums.AssistantProvider,
		assistantProviderId uint64,
	) (*internal_assistant_entity.Assistant, error)

	UpdateAssistantDetail(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		name, description string) (*internal_assistant_entity.Assistant, error)

	CreateAssistant(ctx context.Context,
		auth *types.Authentication,
		name, description string,
		visibility string, source string, sourceIdentifier *uint64,
		language string,
	) (*internal_assistant_entity.Assistant, error)

	DeleteAssistant(ctx context.Context, auth *types.Authentication, assistantId uint64) (*internal_assistant_entity.Assistant, error)

	CreateAssistantProviderModel(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		description string,
		template string,
		providerModelName string,
		modelProperties []*workflow_api.Metadata,
	) (*internal_assistant_entity.AssistantProviderModel, error)

	CreateAssistantProviderWebsocket(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		description string,
		url string,
		headers map[string]string,
		parameters map[string]string,
	) (*internal_assistant_entity.AssistantProviderWebsocket, error)

	CreateAssistantProviderAgentkit(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		description string,
		url string,
		certificate string,
		metadata map[string]string,
		transportSecurity *string,
		tlsVerification *string,
		tlsServerName *string,
		connectTimeoutMs *uint32,
		keepaliveTimeMs *uint32,
		keepaliveTimeoutMs *uint32,
		maxRecvMessageBytes *uint32,
		maxSendMessageBytes *uint32,
	) (*internal_assistant_entity.AssistantProviderAgentkit, error)

	CreateAssistantProviderAgentflow(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		description string,
		schemaVersion string,
		definition map[string]interface{},
	) (*internal_assistant_entity.AssistantProviderAgentflow, error)

	AttachProviderModelToAssistant(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantProvider type_enums.AssistantProvider,
		assistantProviderId uint64,
	) (*internal_assistant_entity.Assistant, error)

	//
	CreateOrUpdateAssistantTag(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		tags []string,
	) (*internal_assistant_entity.AssistantTag, error)
}
