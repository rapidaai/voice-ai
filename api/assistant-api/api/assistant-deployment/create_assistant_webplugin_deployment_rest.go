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

func (deploymentApi *AssistantDeploymentApi) CreateAssistantWebpluginDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	var request openapi.CreateAssistantWebpluginDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		deploymentApi.logger.Errorf("create assistant webplugin deployment invalid request: %v", err)
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(string(request.AssistantId))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.UnclearInputTimeout) && !validator.Between(*request.UnclearInputTimeout, 0.5, 5) {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidUnclearTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeout) && !validator.Between(int(*request.IdealTimeout), 5, 120) {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidIdealTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeoutBackoff) && !validator.Between(int(*request.IdealTimeoutBackoff), 0, 5) {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidTimeoutBackoff.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.MaxSessionDuration) && !validator.Between(int(*request.MaxSessionDuration), 180, 600) {
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidSessionDuration.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidSessionDuration.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidSessionDuration.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidSessionDuration.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidSessionDuration.ErrorMessage),
			},
		})
		return
	}

	var inputAudio *assistant_api.DeploymentAudioProvider
	if validator.NonNil(request.InputAudio) {
		if !validator.NotBlank(request.InputAudio.AudioProvider) {
			c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.ErrorMessage),
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
			c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentInvalidAudioProvider.ErrorMessage),
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

	suggestions := []string{}
	if validator.NonNil(request.Suggestion) {
		suggestions = *request.Suggestion
	}
	deployment, err := deploymentApi.deploymentService.CreateWebPluginDeployment(
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
		suggestions,
		inputAudio,
		outputAudio,
	)
	if err != nil {
		deploymentApi.logger.Errorf("unable to create assistant webplugin deployment: %v", err)
		c.JSON(pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWebpluginDeploymentCreateDeployment.ErrorMessage),
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
	responseSuggestions := []string(deployment.Suggestion)

	c.JSON(http.StatusOK, openapi.GetAssistantWebpluginDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data: &openapi.AssistantWebpluginDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			UnclearInputTimeout:   deployment.UnclearInputTimeout,
			UnclearInputMessage:   deployment.UnclearInputMessage,
			InputAudio:            responseInputAudio,
			OutputAudio:           responseOutputAudio,
			Suggestion:            &responseSuggestions,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		},
	})
}
