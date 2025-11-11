package internal_services

import (
	"context"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	lexatic_backend "github.com/rapidaai/protos"
)

type AssistantWebhookService interface {
	Get(ctx context.Context, auth types.SimplePrinciple, WebhookId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantWebhook, error)
	Delete(ctx context.Context, auth types.SimplePrinciple, WebhookId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantWebhook, error)
	Create(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantEvents []string,
		timeoutSecond uint32,
		httpMethod string,
		httpUrl string,
		httpHeaders, httpBody map[string]string,
		retryStatusCodes []string,
		retryCount, executionPriority uint32,
		description *string,
	) (*internal_assistant_gorm.AssistantWebhook, error)
	Update(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		webhookId uint64,
		assistantEvents []string,
		timeoutSecond uint32,
		httpMethod string,
		httpUrl string,
		httpHeaders, httpBody map[string]string,
		retryStatusCodes []string,
		maxRetryCount, executionPriority uint32,
		description *string,
	) (*internal_assistant_gorm.AssistantWebhook, error)

	GetAll(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*lexatic_backend.Criteria,
		paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantWebhook, error)

	CreateLog(
		ctx context.Context,
		auth types.SimplePrinciple,
		webhookId uint64,
		assistantId, conversationId uint64,
		httpUrl, httpMethod string,
		event string,
		responseStatus int64,
		timeTaken int64,
		retryCount uint32,
		status type_enums.RecordState,
		request, response []byte,
	) (*internal_assistant_gorm.AssistantWebhookLog, error)

	GetAllLog(ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		criterias []*lexatic_backend.Criteria,
		paginate *lexatic_backend.Paginate,
		order *lexatic_backend.Ordering,
	) (int64, []*internal_assistant_gorm.AssistantWebhookLog, error)

	GetLog(ctx context.Context,
		auth types.SimplePrinciple,
		projectId uint64,
		webhookLogId uint64) (*internal_assistant_gorm.AssistantWebhookLog, error)
	GetLogObject(
		ctx context.Context,
		organizationId,
		projectId, webhookLogId uint64) (requestData []byte, responseData []byte, err error)
}
