package workflow_client

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

func TestAssistantServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &assistantServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
	}
	_, err := client.GetAssistant(context.Background(), &types.Authentication{}, &protos.GetAssistantRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetAssistant() error = %v", err)
	}
}

func TestKnowledgeServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &knowledgeServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
		logger:         logger,
	}
	_, err = client.GetKnowledge(context.Background(), &types.Authentication{}, &protos.GetKnowledgeRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetKnowledge() error = %v", err)
	}
}

func TestObservabilityServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &observabilityServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, nil, nil),
	}
	_, err := client.GetAllTelemetry(context.Background(), &types.Authentication{}, &protos.GetAllTelemetryRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("GetAllTelemetry() error = %v", err)
	}
}
