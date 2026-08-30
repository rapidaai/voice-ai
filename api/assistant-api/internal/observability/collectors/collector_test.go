// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package collectors

import (
	"context"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestNew_AppendsTelemetryWhenDefaultTelemetryEnabled(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			Logging:       &configs.TelemetryLoggingConfig{},
		},
	})
	if collector == nil {
		t.Fatal("expected telemetry collector")
	}
}

func TestTimelineConfig_IgnoresTelemetryOpenSearch(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			OpenSearch:    &configs.OpenSearchConfig{Schema: "http", Host: "localhost"},
			Logging:       &configs.TelemetryLoggingConfig{},
		},
	})
	if collector == nil {
		t.Fatal("expected telemetry collector")
	}
}

func TestNew_SkipsInactiveConfig(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{},
	})
	if collector != nil {
		t.Fatalf("expected no collector, got %T", collector)
	}
}

func TestNew_SkipsUnknownDefaultTelemetryType(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		TelemetryConfig: &configs.TelemetryConfig{TelemetryType: "unknown"},
	})
	if collector != nil {
		t.Fatalf("expected no collector, got %T", collector)
	}
}

func TestNewWithEnv_AddsBillingWhenWebHostConfigured(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		AppConfig: config.AppConfig{
			Name:      "assistant-api",
			ServiceID: 1,
			Secret:    "secret",
			Web:       config.ServiceHostConfig{Host: "passthrough:///web-api"},
		},
	})
	if collector == nil {
		t.Fatal("expected billing collector")
	}
	if collector.Key() != "billing" {
		t.Fatalf("collector key = %q, want billing", collector.Key())
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewWithEnv_ComposesTelemetryAndBilling(t *testing.T) {
	collector := NewWithEnv(context.Background(), testLogger(t), &assistant_config.AssistantConfig{
		AppConfig: config.AppConfig{
			Name:      "assistant-api",
			ServiceID: 1,
			Secret:    "secret",
			Web:       config.ServiceHostConfig{Host: "passthrough:///web-api"},
		},
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			Logging:       &configs.TelemetryLoggingConfig{},
		},
	})
	if collector == nil {
		t.Fatal("expected composite collector")
	}
	if collector.Key() != "collectors" {
		t.Fatalf("collector key = %q, want collectors", collector.Key())
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNew_LogsAndSkipsTelemetryWhenCollectorFails(t *testing.T) {
	organizationID := uint64(10)
	projectID := uint64(20)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			{
				ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeTelemetry,
				Provider:          "unknown",
				Enabled:           true,
			},
		},
	}
	collector := NewWithAssistantTelemetry(context.Background(), nil, auth, 30, configurationService)
	if collector == nil {
		t.Fatal("expected assistant telemetry collector")
	}
	if configurationService.getAllCalls != 0 {
		t.Fatalf("NewWithAssistantTelemetry should not load providers, got %d calls", configurationService.getAllCalls)
	}
	if err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 30}, observability.Context{}, observability.RecordLog{
		Level:   observability.LevelInfo,
		Message: "test",
	}); err != nil {
		t.Fatalf("expected unknown telemetry provider to be skipped without error, got %v", err)
	}
	if configurationService.getAllCalls != 1 {
		t.Fatalf("expected one telemetry provider load, got %d", configurationService.getAllCalls)
	}
}

func TestNew_SkipsAssistantTelemetryWithoutRequiredConfig(t *testing.T) {
	organizationID := uint64(10)
	projectID := uint64(20)
	auth := testServiceAuthentication(organizationID, projectID)
	if collector := NewWithAssistantTelemetry(context.Background(), testLogger(t), nil, 30, &recordingAssistantConfigurationService{}); collector != nil {
		t.Fatalf("expected no collector without auth, got %T", collector)
	}
	if collector := NewWithAssistantTelemetry(context.Background(), testLogger(t), auth, 0, &recordingAssistantConfigurationService{}); collector != nil {
		t.Fatalf("expected no collector without assistant ID, got %T", collector)
	}
	if collector := NewWithAssistantTelemetry(context.Background(), testLogger(t), auth, 30, nil); collector != nil {
		t.Fatalf("expected no collector without service, got %T", collector)
	}
}

func TestNew_AssistantTelemetryLoadsByAssistantID(t *testing.T) {
	organizationID := uint64(10)
	projectID := uint64(20)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			{
				Audited:           gorm_model.Audited{Id: 101},
				ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeTelemetry,
				Provider:          "logging",
				Enabled:           true,
			},
			{
				Audited:           gorm_model.Audited{Id: 202},
				ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeTelemetry,
				Provider:          "logging",
				Enabled:           false,
			},
		},
	}
	collector := NewWithAssistantTelemetry(context.Background(), testLogger(t), auth, 30, configurationService)
	if collector == nil {
		t.Fatal("expected assistant telemetry collector")
	}
	if collector.Key() != "telemetry:assistant:30" {
		t.Fatalf("expected assistant telemetry key, got %q", collector.Key())
	}
	if configurationService.getAllCalls != 0 {
		t.Fatalf("NewWithAssistantTelemetry should not load providers, got %d calls", configurationService.getAllCalls)
	}
	if err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 30}, observability.Context{}, observability.RecordLog{
		Level:   observability.LevelInfo,
		Message: "test",
	}); err != nil {
		t.Fatalf("expected assistant telemetry collect to succeed, got %v", err)
	}
	if configurationService.getAllCalls != 1 {
		t.Fatalf("expected telemetry provider load, got %d", configurationService.getAllCalls)
	}
	if configurationService.assistantID != 30 {
		t.Fatalf("expected assistant ID 30, got %d", configurationService.assistantID)
	}
}

func TestNew_AppendsWebhookCollector(t *testing.T) {
	organizationID := uint64(10)
	projectID := uint64(20)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			{
				ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeWebhook,
				Provider:          "http",
				Enabled:           true,
			},
		},
	}

	collector := NewWithWebhookConfiguration(context.Background(), testLogger(t), auth, 30, configurationService, &recordingHTTPLogService{})
	if collector == nil {
		t.Fatal("expected webhook collector")
	}
	if configurationService.getAllCalls != 0 {
		t.Fatalf("NewWithWebhookConfiguration should not load webhooks, got %d calls", configurationService.getAllCalls)
	}
}

func testServiceAuthentication(organizationID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeService,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}

type recordingAssistantConfigurationService struct {
	configurations []*internal_assistant_entity.AssistantConfiguration
	getAllCalls    int
	assistantID    uint64
}

func (s *recordingAssistantConfigurationService) Get(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantConfiguration, error) {
	return nil, nil
}

func (s *recordingAssistantConfigurationService) GetAll(_ context.Context, _ *types.Authentication, assistantID uint64, _ string, _ string, _ []*protos.Criteria, _ *protos.Paginate) (int64, []*internal_assistant_entity.AssistantConfiguration, error) {
	s.getAllCalls++
	s.assistantID = assistantID
	return int64(len(s.configurations)), s.configurations, nil
}

func (s *recordingAssistantConfigurationService) Create(context.Context, *types.Authentication, uint64, string, string, bool, []*protos.Metadata) (*internal_assistant_entity.AssistantConfiguration, error) {
	return nil, nil
}

func (s *recordingAssistantConfigurationService) Update(context.Context, *types.Authentication, uint64, uint64, string, string, bool, []*protos.Metadata) (*internal_assistant_entity.AssistantConfiguration, error) {
	return nil, nil
}

func (s *recordingAssistantConfigurationService) Delete(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantConfiguration, error) {
	return nil, nil
}

type recordingHTTPLogService struct{}

func (s *recordingHTTPLogService) CreateLog(context.Context, *types.Authentication, string, uint64, string, string, uint64, *uint64, string, string, int64, int64, uint32, type_enums.RecordState, *string, []byte, []byte) (*internal_assistant_entity.AssistantHTTPLog, error) {
	return nil, nil
}

func (s *recordingHTTPLogService) GetLog(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantHTTPLog, error) {
	return nil, nil
}

func (s *recordingHTTPLogService) GetAllLog(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate, *protos.Ordering) (int64, []*internal_assistant_entity.AssistantHTTPLog, error) {
	return 0, nil, nil
}

func (s *recordingHTTPLogService) GetLogObject(context.Context, *types.Authentication, uint64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (s *recordingHTTPLogService) RetryLog(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantHTTPLog, error) {
	return nil, nil
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()

	logger, err := commons.NewApplicationLogger(
		commons.Name("observability-collectors-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}
