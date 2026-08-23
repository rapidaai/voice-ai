// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestCollector_SendsWebhookEventPayload(t *testing.T) {
	var got map[string]interface{}
	httpLogService := &recordingHTTPLogService{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("X-Test") != "yes" {
			t.Fatalf("X-Test header = %q, want yes", r.Header.Get("X-Test"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallRinging.String()}, map[string]interface{}{
				WebhookOptionHTTPURLKey:     server.URL,
				WebhookOptionHTTPHeadersKey: map[string]interface{}{"X-Test": "yes"},
			}),
		},
	}
	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                httpLogService,
	})

	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: 10},
		ConversationID: 20,
	}
	err := collector.Collect(context.Background(), scope, observability.Context{Auth: auth}, observability.RecordWebhook{
		ID:        "evt-1",
		Event:     observability.CallRinging,
		ContextID: "call-context-1",
		Payload: observability.CallRingingWebhookPayload{
			V1WebhookPayloadBase: observability.NewV1WebhookPayload(map[string]interface{}{"status": "ringing"}),
			Provider:             "test",
			CallID:               "call-1",
			StatusEvent:          "ringing",
		},
	})
	if err != nil {
		t.Fatalf("CollectWebhook returned error: %v", err)
	}
	assistantPayload, ok := got["assistant"].(map[string]interface{})
	if !ok || assistantPayload["id"] != "10" {
		t.Fatalf("unexpected assistant payload: %+v", got)
	}
	conversationPayload, ok := got["conversation"].(map[string]interface{})
	if !ok || conversationPayload["id"] != "20" {
		t.Fatalf("unexpected conversation payload: %+v", got)
	}
	dataPayload, ok := got["data"].(map[string]interface{})
	if !ok || dataPayload["status_event"] != "ringing" || dataPayload["call_id"] != "call-1" || dataPayload["version"] != observability.WebhookPayloadVersionV1 {
		t.Fatalf("unexpected data payload: %+v", got)
	}
	extraPayload, ok := dataPayload["extra"].(map[string]interface{})
	if !ok || extraPayload["status"] != "ringing" {
		t.Fatalf("unexpected data payload: %+v", got)
	}
	if got["event"] != observability.CallRinging.String() {
		t.Fatalf("unexpected event payload: %+v", got)
	}
	if _, ok := got["context_id"]; ok {
		t.Fatalf("webhook payload should not include observability context_id: %+v", got)
	}
	if len(httpLogService.calls) != 1 {
		t.Fatalf("expected one request log call, got %d", len(httpLogService.calls))
	}
	requestLogCall := httpLogService.calls[0]
	if requestLogCall.source != "webhook" || requestLogCall.sourceRefID != 1 || requestLogCall.sourceEvent != observability.CallRinging.String() {
		t.Fatalf("unexpected request log source: %+v", requestLogCall)
	}
	if requestLogCall.contextID != "call-context-1" {
		t.Fatalf("unexpected request log context_id: %q", requestLogCall.contextID)
	}
	if requestLogCall.assistantID != 10 || requestLogCall.conversationID == nil || *requestLogCall.conversationID != 20 {
		t.Fatalf("unexpected request log scope: %+v", requestLogCall)
	}
	if requestLogCall.status != type_enums.RECORD_COMPLETE || requestLogCall.responseStatus != http.StatusNoContent {
		t.Fatalf("unexpected request log status: %+v", requestLogCall)
	}
	var requestSnapshot map[string]interface{}
	if err := json.Unmarshal(requestLogCall.requestPayload, &requestSnapshot); err != nil {
		t.Fatalf("failed to decode request snapshot: %v", err)
	}
	if requestSnapshot["timeout_ms"] != float64(defaultWebhookTimeoutSeconds*1000) {
		t.Fatalf("timeout_ms = %v, want %d", requestSnapshot["timeout_ms"], defaultWebhookTimeoutSeconds*1000)
	}
}

func TestNew_DoesNotLoadWebhooks(t *testing.T) {
	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallRinging.String()}, map[string]interface{}{WebhookOptionHTTPURLKey: "https://example.com/webhook"}),
		},
	}

	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                &recordingHTTPLogService{},
	})
	if _, ok := collector.(observability.NoopCollector); ok {
		t.Fatal("expected webhook collector")
	}
	if configurationService.getAllCalls != 0 {
		t.Fatalf("New should not load webhooks, got %d calls", configurationService.getAllCalls)
	}
}

func TestCollector_DefaultsTooLargeTimeoutSeconds(t *testing.T) {
	httpLogService := &recordingHTTPLogService{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallRinging.String()}, map[string]interface{}{
				WebhookOptionHTTPURLKey:        server.URL,
				WebhookOptionTimeoutSecondsKey: uint32(defaultWebhookTimeoutSeconds + 1),
			}),
		},
	}
	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                httpLogService,
	})

	err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{Auth: auth}, observability.RecordWebhook{
		Event: observability.CallRinging,
	})
	if err != nil {
		t.Fatalf("CollectWebhook returned error: %v", err)
	}
	if len(httpLogService.calls) != 1 {
		t.Fatalf("expected one request log call, got %d", len(httpLogService.calls))
	}
	var requestSnapshot map[string]interface{}
	if err := json.Unmarshal(httpLogService.calls[0].requestPayload, &requestSnapshot); err != nil {
		t.Fatalf("failed to decode request snapshot: %v", err)
	}
	if requestSnapshot["timeout_ms"] != float64(defaultWebhookTimeoutSeconds*1000) {
		t.Fatalf("timeout_ms = %v, want %d", requestSnapshot["timeout_ms"], defaultWebhookTimeoutSeconds*1000)
	}
}

func TestCollector_LoadsWebhooksOnce(t *testing.T) {
	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallFailed.String()}, map[string]interface{}{WebhookOptionHTTPURLKey: "https://example.com/webhook"}),
		},
	}
	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                &recordingHTTPLogService{},
	})

	for i := 0; i < 2; i++ {
		err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{}, observability.RecordWebhook{
			Event: observability.CallRinging,
		})
		if err != nil {
			t.Fatalf("CollectWebhook returned error: %v", err)
		}
	}
	if configurationService.getAllCalls != 1 {
		t.Fatalf("expected one webhook load, got %d", configurationService.getAllCalls)
	}
}

func TestCollector_IgnoresUnallowedWebhookEvent(t *testing.T) {
	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallFailed.String()}, map[string]interface{}{WebhookOptionHTTPURLKey: "https://example.com/webhook"}),
		},
	}
	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                &recordingHTTPLogService{},
	})

	err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{}, observability.RecordWebhook{
		Event: observability.CallRinging,
	})
	if err != nil {
		t.Fatalf("CollectWebhook returned error: %v", err)
	}
}

func TestCollector_ShouldSendRequiresValidWebhookProvider(t *testing.T) {
	collector := &Collector{}
	webhook := testWebhook(1, []string{observability.CallRinging.String()}, nil)

	webhook.Provider = "HTTP"
	if collector.shouldSend(webhook, observability.CallRinging.String()) {
		t.Fatal("expected uppercase provider to be invalid")
	}

	webhook.Provider = " http"
	if collector.shouldSend(webhook, observability.CallRinging.String()) {
		t.Fatal("expected provider with leading space to be invalid")
	}

	webhook.Provider = "http"
	if !collector.shouldSend(webhook, observability.CallRinging.String()) {
		t.Fatal("expected exact http provider to be valid")
	}
}

func TestCollector_ReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	organizationID := uint64(30)
	projectID := uint64(40)
	auth := testServiceAuthentication(organizationID, projectID)
	configurationService := &recordingAssistantConfigurationService{
		configurations: []*internal_assistant_entity.AssistantConfiguration{
			testWebhook(1, []string{observability.CallFailed.String()}, map[string]interface{}{WebhookOptionHTTPURLKey: server.URL}),
		},
	}
	collector := New(context.Background(), Config{
		Logger:                        testLogger(t),
		Auth:                          auth,
		AssistantID:                   10,
		AssistantConfigurationService: configurationService,
		HTTPLogService:                &recordingHTTPLogService{},
	})

	err := collector.Collect(context.Background(), observability.AssistantScope{AssistantID: 10}, observability.Context{}, observability.RecordWebhook{
		Event: observability.CallFailed,
		Payload: observability.CallFailedWebhookPayload{
			V1WebhookPayloadBase: observability.NewV1WebhookPayload(nil),
			Reason:               "failed",
		},
	})
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func testWebhook(id uint64, events []string, options map[string]interface{}) *internal_assistant_entity.AssistantConfiguration {
	webhook := &internal_assistant_entity.AssistantConfiguration{
		Audited:           gorm_model.Audited{Id: id},
		Provider:          "http",
		Enabled:           true,
		ConfigurationType: internal_assistant_entity.AssistantConfigurationTypeWebhook,
	}
	eventsJSON, _ := json.Marshal(events)
	webhook.Options = append(webhook.Options, &internal_assistant_entity.AssistantConfigurationOption{
		Metadata: *gorm_model.NewMetadata("assistant_events", string(eventsJSON)),
	})
	for key, value := range options {
		webhook.Options = append(webhook.Options, &internal_assistant_entity.AssistantConfigurationOption{
			Metadata: *gorm_model.NewMetadata(key, value),
		})
	}
	return webhook
}

type recordingAssistantConfigurationService struct {
	configurations []*internal_assistant_entity.AssistantConfiguration
	getAllCalls    int
}

type webhookHTTPLogCall struct {
	source         string
	sourceRefID    uint64
	sourceEvent    string
	contextID      string
	assistantID    uint64
	conversationID *uint64
	responseStatus int64
	status         type_enums.RecordState
	requestPayload []byte
}

type recordingHTTPLogService struct {
	calls []webhookHTTPLogCall
}

func (s *recordingAssistantConfigurationService) Get(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantConfiguration, error) {
	return nil, nil
}

func (s *recordingAssistantConfigurationService) GetAll(context.Context, *types.Authentication, uint64, string, string, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantConfiguration, error) {
	s.getAllCalls++
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

func (s *recordingHTTPLogService) CreateLog(
	_ context.Context,
	_ *types.Authentication,
	source string,
	sourceRefID uint64,
	sourceEvent string,
	contextID string,
	assistantID uint64,
	conversationID *uint64,
	_ string,
	_ string,
	responseStatus int64,
	_ int64,
	_ uint32,
	status type_enums.RecordState,
	_ *string,
	requestPayload []byte,
	_ []byte,
) (*internal_assistant_entity.AssistantHTTPLog, error) {
	s.calls = append(s.calls, webhookHTTPLogCall{
		source:         source,
		sourceRefID:    sourceRefID,
		sourceEvent:    sourceEvent,
		contextID:      contextID,
		assistantID:    assistantID,
		conversationID: conversationID,
		responseStatus: responseStatus,
		status:         status,
		requestPayload: requestPayload,
	})
	return &internal_assistant_entity.AssistantHTTPLog{}, nil
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
		commons.Name("observability-webhook-collector-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

func testServiceAuthentication(organizationID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeService,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}
