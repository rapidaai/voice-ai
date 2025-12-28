// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_local_tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type apiRequestToolCaller struct {
	toolCaller
	apiRequestHeader    map[string]string
	apiRequestParameter map[string]string
	apiMethod           string
	apiEndpoint         string
}

func (afkTool *apiRequestToolCaller) Call(
	ctx context.Context,
	messageId string,
	args string,
	communication internal_adapter_requests.Communication,
) (map[string]interface{}, []*types.Metric) {
	start := time.Now()
	metrics := make([]*types.Metric, 0)
	client := rest.NewRestClientWithConfig(afkTool.apiEndpoint, afkTool.apiRequestHeader, 15)
	var output *rest.APIResponse
	var err error

	body := afkTool.Parse(
		afkTool.apiRequestParameter,
		args,
		communication,
	)
	switch afkTool.apiMethod {
	case "POST":
		output, err = client.Post(ctx, "", body, afkTool.apiRequestHeader)
	case "PUT":
		output, err = client.Put(ctx, "", body, afkTool.apiRequestHeader)
	case "PATCH":
		output, err = client.Patch(ctx, "", body, afkTool.apiRequestHeader)
	default:
		output, err = client.Get(ctx, "", body, afkTool.apiRequestHeader)
	}
	metrics = append(metrics, types.NewTimeTakenMetric(time.Since(start)))
	if err != nil {
		return afkTool.Result("Unable to complete request", true), metrics
	}
	v, err := output.ToMap()
	if err != nil {
		return map[string]interface{}{
			"request":  body,
			"response": output.ToString(),
		}, metrics
	}
	return v, metrics
}

func NewApiRequestToolCaller(
	logger commons.Logger,
	toolOptions *internal_assistant_entity.AssistantTool,
	communcation internal_adapter_requests.Communication,
) (ToolCaller, error) {
	opts := toolOptions.GetOptions()
	endpoint, err := opts.GetString("tool.endpoint")
	if err != nil {
		return nil, fmt.Errorf("tool.endpoint is not a valid number: %v", err)
	}
	method, err := opts.GetString("tool.method")
	if err != nil {
		return nil, fmt.Errorf("tool.method is not a valid number: %v", err)
	}
	parameters, err := opts.GetStringMap("tool.parameters")
	if err != nil {
		return nil, fmt.Errorf("tool.parameters is not a valid number: %v", err)
	}
	headers, err := opts.GetStringMap("tool.headers")
	if err != nil {
		logger.Infof("ignoring headers for api requests.")
	}
	return &apiRequestToolCaller{
		toolCaller: toolCaller{
			logger:      logger,
			toolOptions: toolOptions,
		},
		apiRequestHeader:    headers,
		apiRequestParameter: parameters,
		apiEndpoint:         endpoint,
		apiMethod:           method,
	}, nil
}

func (md *apiRequestToolCaller) Parse(
	mapping map[string]string,
	args string,
	communication internal_adapter_requests.Communication,
) map[string]interface{} {
	arguments := make(map[string]interface{})
	for key, value := range mapping {
		if k, ok := strings.CutPrefix(key, "tool."); ok {
			switch k {
			case "name":
				arguments[value] = md.Name()
			case "argument":
				var argMap map[string]interface{}
				err := json.Unmarshal([]byte(args), &argMap)
				if err != nil {
					md.logger.Debugf("the arugment might be string")
					arguments[value] = args
				} else {
					arguments[value] = argMap
				}
			}
		}
		if k, ok := strings.CutPrefix(key, "assistant."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", communication.Assistant().Id)
			case "version":
				arguments[value] = fmt.Sprintf("vrsn_%d", communication.Assistant().AssistantProviderModel.Id)
			}
		}
		if k, ok := strings.CutPrefix(key, "conversation."); ok {
			switch k {
			case "id":
				arguments[value] = fmt.Sprintf("%d", communication.Conversation().Id)
			case "messages":
				arguments[value] = types.ToSimpleMessage(communication.GetHistories())
			}
		}
		if k, ok := strings.CutPrefix(key, "argument."); ok {
			if aArg, ok := communication.GetArgs()[k]; ok {
				arguments[value] = aArg
			}
		}
		if k, ok := strings.CutPrefix(key, "metadata."); ok {
			if mtd, ok := communication.GetMetadata()[k]; ok {
				arguments[value] = mtd
			}
		}
		if k, ok := strings.CutPrefix(key, "option."); ok {
			if ot, ok := communication.GetOptions()[k]; ok {
				arguments[value] = ot
			}
		}

		if k, ok := strings.CutPrefix(key, "custom."); ok {
			arguments[k] = value
		}
	}
	return arguments
}
