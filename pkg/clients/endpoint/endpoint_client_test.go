package endpoint_client

import (
	"context"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	endpoint_api "github.com/rapidaai/protos"
)

type deploymentGRPCClientStub struct {
	endpoint_api.DeploymentClient
}

func TestNewDeploymentServiceClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &deploymentGRPCClientStub{}
	client := NewDeploymentServiceClientWithClient(&config.AppConfig{}, nil, nil, grpcClient)

	if client.(*deploymentServiceClient).deploymentClient != grpcClient {
		t.Fatal("NewDeploymentServiceClientWithClient() did not retain the provided gRPC client")
	}
}

func TestEndpointServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &endpointServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
		logger:         logger,
	}
	_, err = client.GetEndpoint(context.Background(), authentication.Principal{}, &endpoint_api.GetEndpointRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetEndpoint() error = %v", err)
	}
}

func TestDeploymentServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &deploymentServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
	}
	_, err := client.Invoke(context.Background(), authentication.Principal{}, &endpoint_api.InvokeRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("Invoke() error = %v", err)
	}
}
