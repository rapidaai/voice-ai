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
)

func (deploymentApi *AssistantDeploymentApi) GetAssistantApiDeploymentRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		c.JSON(pkg_errors.GetAssistantApiDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantApiDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantApiDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		c.JSON(pkg_errors.GetAssistantApiDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantApiDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantApiDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(c.Param("assistantId"))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}

	deployment, err := deploymentApi.deploymentService.GetAssistantApiDeployment(c.Request.Context(), iAuth, assistantId)
	if err != nil {
		deploymentApi.logger.Errorf("unable to get assistant api deployment: %v", err)
		c.JSON(pkg_errors.GetAssistantApiDeploymentGetDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantApiDeploymentGetDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantApiDeploymentGetDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentGetDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantApiDeploymentGetDeployment.ErrorMessage),
			},
		})
		return
	}
	if !validator.NonNil(deployment) {
		c.JSON(http.StatusOK, openapi.GetAssistantApiDeploymentResponse{
			Code:    utils.Ptr(int32(http.StatusOK)),
			Success: utils.Ptr(true),
			Data:    nil,
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
