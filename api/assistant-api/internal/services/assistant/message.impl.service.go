// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"fmt"
	"time"

	internal_message_gorm "github.com/rapidaai/api/assistant-api/internal/entity/messages"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

func (conversationService *assistantConversationService) GetAllConversationMessage(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
	ordering *protos.Ordering, opts *internal_services.GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversationMessage []*internal_message_gorm.AssistantConversationMessage
		cnt                 int64
		orderClause         = clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   false,
		}
	)
	if ordering != nil {
		orderClause = clause.OrderByColumn{
			Column: clause.Column{Name: ordering.Column},
			Desc:   false,
		}
		if ordering.Order == "desc" {
			orderClause.Desc = true
		}
	}

	qry := db.Model(internal_message_gorm.AssistantConversationMessage{})
	qry = qry.Where("assistant_conversation_id = ?", assistantConversationId)

	if opts != nil && opts.InjectMetadata {
		qry = qry.Preload("Metadatas")
	}
	if opts != nil && opts.InjectMetric {
		qry = qry.Preload("Metrics")
	}

	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}
	tx := qry.Debug().
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(orderClause).Find(&conversationMessage)

	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.GetAllConversationMessage", time.Since(start))
		conversationService.logger.Errorf("Unable to get all conversation message with error %v", tx.Error)
		return cnt, nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.GetAllConversationMessage", time.Since(start))
	return cnt, conversationMessage, nil
}

func (conversationService *assistantConversationService) GetAllAssistantMessage(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
	ordering *protos.Ordering,
	opts *internal_services.GetMessageOption,
) (int64, []*internal_message_gorm.AssistantConversationMessage, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversationMessage []*internal_message_gorm.AssistantConversationMessage
		cnt                 int64
		orderClause         = clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   false,
		}
	)

	if ordering != nil {
		orderClause = clause.OrderByColumn{
			Column: clause.Column{Name: ordering.Column},
			Desc:   false,
		}
		if ordering.Order == "desc" {
			orderClause.Desc = true
		}
	}

	cols := []string{
		"assistant_conversation_messages.message_id",
		"assistant_conversation_messages.assistant_conversation_id",
		"assistant_conversation_messages.assistant_id",
		"assistant_conversation_messages.assistant_provider_model_id",
		"assistant_conversation_messages.id",
		"assistant_conversation_messages.created_date",
		"assistant_conversation_messages.updated_date",
		"assistant_conversation_messages.status",
		"assistant_conversation_messages.source",
		//
		"assistant_conversations.identifier",
		"assistant_conversations.assistant_id",
		"assistant_conversations.assistant_provider_model_id",
		"assistant_conversations.name",
		"assistant_conversations.project_id",
		"assistant_conversations.organization_id",
		"assistant_conversations.source",
		"assistant_conversations.status",
		"assistant_conversations.direction",
	}

	qry := db.Model(internal_message_gorm.AssistantConversationMessage{})
	qry = qry.
		Joins("JOIN assistant_conversations ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id").
		Where("assistant_conversations.assistant_id = ? AND assistant_conversations.organization_id = ? AND assistant_conversations.project_id = ?", assistantId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId())

	if opts != nil && opts.InjectRequest {
		cols = append(cols, "assistant_conversation_messages.request")
	}
	if opts != nil && opts.InjectResponse {
		cols = append(cols, "assistant_conversation_messages.response")
	}

	if opts != nil && opts.InjectMetadata {
		qry = qry.Preload("Metadatas")
	}
	if opts != nil && opts.InjectMetric {
		qry = qry.Preload("Metrics")
	}
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}

	tx := qry.Debug().Select(cols).
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(orderClause).Find(&conversationMessage)

	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.GetAllAssistantMessage", time.Since(start))
		conversationService.logger.Errorf("not able to find any conversations for assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.GetAllAssistantMessage", time.Since(start))
	return cnt, conversationMessage, nil
}

func (conversationService *assistantConversationService) GetAllMessage(
	ctx context.Context,
	auth types.SimplePrinciple,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
	ordering *protos.Ordering,
	opts *internal_services.GetMessageOption,
) (int64, []*internal_message_gorm.AssistantConversationMessage, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversationMessage []*internal_message_gorm.AssistantConversationMessage
		cnt                 int64
		orderClause         = clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   false,
		}
	)

	if ordering != nil {
		orderClause = clause.OrderByColumn{
			Column: clause.Column{Name: ordering.Column},
			Desc:   false,
		}
		if ordering.Order == "desc" {
			orderClause.Desc = true
		}
	}

	cols := []string{
		"assistant_conversation_messages.message_id",
		"assistant_conversation_messages.assistant_conversation_id",
		"assistant_conversation_messages.assistant_id",
		"assistant_conversation_messages.assistant_provider_model_id",
		"assistant_conversation_messages.id",
		"assistant_conversation_messages.created_date",
		"assistant_conversation_messages.updated_date",
		"assistant_conversation_messages.status",
		"assistant_conversation_messages.source",
		//
		"assistant_conversations.identifier",
		"assistant_conversations.assistant_id",
		"assistant_conversations.assistant_provider_model_id",
		"assistant_conversations.name",
		"assistant_conversations.project_id",
		"assistant_conversations.organization_id",
		"assistant_conversations.source",
		"assistant_conversations.status",
		"assistant_conversations.direction",
	}

	qry := db.Model(internal_message_gorm.AssistantConversationMessage{})
	qry = qry.
		Joins("JOIN assistant_conversations ON assistant_conversations.id = assistant_conversation_messages.assistant_conversation_id").
		Where("assistant_conversations.organization_id = ? AND assistant_conversations.project_id = ?", *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId())

	if opts != nil && opts.InjectRequest {
		cols = append(cols, "assistant_conversation_messages.request")
	}
	if opts != nil && opts.InjectResponse {
		cols = append(cols, "assistant_conversation_messages.response")
	}

	if opts != nil && opts.InjectMetadata {
		qry = qry.Preload("Metadatas")
	}
	if opts != nil && opts.InjectMetric {
		qry = qry.Preload("Metrics")
	}
	// if opts != nil && opts.InjectStage {
	// 	qry = qry.Preload("Stages")
	// }
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}

	tx := qry.Debug().Select(cols).
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(orderClause).Find(&conversationMessage)

	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.GetAllMessage", time.Since(start))
		conversationService.logger.Errorf("not able to find any messages for project %v", tx.Error)
		return cnt, nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.GetAllMessage", time.Since(start))
	return cnt, conversationMessage, nil
}

func (conversationService *assistantConversationService) UpdateConversationMessage(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	assistantConversationMessageId string,
	message *types.Message,
	status type_enums.RecordState,
) (*internal_message_gorm.AssistantConversationMessage, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	conversation := &internal_message_gorm.AssistantConversationMessage{
		AssistantConversationId: assistantConversationId,
		MessageId:               assistantConversationMessageId,
		Mutable: gorm_models.Mutable{
			Status: status,
		},
	}
	if auth.GetUserId() != nil {
		conversation.UpdatedBy = *auth.GetUserId()
	}
	conversation.SetResponse(message)
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "message_id"}, {Name: "assistant_conversation_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"response",
			"updated_by", "updated_date"}),
	}).Create(&conversation)
	// Where("message_id = ? AND assistant_conversation_id = ? ", assistantConversationMessageId, assistantConversationId).Updates(conversation)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.UpdateConversationMessage", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation message %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.UpdateConversationMessage", time.Since(start))
	return conversation, nil
}

func (conversationService *assistantConversationService) CreateConversationMessage(
	ctx context.Context,
	auth types.SimplePrinciple,
	source utils.RapidaSource,
	messageId string,
	assistantId, assistantProviderModelId,
	assistantConversationId uint64,
	message *types.Message,
) (*internal_message_gorm.AssistantConversationMessage, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	conversationMessage := &internal_message_gorm.AssistantConversationMessage{
		AssistantConversationId:  assistantConversationId,
		AssistantId:              assistantId,
		AssistantProviderModelId: assistantProviderModelId,
		MessageId:                messageId,
		Source:                   source.Get(),
		Mutable: gorm_models.Mutable{
			CreatedBy: 99,
		},
	}
	if auth.GetUserId() != nil {
		conversationMessage.CreatedBy = *auth.GetUserId()
		conversationMessage.UpdatedBy = *auth.GetUserId()
	}
	conversationMessage.SetRequest(message)
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "message_id"}, {Name: "assistant_conversation_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"request",
			"updated_by", "updated_date"}),
	}).Create(&conversationMessage)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.CreateConversationMessage", time.Since(start))
		conversationService.logger.Errorf("error while creating conversation %v", tx.Error)
		return nil, tx.Error
	}

	conversationService.logger.Benchmark("conversationService.CreateConversationMessage", time.Since(start))
	return conversationMessage, nil
}

func (conversationService *assistantConversationService) ApplyMessageMetadata(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	assistantConversationMessageId string,
	metadata map[string]interface{},
) ([]*internal_message_gorm.AssistantConversationMessageMetadata, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	_mtdata := make([]*internal_message_gorm.AssistantConversationMessageMetadata, 0)
	for k, m := range metadata {
		_mtd := &internal_message_gorm.AssistantConversationMessageMetadata{
			AssistantConversationId:        assistantConversationId,
			AssistantConversationMessageId: assistantConversationMessageId,
			Metadata: gorm_models.Metadata{
				Key: k,
			},
		}
		_mtd.SetValue(m)
		if auth.GetUserId() != nil {
			_mtd.UpdatedBy = *auth.GetUserId()
			_mtd.CreatedBy = *auth.GetUserId()
		}
		_mtdata = append(_mtdata, _mtd)
	}
	if len(_mtdata) == 0 {
		return nil, fmt.Errorf("illegal state for metadata, trying to insert empty slice of metadata")
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}, {Name: "assistant_conversation_message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"updated_by", "updated_date"}),
	}).Create(&_mtdata)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyMessageMetadata", time.Since(start))
		conversationService.logger.Errorf("error while applying message metadata %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyMessageMetadata", time.Since(start))
	return _mtdata, nil
}

/**
*
*
 */

func (conversationService *assistantConversationService) ApplyMessageMetrics(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	assistantConversationMessageId string,
	metrics []*types.Metric,
) ([]*internal_message_gorm.AssistantConversationMessageMetric, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrs := make([]*internal_message_gorm.AssistantConversationMessageMetric, 0)
	for _, mtr := range metrics {
		_mtr := &internal_message_gorm.AssistantConversationMessageMetric{
			Metric: gorm_models.Metric{
				Name:        mtr.GetName(),
				Value:       mtr.GetValue(),
				Description: mtr.GetDescription(),
			},
			AssistantConversationId:        assistantConversationId,
			AssistantConversationMessageId: assistantConversationMessageId,
		}
		if auth.GetUserId() != nil {
			_mtr.UpdatedBy = *auth.GetUserId()
			_mtr.CreatedBy = *auth.GetUserId()
		}
		mtrs = append(mtrs, _mtr)
	}

	if len(mtrs) == 0 {
		return nil, fmt.Errorf("illegal state for metrics, trying to insert empty slice of metric")
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}, {Name: "assistant_conversation_message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
	}).Create(&mtrs)
	if tx.Error != nil {
		conversationService.logger.Benchmark("conversationService.ApplyMessageMetrics", time.Since(start))
		conversationService.logger.Errorf("error while applying message metrics %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("conversationService.ApplyMessageMetrics", time.Since(start))
	return mtrs, nil
}

// func (conversationService *assistantConversationService) ApplyMessageStages(
// 	ctx context.Context,
// 	auth types.SimplePrinciple,
// 	assistantConversationId uint64,
// 	assistantConversationMessageId string,
// 	stages []*protos.AssistantMessageStage,
// ) ([]*internal_message_gorm.AssistantConversationMessageStage, error) {
// 	start := time.Now()
// 	db := conversationService.postgres.DB(ctx)
// 	mtrs := make([]*internal_message_gorm.AssistantConversationMessageStage, 0)
// 	for _, stg := range stages {
// 		_mtr := &internal_message_gorm.AssistantConversationMessageStage{
// 			Stage:                          stg.GetStage(),
// 			StageName:                      stg.GetStage(),
// 			AdditionalData:                 stg.GetAdditionalData(),
// 			TimeTaken:                      stg.GetTimetaken(),
// 			LifecycleId:                    stg.GetLifecycleId(),
// 			StartTimestamp:                 gorm_models.TimeWrapper(stg.GetStartTimestamp().AsTime()),
// 			EndTimestamp:                   gorm_models.TimeWrapper(stg.GetEndTimestamp().AsTime()),
// 			AssistantConversationId:        assistantConversationId,
// 			AssistantConversationMessageId: assistantConversationMessageId,
// 		}

// 		if auth.GetUserId() != nil {
// 			_mtr.UpdatedBy = *auth.GetUserId()
// 			_mtr.CreatedBy = *auth.GetUserId()
// 		}
// 		mtrs = append(mtrs, _mtr)
// 	}

// 	if len(mtrs) == 0 {
// 		return nil, fmt.Errorf("illegal state for stages, trying to insert empty slice of stage")
// 	}

// 	tx := db.Clauses(clause.OnConflict{
// 		Columns: []clause.Column{{Name: "stage"}, {Name: "assistant_conversation_message_id"}},
// 		DoUpdates: clause.AssignmentColumns([]string{
// 			"start_timestamp", "end_timestamp", "time_taken", "additional_data",
// 			"updated_by", "updated_date"}),
// 	}).Create(&mtrs)
// 	if tx.Error != nil {
// 		conversationService.logger.Benchmark("conversationService.ApplyMessageStages", time.Since(start))
// 		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
// 		return nil, tx.Error
// 	}
// 	conversationService.logger.Benchmark("conversationService.ApplyMessageStages", time.Since(start))
// 	return mtrs, nil
// }
