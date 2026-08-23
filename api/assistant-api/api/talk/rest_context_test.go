package assistant_talk_api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutboundRestPropagatesRequestContext(t *testing.T) {
	t.Helper()

	for _, filename := range []string{"outbound_call_rest.go", "outbound_bulk_call_rest.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		require.NoError(t, err)

		checkedCalls := 0
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Run" && selector.Sel.Name != "Record" && selector.Sel.Name != "Observability") {
				return true
			}
			checkedCalls++
			identifier, ok := call.Args[0].(*ast.Ident)
			require.Falsef(t, ok && identifier.Name == "c", "%s passes gin.Context to %s", filename, selector.Sel.Name)
			return true
		})

		require.Positive(t, checkedCalls)
	}
}
