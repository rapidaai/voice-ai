package web_client

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/rapidaai/config"
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
	request    *protos.CreateProductUsageRequest
	response   *protos.GetProductUsageResponse
	err        error
	contextKey any
	hasAuth    bool
}

type productUsageConnectionStub struct {
	err    error
	closed bool
}

func (connection *productUsageConnectionStub) Close() error {
	connection.closed = true
	return connection.err
}

func (client *productUsageGRPCClientStub) CreateProductUsage(ctx context.Context, request *protos.CreateProductUsageRequest, _ ...grpc.CallOption) (*protos.GetProductUsageResponse, error) {
	client.request = request
	client.hasAuth, _ = ctx.Value(client.contextKey).(bool)
	return client.response, client.err
}

func (client *productUsageGRPCClientStub) GetProductUsages(context.Context, *protos.GetProductUsagesRequest, ...grpc.CallOption) (*protos.GetUsagesResponse, error) {
	return nil, errors.New("not implemented")
}

func (client *productUsageGRPCClientStub) GetOrganizationUsages(context.Context, *protos.GetOrganizationUsagesRequest, ...grpc.CallOption) (*protos.GetUsagesResponse, error) {
	return nil, errors.New("not implemented")
}

func TestProductUsageClientCreatesAuthenticatedRequest(t *testing.T) {
	contextKey := struct{}{}
	internalClient := &productUsageInternalClientStub{contextKey: contextKey}
	grpcClient := &productUsageGRPCClientStub{
		contextKey: contextKey,
		response:   &protos.GetProductUsageResponse{Success: true, Data: &protos.ProductUsage{Id: 42}},
	}
	client := &productUsageServiceClient{
		InternalClient:     internalClient,
		logger:             testLogger(t),
		productUsageClient: grpcClient,
	}
	auth := &types.Authentication{}
	request := &protos.CreateProductUsageRequest{UsageType: "stt_duration", Usages: 10, Unit: "nanosecond"}
	response, err := client.CreateProductUsage(context.Background(), auth, request)
	if err != nil {
		t.Fatalf("CreateProductUsage() error = %v", err)
	}
	if response.GetData().GetId() != 42 {
		t.Fatalf("CreateProductUsage() response ID = %d", response.GetData().GetId())
	}
	if internalClient.auth != auth {
		t.Fatal("CreateProductUsage() did not forward authentication")
	}
	if !grpcClient.hasAuth {
		t.Fatal("CreateProductUsage() did not use authenticated context")
	}
	if grpcClient.request != request {
		t.Fatalf("CreateProductUsage() request = %+v", grpcClient.request)
	}
}

func TestNewProductUsageClientWithClientUsesProvidedGRPCClient(t *testing.T) {
	grpcClient := &productUsageGRPCClientStub{}
	client := NewProductUsageClientWithClient(&config.AppConfig{}, nil, grpcClient)

	if client.(*productUsageServiceClient).productUsageClient != grpcClient {
		t.Fatal("NewProductUsageClientWithClient() did not retain the provided gRPC client")
	}
}

func TestProductUsageClientReturnsGRPCError(t *testing.T) {
	expectedErr := errors.New("unavailable")
	client := &productUsageServiceClient{
		InternalClient:     &productUsageInternalClientStub{},
		logger:             testLogger(t),
		productUsageClient: &productUsageGRPCClientStub{err: expectedErr},
	}

	_, err := client.CreateProductUsage(context.Background(), &types.Authentication{}, nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("CreateProductUsage() error = %v", err)
	}
}

func TestProductUsageClientReturnsServiceResponse(t *testing.T) {
	want := &protos.GetProductUsageResponse{Error: &protos.Error{HumanMessage: "usage rejected"}}
	client := &productUsageServiceClient{
		InternalClient:     &productUsageInternalClientStub{},
		logger:             testLogger(t),
		productUsageClient: &productUsageGRPCClientStub{response: want},
	}

	got, err := client.CreateProductUsage(context.Background(), &types.Authentication{}, nil)
	if err != nil {
		t.Fatalf("CreateProductUsage() error = %v", err)
	}
	if got != want {
		t.Fatalf("CreateProductUsage() response = %v, want %v", got, want)
	}
}

func TestProductUsageClientBoundsRequestDuration(t *testing.T) {
	contextKey := struct{}{}
	client := &productUsageServiceClient{
		InternalClient: &productUsageInternalClientStub{contextKey: contextKey},
		logger:         testLogger(t),
		requestTimeout: time.Millisecond,
		productUsageClient: productUsageGRPCClientFunc(func(ctx context.Context, _ *protos.CreateProductUsageRequest, _ ...grpc.CallOption) (*protos.GetProductUsageResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	}

	_, err := client.CreateProductUsage(context.Background(), &types.Authentication{}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CreateProductUsage() error = %v, want deadline exceeded", err)
	}
}

func TestProductUsageClientClosesConnection(t *testing.T) {
	expectedErr := errors.New("close failed")
	connection := &productUsageConnectionStub{err: expectedErr}
	client := &productUsageServiceClient{connection: connection}

	err := client.Close()
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Close() error = %v, want %v", err, expectedErr)
	}
	if !connection.closed {
		t.Fatal("Close() did not close the connection")
	}
}

type productUsageGRPCClientFunc func(context.Context, *protos.CreateProductUsageRequest, ...grpc.CallOption) (*protos.GetProductUsageResponse, error)

func (call productUsageGRPCClientFunc) CreateProductUsage(ctx context.Context, request *protos.CreateProductUsageRequest, options ...grpc.CallOption) (*protos.GetProductUsageResponse, error) {
	return call(ctx, request, options...)
}

func (productUsageGRPCClientFunc) GetProductUsages(context.Context, *protos.GetProductUsagesRequest, ...grpc.CallOption) (*protos.GetUsagesResponse, error) {
	return nil, errors.New("not implemented")
}

func (productUsageGRPCClientFunc) GetOrganizationUsages(context.Context, *protos.GetOrganizationUsagesRequest, ...grpc.CallOption) (*protos.GetUsagesResponse, error) {
	return nil, errors.New("not implemented")
}
