package systemcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMetadataCommandWritesRequiredImageIDs(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "buildkit-metadata.json")
	composeImagesPath := filepath.Join(directory, "compose-images.json")
	images := testComposeImageMapping("ci-project")
	images["endpoint-api"] = "registry.example.test/rapida/endpoint-api:rendered-change"
	writeComposeImageMapping(t, composeImagesPath, images)
	var inspected []string
	runner := func(_ context.Context, name string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		if name != "docker" || len(args) != 5 || args[0] != "image" || args[1] != "inspect" || args[2] != "--format" || args[3] != "{{.Id}}" {
			t.Fatalf("unexpected inspect command: %s %#v", name, args)
		}
		inspected = append(inspected, args[4])
		_, _ = fmt.Fprintf(stdout, "sha256:%064x\n", len(inspected))
		return nil
	}
	if err := writeBuildMetadata(context.Background(), runner, composeImagesPath, output, "ci-builder-123", "system-cache-abc"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var metadata buildMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if err := validateBuildMetadata(metadata); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Images) != len(builtServices) {
		t.Fatalf("images=%d want=%d", len(metadata.Images), len(builtServices))
	}
	for index, service := range builtServices {
		reference := images[service]
		if inspected[index] != reference || metadata.Images[index].Service != service || metadata.Images[index].ImageRef != reference {
			t.Fatalf("image %d inspected=%q metadata=%#v expected service=%q reference=%q", index, inspected[index], metadata.Images[index], service, reference)
		}
	}
}

func TestBuildMetadataRejectsMissingExtraAndMalformedServices(t *testing.T) {
	tests := map[string]string{
		"missing":                     `{"web-api":"ci-web-api","integration-api":"ci-integration-api","endpoint-api":"ci-endpoint-api","assistant-api":"ci-assistant-api","ui":"ci-ui"}`,
		"extra":                       `{"web-api":"ci-web-api","integration-api":"ci-integration-api","endpoint-api":"ci-endpoint-api","assistant-api":"ci-assistant-api","ui":"ci-ui","test-runner":"rapida-system-test-runner:ci","postgres":"postgres:15"}`,
		"non-string":                  `{"web-api":12,"integration-api":"ci-integration-api","endpoint-api":"ci-endpoint-api","assistant-api":"ci-assistant-api","ui":"ci-ui","test-runner":"rapida-system-test-runner:ci"}`,
		"empty-reference":             `{"web-api":"","integration-api":"ci-integration-api","endpoint-api":"ci-endpoint-api","assistant-api":"ci-assistant-api","ui":"ci-ui","test-runner":"rapida-system-test-runner:ci"}`,
		"duplicate-after-normalizing": `{"web-api":"ci-web-api"," web-api ":"other","integration-api":"ci-integration-api","endpoint-api":"ci-endpoint-api","assistant-api":"ci-assistant-api","ui":"ci-ui","test-runner":"rapida-system-test-runner:ci"}`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "compose-images.json")
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			if err := writeBuildMetadata(context.Background(), nil, path, filepath.Join(t.TempDir(), "metadata.json"), "builder", "cache"); err == nil {
				t.Fatal("expected malformed Compose image mapping to fail")
			}
		})
	}
}

func TestBuildMetadataRejectsMissingInspectedImage(t *testing.T) {
	directory := t.TempDir()
	composeImagesPath := filepath.Join(directory, "compose-images.json")
	writeComposeImageMapping(t, composeImagesPath, testComposeImageMapping("ci-project"))
	runner := func(_ context.Context, _ string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		if args[4] == "ci-project-endpoint-api" {
			return fmt.Errorf("image not found")
		}
		_, _ = io.WriteString(stdout, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n")
		return nil
	}
	output := filepath.Join(directory, "metadata.json")
	if err := writeBuildMetadata(context.Background(), runner, composeImagesPath, output, "builder", "cache"); err == nil {
		t.Fatal("expected missing service failure")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("partial metadata output exists: %v", err)
	}
}

func TestBuildMetadataRejectsIncompleteInspectOutput(t *testing.T) {
	directory := t.TempDir()
	composeImagesPath := filepath.Join(directory, "compose-images.json")
	writeComposeImageMapping(t, composeImagesPath, testComposeImageMapping("ci-project"))
	runner := func(_ context.Context, _ string, _ []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		_, _ = io.WriteString(stdout, "not-an-image-id\n")
		return nil
	}
	if err := writeBuildMetadata(context.Background(), runner, composeImagesPath, filepath.Join(directory, "metadata.json"), "builder", "cache"); err == nil {
		t.Fatal("expected incomplete image metadata failure")
	}
}

func TestBuildMetadataCLIUsesWorkflowArguments(t *testing.T) {
	directory := t.TempDir()
	dockerPath := filepath.Join(directory, "docker")
	script := "#!/bin/sh\nset -eu\ntest \"$1 $2 $3\" = 'image inspect --format'\nprintf '%s\\n' 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'\n"
	if err := os.WriteFile(dockerPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BUILDX_BUILDER", "ci-builder")
	t.Setenv("SYSTEM_CACHE_SCOPE", "cache-scope")
	output := filepath.Join(directory, "metadata.json")
	composeImagesPath := filepath.Join(directory, "compose-images.json")
	writeComposeImageMapping(t, composeImagesPath, testComposeImageMapping("ci-project"))
	if err := Execute(context.Background(), []string{"build-metadata", "--compose-images", composeImagesPath, "--output", output}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
}

func testComposeImageMapping(project string) map[string]string {
	return map[string]string{
		"web-api":         project + "-web-api",
		"integration-api": project + "-integration-api",
		"endpoint-api":    project + "-endpoint-api",
		"assistant-api":   project + "-assistant-api",
		"ui":              project + "-ui",
		"test-runner":     "rapida-system-test-runner:ci",
	}
}

func writeComposeImageMapping(t *testing.T, path string, images map[string]string) {
	t.Helper()
	data, err := json.Marshal(images)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptedCLIRejectsLegacyAndIncompleteForms(t *testing.T) {
	tests := [][]string{
		{"migrations", "--report", "/tmp/report.json"},
		{"health", "--deadline", "60s"},
		{"health", "--timeout-per-service", "60s", "--interval", "1s", "--readiness-key", readinessPostgresKey},
		{"ui"},
		{"ui-nginx", "--base-url", "http://nginx:8080", "--require-spa-root", "--require-hashed-asset"},
		{"collect-diagnostics"},
		{"cleanup", "--compose-project", "ci", "--retries", "0", "--interval", "1s"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stderr bytes.Buffer
			if err := Execute(context.Background(), args, io.Discard, &stderr); err == nil {
				t.Fatalf("expected %q to fail", args)
			}
		})
	}
}

func TestRunnerDoesNotPrefixSystemcheckEntrypoint(t *testing.T) {
	path := filepath.Join("..", "test-runner", "Dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`ENTRYPOINT ["systemcheck"]`)) {
		t.Fatal("RFC command examples would become systemcheck systemcheck")
	}
	if !bytes.Contains(data, []byte(`CMD ["systemcheck"]`)) {
		t.Fatal("runner default command must remain systemcheck")
	}
	if !bytes.Contains(data, []byte("/usr/local/bin/systemcheck")) {
		t.Fatal("systemcheck binary is not installed in the runner")
	}
}
