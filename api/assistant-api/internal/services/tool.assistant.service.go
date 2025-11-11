package internal_services

import (
	"context"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	lexatic_backend "github.com/rapidaai/protos"
	workflow_api "github.com/rapidaai/protos"
)

type AssistantToolService interface {
	Get(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantToolId uint64,
		assistantId uint64) (*internal_assistant_gorm.AssistantTool, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*workflow_api.Criteria, paginate *workflow_api.Paginate) (int64, []*internal_assistant_gorm.AssistantTool, error)

	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		name string,
		description string,
		fields map[string]interface{},
		executionMethod string,
		executionOptions []*lexatic_backend.Metadata,
	) (*internal_assistant_gorm.AssistantTool, error)

	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantToolId uint64,
		assistantId uint64,
		name string,
		description string,
		fields map[string]interface{},
		executionMethod string,
		executionOptions []*lexatic_backend.Metadata,
	) (*internal_assistant_gorm.AssistantTool, error)

	Delete(ctx context.Context,
		auth types.SimplePrinciple,
		toolId uint64,
		assistantId uint64) (*internal_assistant_gorm.AssistantTool, error)

	CreateLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId, conversationId uint64,
		toolId uint64,
		messageId string,
		toolName string,
		timeTaken int64,
		executionMethod string,
		status type_enums.RecordState,
		request, response []byte,
	) (*internal_assistant_gorm.AssistantToolLog, error)

	GetLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		toolLogId uint64) (*internal_assistant_gorm.AssistantToolLog, error)

	GetAllLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		criterias []*lexatic_backend.Criteria,
		paginate *lexatic_backend.Paginate,
		order *lexatic_backend.Ordering) (int64, []*internal_assistant_gorm.AssistantToolLog, error)

	GetLogObject(
		ctx context.Context,
		organizationId,
		projectId, toolLogId uint64) (requestData []byte, responseData []byte, err error)
}
