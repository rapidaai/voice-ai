package web_api

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type authenticationPolicyMatrix struct {
	Entries []authenticationPolicyEntry `json:"entries"`
}

type authenticationPolicyEntry struct {
	API              string   `json:"api"`
	File             string   `json:"file"`
	Handler          string   `json:"handler"`
	Route            string   `json:"route"`
	Transport        string   `json:"transport"`
	AllowedAuthTypes []string `json:"allowed_auth_types"`
}

type controllerPolicyDocument struct {
	GRPC map[string]struct {
		Authorization string `yaml:"authorization"`
	} `yaml:"grpc"`
}

func TestWebRPCAuthenticationContract(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../../.."))
	data, err := os.ReadFile(filepath.Join(filepath.Dir(testFile), "testdata/authentication-policy-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix authenticationPolicyMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}

	expected := map[string]map[string]struct{}{}
	trackedRoutes := map[string]string{}
	for _, entry := range matrix.Entries {
		if entry.API != "web-api" {
			continue
		}
		if entry.Transport != "grpc" {
			t.Fatalf("protected controller route %q has transport %q", entry.Route, entry.Transport)
		}
		separator := strings.LastIndexByte(entry.Route, '/')
		if separator < 0 || separator == len(entry.Route)-1 || entry.Route[separator+1:] != entry.Handler {
			t.Fatalf("invalid controller route %q for handler %q", entry.Route, entry.Handler)
		}
		if existing, exists := trackedRoutes[entry.Route]; exists {
			t.Fatalf("duplicate controller route %q for handlers %q and %q", entry.Route, existing, entry.Handler)
		}
		trackedRoutes[entry.Route] = entry.Handler
		if expected[entry.File] == nil {
			expected[entry.File] = map[string]struct{}{}
		}
		expected[entry.File][entry.Handler] = struct{}{}
	}

	policyData, err := os.ReadFile(filepath.Join(repositoryRoot, "pkg/authentication/policy/v1/web-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var policy controllerPolicyDocument
	if err := yaml.Unmarshal(policyData, &policy); err != nil {
		t.Fatal(err)
	}
	protectedRoutes := map[string]string{}
	for route, routePolicy := range policy.GRPC {
		if routePolicy.Authorization == "public" || routePolicy.Authorization == "disabled" {
			continue
		}
		separator := strings.LastIndexByte(route, '/')
		if separator < 0 || separator == len(route)-1 {
			t.Fatalf("invalid protected gRPC route %q", route)
		}
		protectedRoutes[route] = route[separator+1:]
	}
	if !reflect.DeepEqual(trackedRoutes, protectedRoutes) {
		t.Fatalf("controller inventory does not match protected policy routes\ntracked: %#v\nprotected: %#v", trackedRoutes, protectedRoutes)
	}

	for filename, handlers := range expected {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(repositoryRoot, filename), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		candidates := map[string][]*ast.FuncDecl{}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil {
				continue
			}
			_, tracked := handlers[function.Name.Name]
			if !tracked {
				continue
			}
			candidates[function.Name.Name] = append(candidates[function.Name.Name], function)
		}
		for handler := range handlers {
			controllers := candidates[handler]
			if len(controllers) > 1 {
				grpcControllers := make([]*ast.FuncDecl, 0, len(controllers))
				for _, function := range controllers {
					receiver := function.Recv.List[0].Type
					if pointer, ok := receiver.(*ast.StarExpr); ok {
						receiver = pointer.X
					}
					receiverName, ok := receiver.(*ast.Ident)
					if ok && strings.Contains(receiverName.Name, "GRPC") {
						grpcControllers = append(grpcControllers, function)
					}
				}
				controllers = grpcControllers
			}
			if len(controllers) != 1 {
				t.Errorf("%s:%s found %d gRPC controllers", filename, handler, len(controllers))
				continue
			}
			if err := principalAuthenticationPattern(controllers[0]); err != nil {
				t.Errorf("%s:%s %v", filename, handler, err)
			}
		}
	}
}

func TestPrincipalAuthenticationPatternRequiresTerminatingUnauthenticatedGuard(t *testing.T) {
	for _, test := range []struct {
		name      string
		guard     string
		wantError bool
	}{
		{name: "terminating", guard: `return nil, status.Error(codes.Unauthenticated, "required")`},
		{name: "empty", guard: ``, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := fmt.Sprintf(`package fixture
func Handler() (any, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok { %s }
	return principal, nil
}`, test.guard)
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			function := file.Decls[0].(*ast.FuncDecl)
			err = principalAuthenticationPattern(function)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, want error %v", err, test.wantError)
			}
		})
	}
}

func principalAuthenticationPattern(function *ast.FuncDecl) error {
	statements := function.Body.List
	principalIndex := -1
	for index, statement := range statements {
		if matchesAssignment(statement, token.DEFINE, []string{"principal", "ok"}, "middleware_authentication", "Principal", "") {
			principalIndex = index
			break
		}
	}
	if principalIndex < 0 || principalIndex+1 >= len(statements) {
		return fmt.Errorf("does not load the middleware principal")
	}
	condition, ok := statements[principalIndex+1].(*ast.IfStmt)
	if !ok {
		return fmt.Errorf("does not check principal presence immediately")
	}
	unary, ok := condition.Cond.(*ast.UnaryExpr)
	if !ok || unary.Op != token.NOT {
		return fmt.Errorf("principal presence check is not fail closed")
	}
	identifier, identifierOK := unary.X.(*ast.Ident)
	if !identifierOK || identifier.Name != "ok" {
		return fmt.Errorf("principal presence check is not fail closed")
	}
	if len(condition.Body.List) != 1 {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	rejection, ok := condition.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(rejection.Results) != 2 {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	nilResult, ok := rejection.Results[0].(*ast.Ident)
	if !ok || nilResult.Name != "nil" {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	statusCall, ok := rejection.Results[1].(*ast.CallExpr)
	if !ok || len(statusCall.Args) == 0 {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	statusSelector, ok := statusCall.Fun.(*ast.SelectorExpr)
	if !ok || statusSelector.Sel.Name != "Error" {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	statusPackage, ok := statusSelector.X.(*ast.Ident)
	if !ok || statusPackage.Name != "status" {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	codeSelector, ok := statusCall.Args[0].(*ast.SelectorExpr)
	if !ok || codeSelector.Sel.Name != "Unauthenticated" {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	codePackage, ok := codeSelector.X.(*ast.Ident)
	if !ok || codePackage.Name != "codes" {
		return fmt.Errorf("missing principal does not terminate with unauthenticated")
	}
	return nil
}

func matchesAssignment(statement ast.Stmt, assignment token.Token, names []string, packageName, methodName, receiverName string) bool {
	assign, ok := statement.(*ast.AssignStmt)
	if !ok || assign.Tok != assignment || len(assign.Lhs) != len(names) || len(assign.Rhs) != 1 {
		return false
	}
	for index, expression := range assign.Lhs {
		identifier, ok := expression.(*ast.Ident)
		if !ok || identifier.Name != names[index] {
			return false
		}
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != methodName {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	if packageName != "" {
		return identifier.Name == packageName
	}
	return identifier.Name == receiverName
}
