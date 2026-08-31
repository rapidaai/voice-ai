// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRouteMiddleware_AgentRoute(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 42, 7, 8).Error)

	ctx := &sip_infra.SIPRequestContext{
		CallID:       "call-agent",
		RequestURI:   "sip:agent-42;transport=tcp@sip.rapida.ai",
		FromIdentity: "sip:caller@example.com",
		ToIdentity:   "sip:assistant@example.com",
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

	ctx := &sip_infra.SIPRequestContext{CallID: "call-plain", RequestURI: "sip:+15551234568@sip.rapida.ai"}
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
	ctx := &sip_infra.SIPRequestContext{CallID: "call-missing-did", RequestURI: "sip:did-+15551239999@sip.rapida.ai"}
	err := NewRouteMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithPostgres(routeTestPostgres{db: newRouteTestDB(t)}),
	)(ctx)

	require.Error(t, err)
	var sipErr *sip_infra.SIPError
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
			logger := &routeLogRecorder{}
			assistantCalls := 0
			ctx := &sip_infra.SIPRequestContext{
				CallID:     "call-duplicate-did",
				RequestURI: "sip:did-+15551234567@sip.rapida.ai",
				CallAddress: sip_infra.CallAddress{
					FromURI: "sip:+15550001111@carrier.example.com",
					ToURI:   "sip:did-+15551234567@sip.rapida.ai",
				},
			}

			err := NewRouteMiddleware(
				WithContext(context.Background()),
				WithLogger(logger),
				WithServiceID(9007),
				WithPostgres(routeTestPostgres{db: db}),
				WithAssistantService(routeTestAssistantService{getCalls: &assistantCalls}),
			)(ctx)

			require.Error(t, err)
			var sipErr *sip_infra.SIPError
			require.ErrorAs(t, err, &sipErr)
			assert.Equal(t, 500, sipErr.Code)
			assert.ErrorIs(t, sipErr.Err, sip_infra.ErrInvalidConfig)
			assert.Empty(t, ctx.AssistantID)
			assert.Nil(t, ctx.Auth)
			assert.Nil(t, ctx.Assistant)
			assert.Zero(t, assistantCalls)
			assert.Equal(t, map[string]interface{}{
				"call_id":      "call-duplicate-did",
				"route_kind":   "did",
				"phone_source": "did_route",
				"phone_result": "ambiguous",
			}, logger.lastFields())
			assert.NotContains(t, logger.String(), "+15551234567")
			assert.NotContains(t, logger.String(), "43")
			assert.NotContains(t, logger.String(), "44")
			assert.NotContains(t, logger.String(), "sip:")
		})
	}
}

func TestRouteMiddleware_InactiveDuplicateDIDDoesNotCreateAmbiguity(t *testing.T) {
	db := newRouteTestDB(t)
	insertRouteTestAssistant(t, db, 43, 9, 10)
	insertRouteTestAssistant(t, db, 44, 19, 20)
	insertRouteTestDeployment(t, db, 100, 43, type_enums.RECORD_ACTIVE, "+15551234567")
	insertRouteTestDeployment(t, db, 101, 44, type_enums.RECORD_INACTIVE, "+15551234567")
	ctx := &sip_infra.SIPRequestContext{CallID: "call-inactive-duplicate", RequestURI: "sip:did-+15551234567@sip.rapida.ai"}

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
		name         string
		deployments  []routeTestDeployment
		expectedTo   string
		expectedCode int
	}{
		{name: "missing deployment"},
		{name: "valid phone", deployments: []routeTestDeployment{{id: 100, status: type_enums.RECORD_ACTIVE, phones: []string{"+15551234567"}}}, expectedTo: "+15551234567"},
		{name: "invalid phone", deployments: []routeTestDeployment{{id: 100, status: type_enums.RECORD_ACTIVE, phones: []string{"agent-42"}}}, expectedCode: 500},
		{name: "ambiguous deployments", deployments: []routeTestDeployment{{id: 100, status: type_enums.RECORD_ACTIVE}, {id: 101, status: type_enums.RECORD_ACTIVE}}, expectedCode: 500},
		{name: "ambiguous phone options", deployments: []routeTestDeployment{{id: 100, status: type_enums.RECORD_ACTIVE, phones: []string{"+15551234567", "+15557654321"}}}, expectedCode: 500},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newRouteTestDB(t)
			insertRouteTestAssistant(t, db, 42, 7, 8)
			for _, deployment := range test.deployments {
				insertRouteTestDeployment(t, db, deployment.id, 42, deployment.status, deployment.phones...)
			}
			ctx := &sip_infra.SIPRequestContext{CallID: "call-agent-phone", RequestURI: "sip:agent-42@sip.rapida.ai"}

			err := NewRouteMiddleware(
				WithContext(context.Background()),
				WithLogger(newRouteTestLogger(t)),
				WithServiceID(9007),
				WithPostgres(routeTestPostgres{db: db}),
				WithAssistantService(routeTestAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
					42: newRouteTestAssistant(7),
				}}),
			)(ctx)

			if test.expectedCode != 0 {
				require.Error(t, err)
				var sipErr *sip_infra.SIPError
				require.ErrorAs(t, err, &sipErr)
				assert.Equal(t, test.expectedCode, sipErr.Code)
				assert.Nil(t, ctx.Auth)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectedTo, ctx.CallAddress.To)
		})
	}
}

func TestRouteMiddleware_RejectsMissingServiceID(t *testing.T) {
	db := newRouteTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)", 47, 17, 18).Error)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-missing-actor", RequestURI: "sip:agent-47@sip.rapida.ai"}
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
	var sipErr *sip_infra.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, 500, sipErr.Code)
	assert.ErrorIs(t, sipErr.Err, types.ErrServiceActorUnavailable)
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

type routeLogEntry struct {
	message string
	fields  map[string]interface{}
}

type routeLogRecorder struct {
	mu      sync.Mutex
	entries []routeLogEntry
}

func (l *routeLogRecorder) record(message string, args ...interface{}) {
	fields := make(map[string]interface{}, len(args)/2)
	for index := 0; index+1 < len(args); index += 2 {
		key, ok := args[index].(string)
		if ok {
			fields[key] = args[index+1]
		}
	}
	l.mu.Lock()
	l.entries = append(l.entries, routeLogEntry{message: message, fields: fields})
	l.mu.Unlock()
}

func (l *routeLogRecorder) lastFields() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) == 0 {
		return nil
	}
	return l.entries[len(l.entries)-1].fields
}

func (l *routeLogRecorder) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fmt.Sprint(l.entries)
}

func (*routeLogRecorder) Level() zapcore.Level                           { return zapcore.InfoLevel }
func (*routeLogRecorder) Debug(...interface{})                           {}
func (*routeLogRecorder) Debugf(string, ...interface{})                  {}
func (*routeLogRecorder) Debugw(string, ...interface{})                  {}
func (*routeLogRecorder) Info(...interface{})                            {}
func (*routeLogRecorder) Infof(string, ...interface{})                   {}
func (l *routeLogRecorder) Infow(message string, args ...interface{})    { l.record(message, args...) }
func (*routeLogRecorder) Warn(...interface{})                            {}
func (*routeLogRecorder) Warnf(string, ...interface{})                   {}
func (l *routeLogRecorder) Warnw(message string, args ...interface{})    { l.record(message, args...) }
func (*routeLogRecorder) Error(...interface{})                           {}
func (*routeLogRecorder) Errorf(string, ...interface{})                  {}
func (l *routeLogRecorder) Errorw(message string, args ...interface{})   { l.record(message, args...) }
func (*routeLogRecorder) DPanic(...interface{})                          {}
func (*routeLogRecorder) DPanicf(string, ...interface{})                 {}
func (*routeLogRecorder) Panic(...interface{})                           {}
func (*routeLogRecorder) Panicf(string, ...interface{})                  {}
func (*routeLogRecorder) Fatal(...interface{})                           {}
func (*routeLogRecorder) Fatalf(string, ...interface{})                  {}
func (*routeLogRecorder) Benchmark(string, time.Duration)                {}
func (*routeLogRecorder) Tracef(context.Context, string, ...interface{}) {}
func (*routeLogRecorder) Sync() error                                    { return nil }

var _ commons.Logger = (*routeLogRecorder)(nil)

func (entry routeLogEntry) String() string {
	var builder strings.Builder
	builder.WriteString(entry.message)
	for key, value := range entry.fields {
		fmt.Fprintf(&builder, " %s=%v", key, value)
	}
	return builder.String()
}
