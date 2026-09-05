// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"fmt"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRouteMiddleware_AgentRoute(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 42, 7, 8).Error)

	ctx := &sip_runtime.SIPRequestContext{
		CallID:     "call-agent",
		RequestURI: "sip:agent-42;transport=tcp@sip.rapida.ai",
		CallAddress: sip_runtime.CallAddress{
			FromURI: "sip:caller@example.com",
			ToURI:   "sip:assistant@example.com",
		},
	}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			42: newRouteTestAssistant(7),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "42", ctx.AssistantID)
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
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 43, 9, 10).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)", 100, 43, "sip", type_enums.RECORD_ACTIVE.String()).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)", 100, "phone", "+15551234567").Error)

	ctx := &sip_runtime.SIPRequestContext{CallID: "call-did", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			43: newRouteTestAssistant(9),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "43", ctx.AssistantID)
	assert.Equal(t, "+15551234567", ctx.CallAddress.To)
	assert.NotNil(t, ctx.Auth)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_PlainDIDRoute(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 44, 11, 12).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)", 101, 44, "sip", type_enums.RECORD_ACTIVE.String()).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)", 101, "phone", "+15551234568").Error)

	ctx := &sip_runtime.SIPRequestContext{CallID: "call-plain", RequestURI: "sip:+15551234568@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			44: newRouteTestAssistant(11),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "44", ctx.AssistantID)
	assert.Equal(t, "+15551234568", ctx.CallAddress.To)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_DIDRouteNotFound(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-missing-did", RequestURI: "sip:did-+15551239999@sip.rapida.ai"}
	err := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: newRouteTestDB(t)}),
	)(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
	assert.Empty(t, ctx.CallAddress.To)
	assert.Nil(t, ctx.Auth)
}

func TestRouteMiddleware_DuplicateDIDRoutesFailBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name          string
		organization2 uint64
	}{
		{name: "same tenant", organization2: 10},
		{name: "different tenants", organization2: 20},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newRouteTestDB(t)
			insertRouteTestAssistant(t, db, 43, 9, 10)
			insertRouteTestAssistant(t, db, 44, 19, test.organization2)
			insertRouteTestDeployment(t, db, 100, 43, type_enums.RECORD_ACTIVE, "+15551234567")
			insertRouteTestDeployment(t, db, 101, 44, type_enums.RECORD_ACTIVE, "+15551234567")
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
				WithPostgres(routeTestPostgres{db: db}),
				WithAssistantService(routeTestAssistantService{getCalls: &assistantCalls}),
			)(ctx)

			require.Error(t, err)
			var sipErr *sip_runtime.SIPError
			require.ErrorAs(t, err, &sipErr)
			assert.Equal(t, 500, sipErr.Code)
			assert.ErrorIs(t, sipErr.Err, sip_runtime.ErrInvalidConfig)
			assert.Empty(t, ctx.AssistantID)
			assert.Nil(t, ctx.Auth)
			assert.Nil(t, ctx.Assistant)
			assert.Zero(t, assistantCalls)
		})
	}
}

func TestRouteMiddleware_InactiveDuplicateDIDDoesNotCreateAmbiguity(t *testing.T) {
	db := newRouteTestDB(t)
	insertRouteTestAssistant(t, db, 43, 9, 10)
	insertRouteTestAssistant(t, db, 44, 19, 20)
	insertRouteTestDeployment(t, db, 100, 43, type_enums.RECORD_ACTIVE, "+15551234567")
	insertRouteTestDeployment(t, db, 101, 44, type_enums.RECORD_INACTIVE, "+15551234567")
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-inactive-duplicate", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}

	err := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithServiceID(9007),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			43: newRouteTestAssistant(9),
		}}),
	)(ctx)

	require.NoError(t, err)
	assert.Equal(t, "43", ctx.AssistantID)
	assert.Equal(t, "+15551234567", ctx.CallAddress.To)
}

func TestRouteMiddleware_AgentRoutePhoneResolution(t *testing.T) {
	tests := []struct {
		name       string
		phone      string
		expectedTo string
	}{
		{name: "missing deployment"},
		{name: "valid phone", phone: "+15551234567", expectedTo: "+15551234567"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newRouteTestDB(t)
			insertRouteTestAssistant(t, db, 42, 7, 8)
			assistant := newRouteTestAssistant(7)
			if test.phone != "" {
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
				WithPostgres(routeTestPostgres{db: db}),
				WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
					42: assistant,
				}}),
			)(ctx)

			require.NoError(t, err)
			assert.Equal(t, test.expectedTo, ctx.CallAddress.To)
		})
	}
}

func TestRouteMiddleware_RejectsMissingServiceID(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 47, 17, 18).Error)

	ctx := &sip_runtime.SIPRequestContext{CallID: "call-missing-actor", RequestURI: "sip:agent-47@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			47: newRouteTestAssistant(17),
		}}),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 500, sipErr.Code)
	assert.ErrorIs(t, sipErr.Err, types.ErrServiceActorUnavailable)
}

func TestRouteMiddleware_RejectsCredentialPair(t *testing.T) {
	db := newRouteTestDB(t)
	ctx := &sip_runtime.SIPRequestContext{CallID: "call-invalid", RequestURI: "sip:12345:apikey@sip.rapida.ai"}

	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
}

type routeTestPostgres struct {
	db *gorm.DB
}

func (p routeTestPostgres) Connect(_ context.Context) error    { return nil }
func (p routeTestPostgres) Name() string                       { return "route-test" }
func (p routeTestPostgres) IsConnected(_ context.Context) bool { return true }
func (p routeTestPostgres) Disconnect(_ context.Context) error { return nil }
func (p routeTestPostgres) Query(_ context.Context, _ string, _ interface{}) error {
	return nil
}
func (p routeTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

type routeTestAssistantService struct {
	internal_services.AssistantService
	assistants map[uint64]*internal_assistant_entity.Assistant
	getCalls   *int
}

func (s routeTestAssistantService) Get(_ context.Context, _ *types.Authentication, assistantID uint64, _ *uint64, _ *internal_services.GetAssistantOption) (*internal_assistant_entity.Assistant, error) {
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

type routeTestDeployment struct {
	id     uint64
	status type_enums.RecordState
	phones []string
}

func insertRouteTestAssistant(t *testing.T, db *gorm.DB, assistantID, projectID, organizationID uint64) {
	t.Helper()
	require.NoError(t, db.Exec(
		"INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)",
		assistantID,
		projectID,
		organizationID,
	).Error)
}

func insertRouteTestDeployment(t *testing.T, db *gorm.DB, deploymentID, assistantID uint64, status type_enums.RecordState, phones ...string) {
	t.Helper()
	require.NoError(t, db.Exec(
		"INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)",
		deploymentID,
		assistantID,
		"sip",
		status.String(),
	).Error)
	for _, phone := range phones {
		require.NoError(t, db.Exec(
			"INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)",
			deploymentID,
			"phone",
			phone,
		).Error)
	}
}

func newRouteTestAssistant(projectID uint64) *internal_assistant_entity.Assistant {
	assistant := &internal_assistant_entity.Assistant{}
	assistant.ProjectId = projectID
	return assistant
}

func newRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/sip-route.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE assistants (id INTEGER PRIMARY KEY, project_id INTEGER, organization_id INTEGER)").Error)
	require.NoError(t, db.Exec("CREATE TABLE assistant_phone_deployments (id INTEGER PRIMARY KEY, assistant_id INTEGER, telephony_provider TEXT, status TEXT)").Error)
	require.NoError(t, db.Exec("CREATE TABLE assistant_deployment_telephony_options (assistant_deployment_telephony_id INTEGER, key TEXT, value TEXT)").Error)
	return db
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
