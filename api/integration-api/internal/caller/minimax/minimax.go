// Rapida -- Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_minimax_callers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	integration_api "github.com/rapidaai/protos"
)

type MiniMax struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

type MiniMaxError struct {
	Error *struct {
		Message string `json:"message,omitempty"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
	StatusCode int `json:"status_code,omitempty"`
}

type MiniMaxUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type MiniMaxMessageResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Model   string        `json:"model"`
	Created int64         `json:"created"`
	Usage   *MiniMaxUsage `json:"usage"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Content   string `json:"content"`
			Role      string `json:"role"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type MiniMaxStreamChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Model   string        `json:"model"`
	Created int64         `json:"created"`
	Usage   *MiniMaxUsage `json:"usage"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   string `json:"content"`
			Role      string `json:"role"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (e MiniMaxError) ErrorString() string {
	if e.Error != nil {
		return fmt.Sprintf("minimax api error: %s (type=%s, code=%s)", e.Error.Message, e.Error.Type, e.Error.Code)
	}
	return fmt.Sprintf("minimax api error: status_code=%d", e.StatusCode)
}

var (
	API_KEY            = "key"
	API_KEY_HEADER_KEY = "Authorization"
	API_URL            = "https://api.minimax.io/v1/"
	TIMEOUT            = 5 * time.Minute
)

func minimax(logger commons.Logger, credential *integration_api.Credential) MiniMax {
	return MiniMax{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (mm *MiniMax) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: TIMEOUT,
	}
	mm.logger.Debugf("making request to minimax with %+v", req)
	return client.Do(req)
}

func (mm *MiniMax) Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*http.Response, error) {
	credentials := mm.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		mm.logger.Errorf("Unable to get API key for MiniMax")
		return nil, errors.New("unable to resolve the credential")
	}

	var in io.Reader
	if payload != nil {
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			mm.logger.Errorf("Unable to encode the payload for minimax err = %v", err)
			return nil, err
		}
		in = bytes.NewBuffer(encodedPayload)
	}

	req, err := http.NewRequestWithContext(ctx, method, mm.Endpoint(endpoint), in)
	if err != nil {
		mm.logger.Errorf("Unable to build the request for minimax err = %v", err)
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(API_KEY_HEADER_KEY, fmt.Sprintf("Bearer %s", cx.(string)))

	return mm.Do(req)
}

func (mm *MiniMax) CallJSON(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error) {
	resp, err := mm.Call(ctx, endpoint, method, headers, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			mm.logger.Errorf("unable to read response body for minimax with error %v", err)
			return nil, err
		}
		bodyString := string(bodyBytes)
		return &bodyString, nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var apiErr MiniMaxError
	if jsonErr := json.Unmarshal(bodyBytes, &apiErr); jsonErr == nil && apiErr.Error != nil {
		apiErr.StatusCode = resp.StatusCode
		return nil, fmt.Errorf("%s", apiErr.ErrorString())
	}
	return nil, fmt.Errorf("minimax api error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
}

func (mm *MiniMax) Endpoint(urlPath string) string {
	baseURL, _ := url.Parse(API_URL)
	joinedPath := path.Join(baseURL.Path, urlPath)
	baseURL.Path = joinedPath
	return baseURL.String()
}

func (mm *MiniMax) UsageMetrics(usages *MiniMaxUsage) []*protos.Metric {
	metrics := make([]*protos.Metric, 0)
	if usages != nil {
		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.CompletionTokens),
			Description: "Output Token",
		})

		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.PromptTokens),
			Description: "Input Token",
		})

		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokens),
			Description: "Total Token",
		})
	}
	return metrics
}
