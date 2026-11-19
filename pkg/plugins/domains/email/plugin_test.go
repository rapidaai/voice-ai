package plugins_email

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	plugins_types "github.com/rapidaai/pkg/plugins/types"
	vault_api "github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type mockVault struct {
	credential *vault_api.VaultCredential
	err        error
}

func (m *mockVault) GetCredential(ctx context.Context, auth authentication.Principal, vaultId uint64) (*vault_api.VaultCredential, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.credential, nil
}

func (m *mockVault) GetOauth2Credential(ctx context.Context, auth authentication.Principal, vaultId uint64) (*vault_api.VaultCredential, error) {
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
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication(t)})
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
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication(t)})
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
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication(t)})
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
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: testAuthentication(t)})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, plugins_types.StatusFail, result.Status)
}

func testAuthentication(t *testing.T) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 1, ProjectID: 1},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "plugin-test"}},
		nil,
	)
	require.NoError(t, err)
	return principal
}
