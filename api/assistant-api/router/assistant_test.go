package assistant_router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"os"
)

func TestTalkRoutesAcceptRapidaClient(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "assistant.go", nil, 0)
	if err != nil {
		t.Fatalf("parse assistant.go: %v", err)
	}

	for _, name := range []string{"AssistantConversationApiRoute", "TalkApiRoute"} {
		if !functionHasParameter(file, name, "rapidaClient") {
			t.Fatalf("%s does not accept rapidaClient", name)
		}
	}
}

func TestAssistantRouteRegistersProjectDashboardAnalyticsService(t *testing.T) {
	contents, err := os.ReadFile("assistant.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "assistant.go", contents, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "RegisterProjectDashboardAnalyticsServiceServer" {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("ProjectDashboardAnalyticsService is not registered")
	}
}

func functionHasParameter(file *ast.File, functionName, parameterName string) bool {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != functionName {
			continue
		}
		for _, field := range function.Type.Params.List {
			for _, name := range field.Names {
				if name.Name == parameterName {
					return true
				}
			}
		}
	}
	return false
}
