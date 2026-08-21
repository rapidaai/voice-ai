package web_proxy_api

import (
	"context"
	"testing"

	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type fakeIndexerServiceClient struct {
	document_client.IndexerServiceClient
	calls int
	auth  types.SimplePrinciple
}

func (f *fakeIndexerServiceClient) IndexKnowledgeDocument(_ context.Context, auth types.SimplePrinciple, _ *protos.IndexKnowledgeDocumentRequest) (*protos.IndexKnowledgeDocumentResponse, error) {
	f.calls++
	f.auth = auth
	return &protos.IndexKnowledgeDocumentResponse{Success: true}, nil
}

func newDocumentProxyTest(t *testing.T, client document_client.IndexerServiceClient) *indexerApi {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	return &indexerApi{logger: logger, indexerServiceClient: client}
}

func documentProxyContext(principle types.SimplePrinciple) context.Context {
	switch auth := principle.(type) {
	case *types.ProjectScope:
		return context.WithValue(context.Background(), types.CTX_, &types.PlainClaimPrinciple[*types.ProjectScope]{Info: auth})
	case *types.ServiceScope:
		return context.WithValue(context.Background(), types.CTX_, &types.PlainClaimPrinciple[*types.ServiceScope]{Info: auth})
	case *types.OrganizationScope:
		return context.WithValue(context.Background(), types.CTX_, &types.PlainClaimPrinciple[*types.OrganizationScope]{Info: auth})
	default:
		return context.Background()
	}
}

func TestIndexKnowledgeDocumentRejectsMissingProjectContext(t *testing.T) {
	organizationID := uint64(11)
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(&types.OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}), &protos.IndexKnowledgeDocumentRequest{})

	if err == nil || err.Error() != "unauthenticated request for invoke" {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if response.GetSuccess() || client.calls != 0 {
		t.Fatalf("IndexKnowledgeDocument() response success = %v, client calls = %d", response.GetSuccess(), client.calls)
	}
}

func TestIndexKnowledgeDocumentAcceptsProjectScope(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.ProjectScope{
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(auth), &protos.IndexKnowledgeDocumentRequest{})

	if err != nil {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if !response.GetSuccess() || client.calls != 1 || client.auth != auth {
		t.Fatalf("IndexKnowledgeDocument() response success = %v, client calls = %d, auth = %T", response.GetSuccess(), client.calls, client.auth)
	}
}

func TestIndexKnowledgeDocumentAcceptsDelegatedProjectContext(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := &types.ServiceScope{
		OrganizationId: &organizationID,
		ProjectId:      &projectID,
	}
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(auth), &protos.IndexKnowledgeDocumentRequest{})

	if err != nil {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if !response.GetSuccess() || client.calls != 1 || client.auth != auth {
		t.Fatalf("IndexKnowledgeDocument() response success = %v, client calls = %d, auth = %T", response.GetSuccess(), client.calls, client.auth)
	}
}
