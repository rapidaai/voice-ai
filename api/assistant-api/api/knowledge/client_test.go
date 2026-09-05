// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package knowledge_api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	rapida_client "github.com/rapidaai/pkg/clients/rapida"
)

func TestControllersRetainRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}

	knowledge := &knowledgeApi{rapidaClient: client}
	if knowledge.rapidaClient != client {
		t.Fatal("knowledge controller did not retain RapidaClient")
	}

	indexer := &indexerApi{rapidaClient: client}
	if indexer.rapidaClient != client {
		t.Fatal("document controller did not retain RapidaClient")
	}
}

func TestControllersDoNotConstructIndexerClients(t *testing.T) {
	for _, filename := range []string{"knowledge.go", "indexer.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "NewIndexerServiceClient" {
				t.Fatalf("%s constructs an indexer client", filename)
			}
			return true
		})
	}
}
