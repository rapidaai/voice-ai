// The `package internal_services` in this code snippet defines several interfaces for different
// services related to workflows, assistants, knowledge, and knowledge documents. These interfaces
// specify the methods that need to be implemented by concrete types that provide these services.
package internal_services

import (
	"context"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	workflow_api "github.com/rapidaai/protos"
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
	//
	InjectConversations bool

	InjectAnalysis bool
	InjectWebhook  bool
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
		InjectConversations:          true,
	}
}

type AssistantService interface {
	Get(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantProviderId *uint64,
		opts *GetAssistantOption) (*internal_assistant_gorm.Assistant, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		opts *GetAssistantOption) (int64, []*internal_assistant_gorm.Assistant, error)

	GetAllAssistantProviderModel(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderModel, error)

	GetAllAssistantProviderWebsocket(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderWebsocket, error)
	GetAllAssistantProviderAgentkit(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64, criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderAgentkit, error)

	UpdateAssistantVersion(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantProvider type_enums.AssistantProvider,
		assistantProviderId uint64,
	) (*internal_assistant_gorm.Assistant, error)

	UpdateAssistantDetail(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name, description string) (*internal_assistant_gorm.Assistant, error)

	CreateAssistant(ctx context.Context,
		auth types.SimplePrinciple,
		name, description string,
		visibility string, source string, sourceIdentifier *uint64,
		language string,
	) (*internal_assistant_gorm.Assistant, error)

	DeleteAssistant(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.Assistant, error)

	CreateAssistantProviderModel(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		description string,
		template string,
		providerId uint64,
		providerModelName string,
		modelProperties []*workflow_api.Metadata,
	) (*internal_assistant_gorm.AssistantProviderModel, error)

	CreateAssistantProviderWebsocket(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		description string,
		url string,
		headers map[string]string,
		parameters map[string]string,
	) (*internal_assistant_gorm.AssistantProviderWebsocket, error)

	CreateAssistantProviderAgentkit(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		description string,
		url string,
		certificate string,
		metadata map[string]string,
	) (*internal_assistant_gorm.AssistantProviderAgentkit, error)

	AttachProviderModelToAssistant(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantProvider type_enums.AssistantProvider,
		assistantProviderId uint64,
	) (*internal_assistant_gorm.Assistant, error)

	//
	CreateOrUpdateAssistantTag(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		tags []string,
	) (*internal_assistant_gorm.AssistantTag, error)
}

type AssistantDeploymentService interface {
	CreateWhatsappDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		whatsappProviderId uint64, whatsappProvider string,
		opts []*workflow_api.Metadata,
	) (*internal_assistant_gorm.AssistantWhatsappDeployment, error)

	CreatePhoneDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		phoneProviderId uint64, phoneProvider string,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
		opts []*workflow_api.Metadata,
	) (*internal_assistant_gorm.AssistantPhoneDeployment, error)

	CreateApiDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		greeting, mistake *string,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_gorm.AssistantApiDeployment, error)

	CreateDebuggerDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name, icon string,
		greeting, mistake *string,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_gorm.AssistantDebuggerDeployment, error)

	CreateWebPluginDeployment(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name, icon string,
		greeting, mistake *string,
		suggestion []string,
		helpCenterEnabled, productCatalogEnabled, articleCatalogEnabled bool,
		inputAudio, outputAudio *workflow_api.DeploymentAudioProvider,
	) (*internal_assistant_gorm.AssistantWebPluginDeployment, error)

	GetAssistantApiDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantApiDeployment, error)
	GetAssistantDebuggerDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantDebuggerDeployment, error)
	GetAssistantPhoneDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantPhoneDeployment, error)
	GetAssistantWebpluginDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantWebPluginDeployment, error)
	GetAssistantWhatsappDeployment(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.AssistantWhatsappDeployment, error)
}
