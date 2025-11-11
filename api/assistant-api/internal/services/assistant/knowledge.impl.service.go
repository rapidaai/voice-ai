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
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantKnowledgeService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

// CreateAssistantKnowledge implements internal_services.AssistantKnowledgeService.
func (eService *assistantKnowledgeService) Create(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, knowledgeId uint64,
	retrievalMethod gorm_types.RetrievalMethod,
	rerankEnabled bool, scoreThreshold float32, topK uint32, rerankerProviderModelId *uint64,
	rerankerProviderModelName *string,
	rerankerProviderModelOptions []*lexatic_backend.Metadata) (*internal_assistant_gorm.AssistantKnowledge, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	var existingAk internal_assistant_gorm.AssistantKnowledge
	result := db.Where("assistant_id = ? AND knowledge_id = ? AND status = ?", assistantId, knowledgeId, type_enums.RECORD_ACTIVE).First(&existingAk)
	if result.Error == nil {
		eService.logger.Errorf("Knowledge already exists for assistant %d", assistantId)
		return nil, fmt.Errorf("knowledge is already associate with current assistant")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("Error while checking for existing knowledge: %v", result.Error)
		return nil, result.Error
	}

	assistantKnowledgeConfig := &internal_assistant_gorm.AssistantKnowledge{
		AssistantId:     assistantId,
		KnowledgeId:     knowledgeId,
		RetrievalMethod: retrievalMethod,
		ScoreThreshold:  scoreThreshold,
		RerankerEnable:  rerankEnabled,
		TopK:            topK,
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: *auth.GetUserId(),
			UpdatedBy: *auth.GetUserId(),
		},
	}

	if rerankEnabled {
		assistantKnowledgeConfig.RerankerModelProviderId = rerankerProviderModelId
		assistantKnowledgeConfig.RerankerModelProviderName = rerankerProviderModelName
	}
	tx := db.Create(&assistantKnowledgeConfig)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.CreateAssistantKnowledge", time.Since(start))
		eService.logger.Errorf("error while updating tags %v", tx.Error)
		return nil, tx.Error
	}

	if len(rerankerProviderModelOptions) == 0 {
		return assistantKnowledgeConfig, nil
	}

	//
	modelOptions := make([]*internal_assistant_gorm.AssistantKnowledgeRerankerOption, 0)
	for _, v := range rerankerProviderModelOptions {
		modelOptions = append(modelOptions, &internal_assistant_gorm.AssistantKnowledgeRerankerOption{
			AssistantKnowledgeId: assistantKnowledgeConfig.Id,
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
	tx = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_provider_model_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by"}),
	}).Create(modelOptions)
	if tx.Error != nil {
		eService.logger.Errorf("unable to create model options with error %v", tx.Error)
		return nil, tx.Error
	}
	assistantKnowledgeConfig.AssistantKnowledgeRerankerOptions = modelOptions
	eService.logger.Benchmark("assistantService.CreateOrUpdateAssistantKnowledgeConfiguration", time.Since(start))
	return assistantKnowledgeConfig, nil
}

// DeleteAssistantKnowledge implements internal_services.AssistantKnowledgeService.
func (eService *assistantKnowledgeService) Delete(ctx context.Context, auth types.SimplePrinciple, akId uint64, assistantId uint64) (*internal_assistant_gorm.AssistantKnowledge, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	aK := &internal_assistant_gorm.AssistantKnowledge{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		akId,
		assistantId).Updates(&aK)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantKnowledgeService.Delete", time.Since(start))
		eService.logger.Errorf("error while deleting assistant knowledge %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantKnowledgeService.Delete", time.Since(start))
	return aK, nil
}

// GetAllAssistantKnowledge implements internal_services.AssistantKnowledgeService.
func (eService *assistantKnowledgeService) GetAll(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*lexatic_backend.Criteria, paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantKnowledge, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		analysises []*internal_assistant_gorm.AssistantKnowledge
		cnt        int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantKnowledge{})
	qry = qry.
		Preload("Knowledge").
		Preload("AssistantKnowledgeRerankerOptions").
		Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE)
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
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
		}).Find(&analysises)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any AssistantKnowledge %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("AssistantKnowledge.GetAll", time.Since(start))
	return cnt, analysises, nil
}

// GetAssistantKnowledge implements internal_services.AssistantKnowledgeService.
func (eService *assistantKnowledgeService) Get(ctx context.Context, auth types.SimplePrinciple, akId, assistantId uint64) (*internal_assistant_gorm.AssistantKnowledge, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var aK *internal_assistant_gorm.AssistantKnowledge
	tx := db.
		Preload("Knowledge").
		Preload("AssistantKnowledgeRerankerOptions").
		Where("id = ? AND assistant_id = ?", akId, assistantId).
		First(&aK)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantKnowledgeService.Get", time.Since(start))
		eService.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantKnowledgeService.Get", time.Since(start))
	return aK, nil
}

// UpdateAssistantKnowledge implements internal_services.AssistantKnowledgeService.
func (eService *assistantKnowledgeService) Update(ctx context.Context, auth types.SimplePrinciple, akId uint64, assistantId uint64, knowledgeId uint64, retrievalMethod gorm_types.RetrievalMethod, rerankEnabled bool, scoreThreshold float32, topK uint32, rerankerProviderModelId *uint64, rerankerProviderModelName *string, rerankerProviderModelOptions []*lexatic_backend.Metadata) (*internal_assistant_gorm.AssistantKnowledge, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	var existingAk internal_assistant_gorm.AssistantKnowledge
	result := db.Where("assistant_id = ? AND knowledge_id = ? AND status = ? AND id != ?", assistantId, knowledgeId, type_enums.RECORD_ACTIVE, akId).First(&existingAk)
	if result.Error == nil {
		eService.logger.Errorf("Knowledge already exists for assistant %d", assistantId)
		return nil, fmt.Errorf("knowledge is already associate with current assistant")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		eService.logger.Errorf("Error while checking for existing knowledge: %v", result.Error)
		return nil, result.Error
	}

	aK := &internal_assistant_gorm.AssistantKnowledge{
		KnowledgeId:     knowledgeId,
		RetrievalMethod: retrievalMethod,
		ScoreThreshold:  scoreThreshold,
		RerankerEnable:  rerankEnabled,
		TopK:            topK,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	if rerankEnabled {
		aK.RerankerModelProviderId = rerankerProviderModelId
		aK.RerankerModelProviderName = rerankerProviderModelName
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		akId,
		assistantId).Updates(&aK)
	if tx.Error != nil {
		eService.logger.Benchmark("AssistantKnowledgeService.Update", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AssistantKnowledgeService.Update", time.Since(start))
	return aK, nil
}

func NewAssistantKnowledgeService(logger commons.Logger,
	postgres connectors.PostgresConnector, storage storages.Storage) internal_services.AssistantKnowledgeService {
	return &assistantKnowledgeService{
		logger:   logger,
		postgres: postgres,
		storage:  storage,
	}
}
