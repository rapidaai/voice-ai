package document_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type IndexerServiceClient interface {
	IndexKnowledgeDocument(ctx context.Context, auth types.SimplePrinciple,
		in *protos.IndexKnowledgeDocumentRequest) (*protos.IndexKnowledgeDocumentResponse, error)
}

type indexerServiceClient struct {
	clients.InternalClient
	cfg    *config.AppConfig
	logger commons.Logger
	client *http.Client
}

// NewSendgridServiceClientHTTP creates a new Sendgrid service client for HTTP.
func NewIndexerServiceClient(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) IndexerServiceClient {
	logger.Debugf("connecting to integration service via HTTP at %s", config.IntegrationHost)
	return &indexerServiceClient{
		InternalClient: clients.NewInternalClient(config, logger, redis),
		cfg:            config,
		logger:         logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (client *indexerServiceClient) IndexKnowledgeDocument(ctx context.Context, auth types.SimplePrinciple, in *protos.IndexKnowledgeDocumentRequest) (*protos.IndexKnowledgeDocumentResponse, error) {
	reqBody, err := json.Marshal(in)
	if err != nil {
		client.logger.Errorf("unable to marshal request body: %v", err)
		return nil, err
	}

	client.logger.Debugf("sending request to index knowledge document with payload %+v", string(reqBody))
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/v1/knowledge/index/document/", client.cfg.DocumentHost), bytes.NewBuffer(reqBody))
	if err != nil {
		client.logger.Errorf("unable to create HTTP request: %v", err)
		return nil, err
	}

	resp, err := client.client.Do(client.WithHttpAuth(ctx, auth, req))
	if err != nil {
		client.logger.Errorf("unable to send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var res protos.IndexKnowledgeDocumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		client.logger.Errorf("unable to decode response: %v", err)
		return nil, err
	}

	return &res, nil

}
