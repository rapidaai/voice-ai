// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
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

	auth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
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
		WithVaultClient(vault),
		WithApplySIPConfigDefaults(func(config *sip_infra.Config) {
			config.Port = 5090
		}),
	)

	ctx := &sip_infra.SIPRequestContext{CallID: "call-vault", Method: "INVITE"}
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

type routeTestVault struct {
	requestedVaultID uint64
	credential       *protos.VaultCredential
}

func (v *routeTestVault) GetCredential(_ context.Context, _ *types.Authentication, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, nil
}

func (v *routeTestVault) GetOauth2Credential(_ context.Context, _ *types.Authentication, vaultID uint64) (*protos.VaultCredential, error) {
	v.requestedVaultID = vaultID
	return v.credential, nil
}
