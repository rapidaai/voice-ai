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
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type assistantAnalysisService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewAssistantAnalysisService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.AssistantAnalysisService {
	return &assistantAnalysisService{
		logger:   logger,
		postgres: postgres,
	}
}

// Get implements internal_services.AssistantAnalysisService.
func (eService *assistantAnalysisService) Get(ctx context.Context, auth types.SimplePrinciple, analysisId, assistantId uint64) (*internal_assistant_gorm.AssistantAnalysis, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var Analysis *internal_assistant_gorm.AssistantAnalysis
	tx := db.Where("id = ? AND assistant_id = ?", analysisId, assistantId).
		First(&Analysis)
	if tx.Error != nil {
		eService.logger.Benchmark("AnalysisService.Get", time.Since(start))
		eService.logger.Errorf("not able to find any webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("AnalysisService.Get", time.Since(start))
	return Analysis, nil
}

func (eService *assistantAnalysisService) Create(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	name string,
	endpointId uint64,
	endpointVersion string,
	endpointParameters map[string]string,
	executionPriority uint32,
	description *string,
) (*internal_assistant_gorm.AssistantAnalysis, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	analysis := &internal_assistant_gorm.AssistantAnalysis{
		AssistantId:        assistantId,
		Description:        *description,
		Name:               name,
		EndpointId:         endpointId,
		EndpointVersion:    endpointVersion,
		EndpointParameters: endpointParameters,
		ExecutionPriority:  executionPriority,
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RECORD_ACTIVE,
		},
	}
	tx := db.Create(&analysis)
	if tx.Error != nil {
		eService.logger.Benchmark("eService.Create", time.Since(start))
		eService.logger.Errorf("error while creating analysis %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("eService.Create", time.Since(start))
	return analysis, nil
}

func (eService *assistantAnalysisService) Update(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	analysisId uint64,
	name string,
	endpointId uint64,
	endpointVersion string,
	endpointParameters map[string]string,
	executionPriority uint32,
	description *string,
) (*internal_assistant_gorm.AssistantAnalysis, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	analysis := &internal_assistant_gorm.AssistantAnalysis{
		Description:        *description,
		Name:               name,
		EndpointId:         endpointId,
		EndpointVersion:    endpointVersion,
		EndpointParameters: endpointParameters,
		ExecutionPriority:  executionPriority,
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		analysisId,
		assistantId).Updates(&analysis)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantAnalysisService.Update", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantAnalysisService.Update", time.Since(start))
	return analysis, nil
}

func (eService *assistantAnalysisService) Delete(ctx context.Context,
	auth types.SimplePrinciple,
	analysisId uint64,
	assistantId uint64,
) (*internal_assistant_gorm.AssistantAnalysis, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	analysis := &internal_assistant_gorm.AssistantAnalysis{
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			UpdatedBy: *auth.GetUserId(),
		},
	}
	tx := db.Where("id = ? AND assistant_id = ? ",
		analysisId,
		assistantId).Updates(&analysis)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantAnalysisService.Update", time.Since(start))
		eService.logger.Errorf("error while creating webhook %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantAnalysisService.Update", time.Since(start))
	return analysis, nil
}

// GetAll implements internal_services.AssistantAnalysisService.
func (eService *assistantAnalysisService) GetAll(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	criterias []*lexatic_backend.Criteria,
	paginate *lexatic_backend.Paginate) (int64, []*internal_assistant_gorm.AssistantAnalysis, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		analysises []*internal_assistant_gorm.AssistantAnalysis
		cnt        int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantAnalysis{})
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
		}).Find(&analysises)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any Webhooks %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("WebhookService.GetAll", time.Since(start))
	return cnt, analysises, nil
}
