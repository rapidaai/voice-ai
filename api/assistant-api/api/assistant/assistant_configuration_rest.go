// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) CreateAssistantConfigurationRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		platformError := pkg_errors.AssistantConfigurationUnauthenticated
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		platformError := pkg_errors.AssistantConfigurationMissingAuthScope
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}

	var req openapi.CreateAssistantConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformError := pkg_errors.AssistantConfigurationInvalidRequest
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	assistantId, err := utils.StringToUint64(req.AssistantId)
	if err != nil || !validator.NonZero(assistantId) {
		platformError := pkg_errors.AssistantConfigurationInvalidAssistantID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(req.ConfigurationType) {
		platformError := pkg_errors.AssistantConfigurationMissingType
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.OneOf(
		req.ConfigurationType,
		string(internal_assistant_entity.AssistantConfigurationTypeAuthentication),
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		string(internal_assistant_entity.AssistantConfigurationTypeAnalysis),
		string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
		string(internal_assistant_entity.AssistantConfigurationTypeStorage),
	) {
		platformError := pkg_errors.AssistantConfigurationInvalidType
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(req.Provider) {
		platformError := pkg_errors.AssistantConfigurationMissingProvider
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if req.Options != nil {
		for _, option := range *req.Options {
			if !validator.NonNil(option.Key) || !validator.NotBlank(*option.Key) {
				platformError := pkg_errors.AssistantConfigurationInvalidOption
				c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
						ErrorMessage: utils.Ptr(platformError.Error),
						HumanMessage: utils.Ptr(platformError.ErrorMessage),
					},
				})
				return
			}
		}
	}
	options := assistantConfigurationOpenAPIOptions(req.Options)
	configuration, err := assistantApi.assistantConfigService.Create(
		c.Request.Context(),
		iAuth,
		assistantId,
		req.ConfigurationType,
		req.Provider,
		req.Enabled,
		options,
	)
	if err != nil {
		assistantApi.logger.Errorf("unable to create assistant configuration: %v", err)
		platformError := pkg_errors.AssistantConfigurationCreate
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	c.JSON(http.StatusOK, openapi.GetAssistantConfigurationResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    assistantConfigurationOpenAPI(configuration),
	})
}

func (assistantApi *assistantGrpcApi) UpdateAssistantConfigurationRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		platformError := pkg_errors.AssistantConfigurationUnauthenticated
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		platformError := pkg_errors.AssistantConfigurationMissingAuthScope
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}

	assistantId, assistantErr := utils.StringToUint64(c.Param("assistantId"))
	if assistantErr != nil || !validator.NonZero(assistantId) {
		platformError := pkg_errors.AssistantConfigurationInvalidAssistantID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configurationId, configurationErr := utils.StringToUint64(c.Param("id"))
	if configurationErr != nil || !validator.NonZero(configurationId) {
		platformError := pkg_errors.AssistantConfigurationInvalidID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	var req openapi.UpdateAssistantConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformError := pkg_errors.AssistantConfigurationInvalidRequest
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(req.ConfigurationType) {
		platformError := pkg_errors.AssistantConfigurationMissingType
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.OneOf(
		req.ConfigurationType,
		string(internal_assistant_entity.AssistantConfigurationTypeAuthentication),
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		string(internal_assistant_entity.AssistantConfigurationTypeAnalysis),
		string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
		string(internal_assistant_entity.AssistantConfigurationTypeStorage),
	) {
		platformError := pkg_errors.AssistantConfigurationInvalidType
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if !validator.NotBlank(req.Provider) {
		platformError := pkg_errors.AssistantConfigurationMissingProvider
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	if req.Options != nil {
		for _, option := range *req.Options {
			if !validator.NonNil(option.Key) || !validator.NotBlank(*option.Key) {
				platformError := pkg_errors.AssistantConfigurationInvalidOption
				c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
						ErrorMessage: utils.Ptr(platformError.Error),
						HumanMessage: utils.Ptr(platformError.ErrorMessage),
					},
				})
				return
			}
		}
	}
	options := assistantConfigurationOpenAPIOptions(req.Options)
	configuration, err := assistantApi.assistantConfigService.Update(
		c.Request.Context(),
		iAuth,
		configurationId,
		assistantId,
		req.ConfigurationType,
		req.Provider,
		req.Enabled,
		options,
	)
	if err != nil {
		assistantApi.logger.Errorf("unable to update assistant configuration: %v", err)
		platformError := pkg_errors.AssistantConfigurationUpdate
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	c.JSON(http.StatusOK, openapi.GetAssistantConfigurationResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    assistantConfigurationOpenAPI(configuration),
	})
}

func (assistantApi *assistantGrpcApi) GetAssistantConfigurationRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		platformError := pkg_errors.AssistantConfigurationUnauthenticated
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		platformError := pkg_errors.AssistantConfigurationMissingAuthScope
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}

	assistantId, assistantErr := utils.StringToUint64(c.Param("assistantId"))
	if assistantErr != nil || !validator.NonZero(assistantId) {
		platformError := pkg_errors.AssistantConfigurationInvalidAssistantID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configurationId, configurationErr := utils.StringToUint64(c.Param("id"))
	if configurationErr != nil || !validator.NonZero(configurationId) {
		platformError := pkg_errors.AssistantConfigurationInvalidID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configuration, err := assistantApi.assistantConfigService.Get(c.Request.Context(), iAuth, configurationId, assistantId)
	if err != nil {
		assistantApi.logger.Errorf("unable to get assistant configuration: %v", err)
		platformError := pkg_errors.AssistantConfigurationGet
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	c.JSON(http.StatusOK, openapi.GetAssistantConfigurationResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    assistantConfigurationOpenAPI(configuration),
	})
}

func (assistantApi *assistantGrpcApi) GetAllAssistantConfigurationRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		platformError := pkg_errors.AssistantConfigurationUnauthenticated
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		platformError := pkg_errors.AssistantConfigurationMissingAuthScope
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := utils.StringToUint64(c.Param("assistantId"))
	if err != nil || !validator.NonZero(assistantId) {
		platformError := pkg_errors.AssistantConfigurationInvalidAssistantID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configurationType := c.Query("configurationType")
	if validator.NotBlank(configurationType) && !validator.OneOf(
		configurationType,
		string(internal_assistant_entity.AssistantConfigurationTypeAuthentication),
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		string(internal_assistant_entity.AssistantConfigurationTypeAnalysis),
		string(internal_assistant_entity.AssistantConfigurationTypeTelemetry),
		string(internal_assistant_entity.AssistantConfigurationTypeStorage),
	) {
		platformError := pkg_errors.AssistantConfigurationInvalidType
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	page, pageErr := utils.StringToUint32(c.DefaultQuery("page", "1"))
	if pageErr != nil || !validator.NonZero(page) {
		platformError := pkg_errors.AssistantConfigurationInvalidRequest
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	pageSize, pageSizeErr := utils.StringToUint32(c.DefaultQuery("pageSize", "20"))
	if pageSizeErr != nil || !validator.NonZero(pageSize) {
		platformError := pkg_errors.AssistantConfigurationInvalidRequest
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	cnt, configurations, err := assistantApi.assistantConfigService.GetAll(
		c.Request.Context(),
		iAuth,
		assistantId,
		configurationType,
		c.Query("provider"),
		nil,
		&protos.Paginate{
			Page:     page,
			PageSize: pageSize,
		},
	)
	if err != nil {
		assistantApi.logger.Errorf("unable to get assistant configurations: %v", err)
		platformError := pkg_errors.AssistantConfigurationGetAll
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	out := make([]openapi.AssistantConfiguration, 0, len(configurations))
	for _, configuration := range configurations {
		mapped := assistantConfigurationOpenAPI(configuration)
		if mapped != nil {
			out = append(out, *mapped)
		}
	}
	c.JSON(http.StatusOK, openapi.GetAllAssistantConfigurationResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    &out,
		Paginated: &openapi.Paginated{
			CurrentPage: utils.Ptr(page),
			TotalItem:   utils.Ptr(uint32(cnt)),
		},
	})
}

func (assistantApi *assistantGrpcApi) DeleteAssistantConfigurationRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		platformError := pkg_errors.AssistantConfigurationUnauthenticated
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		platformError := pkg_errors.AssistantConfigurationMissingAuthScope
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}

	assistantId, assistantErr := utils.StringToUint64(c.Param("assistantId"))
	if assistantErr != nil || !validator.NonZero(assistantId) {
		platformError := pkg_errors.AssistantConfigurationInvalidAssistantID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configurationId, configurationErr := utils.StringToUint64(c.Param("id"))
	if configurationErr != nil || !validator.NonZero(configurationId) {
		platformError := pkg_errors.AssistantConfigurationInvalidID
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	configuration, err := assistantApi.assistantConfigService.Delete(c.Request.Context(), iAuth, configurationId, assistantId)
	if err != nil {
		assistantApi.logger.Errorf("unable to delete assistant configuration: %v", err)
		platformError := pkg_errors.AssistantConfigurationDelete
		c.JSON(platformError.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(platformError.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(platformError.CodeString())),
				ErrorMessage: utils.Ptr(platformError.Error),
				HumanMessage: utils.Ptr(platformError.ErrorMessage),
			},
		})
		return
	}
	c.JSON(http.StatusOK, openapi.GetAssistantConfigurationResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    assistantConfigurationOpenAPI(configuration),
	})
}

func assistantConfigurationOpenAPIOptions(options *[]openapi.Metadata) []*protos.Metadata {
	if options == nil {
		return nil
	}
	out := make([]*protos.Metadata, 0, len(*options))
	for _, option := range *options {
		if option.Key == nil {
			continue
		}
		value := ""
		if option.Value != nil {
			value = *option.Value
		}
		out = append(out, &protos.Metadata{
			Key:   *option.Key,
			Value: value,
		})
	}
	return out
}

func assistantConfigurationOpenAPI(configuration *internal_assistant_entity.AssistantConfiguration) *openapi.AssistantConfiguration {
	if configuration == nil {
		return nil
	}
	id := openapi.Uint64String(strconv.FormatUint(configuration.Id, 10))
	assistantId := openapi.Uint64String(strconv.FormatUint(configuration.AssistantId, 10))
	projectId := openapi.Uint64String(strconv.FormatUint(configuration.ProjectId, 10))
	organizationId := openapi.Uint64String(strconv.FormatUint(configuration.OrganizationId, 10))
	configurationType := string(configuration.ConfigurationType)
	status := configuration.Status.String()
	createdDate := time.Time(configuration.CreatedDate)
	updatedDate := time.Time(configuration.UpdatedDate)
	options := make([]openapi.Metadata, 0, len(configuration.Options))
	for _, option := range configuration.Options {
		if option == nil {
			continue
		}
		options = append(options, openapi.Metadata{
			Key:   utils.Ptr(option.Key),
			Value: utils.Ptr(option.Value),
		})
	}
	var createdActor *openapi.AuditActor
	createdActorIdentity := types.ActorIdentity{
		Type: types.ActorType(configuration.CreatedActorType),
		ID:   configuration.CreatedActorID,
	}
	if createdActorIdentity.Validate() == nil {
		createdActor = &openapi.AuditActor{}
		_ = utils.Cast(map[string]string{
			"type": createdActorIdentity.Type.String(),
			"id":   strconv.FormatUint(createdActorIdentity.ID, 10),
		}, createdActor)
	}
	var updatedActor *openapi.AuditActor
	updatedActorIdentity := types.ActorIdentity{
		Type: types.ActorType(configuration.UpdatedActorType),
		ID:   configuration.UpdatedActorID,
	}
	if updatedActorIdentity.Validate() == nil {
		updatedActor = &openapi.AuditActor{}
		_ = utils.Cast(map[string]string{
			"type": updatedActorIdentity.Type.String(),
			"id":   strconv.FormatUint(updatedActorIdentity.ID, 10),
		}, updatedActor)
	}
	return &openapi.AssistantConfiguration{
		Id:                &id,
		AssistantId:       &assistantId,
		ProjectId:         &projectId,
		OrganizationId:    &organizationId,
		ConfigurationType: &configurationType,
		Provider:          &configuration.Provider,
		Enabled:           &configuration.Enabled,
		Options:           &options,
		Status:            &status,
		CreatedActor:      createdActor,
		UpdatedActor:      updatedActor,
		CreatedDate:       &createdDate,
		UpdatedDate:       &updatedDate,
	}
}
