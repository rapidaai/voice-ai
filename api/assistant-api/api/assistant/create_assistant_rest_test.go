package assistant_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createAssistantRestAssistantServiceStub struct {
	createAssistantCalled bool
	createProviderCalled  bool
	attachProviderCalled  bool
	createTagCalled       bool
	providerDescription   string
	createAssistantErr    error
	createContextAuthErr  error
}

func attachTestAuthentication(ginContext *gin.Context, auth *types.Authentication) {
	ginContext.Request = ginContext.Request.WithContext(context.WithValue(ginContext.Request.Context(), types.CTX_, auth))
}

func testUserAuthentication(userID, organizationID, projectID uint64) *types.Authentication {
	actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: userID}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &types.UserContext{UserID: userID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}
	if projectID != 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}
	}
	return auth
}

func testProjectAuthentication(organizationID, projectID uint64) *types.Authentication {
	actor := types.ActorIdentity{Type: types.ActorTypeProject, ID: projectID}
	return &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}

func (s *createAssistantRestAssistantServiceStub) Get(context.Context, *types.Authentication, uint64, *uint64, *internal_services.GetAssistantOption) (*internal_assistant_entity.Assistant, error) {
	return nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAll(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate, *internal_services.GetAssistantOption) (int64, []*internal_assistant_entity.Assistant, error) {
	return 0, nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAssistantDashboard(context.Context, *types.Authentication, uint64, *timestamppb.Timestamp, *timestamppb.Timestamp) (*protos.AssistantDashboard, error) {
	return nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAllAssistantProviderModel(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantProviderModel, error) {
	return 0, nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAllAssistantProviderWebsocket(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantProviderWebsocket, error) {
	return 0, nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAllAssistantProviderAgentkit(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantProviderAgentkit, error) {
	return 0, nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) GetAllAssistantProviderAgentflow(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantProviderAgentflow, error) {
	return 0, nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) UpdateAssistantVersion(context.Context, *types.Authentication, uint64, type_enums.AssistantProvider, uint64) (*internal_assistant_entity.Assistant, error) {
	return nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) UpdateAssistantDetail(context.Context, *types.Authentication, uint64, string, string) (*internal_assistant_entity.Assistant, error) {
	return nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) CreateAssistant(ctx context.Context, auth *types.Authentication, name, description string, visibility string, source string, sourceIdentifier *uint64, language string) (*internal_assistant_entity.Assistant, error) {
	s.createAssistantCalled = true
	_, s.createContextAuthErr = types.Authorize(ctx)
	if s.createAssistantErr != nil {
		return nil, s.createAssistantErr
	}
	return &internal_assistant_entity.Assistant{
		Name:        name,
		Description: description,
		Visibility:  visibility,
	}, nil
}

func (s *createAssistantRestAssistantServiceStub) DeleteAssistant(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.Assistant, error) {
	return nil, errors.New("not implemented")
}

func (s *createAssistantRestAssistantServiceStub) CreateAssistantProviderModel(_ context.Context, _ *types.Authentication, _ uint64, providerDescription string, _ string, _ string, _ []*protos.Metadata) (*internal_assistant_entity.AssistantProviderModel, error) {
	s.createProviderCalled = true
	s.providerDescription = providerDescription
	providerModel := &internal_assistant_entity.AssistantProviderModel{}
	providerModel.Id = 2
	return providerModel, nil
}

func (s *createAssistantRestAssistantServiceStub) CreateAssistantProviderWebsocket(_ context.Context, _ *types.Authentication, _ uint64, providerDescription string, _ string, _ map[string]string, _ map[string]string) (*internal_assistant_entity.AssistantProviderWebsocket, error) {
	s.createProviderCalled = true
	s.providerDescription = providerDescription
	websocketProvider := &internal_assistant_entity.AssistantProviderWebsocket{}
	websocketProvider.Id = 2
	return websocketProvider, nil
}

func (s *createAssistantRestAssistantServiceStub) CreateAssistantProviderAgentkit(_ context.Context, _ *types.Authentication, _ uint64, providerDescription string, _ string, _ string, _ map[string]string, _ *string, _ *string, _ *string, _ *uint32, _ *uint32, _ *uint32, _ *uint32, _ *uint32) (*internal_assistant_entity.AssistantProviderAgentkit, error) {
	s.createProviderCalled = true
	s.providerDescription = providerDescription
	agentkitProvider := &internal_assistant_entity.AssistantProviderAgentkit{}
	agentkitProvider.Id = 2
	return agentkitProvider, nil
}

func (s *createAssistantRestAssistantServiceStub) CreateAssistantProviderAgentflow(_ context.Context, _ *types.Authentication, _ uint64, providerDescription string, _ string, _ map[string]interface{}) (*internal_assistant_entity.AssistantProviderAgentflow, error) {
	s.createProviderCalled = true
	s.providerDescription = providerDescription
	agentflowProvider := &internal_assistant_entity.AssistantProviderAgentflow{}
	agentflowProvider.Id = 2
	return agentflowProvider, nil
}

func (s *createAssistantRestAssistantServiceStub) AttachProviderModelToAssistant(context.Context, *types.Authentication, uint64, type_enums.AssistantProvider, uint64) (*internal_assistant_entity.Assistant, error) {
	s.attachProviderCalled = true
	return &internal_assistant_entity.Assistant{}, nil
}

func (s *createAssistantRestAssistantServiceStub) CreateOrUpdateAssistantTag(context.Context, *types.Authentication, uint64, []string) (*internal_assistant_entity.AssistantTag, error) {
	s.createTagCalled = true
	return &internal_assistant_entity.AssistantTag{}, nil
}

type createAssistantRestKnowledgeServiceStub struct{}

func (s createAssistantRestKnowledgeServiceStub) Get(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantKnowledge, error) {
	return nil, errors.New("not implemented")
}

func (s createAssistantRestKnowledgeServiceStub) GetAll(context.Context, *types.Authentication, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_assistant_entity.AssistantKnowledge, error) {
	return 0, nil, errors.New("not implemented")
}

func (s createAssistantRestKnowledgeServiceStub) Create(context.Context, *types.Authentication, uint64, uint64, gorm_types.RetrievalMethod, bool, float32, uint32, *uint64, *string, []*protos.Metadata) (*internal_assistant_entity.AssistantKnowledge, error) {
	return &internal_assistant_entity.AssistantKnowledge{}, nil
}

func (s createAssistantRestKnowledgeServiceStub) Update(context.Context, *types.Authentication, uint64, uint64, uint64, gorm_types.RetrievalMethod, bool, float32, uint32, *uint64, *string, []*protos.Metadata) (*internal_assistant_entity.AssistantKnowledge, error) {
	return nil, errors.New("not implemented")
}

func (s createAssistantRestKnowledgeServiceStub) Delete(context.Context, *types.Authentication, uint64, uint64) (*internal_assistant_entity.AssistantKnowledge, error) {
	return nil, errors.New("not implemented")
}

func TestCreateAssistantRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantApi{
		assistantService:          assistantService,
		assistantKnowledgeService: createAssistantRestKnowledgeServiceStub{},
	}
	requestBody := []byte(`{
		"name": "Support Assistant",
		"description": "Handles support calls",
		"assistantProvider": {
			"description": "Primary model",
			"model": {
				"modelProviderName": "openai",
				"template": {
					"prompt": [{"role": "system", "content": "Help users"}]
				}
			}
		},
		"tags": ["support"]
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createAssistantRestAuth())

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, assistantService.createAssistantCalled)
	require.NoError(t, assistantService.createContextAuthErr)
	assert.True(t, assistantService.createProviderCalled)
	assert.True(t, assistantService.attachProviderCalled)
	assert.True(t, assistantService.createTagCalled)
	assert.Equal(t, "Primary model", assistantService.providerDescription)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
}

func TestCreateAssistantRest_ProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantApi{
		assistantService:          assistantService,
		assistantKnowledgeService: createAssistantRestKnowledgeServiceStub{},
	}
	requestBody := []byte(`{
		"name": "Project Assistant",
		"assistantProvider": {
			"model": {
				"modelProviderName": "openai",
				"template": {
					"prompt": [{"role": "system", "content": "Help users"}]
				}
			}
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, testProjectAuthentication(22, 33))

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, assistantService.createAssistantCalled)
	require.NoError(t, assistantService.createContextAuthErr)
	assert.True(t, assistantService.createProviderCalled)
	assert.True(t, assistantService.attachProviderCalled)
}

func TestCreateAssistantRest_MissingAuthScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantApi := &assistantApi{}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader([]byte(`{}`)))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, testUserAuthentication(11, 22, 0))

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateAssistantRest_MissingName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantApi := &assistantApi{}
	requestBody := []byte(`{"assistantProvider":{"model":{"modelProviderName":"openai"}}}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createAssistantRestAuth())

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantMissingName.Error)
}

func TestCreateAssistantRest_CreateAssistantErrorDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantApi := &assistantApi{
		assistantService: &createAssistantRestAssistantServiceStub{
			createAssistantErr: errors.New("database password leaked"),
		},
	}
	requestBody := []byte(`{"name":"Support Assistant","assistantProvider":{"model":{"modelProviderName":"openai"}}}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createAssistantRestAuth())

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantCreateAssistant.Error)
	assert.NotContains(t, recorder.Body.String(), "database password leaked")
}

func TestCreateAssistantRest_InvalidAgentkitCertificateDoesNotCreateAssistant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantApi{
		assistantService: assistantService,
	}
	requestBody := []byte(`{
		"name": "Support Assistant",
		"assistantProvider": {
			"agentkit": {
				"agentKitUrl": "localhost:9000",
				"certificate": "skip-verify"
			}
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createAssistantRestAuth())

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, assistantService.createAssistantCalled)
	assert.Contains(t, recorder.Body.String(), "certificate must be a CA PEM")

	var response openapi.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotNil(t, response.Code)
	require.NotNil(t, response.Error)
	require.NotNil(t, response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateAssistantInvalidAgentKitCertificate.HTTPStatusCodeInt32(), *response.Code)
	assert.Equal(t, openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitCertificate.CodeString()), *response.Error.ErrorCode)
}

func TestCreateAssistantRest_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assistantApi := &assistantApi{}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/assistant/create-assistant", bytes.NewReader([]byte(`{}`)))
	context.Request.Header.Set("Content-Type", "application/json")

	assistantApi.CreateAssistantRest(context)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantUnauthenticated.Error)
}

func createAssistantRestAuth() *types.Authentication {
	return testUserAuthentication(11, 22, 33)
}
