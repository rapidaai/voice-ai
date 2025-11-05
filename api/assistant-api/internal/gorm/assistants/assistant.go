package internal_assistant_gorm

import (
	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/conversations"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type Assistant struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	Name        string `json:"name" gorm:"type:string"`
	Description string `json:"description" gorm:"type:string"`
	Visibility  string `json:"visibility" gorm:"type:string;default:private"`

	//
	Source           string  `json:"_" gorm:"type:string;size:50"`
	SourceIdentifier *uint64 `json:"_" gorm:"type:bigint;size:20"`

	//
	Language            string                       `json:"language" gorm:"type:string;size:50;default:english"`
	AssistantProvider   type_enums.AssistantProvider `json:"assistantProvider" gorm:"type:string;size:50;not null;default:MODEL"`
	AssistantProviderId uint64                       `json:"assistantProviderId" gorm:"type:bigint;size:20"`

	AssistantProviderModel     *AssistantProviderModel     `json:"assistantProviderModel" gorm:"foreignKey:AssistantProviderId"`
	AssistantProviderAgentkit  *AssistantProviderAgentkit  `json:"assistantProviderAgentkit" gorm:"foreignKey:AssistantProviderId"`
	AssistantProviderWebsocket *AssistantProviderWebsocket `json:"assistantProviderWebsocket" gorm:"foreignKey:AssistantProviderId"`

	AssistantTag *AssistantTag `json:"assistantTag" gorm:"foreignKey:AssistantId"`

	// all the deployments only on need basis
	AssistantDebuggerDeployment  *AssistantDebuggerDeployment                        `json:"debuggerDeployment"  gorm:"foreignKey:AssistantId"`
	AssistantPhoneDeployment     *AssistantPhoneDeployment                           `json:"phoneDeployment"  gorm:"foreignKey:AssistantId"`
	AssistantWhatsappDeployment  *AssistantWhatsappDeployment                        `json:"whatsappDeployment"  gorm:"foreignKey:AssistantId"`
	AssistantWebPluginDeployment *AssistantWebPluginDeployment                       `json:"webPluginDeployment"  gorm:"foreignKey:AssistantId"`
	AssistantApiDeployment       *AssistantApiDeployment                             `json:"apiDeployment"  gorm:"foreignKey:AssistantId"`
	AssistantConversations       []*internal_conversation_gorm.AssistantConversation `json:"assistantConversations"  gorm:"foreignKey:AssistantId"`
	AssistantKnowledges          []*AssistantKnowledge                               `json:"assistantKnowledges"  gorm:"foreignKey:AssistantId"`
	AssistantTools               []*AssistantTool                                    `json:"assistantTools"  gorm:"foreignKey:AssistantId"`
	AssistantAnalyses            []*AssistantAnalysis                                `json:"assistantAnalyses"  gorm:"foreignKey:AssistantId"`
	AssistantWebhooks            []*AssistantWebhook                                 `json:"assistantWebhooks"  gorm:"foreignKey:AssistantId"`
}

func (a *Assistant) IsPhoneDeploymentEnable() bool {
	return a.AssistantPhoneDeployment != nil
}

func (a *Assistant) IsAssistantApiDeploymentEnable() bool {
	return a.AssistantApiDeployment != nil
}

func (a *Assistant) IsWebPluginDeploymentEnable() bool {
	return a.AssistantWhatsappDeployment != nil
}

// AssistantTag represents a tag associated with an assistant in the database.
// It extends the Audited model and includes fields for the assistant ID,
// the tag itself (as a string array), and information about who created and updated the tag.
type AssistantTag struct {
	gorm_model.Audited
	AssistantId uint64                 `json:"assistantId" gorm:"type:bigint;not null"`
	Tag         gorm_types.StringArray `json:"tag" gorm:"type:string;size:200;not null"`
	CreatedBy   uint64                 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy   uint64                 `json:"updatedBy" gorm:"type:bigint;size:20;"`
}
