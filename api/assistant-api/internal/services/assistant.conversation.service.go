// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_message_gorm "github.com/rapidaai/api/assistant-api/internal/entity/messages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
	workflow_api "github.com/rapidaai/protos"
)

type GetConversationOption struct {
	InjectContext   bool
	InjectArgument  bool
	InjectMetadata  bool
	InjectMetric    bool
	InjectOption    bool
	InjectMessage   bool
	InjectRecording bool
}

func NewDefaultGetConversationOption() *GetConversationOption {
	return &GetConversationOption{
		InjectContext:   true,
		InjectArgument:  true,
		InjectMetadata:  true,
		InjectMetric:    true,
		InjectOption:    true,
		InjectRecording: false,
	}
}

func (gco *GetConversationOption) WithFieldSelector(selectors []*workflow_api.FieldSelector) *GetConversationOption {
	for _, v := range selectors {
		switch v.Field {
		case "context":
			gco.InjectContext = true
		case "argument":
			gco.InjectArgument = true
		case "metadata":
			gco.InjectMetadata = true
		case "metric":
			gco.InjectMetric = true
		case "option":
			gco.InjectOption = true
		case "message":
			gco.InjectMessage = true
		case "recording":
			gco.InjectRecording = true
		}
	}
	return gco
}

// WithInjectContext sets the InjectContext option and returns the modified GetConversationOption
func (o *GetConversationOption) WithInjectContext(inject bool) *GetConversationOption {
	o.InjectContext = inject
	return o
}

// WithInjectArgument sets the InjectArgument option and returns the modified GetConversationOption
func (o *GetConversationOption) WithInjectArgument(inject bool) *GetConversationOption {
	o.InjectArgument = inject
	return o
}

// WithInjectMetadata sets the InjectMetadata option and returns the modified GetConversationOption
func (o *GetConversationOption) WithInjectMetadata(inject bool) *GetConversationOption {
	o.InjectMetadata = inject
	return o
}

// WithInjectMetric sets the InjectMetric option and returns the modified GetConversationOption
func (o *GetConversationOption) WithInjectMetric(inject bool) *GetConversationOption {
	o.InjectMetric = inject
	return o
}
func (o *GetConversationOption) WithInjectRecording(inject bool) *GetConversationOption {
	o.InjectRecording = inject
	return o
}

type GetMessageOption struct {
	InjectMetadata bool
	InjectMetric   bool
}

func NewGetMessageOption() *GetMessageOption {
	return &GetMessageOption{}
}

func NewDefaultGetMessageOption() *GetMessageOption {
	return &GetMessageOption{
		InjectMetadata: true,
		InjectMetric:   true,
	}
}

func (gco *GetMessageOption) WithFieldSelector(selectors []*workflow_api.FieldSelector) *GetMessageOption {
	for _, v := range selectors {
		switch v.Field {
		case "metadata":
			gco.InjectMetadata = true
		case "metric":
			gco.InjectMetric = true

		}
	}
	return gco
}

func (opt *GetMessageOption) WithInjectMetric(ij bool) *GetMessageOption {
	opt.InjectMetric = ij
	return opt
}

func (opt *GetMessageOption) WithInjectMetadata(ij bool) *GetMessageOption {
	opt.InjectMetadata = ij
	return opt
}

type AssistantConversationService interface {
	//
	GetAll(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		opts *GetConversationOption,
	) (int64, []*internal_conversation_entity.AssistantConversation, error)

	// later you will ask why two let me tell you one for end user
	// comming from request adapter
	// anotehr is CRM
	GetConversation(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		opts *GetConversationOption) (*internal_conversation_entity.AssistantConversation, error)

	Get(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		opts *GetConversationOption) (*internal_conversation_entity.AssistantConversation, error)

	//
	GetAllConversationMessage(context.Context,
		*types.Authentication,
		uint64,
		[]*workflow_api.Criteria,
		*workflow_api.Paginate,
		*workflow_api.Ordering, *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	GetAllAssistantMessage(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		ordering *workflow_api.Ordering, opts *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	GetAllMessage(
		ctx context.Context,
		auth *types.Authentication,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		ordering *workflow_api.Ordering, opts *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	CreateCustomConversationMetric(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		metrics []*workflow_api.Metric,
	) ([]*internal_conversation_entity.AssistantConversationMetric, error)

	CreateConversationMessage(
		ctx context.Context,
		auth *types.Authentication,
		source utils.RapidaSource,
		assistantId, assistantProviderModelId,
		assistantConversationId uint64,
		messageId string,
		role string,
		message string,
	) (*internal_message_gorm.AssistantConversationMessage, error)

	CreateOrUpdateMessageMetadata(
		ctx context.Context,
		auth *types.Authentication,
		assistantConversationId uint64,
		assistantConversationMessageId string,
		metadata []*protos.Metadata,
	) ([]*internal_message_gorm.AssistantConversationMessageMetadata, error)

	CreateOrUpdateMessageMetrics(ctx context.Context,
		auth *types.Authentication,
		assistantConversationId uint64,
		assistantConversationMessageId string,
		metrics []*protos.Metric,
	) ([]*internal_message_gorm.AssistantConversationMessageMetric, error)

	//
	CreateConversation(
		ctx context.Context,
		auth *types.Authentication,
		identifier string,
		assistantId uint64,
		assistantProviderModelId uint64,
		direction type_enums.ConversationDirection, source utils.RapidaSource) (*internal_conversation_entity.AssistantConversation, error)

	// all about conversation

	CreateOrUpdateConversationArgument(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		arguments map[string]interface{},
	) ([]*internal_conversation_entity.AssistantConversationArgument, error)

	CreateOrUpdateConversationOption(ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		opts map[string]interface{}) ([]*internal_conversation_entity.AssistantConversationOption, error)

	//
	CreateOrUpdateConversationMetrics(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		metrics []*protos.Metric,
	) ([]*internal_conversation_entity.AssistantConversationMetric, error)

	CreateOrUpdateConversationMetadata(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		metadata []*protos.Metadata,
	) ([]*internal_conversation_entity.AssistantConversationMetadata, error)

	CreateConversationRecording(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		assistantConversationId uint64,
		user, assistant, conversation []byte,
	) (*internal_conversation_entity.AssistantConversationRecording, error)
}
