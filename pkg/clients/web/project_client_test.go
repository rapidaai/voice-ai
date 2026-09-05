package web_client

import (
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/protos"
)

type projectGRPCClientStub struct {
	protos.ProjectServiceClient
}

func TestNewProjectClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &projectGRPCClientStub{}
	client := NewProjectClientWithClient(&config.AppConfig{}, nil, nil, grpcClient)

	if client.(*projectServiceClient).projectClient != grpcClient {
		t.Fatal("NewProjectClientWithClient() did not retain the provided gRPC client")
	}
}
