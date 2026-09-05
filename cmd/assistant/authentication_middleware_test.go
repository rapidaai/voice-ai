package main

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/config"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
)

func TestAppRunnerRetainsRapidaClient(t *testing.T) {
	clients := &rapida_client.RapidaClient{}
	app := AppRunner{Clients: clients}

	if app.Clients != clients {
		t.Fatal("AppRunner did not retain the Rapida client")
	}
}

func TestInitInitializesRapidaClient(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatalf("NewApplicationLogger() error = %v", err)
	}
	app := AppRunner{
		Cfg: &assistant_config.AssistantConfig{AppConfig: config.AppConfig{
			Web:         config.ServiceHostConfig{Host: "passthrough:///web-api"},
			Integration: config.ServiceHostConfig{Host: "passthrough:///integration-api"},
			Endpoint:    config.ServiceHostConfig{Host: "passthrough:///endpoint-api"},
			Document:    config.ServiceHostConfig{Host: "http://document-api"},
		}},
		Logger: logger,
	}

	if err := app.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if app.Clients == nil {
		t.Fatal("Init() did not initialize RapidaClient")
	}
	if len(app.Closeable) != 1 {
		t.Fatalf("Closeable count = %d, want 1", len(app.Closeable))
	}
	app.Close(context.Background())
}

func TestCloseUsesRegistrationOrder(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatalf("NewApplicationLogger() error = %v", err)
	}
	var closed []int
	app := AppRunner{
		Logger: logger,
		Closeable: []func(context.Context) error{
			func(context.Context) error { closed = append(closed, 1); return nil },
			func(context.Context) error { closed = append(closed, 2); return nil },
		},
	}

	app.Close(context.Background())
	if len(closed) != 2 || closed[0] != 1 || closed[1] != 2 {
		t.Fatalf("close order = %v, want [1 2]", closed)
	}
}

func TestGRPCUsesUserFirstAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")

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

func TestGinAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
	var middleware []string

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "AuthenticationMiddleware" {
			continue
		}
		for _, statement := range function.Body.List {
			expression, ok := statement.(*ast.ExprStmt)
			if !ok {
				continue
			}
			useCall, ok := expression.X.(*ast.CallExpr)
			if !ok || callName(useCall) != "Use" || len(useCall.Args) != 1 {
				continue
			}
			if installed, ok := useCall.Args[0].(*ast.CallExpr); ok {
				middleware = append(middleware, callName(installed))
			}
		}
	}

	assertAuthenticationMiddlewareOrder(t, middleware, []string{
		"NewAuthenticationMiddleware",
		"NewProjectAuthenticatorMiddleware",
		"NewOrganizationAuthenticatorMiddleware",
		"NewServiceAuthenticatorMiddleware",
	})
}

func TestInitDoesNotRegisterAuditActorCallbacks(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
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
		case "NewAuthenticationMiddleware", "NewProjectAuthenticatorMiddleware", "NewOrganizationAuthenticatorMiddleware", "NewServiceAuthenticatorMiddleware",
			"NewAuthenticationUnaryServerMiddleware", "NewProjectAuthenticatorUnaryServerMiddleware", "NewOrganizationAuthenticatorUnaryServerMiddleware", "NewServiceAuthenticatorUnaryServerMiddleware",
			"NewAuthenticationStreamServerMiddleware", "NewProjectAuthenticatorStreamServerMiddleware", "NewOrganizationAuthenticatorStreamServerMiddleware", "NewServiceAuthenticatorStreamServerMiddleware":
			actual = append(actual, middleware)
		case "NewAuthenticationBoundaryUnaryServerMiddleware", "NewAuthenticationBoundaryStreamServerMiddleware", "NewAuthenticationBoundaryMiddleware",
			"NewCredentialConflictUnaryServerMiddleware", "NewCredentialConflictStreamServerMiddleware", "NewCredentialConflictMiddleware":
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
