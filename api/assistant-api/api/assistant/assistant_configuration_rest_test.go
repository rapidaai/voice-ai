package assistant_api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
)

func TestAssistantConfigurationRestAllowsProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantAPI := &assistantGrpcApi{}
	tests := []struct {
		name    string
		method  string
		path    string
		params  gin.Params
		body    []byte
		handler gin.HandlerFunc
	}{
		{name: "create", method: http.MethodPost, path: "/v1/assistant/configurations", body: []byte(`{}`), handler: assistantAPI.CreateAssistantConfigurationRest},
		{name: "update", method: http.MethodPatch, path: "/v1/assistant/configurations/invalid/invalid", params: gin.Params{{Key: "assistantId", Value: "invalid"}, {Key: "id", Value: "invalid"}}, body: []byte(`{}`), handler: assistantAPI.UpdateAssistantConfigurationRest},
		{name: "get", method: http.MethodGet, path: "/v1/assistant/configurations/invalid/invalid", params: gin.Params{{Key: "assistantId", Value: "invalid"}, {Key: "id", Value: "invalid"}}, handler: assistantAPI.GetAssistantConfigurationRest},
		{name: "list", method: http.MethodGet, path: "/v1/assistant/configurations/invalid", params: gin.Params{{Key: "assistantId", Value: "invalid"}}, handler: assistantAPI.GetAllAssistantConfigurationRest},
		{name: "delete", method: http.MethodDelete, path: "/v1/assistant/configurations/invalid/invalid", params: gin.Params{{Key: "assistantId", Value: "invalid"}, {Key: "id", Value: "invalid"}}, handler: assistantAPI.DeleteAssistantConfigurationRest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(test.method, test.path, bytes.NewReader(test.body))
			context.Request.Header.Set("Content-Type", "application/json")
			context.Params = test.params
			attachTestAuthentication(context, testProjectAuthentication(22, 33))

			test.handler(context)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestAssistantConfigurationOpenAPIIncludesAuditActors(t *testing.T) {
	configuration := &internal_assistant_entity.AssistantConfiguration{
		Audited: gorm_model.Audited{Id: 17},
		Mutable: gorm_model.Mutable{ActorAudit: gorm_model.ActorAudit{
			CreatedActor: &types.ActorIdentity{Type: types.ActorTypeService, ID: 41},
			UpdatedActor: &types.ActorIdentity{Type: types.ActorTypeSystem, ID: 42},
		}},
	}

	result := assistantConfigurationOpenAPI(configuration)

	require.NotNil(t, result.CreatedActor)
	require.Equal(t, "service", string(result.CreatedActor.Type))
	require.Equal(t, "41", string(result.CreatedActor.Id))
	require.NotNil(t, result.UpdatedActor)
	require.Equal(t, "system", string(result.UpdatedActor.Type))
	require.Equal(t, "42", string(result.UpdatedActor.Id))
}

func TestAssistantConfigurationOpenAPIOmitsInvalidAuditActor(t *testing.T) {
	configuration := &internal_assistant_entity.AssistantConfiguration{
		Mutable: gorm_model.Mutable{ActorAudit: gorm_model.ActorAudit{
			CreatedActor: &types.ActorIdentity{Type: types.ActorTypeUnknown},
		}},
	}

	result := assistantConfigurationOpenAPI(configuration)

	require.Nil(t, result.CreatedActor)
}
