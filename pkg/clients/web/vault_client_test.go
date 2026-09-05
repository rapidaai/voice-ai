package web_client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type vaultGRPCClientStub struct {
	protos.VaultServiceClient
}

func TestNewVaultClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &vaultGRPCClientStub{}
	client := NewVaultClientWithClient(&config.AppConfig{}, nil, nil, grpcClient)

	if client.(*vaultServiceClient).vaultClient != grpcClient {
		t.Fatal("NewVaultClientWithClient() did not retain the provided gRPC client")
	}
}

func TestVaultServiceClientGetCredentialRequiresOrganization(t *testing.T) {
	client := &vaultServiceClient{}
	_, err := client.GetCredential(context.Background(), &types.Authentication{}, 1)
	if err == nil || !strings.Contains(err.Error(), "requires organization context") {
		t.Fatalf("GetCredential() error = %v", err)
	}
}

func TestVaultServiceClientGetOauth2CredentialStopsOnAuthError(t *testing.T) {
	client := &vaultServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
	}
	_, err := client.GetOauth2Credential(context.Background(), &types.Authentication{}, 1)
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetOauth2Credential() error = %v", err)
	}
}
