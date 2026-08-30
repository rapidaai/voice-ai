package web_client

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type productUsageInternalClientStub struct {
	clients.InternalClient
	auth       *types.Authentication
	contextKey any
}

func (client *productUsageInternalClientStub) WithAuth(ctx context.Context, auth *types.Authentication) (context.Context, error) {
	client.auth = auth
	if client.contextKey == nil {
		return ctx, nil
	}
	return context.WithValue(ctx, client.contextKey, true), nil
}

type productUsageGRPCClientStub struct {
	request     *protos.CreateProductUsagesRequest
	response    *protos.CreateProductUsagesResponse
	err         error
	contextKey  any
	hasAuth     bool
	deadline    time.Time
	hasDeadline bool
}

func (client *productUsageGRPCClientStub) CreateProductUsages(ctx context.Context, request *protos.CreateProductUsagesRequest, _ ...grpc.CallOption) (*protos.CreateProductUsagesResponse, error) {
	client.request = request
	client.hasAuth, _ = ctx.Value(client.contextKey).(bool)
	client.deadline, client.hasDeadline = ctx.Deadline()
	return client.response, client.err
}

type productUsageCloserStub struct {
	closed bool
}

func (closer *productUsageCloserStub) Close() error {
	closer.closed = true
	return nil
}

func TestProductUsageClientCreatesAuthenticatedRequestWithDeadline(t *testing.T) {
	contextKey := struct{}{}
	internalClient := &productUsageInternalClientStub{contextKey: contextKey}
	grpcClient := &productUsageGRPCClientStub{
		contextKey: contextKey,
		response:   &protos.CreateProductUsagesResponse{Success: true, CreatedCount: 1},
	}
	client := &productUsageServiceClient{
		InternalClient: internalClient,
		client:         grpcClient,
		connection:     io.NopCloser(nilReader{}),
		timeout:        productUsageRequestTimeout,
	}
	auth := &types.Authentication{}
	usage := &protos.ProductUsage{UsageId: "usage-1", UsageType: "stt_duration", Usages: 10, Unit: "nanosecond"}
	startedAt := time.Now()

	response, err := client.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{usage})
	if err != nil {
		t.Fatalf("CreateProductUsages() error = %v", err)
	}
	if response.GetCreatedCount() != 1 {
		t.Fatalf("CreateProductUsages() created count = %d", response.GetCreatedCount())
	}
	if internalClient.auth != auth {
		t.Fatal("CreateProductUsages() did not forward authentication")
	}
	if !grpcClient.hasAuth {
		t.Fatal("CreateProductUsages() did not use authenticated context")
	}
	if !grpcClient.hasDeadline {
		t.Fatal("CreateProductUsages() did not set a deadline")
	}
	deadlineWindow := grpcClient.deadline.Sub(startedAt)
	if deadlineWindow < productUsageRequestTimeout-time.Second || deadlineWindow > productUsageRequestTimeout+time.Second {
		t.Fatalf("CreateProductUsages() deadline window = %v", deadlineWindow)
	}
	if len(grpcClient.request.GetUsages()) != 1 || grpcClient.request.GetUsages()[0] != usage {
		t.Fatalf("CreateProductUsages() request = %+v", grpcClient.request)
	}
}

func TestProductUsageClientReturnsGRPCError(t *testing.T) {
	expectedErr := errors.New("unavailable")
	client := &productUsageServiceClient{
		InternalClient: &productUsageInternalClientStub{},
		client:         &productUsageGRPCClientStub{err: expectedErr},
		timeout:        productUsageRequestTimeout,
	}

	_, err := client.CreateProductUsages(context.Background(), &types.Authentication{}, nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("CreateProductUsages() error = %v", err)
	}
}

func TestProductUsageClientRejectsUnsuccessfulResponse(t *testing.T) {
	client := &productUsageServiceClient{
		InternalClient: &productUsageInternalClientStub{},
		client: &productUsageGRPCClientStub{response: &protos.CreateProductUsagesResponse{
			Error: &protos.Error{HumanMessage: "usage rejected"},
		}},
		timeout: productUsageRequestTimeout,
	}

	_, err := client.CreateProductUsages(context.Background(), &types.Authentication{}, nil)
	if err == nil || err.Error() != "create product usages: usage rejected" {
		t.Fatalf("CreateProductUsages() error = %v", err)
	}
}

func TestProductUsageClientClose(t *testing.T) {
	closer := &productUsageCloserStub{}
	client := &productUsageServiceClient{connection: closer}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !closer.closed {
		t.Fatal("Close() did not close connection")
	}
}

type nilReader struct{}

func (nilReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
