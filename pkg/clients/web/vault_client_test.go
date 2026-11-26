package web_client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/clients"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
)

type vaultGRPCClientStub struct {
	protos.VaultServiceClient
	response *protos.GetCredentialResponse
}

type vaultInternalClientStub struct {
	clients.InternalClient
}

func (vaultInternalClientStub) WithAuth(ctx context.Context, principal authentication.Principal) (context.Context, error) {
	if principal.Tenant().OrganizationID == 0 {
		return nil, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return ctx, nil
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
	_, err := client.GetCredential(context.Background(), authentication.Principal{}, 1)
	var invalid pkg_errors.AuthenticationPrincipalInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("GetCredential() error = %v", err)
	}
}

func TestVaultServiceClientGetOauth2CredentialStopsOnAuthError(t *testing.T) {
	client := &vaultServiceClient{
		InternalClient: vaultInternalClientStub{},
	}
	_, err := client.GetOauth2Credential(context.Background(), authentication.Principal{}, 1)
	var invalid pkg_errors.AuthenticationPrincipalInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("GetOauth2Credential() error = %v", err)
	}
}

func TestVaultServiceClientGetOauth2CredentialReturnsRemoteError(t *testing.T) {
	principal, principalErr := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 11},
		authentication.Tenant{OrganizationID: 7, ProjectID: 11},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "11"}},
		nil,
	)
	if principalErr != nil {
		t.Fatal(principalErr)
	}
	client := &vaultServiceClient{
		InternalClient: vaultInternalClientStub{},
		cfg:            &config.AppConfig{Name: "test", ServiceID: 41, Secret: "secret"},
		logger:         testLogger(t),
		vaultClient: &vaultGRPCClientStub{response: &protos.GetCredentialResponse{
			Error: &protos.Error{HumanMessage: "credential rejected"},
		}},
	}

	_, err := client.GetOauth2Credential(context.Background(), principal, 1)
	if err == nil || !strings.Contains(err.Error(), "credential rejected") {
		t.Fatalf("GetOauth2Credential() error = %v", err)
	}
}
