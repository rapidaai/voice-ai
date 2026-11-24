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

type assistantGRPCClientStub struct {
	protos.AssistantServiceClient
}

type assistantDeploymentGRPCClientStub struct {
	protos.AssistantDeploymentServiceClient
}

type knowledgeGRPCClientStub struct {
	protos.KnowledgeServiceClient
}

type observabilityGRPCClientStub struct {
	protos.ObservabilityServiceClient
}

func TestInjectedWorkflowClientsRetainProvidedGRPCClients(t *testing.T) {
	assistant := &assistantGRPCClientStub{}
	assistantDeployment := &assistantDeploymentGRPCClientStub{}
	knowledge := &knowledgeGRPCClientStub{}
	observability := &observabilityGRPCClientStub{}

	assistantClient := NewAssistantServiceClientWithClients(&config.AppConfig{}, nil, nil, assistant, assistantDeployment)
	knowledgeClient := NewKnowledgeServiceClientWithClient(&config.AppConfig{}, nil, nil, knowledge)
	observabilityClient := NewObservabilityServiceClientWithClient(&config.AppConfig{}, nil, nil, observability)

	if client := assistantClient.(*assistantServiceClient); client.assistantClient != assistant || client.assistantDeploymentClient != assistantDeployment {
		t.Fatal("NewAssistantServiceClientWithClients() did not retain the provided gRPC clients")
	}
	if knowledgeClient.(*knowledgeServiceClient).knowledgeClient != knowledge {
		t.Fatal("NewKnowledgeServiceClientWithClient() did not retain the provided gRPC client")
	}
	if observabilityClient.(*observabilityServiceClient).observabilityClient != observability {
		t.Fatal("NewObservabilityServiceClientWithClient() did not retain the provided gRPC client")
	}
}

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
