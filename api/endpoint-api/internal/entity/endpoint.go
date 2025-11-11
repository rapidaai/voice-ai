package internal_entity

import (
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

type Endpoint struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational

	Name                    string  `json:"name" gorm:"type:string"`
	Description             *string `json:"description" gorm:"type:string"`
	EndpointProviderModelId uint64  `json:"endpointProviderModelId" gorm:"type:bigint;size:20"`

	Visibility *string `json:"visibility" gorm:"type:string;size:50;not null;default:private"`

	Source           *string `json:"source" gorm:"type:string;size:50"`
	SourceIdentifier *uint64 `json:"sourceIdentifier" gorm:"type:bigint;size:20"`
	CacheEnable      bool    `json:"cacheEnable" gorm:"type:boolean;default:false"`
	RetryEnable      bool    `json:"retryEnable" gorm:"type:boolean;default:false"`

	EndpointProviderModel *EndpointProviderModel `json:"endpointProviderModel" gorm:"foreignKey:EndpointProviderModelId"`
	EndpointRetry         *EndpointRetry         `json:"endpointRetry" gorm:"foreignKey:EndpointId"`
	EndpointCaching       *EndpointCaching       `json:"endpointCaching" gorm:"foreignKey:EndpointId"`
	EndpointTag           *EndpointTag           `json:"endpointTag" gorm:"foreignKey:EndpointId"`
}

// this table will immutatble used as version
type EndpointProviderModel struct {
	gorm_model.Audited
	gorm_model.Mutable
	EndpointId  uint64               `json:"endpointId" gorm:"type:bigint;size:20"`
	Description string               `json:"description" gorm:"type:string"`
	Request     gorm_types.PromptMap `json:"chatCompletePrompt" gorm:"type:jsonb"`

	ModelProviderName            string                         `json:"modelProviderName" gorm:"type:string"`
	ModelProviderId              uint64                         `json:"modelProviderId" gorm:"type:bigint;size:20;not null"`
	EndpointProviderModelOptions []*EndpointProviderModelOption `json:"endpointModelOptions" gorm:"foreignKey:EndpointProviderModelId"`
}

type EndpointProviderModelOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	EndpointProviderModelId uint64 `json:"EndpointProviderModelId" gorm:"type:bigint;size:20"`
}

func (a *EndpointProviderModel) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	for _, v := range a.EndpointProviderModelOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

func (epm *EndpointProviderModel) SetPrompt(promptString string) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(promptString), &jsonData)
	if err != nil {
		return
	}
	epm.Request = jsonData
}

type EndpointTag struct {
	gorm_model.Audited
	gorm_model.Mutable
	EndpointId uint64                 `json:"endpointId" gorm:"type:bigint;not null"`
	Tag        gorm_types.StringArray `json:"tag" gorm:"type:string;size:200;not null"`
}
