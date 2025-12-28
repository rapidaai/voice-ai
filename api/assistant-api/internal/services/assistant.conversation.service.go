// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_conversation_gorm "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_message_gorm "github.com/rapidaai/api/assistant-api/internal/entity/messages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	workflow_api "github.com/rapidaai/protos"
)

type GetConversationOption struct {
	InjectContext        bool
	InjectArgument       bool
	InjectMetadata       bool
	InjectMetric         bool
	InjectOption         bool
	InjectMessage        bool
	InjectRecording      bool
	InjectTelephonyEvent bool
}

func NewDefaultGetConversationOption() *GetConversationOption {
	return &GetConversationOption{
		InjectContext:        true,
		InjectArgument:       true,
		InjectMetadata:       true,
		InjectMetric:         true,
		InjectOption:         true,
		InjectRecording:      false,
		InjectTelephonyEvent: true,
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
		case "telephonyEvent":
			gco.InjectTelephonyEvent = true
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
	InjectRequest  bool
	InjectResponse bool
}

func NewGetMessageOption() *GetMessageOption {
	return &GetMessageOption{}
}

func NewDefaultGetMessageOption() *GetMessageOption {
	return &GetMessageOption{
		InjectMetadata: true,
		InjectMetric:   true,
		InjectRequest:  true,
		InjectResponse: true,
	}
}

func (gco *GetMessageOption) WithFieldSelector(selectors []*workflow_api.FieldSelector) *GetMessageOption {
	for _, v := range selectors {
		switch v.Field {
		case "metadata":
			gco.InjectMetadata = true
		case "metric":
			gco.InjectMetric = true
		case "request":
			gco.InjectRequest = true
		case "response":
			gco.InjectResponse = true

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
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		opts *GetConversationOption,
	) (int64, []*internal_conversation_gorm.AssistantConversation, error)

	// later you will ask why two let me tell you one for end user
	// comming from request adapter
	// anotehr is CRM
	GetConversation(ctx context.Context,
		auth types.SimplePrinciple,
		idenifier string,
		assistantId uint64,
		assistantConversationId uint64,
		opts *GetConversationOption) (*internal_conversation_gorm.AssistantConversation, error)

	Get(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		opts *GetConversationOption) (*internal_conversation_gorm.AssistantConversation, error)

	//
	GetAllConversationMessage(context.Context,
		types.SimplePrinciple,
		uint64,
		[]*workflow_api.Criteria,
		*workflow_api.Paginate,
		*workflow_api.Ordering, *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	GetAllMessageActions(context.Context,
		types.SimplePrinciple,
		uint64,
		[]*workflow_api.Criteria,
		*workflow_api.Paginate,
		*workflow_api.Ordering) (int64, []*internal_conversation_gorm.AssistantConversationAction, error)

	GetAllAssistantMessage(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		ordering *workflow_api.Ordering, opts *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	GetAllMessage(
		ctx context.Context,
		auth types.SimplePrinciple,
		criterias []*workflow_api.Criteria,
		paginate *workflow_api.Paginate,
		ordering *workflow_api.Ordering, opts *GetMessageOption) (int64, []*internal_message_gorm.AssistantConversationMessage, error)

	CreateConversationMetric(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		name, description, value string,
	) (*internal_conversation_gorm.AssistantConversationMetric, error)

	CreateCustomConversationMetric(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		metrics []*workflow_api.Metric,
	) ([]*internal_conversation_gorm.AssistantConversationMetric, error)

	CreateConversationMessage(
		ctx context.Context,
		auth types.SimplePrinciple,
		source utils.RapidaSource,
		assistantConversationMessageId string,
		assistantId uint64,
		assistantProviderModelId uint64,
		assistantConversationId uint64,
		message *types.Message,
	) (*internal_message_gorm.AssistantConversationMessage, error)

	//
	UpdateConversationMessage(ctx context.Context,
		auth types.SimplePrinciple,
		assistantConversationId uint64,
		assistantConversationMessageId string,
		message *types.Message,
		status type_enums.RecordState,
	) (*internal_message_gorm.AssistantConversationMessage, error)

	ApplyMessageMetadata(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantConversationId uint64,
		assistantConversationMessageId string,
		metadata map[string]interface{},
	) ([]*internal_message_gorm.AssistantConversationMessageMetadata, error)

	ApplyMessageMetrics(ctx context.Context,
		auth types.SimplePrinciple,
		assistantConversationId uint64,
		assistantConversationMessageId string,
		metrics []*types.Metric,
	) ([]*internal_message_gorm.AssistantConversationMessageMetric, error)

	//
	CreateConversation(
		ctx context.Context,
		auth types.SimplePrinciple,
		identifier string,
		assistantId uint64,
		assistantProviderModelId uint64,
		direction type_enums.ConversationDirection, source utils.RapidaSource) (*internal_conversation_gorm.AssistantConversation, error)

	CreateLLMAction(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		conversationId uint64,
		assistantConversationMessageId string,
		in, out *types.Message, metrics []*types.Metric) (*internal_conversation_gorm.AssistantConversationAction, error)

	CreateToolAction(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		conversationId uint64,
		assistantConversationMessageId string,
		in, out map[string]interface{},
		metrics []*types.Metric) (
		*internal_conversation_gorm.AssistantConversationAction, error)

	// all about conversation
	ApplyConversationMetadata(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		metadata []*types.Metadata,
	) ([]*internal_conversation_gorm.AssistantConversationMetadata, error)

	ApplyConversationArgument(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		arguments map[string]interface{},
	) ([]*internal_conversation_gorm.AssistantConversationArgument, error)

	ApplyConversationOption(ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		opts map[string]interface{}) ([]*internal_conversation_gorm.AssistantConversationOption, error)

	ApplyConversationMetrics(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		metrics []*types.Metric,
	) ([]*internal_conversation_gorm.AssistantConversationMetric, error)

	CreateConversationRecording(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		assistantConversationId uint64,
		body []byte,
	) (*internal_conversation_gorm.AssistantConversationRecording, error)

	ApplyConversationTelephonyEvent(
		ctx context.Context,
		auth types.SimplePrinciple,
		telephony string,
		assistantId uint64,
		assistantConversationId uint64,
		events []*types.Event,
	) ([]*internal_conversation_gorm.AssistantConversationTelephonyEvent, error)
}
