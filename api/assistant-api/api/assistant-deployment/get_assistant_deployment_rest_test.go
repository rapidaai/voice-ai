package assistant_deployment_api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAssistantApiDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-api-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	inputAudio := data["inputAudio"].(map[string]interface{})
	assert.Equal(t, "twilio", inputAudio["audioProvider"])
}

func TestGetAssistantApiDeploymentRest_AllowsProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-api-deployment/invalid", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "invalid"}}
	attachTestAuthentication(context, testProjectAuthentication(22, 33))

	deploymentApi.GetAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetAssistantDebuggerDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-debugger-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
}

func TestGetAssistantPhoneDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-phone-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "twilio", data["phoneProviderName"])
}

func TestGetAssistantWebpluginDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-webplugin-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantWebpluginDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, []interface{}{"Book demo", "Talk to support"}, data["suggestion"])
}

func TestGetAssistantWhatsappDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-whatsapp-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantWhatsappDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "gupshup", data["whatsappProviderName"])
}

func TestGetAssistantApiDeploymentRest_InvalidAssistantID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-api-deployment/abc", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "abc"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, service.getCalled)
	assert.Contains(t, recorder.Body.String(), pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.Error)
}

func TestGetAssistantPhoneDeploymentRest_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-phone-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}

	deploymentApi.GetAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.GetAssistantPhoneDeploymentUnauthenticated.Error)
}

func TestGetAssistantWebpluginDeploymentRest_GetDeploymentErrorDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{
		getErr: errors.New("database password leaked"),
	}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-webplugin-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantWebpluginDeploymentRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.GetAssistantWebpluginDeploymentGetDeployment.Error)
	assert.NotContains(t, recorder.Body.String(), "database password leaked")
}

func TestGetAssistantWhatsappDeploymentRest_NotFoundReturnsSuccessWithNilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{
		getNil: true,
	}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-whatsapp-deployment/123", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAssistantWhatsappDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	assert.Nil(t, response["data"])
}

func TestGetAllAssistantApiDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)
	criterias := url.QueryEscape(`[{"key":"status","logic":"=","value":"ACTIVE"}]`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/assistant-deployment/get-all-api-deployment/123?page=2&pageSize=10&criterias="+criterias,
		nil,
	)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAllAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.getAllCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	assert.Equal(t, uint32(2), service.page)
	assert.Equal(t, uint32(10), service.pageSize)
	require.Len(t, service.criterias, 1)
	assert.Equal(t, "status", service.criterias[0].GetKey())
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].([]interface{})
	require.Len(t, data, 1)
	paginated := response["paginated"].(map[string]interface{})
	assert.Equal(t, float64(1), paginated["totalItem"])
	assert.Equal(t, float64(2), paginated["currentPage"])
}

func TestGetAllAssistantApiDeploymentRest_AllowsProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/assistant-deployment/get-all-api-deployment/invalid", nil)
	context.Params = gin.Params{{Key: "assistantId", Value: "invalid"}}
	attachTestAuthentication(context, testProjectAuthentication(22, 33))

	deploymentApi.GetAllAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetAllAssistantPhoneDeploymentRest_InvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/assistant-deployment/get-all-phone-deployment/123?page=abc",
		nil,
	)
	context.Params = gin.Params{{Key: "assistantId", Value: "123"}}
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.GetAllAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, service.getAllCalled)
	assert.Contains(t, recorder.Body.String(), pkg_errors.GetAllAssistantPhoneDeploymentInvalidRequest.Error)
}
