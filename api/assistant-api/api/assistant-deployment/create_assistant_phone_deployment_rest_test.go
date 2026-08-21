package assistant_deployment_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAssistantPhoneDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)
	requestBody := []byte(`{
		"assistantId": "123",
		"greeting": "Hello",
		"greetingInterruptible": false,
		"idealTimeout": 30,
		"maxSessionDuration": 600,
		"phoneProviderName": "twilio",
		"phoneOptions": [{"key": "phone", "value": "+15551234567"}],
		"inputAudio": {
			"audioProvider": "twilio",
			"audioType": "input",
			"audioOptions": [{"key": "codec", "value": "mulaw"}]
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader(requestBody),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	require.NotNil(t, service.greetingInterruptible)
	assert.False(t, *service.greetingInterruptible)
	assert.Equal(t, "twilio", service.phoneProviderName)
	require.Len(t, service.phoneOptions, 1)
	assert.Equal(t, "phone", service.phoneOptions[0].GetKey())
	require.NotNil(t, service.inputAudio)
	assert.Equal(t, "twilio", service.inputAudio.GetAudioProvider())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	assert.Equal(t, false, data["greetingInterruptible"])
	assert.Equal(t, "twilio", data["phoneProviderName"])
}

func TestCreateAssistantPhoneDeploymentRest_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader([]byte(`{}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.Error)
}

func TestCreateAssistantPhoneDeploymentRest_MissingAuthScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","phoneProviderName":"twilio"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, testProjectAuthentication(22, 33))

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.Error)
}

func TestCreateAssistantPhoneDeploymentRest_MissingPhoneProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","phoneProviderName":""}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.Error)
}

func TestCreateAssistantPhoneDeploymentRest_InvalidAudioProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","phoneProviderName":"twilio","inputAudio":{"audioProvider":""}}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.Error)
}

func TestCreateAssistantPhoneDeploymentRest_CreateDeploymentErrorDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{
		createErr: errors.New("database password leaked"),
	}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-phone-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","phoneProviderName":"twilio"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantPhoneDeploymentRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.Error)
	assert.NotContains(t, recorder.Body.String(), "database password leaked")
}
