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
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	"github.com/rapidaai/pkg/commons"
)

type callbackCallContextStore struct {
	callContext *callcontext.CallContext
}

func (store callbackCallContextStore) Save(context.Context, *callcontext.CallContext) (string, error) {
	return "", errors.New("not used")
}

func (store callbackCallContextStore) Get(context.Context, string) (*callcontext.CallContext, error) {
	return store.callContext, nil
}

func (store callbackCallContextStore) GetByChannelUUID(context.Context, string, uint64, string) (*callcontext.CallContext, error) {
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
		callContextStore: callbackCallContextStore{
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

func (store callbackCallContextStore) Claim(context.Context, string) (*callcontext.CallContext, error) {
	return nil, errors.New("not used")
}

func (store callbackCallContextStore) UpdateField(context.Context, string, string, string) error {
	return errors.New("not used")
}

func (store callbackCallContextStore) UpdateCallStatus(context.Context, string, callcontext.CallStatusUpdate) error {
	return errors.New("not used")
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
		callContextStore: callbackCallContextStore{
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
