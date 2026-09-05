package web_api

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
	AllowedAuthTypes []string `json:"allowed_auth_types"`
}

func TestWebRPCAuthenticationContract(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../../.."))
	data, err := os.ReadFile(filepath.Join(repositoryRoot, "rfcs/0001-actor-aware-audit-identity/jsons/phase-1e-auth-policy-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix authenticationPolicyMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}

	expected := map[string]map[string][]string{}
	for _, entry := range matrix.Entries {
		if entry.API != "web-api" {
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
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(repositoryRoot, filename), nil, 0)
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
			if filename == "api/web-api/api/organization.go" && function.Name.Name == "CreateOrganization" {
				assertOrganizationOnboardingAuthenticationPattern(t, filename, function)
				continue
			}
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
			if !reflect.DeepEqual(actualScopes, expectedScopes) {
				t.Errorf("%s:%s scopes = %v, want %v", filename, function.Name.Name, actualScopes, expectedScopes)
			}
			assertExplicitAuthenticationPattern(t, filename, function)
		}
		for handler := range handlers {
			if seen[handler] == 0 {
				t.Errorf("%s:%s not found", filename, handler)
			}
		}
	}
}

func assertOrganizationOnboardingAuthenticationPattern(t *testing.T, filename string, function *ast.FuncDecl) {
	t.Helper()
	statements := function.Body.List
	if len(statements) < 2 || !matchesAssignment(statements[0], token.DEFINE, []string{"iAuth", "authErr"}, "types", "AuthorizeUser", "") {
		t.Errorf("%s:%s does not use types.AuthorizeUser", filename, function.Name.Name)
		return
	}
	if !matchesErrorCheck(statements[1], "authErr") {
		t.Errorf("%s:%s does not check authErr immediately", filename, function.Name.Name)
	}
}

func assertExplicitAuthenticationPattern(t *testing.T, filename string, function *ast.FuncDecl) {
	t.Helper()
	statements := function.Body.List
	authorizeIndex := -1
	for index, statement := range statements {
		if matchesAssignment(statement, token.DEFINE, []string{"auth", "authErr"}, "types", "Authorize", "") {
			authorizeIndex = index
			break
		}
	}
	if authorizeIndex < 0 || authorizeIndex+3 >= len(statements) {
		t.Errorf("%s:%s does not contain the explicit authorization sequence", filename, function.Name.Name)
		return
	}
	if !matchesErrorCheck(statements[authorizeIndex+1], "authErr") {
		t.Errorf("%s:%s does not check authErr immediately", filename, function.Name.Name)
	}
	if !matchesAssignment(statements[authorizeIndex+2], token.DEFINE, []string{"iAuth", "scopeErr"}, "", "Scope", "auth") {
		t.Errorf("%s:%s does not assign scoped authentication to iAuth", filename, function.Name.Name)
	}
	if !matchesErrorCheck(statements[authorizeIndex+3], "scopeErr") {
		t.Errorf("%s:%s does not check scopeErr immediately", filename, function.Name.Name)
	}

	if function.Type.Results != nil {
		if !containsStatusError(statements[authorizeIndex+1], "Unauthenticated", "authErr") {
			t.Errorf("%s:%s authentication error is not codes.Unauthenticated", filename, function.Name.Name)
		}
		if !containsStatusError(statements[authorizeIndex+3], "PermissionDenied", "scopeErr") {
			t.Errorf("%s:%s scope error is not codes.PermissionDenied", filename, function.Name.Name)
		}
	}

	for _, statement := range statements[authorizeIndex+4:] {
		usesRawAuthentication := false
		ast.Inspect(statement, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && identifier.Name == "auth" {
				usesRawAuthentication = true
			}
			return true
		})
		if usesRawAuthentication {
			t.Errorf("%s:%s uses raw auth after scoping", filename, function.Name.Name)
			break
		}
	}
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

func matchesErrorCheck(statement ast.Stmt, errorName string) bool {
	condition, ok := statement.(*ast.IfStmt)
	if !ok {
		return false
	}
	binary, ok := condition.Cond.(*ast.BinaryExpr)
	if !ok || binary.Op != token.NEQ {
		return false
	}
	identifier, ok := binary.X.(*ast.Ident)
	if !ok || identifier.Name != errorName {
		return false
	}
	nilIdentifier, ok := binary.Y.(*ast.Ident)
	return ok && nilIdentifier.Name == "nil"
}

func containsStatusError(statement ast.Stmt, codeName, errorName string) bool {
	found := false
	ast.Inspect(statement, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		packageIdentifier, packageOK := selector.X.(*ast.Ident)
		if !ok || !packageOK || packageIdentifier.Name != "status" || selector.Sel.Name != "Error" {
			return true
		}
		codeSelector, ok := call.Args[0].(*ast.SelectorExpr)
		codePackage, packageOK := codeSelector.X.(*ast.Ident)
		if !ok || !packageOK || codePackage.Name != "codes" || codeSelector.Sel.Name != codeName {
			return true
		}
		errorCall, ok := call.Args[1].(*ast.CallExpr)
		if !ok || len(errorCall.Args) != 0 {
			return true
		}
		errorSelector, ok := errorCall.Fun.(*ast.SelectorExpr)
		errorIdentifier, identifierOK := errorSelector.X.(*ast.Ident)
		if ok && identifierOK && errorIdentifier.Name == errorName && errorSelector.Sel.Name == "Error" {
			found = true
		}
		return true
	})
	return found
}
