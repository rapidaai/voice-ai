// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"fmt"
	"sync"
	"time"

	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
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
	auth types.SimplePrinciple,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate, opts *internal_services.GetConversationOption) (int64, []*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversations []*internal_conversation_gorm.AssistantConversation
		cnt           int64
	)
	qry := db.Model(internal_conversation_gorm.AssistantConversation{})
	qry = qry.
		Where("assistant_id = ? AND organization_id = ? AND project_id = ?", assistantId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId())

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

	if opts != nil && opts.InjectTelephonyEvent {
		qry = qry.
			Preload("TelephonyEvents")
	}

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
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantConversationId uint64,
	opts *internal_services.GetConversationOption) (*internal_conversation_gorm.AssistantConversation, error) {
	conversationService.logger.Debugf("assistantConversationService.Get with options %+v", opts)
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var assistantConversation *internal_conversation_gorm.AssistantConversation
	qry := db.
		Where("id = ? AND assistant_id = ? AND project_id = ? AND organization_id = ?",
			assistantConversationId,
			assistantId,
			*auth.GetCurrentProjectId(),
			*auth.GetCurrentOrganizationId())

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
				var assistantConversationRecording []*internal_conversation_gorm.AssistantConversationRecording
				tx := db.
					Where("assistant_conversation_id = ? AND status = ?", assistantConversationId, type_enums.RECORD_ACTIVE.String()).
					Find(&assistantConversationRecording)
				if tx.Error != nil {
					conversationService.logger.Warnf("unable to find conversation recording with error %+v", tx.Error)
					return
				}

				assistantConversation.Recordings = make([]*internal_conversation_gorm.AssistantConversationRecording, 0)
				// updating all to public url
				for _, recording := range assistantConversationRecording {
					pUrl, err := conversationService.GetRecordingPublicUrl(ctx, recording.RecordingUrl)
					if err != nil {
						conversationService.logger.Warnf("unable to get public url %+v", tx.Error)
						continue
					}
					recording.RecordingUrl = *pUrl
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
	auth types.SimplePrinciple,
	identifier string,
	assistantId uint64,
	assistantConversationId uint64,
	opts *internal_services.GetConversationOption) (*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var assistantConversation *internal_conversation_gorm.AssistantConversation
	qry := db.
		Where("id = ? AND identifier = ? AND assistant_id = ? AND project_id = ? AND organization_id = ?",
			assistantConversationId,
			identifier,
			assistantId,
			*auth.GetCurrentProjectId(),
			*auth.GetCurrentOrganizationId())

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
	auth types.SimplePrinciple,
	identifier string,
	assistantId uint64,
	assistantProviderModelId uint64,
	direction type_enums.ConversationDirection, source utils.RapidaSource) (*internal_conversation_gorm.AssistantConversation, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	conversation := &internal_conversation_gorm.AssistantConversation{
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		Identifier:               identifier,
		AssistantId:              assistantId,
		AssistantProviderModelId: assistantProviderModelId,
		Source:                   source,
		Direction:                direction,
	}
	if auth.GetUserId() != nil {
		conversation.Mutable.CreatedBy = *auth.GetUserId()
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

func (conversationService *assistantConversationService) ApplyConversationMetadata(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId,
	assistantConversationId uint64,
	metadata []*types.Metadata,
) ([]*internal_conversation_gorm.AssistantConversationMetadata, error) {
	start := time.Now()
	//
	if len(metadata) == 0 {
		conversationService.logger.Warnf("error while updating metadata, empty set of argument found")
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	_metadatas := make([]*internal_conversation_gorm.AssistantConversationMetadata, 0)
	//
	for _, mt := range metadata {
		_meta := &internal_conversation_gorm.AssistantConversationMetadata{
			AssistantConversationId: assistantConversationId,
			Metadata: gorm_models.Metadata{
				Key: mt.Key,
			},
			AssistantId: assistantId,
		}
		_meta.SetValue(mt.Value)
		if auth.GetUserId() != nil {
			_meta.UpdatedBy = *auth.GetUserId()
			_meta.CreatedBy = *auth.GetUserId()
		}
		_metadatas = append(_metadatas, _meta)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&_metadatas)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyConversationMetadata", time.Since(start))
		conversationService.logger.Errorf("error while ApplyConversationMetadata %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyConversationMetadata", time.Since(start))
	return _metadatas, nil
}

func (conversationService *assistantConversationService) ApplyConversationOption(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId,
	assistantConversationId uint64,
	opts map[string]interface{}) ([]*internal_conversation_gorm.AssistantConversationOption, error) {
	start := time.Now()
	if len(opts) == 0 {
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	options := make([]*internal_conversation_gorm.AssistantConversationOption, 0)

	for k, o := range opts {
		option := &internal_conversation_gorm.AssistantConversationOption{
			AssistantConversationId: assistantConversationId,
			Metadata: gorm_models.Metadata{
				Key: k,
			},
			AssistantId: assistantId,
		}
		option.SetValue(o)
		if auth.GetUserId() != nil {
			option.CreatedBy = *auth.GetUserId()
			option.UpdatedBy = *auth.GetUserId()
		}
		options = append(options, option)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&options)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyConversationOptions", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation argument %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyConversationOptions", time.Since(start))
	return options, nil

}

func (conversationService *assistantConversationService) ApplyConversationArgument(ctx context.Context,
	auth types.SimplePrinciple,
	assistantId,
	assistantConversationId uint64,
	arguments map[string]interface{},
) ([]*internal_conversation_gorm.AssistantConversationArgument, error) {
	start := time.Now()
	if len(arguments) == 0 {
		return nil, nil
	}

	db := conversationService.postgres.DB(ctx)
	_arguments := make([]*internal_conversation_gorm.AssistantConversationArgument, 0)

	for k, arg := range arguments {
		ag := &internal_conversation_gorm.AssistantConversationArgument{
			AssistantConversationId: assistantConversationId,
			Argument: gorm_models.Argument{
				Name: k,
			},
		}
		ag.SetValue(arg)
		if auth.GetUserId() != nil {
			ag.UpdatedBy = *auth.GetUserId()
		}
		_arguments = append(_arguments, ag)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
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
func (conversationService *assistantConversationService) ApplyConversationMetrics(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId,
	assistantConversationId uint64,
	metrics []*types.Metric,
) ([]*internal_conversation_gorm.AssistantConversationMetric, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrs := make([]*internal_conversation_gorm.AssistantConversationMetric, 0)
	for _, mtr := range metrics {
		_mtr := &internal_conversation_gorm.AssistantConversationMetric{
			Metric: gorm_models.Metric{
				Name:        mtr.GetName(),
				Value:       mtr.GetValue(),
				Description: mtr.GetDescription(),
			},
			AssistantConversationId: assistantConversationId,
		}

		if auth.GetUserId() != nil {
			_mtr.UpdatedBy = *auth.GetUserId()
			_mtr.CreatedBy = *auth.GetUserId()
		}
		mtrs = append(mtrs, _mtr)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
	}).Create(&mtrs)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyConversationMetrics", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyConversationMetrics", time.Since(start))
	return mtrs, nil
}

/* */
func (conversationService *assistantConversationService) CreateConversationMetric(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantConversationId uint64,
	name, description, value string,
) (*internal_conversation_gorm.AssistantConversationMetric, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	metric := &internal_conversation_gorm.AssistantConversationMetric{
		Metric: gorm_models.Metric{
			Name:        fmt.Sprintf("%s.%s", "custom", name),
			Description: description,
			Value:       value,
		},
		AssistantId:             assistantId,
		AssistantConversationId: assistantConversationId,
	}

	if auth.GetUserId() != nil {
		metric.UpdatedBy = *auth.GetUserId()
		metric.CreatedBy = *auth.GetUserId()
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
	}).Create(&metric)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateConversationMetric", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.CreateConversationMetric", time.Since(start))
	return metric, nil
}

func (conversationService *assistantConversationService) CreateCustomConversationMetric(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantConversationId uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_gorm.AssistantConversationMetric, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrx := make([]*internal_conversation_gorm.AssistantConversationMetric, 0)
	for _, v := range metrics {
		metric := &internal_conversation_gorm.AssistantConversationMetric{
			Metric: gorm_models.Metric{
				Name:        fmt.Sprintf("%s.%s", "custom", v.GetName()),
				Description: v.GetDescription(),
				Value:       v.GetValue(),
			},
			AssistantId:             assistantId,
			AssistantConversationId: assistantConversationId,
		}

		if auth.GetUserId() != nil {
			metric.UpdatedBy = *auth.GetUserId()
			metric.CreatedBy = *auth.GetUserId()
		}
		mtrx = append(mtrx, metric)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "assistant_conversation_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
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
	auth types.SimplePrinciple,
	assistantId,
	assistantConversationId uint64,
	body []byte,
) (*internal_conversation_gorm.AssistantConversationRecording, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)

	s3Prefix := conversationService.ObjectPrefix(*auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId())
	recordingId := gorm_generator.ID()

	key := conversationService.ObjectKey(s3Prefix, recordingId, fmt.Sprintf("recording-%d.wav", assistantConversationId))
	conversationService.storage.Store(ctx, key, body)

	conversationRecording := &internal_conversation_gorm.AssistantConversationRecording{
		Audited: gorm_models.Audited{
			Id: recordingId,
		},
		Organizational: gorm_models.Organizational{
			ProjectId:      *auth.GetCurrentProjectId(),
			OrganizationId: *auth.GetCurrentOrganizationId(),
		},
		AssistantId:             assistantId,
		AssistantConversationId: assistantConversationId,
		RecordingUrl:            key,
	}
	if auth.GetUserId() != nil {
		conversationRecording.Mutable.CreatedBy = *auth.GetUserId()
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
	return fmt.Sprintf("%s/%d__%s", keyPrefix, conversationId, objName)
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

func (eService *assistantConversationService) ApplyConversationTelephonyEvent(
	ctx context.Context,
	auth types.SimplePrinciple,
	telephony string,
	assistantId,
	assistantConversationId uint64,
	events []*types.Event,
) ([]*internal_conversation_gorm.AssistantConversationTelephonyEvent, error) {
	start := time.Now()
	db := eService.postgres.DB(ctx)

	telephonyEvent := make([]*internal_conversation_gorm.AssistantConversationTelephonyEvent, 0)
	for _, v := range events {
		tE := &internal_conversation_gorm.AssistantConversationTelephonyEvent{
			AssistantConversationId: assistantConversationId,
			Event: *gorm_models.NewEvent(
				v.EventType,
				v.Payload,
			),
			Provider:    telephony,
			AssistantId: assistantId,
		}

		telephonyEvent = append(telephonyEvent, tE)
	}

	tx := db.Create(&telephonyEvent)
	if tx.Error != nil {
		eService.logger.Benchmark("eService.CreateConversationTelephonyEvent", time.Since(start))
		eService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	eService.logger.Benchmark("eService.CreateConversationTelephonyEvent", time.Since(start))
	return telephonyEvent, nil
}
