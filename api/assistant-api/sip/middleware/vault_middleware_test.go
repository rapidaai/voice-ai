// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/authentication"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestVaultMiddleware_ResolvesSIPConfig(t *testing.T) {
	projectID := uint64(9)
	organizationID := uint64(10)
	vaultValue, err := structpb.NewStruct(map[string]interface{}{
		"sip_username": "user",
		"sip_password": "pass",
		"sip_server":   "pbx.example.com",
	})
	require.NoError(t, err)

	auth := newVaultTestPrincipal(t, organizationID, projectID)
	assistant := &internal_assistant_entity.Assistant{
		AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
			AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
				TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
					{Metadata: gorm_model.Metadata{Key: "rapida.credential_id", Value: "77"}},
					{Metadata: gorm_model.Metadata{Key: "phone", Value: "+15551234567"}},
				},
			},
		},
	}
	assistant.Id = 42
	vault := &routeTestVault{credential: &protos.VaultCredential{Id: 77, Value: vaultValue}}
	middleware := NewVaultMiddleware(
		WithContext(context.Background()),
		WithLogger(newRouteTestLogger(t)),
		WithRapidaClient(&rapida_client.RapidaClient{Vault: vault}),
		WithSIPConfig(sip_config.NewResolver(&assistant_config.SIPConfig{Port: 5090})),
	)

	ctx := &sip_runtime.SIPRequestContext{CallID: "call-vault", Method: "INVITE"}
	ctx.Auth = auth
	ctx.Assistant = assistant
	err = middleware(ctx)

	require.NoError(t, err)
	require.NotNil(t, ctx.Config)
	assert.Equal(t, uint64(77), vault.requestedVaultID)
	assert.Equal(t, "15551234567", ctx.Config.CallerID)
	assert.Equal(t, 5090, ctx.Config.Port)
	assert.Same(t, vault.credential, ctx.VaultCredential)
}

func TestVaultMiddleware_ReturnsCredentialIDError(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{
		Auth: newVaultTestPrincipal(t, 10, 9),
		Assistant: &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{},
		},
	}
	err := NewVaultMiddleware(WithRapidaClient(&rapida_client.RapidaClient{Vault: &routeTestVault{}}))(ctx)

	require.Error(t, err)
	var sipErr *sip_runtime.SIPError
	require.ErrorAs(t, err, &sipErr)
	assert.Equal(t, sipStatusServerError, sipErr.Code)
	assert.Equal(t, sipMessageConfigurationResolution, sipErr.Message)
	assert.ErrorIs(t, err, sip_config.ErrInvalidConfig)
	assert.ErrorIs(t, err, sip_runtime.ErrCredentialIDRequired)
}

func TestVaultMiddleware_DoesNotMutateContextWhenVaultConfigIsInvalid(t *testing.T) {
	ctx := &sip_runtime.SIPRequestContext{
		Auth: newVaultTestPrincipal(t, 10, 9),
		Assistant: &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
				AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
					TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
						{Metadata: gorm_model.Metadata{Key: "rapida.credential_id", Value: "77"}},
					},
				},
			},
		},
	}
	err := NewVaultMiddleware(WithRapidaClient(&rapida_client.RapidaClient{
		Vault: &routeTestVault{credential: &protos.VaultCredential{Id: 77}},
	}))(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, sip_config.ErrInvalidConfig)
	assert.ErrorIs(t, err, sip_runtime.ErrVaultConfigInvalid)
	assert.Nil(t, ctx.VaultCredential)
	assert.Nil(t, ctx.Config)
}

func newVaultTestPrincipal(t *testing.T, organizationID, projectID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 9007, Issuer: "assistant-api"},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method:   authentication.AuthenticationServiceAssertion,
			Audience: authentication.AudienceRapidaInternal,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceSigningKey,
				Value: "assistant-test-1",
			},
		},
		nil,
	)
	require.NoError(t, err)
	return principal
}

type routeTestVault struct {
	requestedVaultID uint64
	credential       *protos.VaultCredential
	err              error
}

func (v *routeTestVault) GetCredential(_ context.Context, _ authentication.Principal, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, v.err
}

func (v *routeTestVault) GetOauth2Credential(_ context.Context, _ authentication.Principal, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, v.err
}
