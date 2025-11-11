package internal_mistral_callers

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

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type Mistral struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

type MistralError struct {
	Detail []struct {
		Message string `json:"msg,omitempty"`
		Type    string `json:"type,omitempty"`
	} `json:"detail,omitempty"`
	StatusCode int `json:"status_code,omitempty"`
}

type MistralUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type MistralEmbeddingResponse struct {
	ID     string        `json:"id"`
	Object string        `json:"object"`
	Model  string        `json:"model"`
	Usage  *MistralUsage `json:"usage"`
	Data   [][]struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}
type MistralMessageResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Model   string        `json:"model"`
	Created int64         `json:"created"`
	Usage   *MistralUsage `json:"usage"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string   `json:"name"`
					Arguments struct{} `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
			Prefix bool   `json:"prefix"`
			Role   string `json:"role"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (e MistralError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return "undefined error"
	}
	return string(b)
}

var (
	API_KEY            = "key"
	API_KEY_HEADER_KEY = "Authorization"
	API_URL            = "https://api.mistral.ai/"
	TIMEOUT            = 5 * time.Minute
)

func mistral(logger commons.Logger, credential *integration_api.Credential) Mistral {
	return Mistral{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (AICaller *Mistral) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: TIMEOUT,
	}
	AICaller.logger.Debugf("making request to llm with %+v", req)
	return client.Do(req)
}

func (mistralC *Mistral) Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error) {
	credentials := mistralC.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		mistralC.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}

	var in io.Reader
	if payload != nil {
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			mistralC.logger.Errorf("Unable to encode the payload for mistral err = %v", err)
			return nil, err
		}
		in = bytes.NewBuffer(encodedPayload)
	}

	req, err := http.NewRequestWithContext(ctx, method, mistralC.Endpoint(endpoint), in)
	if err != nil {
		mistralC.logger.Errorf("Unable to build the request for mistral err = %v", err)
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(API_KEY_HEADER_KEY, fmt.Sprintf("Bearer %s", cx.(string)))
	return mistralC.do(req)

}

func (mistralC *Mistral) do(req *http.Request) (*string, error) {
	resp, err := mistralC.Do(req)
	if err != nil {
		mistralC.logger.Errorf("unable to complete request for mistral with error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	// Check for valid status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			mistralC.logger.Errorf("unable to read response body for mistral with error %v", err)
			return nil, err
		}
		bodyString := string(bodyBytes)
		return &bodyString, nil
	}

	// Handle non-successful HTTP status code
	var apiErr MistralError
	if err := mistralC.Unmarshal(resp, &apiErr); err != nil {
		mistralC.logger.Errorf("unable to unmarshal error response from mistral with error %v", err)
		return nil, err
	}

	// Ensure the status code is set correctly
	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = resp.StatusCode
	}
	return nil, &apiErr
}

func (vgAI *Mistral) Endpoint(urlPath string) string {
	baseURL, _ := url.Parse(API_URL)
	// Ensure the path is correctly joined
	joinedPath := path.Join(baseURL.Path, urlPath)
	baseURL.Path = joinedPath
	return baseURL.String()
}

func (mistralC *Mistral) Unmarshal(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}

func (mistralC *Mistral) UsageMetrics(usages *MistralUsage) types.Metrics {
	metrics := make(types.Metrics, 0)
	if usages != nil {
		metrics = append(metrics, &types.Metric{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.PromptTokens),
			Description: "Input token",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.CompletionTokens),
			Description: "Output Token",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokens),
			Description: "Total Token",
		})
	}
	return metrics
}
