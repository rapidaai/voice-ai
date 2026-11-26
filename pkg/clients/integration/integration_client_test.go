package integration_client

import (
	"context"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
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
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
	}
	_, err := client.Chat(context.Background(), authentication.Principal{}, "provider", &protos.ChatRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestAuditServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &auditServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
		logger:         logger,
	}
	_, err = client.GetAuditLog(context.Background(), authentication.Principal{}, 1)
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetAuditLog() error = %v", err)
	}
}
