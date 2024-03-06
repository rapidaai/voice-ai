package testing_client

import (
	"context"

	metadata_helper "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	testing_service_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type testingServiceClient struct {
	cfg                  *config.AppConfig
	logger               commons.Logger
	testingServiceClient testing_service_api.TestsuiteServiceClient
}

func NewTestingServiceClientGRPC(config *config.AppConfig, logger commons.Logger) clients.TestingServiceClient {
	logger.Debugf("conntecting to testing service client with %s", config.ExperimentHost)
	conn, err := grpc.Dial(config.ExperimentHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Unable to create connection %v to the provider service", err)
	}
	testingClient := testing_service_api.NewTestsuiteServiceClient(conn)
	return &testingServiceClient{
		cfg:                  config,
		logger:               logger,
		testingServiceClient: testingClient,
	}
}

func (client *testingServiceClient) GetTestSuite(c context.Context, testsuiteId uint64) (*testing_service_api.GetTestSuiteResponse, error) {
	authToken := metadata_helper.ExtractIncoming(c).Get("Authorization")
	authId := metadata_helper.ExtractIncoming(c).Get("X-Auth-Id")
	projectId := metadata_helper.ExtractIncoming(c).Get("X-Auth-P-Id")

	md := metadata.New(map[string]string{"Authorization": authToken, "X-Auth-Id": authId, "X-Auth-P-Id": projectId})
	return client.testingServiceClient.GetTestSuite(metadata.NewOutgoingContext(c, md), &testing_service_api.GetTestSuiteRequest{TestsuiteId: testsuiteId})
}
