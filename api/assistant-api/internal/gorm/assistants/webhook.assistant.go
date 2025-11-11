package internal_assistant_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type AssistantWebhook struct {
	gorm_model.Audited
	gorm_model.Mutable

	AssistantId     uint64                 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantEvents gorm_types.StringArray `json:"assistantEvents" gorm:"type:string;not null;"`
	Description     string                 `json:"description" gorm:"type:text"`

	HttpMethod  string               `json:"httpMethod" gorm:"type:text"`
	HttpUrl     string               `json:"httpUrl" gorm:"type:text"`
	HttpHeaders gorm_types.StringMap `json:"httpHeaders" gorm:"type:string;"`
	HttpBody    gorm_types.StringMap `json:"httpBody" gorm:"type:string;"`

	//
	RetryStatusCodes  gorm_types.StringArray `json:"retryStatusCodes" gorm:"type:string;not null;"`
	MaxRetryCount     uint32                 `json:"maxRetryCount" gorm:"type:int"`
	TimeoutSeconds    uint32                 `json:"timeoutSecond" gorm:"type:int"`
	ExecutionPriority uint32                 `json:"executionPriority" gorm:"type:int"`
}

func (aa *AssistantWebhook) GetExecutionPriority() uint32 {
	return aa.ExecutionPriority
}

func (aa *AssistantWebhook) GetHeaders() map[string]string {
	return aa.HttpHeaders
}

func (aa *AssistantWebhook) GetBody() map[string]string {
	return aa.HttpBody
}

func (aa *AssistantWebhook) GetMethod() string {
	return aa.HttpMethod
}

func (aa *AssistantWebhook) GetUrl() string {
	return aa.HttpUrl
}

func (aa *AssistantWebhook) GetRetryStatusCode() []string {
	return aa.RetryStatusCodes
}

func (aa *AssistantWebhook) GetMaxRetryCount() uint32 {
	return aa.MaxRetryCount
}

func (aa *AssistantWebhook) GetTimeoutSecond() uint32 {
	return aa.TimeoutSeconds
}

type AssistantWebhookLog struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	WebhookId               uint64 `json:"webhookId" gorm:"type:bigint"`
	HttpMethod              string `json:"httpMethod" gorm:"type:string;size:200;not null"`
	HttpUrl                 string `json:"httpUrl" gorm:"type:string;size:400;not null"`
	AssistantId             uint64 `json:"assistantId" gorm:"type:bigint"`
	AssistantConversationId uint64 `json:"assistantConversationId" gorm:"type:bigint"`
	Event                   string `json:"event" gorm:"type:string;size:200;not null"`
	AssetPrefix             string `json:"assetPrefix" gorm:"type:string;size:200;not null"`
	ResponseStatus          int64  `json:"responseStatus" gorm:"type:bigint;size:10"`
	TimeTaken               int64  `json:"timeTaken" gorm:"type:bigint;size:20"`
	RetryCount              uint32 `json:"retryCount" gorm:"type:bigint;size:20"`
}

// ALTER TABLE assistant_webhook_logs rename column assistantConversationId to assistant_conversation_id;
// CREATE TABLE assistant_webhook_logs (
//     id BIGINT PRIMARY KEY,
//     created_date TIMESTAMP NOT NULL DEFAULT NOW(),
//     updated_date TIMESTAMP,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT NOT NULL,
//     updated_by BIGINT,
//     webhook_id BIGINT NOT NULL,
//     project_id BIGINT NOT NULL,
//     organization_id BIGINT NOT NULL,
//     assistant_id BIGINT,
//     conversation_id BIGINT,
//     http_method VARCHAR(200) NOT NULL,
//     http_url VARCHAR(400) NOT NULL,
//     asset_prefix VARCHAR(200) NOT NULL,
//     event VARCHAR(200) NOT NULL,
//     response_status BIGINT,
//     time_taken BIGINT,
//     retry_count BIGINT
// );
