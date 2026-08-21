package web_client

import (
	"context"
	"strings"
	"testing"

	"github.com/rapidaai/pkg/types"
)

type vaultClientAuthenticationPrinciple struct{}

func (vaultClientAuthenticationPrinciple) IsAuthenticated() bool   { return true }
func (vaultClientAuthenticationPrinciple) GetCurrentToken() string { return "" }
func (vaultClientAuthenticationPrinciple) Type() types.AuthType    { return types.AuthTypeProject }

func TestVaultServiceClientGetCredentialRequiresOrganization(t *testing.T) {
	client := &vaultServiceClient{}
	_, err := client.GetCredential(context.Background(), vaultClientAuthenticationPrinciple{}, 1)
	if err == nil || !strings.Contains(err.Error(), "requires organization context") {
		t.Fatalf("GetCredential() error = %v", err)
	}
}
