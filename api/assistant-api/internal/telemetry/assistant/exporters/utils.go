package internal_assistant_telemetry_exporters

import (
	"time"

	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type AssistantConversationPostgresTelemetry struct {
	gorm_model.Organizational

	AssistantId              uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantProviderModelId uint64 `json:"assistantProviderModelId" gorm:"type:bigint;not null"`
	AssistantConversationId  uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`

	StageName  string                 `json:"stageName" gorm:"type:string;not null"`
	StartTime  gorm_model.TimeWrapper `json:"startTime" gorm:"type:timestamp;not null;"`
	EndTime    gorm_model.TimeWrapper `json:"endTime" gorm:"type:timestamp;not null;"`
	Duration   uint64                 `json:"duration"`
	Attributes gorm_types.StringMap   `json:"attributes" gorm:"type:string;not null"`
	SpanID     string                 `json:"spanID" gorm:"type:string;not null"`
	ParentID   string                 `json:"parentID" gorm:"type:string;not null"`
}
type AssistantConversationOpensearchTelemetry struct {
	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;not null"`

	AssistantId              uint64 `json:"assistantId" `
	AssistantProviderModelId uint64 `json:"assistantProviderModelId" `
	AssistantConversationId  uint64 `json:"assistantConversationId" `

	StageName  string            `json:"stageName"`
	StartTime  string            `json:"startTime"` // ISO 8601 format (e.g., "2023-10-12T10:00:00Z")
	EndTime    string            `json:"endTime"`   // ISO 8601 format
	Duration   uint64            `json:"duration"`
	Attributes map[string]string `json:"attributes"`
	SpanID     string            `json:"spanID"`
	ParentID   string            `json:"parentID"`
}

func ToOpensearchTelemetry(
	stg *internal_adapter_telemetry.Telemetry,
	assistantId, assistantProviderModelId, assistantConversactionId uint64,
	projectId, organizationId uint64,
) *AssistantConversationOpensearchTelemetry {
	epm := &AssistantConversationOpensearchTelemetry{
		ProjectId:                projectId,
		OrganizationId:           organizationId,
		AssistantId:              assistantId,
		AssistantProviderModelId: assistantProviderModelId,
		AssistantConversationId:  assistantConversactionId,
	}
	epm.StageName = stg.StageName
	epm.StartTime = stg.StartTime.Format(time.RFC3339Nano)
	epm.EndTime = stg.EndTime.Format(time.RFC3339Nano)
	epm.Duration = uint64(stg.Duration)
	epm.Attributes = gorm_types.StringMap(stg.Attributes)
	epm.SpanID = stg.SpanID
	epm.ParentID = stg.ParentID
	return epm
}

func ToPostgresTelemetry(
	stg *internal_adapter_telemetry.Telemetry,
	assistantId, assistantProviderModelId, assistantConversactionId uint64,
	projectId, organizationId uint64,
) *AssistantConversationPostgresTelemetry {
	epm := &AssistantConversationPostgresTelemetry{
		Organizational: gorm_model.Organizational{
			ProjectId:      projectId,
			OrganizationId: organizationId,
		},
		AssistantId:              assistantId,
		AssistantProviderModelId: assistantProviderModelId,
		AssistantConversationId:  assistantConversactionId,
	}
	epm.StageName = stg.StageName
	epm.StartTime = gorm_model.TimeWrapper(stg.StartTime)
	epm.EndTime = gorm_model.TimeWrapper(stg.StartTime)
	epm.Duration = uint64(stg.Duration)
	epm.Attributes = gorm_types.StringMap(stg.Attributes)
	epm.SpanID = stg.SpanID
	epm.ParentID = stg.ParentID
	return epm
}
