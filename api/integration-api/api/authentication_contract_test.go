package integration_api

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type authenticationPolicyMatrix struct {
	Entries []authenticationPolicyEntry `json:"entries"`
}

type authenticationPolicyEntry struct {
	API              string   `json:"api"`
	File             string   `json:"file"`
	Handler          string   `json:"handler"`
	AllowedAuthTypes []string `json:"allowed_auth_types"`
}

func TestIntegrationRPCAuthenticationContract(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../../.."))
	data, err := os.ReadFile(filepath.Join(repositoryRoot, "rfcs/0001-phase-1e-auth-policy-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix authenticationPolicyMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}

	expected := map[string]map[string][]string{}
	for _, entry := range matrix.Entries {
		if entry.API != "integration-api" {
			continue
		}
		if expected[entry.File] == nil {
			expected[entry.File] = map[string][]string{}
		}
		scopes := make([]string, 0, len(entry.AllowedAuthTypes))
		for _, authType := range entry.AllowedAuthTypes {
			scopes = append(scopes, map[string]string{
				"user": "AuthTypeUser", "project": "AuthTypeProject", "service": "AuthTypeService", "organization": "AuthTypeOrg",
			}[authType])
		}
		sort.Strings(scopes)
		expected[entry.File][entry.Handler] = scopes
	}

	for filename, handlers := range expected {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, filepath.Join(repositoryRoot, filename), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		seen := map[string]int{}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil {
				continue
			}
			expectedScopes, tracked := handlers[function.Name.Name]
			if !tracked {
				continue
			}
			seen[function.Name.Name]++
			assertExplicitAuthenticationPattern(t, fileSet, filename, function)
			hasAuthorize := false
			scopes := map[string]struct{}{}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if packageName, ok := selector.X.(*ast.Ident); ok && packageName.Name == "types" && selector.Sel.Name == "Authorize" {
					hasAuthorize = true
				}
				if selector.Sel.Name == "Scope" {
					for _, argument := range call.Args {
						authType, ok := argument.(*ast.SelectorExpr)
						if !ok {
							continue
						}
						packageName, ok := authType.X.(*ast.Ident)
						if ok && packageName.Name == "types" {
							scopes[authType.Sel.Name] = struct{}{}
						}
					}
				}
				return true
			})
			if !hasAuthorize {
				t.Errorf("%s:%s does not call types.Authorize", filename, function.Name.Name)
			}
			actualScopes := make([]string, 0, len(scopes))
			for scope := range scopes {
				actualScopes = append(actualScopes, scope)
			}
			sort.Strings(actualScopes)
			if !reflect.DeepEqual(actualScopes, expectedScopes) {
				t.Errorf("%s:%s scopes = %v, want %v", filename, function.Name.Name, actualScopes, expectedScopes)
			}
		}
		for handler := range handlers {
			if seen[handler] == 0 {
				t.Errorf("%s:%s not found", filename, handler)
			}
		}
	}
}

func TestIntegrationRPCAuthenticationErrors(t *testing.T) {
	organizationID := uint64(2)
	organizationContext := context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	})
	projectlessUserContext := context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: 1},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	})

	tests := []struct {
		name string
		ctx  context.Context
		code codes.Code
	}{
		{name: "missing authentication", ctx: context.Background(), code: codes.Unauthenticated},
		{name: "disallowed organization scope", ctx: organizationContext, code: codes.PermissionDenied},
		{name: "missing project context", ctx: projectlessUserContext, code: codes.PermissionDenied},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&integrationApi{}).Chat(test.ctx, &protos.ChatRequest{}, "provider")
			if code := status.Code(err); code != test.code {
				t.Fatalf("status code = %v, want %v", code, test.code)
			}
		})
	}
}

func TestDetachedAuditContextPreservesAuthenticationAfterCancellation(t *testing.T) {
	authentication := &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: 41},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 2},
	}
	requestContext, cancel := context.WithCancel(context.WithValue(context.Background(), types.CTX_, authentication))
	detachedContext := detachedAuditContext(requestContext)
	cancel()

	if detachedContext.Err() != nil {
		t.Fatalf("detached context error = %v, want nil", detachedContext.Err())
	}
	resolved, err := types.Authorize(detachedContext)
	if err != nil {
		t.Fatalf("authorize detached context: %v", err)
	}
	actor, err := resolved.Actor()
	if err != nil {
		t.Fatalf("resolve detached actor: %v", err)
	}
	if actor.Type != types.ActorTypeService || actor.ID != 41 {
		t.Fatalf("actor = %+v, want service:41", actor)
	}
}

func assertExplicitAuthenticationPattern(t *testing.T, fileSet *token.FileSet, filename string, function *ast.FuncDecl) {
	t.Helper()
	var source bytes.Buffer
	if err := format.Node(&source, fileSet, function); err != nil {
		t.Fatalf("format %s:%s: %v", filename, function.Name.Name, err)
	}
	for _, required := range []string{
		"auth, authErr := types.Authorize(",
		"if authErr != nil {",
		"status.Error(codes.Unauthenticated, authErr.Error())",
		"iAuth, scopeErr := auth.Scope(",
		"if scopeErr != nil {",
		"status.Error(codes.PermissionDenied, scopeErr.Error())",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("%s:%s missing explicit authentication contract %q", filename, function.Name.Name, required)
		}
	}
}
