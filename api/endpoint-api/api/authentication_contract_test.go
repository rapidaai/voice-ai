package endpoint_api

import (
	"bytes"
	"context"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
)

type endpointLookupStub struct {
	internal_services.EndpointService
	called bool
	option *internal_services.GetEndpointOption
}

func (stub *endpointLookupStub) Get(
	_ context.Context,
	_ authentication.Principal,
	_ uint64,
	_ *uint64,
	option *internal_services.GetEndpointOption,
) (*internal_gorm.Endpoint, error) {
	stub.called = true
	stub.option = option
	return nil, errors.New("stop after endpoint lookup")
}

func TestEndpointRPCAuthenticationContract(t *testing.T) {
	expected := map[string][]string{
		"CreateEndpoint":                   {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"CreateEndpointCacheConfiguration": {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"CreateEndpointProviderModel":      {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"CreateEndpointRetryConfiguration": {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"CreateEndpointTag":                {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"ForkEndpoint":                     {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"GetAllEndpoint":                   {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"GetAllEndpointLog":                {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"GetAllEndpointProviderModel":      {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"GetEndpoint":                      {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"GetEndpointLog":                   {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"UpdateEndpointDetail":             {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"UpdateEndpointVersion":            {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"Invoke":                           {"IdentityOrganizationCredential", "IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"Probe":                            {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
		"Update":                           {"IdentityProjectCredential", "IdentityService", "IdentityUser"},
	}

	actual := make(map[string][]string, len(expected))
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range files {
		if filepath.Ext(filename) != ".go" || len(filename) >= 8 && filename[len(filename)-8:] == "_test.go" {
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
			if _, tracked := expected[function.Name.Name]; !tracked {
				continue
			}
			assertEndpointExplicitAuthenticationPattern(t, fileSet, filename, function)
			hasPrincipal := false
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
				if packageName, ok := selector.X.(*ast.Ident); ok && packageName.Name == "middleware_authentication" && selector.Sel.Name == "Principal" {
					hasPrincipal = true
				}
				if selector.Sel.Name != "OneOf" {
					return true
				}
				for _, argument := range call.Args {
					authType, ok := argument.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					packageName, ok := authType.X.(*ast.Ident)
					if ok && packageName.Name == "authentication" {
						scopes[authType.Sel.Name] = struct{}{}
					}
				}
				return true
			})
			if !hasPrincipal {
				t.Errorf("%s does not read the request principal", function.Name.Name)
			}
			for scope := range scopes {
				actual[function.Name.Name] = append(actual[function.Name.Name], scope)
			}
			sort.Strings(actual[function.Name.Name])
		}
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("RPC authentication scopes = %#v, want %#v", actual, expected)
	}
}

func assertEndpointExplicitAuthenticationPattern(t *testing.T, fileSet *token.FileSet, filename string, function *ast.FuncDecl) {
	t.Helper()
	var source bytes.Buffer
	if err := format.Node(&source, fileSet, function); err != nil {
		t.Fatalf("format %s:%s: %v", filename, function.Name.Name, err)
	}
	for _, required := range []string{
		"iAuth, ok := middleware_authentication.Principal(ctx)",
		"if !ok {",
		"status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())",
		"if !validator.OneOf(iAuth.Subject().Type,",
		"status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("%s:%s missing explicit authentication contract %q", filename, function.Name.Name, required)
		}
	}
}

func TestEndpointRPCAuthenticationErrors(t *testing.T) {
	organizationID := uint64(2)
	projectlessUserContext := contextWithTestPrincipal(t, authentication.Identity{Type: authentication.IdentityUser, ID: 1}, authentication.Tenant{OrganizationID: organizationID})
	organizationContext := contextWithTestPrincipal(t, authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 3}, authentication.Tenant{OrganizationID: organizationID})

	tests := []struct {
		name string
		call func() error
		code codes.Code
	}{
		{
			name: "missing authentication",
			call: func() error {
				_, err := (&endpointGRPCApi{}).ForkEndpoint(context.Background(), &protos.ForkEndpointRequest{})
				return err
			},
			code: codes.Unauthenticated,
		},
		{
			name: "disallowed organization scope",
			call: func() error {
				_, err := (&endpointGRPCApi{}).ForkEndpoint(organizationContext, &protos.ForkEndpointRequest{})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "fork is unimplemented after scope",
			call: func() error {
				_, err := (&endpointGRPCApi{}).ForkEndpoint(projectlessUserContext, &protos.ForkEndpointRequest{})
				return err
			},
			code: codes.Unimplemented,
		},
		{
			name: "probe is unimplemented after scope",
			call: func() error {
				_, err := (&invokerGRPCApi{}).Probe(projectlessUserContext, &protos.ProbeRequest{})
				return err
			},
			code: codes.Unimplemented,
		},
		{
			name: "update is unimplemented after scope",
			call: func() error {
				_, err := (&invokerGRPCApi{}).Update(projectlessUserContext, &protos.UpdateRequest{})
				return err
			},
			code: codes.Unimplemented,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if code := status.Code(test.call()); code != test.code {
				t.Fatalf("status code = %v, want %v", code, test.code)
			}
		})
	}
}

func TestOrganizationInvokeUsesPublicEndpointLookupPolicy(t *testing.T) {
	organizationID := uint64(2)
	ctx := contextWithTestPrincipal(t, authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 3}, authentication.Tenant{OrganizationID: organizationID})
	endpointService := &endpointLookupStub{}
	api := &invokerGRPCApi{invokerApi: invokerApi{endpointService: endpointService}}

	_, _ = api.Invoke(ctx, &protos.InvokeRequest{})

	if !endpointService.called {
		t.Fatal("organization Invoke did not reach endpoint lookup")
	}
	if endpointService.option == nil || !endpointService.option.AllowPublicWithoutProject {
		t.Fatal("organization Invoke did not use public endpoint lookup policy")
	}
}

func contextWithTestPrincipal(t *testing.T, subject authentication.Identity, tenant authentication.Tenant) context.Context {
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
