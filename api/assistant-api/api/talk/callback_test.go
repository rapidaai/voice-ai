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
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type callbackCallContextStore struct {
	callContext *callcontext.CallContext
	updates     []callcontext.CallStatusUpdate
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
		logger: logger,
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
		logger: logger,
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
