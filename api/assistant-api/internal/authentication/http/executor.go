// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_authentication_http

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	OptionHTTPURLKey     = "http_url"
	OptionHTTPMethodKey  = "http_method"
	OptionHTTPHeadersKey = "http_headers"
	OptionHTTPBodyKey    = "http_body"

	ResponseArgumentsKey   = "arguments"
	ResponseArgumentsKeyV1 = "args"
	ResponseMetadataKey    = "metadata"
	ResponseOptionsKey     = "options"
)

type runtimeExecutor struct {
	logger        commons.Logger
	ctx           context.Context
	contextID     string
	callback      internal_type.Callback
	caller        internal_type.InternalCaller
	authenticator *internal_assistant_entity.AssistantConfiguration
	onPacket      func(context.Context, ...internal_type.Packet) error
}

type Option func(*runtimeExecutor)

func WithLogger(logger commons.Logger) Option {
	return func(executor *runtimeExecutor) {
		executor.logger = logger
	}
}

func WithContext(ctx context.Context) Option {
	return func(executor *runtimeExecutor) {
		executor.ctx = ctx
	}
}

func WithConfiguration(authenticator *internal_assistant_entity.AssistantConfiguration) Option {
	return func(executor *runtimeExecutor) {
		executor.authenticator = authenticator
	}
}

func WithCallback(callback internal_type.Callback) Option {
	return func(executor *runtimeExecutor) {
		executor.callback = callback
	}
}

func WithCaller(caller internal_type.InternalCaller) Option {
	return func(executor *runtimeExecutor) {
		executor.caller = caller
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(executor *runtimeExecutor) {
		executor.onPacket = onPacket
	}
}

// New creates a fully wired HTTP authentication executor.
func New(opts ...Option) (internal_type.AuthenticationExecutor, error) {
	executor := &runtimeExecutor{ctx: context.Background()}
	start := time.Now()
	for _, opt := range opts {
		if opt != nil {
			opt(executor)
		}
	}
	if executor.ctx == nil {
		executor.ctx = context.Background()
	}
	if executor.callback != nil {
		executor.onPacket = executor.callback.OnPacket
	}
	if executor.authenticator == nil {
		return nil, fmt.Errorf("authentication http: configuration is required")
	}
	if executor.callback == nil {
		return nil, fmt.Errorf("authentication http: callback is required")
	}
	if executor.onPacket != nil {
		_ = executor.onPacket(executor.ctx,
			internal_type.ObservabilityMetricRecordPacket{
				ContextID: executor.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{
						"provider":         executor.authenticator.Provider,
						"configuration_id": fmt.Sprintf("%d", executor.authenticator.Id),
						"executor":         executor.Name(),
					},
					Metrics: []*protos.Metric{{
						Name:        observability.MetricAuthenticationInitLatencyMs,
						Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
						Description: "Authentication initialization latency in milliseconds",
					}},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				ContextID: executor.contextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: fmt.Sprintf("%s: initialization completed", executor.Name()),
					Attributes: observability.Attributes{
						"component":        observability.ComponentAuthentication.String(),
						"operation":        "initialize_executor",
						"provider":         executor.authenticator.Provider,
						"configuration_id": fmt.Sprintf("%d", executor.authenticator.Id),
						"context_id":       executor.contextID,
						"options":          observability.AttributeValue(executor.Options()),
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}
	return executor, nil
}

func (e *runtimeExecutor) Name() string {
	return e.authenticator.Provider
}

func (e *runtimeExecutor) Options() utils.Option {
	return e.authenticator.GetOptions()
}

func (e *runtimeExecutor) Arguments() (map[string]string, error) {
	return e.authenticator.GetOptions().GetStringMap(OptionHTTPBodyKey)
}

// Execute runs authentication against the configured endpoint.
func (e *runtimeExecutor) Execute(ctx context.Context, input internal_type.AuthenticationInput) (*internal_type.AuthenticationOutput, error) {
	auth := e.authenticator
	url, err := auth.GetOptions().GetString(OptionHTTPURLKey)
	if err != nil || url == "" {
		return nil, fmt.Errorf("authentication: missing %s", OptionHTTPURLKey)
	}
	method := "POST"
	if m, err := auth.GetOptions().GetString(OptionHTTPMethodKey); err == nil && m != "" {
		method = m
	}
	method = strings.ToUpper(method)

	headers := map[string]string{}
	if h, err := auth.GetOptions().GetStringMap(OptionHTTPHeadersKey); err == nil {
		headers = h
	}

	timeout := uint32(5000)
	if raw, err := auth.GetOptions().GetUint32("timeout_ms"); err == nil {
		timeout = raw
	}

	client := rest.NewRestClientWithConfig(url, headers, uint32(timeout/1000))
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()
	startTime := time.Now()
	requestPayload := e.createRequestPayload(url, method, headers, uint32(timeout), input.Arguments)
	response, err := e.send(callCtx, client, method, input.Arguments, headers)
	if err != nil {
		errMsg := err.Error()
		e.onCreateLog(ctx, input.ContextID, url, method, e.authenticator.Id, startTime, type_enums.RECORD_FAILED, 0, &errMsg, requestPayload, nil)
		return nil, fmt.Errorf("authentication: request failed: %w", err)
	}

	result := &internal_type.AuthenticationOutput{
		Authenticated: response.StatusCode >= 200 && response.StatusCode < 300,
	}
	if parsed, err := response.ToMap(); err == nil {
		if args, ok := parsed[ResponseArgumentsKeyV1].(map[string]interface{}); ok {
			result.Arguments = args
		}
		if args, ok := parsed[ResponseArgumentsKey].(map[string]interface{}); ok {
			result.Arguments = args
		}
		if metadata, ok := parsed[ResponseMetadataKey].(map[string]interface{}); ok {
			result.Metadata = metadata
		}
		if options, ok := parsed[ResponseOptionsKey].(map[string]interface{}); ok {
			result.Options = options
		}
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errMsg := fmt.Sprintf("authentication: endpoint returned status %d", response.StatusCode)
		e.onCreateLog(ctx, input.ContextID, url, method, e.authenticator.Id, startTime, type_enums.RECORD_FAILED, int64(response.StatusCode), &errMsg, requestPayload, response.Body)
		result.Authenticated = false
	} else {
		e.onCreateLog(ctx, input.ContextID, url, method, e.authenticator.Id, startTime, type_enums.RECORD_COMPLETE, int64(response.StatusCode), nil, requestPayload, response.Body)
	}

	if !result.Authenticated {
		failBehavior := "block"
		if raw, err := auth.GetOptions().GetString("fail_behavior"); err == nil {
			failBehavior = strings.ToLower(strings.TrimSpace(raw))
		}
		switch failBehavior {
		case "do_nothing", "do-nothing", "none", "allow":
			return result, nil
		default:
			return nil, fmt.Errorf("authentication: unauthenticated")
		}
	}

	return result, nil
}

func (e *runtimeExecutor) createRequestPayload(url, method string, headers map[string]string, timeoutMs uint32, body map[string]interface{}) []byte {
	payload := map[string]interface{}{
		"url":        url,
		"method":     method,
		"headers":    headers,
		"timeout_ms": timeoutMs,
		"body":       body,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		e.logger.Warnw("Failed to serialize authentication request payload snapshot", "error", err)
		return nil
	}
	return data
}

// Close releases executor dependencies.
func (e *runtimeExecutor) Close(_ context.Context) error {
	e.callback = nil
	return nil
}

func (e *runtimeExecutor) send(ctx context.Context, client *rest.RestClient, method string, body map[string]interface{}, headers map[string]string) (*rest.APIResponse, error) {
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

func (e *runtimeExecutor) onCreateLog(
	ctx context.Context,
	contextID string,
	url string,
	method string,
	sourceRefID uint64,
	startTime time.Time,
	status type_enums.RecordState,
	responseStatus int64,
	errorMessage *string,
	requestPayload []byte,
	responsePayload []byte,
) {
	if err := e.callback.OnPacket(ctx, internal_type.HTTPLogCreatePacket{
		ContextID:       contextID,
		Source:          "authentication",
		SourceRefID:     sourceRefID,
		SourceEvent:     "session_authentication",
		HTTPURL:         url,
		HTTPMethod:      method,
		ResponseStatus:  responseStatus,
		TimeTaken:       int64(time.Since(startTime)),
		RetryCount:      0,
		Status:          status,
		ErrorMessage:    errorMessage,
		RequestPayload:  requestPayload,
		ResponsePayload: responsePayload,
	}); err != nil {
		e.logger.Warnw("Failed to enqueue authentication http log", "error", err)
	}
}
