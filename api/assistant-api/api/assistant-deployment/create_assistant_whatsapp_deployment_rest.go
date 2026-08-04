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

func (deploymentApi *AssistantDeploymentApi) CreateAssistantWhatsappDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	var request openapi.CreateAssistantWhatsappDeploymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		deploymentApi.logger.Errorf("create assistant whatsapp deployment invalid request: %v", err)
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(string(request.AssistantId))
	if err != nil || !validator.NonZero(assistantId) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(request.WhatsappProviderName) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentMissingProvider.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.UnclearInputTimeout) && !validator.Between(*request.UnclearInputTimeout, 0.5, 5) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidUnclearTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeout) && !validator.Between(int(*request.IdealTimeout), 5, 120) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidIdealTimeout.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.IdealTimeoutBackoff) && !validator.Between(int(*request.IdealTimeoutBackoff), 0, 5) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidTimeoutBackoff.ErrorMessage),
			},
		})
		return
	}
	if validator.NonNil(request.MaxSessionDuration) && !validator.Between(int(*request.MaxSessionDuration), 180, 600) {
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentInvalidSessionDuration.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidSessionDuration.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentInvalidSessionDuration.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidSessionDuration.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentInvalidSessionDuration.ErrorMessage),
			},
		})
		return
	}

	whatsappOptions := []*assistant_api.Metadata{}
	if validator.NonNil(request.WhatsappOptions) {
		for _, whatsappOption := range *request.WhatsappOptions {
			key := ""
			if validator.NonNil(whatsappOption.Key) {
				key = *whatsappOption.Key
			}
			value := ""
			if validator.NonNil(whatsappOption.Value) {
				value = *whatsappOption.Value
			}
			whatsappOptions = append(whatsappOptions, &assistant_api.Metadata{Key: key, Value: value})
		}
	}

	deployment, err := deploymentApi.deploymentService.CreateWhatsappDeployment(
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
		request.WhatsappProviderName,
		whatsappOptions,
	)
	if err != nil {
		deploymentApi.logger.Errorf("unable to create assistant whatsapp deployment: %v", err)
		c.JSON(pkg_errors.CreateAssistantWhatsappDeploymentCreateDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentCreateDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantWhatsappDeploymentCreateDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentCreateDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantWhatsappDeploymentCreateDeployment.ErrorMessage),
			},
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
