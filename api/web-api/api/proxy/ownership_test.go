package web_proxy_api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProxyConstructorsDoNotCreateDownstreamClients(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	proxyDirectory := filepath.Dir(testFile)
	files := []string{"activity.go", "assistant.go", "assistant_deployment.go", "document.go", "endpoint.go", "invoke.go", "knowledge.go", "observability.go"}
	forbidden := map[string]struct{}{
		"NewAuditServiceClient":             {},
		"NewAssistantServiceClientGRPC":     {},
		"NewIndexerServiceClient":           {},
		"NewEndpointServiceClientGRPC":      {},
		"NewDeploymentServiceClientGRPC":    {},
		"NewKnowledgeServiceClientGRPC":     {},
		"NewObservabilityServiceClientGRPC": {},
	}

	for _, name := range files {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(proxyDirectory, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, found := forbidden[selector.Sel.Name]; found {
				t.Errorf("%s constructs downstream client %s", name, selector.Sel.Name)
			}
			return true
		})
	}
}
