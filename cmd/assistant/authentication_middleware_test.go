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

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/config"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
)

func TestSignalHandlerIsRegisteredBeforeServing(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
	var notifyPosition token.Pos
	var servePosition token.Pos
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "Notify":
			if notifyPosition == 0 {
				notifyPosition = call.Pos()
			}
		case "Serve":
			if servePosition == 0 {
				servePosition = call.Pos()
			}
		}
		return true
	})
	if notifyPosition == 0 || servePosition == 0 || notifyPosition > servePosition {
		t.Fatalf("signal Notify position = %d, first Serve position = %d; signal handling must be registered before serving", notifyPosition, servePosition)
	}
}

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
	if app.Clients.ProductUsage == nil {
		t.Fatal("Init() did not initialize ProductUsage")
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

func TestGRPCAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")

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

func TestGinAuthenticationMiddlewareOrder(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
	var middleware []string

	ast.Inspect(file, func(node ast.Node) bool {
		useCall, ok := node.(*ast.CallExpr)
		if !ok || callName(useCall) != "Use" {
			return true
		}
		for _, argument := range useCall.Args {
			if installed, ok := argument.(*ast.CallExpr); ok {
				middleware = append(middleware, callName(installed))
			}
		}
		return true
	})

	assertAuthenticationMiddlewareOrder(t, middleware, []string{
		"NewProjectHTTPMiddleware",
		"NewOrganizationHTTPMiddleware",
		"NewServiceHTTPMiddleware",
		"NewUserHTTPMiddleware",
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

func TestAssistantDerivesServiceAuthenticationKeyAtConsumers(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
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
	if resolverCalls != 3 || keyCalls != 3 {
		t.Fatalf("NewServicePrincipalResolver calls = %d, ServiceAuthenticationKey calls = %d, want 3 each", resolverCalls, keyCalls)
	}
}

func TestUsesRapidaAuthenticationClient(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
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
	if constructedClients != 0 || resolverCalls != 9 || sharedClientCalls != resolverCalls {
		t.Fatalf("constructed clients = %d, resolver calls = %d, shared client calls = %d, want 0, 9, 9", constructedClients, resolverCalls, sharedClientCalls)
	}
}

func TestAllRoutersPassesRapidaClientToKnowledgeRoutes(t *testing.T) {
	file := parseCommandSource(t, "assistant.go")
	wanted := map[string]bool{
		"KnowledgeApiRoute": false,
		"DocumentApiRoute":  false,
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		if _, ok := wanted[name]; !ok || len(call.Args) == 0 {
			return true
		}
		selector, ok := call.Args[len(call.Args)-1].(*ast.SelectorExpr)
		identifier, identifierOK := selector.X.(*ast.Ident)
		wanted[name] = ok && identifierOK && identifier.Name == "g" && selector.Sel.Name == "Clients"
		return true
	})

	for name, found := range wanted {
		if !found {
			t.Fatalf("%s does not receive g.Clients", name)
		}
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
		case "NewProjectHTTPMiddleware", "NewOrganizationHTTPMiddleware", "NewServiceHTTPMiddleware", "NewUserHTTPMiddleware",
			"NewProjectUnaryMiddleware", "NewOrganizationUnaryMiddleware", "NewServiceUnaryMiddleware", "NewUserUnaryMiddleware",
			"NewProjectStreamMiddleware", "NewOrganizationStreamMiddleware", "NewServiceStreamMiddleware", "NewUserStreamMiddleware":
			actual = append(actual, middleware)
		case "NewAuthenticationMiddleware", "NewProjectAuthenticatorMiddleware", "NewOrganizationAuthenticatorMiddleware", "NewServiceAuthenticatorMiddleware",
			"NewAuthenticationUnaryServerMiddleware", "NewProjectAuthenticatorUnaryServerMiddleware", "NewOrganizationAuthenticatorUnaryServerMiddleware", "NewServiceAuthenticatorUnaryServerMiddleware",
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
