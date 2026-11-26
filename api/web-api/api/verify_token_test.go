// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type verifyTokenService struct {
	value                  authentication.Authentication
	err                    error
	userToken              string
	userID                 uint64
	selectedOrganizationID uint64
	selectedProjectID      uint64
	projectToken           string
	organizationToken      string
}

func (service *verifyTokenService) ResolveUserCredential(_ context.Context, token string, userID uint64, selectedOrganizationID uint64, selectedProjectID uint64) (authentication.Authentication, error) {
	service.userToken, service.userID = token, userID
	service.selectedOrganizationID, service.selectedProjectID = selectedOrganizationID, selectedProjectID
	return service.value, service.err
}

func (service *verifyTokenService) ResolveProjectCredential(_ context.Context, token string) (authentication.Authentication, error) {
	service.projectToken = token
	return service.value, service.err
}

func (service *verifyTokenService) ResolveOrganizationCredential(_ context.Context, token string) (authentication.Authentication, error) {
	service.organizationToken = token
	return service.value, service.err
}

func verifyTokenContext(t *testing.T) context.Context {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 9007, Issuer: "workflow-api"},
		authentication.Tenant{},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func verifyTokenValue(t *testing.T) authentication.Authentication {
	t.Helper()
	value, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 8},
		authentication.Tenant{OrganizationID: 20},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "8"}},
		nil,
		[]authentication.Organization{{ID: 20, Name: "Acme"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func verifyTokenAPI(t *testing.T, service *verifyTokenService) *webAuthGRPCApi {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	return &webAuthGRPCApi{
		webAuthApi:                        webAuthApi{logger: logger},
		userAuthenticationService:         service,
		projectAuthenticationService:      service,
		organizationAuthenticationService: service,
	}
}

func TestVerifyTokenResolvesEveryCredentialType(t *testing.T) {
	service := &verifyTokenService{value: verifyTokenValue(t)}
	api := verifyTokenAPI(t, service)
	ctx := verifyTokenContext(t)

	user, err := api.VerifyToken(ctx, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_User{User: &protos.VerifyTokenRequest_UserCredential{
		Token: "session", UserId: 3, SelectedOrganizationId: 20, SelectedProjectId: 4,
	}}})
	if err != nil || user.GetAuthentication().GetPrincipal().GetSubject().GetId() != 8 ||
		service.userToken != "session" || service.userID != 3 || service.selectedOrganizationID != 20 || service.selectedProjectID != 4 {
		t.Fatalf("VerifyToken(user) = %+v, %v, source = (%q,%d,%d,%d)", user, err, service.userToken, service.userID, service.selectedOrganizationID, service.selectedProjectID)
	}

	project, err := api.VerifyToken(ctx, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Project{Project: &protos.VerifyTokenRequest_ProjectCredential{Token: "project-key"}}})
	if err != nil || project.GetAuthentication() == nil || service.projectToken != "project-key" {
		t.Fatalf("VerifyToken(project) = %+v, %v, source = %q", project, err, service.projectToken)
	}

	organization, err := api.VerifyToken(ctx, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Organization{Organization: &protos.VerifyTokenRequest_OrganizationCredential{Token: "organization-key"}}})
	if err != nil || organization.GetAuthentication() == nil || service.organizationToken != "organization-key" {
		t.Fatalf("VerifyToken(organization) = %+v, %v, source = %q", organization, err, service.organizationToken)
	}
}

func TestVerifyTokenRequiresServicePrincipalAndExactCredential(t *testing.T) {
	service := &verifyTokenService{value: verifyTokenValue(t)}
	api := verifyTokenAPI(t, service)
	project := func(token string) *protos.VerifyTokenRequest {
		return &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Project{Project: &protos.VerifyTokenRequest_ProjectCredential{Token: token}}}
	}

	for name, test := range map[string]struct {
		ctx     context.Context
		request *protos.VerifyTokenRequest
	}{
		"missing service principal": {ctx: context.Background(), request: project("project-key")},
		"missing credential":        {ctx: verifyTokenContext(t), request: &protos.VerifyTokenRequest{}},
		"malformed credential":      {ctx: verifyTokenContext(t), request: project(" project-key")},
		"nil credential message": {ctx: verifyTokenContext(t), request: &protos.VerifyTokenRequest{
			Credential: &protos.VerifyTokenRequest_Project{},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := api.VerifyToken(test.ctx, test.request); status.Code(err) != codes.Unauthenticated {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestVerifyTokenMapsResolutionFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "invalid credential", err: pkg_errors.AuthenticationCredentialInvalidError{}, code: codes.Unauthenticated},
		{name: "canceled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "internal", err: errors.New("database unavailable"), code: codes.Internal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := verifyTokenAPI(t, &verifyTokenService{err: test.err})
			request := &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Project{Project: &protos.VerifyTokenRequest_ProjectCredential{Token: "project-key"}}}
			if _, err := api.VerifyToken(verifyTokenContext(t), request); status.Code(err) != test.code {
				t.Fatalf("code = %v, error = %v", status.Code(err), err)
			}
		})
	}
}
