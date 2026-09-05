// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"errors"
	"fmt"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteMiddleware_AgentRoute(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{
		CallID:     "call-agent",
		RequestURI: "sip:agent-42;transport=tcp@sip.rapida.ai",
		CallAddress: sip_runtime.CallAddress{
			FromURI: "sip:caller@example.com",
			ToURI:   "sip:assistant@example.com",
		},
	}
	route, routeErr := ctx.ResolveRoute()
	require.NoError(t, routeErr)
	require.IsType(t, sip_runtime.AgentCallRoute{}, route)
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			42: newRouteTestAssistantWithPhone(7, 8, "+15551234567"),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.NotNil(t, ctx.Auth)
	assert.NotNil(t, ctx.Assistant)
	assert.Equal(t, types.AuthTypeService, ctx.Auth.Type())
	actor := ctx.Auth.Actor()
	assert.Equal(t, types.ActorIdentity{Type: types.ActorTypeService, ID: 9007}, actor)
	projectContext, authErr := ctx.Auth.ProjectContext()
	require.NoError(t, authErr)
	assert.Equal(t, uint64(7), projectContext.ProjectID)
	assert.Equal(t, uint64(8), projectContext.OrganizationID)
	assert.Equal(t, "sip:caller@example.com", ctx.CallAddress.FromURI)
	assert.Equal(t, "sip:assistant@example.com", ctx.CallAddress.ToURI)
}

func TestRouteMiddleware_DIDRoute(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-did", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}
	route, routeErr := ctx.ResolveRoute()
	require.NoError(t, routeErr)
	require.IsType(t, sip_runtime.DIDCallRoute{}, route)
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithAssistantService(routeTestAssistantService{
			assistants: map[uint64]*internal_assistant_entity.Assistant{
				43: newRouteTestAssistantWithPhone(9, 10, "+15551234567"),
			},
			didAssistantIDs: map[string][]uint64{"+15551234567": {43}},
		}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "+15551234567", ctx.CallAddress.To)
	assert.NotNil(t, ctx.Auth)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_PlainDIDRoute(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-plain", RequestURI: "sip:+15551234568@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithAssistantService(routeTestAssistantService{
			assistants: map[uint64]*internal_assistant_entity.Assistant{
				44: newRouteTestAssistantWithPhone(11, 12, "+15551234568"),
			},
			didAssistantIDs: map[string][]uint64{"+15551234568": {44}},
		}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "+15551234568", ctx.CallAddress.To)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_DIDRouteNotFound(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-missing-did", RequestURI: "sip:did-+15551239999@sip.rapida.ai"}
	err := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithAssistantService(routeTestAssistantService{}),
	)(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
	assert.Empty(t, ctx.CallAddress.To)
	assert.Nil(t, ctx.Auth)
}

func TestRouteMiddleware_DuplicateDIDRoutesUseFirstResolvedAssistant(t *testing.T) {
	tests := []struct {
		name          string
		organization2 uint64
	}{
		{name: "same tenant", organization2: 10},
		{name: "different tenants", organization2: 20},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assistantCalls := 0
			ctx := &sip_runtime.SIPRequestContext{
				CallID:     "call-duplicate-did",
				RequestURI: "sip:did-+15551234567@sip.rapida.ai",
				CallAddress: sip_runtime.CallAddress{
					FromURI: "sip:+15550001111@carrier.example.com",
					ToURI:   "sip:did-+15551234567@sip.rapida.ai",
				},
			}

			err := NewRouteMiddleware(
				WithContext(context.Background()),
				WithLogger(newRouteTestLogger(t)),
				WithServiceID(9007),
				WithAssistantService(routeTestAssistantService{
					assistants: map[uint64]*internal_assistant_entity.Assistant{
						43: newRouteTestAssistantWithPhone(9, 10, "+15551234567"),
						44: newRouteTestAssistantWithPhone(19, test.organization2, "+15551234567"),
					},
					didAssistantIDs: map[string][]uint64{"+15551234567": {43, 44}},
					getCalls:        &assistantCalls,
				}),
			)(ctx)

			require.NoError(t, err)
			require.NotNil(t, ctx.Auth)
			require.NotNil(t, ctx.Assistant)
			assert.Equal(t, uint64(43), ctx.Assistant.Id)
			assert.Equal(t, uint64(10), ctx.Auth.OrganizationValue.OrganizationID)
			assert.Equal(t, 1, assistantCalls)
		})
	}
}

func TestRouteMiddleware_InactiveDuplicateDIDDoesNotCreateAmbiguity(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-inactive-duplicate", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}

	err := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithAssistantService(routeTestAssistantService{
			assistants: map[uint64]*internal_assistant_entity.Assistant{
				43: newRouteTestAssistantWithPhone(9, 10, "+15551234567"),
			},
			didAssistantIDs: map[string][]uint64{"+15551234567": {43}},
		}),
	)(ctx)

	require.NoError(t, err)
	assert.Equal(t, "+15551234567", ctx.CallAddress.To)
}

func TestRouteMiddleware_AgentRoutePhoneResolution(t *testing.T) {
	tests := []struct {
		name          string
		hasDeployment bool
		phone         string
		expectedTo    string
		wantError     bool
	}{
		{name: "missing deployment", wantError: true},
		{name: "missing phone", hasDeployment: true, wantError: true},
		{name: "valid phone", hasDeployment: true, phone: "+15551234567", expectedTo: "+15551234567"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assistant := newRouteTestAssistant(7, 8)
			if test.hasDeployment {
				assistant.AssistantPhoneDeployment = &internal_assistant_entity.AssistantPhoneDeployment{
					AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
						TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
							{Metadata: gorm_model.Metadata{Key: "phone", Value: test.phone}},
						},
					},
				}
			}
			ctx := &sip_runtime.SIPRequestContext{CallID: "call-agent-phone", RequestURI: "sip:agent-42@sip.rapida.ai"}

			err := NewRouteMiddleware(
				WithContext(context.Background()),
				WithLogger(newRouteTestLogger(t)),
				WithServiceID(9007),
				WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
					42: assistant,
				}}),
			)(ctx)

			if test.wantError {
				require.Error(t, err)
				var sipErr *sip_runtime.SIPError
				require.ErrorAs(t, err, &sipErr)
				assert.Equal(t, 500, sipErr.Code)
				assert.ErrorIs(t, sipErr.Err, sip_runtime.ErrInvalidConfig)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectedTo, ctx.CallAddress.To)
		})
	}
}

func TestRouteMiddleware_RejectsCredentialPair(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-invalid", RequestURI: "sip:12345:apikey@sip.rapida.ai"}

	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
	assert.ErrorIs(t, sipErr.Err, sip_runtime.ErrInvalidCallRoute)
}

type routeTestAssistantService struct {
	internal_services.AssistantService
	assistants      map[uint64]*internal_assistant_entity.Assistant
	didAssistantIDs map[string][]uint64
	getCalls        *int
}

func (s routeTestAssistantService) GetAssistantWithPhoneDeploymentByDID(_ context.Context, did string) (*internal_assistant_entity.Assistant, error) {
	assistantIDs := s.didAssistantIDs[did]
	if len(assistantIDs) == 0 {
		return nil, errors.New("assistant not found")
	}
	return s.assistant(assistantIDs[0])
}

func (s routeTestAssistantService) GetAssistantWithPhoneDeploymentById(_ context.Context, agentId uint64) (*internal_assistant_entity.Assistant, error) {
	return s.assistant(agentId)
}

func (s routeTestAssistantService) Get(_ context.Context, _ *types.Authentication, assistantID uint64, _ *uint64, _ *internal_services.GetAssistantOption) (*internal_assistant_entity.Assistant, error) {
	return s.assistant(assistantID)
}

func (s routeTestAssistantService) assistant(assistantID uint64) (*internal_assistant_entity.Assistant, error) {
	if s.getCalls != nil {
		(*s.getCalls)++
	}
	assistant, ok := s.assistants[assistantID]
	if !ok {
		return nil, fmt.Errorf("assistant %d not found", assistantID)
	}
	assistant.Id = assistantID
	return assistant, nil
}

func newRouteTestAssistant(projectID uint64, organizationID uint64) *internal_assistant_entity.Assistant {
	assistant := &internal_assistant_entity.Assistant{}
	assistant.ProjectId = projectID
	assistant.OrganizationId = organizationID
	return assistant
}

func newRouteTestAssistantWithPhone(projectID uint64, organizationID uint64, phone string) *internal_assistant_entity.Assistant {
	assistant := newRouteTestAssistant(projectID, organizationID)
	assistant.AssistantPhoneDeployment = &internal_assistant_entity.AssistantPhoneDeployment{
		AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
			TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
				{Metadata: gorm_model.Metadata{Key: "phone", Value: phone}},
			},
		},
	}
	return assistant
}

func newRouteTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
		commons.Level("error"),
	)
	require.NoError(t, err)
	return logger
}
