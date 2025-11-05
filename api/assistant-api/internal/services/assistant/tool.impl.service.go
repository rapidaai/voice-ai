package internal_assistant_service

import (
	"context"
	"errors"
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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantToolService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

// CreateAssistantTool implements internal_services.AssistantToolService.
func (eService *assistantToolService) Create(ctx context.Context, auth types.SimplePrinciple, assistantId uint64,
	name string, description string, fields map[string]interface{}, executionMethod string, options []*lexatic_backend.Metadata) (*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	var existingTool internal_assistant_gorm.AssistantTool
	result := db.
		Where("assistant_id = ? AND name = ? AND status = ?", assistantId, name, type_enums.RECORD_ACTIVE).
		First(&existingTool)
	if result.Error == nil {
		eService.logger.Errorf("Tool with name %s already exists for assistant %d", name, assistantId)
		return nil, fmt.Errorf("tool with name %s already exists", name)
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("Error while checking for existing tool: %v", result.Error)
		return nil, result.Error
	}

	aTool := &internal_assistant_gorm.AssistantTool{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
		},
		AssistantId:     assistantId,
		Name:            name,
		Description:     description,
		Fields:          fields,
		ExecutionMethod: executionMethod,
	}
	tx := db.Create(&aTool)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantToolService.Create", time.Since(start))
		eService.logger.Errorf("error while create tool %v", tx.Error)
		return nil, tx.Error
	}

	v, err := eService.CreateOrUpdateExecutionOption(ctx, auth, aTool.Id, options)
	if err != nil {
		eService.logger.Benchmark("AssistantToolService.Create", time.Since(start))
		eService.logger.Errorf("error while updating tool options %v", tx.Error)
		return aTool, nil
	}
	aTool.ExecutionOptions = v
	eService.logger.Benchmark("AssistantToolService.Update", time.Since(start))
	return aTool, nil
}

// DeleteAssistantTool implements internal_services.AssistantToolService.
func (eService *assistantToolService) Delete(ctx context.Context, auth types.SimplePrinciple, toolId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	aK := &internal_assistant_gorm.AssistantTool{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		toolId,
		assistantId).Updates(&aK)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantToolService.Delete", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantToolService.Delete", time.Since(start))
	return aK, nil
}

// GetAllAssistantTool implements internal_services.AssistantToolService.
func (eService *assistantToolService) GetAll(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*lexatic_backend.Criteria, paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		aTools []*internal_assistant_gorm.AssistantTool
		cnt    int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantTool{})
	qry = qry.
		Preload("ExecutionOptions", "status = ?", type_enums.RECORD_ACTIVE).
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
		}).Find(&aTools)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any AssistantTool %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("AssistantTool.GetAll", time.Since(start))
	return cnt, aTools, nil
}

// GetAssistantTool implements internal_services.AssistantToolService.
func (eService *assistantToolService) Get(ctx context.Context, auth types.SimplePrinciple, toolId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var aK *internal_assistant_gorm.AssistantTool
	tx := db.
		Preload("ExecutionOptions", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ? AND assistant_id = ?", toolId, assistantId).
		First(&aK)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantToolService.Get", time.Since(start))
		eService.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantToolService.Get", time.Since(start))
	return aK, nil
}

// UpdateAssistantTool implements internal_services.AssistantToolService.
func (eService *assistantToolService) Update(ctx context.Context, auth types.SimplePrinciple,
	toolId uint64,
	assistantId uint64, name string, description string, fields map[string]interface{}, executionMethod string, options []*lexatic_backend.Metadata) (*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	eService.logger.Debugf("id = %d AND assistantId = %d", toolId, assistantId)
	var existingTool internal_assistant_gorm.AssistantTool
	result := db.Where("assistant_id = ? AND name = ? AND status = ? AND id != ?", assistantId, name, type_enums.RECORD_ACTIVE, toolId).First(&existingTool)
	if result.Error == nil {
		eService.logger.Errorf("Tool with name %s already exists for assistant %d", name, assistantId)
		return nil, fmt.Errorf("tool with name %s already exists", name)
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("Error while checking for existing tool: %v", result.Error)
		return nil, result.Error
	}

	//
	aTool := &internal_assistant_gorm.AssistantTool{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
		Name:            name,
		Description:     description,
		Fields:          fields,
		ExecutionMethod: executionMethod,
	}

	tx := db.Where("id = ? AND assistant_id = ? ",
		toolId,
		assistantId).Updates(&aTool)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantToolService.Update", time.Since(start))
		eService.logger.Errorf("error while updating tool %v", tx.Error)
		return nil, tx.Error
	}
	//
	err := eService.MarkAllOptionsAsDeleted(ctx, auth, toolId)
	if err != nil {
		eService.logger.Benchmark("AssistantToolService.Update", time.Since(start))
		eService.logger.Errorf("error while updating tool options %v", tx.Error)
		return aTool, nil
	}
	//
	v, err := eService.CreateOrUpdateExecutionOption(ctx, auth, toolId, options)
	if err != nil {
		eService.logger.Benchmark("AssistantToolService.Update", time.Since(start))
		eService.logger.Errorf("error while updating tool options %v", tx.Error)
		return aTool, nil
	}
	aTool.ExecutionOptions = v
	eService.logger.Benchmark("AssistantToolService.Update", time.Since(start))
	return aTool, nil
}

func (eService *assistantToolService) MarkAllOptionsAsDeleted(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantToolId uint64,
) error {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	tOptions := &internal_assistant_gorm.AssistantToolOption{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("assistant_tool_id = ? ",
		assistantToolId,
	).Updates(&tOptions)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.MarkAllOptionsAsDeleted", time.Since(start))
		eService.logger.Errorf("error while marking options as deleted: %v", tx.Error)
		return tx.Error
	}

	eService.logger.Benchmark("assistantService.MarkAllOptionsAsDeleted", time.Since(start))
	return nil
}

func (eService *assistantToolService) CreateOrUpdateExecutionOption(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantToolId uint64,
	metadata []*lexatic_backend.Metadata,
) ([]*internal_assistant_gorm.AssistantToolOption, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	mtrs := make([]*internal_assistant_gorm.AssistantToolOption, 0)
	for _, mtr := range metadata {
		_mtr := &internal_assistant_gorm.AssistantToolOption{
			Metadata: gorm_models.Metadata{
				Key:   mtr.GetKey(),
				Value: mtr.GetValue(),
			},
			Mutable: gorm_models.Mutable{
				Status: type_enums.RECORD_ACTIVE,
			},
			AssistantToolId: assistantToolId,
		}
		if auth.GetUserId() != nil {
			_mtr.UpdatedBy = *auth.GetUserId()
			_mtr.CreatedBy = *auth.GetUserId()
		}
		mtrs = append(mtrs, _mtr)
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}, {Name: "assistant_tool_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"value",
			"updated_by", "updated_date"}),
	}).Create(&mtrs)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.CreateOrUpdateMetadata", time.Since(start))
		eService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.CreateOrUpdateMetadata", time.Since(start))
	return mtrs, nil
}

func NewAssistantToolService(logger commons.Logger, postgres connectors.PostgresConnector, storage storages.Storage) internal_services.AssistantToolService {
	return &assistantToolService{
		logger:   logger,
		postgres: postgres,
		storage:  storage,
	}
}

func (eService *assistantToolService) CreateLog(
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
) (*internal_assistant_gorm.AssistantToolLog, error) {
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

	toolLog := &internal_assistant_gorm.AssistantToolLog{
		Audited: gorm_models.Audited{
			Id: _auditId,
		},
		AssistantId:                    assistantId,
		AssistantConversationId:        conversationId,
		AssistantConversationMessageId: messageId,
		ExecutionMethod:                executionMethod,
		AssistantToolId:                toolId,
		AssistantToolName:              toolName,
		AssetPrefix:                    s3Prefix,
		TimeTaken:                      timeTaken,
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Mutable: gorm_models.Mutable{
			Status: status,
		},
	}
	tx := db.Create(&toolLog)
	if tx.Error != nil {
		eService.logger.Benchmark("eService.Create", time.Since(start))
		eService.logger.Errorf("error while creating webhook log %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("eService.Create", time.Since(start))
	return toolLog, nil
}

func (eService *assistantToolService) GetLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	toolLogId uint64) (*internal_assistant_gorm.AssistantToolLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var wkg *internal_assistant_gorm.AssistantToolLog
	tx := db.
		Where("id = ? AND organization_id = ? AND project_id = ?", toolLogId, *auth.GetCurrentOrganizationId(), projectId).
		Preload("AssistantTool").
		Preload("AssistantTool.ExecutionOptions").
		First(&wkg)
	if tx.Error != nil {
		eService.logger.Benchmark("ToolService.GetLog", time.Since(start))
		eService.logger.Errorf("not able to find any tool %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("ToolService.GetLog", time.Since(start))
	return wkg, nil
}

func (eService *assistantToolService) GetAllLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	criterias []*lexatic_backend.Criteria,
	paginate *lexatic_backend.Paginate,
	order *lexatic_backend.Ordering) (int64, []*internal_assistant_gorm.AssistantToolLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		toolLogs []*internal_assistant_gorm.AssistantToolLog
		cnt      int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantToolLog{})
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
		}).Find(&toolLogs)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any Webhooks %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("ToolService.GetAllLog", time.Since(start))
	return cnt, toolLogs, nil
}

func (eService *assistantToolService) ObjectPrefix(orgId, projectId uint64) string {
	return fmt.Sprintf("%d/%d/tool", orgId, projectId)
}

func (eService *assistantToolService) ObjectKey(keyPrefix string, auditId uint64, objName string) string {
	return fmt.Sprintf("%s/%d__%s", keyPrefix, auditId, objName)
}

func (eService *assistantToolService) GetLogObject(
	ctx context.Context,
	organizationId,
	projectId, toolLogId uint64) (requestData []byte, responseData []byte, err error) {
	keyPrefix := eService.ObjectPrefix(organizationId, projectId)
	responseKey := eService.ObjectKey(keyPrefix, toolLogId, "response.json")
	requestKey := eService.ObjectKey(keyPrefix, toolLogId, "request.json")

	type _fileStruct struct {
		Key   string
		Data  []byte
		Error error
	}

	responseChan := make(chan _fileStruct)
	requestChan := make(chan _fileStruct)

	utils.Go(ctx, func() {
		eService.logger.Debugf("Getting key from s3 %s", responseKey)
		result := eService.storage.Get(ctx, responseKey)
		if result.Error != nil {
			eService.logger.Errorf("error downloading goroutine: %v", result.Error)
			responseChan <- _fileStruct{Key: responseKey, Error: result.Error}
			close(responseChan)
			return
		}
		responseChan <- _fileStruct{Key: responseKey, Data: result.Data}
		close(responseChan)
	})

	utils.Go(ctx, func() {
		eService.logger.Debugf("Getting key from s3 %s", requestKey)
		result := eService.storage.Get(ctx, requestKey)
		if result.Error != nil {
			eService.logger.Errorf("error downloading goroutine: %v", result.Error)
			requestChan <- _fileStruct{Key: requestKey, Error: result.Error}
			close(requestChan)
			return
		}
		requestChan <- _fileStruct{Key: requestKey, Data: result.Data}
		close(requestChan)

	})

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
