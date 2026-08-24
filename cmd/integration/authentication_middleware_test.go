package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGRPCUsesExplicitAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "integration.go")

	assertAuthenticationMiddlewareOrder(t, interceptorChain(t, file, "ChainUnaryInterceptor"), []string{
		"NewAuthenticationUnaryServerMiddleware",
		"NewProjectAuthenticatorUnaryServerMiddleware",
		"NewOrganizationAuthenticatorUnaryServerMiddleware",
		"NewServiceAuthenticatorUnaryServerMiddleware",
	})
	assertAuthenticationMiddlewareOrder(t, interceptorChain(t, file, "ChainStreamInterceptor"), []string{
		"NewAuthenticationStreamServerMiddleware",
		"NewProjectAuthenticatorStreamServerMiddleware",
		"NewOrganizationAuthenticatorStreamServerMiddleware",
		"NewServiceAuthenticatorStreamServerMiddleware",
	})
}

func TestInitDoesNotRegisterAuditActorCallbacks(t *testing.T) {
	file := parseCommandSource(t, "integration.go")
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && callName(call) == "RegisterAuditActorCallbacks" {
			t.Fatal("Init registers removed audit actor callbacks")
		}
		return true
	})
}

func parseCommandSource(t *testing.T, name string) *ast.File {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(testFile), name), nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return file
}

func interceptorChain(t *testing.T, file *ast.File, chainName string) []string {
	t.Helper()
	var chains [][]string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || callName(call) != chainName {
			return true
		}
		chain := make([]string, 0, len(call.Args))
		for _, argument := range call.Args {
			middleware, ok := argument.(*ast.CallExpr)
			if ok {
				chain = append(chain, callName(middleware))
			}
		}
		chains = append(chains, chain)
		return true
	})
	if len(chains) != 1 {
		t.Fatalf("found %d %s calls, want 1", len(chains), chainName)
	}
	return chains[0]
}

func callName(call *ast.CallExpr) string {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		return function.Name
	case *ast.SelectorExpr:
		return function.Sel.Name
	default:
		return ""
	}
}

func assertAuthenticationMiddlewareOrder(t *testing.T, chain []string, expected []string) {
	t.Helper()
	actual := make([]string, 0, len(expected))
	for _, middleware := range chain {
		switch middleware {
		case "NewAuthenticationUnaryServerMiddleware", "NewProjectAuthenticatorUnaryServerMiddleware", "NewOrganizationAuthenticatorUnaryServerMiddleware", "NewServiceAuthenticatorUnaryServerMiddleware",
			"NewAuthenticationStreamServerMiddleware", "NewProjectAuthenticatorStreamServerMiddleware", "NewOrganizationAuthenticatorStreamServerMiddleware", "NewServiceAuthenticatorStreamServerMiddleware":
			actual = append(actual, middleware)
		case "NewAuthenticationBoundaryUnaryServerMiddleware", "NewAuthenticationBoundaryStreamServerMiddleware",
			"NewCredentialConflictUnaryServerMiddleware", "NewCredentialConflictStreamServerMiddleware":
			t.Fatalf("forbidden middleware %s found in %v", middleware, chain)
		}
	}
	if len(actual) != len(expected) {
		t.Fatalf("authentication middleware = %v, want %v", actual, expected)
	}
	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf("authentication middleware = %v, want %v", actual, expected)
		}
	}
}
