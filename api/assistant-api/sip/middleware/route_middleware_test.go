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
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
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

	ctx := &sip_infra.SIPRequestContext{
		CallID:       "call-agent",
		RequestURI:   "sip:agent-42@sip.rapida.ai",
		FromIdentity: "sip:caller@example.com",
		ToIdentity:   "sip:assistant@example.com",
	}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
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
	projectContext, authErr := ctx.Auth.ProjectContext()
	require.NoError(t, authErr)
	assert.Equal(t, uint64(7), projectContext.ProjectID)
	assert.Equal(t, uint64(8), projectContext.OrganizationID)
	assert.Equal(t, "sip:caller@example.com", ctx.FromIdentity)
	assert.Equal(t, "sip:assistant@example.com", ctx.ToIdentity)
}

func TestRouteMiddleware_DIDRoute(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 43, 9, 10).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)", 100, 43, "sip", type_enums.RECORD_ACTIVE.String()).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)", 100, "phone", "+15551234567").Error)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-did", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			43: newRouteTestAssistant(9),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "43", ctx.AssistantID)
	assert.NotNil(t, ctx.Auth)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_PlainDIDRoute(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 44, 11, 12).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)", 101, 44, "sip", type_enums.RECORD_ACTIVE.String()).Error)
	require.NoError(t, db.Exec("INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)", 101, "phone", "+15551234568").Error)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-plain", RequestURI: "sip:+15551234568@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			44: newRouteTestAssistant(11),
		}}),
	)
	err := middleware(ctx)

	require.NoError(t, err)
	assert.Equal(t, "44", ctx.AssistantID)
	assert.NotNil(t, ctx.Assistant)
}

func TestRouteMiddleware_DoesNotRouteFromIdentity(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 45, 13, 14).Error)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-from", FromIdentity: "sip:agent-45@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			45: newRouteTestAssistant(13),
		}}),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_infra.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
	assert.Empty(t, ctx.AssistantID)
	assert.Nil(t, ctx.Assistant)
}

func TestRouteMiddleware_DoesNotRouteToIdentity(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 46, 15, 16).Error)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-to", ToIdentity: "sip:agent-46@sip.rapida.ai"}
	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
		WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
			46: newRouteTestAssistant(15),
		}}),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_infra.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 404, sipErr.Code)
	assert.Empty(t, ctx.AssistantID)
	assert.Nil(t, ctx.Assistant)
}

func TestRouteMiddleware_RejectsCredentialPair(t *testing.T) {
	db := newRouteTestDB(t)
	ctx := &sip_infra.SIPRequestContext{CallID: "call-invalid", RequestURI: "sip:12345:apikey@sip.rapida.ai"}

	middleware := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: db}),
	)
	err := middleware(ctx)

	require.Error(t, err)
	var sipErr *sip_infra.SIPError
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
}

func (s routeTestAssistantService) Get(_ context.Context, _ *types.Authentication, assistantID uint64, _ *uint64, _ *internal_services.GetAssistantOption) (*internal_assistant_entity.Assistant, error) {
	assistant, ok := s.assistants[assistantID]
	if !ok {
		return nil, fmt.Errorf("assistant %d not found", assistantID)
	}
	assistant.Id = assistantID
	return assistant, nil
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
