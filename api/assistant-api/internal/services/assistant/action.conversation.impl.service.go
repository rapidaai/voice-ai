package internal_assistant_service

import (
	"context"
	"fmt"
	"time"

	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/conversations"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

func (conversationService *assistantConversationService) GetAllMessageActions(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	criterias []*lexatic_backend.Criteria,
	paginate *lexatic_backend.Paginate,
	ordering *lexatic_backend.Ordering,
) (int64, []*internal_conversation_gorm.AssistantConversationAction, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	var (
		conversationMessage []*internal_conversation_gorm.AssistantConversationAction
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

	qry := db.
		Model(internal_conversation_gorm.AssistantConversationAction{}).
		Where("assistant_conversation_id = ? ", assistantConversationId)
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
		Order(orderClause).Find(&conversationMessage)

	if tx.Error != nil {
		conversationService.logger.Benchmark("assistantService.GetAllMessageActions", time.Since(start))
		conversationService.logger.Errorf("not able to find any conversations for assistant %v", tx.Error)
		return cnt, nil, tx.Error
	}
	conversationService.logger.Benchmark("assistantService.GetAllMessageActions", time.Since(start))
	return cnt, conversationMessage, nil
}

func (conversationService *assistantConversationService) CreateLLMAction(
	ctx context.Context,
	auth types.SimplePrinciple,
	conversationId uint64,
	messageId string,
	in, out *types.Message, metrics []*types.Metric) (*internal_conversation_gorm.AssistantConversationAction, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	aca := &internal_conversation_gorm.AssistantConversationAction{
		AssistantConversationMessageId: messageId,
		AssistantConversationId:        conversationId,
	}
	aca.SetLLMCall(in, out)
	tx := db.Create(aca)
	if tx.Error != nil {
		conversationService.logger.Benchmark("assistantService.CreateLLMAction", time.Since(start))
		conversationService.logger.Errorf("error while creating message action %v", tx.Error)
		return nil, tx.Error
	}
	_, err := conversationService.ApplyToolMetrics(ctx, auth, conversationId, aca.Id, messageId, metrics)
	if err != nil {
		conversationService.logger.Errorf("error while creating message action metrics %v", tx.Error)
	}
	conversationService.logger.Benchmark("assistantService.CreateLLMAction", time.Since(start))
	return aca, nil

}
func (conversationService *assistantConversationService) CreateToolAction(
	ctx context.Context,
	auth types.SimplePrinciple,
	conversationId uint64,
	messageId string,
	in, out map[string]interface{}, metrics []*types.Metric) (*internal_conversation_gorm.AssistantConversationAction, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	aca := &internal_conversation_gorm.AssistantConversationAction{
		AssistantConversationMessageId: messageId,
		AssistantConversationId:        conversationId,
	}
	aca.SetToolCall(in, out)
	tx := db.Create(aca)
	if tx.Error != nil {
		conversationService.logger.Benchmark("assistantService.CreateToolAction", time.Since(start))
		conversationService.logger.Errorf("error while creating message action %v", tx.Error)
		return nil, tx.Error
	}
	_, err := conversationService.ApplyToolMetrics(ctx, auth, conversationId, aca.Id, messageId, metrics)
	if err != nil {
		conversationService.logger.Errorf("error while creating message action metrics %v", tx.Error)
	}
	conversationService.logger.Benchmark("assistantService.CreateToolAction", time.Since(start))
	return aca, nil
}

func (conversationService *assistantConversationService) ApplyToolMetrics(
	ctx context.Context,
	auth types.SimplePrinciple,
	assistantConversationId uint64,
	assistantConversationActionId uint64,
	assistantConversationMessageId string,
	metrics []*types.Metric,
) ([]*internal_conversation_gorm.AssistantConversationActionMetric, error) {
	start := time.Now()
	db := conversationService.postgres.DB(ctx)
	mtrs := make([]*internal_conversation_gorm.AssistantConversationActionMetric, 0)
	for _, mtr := range metrics {
		_mtr := &internal_conversation_gorm.AssistantConversationActionMetric{
			Metric: gorm_models.Metric{
				Name:        mtr.GetName(),
				Value:       mtr.GetValue(),
				Description: mtr.GetDescription(),
			},
			AssistantConversationId:        assistantConversationId,
			AssistantConversationMessageId: assistantConversationMessageId,
			AssistantConversationActionId:  assistantConversationActionId,
		}
		if auth.GetUserId() != nil {
			_mtr.UpdatedBy = *auth.GetUserId()
			_mtr.CreatedBy = *auth.GetUserId()
		}
		mtrs = append(mtrs, _mtr)
	}
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}, {Name: "assistant_conversation_action_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value", "description",
			"updated_by", "updated_date"}),
	}).Create(&mtrs)
	if tx.Error != nil {
		conversationService.logger.Benchmark("assistantService.ApplyToolMetrics", time.Since(start))
		conversationService.logger.Errorf("error while updating conversation %v", tx.Error)
		return nil, tx.Error
	}
	conversationService.logger.Benchmark("assistantService.ApplyToolMetrics", time.Since(start))
	return mtrs, nil
}
