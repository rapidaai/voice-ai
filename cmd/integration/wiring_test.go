package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGRPCAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "integration.go")

	assertMiddlewareOrder(t, interceptorChain(t, file, "ChainUnaryInterceptor"),
		"NewCredentialConflictUnaryServerMiddleware",
		"NewServiceAuthenticatorUnaryServerMiddleware",
		"NewProjectAuthenticatorUnaryServerMiddleware",
		"NewAuthenticationUnaryServerMiddleware",
	)
	assertMiddlewareOrder(t, interceptorChain(t, file, "ChainStreamInterceptor"),
		"NewCredentialConflictStreamServerMiddleware",
		"NewServiceAuthenticatorStreamServerMiddleware",
		"NewProjectAuthenticatorStreamServerMiddleware",
		"NewAuthenticationStreamServerMiddleware",
	)
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

func assertMiddlewareOrder(t *testing.T, chain []string, first string, followers ...string) {
	t.Helper()
	firstIndex := middlewareIndex(chain, first)
	if firstIndex < 0 {
		t.Fatalf("%s not found in %v", first, chain)
	}
	for _, follower := range followers {
		followerIndex := middlewareIndex(chain, follower)
		if followerIndex < 0 {
			t.Errorf("%s not found in %v", follower, chain)
		} else if firstIndex >= followerIndex {
			t.Errorf("%s must precede %s in %v", first, follower, chain)
		}
	}
}

func middlewareIndex(chain []string, middleware string) int {
	for index, candidate := range chain {
		if candidate == middleware {
			return index
		}
	}
	return -1
}
