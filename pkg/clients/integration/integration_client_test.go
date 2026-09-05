package integration_client

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type unifiedProviderGRPCClientStub struct {
	protos.UnifiedProviderServiceClient
}

func TestNewIntegrationServiceClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &unifiedProviderGRPCClientStub{}
	client := NewIntegrationServiceClientWithClient(&config.AppConfig{}, nil, nil, grpcClient)

	if client.(*integrationServiceClient).unifiedClient != grpcClient {
		t.Fatal("NewIntegrationServiceClientWithClient() did not retain the provided gRPC client")
	}
}

func TestIntegrationServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &integrationServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
	}
	_, err := client.Chat(context.Background(), &types.Authentication{}, "provider", &protos.ChatRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestAuditServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &auditServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
		logger:         logger,
	}
	_, err = client.GetAuditLog(context.Background(), &types.Authentication{}, 1)
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetAuditLog() error = %v", err)
	}
}
