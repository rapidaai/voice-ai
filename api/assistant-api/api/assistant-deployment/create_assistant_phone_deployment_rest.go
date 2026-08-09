// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
)

func (deploymentApi *AssistantDeploymentApi) CreateAssistantPhoneDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	var request openapi.CreateAssistantPhoneDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		deploymentApi.logger.Errorf("create assistant phone deployment invalid request: %v", err)
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(string(request.AssistantId))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(request.PhoneProviderName) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentMissingPhoneProvider.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.UnclearInputTimeout) && !validator.Between(*request.UnclearInputTimeout, 2, 10) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidUnclearTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeout) && !validator.Between(int(*request.IdealTimeout), 5, 120) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidIdealTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeoutBackoff) && !validator.Between(int(*request.IdealTimeoutBackoff), 0, 5) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidTimeoutBackoff.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidTimeoutBackoff.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidTimeoutBackoff.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidTimeoutBackoff.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.MaxSessionDuration) && !validator.Between(int(*request.MaxSessionDuration), 180, 600) {
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidSessionDuration.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidSessionDuration.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidSessionDuration.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidSessionDuration.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidSessionDuration.ErrorMessage),
			},
		})
		return
	}

	var inputAudio *assistant_api.DeploymentAudioProvider
	if validator.NonNil(request.InputAudio) {
		if !validator.NotBlank(request.InputAudio.AudioProvider) {
			c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.ErrorMessage),
				},
			})
			return
		}
		inputAudioOptions := []*assistant_api.Metadata{}
		if validator.NonNil(request.InputAudio.AudioOptions) {
			for _, audioOption := range *request.InputAudio.AudioOptions {
				key := ""
				if validator.NonNil(audioOption.Key) {
					key = *audioOption.Key
				}
				value := ""
				if validator.NonNil(audioOption.Value) {
					value = *audioOption.Value
				}
				inputAudioOptions = append(inputAudioOptions, &assistant_api.Metadata{Key: key, Value: value})
			}
		}
		inputAudioStatus := ""
		if validator.NonNil(request.InputAudio.Status) {
			inputAudioStatus = *request.InputAudio.Status
		}
		inputAudioType := ""
		if validator.NonNil(request.InputAudio.AudioType) {
			inputAudioType = *request.InputAudio.AudioType
		}
		inputAudio = &assistant_api.DeploymentAudioProvider{
			AudioProvider: request.InputAudio.AudioProvider,
			AudioOptions:  inputAudioOptions,
			Status:        inputAudioStatus,
			AudioType:     inputAudioType,
		}
	}

	var outputAudio *assistant_api.DeploymentAudioProvider
	if validator.NonNil(request.OutputAudio) {
		if !validator.NotBlank(request.OutputAudio.AudioProvider) {
			c.JSON(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentInvalidAudioProvider.ErrorMessage),
				},
			})
			return
		}
		outputAudioOptions := []*assistant_api.Metadata{}
		if validator.NonNil(request.OutputAudio.AudioOptions) {
			for _, audioOption := range *request.OutputAudio.AudioOptions {
				key := ""
				if validator.NonNil(audioOption.Key) {
					key = *audioOption.Key
				}
				value := ""
				if validator.NonNil(audioOption.Value) {
					value = *audioOption.Value
				}
				outputAudioOptions = append(outputAudioOptions, &assistant_api.Metadata{Key: key, Value: value})
			}
		}
		outputAudioStatus := ""
		if validator.NonNil(request.OutputAudio.Status) {
			outputAudioStatus = *request.OutputAudio.Status
		}
		outputAudioType := ""
		if validator.NonNil(request.OutputAudio.AudioType) {
			outputAudioType = *request.OutputAudio.AudioType
		}
		outputAudio = &assistant_api.DeploymentAudioProvider{
			AudioProvider: request.OutputAudio.AudioProvider,
			AudioOptions:  outputAudioOptions,
			Status:        outputAudioStatus,
			AudioType:     outputAudioType,
		}
	}

	phoneOptions := []*assistant_api.Metadata{}
	if validator.NonNil(request.PhoneOptions) {
		for _, phoneOption := range *request.PhoneOptions {
			key := ""
			if validator.NonNil(phoneOption.Key) {
				key = *phoneOption.Key
			}
			value := ""
			if validator.NonNil(phoneOption.Value) {
				value = *phoneOption.Value
			}
			phoneOptions = append(phoneOptions, &assistant_api.Metadata{Key: key, Value: value})
		}
	}

	deployment, err := deploymentApi.deploymentService.CreatePhoneDeployment(
		c,
		auth,
		assistantId,
		request.Greeting,
		request.Mistake,
		request.UnclearInputTimeout,
		request.UnclearInputMessage,
		request.GreetingInterruptible,
		request.IdealTimeout,
		request.IdealTimeoutBackoff,
		request.IdealTimeoutMessage,
		request.MaxSessionDuration,
		request.PhoneProviderName,
		inputAudio,
		outputAudio,
		phoneOptions,
	)
	if err != nil {
		deploymentApi.logger.Errorf("unable to create assistant phone deployment: %v", err)
		c.JSON(pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantPhoneDeploymentCreateDeployment.ErrorMessage),
			},
		})
		return
	}

	deploymentId := openapi.Uint64String(strconv.FormatUint(deployment.Id, 10))
	deploymentAssistantId := openapi.Uint64String(strconv.FormatUint(deployment.AssistantId, 10))
	deploymentStatus := deployment.Status.String()

	var responseInputAudio *openapi.DeploymentAudioProvider
	if validator.NonNil(deployment.InputAudio) {
		inputAudioId := openapi.Uint64String(strconv.FormatUint(deployment.InputAudio.Id, 10))
		inputAudioStatus := deployment.InputAudio.Status.String()
		inputAudioOptions := []openapi.Metadata{}
		for _, audioOption := range deployment.InputAudio.AudioOptions {
			if !validator.NonNil(audioOption) {
				continue
			}
			inputAudioOptions = append(inputAudioOptions, openapi.Metadata{
				Key:   utils.Ptr(audioOption.Key),
				Value: utils.Ptr(audioOption.Value),
			})
		}
		responseInputAudio = &openapi.DeploymentAudioProvider{
			Id:            &inputAudioId,
			AudioType:     &deployment.InputAudio.AudioType,
			AudioProvider: &deployment.InputAudio.AudioProvider,
			AudioOptions:  &inputAudioOptions,
			Status:        &inputAudioStatus,
		}
	}

	var responseOutputAudio *openapi.DeploymentAudioProvider
	if validator.NonNil(deployment.OutputAudio) {
		outputAudioId := openapi.Uint64String(strconv.FormatUint(deployment.OutputAudio.Id, 10))
		outputAudioStatus := deployment.OutputAudio.Status.String()
		outputAudioOptions := []openapi.Metadata{}
		for _, audioOption := range deployment.OutputAudio.AudioOptions {
			if !validator.NonNil(audioOption) {
				continue
			}
			outputAudioOptions = append(outputAudioOptions, openapi.Metadata{
				Key:   utils.Ptr(audioOption.Key),
				Value: utils.Ptr(audioOption.Value),
			})
		}
		responseOutputAudio = &openapi.DeploymentAudioProvider{
			Id:            &outputAudioId,
			AudioType:     &deployment.OutputAudio.AudioType,
			AudioProvider: &deployment.OutputAudio.AudioProvider,
			AudioOptions:  &outputAudioOptions,
			Status:        &outputAudioStatus,
		}
	}

	responsePhoneOptions := []openapi.Metadata{}
	for _, phoneOption := range deployment.TelephonyOption {
		if !validator.NonNil(phoneOption) {
			continue
		}
		responsePhoneOptions = append(responsePhoneOptions, openapi.Metadata{
			Key:   utils.Ptr(phoneOption.Key),
			Value: utils.Ptr(phoneOption.Value),
		})
	}
	phoneProviderName := deployment.TelephonyProvider

	c.JSON(http.StatusOK, openapi.GetAssistantPhoneDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data: &openapi.AssistantPhoneDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			UnclearInputTimeout:   deployment.UnclearInputTimeout,
			UnclearInputMessage:   deployment.UnclearInputMessage,
			InputAudio:            responseInputAudio,
			OutputAudio:           responseOutputAudio,
			PhoneProviderName:     &phoneProviderName,
			PhoneOptions:          &responsePhoneOptions,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		},
	})
}
