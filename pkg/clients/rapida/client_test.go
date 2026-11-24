package rapida_client

import (
	"context"
	"testing"

	"github.com/rapidaai/config"
	web_client "github.com/rapidaai/pkg/clients/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func TestNewCreatesSeparateConnections(t *testing.T) {
	client, err := New(testConfig(), nil, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	if client.Authentication == nil || client.Vault == nil || client.Project == nil || client.ProductUsage == nil {
		t.Fatal("web clients were not initialized")
	}
	var authenticationClient web_client.AuthenticationClient = client.Authentication
	if authenticationClient == nil {
		t.Fatal("unified authentication client was not initialized")
	}
	if client.Integration == nil || client.Audit == nil || client.Deployment == nil || client.Endpoint == nil || client.Indexer == nil {
		t.Fatal("service clients were not initialized")
	}
	if client.Assistant == nil || client.Knowledge == nil || client.Observability == nil {
		t.Fatal("workflow clients were not initialized")
	}
	if len(client.connections) != 7 {
		t.Fatalf("connection count = %d, want 7", len(client.connections))
	}
	seen := make(map[*grpc.ClientConn]struct{}, len(client.connections))
	for _, connection := range client.connections {
		seen[connection] = struct{}{}
	}
	if len(seen) != len(client.connections) {
		t.Fatal("clients unexpectedly share a gRPC connection")
	}
}

func TestCloseClosesEveryConnectionOnce(t *testing.T) {
	client, err := New(testConfig(), nil, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	for index, connection := range client.connections {
		if connection.GetState() != connectivity.Shutdown {
			t.Fatalf("connection %d state = %s, want SHUTDOWN", index, connection.GetState())
		}
	}
}

func testConfig() *config.AppConfig {
	return &config.AppConfig{
		Web:         config.ServiceHostConfig{Host: "passthrough:///web-api"},
		Integration: config.ServiceHostConfig{Host: "passthrough:///integration-api"},
		Endpoint:    config.ServiceHostConfig{Host: "passthrough:///endpoint-api"},
		Document:    config.ServiceHostConfig{Host: "http://document-api"},
	}
}
