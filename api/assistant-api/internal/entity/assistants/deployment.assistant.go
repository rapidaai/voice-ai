package internal_assistant_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

/**
*
 */
type AssistantDeploymentTelephony struct {
	TelephonyProvider string                                `json:"phoneProviderName" gorm:"type:string;size:50;not null;"`
	TelephonyOption   []*AssistantDeploymentTelephonyOption `json:"phoneOptions"  gorm:"foreignKey:AssistantDeploymentTelephonyId"`
}

func (a *AssistantDeploymentTelephony) GetOptions() utils.Option {
	opts := make(map[string]interface{})
	for _, v := range a.TelephonyOption {
		opts[v.Key] = v.Value
	}
	return opts
}

type AssistantDeploymentTelephonyOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantDeploymentTelephonyId uint64 `json:"assistantDeploymentTelephonyId" gorm:"type:bigint;size:20"`
}

type AssistantDeploymentWhatsapp struct {
	WhatsappProvider string                               `json:"whatsappProviderName" gorm:"type:string;size:50;not null;"`
	WhatsappOptions  []*AssistantDeploymentWhatsappOption `json:"whatsappOptions"  gorm:"foreignKey:AssistantDeploymentWhatsappId"`
}

type AssistantDeploymentWhatsappOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantDeploymentWhatsappId uint64 `json:"assistantDeploymentWhatsappId" gorm:"type:bigint;size:20"`
}

// input audio later
type AssistantDeploymentAudio struct {
	gorm_model.Audited
	gorm_model.Mutable
	AssistantDeploymentId uint64                            `json:"assistantDeploymentId"`
	AudioType             string                            `json:"audioType" gorm:"type:string;size:50;not null;index:idx_audio_type"`
	AudioProvider         string                            `json:"audioProvider" gorm:"type:string;size:50;not null;"`
	AudioOptions          []*AssistantDeploymentAudioOption `json:"audioOptions"  gorm:"foreignKey:AssistantDeploymentAudioId"`
}

func (a *AssistantDeploymentAudio) GetName() string {
	return a.AudioProvider
}

func (a *AssistantDeploymentAudio) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	if a.AudioOptions != nil {
		for _, v := range a.AudioOptions {
			opts[v.Key] = v.Value
		}
	}
	return opts
}

type AssistantDeploymentAudioOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantDeploymentAudioId uint64 `json:"assistantDeploymentAudioId" gorm:"type:bigint;size:20"`
}

type AssistantDeployment struct {
	gorm_model.Audited
	gorm_model.Mutable
	AssistantId uint64 `json:"assistantId" gorm:"type:bigint;size:20"`
}

type AssistantDeploymentBehavior struct {
	AssistantDeployment
	Greeting *string `json:"greeting" gorm:"type:string;size:50;not null;"`
	Mistake  *string `json:"mistake" gorm:"type:string;size:50;not null;"`

	IdealTimeout        *uint64 `json:"IdealTimeout"`
	IdealTimeoutMessage *string `json:"idealTimeoutMessage" gorm:"type:string;size:50;not null;"`
	MaxSessionDuration  *uint64 `json:"maxSessionDuration"`
}

type AssistantWebPluginDeployment struct {
	AssistantDeploymentBehavior

	Name       string                 `json:"name" gorm:"type:string;size:50;not null;"`
	Suggestion gorm_types.StringArray `json:"suggestion" gorm:"column:suggestions;type:string"`

	//
	HelpCenterEnabled     bool `json:"helpCenterEnabled" gorm:"type:bool"`
	ProductCatalogEnabled bool `json:"productCatalogEnabled" gorm:"type:bool"`
	ArticleCatalogEnabled bool `json:"articleCatalogEnabled" gorm:"type:bool"`

	InputAudio *AssistantDeploymentAudio `json:"inputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
	OuputAudio *AssistantDeploymentAudio `json:"outputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
}

type AssistantPhoneDeployment struct {
	AssistantDeploymentBehavior
	AssistantDeploymentTelephony
	InputAudio *AssistantDeploymentAudio `json:"inputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
	OuputAudio *AssistantDeploymentAudio `json:"outputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
}

type AssistantWhatsappDeployment struct {
	AssistantDeploymentBehavior
	AssistantDeploymentWhatsapp
}

/**
 */
type AssistantApiDeployment struct {
	AssistantDeploymentBehavior
	InputAudio *AssistantDeploymentAudio `json:"inputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
	OuputAudio *AssistantDeploymentAudio `json:"outputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
}

/**
 */
type AssistantDebuggerDeployment struct {
	AssistantDeploymentBehavior

	Icon       string                    `json:"icon" gorm:"type:string;size:50;not null;"`
	InputAudio *AssistantDeploymentAudio `json:"inputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
	OuputAudio *AssistantDeploymentAudio `json:"outputAudio"  gorm:"foreignKey:AssistantDeploymentId"`
}
