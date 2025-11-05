package internal_assistant_service

import (
	"context"
	"fmt"
	"time"

	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type assistantWebhookService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

func NewAssistantWebhookService(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
	storage storages.Storage) internal_services.AssistantWebhookService {
	return &assistantWebhookService{
		logger:   logger,
		postgres: postgres,
		storage:  storage,
	}
}

// Get implements internal_services.AssistantWebhookService.
func (eService *assistantWebhookService) Get(ctx context.Context, auth types.SimplePrinciple, webhookId, assistantId uint64) (*internal_assistant_gorm.AssistantWebhook, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var Webhook *internal_assistant_gorm.AssistantWebhook
	tx := db.Where("id = ? AND assistant_id = ?", webhookId, assistantId).
		First(&Webhook)
	if tx.Error != nil {
		eService.logger.Benchmark("WebhookService.Get", time.Since(start))
		eService.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("WebhookService.Get", time.Since(start))
	return Webhook, nil
}

func (eService *assistantWebhookService) Create(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantEvents []string,
	timeoutSecond uint32,
	httpMethod string,
	httpUrl string,
	httpHeaders, httpBody map[string]string,
	retryStatusCodes []string,
	maxRetryCount uint32,
	executionPriority uint32,
	description *string,
) (*internal_assistant_gorm.AssistantWebhook, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	webhook := &internal_assistant_gorm.AssistantWebhook{
		AssistantId:       assistantId,
		Description:       *description,
		HttpMethod:        httpMethod,
		HttpUrl:           httpUrl,
		HttpBody:          httpBody,
		HttpHeaders:       httpHeaders,
		RetryStatusCodes:  retryStatusCodes,
		AssistantEvents:   assistantEvents,
		MaxRetryCount:     maxRetryCount,
		TimeoutSeconds:    timeoutSecond,
		ExecutionPriority: executionPriority,
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RECORD_ACTIVE,
		},
	}
	tx := db.Create(&webhook)
	if tx.Error != nil {
		eService.logger.Benchmark("eService.Create", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("eService.Create", time.Since(start))
	return webhook, nil
}

func (eService *assistantWebhookService) Update(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	webhookId uint64,
	assistantEvents []string,
	timeoutSecond uint32,
	httpMethod string,
	httpUrl string,
	httpHeaders, httpBody map[string]string,
	retryStatusCodes []string,
	maxRetryCount uint32,
	executionPriority uint32,
	description *string,
) (*internal_assistant_gorm.AssistantWebhook, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	webhook := &internal_assistant_gorm.AssistantWebhook{
		Description:       *description,
		HttpMethod:        httpMethod,
		HttpUrl:           httpUrl,
		HttpHeaders:       httpHeaders,
		HttpBody:          httpBody,
		RetryStatusCodes:  retryStatusCodes,
		AssistantEvents:   assistantEvents,
		MaxRetryCount:     maxRetryCount,
		TimeoutSeconds:    timeoutSecond,
		ExecutionPriority: executionPriority,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		webhookId,
		assistantId).Updates(&webhook)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantWebhookService.Update", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantWebhookService.Update", time.Since(start))
	return webhook, nil
}

func (eService *assistantWebhookService) Delete(ctx context.Context,
	auth types.SimplePrinciple,
	webhookId uint64,
	assistantId uint64,
) (*internal_assistant_gorm.AssistantWebhook, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	webhook := &internal_assistant_gorm.AssistantWebhook{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
			Status:    type_enums.RECORD_ARCHIEVE,
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		webhookId,
		assistantId).Updates(&webhook)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantWebhookService.Delete", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantWebhookService.Delete", time.Since(start))
	return webhook, nil
}

// GetAll implements internal_services.AssistantWebhookService.
func (eService *assistantWebhookService) GetAll(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	criterias []*lexatic_backend.Criteria,
	paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantWebhook, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		Webhooks []*internal_assistant_gorm.AssistantWebhook
		cnt      int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantWebhook{})
	qry.
		Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE)
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&Webhooks)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any Webhooks %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("WebhookService.GetAll", time.Since(start))
	return cnt, Webhooks, nil
}

func (eService *assistantWebhookService) CreateLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	webhookId uint64,
	assistantId, conversationId uint64,
	httpUrl string,
	httpMethod string,
	event string,
	responseStatus int64,
	timeTaken int64,
	retryCount uint32,
	status type_enums.RecordState,
	request, response []byte,
) (*internal_assistant_gorm.AssistantWebhookLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	s3Prefix := eService.ObjectPrefix(*auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId())
	_auditId := gorm_generator.ID()
	utils.Go(ctx, func() {
		key := eService.ObjectKey(s3Prefix, _auditId, "request.json")
		eService.storage.Store(ctx, key, request)
	})

	utils.Go(ctx, func() {
		key := eService.ObjectKey(s3Prefix, _auditId, "response.json")
		eService.storage.Store(ctx, key, response)
	})

	webhookLog := &internal_assistant_gorm.AssistantWebhookLog{
		Audited: gorm_models.Audited{
			Id: _auditId,
		},
		HttpMethod:              httpMethod,
		HttpUrl:                 httpUrl,
		AssistantId:             assistantId,
		WebhookId:               webhookId,
		AssistantConversationId: conversationId,
		AssetPrefix:             s3Prefix,
		ResponseStatus:          responseStatus,
		Event:                   event,
		TimeTaken:               timeTaken,
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Mutable: gorm_models.Mutable{
			Status: status,
		},
	}
	tx := db.Create(&webhookLog)
	if tx.Error != nil {
		eService.logger.Benchmark("eService.Create", time.Since(start))
		eService.logger.Errorf("error while creating webhook log %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("eService.Create", time.Since(start))
	return webhookLog, nil
}

func (eService *assistantWebhookService) GetLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	webhookLogId uint64) (*internal_assistant_gorm.AssistantWebhookLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var wkg *internal_assistant_gorm.AssistantWebhookLog
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ?", webhookLogId, *auth.GetCurrentOrganizationId(), projectId).
		First(&wkg)
	if tx.Error != nil {
		eService.logger.Benchmark("WebhookService.GetLog", time.Since(start))
		eService.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("WebhookService.GetLog", time.Since(start))
	return wkg, nil
}

func (eService *assistantWebhookService) GetAllLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	criterias []*lexatic_backend.Criteria,
	paginate *lexatic_backend.Paginate,
	order *lexatic_backend.Ordering) (int64, []*internal_assistant_gorm.AssistantWebhookLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		webhookLogs []*internal_assistant_gorm.AssistantWebhookLog
		cnt         int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantWebhookLog{})
	qry.
		Where("organization_id = ? AND project_id = ? ", *auth.GetCurrentOrganizationId(), projectId)
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&webhookLogs)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any Webhooks %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("WebhookService.GetAllLog", time.Since(start))
	return cnt, webhookLogs, nil
}

func (eService *assistantWebhookService) ObjectPrefix(orgId, projectId uint64) string {
	return fmt.Sprintf("%d/%d/webhook", orgId, projectId)
}

func (eService *assistantWebhookService) ObjectKey(keyPrefix string, auditId uint64, objName string) string {
	return fmt.Sprintf("%s/%d__%s", keyPrefix, auditId, objName)
}

func (eService *assistantWebhookService) GetLogObject(
	ctx context.Context,
	organizationId,
	projectId, webhookLogId uint64) (requestData []byte, responseData []byte, err error) {
	keyPrefix := eService.ObjectPrefix(organizationId, projectId)
	responseKey := eService.ObjectKey(keyPrefix, webhookLogId, "response.json")
	requestKey := eService.ObjectKey(keyPrefix, webhookLogId, "request.json")

	type _fileStruct struct {
		Key   string
		Data  []byte
		Error error
	}

	responseChan := make(chan _fileStruct)
	requestChan := make(chan _fileStruct)

	go func(key string) {
		eService.logger.Debugf("Getting key from s3 %s", key)
		result := eService.storage.Get(ctx, key)
		if result.Error != nil {
			eService.logger.Errorf("error downloading goroutine: %v", result.Error)
			responseChan <- _fileStruct{Key: key, Error: result.Error}
			close(responseChan)
			return
		}
		responseChan <- _fileStruct{Key: key, Data: result.Data}
		close(responseChan)
	}(responseKey)

	go func(key string) {
		eService.logger.Debugf("Getting key from s3 %s", key)
		result := eService.storage.Get(ctx, key)
		if result.Error != nil {
			eService.logger.Errorf("error downloading goroutine: %v", result.Error)
			requestChan <- _fileStruct{Key: key, Error: result.Error}
			close(requestChan)
			return
		}
		requestChan <- _fileStruct{Key: key, Data: result.Data}
		close(requestChan)

	}(requestKey)

	for result := range responseChan {
		if result.Error != nil {
			eService.logger.Errorf("error downloading/parsing response: %v", result.Error)
			break
		}
		responseData = result.Data
	}

	for result := range requestChan {
		if result.Error != nil {
			eService.logger.Errorf("error downloading/parsing request: %v", result.Error)
			break
		}
		requestData = result.Data
	}

	return requestData, responseData, nil
}
