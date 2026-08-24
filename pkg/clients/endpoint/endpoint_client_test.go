package endpoint_client

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	endpoint_api "github.com/rapidaai/protos"
)

func TestEndpointServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &endpointServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
		logger:         logger,
	}
	_, err = client.GetEndpoint(context.Background(), &types.Authentication{}, &endpoint_api.GetEndpointRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetEndpoint() error = %v", err)
	}
}

func TestDeploymentServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &deploymentServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
	}
	_, err := client.Invoke(context.Background(), &types.Authentication{}, &endpoint_api.InvokeRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("Invoke() error = %v", err)
	}
}
