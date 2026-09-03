// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSIPLifecycle_NoDirectSessionMutationOutsideOwner(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	requireRuntimeCaller(t, ok)

	coreDir := filepath.Dir(file)
	assistantDir := filepath.Clean(filepath.Join(coreDir, "..", ".."))
	roots := []string{
		filepath.Join(assistantDir, "sip"),
		filepath.Join(assistantDir, "internal", "channel", "telephony", "internal", "sip"),
	}

	allowed := map[string]bool{
		filepath.Join(coreDir, "server_lifecycle.go"): true,
		filepath.Join(coreDir, "session.go"):          true,
	}
	var violations []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || allowed[path] {
				return nil
			}
			found, err := directLifecycleMutations(path)
			if err != nil {
				return err
			}
			violations = append(violations, found...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("direct SIP lifecycle mutation found outside lifecycle owner:\n%s", strings.Join(violations, "\n"))
	}
}

func TestSIPInboundSetup_NoDirectSessionRemoval(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	requireRuntimeCaller(t, ok)

	coreDir := filepath.Dir(file)
	paths := []string{
		filepath.Join(coreDir, "server_inbound.go"),
		filepath.Join(coreDir, "inbound_call.go"),
	}

	var violations []string
	for _, path := range paths {
		found, err := directRemoveSessionCalls(path)
		if err != nil {
			t.Fatal(err)
		}
		violations = append(violations, found...)
	}
	if len(violations) > 0 {
		t.Fatalf("direct inbound setup session removal found outside lifecycle owner:\n%s", strings.Join(violations, "\n"))
	}
}

func requireRuntimeCaller(t *testing.T, ok bool) {
	t.Helper()
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
}

func directRemoveSessionCalls(path string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var violations []string
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		callExpression, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := callExpression.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "removeSession" {
			return true
		}
		position := fileSet.Position(selector.Pos())
		violations = append(violations, formatLifecycleViolation(position.Filename, position.Line, selector.Sel.Name))
		return true
	})
	return violations, nil
}

func directLifecycleMutations(path string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var violations []string
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		callExpression, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := callExpression.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selector.Sel.Name != "End" && selector.Sel.Name != "SetState" {
			return true
		}
		position := fileSet.Position(selector.Pos())
		violations = append(violations, formatLifecycleViolation(position.Filename, position.Line, selector.Sel.Name))
		return true
	})
	return violations, nil
}

func formatLifecycleViolation(path string, lineNumber int, method string) string {
	return path + ":" + strconv.Itoa(lineNumber) + ": direct " + method + " call"
}
