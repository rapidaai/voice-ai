// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"github.com/gin-gonic/gin"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

type Pipeline interface {
	CallID() string
	Validate() bool
}

type CallReceivedPipeline struct {
	ID          string
	Provider    string
	Auth        *types.Authentication
	AssistantID uint64
	GinContext  *gin.Context
	Observer    observability.Recorder
}

func (p CallReceivedPipeline) CallID() string { return p.ID }
func (p CallReceivedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.Provider) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID) &&
		validator.NonNil(p.GinContext) &&
		validator.NonNil(p.Observer)
}

type WebhookParsedPipeline struct {
	ID          string
	Provider    string
	Auth        *types.Authentication
	AssistantID uint64
	CallInfo    *internal_type.CallInfo
	GinContext  *gin.Context
}

func (p WebhookParsedPipeline) CallID() string { return p.ID }
func (p WebhookParsedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.Provider) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID) &&
		validator.NonNil(p.CallInfo) &&
		validator.NonNil(p.GinContext)
}

type AssistantResolvedPipeline struct {
	ID          string
	Provider    string
	Auth        *types.Authentication
	AssistantID uint64
	Assistant   *internal_assistant_entity.Assistant
	CallInfo    *internal_type.CallInfo
	GinContext  *gin.Context
}

func (p AssistantResolvedPipeline) CallID() string { return p.ID }
func (p AssistantResolvedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.Provider) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID) &&
		validator.NonNil(p.Assistant) &&
		validator.NonNil(p.CallInfo) &&
		validator.NonNil(p.GinContext)
}

type ConversationCreatedPipeline struct {
	ID             string
	Provider       string
	Auth           *types.Authentication
	AssistantID    uint64
	Assistant      *internal_assistant_entity.Assistant
	ConversationID uint64
	ContextID      string
	CallInfo       *internal_type.CallInfo
	GinContext     *gin.Context
}

func (p ConversationCreatedPipeline) CallID() string { return p.ID }
func (p ConversationCreatedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.Provider) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID, p.ConversationID) &&
		validator.NotBlank(p.ContextID) &&
		validator.NonNil(p.Assistant) &&
		validator.NonNil(p.CallInfo) &&
		validator.NonNil(p.GinContext)
}

type ProviderAnsweringPipeline struct {
	ID             string
	Provider       string
	Auth           *types.Authentication
	AssistantID    uint64
	ConversationID uint64
	ContextID      string
	CallerNumber   string
	GinContext     *gin.Context
}

func (p ProviderAnsweringPipeline) CallID() string { return p.ID }
func (p ProviderAnsweringPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.Provider) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID, p.ConversationID) &&
		validator.NotBlank(p.ContextID) &&
		validator.NotBlank(p.CallerNumber) &&
		validator.NonNil(p.GinContext)
}

type ProviderAnsweredPipeline struct {
	ID        string
	ContextID string
}

func (p ProviderAnsweredPipeline) CallID() string { return p.ID }
func (p ProviderAnsweredPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.ContextID)
}

type SessionConnectedPipeline struct {
	ID          string
	ContextID   string
	CallContext *callcontext.CallContext
	Talker      internal_type.Talking
	Observer    observability.Recorder
}

func (p SessionConnectedPipeline) CallID() string { return p.ID }
func (p SessionConnectedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NotBlank(p.ContextID) &&
		validator.NonNil(p.CallContext) &&
		validator.NonNil(p.Talker) &&
		validator.NonNil(p.Observer)
}

type SessionInitializedPipeline struct {
	ID   string
	Auth *types.Authentication
}

func (p SessionInitializedPipeline) CallID() string { return p.ID }
func (p SessionInitializedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NonNil(p.Auth)
}

type CallActivePipeline struct {
	ID string
}

func (p CallActivePipeline) CallID() string { return p.ID }
func (p CallActivePipeline) Validate() bool {
	return validator.NotBlank(p.ID)
}

type OutboundRequestedPipeline struct {
	ID          string
	Auth        *types.Authentication
	AssistantID uint64
	Version     string
	ToPhone     string
	FromPhone   string
	Metadata    map[string]interface{}
	Args        map[string]interface{}
	Options     map[string]interface{}
	Observer    observability.Recorder
}

func (p OutboundRequestedPipeline) CallID() string { return p.ID }
func (p OutboundRequestedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NonNil(p.Auth) &&
		validator.AllNonZero(p.AssistantID) &&
		validator.NotBlank(p.ToPhone) &&
		validator.NonNil(p.Observer)
}

type OutboundDialedPipeline struct {
	ID       string
	CallInfo *internal_type.CallInfo
}

func (p OutboundDialedPipeline) CallID() string { return p.ID }
func (p OutboundDialedPipeline) Validate() bool {
	return validator.NotBlank(p.ID) &&
		validator.NonNil(p.CallInfo)
}
