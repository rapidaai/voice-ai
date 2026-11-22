package main

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	web_config "github.com/rapidaai/api/web-api/config"
	app_config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
)

func TestSignalHandlerIsRegisteredBeforeServing(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	var notifyPosition token.Pos
	var servePosition token.Pos
	var newMuxCalls int
	var shutdownCalls int
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callName(call) {
		case "NewMux":
			newMuxCalls++
		case "Notify":
			if notifyPosition == 0 {
				notifyPosition = call.Pos()
			}
		case "Serve":
			if servePosition == 0 {
				servePosition = call.Pos()
			}
		case "Shutdown":
			shutdownCalls++
		}
		return true
	})
	if notifyPosition == 0 || servePosition == 0 || notifyPosition > servePosition {
		t.Fatalf("signal Notify position = %d, first Serve position = %d; signal handling must be registered before serving", notifyPosition, servePosition)
	}
	if newMuxCalls != 1 || shutdownCalls != 1 {
		t.Fatalf("NewMux calls = %d, Shutdown calls = %d, want 1 each", newMuxCalls, shutdownCalls)
	}
}

func TestInitInitializesRapidaClient(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	app := AppRunner{
		Cfg: &web_config.WebAppConfig{AppConfig: app_config.AppConfig{
			Web:         app_config.ServiceHostConfig{Host: "passthrough:///web-api"},
			Integration: app_config.ServiceHostConfig{Host: "passthrough:///integration-api"},
			Endpoint:    app_config.ServiceHostConfig{Host: "passthrough:///endpoint-api"},
			Assistant:   app_config.ServiceHostConfig{Host: "passthrough:///assistant-api"},
			Document:    app_config.ServiceHostConfig{Host: "http://document-api"},
		}},
		Logger: logger,
	}

	if err := app.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if app.Clients == nil || app.Clients.Assistant == nil || app.Clients.Endpoint == nil || app.Clients.Audit == nil {
		t.Fatal("Init() did not initialize the Rapida proxy clients")
	}
	if len(app.Closeable) != 1 {
		t.Fatalf("Closeable count = %d, want 1", len(app.Closeable))
	}
	app.Close(context.Background())
}

func TestCORSAllowsAuthenticationSelectionHeaders(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("cors-test"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &AppRunner{E: gin.New(), Logger: logger}
	runner.CorsMiddleware()
	runner.E.GET("/protected", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodOptions, "/protected", nil)
	request.Header.Set("Origin", "https://console.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "x-auth-id,x-organization-id,x-project-id,x-api-key,x-org-key,x-internal-service-key")
	response := httptest.NewRecorder()
	runner.E.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", response.Code, http.StatusNoContent)
	}
	allowed := strings.ToLower(response.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"x-auth-id", "x-organization-id", "x-project-id", "x-api-key", "x-org-key", "x-internal-service-key"} {
		if !strings.Contains(allowed, header) {
			t.Errorf("Access-Control-Allow-Headers = %q, missing %q", allowed, header)
		}
	}
}

func TestAuthenticationMiddlewareIsNotAppRunnerState(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			for _, specification := range declaration.Specs {
				typeSpecification, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpecification.Name.Name != "AppRunner" {
					continue
				}
				structure := typeSpecification.Type.(*ast.StructType)
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						if name.Name == "HTTPAuthentication" || name.Name == "GRPCAuthentication" {
							t.Fatalf("authentication middleware stored on AppRunner field %s", name.Name)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if declaration.Name.Name == "InitializePrincipalAuthentication" {
				t.Fatal("authentication composition remains an AppRunner lifecycle method")
			}
		}
	}
}

func TestGRPCUsesPrincipalAuthenticationInterceptors(t *testing.T) {
	file := parseCommandSource(t, "web.go")

	assertAuthenticationChain(t, interceptorChain(t, file, "ChainUnaryInterceptor"), []string{
		"NewRequestLoggerUnaryServerMiddleware",
		"NewRecoveryUnaryServerMiddleware",
		"NewProjectUnaryMiddleware",
		"NewOrganizationUnaryMiddleware",
		"NewUserUnaryMiddleware",
		"NewServiceUnaryMiddleware",
		"NewPolicyUnaryMiddleware",
	})
	assertAuthenticationChain(t, interceptorChain(t, file, "ChainStreamInterceptor"), []string{
		"NewRequestLoggerStreamServerMiddleware",
		"NewRecoveryStreamServerMiddleware",
		"NewProjectStreamMiddleware",
		"NewOrganizationStreamMiddleware",
		"NewUserStreamMiddleware",
		"NewServiceStreamMiddleware",
		"NewPolicyStreamMiddleware",
	})
}

func TestWebDerivesAuthenticationKeysAtConsumers(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	expected := map[string]struct {
		argument int
		key      string
		uses     int
	}{
		"NewOrganizationResolver":     {argument: 0, key: "OrganizationCredentialFingerprintKey", uses: 3},
		"NewServicePrincipalResolver": {argument: 1, key: "ServiceAuthenticationKey", uses: 3},
		"AllRouters":                  {argument: 0, key: "OrganizationCredentialFingerprintKey", uses: 1},
	}
	consumerCalls := map[string]int{}
	keyCalls := map[string]int{}
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
		name := callName(call)
		switch name {
		case "ServiceAuthenticationKey", "OrganizationCredentialFingerprintKey":
			keyCalls[name]++
		}
		consumer, ok := expected[name]
		if !ok {
			return true
		}
		if len(call.Args) <= consumer.argument {
			t.Fatalf("%s has no key argument", name)
		}
		keyCall, ok := call.Args[consumer.argument].(*ast.CallExpr)
		if !ok || callName(keyCall) != consumer.key {
			t.Fatalf("%s key is not a direct %s call", name, consumer.key)
		}
		consumerCalls[name]++
		return true
	})
	for name, consumer := range expected {
		if consumerCalls[name] != consumer.uses {
			t.Fatalf("%s direct key uses = %d, want %d", name, consumerCalls[name], consumer.uses)
		}
	}
	if keyCalls["ServiceAuthenticationKey"] != 3 || keyCalls["OrganizationCredentialFingerprintKey"] != 4 {
		t.Fatalf("authentication key calls = %v", keyCalls)
	}
}

func TestWebConstructsResolverPerTransportMiddleware(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	resolverForMiddleware := map[string]string{
		"NewProjectHTTPMiddleware":        "NewProjectResolver",
		"NewProjectUnaryMiddleware":       "NewProjectResolver",
		"NewProjectStreamMiddleware":      "NewProjectResolver",
		"NewOrganizationHTTPMiddleware":   "NewOrganizationResolver",
		"NewOrganizationUnaryMiddleware":  "NewOrganizationResolver",
		"NewOrganizationStreamMiddleware": "NewOrganizationResolver",
		"NewUserHTTPMiddleware":           "NewUserResolver",
		"NewUserUnaryMiddleware":          "NewUserResolver",
		"NewUserStreamMiddleware":         "NewUserResolver",
		"NewServiceHTTPMiddleware":        "NewServicePrincipalResolver",
		"NewServiceUnaryMiddleware":       "NewServicePrincipalResolver",
		"NewServiceStreamMiddleware":      "NewServicePrincipalResolver",
	}
	constructorCalls := map[string]int{}
	middlewareCalls := map[string]int{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		switch name {
		case "NewProjectResolver", "NewOrganizationResolver", "NewUserResolver", "NewServicePrincipalResolver":
			constructorCalls[name]++
		}
		expectedResolver, isAuthenticationMiddleware := resolverForMiddleware[name]
		if !isAuthenticationMiddleware {
			return true
		}
		if len(call.Args) == 0 {
			t.Fatalf("%s has no resolver argument", name)
		}
		resolverCall, ok := call.Args[0].(*ast.CallExpr)
		if !ok || callName(resolverCall) != expectedResolver {
			t.Fatalf("%s resolver is not a direct %s call", name, expectedResolver)
		}
		middlewareCalls[expectedResolver]++
		return true
	})
	for _, constructor := range []string{"NewProjectResolver", "NewOrganizationResolver", "NewUserResolver", "NewServicePrincipalResolver"} {
		if constructorCalls[constructor] != 3 || middlewareCalls[constructor] != 3 {
			t.Fatalf("%s calls = %d, direct middleware uses = %d, want one per HTTP, unary, and stream middleware", constructor, constructorCalls[constructor], middlewareCalls[constructor])
		}
	}
}

func TestInitDoesNotRegisterAuditActorCallbacks(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && callName(call) == "RegisterAuditActorCallbacks" {
			t.Fatal("Init registers removed audit actor callbacks")
		}
		return true
	})
}

func TestHTTPUsesPrincipalAuthenticationMiddleware(t *testing.T) {
	file := parseCommandSource(t, "web.go")
	var chains [][]string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "main" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && callName(call) == "AllMiddlewares" {
				chain := make([]string, 0, len(call.Args))
				for _, argument := range call.Args {
					middleware, ok := argument.(*ast.CallExpr)
					if ok {
						chain = append(chain, callName(middleware))
					}
				}
				chains = append(chains, chain)
			}
			return true
		})
	}
	if len(chains) != 1 {
		t.Fatalf("AllMiddlewares calls = %d, want 1", len(chains))
	}
	assertAuthenticationChain(t, chains[0], []string{"NewProjectHTTPMiddleware", "NewOrganizationHTTPMiddleware", "NewUserHTTPMiddleware", "NewServiceHTTPMiddleware", "NewPolicyHTTPMiddleware"})
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

func assertAuthenticationChain(t *testing.T, chain []string, expected []string) {
	t.Helper()
	if len(chain) != len(expected) {
		t.Fatalf("authentication chain = %v, want %v", chain, expected)
	}
	for index := range expected {
		if chain[index] != expected[index] {
			t.Fatalf("authentication chain = %v, want %v", chain, expected)
		}
	}
}
