// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	telemetry "github.com/rapidaai/pkg/telemetry"
	"github.com/rapidaai/pkg/telemetry/providers"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

type Provider struct {
	Name    string
	Options map[string]interface{}
}

type Config struct {
	Logger                        commons.Logger
	Providers                     Provider
	Exporters                     telemetry.Exporter
	Key                           string
	Auth                          *types.Authentication
	AssistantID                   uint64
	AssistantConfigurationService internal_services.AssistantConfigurationService
}

type Collector struct {
	logger                        commons.Logger
	provider                      Provider
	exporter                      telemetry.Exporter
	initialized                   bool
	mu                            sync.Mutex
	key                           string
	auth                          *types.Authentication
	assistantID                   uint64
	assistantConfigurationService internal_services.AssistantConfigurationService
	assistantCollectors           *observability.Collectors
	assistantCollectorsLoaded     bool
}

func NewWithAssistantConfig(ctx context.Context, logger commons.Logger, assistantConfig *assistant_config.AssistantConfig) []observability.Collector {
	if assistantConfig == nil || assistantConfig.TelemetryConfig == nil || assistantConfig.TelemetryConfig.Type() == "" {
		return nil
	}

	collector, err := New(ctx, Config{
		Logger: logger,
		Providers: Provider{
			Name:    string(assistantConfig.TelemetryConfig.Type()),
			Options: assistantConfig.TelemetryConfig.ToMap(),
		},
	})
	if err != nil {
		if logger != nil {
			logger.Warnf("unable to create telemetry collector: %v", err)
		}
		return nil
	}
	return []observability.Collector{collector}
}

func New(ctx context.Context, cfg Config) (observability.Collector, error) {
	if validator.NonNil(cfg.AssistantConfigurationService) || cfg.AssistantID != 0 || validator.NonNil(cfg.Auth) {
		if !validator.NonNil(cfg.Auth) || cfg.AssistantID == 0 || !validator.NonNil(cfg.AssistantConfigurationService) {
			return observability.NoopCollector{}, nil
		}
		return &Collector{
			logger:                        cfg.Logger,
			auth:                          cfg.Auth,
			assistantID:                   cfg.AssistantID,
			assistantConfigurationService: cfg.AssistantConfigurationService,
			key:                           "telemetry:assistant:" + strconv.FormatUint(cfg.AssistantID, 10),
			initialized:                   true,
		}, nil
	}

	key := "telemetry"
	if providerName := strings.TrimSpace(cfg.Providers.Name); validator.NotBlank(providerName) {
		key = "telemetry:" + providerName
	}
	if validator.NotBlank(cfg.Key) {
		key = cfg.Key
	}
	if validator.NonNil(cfg.Exporters) {
		return &Collector{exporter: cfg.Exporters, initialized: true, key: key}, nil
	}
	providerName := strings.TrimSpace(cfg.Providers.Name)
	if !validator.NotBlank(providerName) {
		return observability.NoopCollector{}, nil
	}
	switch telemetry.ExporterType(providerName) {
	case telemetry.OTLP_HTTP, telemetry.OTLP_GRPC, telemetry.XRAY, telemetry.GOOGLE_TRACE,
		telemetry.AZURE_MONITOR, telemetry.DATADOG, telemetry.OPENSEARCH, telemetry.LOGGING:
	default:
		return nil, errors.New("telemetry: unknown exporter type " + strconv.Quote(providerName))
	}
	options := make(map[string]interface{}, len(cfg.Providers.Options))
	for key, value := range cfg.Providers.Options {
		options[key] = value
	}
	return &Collector{
		logger: cfg.Logger,
		provider: Provider{
			Name:    providerName,
			Options: options,
		},
		key: key,
	}, nil
}

func (c *Collector) Key() string {
	return c.key
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, observationContext observability.Context, record observability.Record) error {
	if validator.NonNil(c.assistantConfigurationService) {
		c.mu.Lock()
		if !c.assistantCollectorsLoaded {
			_, telemetryConfigurations, err := c.assistantConfigurationService.GetAll(
				ctx,
				c.auth,
				c.assistantID,
				string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
				"",
				nil,
				&protos.Paginate{},
			)
			if err != nil {
				c.mu.Unlock()
				return err
			}

			configuredCollectors := make([]observability.Collector, 0, len(telemetryConfigurations))
			for _, telemetryConfiguration := range telemetryConfigurations {
				if telemetryConfiguration == nil || !telemetryConfiguration.Enabled {
					continue
				}
				collectorKey := ""
				if telemetryConfiguration.Id != 0 {
					collectorKey = "telemetry:assistant:" + strconv.FormatUint(telemetryConfiguration.Id, 10)
				}
				options := make(map[string]interface{}, len(telemetryConfiguration.GetOptions()))
				for key, value := range telemetryConfiguration.GetOptions() {
					options[key] = value
				}
				collector, err := New(ctx, Config{
					Logger: c.logger,
					Providers: Provider{
						Name:    telemetryConfiguration.Provider,
						Options: options,
					},
					Key: collectorKey,
				})
				if err != nil {
					continue
				}
				configuredCollectors = append(configuredCollectors, collector)
			}
			c.assistantCollectors = observability.NewCollectors(configuredCollectors...)
			c.assistantCollectorsLoaded = true
		}
		collectors := c.assistantCollectors
		c.mu.Unlock()
		if collectors == nil {
			return nil
		}
		return collectors.Collect(ctx, scope, observationContext, record)
	}

	c.mu.Lock()
	if !c.initialized {
		exporter, err := newExporter(ctx, c.logger, c.provider)
		if err != nil {
			c.mu.Unlock()
			return err
		}
		c.exporter = exporter
		c.initialized = true
	}
	exporter := c.exporter
	c.mu.Unlock()
	if !validator.NonNil(exporter) {
		return nil
	}

	switch typed := record.(type) {
	case observability.RecordLog:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		attributes := make(map[string]string, len(typed.Attributes))
		for key, value := range typed.Attributes {
			attributes[key] = value
		}
		return exporter.Export(ctx, c.toTelemetryScope(scope), telemetry.LogRecord{
			ID:         typed.ID,
			Context:    map[string]string{"traceId": observationContext.TraceID},
			Level:      string(typed.Level),
			Message:    typed.Message,
			Attributes: attributes,
			OccurredAt: occurredAt,
		})
	case observability.RecordEvent:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		attributes := make(map[string]string, len(typed.Attributes))
		for key, value := range typed.Attributes {
			attributes[key] = value
		}
		return exporter.Export(ctx, c.toTelemetryScope(scope), telemetry.EventRecord{
			ID:         typed.ID,
			Context:    map[string]string{"traceId": observationContext.TraceID},
			Event:      typed.Event.String(),
			Component:  typed.Component.String(),
			Attributes: attributes,
			OccurredAt: occurredAt,
		})
	case observability.RecordMetric:
		if !validator.NotEmpty(typed.Metrics) {
			return nil
		}
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		attributes := make(map[string]string, len(typed.Attributes))
		for key, value := range typed.Attributes {
			attributes[key] = value
		}
		var errs []error
		for _, metric := range typed.Metrics {
			if metric == nil {
				continue
			}
			if err := exporter.Export(ctx, c.toTelemetryScope(scope), telemetry.MetricRecord{
				ID:          typed.ID,
				Context:     map[string]string{"traceId": observationContext.TraceID},
				Name:        metric.GetName(),
				Value:       metric.GetValue(),
				Description: metric.GetDescription(),
				Attributes:  attributes,
				OccurredAt:  occurredAt,
			}); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	default:
		return nil
	}
}

func (c *Collector) toTelemetryScope(scope observability.Scope) telemetry.Scope {
	global := scope.GlobalScopeValue()
	scopeAttributes := map[string]string{}
	switch typed := scope.(type) {
	case observability.AssistantScope:
		scopeAttributes["assistantId"] = strconv.FormatUint(typed.AssistantScopeID(), 10)
	case observability.ConversationScope:
		scopeAttributes["assistantId"] = strconv.FormatUint(typed.AssistantScopeID(), 10)
		scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.ConversationScopeID(), 10)
	case observability.MessageScope:
		scopeAttributes["assistantId"] = strconv.FormatUint(typed.AssistantScopeID(), 10)
		scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.ConversationScopeID(), 10)
		scopeAttributes["messageId"] = typed.ContextID()
		scopeAttributes["messageRole"] = string(typed.MessageScopeRole())
	}
	return telemetry.Scope{
		ProjectID:       global.ProjectID,
		OrganizationID:  global.OrganizationID,
		Name:            string(scope.ScopeType()),
		ScopeAttributes: scopeAttributes,
	}
}

func (c *Collector) Close(ctx context.Context) error {
	if validator.NonNil(c.assistantConfigurationService) {
		c.mu.Lock()
		collectors := c.assistantCollectors
		c.mu.Unlock()
		if collectors == nil {
			return nil
		}
		return collectors.Close(ctx)
	}

	var errs []error
	c.mu.Lock()
	if !c.initialized {
		c.initialized = true
		c.mu.Unlock()
		return nil
	}
	exporter := c.exporter
	c.mu.Unlock()
	if validator.NonNil(exporter) {
		if err := exporter.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func newExporter(ctx context.Context, logger commons.Logger, provider Provider) (telemetry.Exporter, error) {
	providerName := strings.TrimSpace(provider.Name)
	if !validator.NotBlank(providerName) {
		return nil, nil
	}
	if len(provider.Options) == 0 {
		return providers.NewExporterFromOptions(logger, ctx, providerName, nil)
	}
	options := make(map[string]interface{}, len(provider.Options))
	for key, value := range provider.Options {
		options[key] = value
	}
	return providers.NewExporterFromOptions(logger, ctx, providerName, options)
}
