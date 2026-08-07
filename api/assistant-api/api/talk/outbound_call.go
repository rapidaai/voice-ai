// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"errors"
	"fmt"

	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/preset"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// CreatePhoneCall initiates an outbound phone call.
// Thin controller — pipeline handles: validate, load assistant, create conversation,
// save context, create observer, dispatch. Controller just validates input and waits.
func (cApi *ConversationGrpcApi) CreatePhoneCall(ctx context.Context, ir *protos.CreatePhoneCallRequest) (*protos.CreatePhoneCallResponse, error) {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallUnauthenticated.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallUnauthenticated.Error,
				HumanMessage: pkg_errors.CreatePhoneCallUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallUnauthenticated.Error)
	}

	observer := cApi.Observability(ctx, auth, observability.WithGracePeriod())
	defer observer.Close(context.Background())
	_ = observer.Record(ctx, observability.Scope(observability.ProjectScope{}), observability.RecordLog{
		Level:   observability.LevelInfo,
		Message: "CreatePhoneCall request received",
		Attributes: observability.Attributes{
			"payload": observability.AttributeValue(ir),
		},
	})

	if utils.IsEmpty(ir.GetToNumber()) {
		scope := observability.Scope(observability.ProjectScope{})
		if ir.GetAssistant().GetAssistantId() > 0 {
			scope = observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}
		}
		_ = observer.Record(ctx, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: missing to_number",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "to_number",
				"error_code":    pkg_errors.CreatePhoneCallMissingToNumber.CodeString(),
				"error":         pkg_errors.CreatePhoneCallMissingToNumber.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallMissingToNumber.HTTPStatusCode),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallMissingToNumber.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallMissingToNumber.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallMissingToNumber.Error,
				HumanMessage: pkg_errors.CreatePhoneCallMissingToNumber.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallMissingToNumber.Error)
	}

	preset.AssistantDefinition(ir.GetAssistant())
	if !validator.OfAssistantDefinition(ir.GetAssistant()) {
		scope := observability.Scope(observability.ProjectScope{})
		if ir.GetAssistant().GetAssistantId() > 0 {
			scope = observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}
		}
		_ = observer.Record(ctx, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: invalid assistant",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "assistant",
				"error_code":    pkg_errors.CreatePhoneCallInvalidAssistant.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidAssistant.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidAssistant.HTTPStatusCode),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInvalidAssistant.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInvalidAssistant.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInvalidAssistant.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInvalidAssistant.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInvalidAssistant.Error)
	}

	mtd, err := utils.AnyMapToInterfaceMap(ir.GetMetadata())
	if err != nil {
		cApi.logger.Errorf("create phone call invalid metadata: %v", err)
		scope := observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}
		_ = observer.Record(ctx, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: invalid metadata",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "metadata",
				"error_code":    pkg_errors.CreatePhoneCallInvalidMetadata.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidMetadata.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidMetadata.HTTPStatusCode),
				"detail":        err.Error(),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInvalidMetadata.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInvalidMetadata.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInvalidMetadata.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInvalidMetadata.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInvalidMetadata.Error)
	}
	args, err := utils.AnyMapToInterfaceMap(ir.GetArgs())
	if err != nil {
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: invalid arguments",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "args",
				"error_code":    pkg_errors.CreatePhoneCallInvalidArguments.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidArguments.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidArguments.HTTPStatusCode),
				"detail":        err.Error(),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInvalidArguments.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInvalidArguments.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInvalidArguments.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInvalidArguments.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInvalidArguments.Error)
	}
	opts, err := utils.AnyMapToInterfaceMap(ir.GetOptions())
	if err != nil {
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: invalid options",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         "options",
				"error_code":    pkg_errors.CreatePhoneCallInvalidOptions.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidOptions.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCode),
				"detail":        err.Error(),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInvalidOptions.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInvalidOptions.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInvalidOptions.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInvalidOptions.Error)
	}
	if value, err := utils.Option(opts).GetFloat64(internal_options.ExperienceOptionUnclearInputTimeout); err == nil && !validator.Between(value, 2, 10) {
		_ = observer.Record(ctx, observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall validation failed: invalid options",
			Attributes: observability.Attributes{
				"failure_stage": "validation",
				"field":         internal_options.ExperienceOptionUnclearInputTimeout,
				"error_code":    pkg_errors.CreatePhoneCallInvalidOptions.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInvalidOptions.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCode),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInvalidOptions.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInvalidOptions.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInvalidOptions.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInvalidOptions.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInvalidOptions.Error)
	}

	// Pipeline handles the full outbound flow
	result := cApi.channelPipeline.Run(ctx, channel_pipeline.OutboundRequestedPipeline{
		ID:          fmt.Sprintf("%d", ir.GetAssistant().GetAssistantId()),
		Auth:        auth,
		AssistantID: ir.GetAssistant().GetAssistantId(),
		Version:     ir.GetAssistant().GetVersion(),
		ToPhone:     ir.GetToNumber(),
		FromPhone:   ir.GetFromNumber(),
		Metadata:    mtd,
		Args:        args,
		Options:     opts,
		Observer:    observer,
	})

	if result.Error != nil {
		cApi.logger.Errorf("outbound call failed: %v", result.Error)
		scope := observability.Scope(observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()})
		if result.ConversationID > 0 {
			scope = observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: ir.GetAssistant().GetAssistantId()},
				ConversationID: result.ConversationID,
			}
		}
		_ = observer.Record(ctx, scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "CreatePhoneCall outbound dispatch failed",
			Attributes: observability.Attributes{
				"failure_stage": "dispatch",
				"context_id":    result.ContextID,
				"error_code":    pkg_errors.CreatePhoneCallInitiateOutbound.CodeString(),
				"error":         pkg_errors.CreatePhoneCallInitiateOutbound.Error,
				"http_status":   fmt.Sprintf("%d", pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCode),
				"detail":        result.Error.Error(),
			},
		})
		return &protos.CreatePhoneCallResponse{
			Code:    pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreatePhoneCallInitiateOutbound.Code),
				ErrorMessage: pkg_errors.CreatePhoneCallInitiateOutbound.Error,
				HumanMessage: pkg_errors.CreatePhoneCallInitiateOutbound.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreatePhoneCallInitiateOutbound.Error)
	}

	cApi.logger.Infof("outbound call dispatched: contextId=%s, conversationId=%d",
		result.ContextID, result.ConversationID)

	return utils.Success[protos.CreatePhoneCallResponse, *protos.AssistantConversation](&protos.AssistantConversation{
		Id: result.ConversationID,
	})
}

// InitiateBulkAssistantTalk implements protos.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateBulkPhoneCall(ctx context.Context, ir *protos.CreateBulkPhoneCallRequest) (*protos.CreateBulkPhoneCallResponse, error) {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return &protos.CreateBulkPhoneCallResponse{
			Code:    pkg_errors.CreateBulkPhoneCallUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallUnauthenticated.Code),
				ErrorMessage: pkg_errors.CreateBulkPhoneCallUnauthenticated.Error,
				HumanMessage: pkg_errors.CreateBulkPhoneCallUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateBulkPhoneCallUnauthenticated.Error)
	}

	if len(ir.GetPhoneCalls()) == 0 {
		return &protos.CreateBulkPhoneCallResponse{
			Code:    pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.HTTPStatusCodeInt32(),
			Success: false,
			Error: &protos.Error{
				ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Code),
				ErrorMessage: pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Error,
				HumanMessage: pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Error)
	}

	conversations := make([]*protos.AssistantConversation, 0, len(ir.GetPhoneCalls()))
	for _, phoneCall := range ir.GetPhoneCalls() {
		if utils.IsEmpty(phoneCall.GetToNumber()) {
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallMissingToNumber.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallMissingToNumber.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallMissingToNumber.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallMissingToNumber.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallMissingToNumber.Error)
		}

		preset.AssistantDefinition(phoneCall.GetAssistant())
		if !validator.OfAssistantDefinition(phoneCall.GetAssistant()) {
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInvalidAssistant.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInvalidAssistant.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInvalidAssistant.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInvalidAssistant.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInvalidAssistant.Error)
		}

		mtd, err := utils.AnyMapToInterfaceMap(phoneCall.GetMetadata())
		if err != nil {
			cApi.logger.Errorf("create bulk phone call invalid metadata: %v", err)
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInvalidMetadata.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInvalidMetadata.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInvalidMetadata.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInvalidMetadata.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInvalidMetadata.Error)
		}
		args, err := utils.AnyMapToInterfaceMap(phoneCall.GetArgs())
		if err != nil {
			cApi.logger.Errorf("create bulk phone call invalid arguments: %v", err)
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInvalidArguments.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInvalidArguments.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInvalidArguments.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInvalidArguments.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInvalidArguments.Error)
		}
		opts, err := utils.AnyMapToInterfaceMap(phoneCall.GetOptions())
		if err != nil {
			cApi.logger.Errorf("create bulk phone call invalid options: %v", err)
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInvalidOptions.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInvalidOptions.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInvalidOptions.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInvalidOptions.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInvalidOptions.Error)
		}
		if value, err := utils.Option(opts).GetFloat64(internal_options.ExperienceOptionUnclearInputTimeout); err == nil && !validator.Between(value, 2, 10) {
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInvalidOptions.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInvalidOptions.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInvalidOptions.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInvalidOptions.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInvalidOptions.Error)
		}

		observer := cApi.Observability(ctx, auth, observability.WithGracePeriod())
		result := cApi.channelPipeline.Run(ctx, channel_pipeline.OutboundRequestedPipeline{
			ID:          fmt.Sprintf("%d", phoneCall.GetAssistant().GetAssistantId()),
			Auth:        auth,
			AssistantID: phoneCall.GetAssistant().GetAssistantId(),
			Version:     phoneCall.GetAssistant().GetVersion(),
			ToPhone:     phoneCall.GetToNumber(),
			FromPhone:   phoneCall.GetFromNumber(),
			Metadata:    mtd,
			Args:        args,
			Options:     opts,
			Observer:    observer,
		})
		if err := observer.Close(context.Background()); err != nil {
			cApi.logger.Errorf("failed to close bulk outbound observability recorder: %v", err)
		}
		if result.Error != nil {
			cApi.logger.Errorf("bulk outbound call failed: %v", result.Error)
			return &protos.CreateBulkPhoneCallResponse{
				Code:    pkg_errors.CreateBulkPhoneCallInitiateOutbound.HTTPStatusCodeInt32(),
				Success: false,
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.CreateBulkPhoneCallInitiateOutbound.Code),
					ErrorMessage: pkg_errors.CreateBulkPhoneCallInitiateOutbound.Error,
					HumanMessage: pkg_errors.CreateBulkPhoneCallInitiateOutbound.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateBulkPhoneCallInitiateOutbound.Error)
		}

		cApi.logger.Infof("bulk outbound call dispatched: contextId=%s, conversationId=%d",
			result.ContextID, result.ConversationID)

		conversations = append(conversations, &protos.AssistantConversation{
			Id: result.ConversationID,
		})
	}

	return utils.Success[protos.CreateBulkPhoneCallResponse, []*protos.AssistantConversation](conversations)
}
