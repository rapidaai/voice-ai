package tests_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSecurityHookPackageSelection(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    []string
		packages []string
		exitCode string
	}{
		{
			name:     "mixed_packages",
			files:    []string{"production/code.go", "integration/suite_test.go", "unit/unit_test.go"},
			packages: []string{"./production", "./unit"},
		},
		{name: "integration_only", files: []string{"integration/suite_test.go"}},
		{name: "malformed_source", files: []string{"broken/code.go"}, packages: []string{"./broken"}, exitCode: "7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := t.TempDir()
			require.NoError(t, exec.Command("git", "init", "-q", repository).Run())
			for path, content := range map[string]string{
				"go.mod":                    "module hook.test\n\ngo 1.25.0\n",
				"production/code.go":        "package production\n",
				"integration/suite_test.go": "//go:build integration\n\npackage integration\n",
				"unit/unit_test.go":         "package unit\n",
				"broken/code.go":            "package broken\nfunc Broken(\n",
			} {
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(repository, path)), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(repository, path), []byte(content), 0o644))
			}
			fakeBin := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(fakeBin, "golangci-lint"), []byte(
				"#!/bin/sh\nprintf '%s\\n' 'LINTER' \"CGO_ENABLED=$CGO_ENABLED\" \"$@\"\nexit \"${TEST_LINT_EXIT_CODE:-0}\"\n"), 0o755))
			hook, err := filepath.Abs("../../../../../bin/pre-commit-go")
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "bash", append([]string{hook, "security"}, test.files...)...)
			command.Dir = repository
			command.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GOWORK=off", "GOPROXY=off", "GOLANGCI_LINT_CACHE="+t.TempDir(), "TEST_LINT_EXIT_CODE="+test.exitCode)
			output, err := command.CombinedOutput()
			if test.exitCode == "" {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
				require.Equal(t, 7, command.ProcessState.ExitCode())
			}
			if len(test.packages) == 0 {
				require.NotContains(t, string(output), "LINTER")
				require.Contains(t, string(output), "all Go files are excluded")
				return
			}
			require.Contains(t, string(output), "LINTER")
			require.Contains(t, string(output), "CGO_ENABLED=1")
			require.Contains(t, string(output), "--build-tags=nocgo")
			require.Contains(t, string(output), "--enable-only\ngosec")
			var packages []string
			for _, argument := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(argument, "./") {
					packages = append(packages, argument)
				}
			}
			require.Equal(t, test.packages, packages)
		})
	}
}
