// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"strings"

	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

type middlewareOption struct {
	ctx                    context.Context
	logger                 commons.Logger
	assistantService       internal_services.AssistantService
	rapidaClient           *rapida_client.RapidaClient
	applySIPConfigDefaults func(*sip_runtime.Config)
	ServiceID              uint64
}

func WithContext(ctx context.Context) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.logger = logger
	}
}

func WithAssistantService(assistantService internal_services.AssistantService) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.assistantService = assistantService
	}
}

func WithRapidaClient(rapidaClient *rapida_client.RapidaClient) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.rapidaClient = rapidaClient
	}
}

func WithApplySIPConfigDefaults(applySIPConfigDefaults func(*sip_runtime.Config)) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.applySIPConfigDefaults = applySIPConfigDefaults
	}
}

func WithServiceID(ServiceID uint64) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.ServiceID = ServiceID
	}
}

func NewRouteMiddleware(options ...func(*middlewareOption)) sip_runtime.Middleware {
	m := &middlewareOption{ctx: context.Background()}
	for _, option := range options {
		if validator.NonNil(option) {
			option(m)
		}
	}
	return func(ctx *sip_runtime.SIPRequestContext) error {
		route, err := ctx.ResolveRoute()
		if err != nil {
			return &sip_runtime.SIPError{Code: 404, Message: "Invalid SIP route", Err: err}
		}

		switch resolvedRoute := route.(type) {
		case sip_runtime.AgentCallRoute:
			return m.resolveAgentCallRoute(ctx, resolvedRoute)
		case sip_runtime.DIDCallRoute:
			return m.resolveDIDCallRoute(ctx, resolvedRoute)
		default:
			return &sip_runtime.SIPError{Code: 404, Message: "Unsupported SIP route", Err: sip_runtime.ErrAuthRequired}
		}
	}
}

func (m *middlewareOption) resolveAgentCallRoute(ctx *sip_runtime.SIPRequestContext, route sip_runtime.AgentCallRoute) error {
	if !validator.NonNil(m.assistantService) {
		return &sip_runtime.SIPError{Code: 500, Message: "SIP assistant resolver not configured", Err: sip_runtime.ErrInvalidConfig}
	}

	assistant, err := m.assistantService.GetAssistantWithPhoneDeploymentById(m.ctx, route.AssistantID)
	if err != nil {
		return &sip_runtime.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_runtime.ErrAuthRequired}
	}
	if !validator.AllNonZero(assistant.Id, assistant.ProjectId, assistant.OrganizationId) {
		return &sip_runtime.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_runtime.ErrAuthRequired}
	}

	ctx.Auth = &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: m.ServiceID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: assistant.OrganizationId},
		ProjectValue:      &types.ProjectContext{OrganizationID: assistant.OrganizationId, ProjectID: assistant.ProjectId},
	}
	if !validator.NonNil(assistant.AssistantPhoneDeployment) {
		return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_runtime.ErrInvalidConfig}
	}
	phone, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
	if err != nil || !validator.NotBlank(phone) {
		return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_runtime.ErrInvalidConfig}
	}
	ctx.CallAddress.To = strings.TrimSpace(phone)
	ctx.Assistant = assistant

	return nil
}

func (m *middlewareOption) resolveDIDCallRoute(ctx *sip_runtime.SIPRequestContext, route sip_runtime.DIDCallRoute) error {
	if !validator.NonNil(m.assistantService) {
		return &sip_runtime.SIPError{Code: 500, Message: "SIP assistant resolver not configured", Err: sip_runtime.ErrInvalidConfig}
	}

	assistant, err := m.assistantService.GetAssistantWithPhoneDeploymentByDID(m.ctx, route.DID)
	if err != nil {
		return &sip_runtime.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_runtime.ErrAuthRequired}
	}
	if !validator.AllNonZero(assistant.Id, assistant.ProjectId, assistant.OrganizationId) {
		return &sip_runtime.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_runtime.ErrAuthRequired}
	}
	if !validator.NonNil(assistant.AssistantPhoneDeployment) {
		return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_runtime.ErrInvalidConfig}
	}
	phone, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
	if err != nil {
		return &sip_runtime.SIPError{Code: 500, Message: "Failed to resolve SIP configuration", Err: sip_runtime.ErrInvalidConfig}
	}
	ctx.CallAddress.To = strings.TrimSpace(phone)
	ctx.Auth = &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: m.ServiceID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: assistant.OrganizationId},
		ProjectValue:      &types.ProjectContext{OrganizationID: assistant.OrganizationId, ProjectID: assistant.ProjectId},
	}
	ctx.Assistant = assistant
	return nil
}
