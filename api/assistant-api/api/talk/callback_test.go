package assistant_talk_api

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type callbackCallContextStore struct {
	callContext *callcontext.CallContext
	updates     []callcontext.CallStatusUpdate
}

type callbackConversationServiceStub struct {
	internal_services.AssistantConversationService

	mu             sync.Mutex
	metrics        []*protos.Metric
	metricRecorded chan struct{}
	metricOnce     sync.Once
}

func (service *callbackConversationServiceStub) CreateOrUpdateConversationMetrics(
	_ context.Context,
	_ *types.Authentication,
	_ uint64,
	_ uint64,
	metrics []*protos.Metric,
) ([]*internal_conversation_entity.AssistantConversationMetric, error) {
	service.mu.Lock()
	service.metrics = append(service.metrics, metrics...)
	service.mu.Unlock()
	service.metricOnce.Do(func() { close(service.metricRecorded) })
	return nil, nil
}

func (service *callbackConversationServiceStub) CreateOrUpdateConversationMetadata(
	context.Context,
	*types.Authentication,
	uint64,
	uint64,
	[]*protos.Metadata,
) ([]*internal_conversation_entity.AssistantConversationMetadata, error) {
	return nil, nil
}

func (service *callbackConversationServiceStub) recordedMetrics() []*protos.Metric {
	service.mu.Lock()
	defer service.mu.Unlock()
	return append([]*protos.Metric(nil), service.metrics...)
}

func (store *callbackCallContextStore) Save(context.Context, *callcontext.CallContext) (string, error) {
	return "", errors.New("not used")
}

func (store *callbackCallContextStore) Get(context.Context, string) (*callcontext.CallContext, error) {
	return store.callContext, nil
}

func (store *callbackCallContextStore) GetByChannelUUID(context.Context, string, uint64, string) (*callcontext.CallContext, error) {
	return store.callContext, nil
}

func TestUniversalCallbackDoesNotLogPersistedAuthenticationMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logDirectory := t.TempDir()
	logger, err := commons.NewApplicationLogger(
		commons.Name("universal-callback-security"),
		commons.Path(logDirectory),
		commons.EnableConsole(false),
		commons.EnableFile(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	maliciousContextID := "context-id\nFORGED LOG ENTRY"
	maliciousAuthType := "invalid-auth-type\nFORGED AUTH LOG ENTRY"
	api := &ConversationApi{
		logger:       logger,
		rapidaClient: &rapida_client.RapidaClient{},
		callContextStore: &callbackCallContextStore{
			callContext: &callcontext.CallContext{ContextID: maliciousContextID, AuthType: maliciousAuthType},
		},
		inboundDispatcher: channel_telephony.NewInboundDispatcher(channel_telephony.WithLogger(logger)),
	}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/callback?Event=Hangup&CallUUID="+url.QueryEscape(maliciousContextID)+"&CallStatus=completed", nil)
	ginContext.Params = gin.Params{{Key: "telephony", Value: "vobiz"}, {Key: "assistantId", Value: "1"}}

	api.UnviersalCallback(ginContext)
	if err := logger.Sync(); err != nil {
		t.Fatal(err)
	}

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	logData, err := fs.ReadFile(os.DirFS(logDirectory), "universal-callback-security.log")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logData), maliciousContextID) || strings.Contains(string(logData), maliciousAuthType) || strings.Contains(string(logData), "FORGED") {
		t.Fatalf("persisted callback data was written to logs: %s", logData)
	}
}

func (store *callbackCallContextStore) Claim(context.Context, string) (*callcontext.CallContext, error) {
	return nil, errors.New("not used")
}

func (store *callbackCallContextStore) UpdateField(context.Context, string, string, string) error {
	return errors.New("not used")
}

func (store *callbackCallContextStore) UpdateCallStatus(_ context.Context, _ string, update callcontext.CallStatusUpdate) error {
	store.updates = append(store.updates, update)
	return nil
}

func TestCallbackByContextDoesNotLogUserProvidedContextID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logDirectory := t.TempDir()
	logger, err := commons.NewApplicationLogger(
		commons.Name("callback-security"),
		commons.Path(logDirectory),
		commons.EnableConsole(false),
		commons.EnableFile(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	maliciousContextID := "context-id\nFORGED LOG ENTRY"
	maliciousAuthType := "invalid-auth-type\nFORGED AUTH LOG ENTRY"
	api := &ConversationApi{
		logger:       logger,
		rapidaClient: &rapida_client.RapidaClient{},
		callContextStore: &callbackCallContextStore{
			callContext: &callcontext.CallContext{ContextID: maliciousContextID, AuthType: maliciousAuthType},
		},
	}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/callback", nil)
	ginContext.Params = gin.Params{{Key: "contextId", Value: maliciousContextID}}

	api.CallbackByContext(ginContext)
	if err := logger.Sync(); err != nil {
		t.Fatal(err)
	}

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	logData, err := fs.ReadFile(os.DirFS(logDirectory), "callback-security.log")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logData), maliciousContextID) || strings.Contains(string(logData), maliciousAuthType) || strings.Contains(string(logData), "FORGED") {
		t.Fatalf("user-provided callback data was written to logs: %s", logData)
	}
}

func TestCallbackByContextRestoresAuthenticationForEveryProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(false))
	if err != nil {
		t.Fatal(err)
	}

	providerRequests := []struct {
		name        string
		method      string
		body        string
		contentType string
	}{
		{name: "twilio", method: http.MethodPost, body: "CallSid=twilio-call&CallStatus=completed", contentType: "application/x-www-form-urlencoded"},
		{name: "exotel", method: http.MethodPost, body: "CallSid=exotel-call&Status=completed", contentType: "application/x-www-form-urlencoded"},
		{name: "vonage", method: http.MethodGet, body: "status=completed&uuid=vonage-call&duration=1"},
		{name: "telnyx", method: http.MethodPost, body: `{"data":{"event_type":"call.hangup","payload":{"call_control_id":"telnyx-call"}}}`, contentType: "application/json"},
		{name: "asterisk", method: http.MethodPost, body: `{"type":"ChannelDestroyed","channel":{"id":"asterisk-call"},"cause":16,"cause_txt":"NORMAL_CLEARING"}`, contentType: "application/json"},
		{name: "sip", method: http.MethodPost, body: `{"event":"completed","call_id":"sip-call"}`, contentType: "application/json"},
		{name: "vobiz", method: http.MethodPost, body: "Event=Hangup&CallUUID=vobiz-call&CallStatus=completed", contentType: "application/x-www-form-urlencoded"},
	}

	projectActorType := types.ActorTypeProject.String()
	userActorType := types.ActorTypeUser.String()
	userID := uint64(31)
	authContexts := []struct {
		name       string
		authType   types.AuthType
		authUserID *uint64
		actorType  *string
		actorID    uint64
	}{
		{name: "project API key", authType: types.AuthTypeProject, actorType: &projectActorType, actorID: 41},
		{name: "personal access token", authType: types.AuthTypeUser, authUserID: &userID, actorType: &userActorType, actorID: userID},
	}

	for _, authContext := range authContexts {
		for _, providerRequest := range providerRequests {
			t.Run(authContext.name+"/"+providerRequest.name, func(t *testing.T) {
				store := &callbackCallContextStore{callContext: &callcontext.CallContext{
					ContextID:      "callback-context",
					AssistantID:    11,
					ConversationID: 22,
					ProjectID:      21,
					OrganizationID: 20,
					AuthType:       authContext.authType.String(),
					AuthUserID:     authContext.authUserID,
					AuthActorType:  authContext.actorType,
					AuthActorID:    &authContext.actorID,
					Provider:       providerRequest.name,
					Direction:      "outbound",
				}}
				api := &ConversationApi{
					cfg:              &config.AssistantConfig{},
					logger:           logger,
					rapidaClient:     &rapida_client.RapidaClient{},
					callContextStore: store,
					inboundDispatcher: channel_telephony.NewInboundDispatcher(
						channel_telephony.WithConfig(&config.AssistantConfig{}),
						channel_telephony.WithLogger(logger),
						channel_telephony.WithTelephonyOption(channel_telephony.TelephonyOption{SIPServer: &sip_infra.Server{}}),
					),
				}

				requestTarget := "/callback"
				requestBody := strings.NewReader(providerRequest.body)
				if providerRequest.method == http.MethodGet {
					requestTarget += "?" + providerRequest.body
					requestBody = strings.NewReader("")
				}
				recorder := httptest.NewRecorder()
				ginContext, _ := gin.CreateTestContext(recorder)
				ginContext.Request = httptest.NewRequest(providerRequest.method, requestTarget, requestBody)
				if providerRequest.contentType != "" {
					ginContext.Request.Header.Set("Content-Type", providerRequest.contentType)
				}
				ginContext.Params = gin.Params{{Key: "contextId", Value: store.callContext.ContextID}}

				api.CallbackByContext(ginContext)

				if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
					t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
				}
				if len(store.updates) != 1 || store.updates[0].CallStatus != callcontext.CallStatusCompleted {
					t.Fatalf("updates = %+v, want one completed update", store.updates)
				}
			})
		}
	}
}

func TestCallbackByContextRejectsInvalidStoredAuthenticationForEveryProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(false))
	if err != nil {
		t.Fatal(err)
	}

	for _, provider := range []string{"twilio", "exotel", "vonage", "telnyx", "asterisk", "sip", "vobiz"} {
		t.Run(provider, func(t *testing.T) {
			store := &callbackCallContextStore{callContext: &callcontext.CallContext{
				ContextID:      "invalid-auth-context",
				AssistantID:    11,
				ConversationID: 22,
				Provider:       provider,
			}}
			api := &ConversationApi{
				logger:           logger,
				rapidaClient:     &rapida_client.RapidaClient{},
				callContextStore: store,
				inboundDispatcher: channel_telephony.NewInboundDispatcher(
					channel_telephony.WithConfig(&config.AssistantConfig{}),
					channel_telephony.WithLogger(logger),
					channel_telephony.WithTelephonyOption(channel_telephony.TelephonyOption{SIPServer: &sip_infra.Server{}}),
				),
			}
			recorder := httptest.NewRecorder()
			ginContext, _ := gin.CreateTestContext(recorder)
			ginContext.Request = httptest.NewRequest(http.MethodPost, "/callback", nil)
			ginContext.Params = gin.Params{{Key: "contextId", Value: store.callContext.ContextID}}

			api.CallbackByContext(ginContext)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
			if len(store.updates) != 0 {
				t.Fatalf("unexpected updates: %+v", store.updates)
			}
		})
	}
}

func TestFailedCallbacksPersistCallAndConversationStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(false))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		target string
		params gin.Params
		invoke func(*ConversationApi, *gin.Context)
	}{
		{
			name:   "context callback",
			target: "/callback",
			params: gin.Params{{Key: "contextId", Value: "callback-context"}},
			invoke: (*ConversationApi).CallbackByContext,
		},
		{
			name:   "catch-all callback",
			target: "/callback?CallSid=twilio-call&CallStatus=busy",
			params: gin.Params{{Key: "telephony", Value: "twilio"}, {Key: "assistantId", Value: "11"}},
			invoke: (*ConversationApi).UnviersalCallback,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actorType := types.ActorTypeProject.String()
			actorID := uint64(41)
			store := &callbackCallContextStore{callContext: &callcontext.CallContext{
				ContextID:      "callback-context",
				AssistantID:    11,
				ConversationID: 22,
				ProjectID:      21,
				OrganizationID: 20,
				AuthType:       types.AuthTypeProject.String(),
				AuthActorType:  &actorType,
				AuthActorID:    &actorID,
				Provider:       "twilio",
				Direction:      "outbound",
			}}
			conversationService := &callbackConversationServiceStub{metricRecorded: make(chan struct{})}
			api := &ConversationApi{
				cfg:                          &config.AssistantConfig{},
				logger:                       logger,
				rapidaClient:                 &rapida_client.RapidaClient{},
				callContextStore:             store,
				assistantConversationService: conversationService,
				inboundDispatcher: channel_telephony.NewInboundDispatcher(
					channel_telephony.WithConfig(&config.AssistantConfig{}),
					channel_telephony.WithLogger(logger),
				),
			}

			requestBody := strings.NewReader("CallSid=twilio-call&CallStatus=busy")
			if test.name == "catch-all callback" {
				requestBody = strings.NewReader("")
			}
			recorder := httptest.NewRecorder()
			ginContext, _ := gin.CreateTestContext(recorder)
			ginContext.Request = httptest.NewRequest(http.MethodPost, test.target, requestBody)
			ginContext.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			ginContext.Params = test.params

			test.invoke(api, ginContext)

			if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
				t.Fatalf("status = %d, want 2xx", recorder.Code)
			}
			if len(store.updates) != 1 || store.updates[0].CallStatus != callcontext.CallStatusFailed {
				t.Fatalf("updates = %+v, want one failed update", store.updates)
			}
			select {
			case <-conversationService.metricRecorded:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for callback metrics")
			}
			assertFailedStatusMetrics(t, conversationService.recordedMetrics(), "busy")
		})
	}
}

func assertFailedStatusMetrics(t *testing.T, metrics []*protos.Metric, reason string) {
	t.Helper()
	if len(metrics) != 2 {
		t.Fatalf("metric count = %d, want 2", len(metrics))
	}

	values := make(map[string]string, len(metrics))
	descriptions := make(map[string]string, len(metrics))
	for _, metric := range metrics {
		values[metric.GetName()] = metric.GetValue()
		descriptions[metric.GetName()] = metric.GetDescription()
	}

	for _, name := range []string{observability.MetricCallStatus, observability.MetricConversationStatus} {
		if values[name] != observability.MetricCallStatusFailed {
			t.Errorf("metric %q = %q, want %q", name, values[name], observability.MetricCallStatusFailed)
		}
		if descriptions[name] != reason {
			t.Errorf("metric %q description = %q, want %q", name, descriptions[name], reason)
		}
	}
}
