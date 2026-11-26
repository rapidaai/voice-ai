// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package integration_api

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/authentication"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
)

func TestIntegrationOperationsUsePrincipalAndProjectTenant(t *testing.T) {
	tracked := map[string]struct{}{
		"GetAuditLog": {}, "GetAllAuditLog": {}, "CreateMetadata": {}, "Chat": {},
		"StreamChatBidirectionalUnified": {}, "Embedding": {}, "Reranking": {},
	}
	found := make(map[string]struct{}, len(tracked))
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range files {
		if strings.HasSuffix(filename, "_test.go") || filename == "unified_provider.go" {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil {
				continue
			}
			if _, ok := tracked[function.Name.Name]; !ok {
				continue
			}
			var source bytes.Buffer
			if err := format.Node(&source, fileSet, function); err != nil {
				t.Fatalf("format %s:%s: %v", filename, function.Name.Name, err)
			}
			for _, required := range []string{
				"middleware_authentication.Principal(",
				"authentication.IdentityUser",
				"authentication.IdentityProjectCredential",
				"authentication.IdentityService",
				"projectContext := iAuth.Tenant()",
				"projectContext.OrganizationID == 0 || projectContext.ProjectID == 0",
			} {
				if !strings.Contains(source.String(), required) {
					t.Errorf("%s:%s missing authentication contract %q", filename, function.Name.Name, required)
				}
			}
			found[function.Name.Name] = struct{}{}
		}
	}
	if len(found) != len(tracked) {
		t.Fatalf("found authentication contracts for %v, want %v", found, tracked)
	}
}

func TestIntegrationRPCAuthenticationFailures(t *testing.T) {
	api := &auditLoggingGRPCApi{}
	_, err := api.GetAuditLog(context.Background(), &protos.GetAuditLogRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing principal status = %v, want %v", status.Code(err), codes.Unauthenticated)
	}

	organizationContext := contextWithIntegrationPrincipal(
		t,
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 3},
		authentication.Tenant{OrganizationID: 7},
	)
	_, err = api.GetAuditLog(organizationContext, &protos.GetAuditLogRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("organization principal status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}

	projectlessUserContext := contextWithIntegrationPrincipal(
		t,
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 7},
	)
	_, err = api.GetAuditLog(projectlessUserContext, &protos.GetAuditLogRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("projectless User status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func contextWithIntegrationPrincipal(t *testing.T, subject authentication.Identity, tenant authentication.Tenant) context.Context {
	t.Helper()
	proof := authentication.Proof{
		Method: authentication.AuthenticationUserSession,
		Reference: authentication.ProofReference{
			Type:  authentication.ProofReferenceUserSession,
			Value: "test-session",
		},
	}
	if subject.Type == authentication.IdentityOrganizationCredential {
		proof = authentication.Proof{
			Method: authentication.AuthenticationOrganizationCredential,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceCredential,
				Value: "3",
			},
		}
	}
	principal, err := authentication.NewDirectPrincipal(subject, tenant, proof, nil)
	if err != nil {
		t.Fatalf("construct test principal: %v", err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatalf("attach test principal: %v", err)
	}
	return ctx
}
