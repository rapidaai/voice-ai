package document_client

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

func TestIndexerServiceClientStopsBeforeHTTPOnAuthError(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	client := &indexerServiceClient{
		InternalClient: clients.NewInternalClient(&config.AppConfig{Secret: "secret"}, "test-api", logger, nil),
		cfg:            &config.AppConfig{},
		logger:         logger,
	}
	_, err = client.IndexKnowledgeDocument(context.Background(), authentication.Principal{}, &protos.IndexKnowledgeDocumentRequest{})
	if _, ok := err.(pkg_errors.AuthenticationPrincipalInvalidError); !ok {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
}
