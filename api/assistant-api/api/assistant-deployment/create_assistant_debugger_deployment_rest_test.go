package assistant_deployment_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type createDebuggerDeploymentRestServiceStub struct {
	createCalled          bool
	createErr             error
	getCalled             bool
	getErr                error
	getNil                bool
	getAllCalled          bool
	getAllErr             error
	page                  uint32
	pageSize              uint32
	criterias             []*protos.Criteria
	assistantId           uint64
	greeting              *string
	greetingInterruptible *bool
	inputAudio            *protos.DeploymentAudioProvider
	outputAudio           *protos.DeploymentAudioProvider
	maxSessionDuration    *uint64
	suggestion            []string
	phoneProviderName     string
	phoneOptions          []*protos.Metadata
	whatsappProvider      string
	whatsappOptions       []*protos.Metadata
}

func (s *createDebuggerDeploymentRestServiceStub) CreateWhatsappDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	greeting *string,
	mistake *string,
	unclearInputTimeout *float64,
	unclearInputMessage *string,
	greetingInterruptible *bool,
	idealTimeout *uint64,
	idealTimeoutBackoff *uint64,
	idealTimeoutMessage *string,
	maxSessionDuration *uint64,
	whatsappProvider string,
	whatsappOptions []*protos.Metadata,
) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	_, _, _, _, _ = mistake, unclearInputTimeout, unclearInputMessage, idealTimeoutBackoff, idealTimeoutMessage
	s.createCalled = true
	s.assistantId = assistantId
	s.greeting = greeting
	s.greetingInterruptible = greetingInterruptible
	s.maxSessionDuration = maxSessionDuration
	s.whatsappProvider = whatsappProvider
	s.whatsappOptions = whatsappOptions
	if s.createErr != nil {
		return nil, s.createErr
	}

	whatsappDeployment := &internal_assistant_entity.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           idealTimeout,
			MaxSessionDuration:    maxSessionDuration,
		},
		AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
			WhatsappProvider: whatsappProvider,
		},
	}
	for _, option := range whatsappOptions {
		optionEntity := &internal_assistant_entity.AssistantDeploymentWhatsappOption{}
		optionEntity.Key = option.GetKey()
		optionEntity.Value = option.GetValue()
		whatsappDeployment.WhatsappOptions = append(whatsappDeployment.WhatsappOptions, optionEntity)
	}
	return whatsappDeployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) CreatePhoneDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	greeting *string,
	mistake *string,
	unclearInputTimeout *float64,
	unclearInputMessage *string,
	greetingInterruptible *bool,
	idealTimeout *uint64,
	idealTimeoutBackoff *uint64,
	idealTimeoutMessage *string,
	maxSessionDuration *uint64,
	phoneProviderName string,
	inputAudio *protos.DeploymentAudioProvider,
	outputAudio *protos.DeploymentAudioProvider,
	phoneOptions []*protos.Metadata,
) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	_, _, _, _, _, _, _ = mistake, unclearInputTimeout, unclearInputMessage, idealTimeout, idealTimeoutBackoff, idealTimeoutMessage, outputAudio
	s.createCalled = true
	s.assistantId = assistantId
	s.greeting = greeting
	s.greetingInterruptible = greetingInterruptible
	s.inputAudio = inputAudio
	s.outputAudio = outputAudio
	s.maxSessionDuration = maxSessionDuration
	s.phoneProviderName = phoneProviderName
	s.phoneOptions = phoneOptions
	if s.createErr != nil {
		return nil, s.createErr
	}

	inputAudioEntity := deploymentAudioProviderEntityFromProto(inputAudio)
	phoneDeployment := &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			MaxSessionDuration:    maxSessionDuration,
		},
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: phoneProviderName,
		},
		InputAudio: inputAudioEntity,
	}
	for _, option := range phoneOptions {
		optionEntity := &internal_assistant_entity.AssistantDeploymentTelephonyOption{}
		optionEntity.Key = option.GetKey()
		optionEntity.Value = option.GetValue()
		phoneDeployment.TelephonyOption = append(phoneDeployment.TelephonyOption, optionEntity)
	}
	return phoneDeployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) CreateApiDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	greeting *string,
	mistake *string,
	unclearInputTimeout *float64,
	unclearInputMessage *string,
	greetingInterruptible *bool,
	idealTimeout *uint64,
	idealTimeoutBackoff *uint64,
	idealTimeoutMessage *string,
	maxSessionDuration *uint64,
	inputAudio *protos.DeploymentAudioProvider,
	outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantApiDeployment, error) {
	_, _, _, _, _ = mistake, unclearInputTimeout, unclearInputMessage, idealTimeoutBackoff, idealTimeoutMessage
	s.createCalled = true
	s.assistantId = assistantId
	s.greeting = greeting
	s.greetingInterruptible = greetingInterruptible
	s.inputAudio = inputAudio
	s.outputAudio = outputAudio
	s.maxSessionDuration = maxSessionDuration
	if s.createErr != nil {
		return nil, s.createErr
	}

	inputAudioEntity := deploymentAudioProviderEntityFromProto(inputAudio)
	outputAudioEntity := deploymentAudioProviderEntityFromProto(outputAudio)
	return &internal_assistant_entity.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           idealTimeout,
			MaxSessionDuration:    maxSessionDuration,
		},
		InputAudio:  inputAudioEntity,
		OutputAudio: outputAudioEntity,
	}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) CreateDebuggerDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	greeting, _ *string,
	unclearInputTimeout *float64,
	unclearInputMessage *string,
	greetingInterruptible *bool,
	idealTimeout *uint64,
	_ *uint64,
	_ *string,
	maxSessionDuration *uint64,
	inputAudio, _ *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	s.createCalled = true
	s.assistantId = assistantId
	s.greeting = greeting
	s.greetingInterruptible = greetingInterruptible
	s.inputAudio = inputAudio
	s.outputAudio = nil
	s.maxSessionDuration = maxSessionDuration
	if s.createErr != nil {
		return nil, s.createErr
	}

	inputAudioEntity := deploymentAudioProviderEntityFromProto(inputAudio)
	return &internal_assistant_entity.AssistantDebuggerDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           idealTimeout,
			MaxSessionDuration:    maxSessionDuration,
		},
		InputAudio: inputAudioEntity,
	}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) CreateWebPluginDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	greeting *string,
	mistake *string,
	unclearInputTimeout *float64,
	unclearInputMessage *string,
	greetingInterruptible *bool,
	idealTimeout *uint64,
	idealTimeoutBackoff *uint64,
	idealTimeoutMessage *string,
	maxSessionDuration *uint64,
	suggestion []string,
	inputAudio *protos.DeploymentAudioProvider,
	outputAudio *protos.DeploymentAudioProvider,
) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	_, _, _, _, _ = mistake, unclearInputTimeout, unclearInputMessage, idealTimeoutBackoff, idealTimeoutMessage
	s.createCalled = true
	s.assistantId = assistantId
	s.greeting = greeting
	s.greetingInterruptible = greetingInterruptible
	s.inputAudio = inputAudio
	s.outputAudio = outputAudio
	s.maxSessionDuration = maxSessionDuration
	s.suggestion = suggestion
	if s.createErr != nil {
		return nil, s.createErr
	}

	inputAudioEntity := deploymentAudioProviderEntityFromProto(inputAudio)
	outputAudioEntity := deploymentAudioProviderEntityFromProto(outputAudio)
	return &internal_assistant_entity.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting:              greeting,
			GreetingInterruptible: greetingInterruptible,
			UnclearInputTimeout:   unclearInputTimeout,
			UnclearInputMessage:   unclearInputMessage,
			IdleTimeout:           idealTimeout,
			MaxSessionDuration:    maxSessionDuration,
		},
		Suggestion:  gorm_types.StringArray(suggestion),
		InputAudio:  inputAudioEntity,
		OutputAudio: outputAudioEntity,
	}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAssistantApiDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
) (*internal_assistant_entity.AssistantApiDeployment, error) {
	s.getCalled = true
	s.assistantId = assistantId
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getNil {
		return nil, nil
	}
	greeting := "Hello"
	inputAudio := &internal_assistant_entity.AssistantDeploymentAudio{
		AudioType:     "input",
		AudioProvider: "twilio",
	}
	inputAudio.Id = 321
	inputAudio.Status = type_enums.RECORD_ACTIVE
	inputAudioOption := &internal_assistant_entity.AssistantDeploymentAudioOption{}
	inputAudioOption.Key = "codec"
	inputAudioOption.Value = "mulaw"
	inputAudio.AudioOptions = append(inputAudio.AudioOptions, inputAudioOption)
	deployment := &internal_assistant_entity.AssistantApiDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting: &greeting,
		},
		InputAudio: inputAudio,
	}
	deployment.Id = 654
	deployment.Status = type_enums.RECORD_ACTIVE
	return deployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAssistantDebuggerDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	s.getCalled = true
	s.assistantId = assistantId
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getNil {
		return nil, nil
	}
	greeting := "Hello"
	deployment := &internal_assistant_entity.AssistantDebuggerDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting: &greeting,
		},
	}
	deployment.Id = 655
	deployment.Status = type_enums.RECORD_ACTIVE
	return deployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAssistantPhoneDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	s.getCalled = true
	s.assistantId = assistantId
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getNil {
		return nil, nil
	}
	greeting := "Hello"
	deployment := &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting: &greeting,
		},
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyProvider: "twilio",
		},
	}
	deployment.Id = 656
	deployment.Status = type_enums.RECORD_ACTIVE
	phoneOption := &internal_assistant_entity.AssistantDeploymentTelephonyOption{}
	phoneOption.Key = "phone"
	phoneOption.Value = "+15551234567"
	deployment.TelephonyOption = append(deployment.TelephonyOption, phoneOption)
	return deployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAssistantWebpluginDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	s.getCalled = true
	s.assistantId = assistantId
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getNil {
		return nil, nil
	}
	greeting := "Hello"
	deployment := &internal_assistant_entity.AssistantWebPluginDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting: &greeting,
		},
		Suggestion: gorm_types.StringArray{"Book demo", "Talk to support"},
	}
	deployment.Id = 657
	deployment.Status = type_enums.RECORD_ACTIVE
	return deployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAssistantWhatsappDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	s.getCalled = true
	s.assistantId = assistantId
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getNil {
		return nil, nil
	}
	greeting := "Hello"
	deployment := &internal_assistant_entity.AssistantWhatsappDeployment{
		AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
			AssistantDeployment: internal_assistant_entity.AssistantDeployment{
				AssistantId: assistantId,
			},
			Greeting: &greeting,
		},
		AssistantDeploymentWhatsapp: internal_assistant_entity.AssistantDeploymentWhatsapp{
			WhatsappProvider: "gupshup",
		},
	}
	deployment.Id = 658
	deployment.Status = type_enums.RECORD_ACTIVE
	whatsappOption := &internal_assistant_entity.AssistantDeploymentWhatsappOption{}
	whatsappOption.Key = "template"
	whatsappOption.Value = "welcome"
	deployment.WhatsappOptions = append(deployment.WhatsappOptions, whatsappOption)
	return deployment, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAllAssistantApiDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantApiDeployment, error) {
	s.getAllCalled = true
	s.assistantId = assistantId
	s.criterias = criterias
	s.page = paginate.GetPage()
	s.pageSize = paginate.GetPageSize()
	if s.getAllErr != nil {
		return 0, nil, s.getAllErr
	}
	deployment, err := s.GetAssistantApiDeployment(context.Background(), nil, assistantId)
	if err != nil {
		return 0, nil, err
	}
	return 1, []*internal_assistant_entity.AssistantApiDeployment{deployment}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAllAssistantDebuggerDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	s.getAllCalled = true
	s.assistantId = assistantId
	s.criterias = criterias
	s.page = paginate.GetPage()
	s.pageSize = paginate.GetPageSize()
	if s.getAllErr != nil {
		return 0, nil, s.getAllErr
	}
	deployment, err := s.GetAssistantDebuggerDeployment(context.Background(), nil, assistantId)
	if err != nil {
		return 0, nil, err
	}
	return 1, []*internal_assistant_entity.AssistantDebuggerDeployment{deployment}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAllAssistantPhoneDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantPhoneDeployment, error) {
	s.getAllCalled = true
	s.assistantId = assistantId
	s.criterias = criterias
	s.page = paginate.GetPage()
	s.pageSize = paginate.GetPageSize()
	if s.getAllErr != nil {
		return 0, nil, s.getAllErr
	}
	deployment, err := s.GetAssistantPhoneDeployment(context.Background(), nil, assistantId)
	if err != nil {
		return 0, nil, err
	}
	return 1, []*internal_assistant_entity.AssistantPhoneDeployment{deployment}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAllAssistantWebpluginDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	s.getAllCalled = true
	s.assistantId = assistantId
	s.criterias = criterias
	s.page = paginate.GetPage()
	s.pageSize = paginate.GetPageSize()
	if s.getAllErr != nil {
		return 0, nil, s.getAllErr
	}
	deployment, err := s.GetAssistantWebpluginDeployment(context.Background(), nil, assistantId)
	if err != nil {
		return 0, nil, err
	}
	return 1, []*internal_assistant_entity.AssistantWebPluginDeployment{deployment}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) GetAllAssistantWhatsappDeployment(
	_ context.Context,
	_ *types.Authentication,
	assistantId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	s.getAllCalled = true
	s.assistantId = assistantId
	s.criterias = criterias
	s.page = paginate.GetPage()
	s.pageSize = paginate.GetPageSize()
	if s.getAllErr != nil {
		return 0, nil, s.getAllErr
	}
	deployment, err := s.GetAssistantWhatsappDeployment(context.Background(), nil, assistantId)
	if err != nil {
		return 0, nil, err
	}
	return 1, []*internal_assistant_entity.AssistantWhatsappDeployment{deployment}, nil
}

func (s *createDebuggerDeploymentRestServiceStub) DisableAssistantApiDeployment(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.AssistantApiDeployment, error) {
	return nil, errors.New("not implemented")
}

func (s *createDebuggerDeploymentRestServiceStub) DisableAssistantDebuggerDeployment(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.AssistantDebuggerDeployment, error) {
	return nil, errors.New("not implemented")
}

func (s *createDebuggerDeploymentRestServiceStub) DisableAssistantPhoneDeployment(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.AssistantPhoneDeployment, error) {
	return nil, errors.New("not implemented")
}

func (s *createDebuggerDeploymentRestServiceStub) DisableAssistantWebpluginDeployment(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.AssistantWebPluginDeployment, error) {
	return nil, errors.New("not implemented")
}

func (s *createDebuggerDeploymentRestServiceStub) DisableAssistantWhatsappDeployment(context.Context, *types.Authentication, uint64) (*internal_assistant_entity.AssistantWhatsappDeployment, error) {
	return nil, errors.New("not implemented")
}

func TestCreateAssistantDebuggerDeploymentRest_HappyPath(t *testing.T) {
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
		}
	}`)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader(requestBody),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
	assert.Equal(t, uint64(123), service.assistantId)
	require.NotNil(t, service.greeting)
	assert.Equal(t, "Hello", *service.greeting)
	require.NotNil(t, service.greetingInterruptible)
	assert.False(t, *service.greetingInterruptible)
	require.NotNil(t, service.maxSessionDuration)
	assert.Equal(t, uint64(600), *service.maxSessionDuration)
	require.NotNil(t, service.inputAudio)
	assert.Equal(t, "twilio", service.inputAudio.GetAudioProvider())
	require.Len(t, service.inputAudio.GetAudioOptions(), 1)
	assert.Equal(t, "codec", service.inputAudio.GetAudioOptions()[0].GetKey())

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "123", data["assistantId"])
	assert.Equal(t, false, data["greetingInterruptible"])
}

func TestCreateAssistantDebuggerDeploymentRest_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentUnauthenticated.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_ProjectScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, testProjectAuthentication(22, 33))

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.createCalled)
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidAssistantID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"abc"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidAssistantID.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidAudioProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","inputAudio":{"audioProvider":""}}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidAudioProvider.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidIdealTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","idealTimeout":4}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidUnclearInputTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(
			http.MethodPost,
			"/v1/assistant-deployment/create-debugger-deployment",
			bytes.NewReader([]byte(fmt.Sprintf(`{"assistantId":"123","unclearInputTimeout":%f}`, unclearInputTimeout))),
		)
		context.Request.Header.Set("Content-Type", "application/json")
		attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

		deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidUnclearTimeout.Error)
	}
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidIdealTimeoutBackoff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","idealTimeoutBackoff":6}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_InvalidMaxSessionDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, &createDebuggerDeploymentRestServiceStub{})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123","maxSessionDuration":179}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.Error)
}

func TestCreateAssistantDebuggerDeploymentRest_CreateDeploymentErrorDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &createDebuggerDeploymentRestServiceStub{
		createErr: errors.New("database password leaked"),
	}
	deploymentApi := newCreateDebuggerDeploymentRestApi(t, service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/assistant-deployment/create-debugger-deployment",
		bytes.NewReader([]byte(`{"assistantId":"123"}`)),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	attachTestAuthentication(context, createDebuggerDeploymentRestAuth())

	deploymentApi.CreateAssistantDebuggerDeploymentRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateAssistantDebuggerDeploymentCreateDeployment.Error)
	assert.NotContains(t, recorder.Body.String(), "database password leaked")
}

func TestCreateAssistantDebuggerDeploymentGRPC_InvalidIdealTimeout(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createDebuggerDeploymentGRPCRequest()
	request.GetDebugger().IdealTimeout = 4

	response, err := deploymentApi.CreateAssistantDebuggerDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidIdealTimeout.Code), response.Error.ErrorCode)
}

func TestCreateAssistantApiDeploymentGRPC_InvalidIdealTimeout(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createApiDeploymentGRPCRequest()
	request.GetApi().IdealTimeout = 4

	response, err := deploymentApi.CreateAssistantApiDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.Code), response.Error.ErrorCode)
}

func TestCreateAssistantPhoneDeploymentGRPC_InvalidIdealTimeout(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createPhoneDeploymentGRPCRequest()
	request.GetPhone().IdealTimeout = 4

	response, err := deploymentApi.CreateAssistantPhoneDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.Code), response.Error.ErrorCode)
}

func TestCreateAssistantWebpluginDeploymentGRPC_InvalidIdealTimeout(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createWebpluginDeploymentGRPCRequest()
	request.GetPlugin().IdealTimeout = 4

	response, err := deploymentApi.CreateAssistantWebpluginDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.Code), response.Error.ErrorCode)
}

func TestCreateAssistantWhatsappDeploymentGRPC_InvalidIdealTimeout(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createWhatsappDeploymentGRPCRequest()
	request.GetWhatsapp().IdealTimeout = 4

	response, err := deploymentApi.CreateAssistantWhatsappDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.Code), response.Error.ErrorCode)
}

func TestCreateAssistantDebuggerDeploymentGRPC_InvalidUnclearInputTimeout(t *testing.T) {
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		service := &createDebuggerDeploymentRestServiceStub{}
		deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
		request := createDebuggerDeploymentGRPCRequest()
		request.GetDebugger().UnclearInputTimeout = &unclearInputTimeout

		response, err := deploymentApi.CreateAssistantDebuggerDeployment(createDebuggerDeploymentGRPCContext(), request)

		require.Error(t, err)
		require.NotNil(t, response)
		assert.False(t, service.createCalled)
		assert.Equal(t, pkg_errors.CreateAssistantDebuggerDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(), response.Code)
		require.NotNil(t, response.Error)
		assert.Equal(t, uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidUnclearTimeout.Code), response.Error.ErrorCode)
	}
}

func TestCreateAssistantApiDeploymentGRPC_InvalidUnclearInputTimeout(t *testing.T) {
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		service := &createDebuggerDeploymentRestServiceStub{}
		deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
		request := createApiDeploymentGRPCRequest()
		request.GetApi().UnclearInputTimeout = &unclearInputTimeout

		response, err := deploymentApi.CreateAssistantApiDeployment(createDebuggerDeploymentGRPCContext(), request)

		require.Error(t, err)
		require.NotNil(t, response)
		assert.False(t, service.createCalled)
		assert.Equal(t, pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(), response.Code)
		require.NotNil(t, response.Error)
		assert.Equal(t, uint64(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.Code), response.Error.ErrorCode)
	}
}

func TestCreateAssistantPhoneDeploymentGRPC_InvalidUnclearInputTimeout(t *testing.T) {
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		service := &createDebuggerDeploymentRestServiceStub{}
		deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
		request := createPhoneDeploymentGRPCRequest()
		request.GetPhone().UnclearInputTimeout = &unclearInputTimeout

		response, err := deploymentApi.CreateAssistantPhoneDeployment(createDebuggerDeploymentGRPCContext(), request)

		require.Error(t, err)
		require.NotNil(t, response)
		assert.False(t, service.createCalled)
		assert.Equal(t, pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(), response.Code)
		require.NotNil(t, response.Error)
		assert.Equal(t, uint64(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.Code), response.Error.ErrorCode)
	}
}

func TestCreateAssistantWebpluginDeploymentGRPC_InvalidUnclearInputTimeout(t *testing.T) {
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		service := &createDebuggerDeploymentRestServiceStub{}
		deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
		request := createWebpluginDeploymentGRPCRequest()
		request.GetPlugin().UnclearInputTimeout = &unclearInputTimeout

		response, err := deploymentApi.CreateAssistantWebpluginDeployment(createDebuggerDeploymentGRPCContext(), request)

		require.Error(t, err)
		require.NotNil(t, response)
		assert.False(t, service.createCalled)
		assert.Equal(t, pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(), response.Code)
		require.NotNil(t, response.Error)
		assert.Equal(t, uint64(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.Code), response.Error.ErrorCode)
	}
}

func TestCreateAssistantWhatsappDeploymentGRPC_InvalidUnclearInputTimeout(t *testing.T) {
	for _, unclearInputTimeout := range []float64{1.9, 10.1} {
		service := &createDebuggerDeploymentRestServiceStub{}
		deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
		request := createWhatsappDeploymentGRPCRequest()
		request.GetWhatsapp().UnclearInputTimeout = &unclearInputTimeout

		response, err := deploymentApi.CreateAssistantWhatsappDeployment(createDebuggerDeploymentGRPCContext(), request)

		require.Error(t, err)
		require.NotNil(t, response)
		assert.False(t, service.createCalled)
		assert.Equal(t, pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32(), response.Code)
		require.NotNil(t, response.Error)
		assert.Equal(t, uint64(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.Code), response.Error.ErrorCode)
	}
}

func TestCreateAssistantDebuggerDeploymentGRPC_InvalidIdealTimeoutBackoff(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createDebuggerDeploymentGRPCRequest()
	request.GetDebugger().IdealTimeoutBackoff = 6

	response, err := deploymentApi.CreateAssistantDebuggerDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidTimeoutBackoff.Code), response.Error.ErrorCode)
}

func TestCreateAssistantDebuggerDeploymentGRPC_InvalidMaxSessionDuration(t *testing.T) {
	service := &createDebuggerDeploymentRestServiceStub{}
	deploymentApi := newCreateDebuggerDeploymentGRPCApi(t, service)
	request := createDebuggerDeploymentGRPCRequest()
	request.GetDebugger().MaxSessionDuration = 179

	response, err := deploymentApi.CreateAssistantDebuggerDeployment(createDebuggerDeploymentGRPCContext(), request)

	require.Error(t, err)
	require.NotNil(t, response)
	assert.False(t, service.createCalled)
	assert.Equal(t, pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.HTTPStatusCodeInt32(), response.Code)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantDebuggerDeploymentInvalidSessionDuration.Code), response.Error.ErrorCode)
}

func newCreateDebuggerDeploymentRestApi(
	t *testing.T,
	service internal_services.AssistantDeploymentService,
) *AssistantDeploymentApi {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return &AssistantDeploymentApi{
		logger:            logger,
		deploymentService: service,
	}
}

func newCreateDebuggerDeploymentGRPCApi(
	t *testing.T,
	service internal_services.AssistantDeploymentService,
) *assistantDeploymentGrpcApi {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return &assistantDeploymentGrpcApi{
		AssistantDeploymentApi: AssistantDeploymentApi{
			logger:            logger,
			deploymentService: service,
		},
	}
}

func createDebuggerDeploymentRestAuth() *types.Authentication {
	return testUserAuthentication(11, 22, 33)
}

func createDebuggerDeploymentGRPCContext() context.Context {
	return context.WithValue(context.Background(), types.CTX_, createDebuggerDeploymentRestAuth())
}

func createDebuggerDeploymentGRPCRequest() *protos.CreateAssistantDeploymentRequest {
	return &protos.CreateAssistantDeploymentRequest{
		Deployment: &protos.CreateAssistantDeploymentRequest_Debugger{
			Debugger: &protos.AssistantDebuggerDeployment{
				AssistantId:         123,
				IdealTimeout:        30,
				IdealTimeoutBackoff: 1,
				MaxSessionDuration:  600,
			},
		},
	}
}

func createApiDeploymentGRPCRequest() *protos.CreateAssistantDeploymentRequest {
	return &protos.CreateAssistantDeploymentRequest{
		Deployment: &protos.CreateAssistantDeploymentRequest_Api{
			Api: &protos.AssistantApiDeployment{
				AssistantId:         123,
				IdealTimeout:        30,
				IdealTimeoutBackoff: 1,
				MaxSessionDuration:  600,
			},
		},
	}
}

func createPhoneDeploymentGRPCRequest() *protos.CreateAssistantDeploymentRequest {
	return &protos.CreateAssistantDeploymentRequest{
		Deployment: &protos.CreateAssistantDeploymentRequest_Phone{
			Phone: &protos.AssistantPhoneDeployment{
				AssistantId:         123,
				IdealTimeout:        30,
				IdealTimeoutBackoff: 1,
				MaxSessionDuration:  600,
				PhoneProviderName:   "twilio",
			},
		},
	}
}

func createWebpluginDeploymentGRPCRequest() *protos.CreateAssistantDeploymentRequest {
	return &protos.CreateAssistantDeploymentRequest{
		Deployment: &protos.CreateAssistantDeploymentRequest_Plugin{
			Plugin: &protos.AssistantWebpluginDeployment{
				AssistantId:         123,
				IdealTimeout:        30,
				IdealTimeoutBackoff: 1,
				MaxSessionDuration:  600,
			},
		},
	}
}

func createWhatsappDeploymentGRPCRequest() *protos.CreateAssistantDeploymentRequest {
	return &protos.CreateAssistantDeploymentRequest{
		Deployment: &protos.CreateAssistantDeploymentRequest_Whatsapp{
			Whatsapp: &protos.AssistantWhatsappDeployment{
				AssistantId:          123,
				IdealTimeout:         30,
				IdealTimeoutBackoff:  1,
				MaxSessionDuration:   600,
				WhatsappProviderName: "twilio",
			},
		},
	}
}

func deploymentAudioProviderEntityFromProto(
	audio *protos.DeploymentAudioProvider,
) *internal_assistant_entity.AssistantDeploymentAudio {
	if audio == nil {
		return nil
	}

	entity := &internal_assistant_entity.AssistantDeploymentAudio{
		AudioProvider: audio.GetAudioProvider(),
		AudioType:     audio.GetAudioType(),
	}
	entity.Status = type_enums.RECORD_ACTIVE
	for _, option := range audio.GetAudioOptions() {
		optionEntity := &internal_assistant_entity.AssistantDeploymentAudioOption{}
		optionEntity.Key = option.GetKey()
		optionEntity.Value = option.GetValue()
		entity.AudioOptions = append(entity.AudioOptions, optionEntity)
	}
	return entity
}
