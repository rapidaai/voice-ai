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

func TestCreateAssistantApiDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)
	requestBody := []byte(`{
		"assistantId": "123",
		"greeting": "Hello",
		"greetingInterruptible": false,
		"idealTimeout": 30,
		"maxSessionDuration": 600,
		"inputAudio": {
			"audioProvider": "twilio",
			"audioType": "input",
			"audioOptions": [{"key": "codec", "value": "mulaw"}]
		},
		"outputAudio": {
			"audioProvider": "openai",
			"audioType": "output"
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-api-deployment",
		bytes.NewReader(requestBody),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	require.NotNil(t, service.greetingInterruptible)
	assert.False(t, *service.greetingInterruptible)
	require.NotNil(t, service.inputAudio)
	assert.Equal(t, "twilio", service.inputAudio.GetAudioProvider())
	require.NotNil(t, service.outputAudio)
	assert.Equal(t, "openai", service.outputAudio.GetAudioProvider())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	assert.Equal(t, false, data["greetingInterruptible"])
}

func TestCreateAssistantApiDeploymentRest_InvalidAudioProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-api-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","outputAudio":{"audioProvider":""}}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantApiDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.Error)
}

func TestCreateAssistantWebpluginDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)
	requestBody := []byte(`{
		"assistantId": "123",
		"greeting": "Hello",
		"greetingInterruptible": false,
		"idealTimeout": 30,
		"maxSessionDuration": 600,
		"suggestion": ["Book demo", "Talk to support"],
		"inputAudio": {
			"audioProvider": "browser",
			"audioType": "input"
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-webplugin-deployment",
		bytes.NewReader(requestBody),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantWebpluginDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	require.NotNil(t, service.greetingInterruptible)
	assert.False(t, *service.greetingInterruptible)
	assert.Equal(t, []string{"Book demo", "Talk to support"}, service.suggestion)
	require.NotNil(t, service.inputAudio)
	assert.Equal(t, "browser", service.inputAudio.GetAudioProvider())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	assert.Equal(t, false, data["greetingInterruptible"])
	assert.Equal(t, []interface{}{"Book demo", "Talk to support"}, data["suggestion"])
}

func TestCreateAssistantWebpluginDeploymentRest_CreateDeploymentErrorDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{
		createErr: errors.New("database password leaked"),
	}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-webplugin-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantWebpluginDeploymentRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.Error)
	assert.NotContains(t, recorder.Body.String(), "database password leaked")
}

func TestCreateAssistantWhatsappDeploymentRest_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)
	requestBody := []byte(`{
		"assistantId": "123",
		"greeting": "Hello",
		"greetingInterruptible": false,
		"idealTimeout": 30,
		"maxSessionDuration": 600,
		"whatsappProviderName": "gupshup",
		"whatsappOptions": [{"key": "template", "value": "welcome"}]
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-whatsapp-deployment",
		bytes.NewReader(requestBody),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantWhatsappDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	require.NotNil(t, service.greetingInterruptible)
	assert.False(t, *service.greetingInterruptible)
	assert.Equal(t, "gupshup", service.whatsappProvider)
	require.Len(t, service.whatsappOptions, 1)
	assert.Equal(t, "template", service.whatsappOptions[0].GetKey())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	assert.Equal(t, false, data["greetingInterruptible"])
	assert.Equal(t, "gupshup", data["whatsappProviderName"])
}

func TestCreateAssistantWhatsappDeploymentRest_MissingWhatsappProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-whatsapp-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","whatsappProviderName":""}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantWhatsappDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.Error)
}
