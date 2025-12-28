// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	endpoint_client_builders "github.com/rapidaai/pkg/clients/endpoint/builders"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (md *GenericRequestor) OnRecieveMessage(msg *types.Message) *types.Message {
	return msg
}

func (md *GenericRequestor) OnSendMessage(msg *types.Message) *types.Message {
	return msg
}

func (md *GenericRequestor) OnBeginConversation() error {
	utils.Go(md.Context(), func() {
		for _, webhook := range md.assistant.AssistantWebhooks {
			if slices.Contains(webhook.AssistantEvents, utils.ConversationBegin.Get()) {
				arguments := md.Parse(utils.ConversationBegin, webhook.GetBody())
				md.Webhook(utils.ConversationBegin.Get(), arguments, webhook)
			}
		}
	})
	return nil
}
func (md *GenericRequestor) OnResumeConversation() error {
	for _, webhook := range md.assistant.AssistantWebhooks {
		if slices.Contains(webhook.AssistantEvents, utils.ConversationBegin.Get()) {
			arguments := md.Parse(utils.ConversationResume, webhook.GetBody())
			md.Webhook(utils.ConversationBegin.Get(), arguments, webhook)
		}
	}
	return nil
}
func (md *GenericRequestor) OnErrorConversation() error {
	for _, webhook := range md.assistant.AssistantWebhooks {
		if slices.Contains(webhook.AssistantEvents, utils.ConversationFailed.Get()) {
			arguments := md.Parse(utils.ConversationFailed, webhook.GetBody())
			md.Webhook(utils.ConversationFailed.Get(), arguments, webhook)
		}
	}
	return nil
}
func (md *GenericRequestor) OnEndConversation() error {
	utils.Go(md.Context(), func() {
		if len(md.assistant.AssistantAnalyses) > 0 {
			output := make(map[string]interface{})
			for _, a := range md.assistant.AssistantAnalyses {
				aArgs := md.Parse(utils.ConversationCompleted, a.GetParameters())
				o, err := md.Analysis(
					a.GetEndpointId(),
					a.GetEndpointVersion(),
					aArgs,
				)
				if err != nil {
					md.logger.Errorf("error while executing analysis, check the config")
					continue
				}
				output[fmt.Sprintf("analysis.%s", a.GetName())] = o
			}

			md.SetMetadata(md.Auth(), output)
		}
		for _, webhook := range md.assistant.AssistantWebhooks {
			if slices.Contains(webhook.AssistantEvents, utils.ConversationCompleted.Get()) {
				arguments := md.Parse(utils.ConversationCompleted, webhook.GetBody())
				md.Webhook(utils.ConversationCompleted.Get(), arguments, webhook)
			}
		}
	})
	return nil
}
func (hk *GenericRequestor) Analysis(
	endpointId uint64, endpointVersion string,
	arguments map[string]interface{},
) (map[string]interface{}, error) {
	ivk, err := hk.analyze(
		hk.Context(),
		&protos.EndpointDefinition{
			EndpointId: endpointId,
			Version:    endpointVersion,
		},
		arguments,
		nil, nil,
	)
	if err != nil {
		hk.logger.Errorf("error while calling analysis %v", err)
		return nil, err
	}
	if ivk.GetSuccess() {
		if data := ivk.GetData(); len(data) > 0 {
			var contentData map[string]interface{}
			if err := json.Unmarshal(data[0].Content, &contentData); err != nil {
				return map[string]interface{}{
					"result": string(data[0].Content),
				}, nil
			}
			return contentData, nil
		}

	}
	return nil, fmt.Errorf("empty response from endpoint")
}

func (md *GenericRequestor) Webhook(
	event string,
	arguments map[string]interface{},
	webhook *internal_assistant_entity.AssistantWebhook) {
	utils.Go(md.Context(), func() {
		startTime := time.Now()
		var res *rest.APIResponse
		var err error
		var statusCode int

		retryCount := uint32(0)
		maxRetryCount := webhook.GetMaxRetryCount()
		retryStatusCodes := webhook.GetRetryStatusCode()

		for retryCount <= maxRetryCount {
			res, err = md.webhook(md.Context(),
				webhook.GetTimeoutSecond(),
				webhook.GetUrl(),
				webhook.GetMethod(),
				webhook.GetHeaders(),
				arguments,
			)

			if err != nil {
				md.logger.Error("Webhook execution failed", "error", err)
				statusCode = 500
			} else {
				statusCode = res.StatusCode
				if !slices.Contains(retryStatusCodes, strconv.Itoa(statusCode)) {
					break
				}
			}

			retryCount++
			if retryCount <= maxRetryCount {
				time.Sleep(time.Second * 2)
			}
		}

		c, serializeErr := utils.Serialize(arguments)
		if serializeErr != nil {
			md.logger.Error("Failed to serialize arguments", "error", serializeErr)
		}

		v, err := res.ToJSON()
		if err != nil {
			md.logger.Error("Failed to convert response to JSON", "error", err)
		}
		logErr := md.CreateWebhookLog(
			webhook.Id,
			webhook.HttpUrl,
			webhook.HttpMethod,
			event,
			int64(statusCode),
			int64(time.Since(startTime)),
			uint32(retryCount),
			type_enums.RECORD_COMPLETE,
			c,
			v,
		)
		if logErr != nil {
			md.logger.Error("Failed to create webhook log", "error", logErr)
		}
	})
}

func (md *GenericRequestor) Parse(
	event utils.AssistantWebhookEvent,
	mapping map[string]string,
) map[string]interface{} {
	arguments := make(map[string]interface{})
	for key, value := range mapping {
		if k, ok := strings.CutPrefix(key, "event."); ok {
			switch k {
			case "type":
				arguments[value] = event.Get()
			case "data":
				analysisData := make(map[string]interface{})
				for k, v := range md.GetMetadata() {
					if analysisKey, ok := strings.CutPrefix(k, "analysis."); ok {
						analysisData[analysisKey] = v
					}
				}
				arguments[value] = map[string]interface{}{
					"assistant": map[string]interface{}{
						"id":      fmt.Sprintf("%d", md.assistant.Id),
						"version": fmt.Sprintf("vrsn_%d", md.assistant.AssistantProviderId),
					},
					"conversation": map[string]interface{}{
						"id":       fmt.Sprintf("%d", md.assistantConversation.Id),
						"messages": types.ToSimpleMessage(md.GetHistories()),
					},
					"analysis": analysisData,
				}
			}
		}
		if k, ok := strings.CutPrefix(key, "assistant."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", md.assistant.Id)
			case "version":
				arguments[value] = fmt.Sprintf("vrsn_%d", md.assistant.AssistantProviderId)
			}
		}
		if k, ok := strings.CutPrefix(key, "conversation."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", md.assistantConversation.Id)
			case "messages":
				arguments[value] = types.ToSimpleMessage(md.GetHistories())
			}
		}
		if k, ok := strings.CutPrefix(key, "argument."); ok {
			if aArg, ok := md.GetArgs()[k]; ok {
				arguments[value] = aArg
			}
		}
		if k, ok := strings.CutPrefix(key, "metadata."); ok {
			if mtd, ok := md.GetMetadata()[k]; ok {
				arguments[value] = mtd
			}
		}
		if k, ok := strings.CutPrefix(key, "option."); ok {
			if ot, ok := md.GetOptions()[k]; ok {
				arguments[value] = ot
			}
		}

		if ok := strings.HasPrefix(key, "analysis."); ok {
			if ot, ok := md.GetMetadata()[key]; ok {
				arguments[value] = ot
			}
		}

	}
	return arguments
}

func (ae *GenericRequestor) analyze(
	ctx context.Context,
	endpointDef *protos.EndpointDefinition,
	arguments, metadata, opts map[string]interface{},
) (*protos.InvokeResponse, error) {
	inputBuilder := endpoint_client_builders.NewInputInvokeBuilder(ae.logger)
	return ae.DeploymentCaller().Invoke(
		ctx,
		ae.Auth(),
		inputBuilder.Invoke(
			endpointDef,
			inputBuilder.Arguments(arguments, nil),
			inputBuilder.Metadata(metadata, nil),
			inputBuilder.Options(opts, nil),
		),
	)
}

func (aw *GenericRequestor) webhook(
	ctx context.Context,
	timeout uint32,
	baseUrl string,
	method string,
	headers map[string]string,
	body map[string]interface{},
) (*rest.APIResponse, error) {
	client := rest.NewRestClientWithConfig(baseUrl, headers, timeout)
	switch method {
	case "POST":
		return client.Post(ctx, "", body, headers)
	case "PUT":
		return client.Put(ctx, "", body, headers)
	case "PATCH":
		return client.Patch(ctx, "", body, headers)
	default:
		return client.Get(ctx, "", body, headers)
	}
}
