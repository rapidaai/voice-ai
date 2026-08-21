package plugins_sms

import (
	"context"
	"errors"
	"testing"

	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	plugins_types "github.com/rapidaai/pkg/plugins/types"
	"github.com/rapidaai/pkg/types"
	vault_api "github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type mockVault struct {
	credential *vault_api.VaultCredential
	err        error
}

func (m *mockVault) GetCredential(ctx context.Context, auth *types.Authentication, vaultId uint64) (*vault_api.VaultCredential, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.credential, nil
}

func (m *mockVault) GetOauth2Credential(ctx context.Context, auth *types.Authentication, vaultId uint64) (*vault_api.VaultCredential, error) {
	return nil, errors.New("not used")
}

var _ web_client.VaultClient = (*mockVault)(nil)

func testLogger(t *testing.T) commons.Logger {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	require.NoError(t, err)
	return logger
}

func TestPlugin_ExecuteTwilioSuccess(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{
		"account_sid": "AC123",
		"auth_token":  "token",
		"from_number": "+15550000000",
	})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_sms",
		Provider:  "twilio",
		Input:     map[string]interface{}{"to": "+15551111111", "body": "hello"},
		Config:    map[string]interface{}{"provider": "twilio", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	assert.Equal(t, plugins_types.StatusSuccess, result.Status)
	assert.Equal(t, "twilio", result.Provider)
}

func TestPlugin_ExecuteTwilioMissingInput(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{
		"account_sid": "AC123",
		"auth_token":  "token",
		"from_number": "+15550000000",
	})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_sms",
		Provider:  "twilio",
		Input:     map[string]interface{}{"to": "+15551111111"},
		Config:    map[string]interface{}{"provider": "twilio", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	assert.Equal(t, plugins_types.StatusFail, result.Status)
}

func testAuthentication() *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: 1},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 1},
		ProjectValue:      &types.ProjectContext{OrganizationID: 1, ProjectID: 1},
	}
}
