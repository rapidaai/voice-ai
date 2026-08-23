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
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewWithEnv(ctx context.Context, logger commons.Logger, config *assistant_config.AssistantConfig) observability.Collector {
	if config == nil {
		return nil
	}

	if config.TelemetryConfig != nil && config.TelemetryConfig.Type() != "" {
		if collector, err := telemetry.New(ctx, telemetry.Config{Logger: logger,
			Providers: telemetry.Provider{
				Name:    string(config.TelemetryConfig.Type()),
				Options: config.TelemetryConfig.ToMap(),
			}}); err == nil {
			return collector
		}
	}
	return nil
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
