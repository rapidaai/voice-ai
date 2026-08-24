package document_client

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

func TestIndexerServiceClientStopsBeforeHTTPOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &indexerServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, logger, nil),
		cfg:            &config.AppConfig{},
		logger:         logger,
	}
	_, err = client.IndexKnowledgeDocument(context.Background(), &types.Authentication{}, &protos.IndexKnowledgeDocumentRequest{})
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
}
