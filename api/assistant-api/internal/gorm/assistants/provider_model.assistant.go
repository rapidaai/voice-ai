package internal_assistant_gorm

import (
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

type AssistantProvider struct {
	gorm_model.Audited
	AssistantId uint64 `json:"assistantId" gorm:"type:bigint;size:20"`
	CreatedBy   uint64 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	Status      string `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
	Description string `json:"description" gorm:"type:string"`
}

// CREATE TABLE assistant_provider_agentkits (
// 	    id bigint NOT NULL,
// 	    created_date timestamp without time zone NOT NULL DEFAULT now(),
// 	    updated_date timestamp without time zone DEFAULT NULL,
// 	    status character varying(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
// 	    created_by bigint NOT NULL,
// 	    assistant_id bigint NOT NULL,
// 		description character varying(400) NOT NULL,
// 		url character varying(200) NOT NULL,
// 		certificate TEXT DEFAULT NULL,
// 		metadata TEXT DEFAULT NULL,
// 	    CONSTRAINT assistant_provider_agentkits_pkey PRIMARY KEY (id)
// );

// CREATE TABLE assistant_provider_websockets (
//
//	    id bigint NOT NULL,
//	    created_date timestamp without time zone NOT NULL DEFAULT now(),
//	    updated_date timestamp without time zone DEFAULT NULL,
//	    status character varying(50) NOT NULL DEFAULT 'ACTIVE'::character varying,
//	    created_by bigint NOT NULL,
//	    assistant_id bigint NOT NULL,
//		description character varying(400) NOT NULL,
//		url character varying(200) NOT NULL,
//		headers TEXT DEFAULT NULL,
//		parameters TEXT DEFAULT NULL,
//	    CONSTRAINT assistant_provider_websockets_pkey PRIMARY KEY (id)
//
// );

type AssistantProviderAgentkit struct {
	AssistantProvider
	//
	Url         string               `json:"url" gorm:"type:string"`
	Certificate string               `json:"certificate" gorm:"type:string;size:400;not null;"`
	Metadata    gorm_types.StringMap `json:"metadata" gorm:"type:string;size:400;not null;"`
}

type AssistantProviderWebsocket struct {
	AssistantProvider

	//
	Url        string               `json:"url" gorm:"type:string"`
	Headers    gorm_types.StringMap `json:"headers" gorm:"type:string;size:400;not null;"`
	Parameters gorm_types.StringMap `json:"parameters" gorm:"type:string;size:400;not null;"`
}

type AssistantProviderModel struct {
	AssistantProvider
	//
	Template              gorm_types.PromptMap            `json:"template" gorm:"type:jsonb"`
	AssistantId           uint64                          `json:"assistantId" gorm:"type:bigint;size:20"`
	ModelProviderName     string                          `json:"modelProviderName" gorm:"type:string"`
	ModelProviderId       uint64                          `json:"modelProviderId" gorm:"type:bigint;size:20;not null"`
	AssistantModelOptions []*AssistantProviderModelOption `json:"assistantModelOptions" gorm:"foreignKey:AssistantProviderModelId"`
}

type AssistantProviderModelOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantProviderModelId uint64 `json:"AssistantProviderModelId" gorm:"type:bigint;size:20"`
}

func (a *AssistantProviderModel) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	for _, v := range a.AssistantModelOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

func (epm *AssistantProviderModel) SetPrompt(promptString string) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(promptString), &jsonData)
	if err != nil {
		return
	}
	epm.Template = jsonData
}
