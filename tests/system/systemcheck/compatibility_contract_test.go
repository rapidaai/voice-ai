package systemcheck

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCIWorkflowsDoNotCheckoutSubmodules(t *testing.T) {
	root := repositoryRoot(t)
	for _, relative := range []string{
		".github/workflows/reusable-assistant-native-ci.yml",
		".github/workflows/reusable-system-ci.yml",
	} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "submodules:") {
			t.Errorf("%s still configures submodule checkout", relative)
		}
	}
}

func TestMaterializeBaselineUsesTrackedGeneratedProtobufFiles(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init")
	runGit(t, repository, "config", "user.email", "ci@example.test")
	runGit(t, repository, "config", "user.name", "CI")
	writeContractFixture(t, repository, "base")
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-m", "base")
	baseSHA := strings.TrimSpace(runGit(t, repository, "rev-parse", "HEAD"))

	writeContractFixture(t, repository, "head")
	writeFile(t, filepath.Join(repository, "protos", "nested", "ignored.go"), "package ignored\n")
	runGit(t, repository, "add", ".")
	runGit(t, repository, "commit", "-m", "head")

	baseline := filepath.Join(t.TempDir(), "baseline")
	output := filepath.Join(t.TempDir(), "github-output")
	summary := filepath.Join(t.TempDir(), "summary")
	command := exec.Command("bash", filepath.Join(repositoryRoot(t), "scripts/contracts/materialize-baseline.sh"), baseline)
	command.Dir = repository
	command.Env = append(os.Environ(),
		"EVENT_NAME=pull_request",
		"PULL_REQUEST_BASE_SHA="+baseSHA,
		"GITHUB_OUTPUT="+output,
		"GITHUB_STEP_SUMMARY="+summary,
	)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("materialize baseline: %v\n%s", err, combined)
	}

	for _, relative := range []string{
		"go.mod",
		"go.sum",
		"docker-compose.yml",
		"openapi/artifacts/assistant-api.yaml",
		"openapi/artifacts/talk-api.yaml",
		"openapi/artifacts/common.yaml",
		"protos/example.pb.go",
	} {
		data, err := os.ReadFile(filepath.Join(baseline, relative))
		if err != nil {
			t.Fatalf("read materialized %s: %v", relative, err)
		}
		if !strings.Contains(string(data), "base") {
			t.Errorf("materialized %s does not come from base commit", relative)
		}
	}
	if _, err := os.Stat(filepath.Join(baseline, "protos", "nested", "ignored.go")); !os.IsNotExist(err) {
		t.Fatalf("nested non-contract Go file was materialized: %v", err)
	}
	assertFileContains(t, output, "directory="+baseline)
	assertFileContains(t, output, "target-sha="+baseSHA)
	assertFileContains(t, summary, "Target commit: `"+baseSHA+"`")
}

func TestCompatibilityScriptUsesDescriptorImagesAndRecordsHashes(t *testing.T) {
	fixture, logPath, summaryPath := compatibilityFixture(t, true)
	command := exec.Command("bash", filepath.Join(repositoryRoot(t), "scripts/contracts/check-compatibility.sh"), filepath.Join(fixture, "baseline"))
	command.Dir = fixture
	command.Env = append(os.Environ(), "PATH="+filepath.Join(fixture, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summaryPath)
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("check compatibility: %v\n%s", err, combined)
	}

	log := readFile(t, logPath)
	if !strings.Contains(log, "buf breaking ") || !strings.Contains(log, ".binpb --against ") {
		t.Fatalf("protobuf comparison did not use descriptor images:\n%s", log)
	}
	currentHash := fmt.Sprintf("%x", sha256.Sum256([]byte("current-descriptor\n")))
	baselineHash := fmt.Sprintf("%x", sha256.Sum256([]byte("baseline-descriptor\n")))
	assertFileContains(t, summaryPath, "Current descriptor SHA-256: `"+currentHash+"`")
	assertFileContains(t, summaryPath, "Baseline descriptor SHA-256: `"+baselineHash+"`")
}

func TestCompatibilityScriptFailsWhenDescriptorGenerationProducesNoImage(t *testing.T) {
	fixture, _, summaryPath := compatibilityFixture(t, false)
	command := exec.Command("bash", filepath.Join(repositoryRoot(t), "scripts/contracts/check-compatibility.sh"), filepath.Join(fixture, "baseline"))
	command.Dir = fixture
	command.Env = append(os.Environ(), "PATH="+filepath.Join(fixture, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summaryPath)
	if err := command.Run(); err == nil {
		t.Fatal("expected missing descriptor image to fail closed")
	}
}

func TestCompatibilityScriptRecordsDescriptorHashesBeforeBreakingFailure(t *testing.T) {
	fixture, _, summaryPath := compatibilityFixture(t, true)
	writeExecutable(t, filepath.Join(fixture, "tests", "system", "bin", "buf"), "#!/bin/sh\nexit 100\n")
	command := exec.Command("bash", filepath.Join(repositoryRoot(t), "scripts/contracts/check-compatibility.sh"), filepath.Join(fixture, "baseline"))
	command.Dir = fixture
	command.Env = append(os.Environ(), "PATH="+filepath.Join(fixture, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summaryPath)
	if err := command.Run(); err == nil {
		t.Fatal("expected breaking-change detection to fail")
	}

	currentHash := fmt.Sprintf("%x", sha256.Sum256([]byte("current-descriptor\n")))
	baselineHash := fmt.Sprintf("%x", sha256.Sum256([]byte("baseline-descriptor\n")))
	assertFileContains(t, summaryPath, "Current descriptor SHA-256: `"+currentHash+"`")
	assertFileContains(t, summaryPath, "Baseline descriptor SHA-256: `"+baselineHash+"`")
}

func compatibilityFixture(t *testing.T, writeDescriptors bool) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	baseline := filepath.Join(root, "baseline")
	for _, directory := range []string{
		filepath.Join(root, "openapi", "artifacts"),
		filepath.Join(root, "protos"),
		filepath.Join(root, "tests", "system", "cmd", "protobuf-descriptor"),
		filepath.Join(root, "tests", "system", "bin"),
		filepath.Join(baseline, "openapi", "artifacts"),
		filepath.Join(baseline, "protos"),
		filepath.Join(root, "bin"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, moduleRoot := range []string{root, baseline} {
		writeFile(t, filepath.Join(moduleRoot, "go.mod"), "module github.com/rapidaai\n")
		writeFile(t, filepath.Join(moduleRoot, "go.sum"), "fixture\n")
		writeFile(t, filepath.Join(moduleRoot, "protos", "example.pb.go"), "package protos\n")
		for _, name := range []string{"assistant-api.yaml", "talk-api.yaml", "common.yaml"} {
			writeFile(t, filepath.Join(moduleRoot, "openapi", "artifacts", name), "openapi: 3.0.3\n")
		}
	}
	writeFile(t, filepath.Join(root, "tests", "system", "cmd", "protobuf-descriptor", "main.go"), "package main\n")

	logPath := filepath.Join(root, "commands.log")
	descriptorWrite := ""
	if writeDescriptors {
		descriptorWrite = `
output=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output" ]; then output=$2; shift 2; else shift; fi
done
case "$PWD" in
  */baseline) printf '%s\n' baseline-descriptor > "$output" ;;
  *) printf '%s\n' current-descriptor > "$output" ;;
esac`
	}
	writeExecutable(t, filepath.Join(root, "bin", "go"), `#!/bin/sh
set -eu
printf 'go %s\n' "$*" >> "$COMMAND_LOG"
case "$*" in
  *protobuf-descriptor/main.go*)`+descriptorWrite+` ;;
esac
`)
	writeExecutable(t, filepath.Join(root, "bin", "python3"), "#!/bin/sh\nset -eu\nprintf 'python3 %s\\n' \"$*\" >> \"$COMMAND_LOG\"\n")
	writeExecutable(t, filepath.Join(root, "tests", "system", "bin", "oasdiff"), "#!/bin/sh\nset -eu\nprintf 'oasdiff %s\\n' \"$*\" >> \"$COMMAND_LOG\"\n")
	writeExecutable(t, filepath.Join(root, "tests", "system", "bin", "buf"), "#!/bin/sh\nset -eu\nprintf 'buf %s\\n' \"$*\" >> \"$COMMAND_LOG\"\n")
	t.Setenv("COMMAND_LOG", logPath)
	return root, logPath, filepath.Join(root, "summary")
}

func writeContractFixture(t *testing.T, root, marker string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.test/"+marker+"\n")
	writeFile(t, filepath.Join(root, "go.sum"), marker+"\n")
	writeFile(t, filepath.Join(root, "docker-compose.yml"), "# "+marker+"\n")
	writeFile(t, filepath.Join(root, "protos", "example.pb.go"), "// "+marker+"\npackage protos\n")
	for _, name := range []string{"assistant-api.yaml", "talk-api.yaml", "common.yaml"} {
		writeFile(t, filepath.Join(root, "openapi", "artifacts", name), "# "+marker+"\n")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	writeFile(t, path, content)
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertFileContains(t *testing.T, path, expected string) {
	t.Helper()
	if content := readFile(t, path); !strings.Contains(content, expected) {
		t.Fatalf("%s does not contain %q:\n%s", path, expected, content)
	}
}
