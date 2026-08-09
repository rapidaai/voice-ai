// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/preset"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// CreateBulkPhoneCallRest initiates outbound phone calls from the REST API.
func (cApi *ConversationApi) CreateBulkPhoneCallRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateBulkPhoneCallUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallUnauthenticated.ErrorMessage),
			},
		})
		return
	}

	var ir openapi.CreateBulkPhoneCallRequest
	if err := c.ShouldBindJSON(&ir); err != nil {
		cApi.logger.Errorf("create bulk phone call invalid request: %v", err)
		c.JSON(pkg_errors.CreateBulkPhoneCallInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	if !validator.NonNil(ir.PhoneCalls) || !validator.NotEmpty(*ir.PhoneCalls) {
		c.JSON(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.ErrorMessage),
			},
		})
		return
	}

	conversations := make([]openapi.AssistantConversation, 0, len(*ir.PhoneCalls))
	observer := cApi.Observability(c, auth, observability.WithGracePeriod())
	defer observer.Close(context.Background())
	for _, phoneCall := range *ir.PhoneCalls {
		if !validator.NonNil(phoneCall.ToNumber) || !validator.NotBlank(*phoneCall.ToNumber) {
			c.JSON(pkg_errors.CreateBulkPhoneCallMissingToNumber.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingToNumber.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallMissingToNumber.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingToNumber.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallMissingToNumber.ErrorMessage),
				},
			})
			return
		}

		var assistantID uint64
		version := ""
		if validator.NonNil(phoneCall.Assistant) {
			if validator.NonNil(phoneCall.Assistant.AssistantId) {
				assistantID, _ = utils.StringToUint64(*phoneCall.Assistant.AssistantId)
			}
			if validator.NonNil(phoneCall.Assistant.Version) {
				version = *phoneCall.Assistant.Version
			}
		}
		assistant := &protos.AssistantDefinition{
			AssistantId: assistantID,
			Version:     version,
		}
		preset.AssistantDefinition(assistant)
		if !validator.OfAssistantDefinition(assistant) {
			c.JSON(pkg_errors.CreateBulkPhoneCallInvalidAssistant.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidAssistant.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallInvalidAssistant.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidAssistant.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidAssistant.ErrorMessage),
				},
			})
			return
		}

		fromNumber := ""
		if validator.NonNil(phoneCall.FromNumber) {
			fromNumber = *phoneCall.FromNumber
		}
		var metadata map[string]interface{}
		if validator.NonNil(phoneCall.Metadata) {
			metadata = *phoneCall.Metadata
		}
		var args map[string]interface{}
		if validator.NonNil(phoneCall.Args) {
			args = *phoneCall.Args
		}
		var opts map[string]interface{}
		if validator.NonNil(phoneCall.Options) {
			opts = *phoneCall.Options
		}
		if value, err := utils.Option(opts).GetFloat64(internal_options.ExperienceOptionUnclearInputTimeout); err == nil && !validator.Between(value, 2, 10) {
			c.JSON(pkg_errors.CreateBulkPhoneCallInvalidOptions.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidOptions.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallInvalidOptions.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidOptions.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInvalidOptions.ErrorMessage),
				},
			})
			return
		}
		result := cApi.channelPipeline.Run(c, channel_pipeline.OutboundRequestedPipeline{
			ID:          fmt.Sprintf("%d", assistant.GetAssistantId()),
			Auth:        auth,
			AssistantID: assistant.GetAssistantId(),
			Version:     assistant.GetVersion(),
			ToPhone:     *phoneCall.ToNumber,
			FromPhone:   fromNumber,
			Metadata:    metadata,
			Args:        args,
			Options:     opts,
			Observer:    observer,
		})

		if result.Error != nil {
			cApi.logger.Errorf("bulk outbound call failed: %v", result.Error)
			c.JSON(pkg_errors.CreateBulkPhoneCallInitiateOutbound.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateBulkPhoneCallInitiateOutbound.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateBulkPhoneCallInitiateOutbound.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInitiateOutbound.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateBulkPhoneCallInitiateOutbound.ErrorMessage),
				},
			})
			return
		}

		cApi.logger.Infof("bulk outbound call dispatched: contextId=%s, conversationId=%d",
			result.ContextID, result.ConversationID)

		conversations = append(conversations, openapi.AssistantConversation{
			Id: utils.Ptr(openapi.Uint64String(strconv.FormatUint(result.ConversationID, 10))),
		})
	}

	c.JSON(http.StatusOK, openapi.CreateBulkPhoneCallResponse{
		Code:    utils.Ptr(int32(200)),
		Success: utils.Ptr(true),
		Data:    &conversations,
	})
}
