package web_proxy_api

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type fakeIndexerServiceClient struct {
	document_client.IndexerServiceClient
	calls int
	auth  *types.Authentication
}

func (f *fakeIndexerServiceClient) IndexKnowledgeDocument(_ context.Context, auth *types.Authentication, _ *protos.IndexKnowledgeDocumentRequest) (*protos.IndexKnowledgeDocumentResponse, error) {
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

func documentProxyContext(auth *types.Authentication) context.Context {
	return context.WithValue(context.Background(), types.CTX_, auth)
}

func TestIndexKnowledgeDocumentRejectsMissingProjectContext(t *testing.T) {
	organizationID := uint64(11)
	actor := types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 31}
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(&types.Authentication{
		AuthType:          types.AuthTypeOrg,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}), &protos.IndexKnowledgeDocumentRequest{})

	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if response != nil || client.calls != 0 {
		t.Fatalf("IndexKnowledgeDocument() response = %v, client calls = %d", response, client.calls)
	}
}

func TestIndexKnowledgeDocumentAcceptsProjectScope(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	actor := types.ActorIdentity{Type: types.ActorTypeProject, ID: 32}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
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
	actor := types.ActorIdentity{Type: types.ActorTypeService, ID: 33}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
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
