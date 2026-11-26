package main

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	endpoint_config "github.com/rapidaai/api/endpoint-api/config"
	app_config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
)

func TestInitInitializesRapidaClient(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	app := AppRunner{
		Cfg: &endpoint_config.EndpointConfig{AppConfig: app_config.AppConfig{
			Web:         app_config.ServiceHostConfig{Host: "passthrough:///web-api"},
			Integration: app_config.ServiceHostConfig{Host: "passthrough:///integration-api"},
			Endpoint:    app_config.ServiceHostConfig{Host: "passthrough:///endpoint-api"},
			Document:    app_config.ServiceHostConfig{Host: "http://document-api"},
		}},
		Logger: logger,
	}

	if err := app.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if app.Clients == nil || app.Clients.Authentication == nil {
		t.Fatal("Init() did not initialize the Rapida authentication client")
	}
	if len(app.Closeable) != 1 {
		t.Fatalf("Closeable count = %d, want 1", len(app.Closeable))
	}
	app.Close(context.Background())
}

func TestGRPCUsesPrincipalAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "endpoint.go")

	assertAuthenticationMiddlewareOrder(t, interceptorChain(t, file, "ChainUnaryInterceptor"), []string{
		"NewProjectUnaryMiddleware",
		"NewOrganizationUnaryMiddleware",
		"NewServiceUnaryMiddleware",
		"NewUserUnaryMiddleware",
	})
	assertAuthenticationMiddlewareOrder(t, interceptorChain(t, file, "ChainStreamInterceptor"), []string{
		"NewProjectStreamMiddleware",
		"NewOrganizationStreamMiddleware",
		"NewServiceStreamMiddleware",
		"NewUserStreamMiddleware",
	})
}

func TestInitDoesNotRegisterAuditActorCallbacks(t *testing.T) {
	file := parseCommandSource(t, "endpoint.go")
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && callName(call) == "RegisterAuditActorCallbacks" {
			t.Fatal("Init registers removed audit actor callbacks")
		}
		return true
	})
}

func TestEndpointDerivesServiceAuthenticationKeyAtConsumers(t *testing.T) {
	file := parseCommandSource(t, "endpoint.go")
	resolverCalls := 0
	keyCalls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok {
			value := strings.ToLower(literal.Value)
			if strings.Contains(value, "root") && strings.Contains(value, "fingerprint") {
				t.Fatal("root-secret fingerprint logging restored")
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "ServiceAuthenticationKey":
			keyCalls++
		case "NewServicePrincipalResolver":
			resolverCalls++
			if len(call.Args) != 2 {
				t.Fatalf("NewServicePrincipalResolver args = %d, want 2", len(call.Args))
			}
			keyCall, ok := call.Args[1].(*ast.CallExpr)
			if !ok || callName(keyCall) != "ServiceAuthenticationKey" {
				t.Fatal("NewServicePrincipalResolver key is not a direct ServiceAuthenticationKey call")
			}
		}
		return true
	})
	if resolverCalls != 2 || keyCalls != 2 {
		t.Fatalf("NewServicePrincipalResolver calls = %d, ServiceAuthenticationKey calls = %d, want 2 each", resolverCalls, keyCalls)
	}
}

func TestUsesRapidaAuthenticationClient(t *testing.T) {
	file := parseCommandSource(t, "endpoint.go")
	constructedClients := 0
	resolverCalls := 0
	sharedClientCalls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "NewAuthenticationClient":
			constructedClients++
		case "NewAuthenticationResolverClient":
			t.Fatal("removed authentication resolver client is constructed")
		case "NewProjectPrincipalResolver", "NewOrganizationPrincipalResolver", "NewUserPrincipalResolver":
			resolverCalls++
			if len(call.Args) != 1 {
				return true
			}
			authentication, ok := call.Args[0].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			clients, clientsOK := authentication.X.(*ast.SelectorExpr)
			if !clientsOK {
				return true
			}
			appRunner, appRunnerOK := clients.X.(*ast.Ident)
			if appRunnerOK && authentication.Sel.Name == "Authentication" && clients.Sel.Name == "Clients" && appRunner.Name == "appRunner" {
				sharedClientCalls++
			}
		}
		return true
	})
	if constructedClients != 0 || resolverCalls != 6 || sharedClientCalls != resolverCalls {
		t.Fatalf("constructed clients = %d, resolver calls = %d, shared client calls = %d, want 0, 6, 6", constructedClients, resolverCalls, sharedClientCalls)
	}
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
		case "NewProjectUnaryMiddleware", "NewOrganizationUnaryMiddleware", "NewServiceUnaryMiddleware", "NewUserUnaryMiddleware",
			"NewProjectStreamMiddleware", "NewOrganizationStreamMiddleware", "NewServiceStreamMiddleware", "NewUserStreamMiddleware":
			actual = append(actual, middleware)
		case "NewAuthenticationUnaryServerMiddleware", "NewProjectAuthenticatorUnaryServerMiddleware", "NewOrganizationAuthenticatorUnaryServerMiddleware", "NewServiceAuthenticatorUnaryServerMiddleware",
			"NewAuthenticationStreamServerMiddleware", "NewProjectAuthenticatorStreamServerMiddleware", "NewOrganizationAuthenticatorStreamServerMiddleware", "NewServiceAuthenticatorStreamServerMiddleware":
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
