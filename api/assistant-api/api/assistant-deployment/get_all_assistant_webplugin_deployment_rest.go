// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"encoding/json"
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

func (deploymentApi *AssistantDeploymentApi) GetAllAssistantWebpluginDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !hasUserProjectCapability(auth) {
		c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(c.Param("assistantId"))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}

	paginate := &assistant_api.Paginate{Page: 1, PageSize: 20}
	if c.Query("page") != "" {
		page, err := utils.StringToUint32(c.Query("page"))
		if err != nil || !validator.NonZero(page) {
			c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		paginate.Page = page
	}
	if c.Query("pageSize") != "" {
		pageSize, err := utils.StringToUint32(c.Query("pageSize"))
		if err != nil || !validator.NonZero(pageSize) {
			c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		paginate.PageSize = pageSize
	}

	criterias := []*assistant_api.Criteria{}
	if c.Query("criterias") != "" {
		requestCriterias := []openapi.Criteria{}
		if err := json.Unmarshal([]byte(c.Query("criterias")), &requestCriterias); err != nil {
			c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		for _, criteria := range requestCriterias {
			key := ""
			if validator.NonNil(criteria.Key) {
				key = *criteria.Key
			}
			value := ""
			if validator.NonNil(criteria.Value) {
				value = *criteria.Value
			}
			logic := ""
			if validator.NonNil(criteria.Logic) {
				logic = *criteria.Logic
			}
			criterias = append(criterias, &assistant_api.Criteria{Key: key, Value: value, Logic: logic})
		}
	}

	totalItems, deployments, err := deploymentApi.deploymentService.GetAllAssistantWebpluginDeployment(c, auth, assistantId, criterias, paginate)
	if err != nil {
		deploymentApi.logger.Errorf("unable to get all assistant webplugin deployments: %v", err)
		c.JSON(pkg_errors.GetAllAssistantWebpluginDeploymentGetDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentGetDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWebpluginDeploymentGetDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentGetDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWebpluginDeploymentGetDeployment.ErrorMessage),
			},
		})
		return
	}

	responseDeployments := []openapi.AssistantWebpluginDeployment{}
	for _, deployment := range deployments {
		if !validator.NonNil(deployment) {
			continue
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
		responseDeployments = append(responseDeployments, openapi.AssistantWebpluginDeployment{
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
		})
	}
	totalItem := uint32(totalItems)
	currentPage := paginate.GetPage()
	c.JSON(http.StatusOK, openapi.GetAllAssistantWebpluginDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    &responseDeployments,
		Paginated: &openapi.Paginated{
			TotalItem:   &totalItem,
			CurrentPage: &currentPage,
		},
	})
}
