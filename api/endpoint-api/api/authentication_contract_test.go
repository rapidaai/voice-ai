package endpoint_api

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type endpointLookupStub struct {
	internal_services.EndpointService
	called bool
	option *internal_services.GetEndpointOption
}

func (stub *endpointLookupStub) Get(
	_ context.Context,
	_ types.SimplePrinciple,
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
		"CreateEndpoint":                   {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"CreateEndpointCacheConfiguration": {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"CreateEndpointProviderModel":      {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"CreateEndpointRetryConfiguration": {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"CreateEndpointTag":                {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"ForkEndpoint":                     {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"GetAllEndpoint":                   {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"GetAllEndpointLog":                {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"GetAllEndpointProviderModel":      {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"GetEndpoint":                      {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"GetEndpointLog":                   {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"UpdateEndpointDetail":             {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"UpdateEndpointVersion":            {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"Invoke":                           {"AuthTypeOrg", "AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"Probe":                            {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
		"Update":                           {"AuthTypeProject", "AuthTypeService", "AuthTypeUser"},
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
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
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
				if selector.Sel.Name != "Scope" {
					return true
				}
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
				return true
			})
			if !hasAuthorize {
				t.Errorf("%s does not call types.Authorize", function.Name.Name)
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

func TestEndpointRPCAuthenticationErrors(t *testing.T) {
	organizationID := uint64(2)
	projectlessUserContext := context.WithValue(context.Background(), types.CTX_, &types.PlainAuthPrinciple{
		User:             types.UserInfo{Id: 1},
		OrganizationRole: &types.OrganizaitonRole{OrganizationId: organizationID},
	})
	organizationContext := context.WithValue(context.Background(), types.CTX_, &types.OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	})

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
			name: "fork requires project context",
			call: func() error {
				_, err := (&endpointGRPCApi{}).ForkEndpoint(projectlessUserContext, &protos.ForkEndpointRequest{})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "probe requires project context",
			call: func() error {
				_, err := (&invokerGRPCApi{}).Probe(projectlessUserContext, &protos.ProbeRequest{})
				return err
			},
			code: codes.PermissionDenied,
		},
		{
			name: "update requires project context",
			call: func() error {
				_, err := (&invokerGRPCApi{}).Update(projectlessUserContext, &protos.UpdateRequest{})
				return err
			},
			code: codes.PermissionDenied,
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
	ctx := context.WithValue(context.Background(), types.CTX_, &types.OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	})
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
