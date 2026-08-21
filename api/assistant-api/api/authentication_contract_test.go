package assistant_api

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

type authenticationPolicyMatrix struct {
	Entries []authenticationPolicyEntry `json:"entries"`
}

type authenticationPolicyEntry struct {
	API              string   `json:"api"`
	File             string   `json:"file"`
	Handler          string   `json:"handler"`
	Transport        string   `json:"transport"`
	AllowedAuthTypes []string `json:"allowed_auth_types"`
}

type expectedAuthenticationContract struct {
	scopes    []string
	transport string
}

func TestAssistantAuthenticationContract(t *testing.T) {
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

	expected := map[string]map[string]expectedAuthenticationContract{}
	entryCount := 0
	for _, entry := range matrix.Entries {
		if entry.API != "assistant-api" {
			continue
		}
		entryCount++
		if expected[entry.File] == nil {
			expected[entry.File] = map[string]expectedAuthenticationContract{}
		}
		scopes := make([]string, 0, len(entry.AllowedAuthTypes))
		for _, authType := range entry.AllowedAuthTypes {
			scopes = append(scopes, map[string]string{
				"user": "AuthTypeUser", "project": "AuthTypeProject", "service": "AuthTypeService", "organization": "AuthTypeOrg",
			}[authType])
		}
		sort.Strings(scopes)
		expected[entry.File][entry.Handler] = expectedAuthenticationContract{scopes: scopes, transport: entry.Transport}
	}
	if entryCount != 101 {
		t.Fatalf("assistant policy entries = %d, want 101", entryCount)
	}

	for filename, handlers := range expected {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(repositoryRoot, filename), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		seen := map[string]int{}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			expectedContract, tracked := handlers[function.Name.Name]
			if !tracked {
				continue
			}
			seen[function.Name.Name]++
			authorizeAssignment, authFailure, scopeAssignment, scopeFailure := authenticationStatements(function)
			if authorizeAssignment == nil || authFailure == nil || scopeAssignment == nil || scopeFailure == nil {
				t.Errorf("%s:%s does not use the explicit Authorize/fail/Scope/fail statement order", filename, function.Name.Name)
				continue
			}
			assertAssignmentNames(t, filename, function.Name.Name, authorizeAssignment, "auth", "authErr")
			assertAssignmentNames(t, filename, function.Name.Name, scopeAssignment, "iAuth", "scopeErr")
			assertErrorCondition(t, filename, function.Name.Name, authFailure, "authErr")
			assertErrorCondition(t, filename, function.Name.Name, scopeFailure, "scopeErr")
			if expectedContract.transport == "grpc" {
				assertGRPCStatusReturn(t, filename, function.Name.Name, authFailure, "Unauthenticated", "authErr")
				assertGRPCStatusReturn(t, filename, function.Name.Name, scopeFailure, "PermissionDenied", "scopeErr")
			}

			scopes := map[string]struct{}{}
			ast.Inspect(scopeAssignment, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
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
			actualScopes := make([]string, 0, len(scopes))
			for scope := range scopes {
				actualScopes = append(actualScopes, scope)
			}
			sort.Strings(actualScopes)
			if !reflect.DeepEqual(actualScopes, expectedContract.scopes) {
				t.Errorf("%s:%s scopes = %v, want %v", filename, function.Name.Name, actualScopes, expectedContract.scopes)
			}
			assertNoUnscopedAuthenticationUse(t, filename, function.Name.Name, function, authorizeAssignment, scopeAssignment)
		}
		for handler := range handlers {
			if seen[handler] == 0 {
				t.Errorf("%s:%s not found", filename, handler)
			}
		}
	}
}

func authenticationStatements(function *ast.FuncDecl) (*ast.AssignStmt, *ast.IfStmt, *ast.AssignStmt, *ast.IfStmt) {
	for index := 0; index+3 < len(function.Body.List); index++ {
		authorizeAssignment, authorizeOK := function.Body.List[index].(*ast.AssignStmt)
		authFailure, authFailureOK := function.Body.List[index+1].(*ast.IfStmt)
		scopeAssignment, scopeOK := function.Body.List[index+2].(*ast.AssignStmt)
		scopeFailure, scopeFailureOK := function.Body.List[index+3].(*ast.IfStmt)
		if authorizeOK && authFailureOK && scopeOK && scopeFailureOK && assignmentCalls(authorizeAssignment, "Authorize") && assignmentCalls(scopeAssignment, "Scope") {
			return authorizeAssignment, authFailure, scopeAssignment, scopeFailure
		}
	}
	return nil, nil, nil, nil
}

func assignmentCalls(assignment *ast.AssignStmt, name string) bool {
	if len(assignment.Rhs) != 1 {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == name
}

func assertAssignmentNames(t *testing.T, filename, handler string, assignment *ast.AssignStmt, first, second string) {
	t.Helper()
	if len(assignment.Lhs) != 2 {
		t.Errorf("%s:%s assignment has %d results, want 2", filename, handler, len(assignment.Lhs))
		return
	}
	for index, want := range []string{first, second} {
		identifier, ok := assignment.Lhs[index].(*ast.Ident)
		if !ok || identifier.Name != want {
			t.Errorf("%s:%s assignment result %d = %v, want %s", filename, handler, index, assignment.Lhs[index], want)
		}
	}
}

func assertErrorCondition(t *testing.T, filename, handler string, failure *ast.IfStmt, errorName string) {
	t.Helper()
	condition, ok := failure.Cond.(*ast.BinaryExpr)
	identifier, identifierOK := condition.X.(*ast.Ident)
	nilIdentifier, nilOK := condition.Y.(*ast.Ident)
	if !ok || !identifierOK || identifier.Name != errorName || !nilOK || nilIdentifier.Name != "nil" || condition.Op.String() != "!=" {
		t.Errorf("%s:%s failure condition must be %s != nil", filename, handler, errorName)
	}
}

func assertGRPCStatusReturn(t *testing.T, filename, handler string, failure *ast.IfStmt, codeName, errorName string) {
	t.Helper()
	var statusCall *ast.CallExpr
	ast.Inspect(failure.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		packageName, packageOK := selector.X.(*ast.Ident)
		if ok && packageOK && packageName.Name == "status" && selector.Sel.Name == "Error" {
			statusCall = call
		}
		return true
	})
	if statusCall == nil || len(statusCall.Args) != 2 {
		t.Errorf("%s:%s failure does not return status.Error", filename, handler)
		return
	}
	code, codeOK := statusCall.Args[0].(*ast.SelectorExpr)
	codePackage, packageOK := code.X.(*ast.Ident)
	if !codeOK || !packageOK || codePackage.Name != "codes" || code.Sel.Name != codeName {
		t.Errorf("%s:%s status code = %v, want codes.%s", filename, handler, statusCall.Args[0], codeName)
	}
	errorCall, callOK := statusCall.Args[1].(*ast.CallExpr)
	errorSelector, selectorOK := errorCall.Fun.(*ast.SelectorExpr)
	errorIdentifier, identifierOK := errorSelector.X.(*ast.Ident)
	if !callOK || !selectorOK || !identifierOK || errorIdentifier.Name != errorName || errorSelector.Sel.Name != "Error" {
		t.Errorf("%s:%s status message must be %s.Error()", filename, handler, errorName)
	}
}

func assertNoUnscopedAuthenticationUse(t *testing.T, filename, handler string, function *ast.FuncDecl, authorizeAssignment, scopeAssignment *ast.AssignStmt) {
	t.Helper()
	authIdentifier, ok := authorizeAssignment.Lhs[0].(*ast.Ident)
	if !ok || authIdentifier.Obj == nil {
		return
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && identifier.Obj == authIdentifier.Obj && identifier.Pos() > scopeAssignment.End() {
			t.Errorf("%s:%s uses unscoped auth after Scope", filename, handler)
		}
		return true
	})
}
