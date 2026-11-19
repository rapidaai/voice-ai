package workflow_client

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

func TestAssistantServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &assistantServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
	}
	_, err := client.GetAssistant(context.Background(), authentication.Principal{}, &protos.GetAssistantRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetAssistant() error = %v", err)
	}
}

func TestKnowledgeServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &knowledgeServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
		logger:         logger,
	}
	_, err = client.GetKnowledge(context.Background(), authentication.Principal{}, &protos.GetKnowledgeRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetKnowledge() error = %v", err)
	}
}

func TestObservabilityServiceClientStopsBeforeGRPCOnAuthError(t *testing.T) {
	client := &observabilityServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", nil, nil),
	}
	_, err := client.GetAllTelemetry(context.Background(), authentication.Principal{}, &protos.GetAllTelemetryRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("GetAllTelemetry() error = %v", err)
	}
}
