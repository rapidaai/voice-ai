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

func (deploymentApi *AssistantDeploymentApi) CreateAssistantApiDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	var request openapi.CreateAssistantApiDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		deploymentApi.logger.Errorf("create assistant api deployment invalid request: %v", err)
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(string(request.AssistantId))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.UnclearInputTimeout) && !validator.Between(*request.UnclearInputTimeout, 2, 10) {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidUnclearTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeout) && !validator.Between(int(*request.IdealTimeout), 5, 120) {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidIdealTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeoutBackoff) && !validator.Between(int(*request.IdealTimeoutBackoff), 0, 5) {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidTimeoutBackoff.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidTimeoutBackoff.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidTimeoutBackoff.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidTimeoutBackoff.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.MaxSessionDuration) && !validator.Between(int(*request.MaxSessionDuration), 180, 600) {
		c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidSessionDuration.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidSessionDuration.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidSessionDuration.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidSessionDuration.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidSessionDuration.ErrorMessage),
			},
		})
		return
	}

	var inputAudio *assistant_api.DeploymentAudioProvider
	if validator.NonNil(request.InputAudio) {
		if !validator.NotBlank(request.InputAudio.AudioProvider) {
			c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.ErrorMessage),
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
			c.JSON(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentInvalidAudioProvider.ErrorMessage),
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

	deployment, err := deploymentApi.deploymentService.CreateApiDeployment(
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
		inputAudio,
		outputAudio,
	)
	if err != nil {
		deploymentApi.logger.Errorf("unable to create assistant api deployment: %v", err)
		c.JSON(pkg_errors.CreateAssistantApiDeploymentCreateDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantApiDeploymentCreateDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantApiDeploymentCreateDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentCreateDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantApiDeploymentCreateDeployment.ErrorMessage),
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

	c.JSON(http.StatusOK, openapi.GetAssistantApiDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data: &openapi.AssistantApiDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			UnclearInputTimeout:   deployment.UnclearInputTimeout,
			UnclearInputMessage:   deployment.UnclearInputMessage,
			InputAudio:            responseInputAudio,
			OutputAudio:           responseOutputAudio,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		},
	})
}
