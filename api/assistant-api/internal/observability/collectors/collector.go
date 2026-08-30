// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package collectors

import (
	"context"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/billing"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewWithEnv(ctx context.Context, logger commons.Logger, config *assistant_config.AssistantConfig) observability.Collector {
	if config == nil {
		return nil
	}

	configuredCollectors := make([]observability.Collector, 0, 2)
	if config.TelemetryConfig != nil && config.TelemetryConfig.Type() != "" {
		if collector, err := telemetry.New(ctx, telemetry.Config{Logger: logger,
			Providers: telemetry.Provider{
				Name:    string(config.TelemetryConfig.Type()),
				Options: config.TelemetryConfig.ToMap(),
			}}); err == nil {
			configuredCollectors = append(configuredCollectors, collector)
		} else if logger != nil {
			logger.Warnf("unable to create telemetry collector: %v", err)
		}
	}

	if config.Web.Host != "" {
		publisher, err := web_client.NewProductUsageServiceClientGRPC(&config.AppConfig, logger)
		if err == nil {
			configuredCollectors = append(configuredCollectors, billing.New(publisher))
		} else if logger != nil {
			logger.Warnf("unable to create billing collector: %v", err)
		}
	}

	switch len(configuredCollectors) {
	case 0:
		return nil
	case 1:
		return configuredCollectors[0]
	default:
		return observability.NewCollectors(configuredCollectors...)
	}
}

func NewWithAssistantTelemetry(ctx context.Context, logger commons.Logger, auth *types.Authentication, assistantID uint64, assistantConfigurationService internal_services.AssistantConfigurationService) observability.Collector {
	collector, err := telemetry.New(ctx, telemetry.Config{
		Logger:                        logger,
		Auth:                          auth,
		AssistantID:                   assistantID,
		AssistantConfigurationService: assistantConfigurationService,
	})
	if err != nil {
		return nil
	}
	if _, ok := collector.(observability.NoopCollector); ok {
		return nil
	}
	return collector
}

func NewWithWebhookConfiguration(ctx context.Context, logger commons.Logger, auth *types.Authentication, assistantID uint64, assistantConfigurationService internal_services.AssistantConfigurationService, httpLogService internal_services.AssistantHTTPLogService) observability.Collector {
	collector := webhook.New(ctx, webhook.Config{
		Logger:                        logger,
		Auth:                          auth,
		AssistantID:                   assistantID,
		AssistantConfigurationService: assistantConfigurationService,
		HTTPLogService:                httpLogService,
	})
	if _, ok := collector.(observability.NoopCollector); ok {
		return nil
	}
	return collector
}
