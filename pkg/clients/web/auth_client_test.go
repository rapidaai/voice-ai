// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	authentication_jwt "github.com/rapidaai/pkg/authentication/jwt"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	authentication_proto "github.com/rapidaai/protos/authentication"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type authenticationServiceClientStub struct {
	protos.AuthenticationServiceClient
	response *protos.VerifyTokenResponse
	err      error
	request  *protos.VerifyTokenRequest
}

func (stub *authenticationServiceClientStub) VerifyToken(ctx context.Context, request *protos.VerifyTokenRequest, _ ...grpc.CallOption) (*protos.VerifyTokenResponse, error) {
	stub.request = request
	stub.capture(ctx)
	return stub.response, stub.err
}

func (stub *authenticationServiceClientStub) capture(ctx context.Context) {
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	values := outgoing.Get(types.SERVICE_SCOPE_KEY)
	if len(values) != 1 || values[0] == "" {
		stub.err = errors.New("missing service assertion")
	}
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	return logger
}

func authenticationClientForTest(t *testing.T, response *authentication_proto.Authentication) (*authenticationClient, *authenticationServiceClientStub) {
	t.Helper()
	issuer, err := authentication_jwt.NewIssuer("web-api", bytes.Repeat([]byte{1}, sha256.Size))
	if err != nil {
		t.Fatal(err)
	}
	stub := &authenticationServiceClientStub{response: &protos.VerifyTokenResponse{Authentication: response}}
	return &authenticationClient{internalClient: clients.NewInternalClientWithIssuer(issuer, nil, nil), client: stub}, stub
}

func TestAuthenticationClientResolvesEveryCredentialType(t *testing.T) {
	user, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityUser, ID: 3},
		authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "4"}},
		&authentication.User{ID: 3, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	project, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 5},
		authentication.Tenant{OrganizationID: 11, ProjectID: 12},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "5"}},
		nil,
		[]authentication.Organization{{ID: 11, Name: "Organization", Projects: []authentication.Project{{ID: 12, Name: "Project"}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	organization, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 6},
		authentication.Tenant{OrganizationID: 11},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "6"}},
		nil,
		[]authentication.Organization{{ID: 11, Name: "Organization"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	client, stub := authenticationClientForTest(t, user.Proto())
	if resolved, err := client.ResolveUserCredential(context.Background(), "session", 3, 11, 0); err != nil || resolved.Principal().Subject().Type != authentication.IdentityUser ||
		stub.request.GetUser().GetToken() != "session" || stub.request.GetUser().GetUserId() != 3 ||
		stub.request.GetUser().GetSelectedOrganizationId() != 11 || stub.request.GetUser().GetSelectedProjectId() != 0 {
		t.Fatalf("ResolveUserCredential() = %+v, %v, request = %+v", resolved.Principal(), err, stub.request)
	}
	stub.response.Authentication = project.Proto()
	if resolved, err := client.ResolveProjectCredential(context.Background(), "project-key"); err != nil || resolved.Principal().Subject().Type != authentication.IdentityProjectCredential || stub.request.GetProject().GetToken() != "project-key" {
		t.Fatalf("ResolveProjectCredential() = %+v, %v, request = %+v", resolved.Principal(), err, stub.request)
	}
	stub.response.Authentication = organization.Proto()
	if resolved, err := client.ResolveOrganizationCredential(context.Background(), "organization-key"); err != nil || resolved.Principal().Subject().Type != authentication.IdentityOrganizationCredential || stub.request.GetOrganization().GetToken() != "organization-key" {
		t.Fatalf("ResolveOrganizationCredential() = %+v, %v, request = %+v", resolved.Principal(), err, stub.request)
	}
}

func TestAuthenticationClientRejectsMissingAggregate(t *testing.T) {
	client, _ := authenticationClientForTest(t, nil)
	_, err := client.ResolveProjectCredential(context.Background(), "project-key")
	var invalid pkg_errors.AuthenticationInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("ResolveProjectCredential() error = %T, want AuthenticationInvalidError", err)
	}
}
