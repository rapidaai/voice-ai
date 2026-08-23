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

// CreatePhoneCallRest initiates an outbound phone call from the REST API.
func (cApi *ConversationApi) CreatePhoneCallRest(c *gin.Context) {
	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		c.JSON(pkg_errors.CreatePhoneCallUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": scopeErr.Error()})
		return
	}

	observer := cApi.Observability(c, iAuth, observability.WithGracePeriod())
	defer observer.Close(context.Background())

	var ir openapi.CreatePhoneCallRequest
	if err := c.ShouldBindJSON(&ir); err != nil {
		cApi.logger.Errorf("create phone call invalid request: %v", err)
		_ = observer.Record(c, observability.ProjectScope{}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall REST validation failed: invalid request body",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "request",
				"error_code":    pkg_errors.CreatePhoneCallInvalidRequest.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidRequest.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidRequest.HTTPStatusCode),
				"detail":        err.Error(),
			},
		})
		c.JSON(pkg_errors.CreatePhoneCallInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	_ = observer.Record(c, observability.ProjectScope{}, observability.RecordLog{
		Level:   observability.LevelInfo,
		Message: "CreatePhoneCall REST request received",
		Attributes: observability.Attributes{
			"payload": observability.AttributeValue(ir),
		},
	})
	if !validator.NonNil(ir.ToNumber) || !validator.NotBlank(*ir.ToNumber) {
		_ = observer.Record(c, observability.ProjectScope{}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall REST validation failed: missing to_number",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "to_number",
				"error_code":    pkg_errors.CreatePhoneCallMissingToNumber.CodeString(),
				"error":         pkg_errors.CreatePhoneCallMissingToNumber.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallMissingToNumber.HTTPStatusCode),
			},
		})
		c.JSON(pkg_errors.CreatePhoneCallMissingToNumber.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallMissingToNumber.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallMissingToNumber.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallMissingToNumber.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallMissingToNumber.ErrorMessage),
			},
		})
		return
	}

	var assistantID uint64
	version := ""
	if validator.NonNil(ir.Assistant) {
		if validator.NonNil(ir.Assistant.AssistantId) {
			assistantID, _ = utils.StringToUint64(*ir.Assistant.AssistantId)
		}
		if validator.NonNil(ir.Assistant.Version) {
			version = *ir.Assistant.Version
		}
	}
	assistant := &protos.AssistantDefinition{
		AssistantId: assistantID,
		Version:     version,
	}
	preset.AssistantDefinition(assistant)
	if !validator.OfAssistantDefinition(assistant) {
		scope := observability.Scope(observability.ProjectScope{})
		if assistant.GetAssistantId() > 0 {
			scope = observability.AssistantScope{AssistantID: assistant.GetAssistantId()}
		}
		_ = observer.Record(c, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall REST validation failed: invalid assistant",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "assistant",
				"error_code":    pkg_errors.CreatePhoneCallInvalidAssistant.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidAssistant.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidAssistant.HTTPStatusCode),
			},
		})
		c.JSON(pkg_errors.CreatePhoneCallInvalidAssistant.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallInvalidAssistant.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallInvalidAssistant.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidAssistant.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidAssistant.ErrorMessage),
			},
		})
		return
	}

	fromNumber := ""
	if validator.NonNil(ir.FromNumber) {
		fromNumber = *ir.FromNumber
	}
	var metadata map[string]interface{}
	if validator.NonNil(ir.Metadata) {
		metadata = *ir.Metadata
	}
	var args map[string]interface{}
	if validator.NonNil(ir.Args) {
		args = *ir.Args
	}
	var opts map[string]interface{}
	if validator.NonNil(ir.Options) {
		opts = *ir.Options
	}
	if value, err := utils.Option(opts).GetFloat64(internal_options.ExperienceOptionUnclearInputTimeout); err == nil && !validator.Between(value, 2, 10) {
		c.JSON(pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallInvalidOptions.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidOptions.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallInvalidOptions.ErrorMessage),
			},
		})
		return
	}

	result := cApi.channelPipeline.Run(c, channel_pipeline.OutboundRequestedPipeline{
		ID:          fmt.Sprintf("%d", assistant.GetAssistantId()),
		Auth:        iAuth,
		AssistantID: assistant.GetAssistantId(),
		Version:     assistant.GetVersion(),
		ToPhone:     *ir.ToNumber,
		FromPhone:   fromNumber,
		Metadata:    metadata,
		Args:        args,
		Options:     opts,
		Observer:    observer,
	})
	if result.Error != nil {
		cApi.logger.Errorf("outbound call failed: %v", result.Error)
		scope := observability.Scope(observability.AssistantScope{AssistantID: assistant.GetAssistantId()})
		if result.ConversationID > 0 {
			scope = observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: assistant.GetAssistantId()},
				ConversationID: result.ConversationID,
			}
		}
		_ = observer.Record(c, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall REST outbound dispatch failed",
			Attributes: observability.Attributes{
				"failure_stage": "dispatch",
				"context_id":    result.ContextID,
				"error_code":    pkg_errors.CreatePhoneCallInitiateOutbound.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInitiateOutbound.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCode),
				"detail":        result.Error.Error(),
			},
		})
		c.JSON(pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreatePhoneCallInitiateOutbound.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreatePhoneCallInitiateOutbound.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreatePhoneCallInitiateOutbound.ErrorMessage),
			},
		})
		return
	}

	cApi.logger.Infof("outbound call dispatched: contextId=%s, conversationId=%d",
		result.ContextID, result.ConversationID)

	c.JSON(http.StatusOK, openapi.CreatePhoneCallResponse{
		Code:    utils.Ptr(int32(200)),
		Success: utils.Ptr(true),
		Data: &openapi.AssistantConversation{
			Id: utils.Ptr(openapi.Uint64String(strconv.FormatUint(result.ConversationID, 10))),
		},
	})
}
