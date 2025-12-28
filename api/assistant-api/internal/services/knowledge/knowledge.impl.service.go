// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_knowledge_service

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"

	"gorm.io/gorm/clause"
)

type knowledgeService struct {
	logger   commons.Logger
	config   *config.AssistantConfig
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

func NewKnowledgeService(config *config.AssistantConfig,
	logger commons.Logger, postgres connectors.PostgresConnector, storage storages.Storage) internal_services.KnowledgeService {
	return &knowledgeService{
		logger:   logger,
		config:   config,
		postgres: postgres,
		storage:  storage,
	}
}

func (knowledge *knowledgeService) GetAll(
	ctx context.Context,
	auth types.SimplePrinciple,
	criterias []*protos.Criteria,
	paginate *protos.Paginate) (int64, *[]internal_knowledge_gorm.Knowledge, error) {
	db := knowledge.postgres.DB(ctx)
	var knowledges []internal_knowledge_gorm.Knowledge
	var cnt int64
	qry := db.Model(internal_knowledge_gorm.Knowledge{}).
		Where("organization_id = ? AND status = ? And project_id = ?", *auth.GetCurrentOrganizationId(), type_enums.RECORD_ACTIVE.String(), *auth.GetCurrentProjectId())
	for _, ct := range criterias {
		switch ct.GetLogic() {
		case "or":
			qry.Or(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
		case "like":
			qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), fmt.Sprintf("%%%s%%", ct.GetValue()))
		default:
			qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())

		}
	}
	tx := qry.
		Preload("KnowledgeTag").
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
		}).
		Find(&knowledges)
	if tx.Error != nil {
		knowledge.logger.Debugf("unable to find any knowledge for given project %v and organization  %v", *auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId())
		return cnt, nil, tx.Error
	}

	return cnt, &knowledges, nil
}

func (knowledge *knowledgeService) Get(ctx context.Context, auth types.SimplePrinciple, knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error) {
	db := knowledge.postgres.DB(ctx)
	var _knowledge internal_knowledge_gorm.Knowledge
	tx := db.
		Preload("KnowledgeTag").
		Preload("KnowledgeEmbeddingModelOptions").
		Where("id = ? AND status = ? AND project_id = ? AND organization_id = ?", knowledgeId, type_enums.RECORD_ACTIVE.String(), *auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId()).First(&_knowledge)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &_knowledge, nil
}

func (knowledge *knowledgeService) CreateOrUpdateKnowledgeTag(ctx context.Context,
	auth types.SimplePrinciple,
	knowledgeId uint64,
	tags []string,
) (*internal_knowledge_gorm.KnowledgeTag, error) {

	db := knowledge.postgres.DB(ctx)
	knowledgeTag := &internal_knowledge_gorm.KnowledgeTag{
		KnowledgeId: knowledgeId,
		Tag:         tags,
		CreatedBy:   *auth.GetUserId(),
		UpdatedBy:   *auth.GetUserId(),
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "knowledge_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"tag",
			"updated_by"}),
	}).Create(&knowledgeTag)

	if tx.Error != nil {
		knowledge.logger.Errorf("error while updating tags %v", tx.Error)
		return nil, tx.Error
	}
	return knowledgeTag, nil
}

func (knowledge *knowledgeService) CreateKnowledge(ctx context.Context, auth types.SimplePrinciple,
	name string, description, visibility *string,
	embeddingModelProviderName string,
	embeddingModelOptions []*protos.Metadata,
) (*internal_knowledge_gorm.Knowledge, error) {
	db := knowledge.postgres.DB(ctx)
	knowledgeId := gorm_generator.ID()
	_knowledge := &internal_knowledge_gorm.Knowledge{
		Audited: gorm_models.Audited{
			Id: knowledgeId,
		},
		Name:           name,
		ProjectId:      *auth.GetCurrentProjectId(),
		OrganizationId: *auth.GetCurrentOrganizationId(),
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
		},
		EmbeddingModelProviderName: embeddingModelProviderName,
		StorageNamespace:           commons.KnowledgeIndex(knowledge.config.IsDevelopment(), *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), knowledgeId),
	}

	if visibility != nil {
		_knowledge.Visibility = *visibility
	}
	if description != nil {
		_knowledge.Description = *description
	}
	if err := db.Save(_knowledge).Error; err != nil {
		knowledge.logger.Errorf("unable to create assistant with error %+v", err)
		return nil, err
	}

	if len(embeddingModelOptions) == 0 {
		return _knowledge, nil
	}
	modelOptions := make([]*internal_knowledge_gorm.KnowledgeEmbeddingModelOption, 0)
	for _, v := range embeddingModelOptions {
		modelOptions = append(modelOptions, &internal_knowledge_gorm.KnowledgeEmbeddingModelOption{
			KnowledgeId: _knowledge.Id,
			Mutable: gorm_models.Mutable{
				CreatedBy: *auth.GetUserId(),
				UpdatedBy: *auth.GetUserId(),
				Status:    type_enums.RECORD_ACTIVE,
			},
			Metadata: gorm_models.Metadata{
				Key:   v.GetKey(),
				Value: v.GetValue(),
			},
		})
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "knowledge_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(modelOptions)
	_knowledge.KnowledgeEmbeddingModelOptions = modelOptions
	if tx.Error != nil {
		knowledge.logger.Errorf("unable to create model options with error %v", tx.Error)
		return nil, tx.Error
	}
	return _knowledge, nil

}

func (eService *knowledgeService) UpdateKnowledgeDetail(ctx context.Context,
	auth types.SimplePrinciple,
	knowledgeId uint64,
	name, description string) (*internal_knowledge_gorm.Knowledge, error) {
	db := eService.postgres.DB(ctx)
	ed := &internal_knowledge_gorm.Knowledge{
		Name:        name,
		Description: description,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", knowledgeId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Errorf("error while updating for assistant %v", tx.Error)
		return nil, tx.Error
	}
	return ed, nil
}

func (eService *knowledgeService) CreateLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	knowledgeId uint64,
	retrievalMethod string,
	topK uint32,
	scoreThreshold float32,
	documentCount int,
	timeTaken int64,
	additionalData map[string]string,
	status type_enums.RecordState,
	request, response []byte,
) (*internal_knowledge_gorm.KnowledgeLog, error) {
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

	toolLog := &internal_knowledge_gorm.KnowledgeLog{
		Audited: gorm_models.Audited{
			Id: _auditId,
		},
		KnowledgeId:     knowledgeId,
		AssetPrefix:     s3Prefix,
		TimeTaken:       timeTaken,
		RetrievalMethod: retrievalMethod,
		TopK:            topK,
		ScoreThreshold:  scoreThreshold,
		DocumentCount:   documentCount,
		AdditionalData:  additionalData,
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

func (eService *knowledgeService) GetLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	toolLogId uint64) (*internal_knowledge_gorm.KnowledgeLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var wkg *internal_knowledge_gorm.KnowledgeLog
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ?", toolLogId, *auth.GetCurrentOrganizationId(), projectId).
		First(&wkg)
	if tx.Error != nil {
		eService.logger.Benchmark("ToolService.GetLog", time.Since(start))
		eService.logger.Errorf("not able to find any tool %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("ToolService.GetLog", time.Since(start))
	return wkg, nil
}

func (eService *knowledgeService) GetAllLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
	order *protos.Ordering) (int64, []*internal_knowledge_gorm.KnowledgeLog, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		toolLogs []*internal_knowledge_gorm.KnowledgeLog
		cnt      int64
	)
	qry := db.Model(internal_knowledge_gorm.KnowledgeLog{})
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

func (eService *knowledgeService) ObjectPrefix(orgId, projectId uint64) string {
	return fmt.Sprintf("%d/%d/knowledge", orgId, projectId)
}

func (eService *knowledgeService) ObjectKey(keyPrefix string, auditId uint64, objName string) string {
	return fmt.Sprintf("%s/%d__%s", keyPrefix, auditId, objName)
}

func (eService *knowledgeService) GetLogObject(
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
