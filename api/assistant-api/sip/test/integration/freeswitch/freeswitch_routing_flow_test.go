// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package freeswitch_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_middleware "github.com/rapidaai/api/assistant-api/sip/middleware"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/connectors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	integrationServiceID      = uint64(9007)
	integrationProjectID      = uint64(7001)
	integrationOrganizationID = uint64(8001)
	integrationCredentialID   = uint64(6001)
)

type inboundRouteCase struct {
	name        string
	routeUser   string
	headerValue string
	assistantID uint64
	phone       string
}

type inboundRouteCapture struct {
	requestURI string
	address    sip_runtime.CallAddress
	session    *sip_runtime.Session
}

func TestFreeSWITCHInboundRouteAuthenticationCompleteFlow(t *testing.T) {
	tests := []inboundRouteCase{
		{
			name:        "agent URI",
			routeUser:   "agent-4101",
			headerValue: "agent-uri",
			assistantID: 4101,
			phone:       "+15551234101",
		},
		{
			name:        "DID URI",
			routeUser:   "did-+15551234102",
			headerValue: "did-uri",
			assistantID: 4102,
			phone:       "+15551234102",
		},
		{
			name:        "plain DID URI",
			routeUser:   "+15551234103",
			headerValue: "plain-did-uri",
			assistantID: 4103,
			phone:       "+15551234103",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credentials := loadSIPCredentialConfig(t)
			harness := newFreeSWITCHHarness(t, credentials)
			database := newIntegrationRouteDatabase(t)
			insertIntegrationRoute(t, database, test.assistantID, test.phone)

			assistant := newIntegrationRouteAssistant(test.assistantID, test.phone)
			vaultCredential := newIntegrationVaultCredential(t, harness, credentials)
			vaultClient := &integrationVaultClient{credential: vaultCredential}
			routeContext := make(chan *sip_runtime.SIPRequestContext, 1)
			answered := make(chan inboundRouteCapture, 1)
			remoteBye := make(chan *sip_runtime.Session, 1)

			harness.server.SetMiddlewares([]sip_runtime.Middleware{
				sip_middleware.NewRouteMiddleware(
					sip_middleware.WithContext(context.Background()),
					sip_middleware.WithLogger(harness.logger),
					sip_middleware.WithAssistantService(integrationAssistantService{assistants: map[uint64]*internal_assistant_entity.Assistant{
						test.assistantID: assistant,
					}}),
					sip_middleware.WithServiceID(integrationServiceID),
				),
				sip_middleware.NewVaultMiddleware(
					sip_middleware.WithContext(context.Background()),
					sip_middleware.WithLogger(harness.logger),
					sip_middleware.WithRapidaClient(&rapida_client.RapidaClient{Vault: vaultClient}),
					sip_middleware.WithApplySIPConfigDefaults(func(config *sip_runtime.Config) {
						config.Transport = sip_runtime.TransportUDP
						config.RTPPortRangeStart = harness.config.rtpPortFrom
						config.RTPPortRangeEnd = harness.config.rtpPortTo
						config.InviteTimeout = callSetupTimeout
						config.SessionTimeout = callSetupTimeout
					}),
				),
				func(ctx *sip_runtime.SIPRequestContext) error {
					routeContext <- cloneSIPRequestContext(ctx)
					ctx.CallAddress.From = "+19999999999"
					ctx.CallAddress.FromURI = "sip:mutated-from@example.invalid"
					ctx.CallAddress.ToURI = "sip:mutated-to@example.invalid"
					ctx.CallAddress.Headers["x-rapida-integration-test"] = "mutated"
					ctx.CallAddress.Headers["x-middleware-only"] = "mutated"
					return nil
				},
			})
			harness.server.SetOnInvite(func(session *sip_runtime.Session, requestURI string, address sip_runtime.CallAddress) error {
				answered <- inboundRouteCapture{requestURI: requestURI, address: address, session: session}
				return nil
			})
			harness.server.SetOnBye(func(session *sip_runtime.Session) error {
				remoteBye <- session
				return nil
			})

			callerPhone := "+15551234999"
			callUUID := harness.originateDirectInboundCall(test.routeUser, callerPhone, map[string]string{
				"X-Rapida-Integration-Test": "inbound-ok",
				"X-Rapida-Route":            test.headerValue,
			})
			t.Cleanup(func() {
				_, _ = harness.runFreeSWITCHCommand("uuid_kill " + callUUID)
			})

			resolved := receiveSIPRequestContext(t, routeContext, callSetupTimeout)
			require.Equal(t, test.phone, resolved.CallAddress.To)
			require.Equal(t, test.assistantID, resolved.Assistant.Id)
			require.NotNil(t, resolved.Auth)
			require.Same(t, assistant, resolved.Assistant)
			require.Same(t, vaultCredential, resolved.VaultCredential)
			require.Equal(t, integrationCredentialID, vaultClient.requestedVaultID)

			capture := receiveInboundRouteCapture(t, answered, callSetupTimeout)
			require.Contains(t, capture.requestURI, "sip:"+test.routeUser+"@")
			require.Equal(t, callerPhone, capture.address.From)
			require.Equal(t, test.phone, capture.address.To)
			require.Contains(t, capture.address.FromURI, callerPhone)
			require.Contains(t, capture.address.ToURI, test.routeUser)
			require.Equal(t, "inbound-ok", capture.address.Headers["x-rapida-integration-test"])
			require.Equal(t, test.headerValue, capture.address.Headers["x-rapida-route"])
			require.NotContains(t, capture.address.Headers, "x-middleware-only")

			auth := capture.session.GetAuth()
			require.NotNil(t, auth)
			require.Equal(t, types.ActorIdentity{Type: types.ActorTypeService, ID: integrationServiceID}, auth.Actor())
			projectContext, err := auth.ProjectContext()
			require.NoError(t, err)
			require.Equal(t, integrationProjectID, projectContext.ProjectID)
			require.Equal(t, integrationOrganizationID, projectContext.OrganizationID)
			require.Equal(t, test.assistantID, capture.session.GetAssistant().Id)
			require.Same(t, vaultCredential, capture.session.GetVaultCredential())
			require.Equal(t, credentials.username, capture.session.GetConfig().Username)
			require.Equal(t, credentials.password, capture.session.GetConfig().Password)
			require.Equal(t, strings.TrimPrefix(test.phone, "+"), capture.session.GetConfig().CallerID)

			waitForCallState(t, capture.session, sip_runtime.CallStateConnected, callSetupTimeout)
			harness.hangupFreeSWITCHCall(callUUID)
			remoteByeSession := receiveInboundSession(t, remoteBye, callTeardownTimeout)
			require.Equal(t, capture.session.GetCallID(), remoteByeSession.GetCallID())
			waitForTerminalCallState(t, capture.session, callTeardownTimeout)
		})
	}
}

func cloneSIPRequestContext(ctx *sip_runtime.SIPRequestContext) *sip_runtime.SIPRequestContext {
	clone := *ctx
	clone.CallAddress.Headers = make(map[string]string, len(ctx.CallAddress.Headers))
	for name, value := range ctx.CallAddress.Headers {
		clone.CallAddress.Headers[name] = value
	}
	return &clone
}

func receiveSIPRequestContext(t *testing.T, contexts <-chan *sip_runtime.SIPRequestContext, timeout time.Duration) *sip_runtime.SIPRequestContext {
	t.Helper()
	select {
	case ctx := <-contexts:
		require.NotNil(t, ctx)
		return ctx
	case <-time.After(timeout):
		t.Fatal("timed out waiting for resolved SIP request context")
		return nil
	}
}

func receiveInboundRouteCapture(t *testing.T, captures <-chan inboundRouteCapture, timeout time.Duration) inboundRouteCapture {
	t.Helper()
	select {
	case capture := <-captures:
		require.NotNil(t, capture.session)
		return capture
	case <-time.After(timeout):
		t.Fatal("timed out waiting for inbound route capture")
		return inboundRouteCapture{}
	}
}

type integrationPostgres struct {
	db *gorm.DB
}

func (p integrationPostgres) Connect(context.Context) error                    { return nil }
func (p integrationPostgres) Name() string                                     { return "sip-integration" }
func (p integrationPostgres) IsConnected(context.Context) bool                 { return true }
func (p integrationPostgres) Disconnect(context.Context) error                 { return nil }
func (p integrationPostgres) Query(context.Context, string, interface{}) error { return nil }
func (p integrationPostgres) DB(ctx context.Context) *gorm.DB                  { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = integrationPostgres{}

type integrationAssistantService struct {
	internal_services.AssistantService
	assistants map[uint64]*internal_assistant_entity.Assistant
}

func (s integrationAssistantService) GetAssistantWithPhoneDeploymentById(
	_ context.Context,
	agentId uint64,
) (*internal_assistant_entity.Assistant, error) {
	return s.assistant(agentId)
}

func (s integrationAssistantService) GetAssistantWithPhoneDeploymentByDID(
	_ context.Context,
	did string,
) (*internal_assistant_entity.Assistant, error) {
	for assistantID, assistant := range s.assistants {
		if assistant.AssistantPhoneDeployment == nil {
			continue
		}
		phone, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
		if err == nil && phone == did {
			return s.assistant(assistantID)
		}
	}
	return nil, fmt.Errorf("assistant for DID %s not found", did)
}

func (s integrationAssistantService) Get(
	_ context.Context,
	_ *types.Authentication,
	assistantID uint64,
	_ *uint64,
	_ *internal_services.GetAssistantOption,
) (*internal_assistant_entity.Assistant, error) {
	return s.assistant(assistantID)
}

func (s integrationAssistantService) assistant(assistantID uint64) (*internal_assistant_entity.Assistant, error) {
	assistant, ok := s.assistants[assistantID]
	if !ok {
		return nil, fmt.Errorf("assistant %d not found", assistantID)
	}
	return assistant, nil
}

type integrationVaultClient struct {
	credential       *protos.VaultCredential
	requestedVaultID uint64
}

func (v *integrationVaultClient) GetCredential(_ context.Context, _ *types.Authentication, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, nil
}

func (v *integrationVaultClient) GetOauth2Credential(_ context.Context, _ *types.Authentication, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, nil
}

func newIntegrationRouteDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/sip-integration.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec("CREATE TABLE assistants (id INTEGER PRIMARY KEY, project_id INTEGER, organization_id INTEGER)").Error)
	require.NoError(t, database.Exec("CREATE TABLE assistant_phone_deployments (id INTEGER PRIMARY KEY, assistant_id INTEGER, telephony_provider TEXT, status TEXT)").Error)
	require.NoError(t, database.Exec("CREATE TABLE assistant_deployment_telephony_options (assistant_deployment_telephony_id INTEGER, key TEXT, value TEXT)").Error)
	return database
}

func insertIntegrationRoute(t *testing.T, database *gorm.DB, assistantID uint64, phone string) {
	t.Helper()
	deploymentID := assistantID + 10000
	require.NoError(t, database.Exec(
		"INSERT INTO assistants (id, project_id, organization_id) VALUES (?, ?, ?)",
		assistantID,
		integrationProjectID,
		integrationOrganizationID,
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status) VALUES (?, ?, ?, ?)",
		deploymentID,
		assistantID,
		"sip",
		type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?)",
		deploymentID,
		"phone",
		phone,
	).Error)
}

func newIntegrationRouteAssistant(assistantID uint64, phone string) *internal_assistant_entity.Assistant {
	deploymentID := assistantID + 10000
	return &internal_assistant_entity.Assistant{
		Audited: gorm_model.Audited{Id: assistantID},
		Organizational: gorm_model.Organizational{
			ProjectId:      integrationProjectID,
			OrganizationId: integrationOrganizationID,
		},
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
				AssistantDeployment: internal_assistant_entity.AssistantDeployment{
					Audited:     gorm_model.Audited{Id: deploymentID},
					AssistantId: assistantID,
				},
			},
			AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
				TelephonyProvider: "sip",
				TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
					{Metadata: gorm_model.Metadata{Key: "phone", Value: phone}},
					{Metadata: gorm_model.Metadata{Key: "rapida.credential_id", Value: fmt.Sprint(integrationCredentialID)}},
				},
			},
		},
	}
}

func newIntegrationVaultCredential(t *testing.T, harness *freeSWITCHHarness, credentials sipCredentialConfig) *protos.VaultCredential {
	t.Helper()
	value, err := structpb.NewStruct(map[string]interface{}{
		"sip_server":   harness.config.fsSIPHost,
		"sip_port":     harness.config.fsSIPPort,
		"sip_username": credentials.username,
		"sip_password": credentials.password,
		"sip_realm":    credentials.realm,
		"sip_domain":   credentials.domain,
	})
	require.NoError(t, err)
	return &protos.VaultCredential{Id: integrationCredentialID, Value: value}
}
