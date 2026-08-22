package assistant_talk_api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
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
	return nil, errors.New("not used")
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
	api := &ConversationApi{
		logger: logger,
		callContextStore: callbackCallContextStore{
			callContext: &callcontext.CallContext{ContextID: maliciousContextID},
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
	logData, err := os.ReadFile(filepath.Join(logDirectory, "callback-security.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logData), maliciousContextID) || strings.Contains(string(logData), "FORGED LOG ENTRY") {
		t.Fatalf("user-provided context id was written to logs: %s", logData)
	}
}
