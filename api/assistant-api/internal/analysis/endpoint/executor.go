// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_analysis_endpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client_builders "github.com/rapidaai/pkg/clients/endpoint/builders"
	"github.com/rapidaai/pkg/commons"
	rapida_types "github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	AnalysisOptionEndpointIDKey      = "endpoint_id"
	AnalysisOptionEndpointVersionKey = "endpoint_version"
	AnalysisOptionEndpointParamsKey  = "endpoint_parameters"
)

type runtimeExecutor struct {
	logger       commons.Logger
	ctx          context.Context
	caller       internal_type.InternalCaller
	analysis     *internal_assistant_entity.AssistantConfiguration
	onPacket     func(context.Context, ...internal_type.Packet) error
	inputBuilder endpoint_client_builders.InputInvokeBuilder
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

func WithConfiguration(analysis *internal_assistant_entity.AssistantConfiguration) Option {
	return func(executor *runtimeExecutor) {
		executor.analysis = analysis
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

// New creates a fully wired endpoint-based analysis executor.
func New(opts ...Option) (internal_type.AnalysisExecutor, error) {
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
	if executor.analysis == nil {
		return nil, fmt.Errorf("analysis endpoint: configuration is required")
	}
	if executor.caller == nil {
		return nil, fmt.Errorf("analysis endpoint: caller is required")
	}
	executor.inputBuilder = endpoint_client_builders.NewInputInvokeBuilder(executor.logger)
	if executor.onPacket != nil {
		_ = executor.onPacket(executor.ctx,
			internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordMetric{
					Attributes: observability.Attributes{
						"provider":         executor.analysis.Provider,
						"configuration_id": fmt.Sprintf("%d", executor.analysis.Id),
						"executor":         executor.Name(),
					},
					Metrics: []*protos.Metric{{
						Name:        observability.MetricAnalysisInitLatencyMs,
						Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
						Description: "Analysis initialization latency in milliseconds",
					}},
				},
			},
			internal_type.ObservabilityLogRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: fmt.Sprintf("%s: initialization completed", executor.Name()),
					Attributes: observability.Attributes{
						"component":        observability.ComponentAnalysis.String(),
						"operation":        "initialize_executor",
						"provider":         executor.analysis.Provider,
						"configuration_id": fmt.Sprintf("%d", executor.analysis.Id),
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
	name, _ := e.analysis.GetOptions().GetString("name")
	if name == "" {
		name = fmt.Sprintf("%d", e.analysis.Id)
	}
	return fmt.Sprintf("endpoint-analysis-%s", name)
}

func (e *runtimeExecutor) Options() utils.Option {
	return e.analysis.GetOptions()
}

func (e *runtimeExecutor) Arguments() (map[string]string, error) {
	return e.analysis.GetOptions().GetStringMap(AnalysisOptionEndpointParamsKey)
}

func (e *runtimeExecutor) GetEndpointId() (uint64, error) {
	return e.analysis.GetOptions().GetUint64(AnalysisOptionEndpointIDKey)
}

func (e *runtimeExecutor) GetEndpointVersion() (string, error) {
	return e.analysis.GetOptions().GetString(AnalysisOptionEndpointVersionKey)
}

// Execute runs one analysis and returns its metadata to the finalization flow.
func (e *runtimeExecutor) Execute(ctx context.Context, input internal_type.AnalysisInput) (*internal_type.AnalysisOutput, error) {
	endpointID, err := e.GetEndpointId()
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint ID: %w", err)
	}
	endpointVersion, err := e.GetEndpointVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint version: %w", err)
	}
	response, err := e.caller.DeploymentCaller().Invoke(
		ctx,
		input.Auth,
		e.inputBuilder.Invoke(
			&protos.EndpointDefinition{
				EndpointId: endpointID,
				Version:    endpointVersion,
			},
			e.inputBuilder.Arguments(input.Arguments, nil),
			nil,
			nil,
		),
	)
	if err != nil {
		return nil, err
	}
	if !response.GetSuccess() || len(response.GetData()) == 0 {
		return nil, fmt.Errorf("empty response from endpoint")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response.GetData()[0]), &parsed); err != nil {
		parsed = map[string]interface{}{"result": response.GetData()[0]}
	}

	name, _ := e.analysis.GetOptions().GetString("name")
	if name == "" {
		name = fmt.Sprintf("%d", e.analysis.Id)
	}
	metadata := rapida_types.NewMetadata(fmt.Sprintf("analysis.%s", name), parsed)
	return &internal_type.AnalysisOutput{
		Metadata: &protos.Metadata{Key: metadata.Key, Value: metadata.Value},
	}, nil
}

// Close releases executor dependencies.
func (e *runtimeExecutor) Close(_ context.Context) error {
	e.caller = nil
	return nil
}
