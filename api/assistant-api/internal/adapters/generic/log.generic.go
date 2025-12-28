// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	protos "github.com/rapidaai/protos"
)

const CONVERSACTION_PAGE_HISTORY uint32 = 50

func (kr *GenericRequestor) CreateKnowledgeLog(knowledgeId uint64, retrievalMethod string,
	topK uint32,
	scoreThreshold float32,
	documentCount int,
	timeTaken int64,
	additionalData map[string]string,
	status type_enums.RecordState,
	request, response []byte) error {
	_, err := kr.knowledgeService.CreateLog(
		kr.Context(),
		kr.Auth(),
		knowledgeId,
		retrievalMethod,
		topK,
		scoreThreshold,
		documentCount,
		timeTaken,
		additionalData,
		status,
		request, response,
	)
	return err
}

func (cr *GenericRequestor) CreateWebhookLog(

	webhookID uint64, httpUrl, httpMethod, event string,
	responseStatus int64,
	timeTaken int64,
	retryCount uint32,
	status type_enums.RecordState,
	request, response []byte) error {
	_, err := cr.webhookService.CreateLog(
		cr.ctx,
		cr.auth,
		webhookID,
		cr.assistant.Id,
		cr.assistantConversation.Id,
		httpUrl, httpMethod,
		event,
		responseStatus,
		timeTaken,
		retryCount,
		status,
		request, response)
	return err
}

func (cr *GenericRequestor) GetConversationLogs() []*protos.Message {
	messages := make([]*protos.Message, 0)
	cnt, conversactions, err := cr.
		conversationService.
		GetAllMessageActions(
			cr.ctx,
			cr.auth,
			cr.assistantConversation.Id,
			[]*protos.Criteria{
				{
					Key:   "action_type",
					Value: "llm-call",
					Logic: "=",
				},
			},
			&protos.Paginate{
				Page:     1,
				PageSize: CONVERSACTION_PAGE_HISTORY,
			},
			&protos.Ordering{
				Column: "created_date",
				Order:  "asc",
			},
		)

	if cnt == 0 || err != nil {
		return messages
	}

	for _, x := range conversactions {
		if x.Status == type_enums.RECORD_SUCCESS || x.Status == type_enums.RECORD_ACTIVE {
			messages = append(messages, x.RequestMessage())
			messages = append(messages, x.ResponseMessage())
		}
	}
	return messages
}

func (cr *GenericRequestor) CreateConversationMessageLog(
	messageid string, in, out *types.Message, metrics []*types.Metric) error {
	cr.conversationService.CreateLLMAction(
		cr.Context(),
		cr.Auth(),
		cr.assistant.Id,
		cr.assistantConversation.Id,
		messageid,
		in, out, metrics)
	return nil
}

func (cr *GenericRequestor) CreateConversationToolLog(
	messageid string, in, out map[string]interface{}, metrics []*types.Metric) error {
	cr.conversationService.CreateToolAction(
		cr.Context(),
		cr.Auth(),
		cr.assistant.Id,
		cr.assistantConversation.Id,
		messageid,
		in, out, metrics)
	return nil
}

func (cr *GenericRequestor) CreateToolLog(
	toolId uint64,
	messageId string,
	toolName string,
	executionMethod string,
	status type_enums.RecordState,
	timeTaken int64,
	request, response []byte) error {
	_, err := cr.assistantToolService.CreateLog(
		cr.Context(), cr.Auth(), cr.assistant.Id,
		cr.assistantConversation.Id, toolId, messageId, toolName, timeTaken, executionMethod,
		status, request, response,
	)
	return err
}
