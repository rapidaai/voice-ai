package web_proxy_api

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/authentication"
	document_client "github.com/rapidaai/pkg/clients/document"
	"github.com/rapidaai/pkg/commons"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
)

type fakeIndexerServiceClient struct {
	document_client.IndexerServiceClient
	calls int
	auth  authentication.Principal
}

func (f *fakeIndexerServiceClient) IndexKnowledgeDocument(_ context.Context, principal authentication.Principal, _ *protos.IndexKnowledgeDocumentRequest) (*protos.IndexKnowledgeDocumentResponse, error) {
	f.calls++
	f.auth = principal
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

func documentProxyContext(principal authentication.Principal) context.Context {
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx
}

func TestIndexKnowledgeDocumentRejectsMissingAuthentication(t *testing.T) {
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(context.Background(), &protos.IndexKnowledgeDocumentRequest{})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if response != nil || client.calls != 0 {
		t.Fatalf("IndexKnowledgeDocument() response = %v, client calls = %d", response, client.calls)
	}
}

func TestIndexKnowledgeDocumentAcceptsProjectScope(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 32},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "32"},
		}, nil,
	)
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(principal), &protos.IndexKnowledgeDocumentRequest{})

	if err != nil {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if !response.GetSuccess() || client.calls != 1 || client.auth.Subject() != principal.Subject() || client.auth.Tenant() != principal.Tenant() {
		t.Fatalf("IndexKnowledgeDocument() response success = %v, client calls = %d, auth = %T", response.GetSuccess(), client.calls, client.auth)
	}
}

func TestIndexKnowledgeDocumentAcceptsDelegatedProjectContext(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	principal, _ := authentication.NewDelegatedPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 32},
		authentication.ServiceActor{ID: 33, Issuer: "workflow-api"},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-current"},
		}, nil,
	)
	client := &fakeIndexerServiceClient{}
	api := newDocumentProxyTest(t, client)

	response, err := api.IndexKnowledgeDocument(documentProxyContext(principal), &protos.IndexKnowledgeDocumentRequest{})

	if err != nil {
		t.Fatalf("IndexKnowledgeDocument() error = %v", err)
	}
	if !response.GetSuccess() || client.calls != 1 || client.auth.Subject() != principal.Subject() || client.auth.Tenant() != principal.Tenant() {
		t.Fatalf("IndexKnowledgeDocument() response success = %v, client calls = %d, auth = %T", response.GetSuccess(), client.calls, client.auth)
	}
}
