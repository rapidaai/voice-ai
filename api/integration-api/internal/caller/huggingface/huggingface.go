package internal_huggingface_callers

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
	integration_api "github.com/rapidaai/protos"
)

type HuggingfaceInferenceParameters struct {
	Temperature       float64 `json:"temperature"`
	TopP              float64 `json:"top_p,omitempty"`
	TopK              int     `json:"top_k,omitempty"`
	MinLength         int     `json:"min_length,omitempty"`
	MaxLength         int     `json:"max_length,omitempty"`
	RepetitionPenalty float64 `json:"repetition_penalty,omitempty"`
	Seed              int     `json:"seed,omitempty"`
}

type HuggingfaceInferencePayload struct {
	Model      string                         `json:"-"`
	Inputs     string                         `json:"inputs"`
	Parameters HuggingfaceInferenceParameters `json:"parameters,omitempty"`
}

type HuggingfaceError struct {
	Err        string `json:"error"`
	StatusCode int    `json:"status_code"`
}

func (e HuggingfaceError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return "undefined error"
	}
	return string(b)
}

type HuggingfaceEmbeddingPayload struct {
	Options map[string]any
	Inputs  []string `json:"inputs"`
}

type HuggingfaceInferenceResponse struct {
	Text string `json:"generated_text"`
}

type Huggingface struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
	endpoint   string
}

var (
	DEFUALT_URL = "https://api-inference.huggingface.co"
	AUTH_URL    = "https://huggingface.co"
	API_URL     = "url"
	TIMEOUT     = 5 * time.Minute
	API_KEY     = "key"
)

func huggingface(logger commons.Logger, endpoint string, credential *integration_api.Credential) Huggingface {
	return Huggingface{
		logger:   logger,
		endpoint: endpoint,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (AICaller *Huggingface) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: TIMEOUT,
	}
	AICaller.logger.Debugf("making request to llm with %+v", req)
	return client.Do(req)
}

func (vgAI *Huggingface) Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error) {

	credentials := vgAI.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		vgAI.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}

	var in io.Reader
	if payload != nil {
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			vgAI.logger.Errorf("Unable to encode the payload for huggingface err = %v", err)
			return nil, err
		}
		in = bytes.NewBuffer(encodedPayload)
	}

	req, err := http.NewRequestWithContext(ctx, method, vgAI.Endpoint(endpoint), in)
	if err != nil {
		vgAI.logger.Errorf("Unable to build the request for huggingface err = %v", err)
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cx))
	// Authorization":"Bearer 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"
	return vgAI.do(req)

}

func (hg *Huggingface) do(req *http.Request) (*string, error) {
	resp, err := hg.Do(req)
	if err != nil {
		hg.logger.Errorf("unable to complete request for anthropic with error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	// Check for valid status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			hg.logger.Errorf("unable to read response body for huggingface with error %v", err)
			return nil, err
		}
		bodyString := string(bodyBytes)
		return &bodyString, nil
	}

	hg.logger.Debugf("response from huggingface with %v and body", err, resp)
	// Handle non-successful HTTP status code
	var apiErr HuggingfaceError
	if err := hg.Unmarshal(resp, &apiErr); err != nil {
		hg.logger.Errorf("unable to unmarshal error response from huggingface with error %v", err)
		return nil, HuggingfaceError{
			StatusCode: resp.StatusCode,
		}
	}

	// Ensure the status code is set correctly
	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = resp.StatusCode
	}

	return nil, &apiErr
}

func (vgAI *Huggingface) Endpoint(urlPath string) string {
	baseURL, _ := url.Parse(vgAI.endpoint)
	// Ensure the path is correctly joined
	joinedPath := path.Join(baseURL.Path, urlPath)
	baseURL.Path = joinedPath
	return baseURL.String()
}

func (vgAI *Huggingface) Unmarshal(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}
