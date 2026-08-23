// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type assistantConversationService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

func NewAssistantConversationService(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
	storage storages.Storage) internal_services.AssistantConversationService {
	return &assistantConversationService{
		logger:   logger,
		postgres: postgres,
		storage:  storage,
	}
}

func (conversationService *assistantConversationService) GetAll(ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate, opts *internal_services.GetConversationOption) (int64, []*internal_conversation_entity.AssistantConversation, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return 0, nil, err
	}

	const numericMetricValuePattern = `^[0-9]+(\.[0-9]+)?$`

	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversations []*internal_conversation_entity.AssistantConversation
		cnt           int64
	)
	qry := db.Model(internal_conversation_entity.AssistantConversation{})
	qry = qry.
		Where("assistant_id = ? AND organization_id = ? AND project_id = ?", assistantId, projectContext.OrganizationID, projectContext.ProjectID)

	if opts != nil && opts.InjectMetric {
		qry = qry.
			Preload("Metrics")
	}

	if opts != nil && opts.InjectMetadata {
		qry = qry.
			Preload("Metadatas")
	}

	if opts != nil && opts.InjectArgument {
		qry = qry.
			Preload("Arguments")
	}

	for _, ct := range criterias {
		if ct == nil || ct.GetKey() == "" || ct.GetValue() == "" {
			continue
		}
		switch ct.GetKey() {
		case "id":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("assistant_conversations.id = ?", ct.GetValue())
			}
		case "created_date":
			switch ct.GetLogic() {
			case ">=":
				qry = qry.Where("assistant_conversations.created_date >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("assistant_conversations.created_date <= ?", ct.GetValue())
			default:
				qry = qry.Where("assistant_conversations.created_date = ?", ct.GetValue())
			}
		case "identifier":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("assistant_conversations.identifier ILIKE ?", "%"+ct.GetValue()+"%")
			default:
				qry = qry.Where("assistant_conversations.identifier = ?", ct.GetValue())
			}
		case "source":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("assistant_conversations.source = ?", ct.GetValue())
			}
		case "direction":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("assistant_conversations.direction = ?", ct.GetValue())
			}
		case "assistant_provider_model_id":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("assistant_conversations.assistant_provider_model_id = ?", ct.GetValue())
			}
		case "status":
			switch ct.GetLogic() {
			default:
				if ct.GetValue() == "ACTIVE" {
					qry = qry.Where(
						"(EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value = ?) OR NOT EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ?))",
						"status",
						ct.GetValue(),
						"status",
					)
				} else {
					qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value = ?)", "status", ct.GetValue())
				}
			}
		case "call.status":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value = ?)", "call.status", ct.GetValue())
			}
		case "client.direction":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.direction", ct.GetValue())
			}
		case "client.channel":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.channel", ct.GetValue())
			}
		case "client.provider_call_id":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value ILIKE ?)", "client.provider_call_id", "%"+ct.GetValue()+"%")
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.provider_call_id", ct.GetValue())
			}
		case "client.codec":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.codec", ct.GetValue())
			}
		case "client.sample_rate":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.sample_rate", ct.GetValue())
			}
		case "client.context_id":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value ILIKE ?)", "client.context_id", "%"+ct.GetValue()+"%")
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.context_id", ct.GetValue())
			}
		case "client.phone":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value ILIKE ?)", "client.phone", "%"+ct.GetValue()+"%")
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.phone", ct.GetValue())
			}
		case "client.assistant_phone":
			switch ct.GetLogic() {
			case "contains":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value ILIKE ?)", "client.assistant_phone", "%"+ct.GetValue()+"%")
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "client.assistant_phone", ct.GetValue())
			}
		case "disconnect_reason":
			switch ct.GetLogic() {
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metadata WHERE assistant_conversation_metadata.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metadata.key = ? AND assistant_conversation_metadata.value = ?)", "disconnect_reason", ct.GetValue())
			}
		case observability.MetricRecordingInitLatencyMs,
			observability.MetricAnalysisInitLatencyMs,
			observability.MetricAuthenticationInitLatencyMs,
			observability.MetricAuthenticationLatencyMs,
			observability.MetricStorageInitLatencyMs,
			observability.MetricSTTInitLatencyMs,
			observability.MetricSTTLatencyMs,
			observability.MetricSTTTimeToFirstTokenMs,
			observability.MetricSTTTimeToLastTokenMs,
			observability.MetricTTSInitLatencyMs,
			observability.MetricLLMInitLatencyMs,
			observability.MetricDenoiseInitLatencyMs,
			observability.MetricEOSInitLatencyMs,
			observability.MetricVADInitLatencyMs,
			observability.MetricConversationDuration,
			observability.MetricCallDurationMs,
			observability.MetricConversationTTSDuration,
			observability.MetricConversationSTTDuration:
			if !validator.Numeric(ct.GetValue()) {
				return cnt, nil, fmt.Errorf("invalid numeric metric filter value %q for %s", ct.GetValue(), ct.GetKey())
			}
			metricValue, _ := strconv.ParseFloat(ct.GetValue(), 64)
			switch ct.GetLogic() {
			case ">=":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value ~ ? AND CAST(assistant_conversation_metrics.value AS numeric) >= ?)", ct.GetKey(), numericMetricValuePattern, metricValue)
			case "<=":
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value ~ ? AND CAST(assistant_conversation_metrics.value AS numeric) <= ?)", ct.GetKey(), numericMetricValuePattern, metricValue)
			default:
				qry = qry.Where("EXISTS (SELECT 1 FROM assistant_conversation_metrics WHERE assistant_conversation_metrics.assistant_conversation_id = assistant_conversations.id AND assistant_conversation_metrics.name = ? AND assistant_conversation_metrics.value ~ ? AND CAST(assistant_conversation_metrics.value AS numeric) = ?)", ct.GetKey(), numericMetricValuePattern, metricValue)
			}
		}
	}

	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					paginate.GetPage(),
					paginate.GetPageSize(),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&conversations)

	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.GetAll", time.Since(start))
		conversationService.logger.Errorf("not able to find any conversations for assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.GetAll", time.Since(start))
	return cnt, conversations, nil

}
func (conversationService *assistantConversationService) Get(
	ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	assistantConversationId uint64,
	opts *internal_services.GetConversationOption) (*internal_conversation_entity.AssistantConversation, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	conversationService.logger.Debugf("assistantConversationService.Get with options %+v", opts)
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var assistantConversation *internal_conversation_entity.AssistantConversation
	qry := db.
		Where("id = ? AND assistant_id = ? AND project_id = ? AND organization_id = ?",
			assistantConversationId,
			assistantId,
			projectContext.ProjectID,
			projectContext.OrganizationID)

	if opts != nil && opts.InjectMetric {
		qry = qry.
			Preload("Metrics")
	}

	if opts != nil && opts.InjectMetadata {
		qry = qry.
			Preload("Metadatas")
	}

	if opts != nil && opts.InjectArgument {
		qry = qry.
			Preload("Arguments")
	}

	if opts != nil && opts.InjectOption {
		qry = qry.
			Preload("Options")
	}

	tx := qry.First(&assistantConversation)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.Get", time.Since(start))
		conversationService.logger.Errorf("not able to find conversation with id %d  with error %v", assistantConversationId, tx.Error)
		return nil, tx.Error
	}
	var wg sync.WaitGroup
	if opts != nil && opts.InjectRecording {
		wg.Add(1)
		utils.Go(ctx,
			func() {
				defer wg.Done()
				var assistantConversationRecording []*internal_conversation_entity.AssistantConversationRecording
				tx := db.
					Where("assistant_conversation_id = ? AND status = ?", assistantConversationId, type_enums.RECORD_ACTIVE.String()).
					Find(&assistantConversationRecording)
				if tx.Error != nil {
					conversationService.logger.Warnf("unable to find conversation recording with error %+v", tx.Error)
					return
				}

				assistantConversation.Recordings = make([]*internal_conversation_entity.AssistantConversationRecording, 0)
				// updating all to public url
				for _, recording := range assistantConversationRecording {
					assistantUrl, err := conversationService.GetRecordingPublicUrl(ctx, recording.AssistantRecordingUrl)
					if err != nil {
						conversationService.logger.Warnf("unable to get assistant public url %+v", err)
						continue
					}
					userUrl, err := conversationService.GetRecordingPublicUrl(ctx, recording.UserRecordingUrl)
					if err != nil {
						conversationService.logger.Warnf("unable to get user public url %+v", err)
						continue
					}
					if recording.ConversationRecordingUrl != "" {
						conversationUrl, err := conversationService.GetRecordingPublicUrl(ctx, recording.ConversationRecordingUrl)
						if err != nil {
							conversationService.logger.Warnf("unable to get conversation public url %+v", err)
							continue
						}
						recording.ConversationRecordingUrl = *conversationUrl
					}
					recording.AssistantRecordingUrl = *assistantUrl
					recording.UserRecordingUrl = *userUrl
					assistantConversation.Recordings = append(assistantConversation.Recordings, recording)
				}
			})
	}
	wg.Wait()
	conversationService.logger.Benchmark("conversationService.Get", time.Since(start))
	return assistantConversation, nil
}

func (conversationService *assistantConversationService) GetConversation(
	ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	assistantConversationId uint64,
	opts *internal_services.GetConversationOption) (*internal_conversation_entity.AssistantConversation, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var assistantConversation *internal_conversation_entity.AssistantConversation
	qry := db.
		Where("id = ? AND assistant_id = ? AND project_id = ? AND organization_id = ?",
			assistantConversationId,
			assistantId,
			projectContext.ProjectID,
			projectContext.OrganizationID)

	if opts != nil && opts.InjectMetric {
		qry = qry.
			Preload("Metrics")
	}

	if opts != nil && opts.InjectMetadata {
		qry = qry.
			Preload("Metadatas")
	}

	if opts != nil && opts.InjectArgument {
		qry = qry.
			Preload("Arguments")
	}

	if opts != nil && opts.InjectOption {
		qry = qry.
			Preload("Options")
	}

	tx := qry.First(&assistantConversation)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.Get", time.Since(start))
		conversationService.logger.Errorf("not able to find conversation with id %d  with error %v", assistantConversationId, tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.Get", time.Since(start))
	return assistantConversation, nil
}

func (conversationService *assistantConversationService) CreateConversation(
	ctx context.Context,
	auth *types.Authentication,
	identifier string,
	assistantId uint64,
	assistantProviderModelId uint64,
	direction type_enums.ConversationDirection, source utils.RapidaSource) (*internal_conversation_entity.AssistantConversation, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	conversation := &internal_conversation_entity.AssistantConversation{
		Organizational: gorm_models.Organizational{
			ProjectId:      projectContext.ProjectID,
			OrganizationId: projectContext.OrganizationID,
		},
		Identifier:               identifier,
		AssistantId:              assistantId,
		AssistantProviderModelId: assistantProviderModelId,
		Source:                   source,
		Direction:                direction,
		Mutable: gorm_models.Mutable{
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
		},
	}
	tx := db.Create(&conversation)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateConversation", time.Since(start))
		conversationService.logger.Errorf("error while creating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateConversation", time.Since(start))
	return conversation, nil
}

func (conversationService *assistantConversationService) CreateOrUpdateConversationMetadata(
	ctx context.Context,
	auth *types.Authentication,
	assistantId,
	assistantConversationId uint64,
	metadata []*protos.Metadata,
) ([]*internal_conversation_entity.AssistantConversationMetadata, error) {

	start := time.Now()
	//
	if len(metadata) == 0 {
		conversationService.logger.Warnf("error while updating metadata, empty set of argument found")
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	_metadatas := make([]*internal_conversation_entity.AssistantConversationMetadata, 0)
	//
	for _, mt := range metadata {
		_meta := &internal_conversation_entity.AssistantConversationMetadata{
			AssistantConversationId: assistantConversationId,
			Metadata: gorm_models.Metadata{
				Key: mt.Key,
			},
			AssistantId: assistantId,
			Mutable: gorm_models.Mutable{
				CreatedActorType: auth.Actor().Type.String(),
				CreatedActorID:   auth.Actor().ID,
			},
		}
		_meta.SetValue(mt.Value)
		_metadatas = append(_metadatas, _meta)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "key"}},
		DoUpdates: append(clause.AssignmentColumns([]string{"value", "updated_date"}),
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		),
	}).Create(&_metadatas)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationMetadata", time.Since(start))
		conversationService.logger.Errorf("error while CreateOrUpdateConversationMetadata %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationMetadata", time.Since(start))
	return _metadatas, nil
}

func (conversationService *assistantConversationService) CreateOrUpdateConversationOption(ctx context.Context,
	auth *types.Authentication,
	assistantId,
	assistantConversationId uint64,
	opts map[string]interface{}) ([]*internal_conversation_entity.AssistantConversationOption, error) {

	start := time.Now()
	if len(opts) == 0 {
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	options := make([]*internal_conversation_entity.AssistantConversationOption, 0)

	for k, o := range opts {
		option := &internal_conversation_entity.AssistantConversationOption{
			AssistantConversationId: assistantConversationId,
			Metadata: gorm_models.Metadata{
				Key: k,
			},
			AssistantId: assistantId,
			Mutable: gorm_models.Mutable{
				CreatedActorType: auth.Actor().Type.String(),
				CreatedActorID:   auth.Actor().ID,
			},
		}
		option.SetValue(o)
		options = append(options, option)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "key"}},
		DoUpdates: append(clause.AssignmentColumns([]string{"value", "updated_date"}),
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		),
	}).Create(&options)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationOptions", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation argument %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationOptions", time.Since(start))
	return options, nil

}

func (conversationService *assistantConversationService) CreateOrUpdateConversationArgument(ctx context.Context,
	auth *types.Authentication,
	assistantId,
	assistantConversationId uint64,
	arguments map[string]interface{},
) ([]*internal_conversation_entity.AssistantConversationArgument, error) {

	start := time.Now()
	if len(arguments) == 0 {
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	_arguments := make([]*internal_conversation_entity.AssistantConversationArgument, 0)

	for k, arg := range arguments {
		ag := &internal_conversation_entity.AssistantConversationArgument{
			AssistantConversationId: assistantConversationId,
			Argument: gorm_models.Argument{
				Name: k,
			},
			Mutable: gorm_models.Mutable{
				CreatedActorType: auth.Actor().Type.String(),
				CreatedActorID:   auth.Actor().ID,
			},
		}
		ag.SetValue(arg)
		_arguments = append(_arguments, ag)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: append(clause.AssignmentColumns([]string{"value", "updated_date"}),
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		),
	}).Create(&_arguments)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyConversationArgument", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation argument %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyConversationArgument", time.Since(start))
	return _arguments, nil
}

/**
* NOTE
* Feedback about the conversation
* Once the conversation is over the user will be prompted about conversation quality and xyz defined by the client
* client push the feedback as string and it will be stored as metrics later there might be different kind of feedback client can ask
**/
func (conversationService *assistantConversationService) CreateOrUpdateConversationMetrics(
	ctx context.Context,
	auth *types.Authentication,
	assistantId,
	assistantConversationId uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_entity.AssistantConversationMetric, error) {

	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrs := make([]*internal_conversation_entity.AssistantConversationMetric, 0)
	for _, mtr := range metrics {
		_mtr := &internal_conversation_entity.AssistantConversationMetric{
			Metric: gorm_models.Metric{
				Name:        mtr.GetName(),
				Value:       mtr.GetValue(),
				Description: mtr.GetDescription(),
			},
			AssistantConversationId: assistantConversationId,
			AssistantId:             assistantId,
			Mutable: gorm_models.Mutable{
				CreatedActorType: auth.Actor().Type.String(),
				CreatedActorID:   auth.Actor().ID,
			},
		}

		mtrs = append(mtrs, _mtr)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: append(clause.AssignmentColumns([]string{"value", "description", "updated_date"}),
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		),
	}).Create(&mtrs)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationMetrics", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateOrUpdateConversationMetrics", time.Since(start))
	return mtrs, nil
}

func (conversationService *assistantConversationService) CreateCustomConversationMetric(
	ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	assistantConversationId uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_entity.AssistantConversationMetric, error) {

	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrx := make([]*internal_conversation_entity.AssistantConversationMetric, 0)
	for _, v := range metrics {
		metric := &internal_conversation_entity.AssistantConversationMetric{
			Metric: gorm_models.Metric{
				Name:        fmt.Sprintf("%s.%s", "custom", v.GetName()),
				Description: v.GetDescription(),
				Value:       v.GetValue(),
			},
			AssistantId:             assistantId,
			AssistantConversationId: assistantConversationId,
			Mutable: gorm_models.Mutable{
				CreatedActorType: auth.Actor().Type.String(),
				CreatedActorID:   auth.Actor().ID,
			},
		}

		mtrx = append(mtrx, metric)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: append(clause.AssignmentColumns([]string{"value", "description", "updated_date"}),
			clause.Assignment{Column: clause.Column{Name: "updated_actor_type"}, Value: auth.Actor().Type.String()},
			clause.Assignment{Column: clause.Column{Name: "updated_actor_id"}, Value: auth.Actor().ID},
		),
	}).Create(&mtrx)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateCustomConversationMetric", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateCustomConversationMetric", time.Since(start))
	return mtrx, nil
}

func (conversationService *assistantConversationService) CreateConversationRecording(
	ctx context.Context,
	auth *types.Authentication,
	assistantId,
	assistantConversationId uint64,
	user, assistant, conversation []byte,
) (*internal_conversation_entity.AssistantConversationRecording, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := conversationService.postgres.DB(ctx)

	s3Prefix := conversationService.ObjectPrefix(projectContext.OrganizationID, projectContext.ProjectID)
	recordingId := gorm_generator.ID()

	userKey := conversationService.ObjectKey(s3Prefix, assistantConversationId, fmt.Sprintf("user-%d.wav", recordingId))
	assistantKey := conversationService.ObjectKey(s3Prefix, assistantConversationId, fmt.Sprintf("assistant-%d.wav", recordingId))
	conversationKey := conversationService.ObjectKey(s3Prefix, assistantConversationId, fmt.Sprintf("conversation-%d.wav", recordingId))

	recordings := []struct {
		label string
		key   string
		data  []byte
	}{
		{label: "user", key: userKey, data: user},
		{label: "assistant", key: assistantKey, data: assistant},
		{label: "conversation", key: conversationKey, data: conversation},
	}
	for _, recording := range recordings {
		result := conversationService.storage.Store(ctx, recording.key, recording.data)
		if result.Error != nil {
			conversationService.logger.Benchmark("conversationService.CreateConversationRecording", time.Since(start))
			conversationService.logger.Errorf("error while storing %s conversation recording %s: %v", recording.label, recording.key, result.Error)
			return nil, fmt.Errorf("store %s recording: %w", recording.label, result.Error)
		}
	}

	conversationRecording := &internal_conversation_entity.AssistantConversationRecording{
		Audited: gorm_models.Audited{
			Id: recordingId,
		},
		Organizational: gorm_models.Organizational{
			ProjectId:      projectContext.ProjectID,
			OrganizationId: projectContext.OrganizationID,
		},
		AssistantId:              assistantId,
		AssistantConversationId:  assistantConversationId,
		AssistantRecordingUrl:    assistantKey,
		UserRecordingUrl:         userKey,
		ConversationRecordingUrl: conversationKey,
		Mutable: gorm_models.Mutable{
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
		},
	}
	tx := db.Create(&conversationRecording)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateConversationRecording", time.Since(start))
		conversationService.logger.Errorf("error while creating conversation recording %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateConversationRecording", time.Since(start))
	return conversationRecording, nil
}

func (eService *assistantConversationService) ObjectKey(keyPrefix string, conversationId uint64, objName string) string {
	return fmt.Sprintf("%s/%d/%s", keyPrefix, conversationId, objName)
}

func (eService *assistantConversationService) ObjectPrefix(orgId, projectId uint64) string {
	return fmt.Sprintf("%d/%d/recording", orgId, projectId)
}

func (eService *assistantConversationService) GetRecordingPublicUrl(ctx context.Context, key string) (*string, error) {
	output := eService.storage.GetUrl(ctx, key)
	if output.Error != nil {
		return nil, output.Error
	}
	return utils.Ptr(output.CompletePath), nil
}
