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

func (deploymentApi *AssistantDeploymentApi) GetAssistantWhatsappDeploymentRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		c.JSON(pkg_errors.GetAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantWhatsappDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		c.JSON(pkg_errors.GetAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantWhatsappDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(c.Param("assistantId"))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.GetAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantWhatsappDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}

	deployment, err := deploymentApi.deploymentService.GetAssistantWhatsappDeployment(c.Request.Context(), iAuth, assistantId)
	if err != nil {
		deploymentApi.logger.Errorf("unable to get assistant whatsapp deployment: %v", err)
		c.JSON(pkg_errors.GetAssistantWhatsappDeploymentGetDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentGetDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAssistantWhatsappDeploymentGetDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentGetDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAssistantWhatsappDeploymentGetDeployment.ErrorMessage),
			},
		})
		return
	}
	if !validator.NonNil(deployment) {
		c.JSON(http.StatusOK, openapi.GetAssistantWhatsappDeploymentResponse{
			Code:    utils.Ptr(int32(http.StatusOK)),
			Success: utils.Ptr(true),
			Data:    nil,
		})
		return
	}

	deploymentId := openapi.Uint64String(strconv.FormatUint(deployment.Id, 10))
	deploymentAssistantId := openapi.Uint64String(strconv.FormatUint(deployment.AssistantId, 10))
	deploymentStatus := deployment.Status.String()

	responseWhatsappOptions := []openapi.Metadata{}
	for _, whatsappOption := range deployment.WhatsappOptions {
		if !validator.NonNil(whatsappOption) {
			continue
		}
		responseWhatsappOptions = append(responseWhatsappOptions, openapi.Metadata{
			Key:   utils.Ptr(whatsappOption.Key),
			Value: utils.Ptr(whatsappOption.Value),
		})
	}
	whatsappProviderName := deployment.WhatsappProvider

	c.JSON(http.StatusOK, openapi.GetAssistantWhatsappDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data: &openapi.AssistantWhatsappDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			UnclearInputTimeout:   deployment.UnclearInputTimeout,
			UnclearInputMessage:   deployment.UnclearInputMessage,
			WhatsappProviderName:  &whatsappProviderName,
			WhatsappOptions:       &responseWhatsappOptions,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		},
	})
}
