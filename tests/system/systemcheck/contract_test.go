package systemcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestComposeContract(t *testing.T) {
	baseServices := map[string]any{}
	for _, name := range inheritedServices {
		service := map[string]any{
			"container_name": "fixed-" + name,
			"ports":          []any{map[string]any{"published": "1234", "target": float64(1234)}},
			"networks":       map[string]any{"api-network": nil},
			"healthcheck":    map[string]any{"test": []any{"CMD", "true"}},
		}
		if contains([]string{"ui", "web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
			service["build"] = map[string]any{"context": "/repo", "dockerfile": "docker/" + name + "/Dockerfile"}
		}
		switch name {
		case "postgres":
			service["volumes"] = []any{
				map[string]any{"type": "bind", "source": "/home/user/db", "target": "/var/lib/postgresql/data"},
				map[string]any{"type": "bind", "source": "/repo/docker/postgres/init.sql", "target": "/docker-entrypoint-initdb.d/init.sql"},
			}
		case "redis":
			service["volumes"] = []any{map[string]any{"type": "bind", "source": "/home/user/redis", "target": "/data"}}
		case "nginx", "web-api", "integration-api", "endpoint-api", "assistant-api":
			service["volumes"] = []any{
				map[string]any{"type": "bind", "source": "/repo/config.yml", "target": "/opt/apps/env/config.yml", "read_only": true},
				map[string]any{"type": "bind", "source": "/home/user/assets", "target": "/app/rapida-data/assets"},
			}
		}
		baseServices[name] = service
	}
	base := map[string]any{"services": baseServices, "networks": map[string]any{"api-network": map[string]any{}}, "volumes": map[string]any{"go-mod-cache": map[string]any{}}}
	merged := cloneMap(base)
	mergedServices := mapValue(merged, "services")
	for _, name := range inheritedServices {
		service := mergedServices[name].(map[string]any)
		delete(service, "container_name")
		delete(service, "ports")
		if build, ok := service["build"].(map[string]any); ok {
			build["cache_from"] = []any{"type=gha,scope=test-" + name}
			build["cache_to"] = []any{"type=gha,scope=test-" + name + ",mode=max"}
		}
		switch name {
		case "postgres":
			service["volumes"] = []any{
				map[string]any{"type": "volume", "source": "postgres-data", "target": "/var/lib/postgresql/data"},
				map[string]any{"type": "bind", "source": "/repo/docker/postgres/init.sql", "target": "/docker-entrypoint-initdb.d/init.sql"},
			}
		case "redis":
			service["volumes"] = []any{}
			service["tmpfs"] = []any{"/data"}
		case "nginx", "web-api", "integration-api", "endpoint-api", "assistant-api":
			service["volumes"] = []any{
				map[string]any{"type": "bind", "source": "/repo/config.yml", "target": "/opt/apps/env/config.yml", "read_only": true},
				map[string]any{"type": "volume", "source": "system-assets", "target": "/app/rapida-data/assets"},
			}
		}
		if contains([]string{"web-api", "integration-api", "endpoint-api", "assistant-api"}, name) {
			service["command"] = []any{"./" + name, "-skip-migration"}
		}
	}
	for _, name := range addedCIServices {
		mergedServices[name] = map[string]any{}
	}
	merged["volumes"] = map[string]any{"go-mod-cache": map[string]any{}, "postgres-data": map[string]any{}, "system-assets": map[string]any{}, "system-reports": map[string]any{}}
	directory := t.TempDir()
	basePath, mergedPath := filepath.Join(directory, "base.json"), filepath.Join(directory, "merged.json")
	writeJSONFile(t, basePath, base)
	writeJSONFile(t, mergedPath, merged)
	overridePath := filepath.Join("..", "..", "..", "docker-compose.ci.yml")
	if err := validateComposeContract(basePath, overridePath, mergedPath, "2.24.4", "docker/ci"); err != nil {
		t.Fatal(err)
	}
}

func TestComposeContractRejectsUnapprovedFieldChange(t *testing.T) {
	base := map[string]any{"services": map[string]any{}, "networks": map[string]any{}, "volumes": map[string]any{}}
	for _, name := range inheritedServices {
		base["services"].(map[string]any)[name] = map[string]any{"image": "same"}
	}
	merged := cloneMap(base)
	for _, name := range addedCIServices {
		merged["services"].(map[string]any)[name] = map[string]any{}
	}
	merged["services"].(map[string]any)["web-api"].(map[string]any)["image"] = "changed"
	directory := t.TempDir()
	basePath, mergedPath := filepath.Join(directory, "base.json"), filepath.Join(directory, "merged.json")
	writeJSONFile(t, basePath, base)
	writeJSONFile(t, mergedPath, merged)
	overridePath := filepath.Join("..", "..", "..", "docker-compose.ci.yml")
	if err := validateComposeContract(basePath, overridePath, mergedPath, "2.24.4", "docker/ci"); err == nil {
		t.Fatal("expected contract failure")
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestServiceImageDigests(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if err := verifyServiceImages(filepath.Join(root, "docker-compose.yml"), filepath.Join(root, "tests/system/service-images.lock"), filepath.Join(root, "docker-compose.yml"), true); err != nil {
		t.Fatal(err)
	}
}

func TestServiceImageDigestsRejectLockDrift(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "tests/system/service-images.lock"))
	if err != nil {
		t.Fatal(err)
	}
	drifted := bytes.Replace(data, []byte("postgres:15-alpine@"), []byte("postgres:16-alpine@"), 1)
	lock := filepath.Join(t.TempDir(), "service-images.lock")
	if err := os.WriteFile(lock, drifted, 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyServiceImages(filepath.Join(root, "docker-compose.yml"), lock, "", false); err == nil {
		t.Fatal("expected image lock mismatch")
	}
}

func TestMigrationReportIncludesVersionAndDirty(t *testing.T) {
	states := map[string]string{"web_db": "4|false", "integration_db": "1|false", "endpoint_db": "1|false", "assistant_db": "54|false"}
	runner := func(_ context.Context, _ string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		for index, arg := range args {
			if arg == "-d" {
				_, _ = io.WriteString(stdout, states[args[index+1]])
				break
			}
		}
		return nil
	}
	report := filepath.Join(t.TempDir(), "migrations.json")
	if err := checkMigrations(context.Background(), runner, io.Discard, report, true, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var records []migrationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 || records[3].Version != 54 || records[3].ExpectedVersion != 54 || records[3].Dirty {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestMigrationReportSharedVolumeContract(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	composeWrapper, err := resolveComposeWrapper()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(composeWrapper,
		"-f", filepath.Join(root, "docker-compose.yml"),
		"-f", filepath.Join(root, "docker-compose.ci.yml"),
		"config", "--format", "json",
	)
	command.Dir = root
	command.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=ci-contract", "SYSTEM_CACHE_SCOPE=contract")
	rendered, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var model map[string]any
	if err := json.Unmarshal(rendered, &model); err != nil {
		t.Fatal(err)
	}
	testRunner, ok := mapValue(model, "services")["test-runner"].(map[string]any)
	if !ok || !hasVolumeTarget(testRunner, "/reports", "system-reports") {
		t.Fatal("rendered test-runner must mount system-reports at /reports")
	}
	script, err := os.ReadFile(filepath.Join(root, ".github/actions/system-test/system-phase.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(script, []byte("systemcheck migrations --require-clean --require-head")) || !bytes.Contains(script, []byte("--report /reports/migrations.json")) {
		t.Fatal("migration producer must write /reports/migrations.json")
	}
	called := false
	runner := func(_ context.Context, name string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		called = true
		if name != composeWrapper {
			t.Fatalf("migration diagnostics invoked %q instead of %q", name, composeWrapper)
		}
		expected := []string{"-p", "ci-contract", "-f", "docker-compose.yml", "-f", "docker-compose.ci.yml", "run", "--rm", "--no-deps", "--entrypoint", "cat", "test-runner", "/reports/migrations.json"}
		if !reflect.DeepEqual(args, expected) {
			t.Fatalf("migration diagnostic args=%q want=%q", args, expected)
		}
		_, _ = io.WriteString(stdout, `[{"service":"web-api","version":4,"expectedVersion":4,"dirty":false},{"service":"integration-api","version":1,"expectedVersion":1,"dirty":false},{"service":"endpoint-api","version":1,"expectedVersion":1,"dirty":false},{"service":"assistant-api","version":54,"expectedVersion":54,"dirty":false}]`)
		return nil
	}
	if _, err := collectMigrationReport(context.Background(), runner, composeWrapper, "ci-contract"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("migration diagnostic reader was not invoked")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestReadinessValidatorUsesIndependentServiceTimeouts(t *testing.T) {
	var mutex sync.Mutex
	attempts := map[string]int{}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		key := request.URL.Host + request.URL.Path
		mutex.Lock()
		attempts[key]++
		attempt := attempts[key]
		mutex.Unlock()
		status, body := http.StatusOK, `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":true,"healthy":true}}`
		if strings.HasSuffix(request.URL.Path, "/healthz/") && attempt == 1 {
			time.Sleep(10 * time.Millisecond)
			status = http.StatusServiceUnavailable
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})}
	started := time.Now()
	if err := checkHealth(context.Background(), client, 25*time.Millisecond, time.Millisecond, readinessPostgresKey, io.Discard); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 35*time.Millisecond {
		t.Fatal("test did not exercise independent service budgets")
	}
}
