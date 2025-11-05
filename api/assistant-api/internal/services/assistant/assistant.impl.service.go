package internal_assistant_service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	assistant_grpc_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantService struct {
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	opensearch connectors.OpenSearchConnector
	cfg        *config.AssistantConfig
}

func NewAssistantService(cfg *config.AssistantConfig, logger commons.Logger, postgres connectors.PostgresConnector, opensearch connectors.OpenSearchConnector) internal_services.AssistantService {
	return &assistantService{
		logger:     logger,
		postgres:   postgres,
		opensearch: opensearch,
		cfg:        cfg,
	}
}

func (eService *assistantService) CreateAssistantProviderWebsocket(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	description string,
	url string,
	headers map[string]string,
	parameters map[string]string,
) (*internal_assistant_gorm.AssistantProviderWebsocket, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	epm := &internal_assistant_gorm.AssistantProviderWebsocket{
		AssistantProvider: internal_assistant_gorm.AssistantProvider{
			Description: description,
			CreatedBy:   *auth.GetUserId(),
			AssistantId: assistantId,
		},
		Url:        url,
		Headers:    headers,
		Parameters: parameters,
	}
	tx := db.Save(epm)
	if err := tx.Error; err != nil {
		eService.logger.Benchmark("assistantService.CreateAssistantProviderWebsocket", time.Since(start))
		eService.logger.Errorf("unable to create assistant provider websocket.")
		return nil, err
	}
	eService.logger.Benchmark("assistantService.CreateAssistantProviderWebsocket", time.Since(start))
	return epm, nil
}

func (eService *assistantService) CreateAssistantProviderAgentkit(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	description string,
	url string,
	certificate string,
	metadata map[string]string,
) (*internal_assistant_gorm.AssistantProviderAgentkit, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	epm := &internal_assistant_gorm.AssistantProviderAgentkit{
		AssistantProvider: internal_assistant_gorm.AssistantProvider{
			Description: description,
			CreatedBy:   *auth.GetUserId(),
			AssistantId: assistantId,
		},
		Url:         url,
		Certificate: certificate,
		Metadata:    metadata,
	}
	tx := db.Save(epm)
	if err := tx.Error; err != nil {
		eService.logger.Benchmark("assistantService.CreateAssistantProviderAgentkit", time.Since(start))
		eService.logger.Errorf("unable to create assistant provider agentKit.")
		return nil, err
	}
	eService.logger.Benchmark("assistantService.CreateAssistantProviderAgentkit", time.Since(start))
	return epm, nil
}

func (eService *assistantService) DeleteAssistant(ctx context.Context, auth types.SimplePrinciple, assistantId uint64) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	ed := &internal_assistant_gorm.Assistant{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
			Status:    type_enums.RECORD_ARCHIEVE,
		},
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", assistantId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.DeleteAssistant", time.Since(start))
		eService.logger.Errorf("error while updating assistant %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.DeleteAssistant", time.Since(start))
	return ed, nil
}

func (eService *assistantService) GetAllAssistantTool(ctx context.Context, auth types.SimplePrinciple, assistantId uint64, criterias []*assistant_grpc_api.Criteria, paginate *assistant_grpc_api.Paginate) (int64, []*internal_assistant_gorm.AssistantTool, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		assistantTools []*internal_assistant_gorm.AssistantTool
		cnt            int64
	)
	qry := db.Model(internal_assistant_gorm.AssistantTool{})
	qry.
		Preload("Tool").
		Where("organization_id = ? AND project_id = ? AND assistant_id = ? AND status = ?", *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), assistantId, type_enums.RECORD_ACTIVE.String())
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
		}).Find(&assistantTools)

	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.GetAllAssistantTool", time.Since(start))
		eService.logger.Errorf("not able to find any assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.GetAllAssistantTool", time.Since(start))
	return cnt, assistantTools, nil

}

func (eService *assistantService) Get(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantProviderId *uint64,
	opts *internal_services.GetAssistantOption) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	// call assistant get the details
	var assistant *internal_assistant_gorm.Assistant
	tx := db.Where("assistants.id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE.String()).First(&assistant)
	if tx.Error != nil {
		return nil, tx.Error
	}
	//
	if assistant.Visibility != "public" {
		if *auth.GetCurrentOrganizationId() != assistant.OrganizationId || *auth.GetCurrentProjectId() != assistant.ProjectId {
			return nil, fmt.Errorf("you don't have access to the assistant")
		}
	}

	// get assistant
	var wg sync.WaitGroup
	if opts.InjectTag {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var tag *internal_assistant_gorm.AssistantTag
				tx := db.Where("assistant_id = ?", assistantId).First(&tag)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant tag with error %+v", tx.Error)
					return
				}
				assistant.AssistantTag = tag
			})
	}

	if opts.InjectKnowledgeConfiguration {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var knowledgeConfigs []*internal_assistant_gorm.AssistantKnowledge
				tx := db.
					Preload("Knowledge").
					Preload("Knowledge.KnowledgeEmbeddingModelOptions").
					Preload("AssistantKnowledgeRerankerOptions").
					Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE.String()).Find(&knowledgeConfigs)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant knowledge with error %+v", tx.Error)
					return
				}
				assistant.AssistantKnowledges = knowledgeConfigs
			})
	}

	if opts.InjectTool {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var assistantTools []*internal_assistant_gorm.AssistantTool
				tx := db.
					Preload("ExecutionOptions", "status = ?", type_enums.RECORD_ACTIVE).
					Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE.String()).
					Find(&assistantTools)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant skills with error %+v", tx.Error)
					return
				}
				assistant.AssistantTools = assistantTools
			})
	}

	//
	//  injecting deployment
	//
	if opts.InjectApiDeployment {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var deployment *internal_assistant_gorm.AssistantApiDeployment
				tx := db.
					Preload("InputAudio", "audio_type = ?", "input").
					Preload("OuputAudio", "audio_type = ?", "output").
					Preload("InputAudio.AudioOptions").
					Preload("OuputAudio.AudioOptions").
					Order(clause.OrderByColumn{
						Column: clause.Column{Name: "created_date"},
						Desc:   true,
					}).
					Where("assistant_id = ?", assistantId).First(&deployment)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant api deployment with error %+v", tx.Error)
					return
				}
				assistant.AssistantApiDeployment = deployment
			})
	}
	if opts.InjectDebuggerDeployment {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var deployment *internal_assistant_gorm.AssistantDebuggerDeployment
				tx := db.
					Preload("InputAudio", "audio_type = ?", "input").
					Preload("OuputAudio", "audio_type = ?", "output").
					Preload("InputAudio.AudioOptions").
					Preload("OuputAudio.AudioOptions").
					Where("assistant_id = ?", assistantId).
					Order(clause.OrderByColumn{
						Column: clause.Column{Name: "created_date"},
						Desc:   true,
					}).
					First(&deployment)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant debugger deployment with error %+v", tx.Error)
					return
				}
				assistant.AssistantDebuggerDeployment = deployment
			})
	}
	if opts.InjectWebpluginDeployment {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var deployment *internal_assistant_gorm.AssistantWebPluginDeployment
				tx := db.
					Preload("InputAudio", "audio_type = ?", "input").
					Preload("OuputAudio", "audio_type = ?", "output").
					Preload("InputAudio.AudioOptions").
					Preload("OuputAudio.AudioOptions").
					Order(clause.OrderByColumn{
						Column: clause.Column{Name: "created_date"},
						Desc:   true,
					}).
					Where("assistant_id = ?", assistantId).First(&deployment)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant debugger deployment with error %+v", tx.Error)
					return
				}
				assistant.AssistantWebPluginDeployment = deployment
			})
	}
	if opts.InjectWhatsappDeployment {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var deployment *internal_assistant_gorm.AssistantWhatsappDeployment
				tx := db.
					Where("assistant_id = ?", assistantId).First(&deployment)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant whatsapp deployment with error %+v", tx.Error)
					return
				}
				assistant.AssistantWhatsappDeployment = deployment
			})
	}
	if opts.InjectPhoneDeployment {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var deployment *internal_assistant_gorm.AssistantPhoneDeployment
				tx := db.Debug().
					Preload("InputAudio", "audio_type = ?", "input").
					Preload("OuputAudio", "audio_type = ?", "output").
					Preload("InputAudio.AudioOptions").
					Preload("OuputAudio.AudioOptions").
					Preload("TelephonyOption").
					Order(clause.OrderByColumn{
						Column: clause.Column{Name: "created_date"},
						Desc:   true,
					}).
					Where("assistant_id = ?", assistantId).First(&deployment)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant phone deployment with error %+v", tx.Error)
					return
				}
				assistant.AssistantPhoneDeployment = deployment
			})
	}

	//
	//
	if opts.InjectAssistantProvider {
		// if version is already there the load from that version
		if assistantProviderId != nil {
			assistant.AssistantProviderId = *assistantProviderId
		}

		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var providerModel *internal_assistant_gorm.AssistantProviderModel
				tx := db.
					Preload("AssistantModelOptions").
					Where("assistant_id = ? AND id = ?",
						assistantId,
						assistant.AssistantProviderId).
					First(&providerModel)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant provider model with error %+v", tx.Error)
					return
				}
				assistant.AssistantProviderModel = providerModel
			})

		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var websocket *internal_assistant_gorm.AssistantProviderWebsocket
				tx := db.
					Where("assistant_id = ? AND id = ?",
						assistantId,
						assistant.AssistantProviderId).
					First(&websocket)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant provider model with error %+v", tx.Error)
					return
				}
				assistant.AssistantProviderWebsocket = websocket
			})

		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var agentkit *internal_assistant_gorm.AssistantProviderAgentkit
				tx := db.
					Where("assistant_id = ? AND id = ?",
						assistantId,
						assistant.AssistantProviderId).
					First(&agentkit)
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant provider model with error %+v", tx.Error)
					return
				}
				assistant.AssistantProviderAgentkit = agentkit
			})
	}

	if opts.InjectAnalysis {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var webhooks []*internal_assistant_gorm.AssistantWebhook
				tx := db.
					Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE.String()).
					Find(&webhooks).
					Order("execution_priority DESC")
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant provider model with error %+v", tx.Error)
					return
				}
				assistant.AssistantWebhooks = webhooks
			})
	}

	if opts.InjectWebhook {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var analysis []*internal_assistant_gorm.AssistantAnalysis
				tx := db.
					Where("assistant_id = ? AND status = ?", assistantId, type_enums.RECORD_ACTIVE.String()).
					Find(&analysis).
					Order("execution_priority DESC")
				if tx.Error != nil {
					eService.logger.Warnf("unable to find assistant provider model with error %+v", tx.Error)
					return
				}
				assistant.AssistantAnalyses = analysis
			})
	}
	wg.Wait()
	eService.logger.Benchmark("assistantService.Get", time.Since(start))
	return assistant, nil
}

func (eService *assistantService) UpdateAssistantVersion(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantProvider type_enums.AssistantProvider,
	assistantProviderId uint64) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	ed := &internal_assistant_gorm.Assistant{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
		AssistantProvider:   assistantProvider,
		AssistantProviderId: assistantProviderId,
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", assistantId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.UpdateAssistantVersion", time.Since(start))
		eService.logger.Errorf("error while updating assistant %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.UpdateAssistantVersion", time.Since(start))
	return ed, nil
}

func (eService *assistantService) GetAll(ctx context.Context, auth types.SimplePrinciple, criterias []*assistant_grpc_api.Criteria, paginate *assistant_grpc_api.Paginate, opts *internal_services.GetAssistantOption) (int64, []*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		assistants []*internal_assistant_gorm.Assistant
		cnt        int64
	)
	qry := db.Model(internal_assistant_gorm.Assistant{})
	qry = qry.
		Preload("AssistantTag").
		Preload("AssistantProviderModel")

	if opts.InjectWhatsappDeployment {
		qry = qry.Preload("AssistantWhatsappDeployment", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_date DESC")
		})
	}
	if opts.InjectPhoneDeployment {
		qry = qry.Preload("AssistantPhoneDeployment", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_date DESC")
		})
	}
	if opts.InjectApiDeployment {
		qry = qry.Preload("AssistantApiDeployment", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_date DESC")
		})
	}
	if opts.InjectDebuggerDeployment {
		qry = qry.Preload("AssistantDebuggerDeployment", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_date DESC")
		})
	}
	if opts.InjectWebpluginDeployment {
		qry = qry.Preload("AssistantWebPluginDeployment", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_date DESC")
		})
	}

	if opts.InjectConversations {
		qry = qry.Preload("AssistantConversations", func(db *gorm.DB) *gorm.DB {
			thirtyDaysAgo := time.Now().AddDate(0, 0, -31)
			return db.Where("created_date >= ?", thirtyDaysAgo).Order("updated_date DESC")
		})
	}

	qry = qry.Where("organization_id = ? AND project_id = ? AND status = ?", *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), type_enums.RECORD_ACTIVE.String())
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
		}).Find(&assistants)

	if tx.Error != nil {
		eService.logger.Errorf("not able to find any assistant %v", tx.Error)
		eService.logger.Benchmark("assistantService.GetAll", time.Since(start))
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.GetAll", time.Since(start))
	return cnt, assistants, nil
}

func (eService *assistantService) GetAllAssistantProviderModel(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64, criterias []*assistant_grpc_api.Criteria,
	paginate *assistant_grpc_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderModel, error) {

	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		epms []*internal_assistant_gorm.AssistantProviderModel
		cnt  int64
	)
	// use projectId and orgId to validate that he has access to the assistant
	qry := db.Model(internal_assistant_gorm.AssistantProviderModel{})
	qry.
		Preload("AssistantModelOptions").
		Where("assistant_id = ? ", assistantId)
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
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
		}).Find(&epms)

	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.GetAllAssistantProviderModel", time.Since(start))
		eService.logger.Errorf("not able to find any assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.GetAllAssistantProviderModel", time.Since(start))
	return cnt, epms, nil
}

func (eService *assistantService) GetAllAssistantProviderAgentkit(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64, criterias []*assistant_grpc_api.Criteria,
	paginate *assistant_grpc_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderAgentkit, error) {

	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		epms []*internal_assistant_gorm.AssistantProviderAgentkit
		cnt  int64
	)
	// use projectId and orgId to validate that he has access to the assistant
	qry := db.Model(internal_assistant_gorm.AssistantProviderAgentkit{})
	qry.
		Where("assistant_id = ? ", assistantId)
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
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
		}).Find(&epms)

	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.GetAllAssistantProviderAgentkit", time.Since(start))
		eService.logger.Errorf("not able to find any assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.GetAllAssistantProviderAgentkit", time.Since(start))
	return cnt, epms, nil
}

func (eService *assistantService) GetAllAssistantProviderWebsocket(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64, criterias []*assistant_grpc_api.Criteria,
	paginate *assistant_grpc_api.Paginate) (int64, []*internal_assistant_gorm.AssistantProviderWebsocket, error) {

	start := time.Now()
	db := eService.postgres.DB(ctx)
	var (
		epms []*internal_assistant_gorm.AssistantProviderWebsocket
		cnt  int64
	)
	// use projectId and orgId to validate that he has access to the assistant
	qry := db.Model(internal_assistant_gorm.AssistantProviderWebsocket{})
	qry.
		Where("assistant_id = ? ", assistantId)
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
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
		}).Find(&epms)

	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.GetAllAssistantProviderWebsocket", time.Since(start))
		eService.logger.Errorf("not able to find any assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.GetAllAssistantProviderWebsocket", time.Since(start))
	return cnt, epms, nil
}

func (eService *assistantService) CreateAssistant(ctx context.Context,
	auth types.SimplePrinciple,
	name, description string,
	visibility string,
	source string,
	sourceIdentifier *uint64,
	language string) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	ep := &internal_assistant_gorm.Assistant{
		Mutable: gorm_models.Mutable{
			CreatedBy: *auth.GetUserId(),
			Status:    type_enums.RECORD_ACTIVE,
		},
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Visibility:  visibility,
		Name:        name,
		Description: description,
		Source:      source,
		Language:    language,
	}

	if sourceIdentifier != nil {
		ep.SourceIdentifier = sourceIdentifier
	}

	if err := db.Save(ep).Error; err != nil {
		eService.logger.Benchmark("assistantService.CreateAssistant", time.Since(start))
		eService.logger.Errorf("unable to create assistant with error %+v", err)
		return nil, err
	}
	eService.logger.Benchmark("assistantService.CreateAssistant", time.Since(start))
	return ep, nil
}

func (eService *assistantService) UpdateAssistantDetail(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	name, description string) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	ed := &internal_assistant_gorm.Assistant{
		Mutable: gorm_models.Mutable{
			UpdatedBy: *auth.GetUserId(),
		},
		Name:        name,
		Description: description,
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", assistantId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Updates(ed)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.UpdateAssistantDetail", time.Since(start))
		eService.logger.Errorf("error while updating for assistant %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.UpdateAssistantDetail", time.Since(start))
	return ed, nil
}

func (eService *assistantService) CreateAssistantProviderModel(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	description string,
	promptRequest string,
	modelProviderId uint64,
	modelProviderName string,
	options []*lexatic_backend.Metadata,
) (*internal_assistant_gorm.AssistantProviderModel, error) {
	start := time.Now()

	db := eService.postgres.DB(ctx)
	epm := &internal_assistant_gorm.AssistantProviderModel{
		AssistantProvider: internal_assistant_gorm.AssistantProvider{
			Description: description,
			CreatedBy:   *auth.GetUserId(),
		},
		AssistantId:       assistantId,
		ModelProviderId:   modelProviderId,
		ModelProviderName: modelProviderName,
	}
	epm.SetPrompt(promptRequest)
	tx := db.Save(epm)
	if err := tx.Error; err != nil {
		eService.logger.Benchmark("assistantService.CreateAssistantProviderModel", time.Since(start))
		eService.logger.Errorf("unable to create assistant.")
		return nil, err
	}

	if len(options) == 0 {
		return epm, nil
	}
	modelOptions := make([]*internal_assistant_gorm.AssistantProviderModelOption, 0)
	for _, v := range options {
		modelOptions = append(modelOptions, &internal_assistant_gorm.AssistantProviderModelOption{
			AssistantProviderModelId: epm.Id,
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
	epm.AssistantModelOptions = modelOptions
	eService.logger.Benchmark("assistantService.CreateAssistantProviderModel", time.Since(start))
	return epm, nil
}

func (eService *assistantService) AttachProviderModelToAssistant(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	providerType type_enums.AssistantProvider,
	assistantProviderId uint64) (*internal_assistant_gorm.Assistant, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)
	ed := &internal_assistant_gorm.Assistant{
		AssistantProvider:   providerType,
		AssistantProviderId: assistantProviderId,
		Mutable:             gorm_models.Mutable{UpdatedBy: *auth.GetUserId()},
	}
	tx := db.Where("id = ? AND project_id = ? AND organization_id = ?", assistantId,
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId(),
	).Clauses(clause.Returning{}).Updates(ed)
	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.AttachProviderModelToAssistant", time.Since(start))
		eService.logger.Errorf("error while updating for assistant provider model %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.AttachProviderModelToAssistant", time.Since(start))
	return ed, nil
}

func (eService *assistantService) CreateOrUpdateAssistantTag(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	tags []string,
) (*internal_assistant_gorm.AssistantTag, error) {
	start := time.Now()

	db := eService.postgres.DB(ctx)
	assistantTag := &internal_assistant_gorm.AssistantTag{
		AssistantId: assistantId,
		Tag:         tags,
		CreatedBy:   *auth.GetUserId(),
		UpdatedBy:   *auth.GetUserId(),
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"tag",
			"updated_by"}),
	}).Create(&assistantTag)

	if tx.Error != nil {
		eService.logger.Benchmark("assistantService.CreateOrUpdateAssistantTag", time.Since(start))
		eService.logger.Errorf("error while updating tags %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("assistantService.CreateOrUpdateAssistantTag", time.Since(start))
	return assistantTag, nil
}
