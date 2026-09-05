package assistant_router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
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
