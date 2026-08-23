package plugins_email

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

func TestPlugin_Validate(t *testing.T) {
	p := NewPlugin()
	err := p.Validate(map[string]interface{}{"provider": "noop", "credential_id": float64(10)})
	require.NoError(t, err)
}

func TestPlugin_ValidateMissingFields(t *testing.T) {
	p := NewPlugin()
	err := p.Validate(map[string]interface{}{"provider": "noop"})
	require.Error(t, err)
}

func TestPlugin_ExecuteSuccess(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{"api_key": "x"})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_email",
		Provider:  "noop",
		Input:     map[string]interface{}{"to": "x@y.com"},
		Config:    map[string]interface{}{"provider": "noop", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, plugins_types.StatusSuccess, result.Status)
	assert.Equal(t, "noop", result.Provider)
}

func TestPlugin_ExecuteVaultFail(t *testing.T) {
	p := NewPlugin()
	vault := &mockVault{err: errors.New("vault down")}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_email",
		Provider:  "noop",
		Input:     map[string]interface{}{"to": "x@y.com"},
		Config:    map[string]interface{}{"provider": "noop", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, plugins_types.StatusFail, result.Status)
	assert.Contains(t, result.Error, "vault")
}

func TestPlugin_ExecuteSendgridSuccess(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{"api_key": "sg-key"})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_email",
		Provider:  "sendgrid",
		Input: map[string]interface{}{
			"to":      "user@example.com",
			"subject": "hello",
			"text":    "world",
		},
		Config: map[string]interface{}{"provider": "sendgrid", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, plugins_types.StatusSuccess, result.Status)
	assert.Equal(t, "sendgrid", result.Provider)
}

func TestPlugin_ExecuteSendgridMissingInput(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{"api_key": "sg-key"})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "send_email",
		Provider:  "sendgrid",
		Input: map[string]interface{}{
			"to": "user@example.com",
		},
		Config: map[string]interface{}{"provider": "sendgrid", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication()})
	require.NoError(t, err)
	require.NotNil(t, result)
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
