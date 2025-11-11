package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
)

type APIClient interface {
	Get(ctx context.Context, endpoint string, params map[string]interface{}, headers map[string]string) (*APIResponse, error)
	Post(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error)
	Put(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error)
	Patch(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error)
	Delete(ctx context.Context, endpoint string, headers map[string]string) (*APIResponse, error)
	Request(ctx context.Context, method, endpoint string, body interface{}, params map[string]interface{}, headers map[string]string) (*APIResponse, error)
}

type APIResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
	Status     string      `json:"status"`
}

func (r *APIResponse) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func (r *APIResponse) ToMap() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(r.Body, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *APIResponse) ToString() string {
	return string(r.Body)
}

type RestClient struct {
	logger     commons.Logger
	cfg        *config.AppConfig
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
}

func NewRestClient(logger commons.Logger, cfg *config.AppConfig, baseURL string) *RestClient {
	return &RestClient{
		logger:  logger,
		cfg:     cfg,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		Headers: make(map[string]string),
	}
}

func NewRestClientWithConfig(baseURL string, defaultHeaders map[string]string, timeoutSecond uint32) *RestClient {
	headers := make(map[string]string)
	for k, v := range defaultHeaders {
		headers[k] = v
	}
	return &RestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSecond) * time.Second,
		},
		Headers: headers,
	}
}

func (c *RestClient) Get(ctx context.Context, endpoint string, params map[string]interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, http.MethodGet, endpoint, nil, params, headers)
}

func (c *RestClient) Post(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, http.MethodPost, endpoint, body, nil, headers)
}

func (c *RestClient) Put(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, http.MethodPut, endpoint, body, nil, headers)
}

func (c *RestClient) Patch(ctx context.Context, endpoint string, body interface{}, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, http.MethodPatch, endpoint, body, nil, headers)
}

func (c *RestClient) Delete(ctx context.Context, endpoint string, headers map[string]string) (*APIResponse, error) {
	return c.Request(ctx, http.MethodDelete, endpoint, nil, nil, headers)
}

func (c *RestClient) Request(ctx context.Context, method, endpoint string, body interface{}, params map[string]interface{}, headers map[string]string) (*APIResponse, error) {
	fullURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
		Status:     resp.Status,
	}, nil
}

func (c *RestClient) buildURL(endpoint string, params map[string]interface{}) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}

	ep, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	finalURL := base.ResolveReference(ep)

	if params != nil {
		query := finalURL.Query()
		for key, value := range params {
			query.Set(key, fmt.Sprintf("%v", value))
		}
		finalURL.RawQuery = query.Encode()
	}

	return finalURL.String(), nil
}

func (c *RestClient) SetDefaultHeader(key, value string) {
	c.Headers[key] = value
}

func (c *RestClient) RemoveDefaultHeader(key string) {
	delete(c.Headers, key)
}
