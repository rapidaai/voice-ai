// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_router

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestKnowledgeRoutesAcceptRapidaClient(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "knowledge.go", nil, 0)
	if err != nil {
		t.Fatalf("parse knowledge.go: %v", err)
	}

	for _, name := range []string{"KnowledgeApiRoute", "DocumentApiRoute"} {
		if !functionHasParameter(file, name, "rapidaClient") {
			t.Fatalf("%s does not accept rapidaClient", name)
		}
	}
}
