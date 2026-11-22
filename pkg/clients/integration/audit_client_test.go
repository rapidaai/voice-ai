package integration_client

import (
	"context"
	"math"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

type auditGRPCClientStub struct {
	protos.AuditLoggingServiceClient
	getOptions    []grpc.CallOption
	getAllOptions []grpc.CallOption
}

func (stub *auditGRPCClientStub) GetAuditLog(_ context.Context, _ *protos.GetAuditLogRequest, options ...grpc.CallOption) (*protos.GetAuditLogResponse, error) {
	stub.getOptions = options
	return &protos.GetAuditLogResponse{}, nil
}

func (stub *auditGRPCClientStub) GetAllAuditLog(_ context.Context, _ *protos.GetAllAuditLogRequest, options ...grpc.CallOption) (*protos.GetAllAuditLogResponse, error) {
	stub.getAllOptions = options
	return &protos.GetAllAuditLogResponse{}, nil
}

func TestNewAuditServiceClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &auditGRPCClientStub{}
	client := NewAuditServiceClientWithClient(&config.AppConfig{}, nil, nil, grpcClient)

	if client.(*auditServiceClient).auditLoggingClient != grpcClient {
		t.Fatal("NewAuditServiceClientWithClient() did not retain the provided gRPC client")
	}
}

func TestAuditServiceClientUsesUnlimitedMessageSizesPerCall(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	grpcClient := &auditGRPCClientStub{}
	client := NewAuditServiceClientWithClient(&config.AppConfig{Secret: "test-secret"}, logger, nil, grpcClient)
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 2, ProjectID: 3},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type: authentication.ProofReferenceUserSession, Value: "session-1",
			},
		},
		[]authentication.PermissionKey{"project.audit.read"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.GetAuditLog(context.Background(), principal, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAllAuditLog(context.Background(), principal, &protos.GetAllAuditLogRequest{}); err != nil {
		t.Fatal(err)
	}
	assertUnlimitedMessageSizes(t, grpcClient.getOptions)
	assertUnlimitedMessageSizes(t, grpcClient.getAllOptions)
}

func assertUnlimitedMessageSizes(t *testing.T, options []grpc.CallOption) {
	t.Helper()
	var hasReceiveLimit bool
	var hasSendLimit bool
	for _, option := range options {
		switch option := option.(type) {
		case grpc.MaxRecvMsgSizeCallOption:
			hasReceiveLimit = option.MaxRecvMsgSize == math.MaxInt64
		case grpc.MaxSendMsgSizeCallOption:
			hasSendLimit = option.MaxSendMsgSize == math.MaxInt64
		}
	}
	if !hasReceiveLimit || !hasSendLimit {
		t.Fatalf("unlimited receive = %v, unlimited send = %v", hasReceiveLimit, hasSendLimit)
	}
}
