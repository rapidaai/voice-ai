package web_client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type vaultGRPCClientStub struct {
	protos.VaultServiceClient
	response *protos.GetCredentialResponse
}

func (c *vaultGRPCClientStub) GetOauth2Credential(context.Context, *protos.GetCredentialRequest, ...grpc.CallOption) (*protos.GetCredentialResponse, error) {
	return c.response, nil
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

func TestVaultServiceClientGetOauth2CredentialReturnsRemoteError(t *testing.T) {
	projectContext := types.ProjectContext{OrganizationID: 7, ProjectID: 11}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
		ProjectValue:      &projectContext,
	}
	client := &vaultServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Name: "test", ServiceID: 41, Secret: "secret"}, testLogger(t), nil),
		cfg:            &config.AppConfig{Name: "test", ServiceID: 41, Secret: "secret"},
		logger:         testLogger(t),
		vaultClient: &vaultGRPCClientStub{response: &protos.GetCredentialResponse{
			Error: &protos.Error{HumanMessage: "credential rejected"},
		}},
	}

	_, err := client.GetOauth2Credential(context.Background(), auth, 1)
	if err == nil || !strings.Contains(err.Error(), "credential rejected") {
		t.Fatalf("GetOauth2Credential() error = %v", err)
	}
}
