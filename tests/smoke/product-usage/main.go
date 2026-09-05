package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connection, err := grpc.NewClient("web-api:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("connect to web-api: %w", err))
	}
	defer connection.Close()

	client := protos.NewProductUsageServiceClient(connection)
	requests := []struct {
		name     string
		metadata metadata.MD
	}{
		{
			name: "personal access token",
			metadata: metadata.Pairs(
				types.AUTHORIZATION_KEY, os.Getenv("CI_STACK_AUTH_TOKEN"),
				types.AUTH_KEY, "1",
				types.PROJECT_KEY, "1",
			),
		},
		{
			name:     "project API key",
			metadata: metadata.Pairs(types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+os.Getenv("CI_STACK_PROJECT_API_KEY")),
		},
	}

	for _, request := range requests {
		response, callErr := client.CreateProductUsage(metadata.NewOutgoingContext(ctx, request.metadata), &protos.CreateProductUsageRequest{
			UsageType:  string(types.ProductUsageLLMDuration),
			Usages:     int64(time.Second),
			Unit:       string(types.ProductUsageUnitNanosecond),
			OccurredAt: timestamppb.Now(),
		})
		if callErr != nil {
			panic(fmt.Errorf("create product usage with %s: %w", request.name, callErr))
		}
		if !response.GetSuccess() || response.GetData().GetId() == 0 {
			panic(fmt.Errorf("create product usage with %s returned success=%t id=%d", request.name, response.GetSuccess(), response.GetData().GetId()))
		}
		fmt.Printf("%s product usage authentication passed\n", request.name)
	}
}
