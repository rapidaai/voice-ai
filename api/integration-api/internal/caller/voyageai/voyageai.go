package internal_voyageai_callers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type Voyageai struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

type VoyageaiError struct {
	Detail     string `json:"detail"`
	StatusCode int    `json:"status"`
}

func (e VoyageaiError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return "undefined error"
	}
	return string(b)
}

var (
	API_KEY = "key"
	API_URL = "https://api.voyageai.com/v1/"
	TIMEOUT = 5 * time.Minute
)

type VoyageaiUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type VoyageaiEmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type VoyageaiRerankingData struct {
	RelevanceScore float64 `json:"relevance_score"`
	Index          int32   `json:"index"`
}

type VoyageaiEmbeddingResponse struct {
	VoyageaiResponse[VoyageaiEmbeddingData]
}

type VoyageaiRerankingResponse struct {
	VoyageaiResponse[VoyageaiRerankingData]
}

type VoyageaiResponse[T any] struct {
	Object string         `json:"object"`
	Data   []T            `json:"data"`
	Model  string         `json:"model"`
	Usage  *VoyageaiUsage `json:"usage"`
}

// embeddings

func voyageai(logger commons.Logger, credential *integration_api.Credential) Voyageai {
	return Voyageai{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (AICaller *Voyageai) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: TIMEOUT,
	}
	AICaller.logger.Debugf("making request to llm with %+v", req)
	return client.Do(req)
}

func (vgAI *Voyageai) Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error) {

	credentials := vgAI.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		vgAI.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, vgAI.Endpoint(endpoint), bytes.NewBuffer(encodedPayload))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cx))
	return vgAI.do(req)

}

func (vgAI *Voyageai) do(req *http.Request) (*string, error) {
	resp, err := vgAI.Do(req)
	if err != nil {
		vgAI.logger.Errorf("unable to complete request for Voyageai with error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	// Check for valid status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			vgAI.logger.Errorf("unable to read response body for Voyageai with error %v", err)
			return nil, err
		}
		bodyString := string(bodyBytes)
		return &bodyString, nil
	}

	// Handle non-successful HTTP status code
	var apiErr VoyageaiError
	if err := vgAI.Unmarshal(resp, &apiErr); err != nil {
		vgAI.logger.Errorf("unable to unmarshal error response from Voyageai with error %v", err)
		return nil, err
	}

	// Ensure the status code is set correctly
	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = resp.StatusCode
	}

	return nil, &apiErr
}

func (vgAI *Voyageai) Endpoint(url string) string {
	return fmt.Sprintf("%s%s", API_URL, url)
}

func (vgAI *Voyageai) Unmarshal(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}

func (vgAI *Voyageai) UsageMetrics(usages *VoyageaiUsage) types.Metrics {
	metrics := make(types.Metrics, 0)
	if usages != nil {

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokens),
			Description: "Total Token",
		})
	}
	return metrics
}
