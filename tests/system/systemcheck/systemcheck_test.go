package systemcheck

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestParseOpenAPIDirectoryResolvesCommonSchema(t *testing.T) {
	directory := t.TempDir()
	common := "components:\n  schemas:\n    Shared:\n      type: object\n"
	root := "openapi: 3.0.3\npaths: {}\ncomponents:\n  schemas:\n    Local:\n      $ref: ./common.yaml#/components/schemas/Shared\n"
	if err := os.WriteFile(filepath.Join(directory, "common.yaml"), []byte(common), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"assistant-api.yaml", "talk-api.yaml"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(root), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := parseOpenAPIDirectory(directory, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestParseOpenAPIDirectoryRejectsMissingReference(t *testing.T) {
	directory := t.TempDir()
	root := "openapi: 3.0.3\npaths: {}\ncomponents:\n  schemas:\n    Local:\n      $ref: ./common.yaml#/components/schemas/Missing\n"
	if err := os.WriteFile(filepath.Join(directory, "common.yaml"), []byte("components:\n  schemas: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"assistant-api.yaml", "talk-api.yaml"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(root), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := parseOpenAPIDirectory(directory, io.Discard); err == nil {
		t.Fatal("expected missing reference failure")
	}
}

func TestParseOpenAPIDirectoryRejectsMalformedYAML(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "assistant-api.yaml"), []byte("openapi: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "talk-api.yaml"), []byte("openapi: 3.0.3\npaths: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := parseOpenAPIDirectory(directory, io.Discard); err == nil {
		t.Fatal("expected YAML failure")
	}
}

func TestParseOpenAPIDirectoryRejectsTraversalBeforeReadingTarget(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "openapi")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.yaml"), []byte("not: [valid"), 0600); err != nil {
		t.Fatal(err)
	}
	root := "openapi: 3.0.3\npaths: {}\ncomponents:\n  schemas:\n    Outside:\n      $ref: ../outside.yaml\n"
	for _, name := range []string{"assistant-api.yaml", "talk-api.yaml"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(root), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err := parseOpenAPIDirectory(directory, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "reference escapes directory") {
		t.Fatalf("expected traversal rejection before target parsing, got %v", err)
	}
}

func TestComposeImageIDsRejectMalformedNonEmptyID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose-images.json")
	data := `[{"Service":"assistant-api","ID":"not-an-image-id"}]`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := composeImageIDs(path); err == nil {
		t.Fatal("expected malformed non-empty image ID to fail")
	}
}

func TestParseMigrationState(t *testing.T) {
	version, dirty, err := parseMigrationState("54|f\n")
	if err != nil || version != 54 || dirty {
		t.Fatalf("got version=%d dirty=%t err=%v", version, dirty, err)
	}
	if _, _, err := parseMigrationState("54|t"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseMigrationState("bad"); err == nil {
		t.Fatal("expected invalid state error")
	}
}

func TestMigrationHeadsMatchRepository(t *testing.T) {
	for _, spec := range migrations {
		matches, err := filepath.Glob(filepath.Join("..", "..", "..", "api", spec.Service, "migrations", "*.up.sql"))
		if err != nil {
			t.Fatal(err)
		}
		var head int
		for _, match := range matches {
			var version int
			if _, err := fmt.Sscanf(filepath.Base(match), "%06d_", &version); err == nil && version > head {
				head = version
			}
		}
		if head != spec.Head {
			t.Errorf("%s head=%d, systemcheck=%d", spec.Service, head, spec.Head)
		}
	}
}

func TestCheckEndpointSemanticJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":200,"success":true,"data":{"healthy":true}}`)
	}))
	defer server.Close()
	if err := checkEndpoint(context.Background(), server.Client(), server.URL, "healthy"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckEndpointRejectsFalseSemanticValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":200,"success":true,"data":{"healthy":false}}`)
	}))
	defer server.Close()
	if err := checkEndpoint(context.Background(), server.Client(), server.URL, "healthy"); err == nil {
		t.Fatal("expected semantic failure")
	}
}

func TestCheckEndpointRejectsUnrelatedReadinessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":200,"success":true,"data":{"redis":true}}`)
	}))
	defer server.Close()
	if err := checkEndpoint(context.Background(), server.Client(), server.URL, readinessPostgresKey); err == nil {
		t.Fatal("expected exact postgres readiness key failure")
	}
}

func TestReadinessValidator(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{name: "valid", status: 200, body: `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":true}}`},
		{name: "http status", status: 503, body: `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":true}}`, wantError: true},
		{name: "malformed", status: 200, body: `{`, wantError: true},
		{name: "wrong code", status: 200, body: `{"code":500,"success":true,"data":{"PSQL psql://postgres:5432":true}}`, wantError: true},
		{name: "unsuccessful", status: 200, body: `{"code":200,"success":false,"data":{"PSQL psql://postgres:5432":true}}`, wantError: true},
		{name: "missing", status: 200, body: `{"code":200,"success":true,"data":{}}`, wantError: true},
		{name: "wrong key", status: 200, body: `{"code":200,"success":true,"data":{"postgres":true}}`, wantError: true},
		{name: "arbitrary dependency", status: 200, body: `{"code":200,"success":true,"data":{"redis":true}}`, wantError: true},
		{name: "non boolean", status: 200, body: `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":"true"}}`, wantError: true},
		{name: "false", status: 200, body: `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":false}}`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			err := checkEndpoint(context.Background(), server.Client(), server.URL, readinessPostgresKey)
			if (err != nil) != test.wantError {
				t.Fatalf("err=%v wantError=%t", err, test.wantError)
			}
		})
	}
}

func TestCheckUIFetchesHashedAsset(t *testing.T) {
	talkBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != talkProxyRPCPath {
			writeGRPCWebResponse(w, "12", "unknown talk method")
			return
		}
		writeGRPCWebResponse(w, "0", "talk-response")
	}))
	defer talkBackend.Close()
	webBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveHTTPRouteContract(w, r) {
			return
		}
		if r.URL.Path != webProxyRPCPath {
			writeGRPCWebResponse(w, "12", "unknown web method")
			return
		}
		writeGRPCWebResponse(w, "0", "web-response")
	}))
	defer webBackend.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveHTTPRouteContract(w, r) {
			return
		}
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `<script src="/static/js/main.1234abcd.js"></script>`)
		case "/static/js/main.1234abcd.js":
			io.WriteString(w, "ok")
		case talkProxyRPCPath:
			writeGRPCWebResponse(w, "0", "talk-response")
		case webProxyRPCPath:
			writeGRPCWebResponse(w, "0", "web-response")
		default:
			writeGRPCWebResponse(w, "12", "unknown proxy method")
		}
	}))
	defer server.Close()
	client := proxyRouteClient(server.Client(), map[string]string{
		talkProxyUpstream: talkBackend.URL,
		webProxyUpstream:  webBackend.URL,
	})
	if err := checkUI(context.Background(), client, server.URL, defaultAssetPattern, workflowProxyRouteArguments(), workflowHTTPRouteArguments()); err != nil {
		t.Fatal(err)
	}
}

func TestCheckUIRejectsArbitraryProxy404(t *testing.T) {
	backend := httptest.NewServer(http.NotFoundHandler())
	defer backend.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			_, _ = io.WriteString(w, `<script src="/static/js/main.1234abcd.js"></script>`)
			return
		}
		if r.URL.Path == "/static/js/main.1234abcd.js" {
			_, _ = io.WriteString(w, "asset")
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := proxyRouteClient(server.Client(), map[string]string{talkProxyUpstream: backend.URL, webProxyUpstream: backend.URL})
	if err := checkUI(context.Background(), client, server.URL, defaultAssetPattern, workflowProxyRouteArguments(), workflowHTTPRouteArguments()); err == nil {
		t.Fatal("expected arbitrary 404 proxy response to fail")
	}
}

func TestCheckUIRejectsWrongBackendWithGenericEqualBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeGRPCWebResponse(w, "0", "generic-response")
	}))
	defer backend.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = io.WriteString(w, `<script src="/static/js/main.1234abcd.js"></script>`)
		case "/static/js/main.1234abcd.js":
			_, _ = io.WriteString(w, "asset")
		default:
			writeGRPCWebResponse(w, "12", "generic-response")
		}
	}))
	defer server.Close()
	client := proxyRouteClient(server.Client(), map[string]string{talkProxyUpstream: backend.URL, webProxyUpstream: backend.URL})
	if err := checkUI(context.Background(), client, server.URL, defaultAssetPattern, workflowProxyRouteArguments(), workflowHTTPRouteArguments()); err == nil {
		t.Fatal("expected wrong backend gRPC status to fail despite equal HTTP body")
	}
}

func TestCheckUIRejectsEqualUnimplementedResponses(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeGRPCWebResponse(w, "12", "generic-response")
	}))
	defer backend.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = io.WriteString(w, `<script src="/static/js/main.1234abcd.js"></script>`)
		case "/static/js/main.1234abcd.js":
			_, _ = io.WriteString(w, "asset")
		default:
			writeGRPCWebResponse(w, "12", "generic-response")
		}
	}))
	defer server.Close()
	client := proxyRouteClient(server.Client(), map[string]string{talkProxyUpstream: backend.URL, webProxyUpstream: backend.URL})
	if err := checkUI(context.Background(), client, server.URL, defaultAssetPattern, workflowProxyRouteArguments(), workflowHTTPRouteArguments()); err == nil {
		t.Fatal("expected equal unimplemented responses to fail")
	}
}

func TestCheckUIRejectsHTTPRouteServedByUI(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveHTTPRouteContract(w, r) {
			return
		}
		writeGRPCWebResponse(w, "0", "response")
	}))
	defer backend.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = io.WriteString(w, `<script src="/static/js/main.1234abcd.js"></script>`)
		case "/static/js/main.1234abcd.js":
			_, _ = io.WriteString(w, "asset")
		case "/v1/__systemcheck__":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html>ui fallback</html>")
		default:
			if serveHTTPRouteContract(w, r) {
				return
			}
			writeGRPCWebResponse(w, "0", "response")
		}
	}))
	defer server.Close()
	client := proxyRouteClient(server.Client(), map[string]string{talkProxyUpstream: backend.URL, webProxyUpstream: backend.URL})
	if err := checkUI(context.Background(), client, server.URL, defaultAssetPattern, workflowProxyRouteArguments(), workflowHTTPRouteArguments()); err == nil {
		t.Fatal("expected UI fallback for /v1/ to fail HTTP upstream parity")
	}
}

func TestProxyRPCPathsMatchGeneratedDescriptors(t *testing.T) {
	root := filepath.Join("..", "..", "..", "protos")
	checks := map[string]string{
		"talk-api_grpc.pb.go": `TalkService_GetAllAssistantConversation_FullMethodName = "` + talkProxyRPCPath + `"`,
		"web-api_grpc.pb.go":  `AuthenticationService_ForgotPassword_FullMethodName = "` + webProxyRPCPath + `"`,
	}
	for file, expected := range checks {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), expected) {
			t.Fatalf("%s does not contain %q", file, expected)
		}
	}
}

func TestProxyRoutesAcceptExactWorkflowArguments(t *testing.T) {
	routes, err := parseProxyRoutes(workflowProxyRouteArguments())
	if err != nil {
		t.Fatal(err)
	}
	if routes[0].RPCPath != talkProxyRPCPath || routes[0].Upstream != talkProxyUpstream || routes[1].RPCPath != webProxyRPCPath || routes[1].Upstream != webProxyUpstream {
		t.Fatalf("unexpected parsed routes: %#v", routes)
	}
}

func TestParseProxyRoutesRejectsInvalidContracts(t *testing.T) {
	tests := map[string][]string{
		"prefix-only":     {"/talk_api=" + talkProxyUpstream, webProxyRPCPath + "=" + webProxyUpstream},
		"duplicate":       {talkProxyRPCPath + "=" + talkProxyUpstream, talkProxyRPCPath + "=" + talkProxyUpstream},
		"swapped":         {talkProxyRPCPath + "=" + webProxyUpstream, webProxyRPCPath + "=" + talkProxyUpstream},
		"talk-wrong-host": {talkProxyRPCPath + "=web-api:9007", webProxyRPCPath + "=" + webProxyUpstream},
		"web-wrong-host":  {talkProxyRPCPath + "=" + talkProxyUpstream, webProxyRPCPath + "=assistant-api:9001"},
		"talk-wrong-port": {talkProxyRPCPath + "=assistant-api:9001", webProxyRPCPath + "=" + webProxyUpstream},
		"web-wrong-port":  {talkProxyRPCPath + "=" + talkProxyUpstream, webProxyRPCPath + "=web-api:9007"},
	}
	for name, routes := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProxyRoutes(routes); err == nil {
				t.Fatalf("expected invalid proxy routes to fail: %q", routes)
			}
		})
	}
}

func TestHTTPRoutesAcceptExactWorkflowArguments(t *testing.T) {
	routes, err := parseHTTPRoutes(workflowHTTPRouteArguments())
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != len(httpRouteContracts) {
		t.Fatalf("unexpected parsed HTTP routes: %#v", routes)
	}
}

func TestParseHTTPRoutesRejectsInvalidContracts(t *testing.T) {
	tests := map[string][]string{
		"missing":    workflowHTTPRouteArguments()[:3],
		"unknown":    append(workflowHTTPRouteArguments()[:3], "/unknown/=web-api:9001"),
		"duplicate":  append(workflowHTTPRouteArguments()[:3], workflowHTTPRouteArguments()[0]),
		"wrong-host": append(workflowHTTPRouteArguments()[:3], "/healthz/=ui:3000"),
	}
	for name, routes := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseHTTPRoutes(routes); err == nil {
				t.Fatalf("expected invalid HTTP routes to fail: %q", routes)
			}
		})
	}
}

func TestGRPCWebTrailerStatus(t *testing.T) {
	payload := []byte("grpc-status: 7\r\ngrpc-message: denied\r\n")
	frame := make([]byte, 5, 5+len(payload))
	frame[0] = 0x80
	binary.BigEndian.PutUint32(frame[1:], uint32(len(payload)))
	frame = append(frame, payload...)
	if status := grpcWebTrailerStatus(frame); status != "7" {
		t.Fatalf("status=%q", status)
	}
}

func writeGRPCWebResponse(writer http.ResponseWriter, grpcStatus, body string) {
	writer.Header().Set("Content-Type", "application/grpc-web+proto")
	writer.Header().Set("Grpc-Status", grpcStatus)
	writer.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(writer, body)
}

func workflowProxyRouteArguments() []string {
	return []string{
		"/talk_api.TalkService/GetAllAssistantConversation=assistant-api:9007",
		"/web_api.AuthenticationService/ForgotPassword=web-api:9001",
	}
}

func workflowHTTPRouteArguments() []string {
	return []string{
		"/v1/__systemcheck__=web-api:9001",
		"/oauth/__systemcheck__=web-api:9001",
		"/readiness/=web-api:9001",
		"/healthz/=web-api:9001",
	}
}

func serveHTTPRouteContract(writer http.ResponseWriter, request *http.Request) bool {
	switch request.URL.Path {
	case "/v1/__systemcheck__", "/oauth/__systemcheck__":
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(writer, "404 page not found")
	case "/readiness/":
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(writer, `{"code":200,"success":true,"data":{"PSQL psql://postgres:5432":true}}`)
	case "/healthz/":
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(writer, `{"code":200,"success":true,"data":{"healthy":true}}`)
	default:
		return false
	}
	return true
}

func proxyRouteClient(client *http.Client, upstreams map[string]string) *http.Client {
	result := *client
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	result.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		target, ok := upstreams[request.URL.Host]
		if !ok {
			return transport.RoundTrip(request)
		}
		clone := request.Clone(request.Context())
		urlCopy := *request.URL
		urlCopy.Scheme = "http"
		urlCopy.Host = strings.TrimPrefix(target, "http://")
		clone.URL = &urlCopy
		clone.Host = urlCopy.Host
		return transport.RoundTrip(clone)
	})
	return &result
}

func TestAssistantSmokeProtectsAndDeletesEnvironment(t *testing.T) {
	directory := t.TempDir()
	var environmentPath string
	runner := func(_ context.Context, name string, args []string, stdin io.Reader, _, _ io.Writer, _ []string) error {
		if name == "psql" {
			if stdin == nil {
				t.Fatal("seed was not provided on stdin")
			}
			return nil
		}
		for index, arg := range args {
			if arg == "--environment" {
				environmentPath = args[index+1]
			}
		}
		info, err := os.Stat(environmentPath)
		if err != nil {
			return err
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("mode=%o", info.Mode().Perm())
		}
		return nil
	}
	if err := assistantSmoke(context.Background(), runner, "collection.json", "http://assistant-api:9007", directory, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(environmentPath); !os.IsNotExist(err) {
		t.Fatalf("environment file remains: %v", err)
	}
}

func TestAssistantSmokeDeletesEnvironmentOnNewmanFailure(t *testing.T) {
	directory := t.TempDir()
	var environmentPath string
	runner := func(_ context.Context, name string, args []string, _ io.Reader, _, _ io.Writer, _ []string) error {
		if name == "psql" {
			return nil
		}
		for index, arg := range args {
			if arg == "--environment" {
				environmentPath = args[index+1]
			}
		}
		return fmt.Errorf("newman failed")
	}
	if err := assistantSmoke(context.Background(), runner, "collection.json", "http://assistant-api:9007", directory, io.Discard, io.Discard); err == nil {
		t.Fatal("expected smoke failure")
	}
	if _, err := os.Stat(environmentPath); !os.IsNotExist(err) {
		t.Fatalf("environment file remains: %v", err)
	}
}

func TestSecretNonEmission(t *testing.T) {
	directory := t.TempDir()
	var token string
	var childEnvironment []string
	runner := func(_ context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer, env []string) error {
		if name == "psql" {
			data, _ := io.ReadAll(stdin)
			match := regexp.MustCompile(`'auth-token','([a-f0-9]+)'`).FindSubmatch(data)
			if len(match) == 2 {
				token = string(match[1])
			}
			return nil
		}
		childEnvironment = append([]string(nil), env...)
		for _, arg := range args {
			if arg == token {
				t.Fatal("token was passed as a command argument")
			}
		}
		_, _ = fmt.Fprintf(stdout, "authorization=%s password=sentinel-db\n", token)
		_, _ = fmt.Fprintln(stderr, "token=eyJabcdefgh.abcdefgh.abcdefgh")
		return nil
	}
	var stdout, stderr strings.Builder
	outputPath := filepath.Join(directory, "github-output")
	summaryPath := filepath.Join(directory, "github-summary")
	t.Setenv("GITHUB_OUTPUT", outputPath)
	t.Setenv("GITHUB_STEP_SUMMARY", summaryPath)
	t.Setenv("SYSTEMCHECK_SECRETS", "sentinel-db")
	if err := assistantSmoke(context.Background(), runner, "collection.json", "http://assistant-api:9007", directory, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"stdout": stdout.String(), "stderr": stderr.String()} {
		if token == "" || strings.Contains(value, token) || strings.Contains(value, "sentinel-db") || strings.Contains(value, "eyJabcdefgh") {
			t.Fatalf("%s leaked token: %q", name, value)
		}
	}
	if !contains(childEnvironment, "GITHUB_OUTPUT=") || !contains(childEnvironment, "GITHUB_STEP_SUMMARY=") {
		t.Fatalf("child output channels not cleared: %#v", childEnvironment)
	}
	for _, path := range []string{outputPath, summaryPath} {
		if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), token) {
			t.Fatalf("%s leaked token", path)
		}
	}
}

func TestCollectDiagnosticsIncludesMigrationVersionAndDirty(t *testing.T) {
	runner := func(_ context.Context, _ string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--entrypoint cat"):
			_, _ = io.WriteString(stdout, `[{"service":"web-api","version":4,"expectedVersion":4,"dirty":false},{"service":"integration-api","version":1,"expectedVersion":1,"dirty":false},{"service":"endpoint-api","version":1,"expectedVersion":1,"dirty":false},{"service":"assistant-api","version":54,"expectedVersion":54,"dirty":false}]`)
		case strings.Contains(joined, " ps "):
			_, _ = io.WriteString(stdout, `{"ExitCode":1,"Health":"unhealthy"}`)
		case strings.Contains(joined, " logs "):
			_, _ = io.WriteString(stdout, "ERROR failed\n")
		}
		return nil
	}
	directory := t.TempDir()
	if err := collectDiagnostics(context.Background(), runner, "ci-test", directory); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []diagnosticRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != len(diagnosticServices) {
		t.Fatalf("diagnostic services=%d want=%d", len(records), len(diagnosticServices))
	}
	byService := make(map[string]diagnosticRecord, len(records))
	for _, record := range records {
		byService[record.Service] = record
	}
	for _, service := range diagnosticServices {
		if _, ok := byService[service]; !ok {
			t.Fatalf("missing diagnostic service %s", service)
		}
	}
	for _, spec := range migrations {
		record := byService[spec.Service]
		if record.MigrationVersion != spec.Head || record.MigrationExpectedVersion != spec.Head || record.MigrationDirty == nil || *record.MigrationDirty {
			t.Fatalf("migration diagnostics missing for %s: %#v", spec.Service, record)
		}
	}
}

func TestCollectDiagnosticsUsesPinnedComposeWrapper(t *testing.T) {
	composeWrapper, err := resolveComposeWrapper()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	runner := func(_ context.Context, name string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		if name != composeWrapper {
			t.Fatalf("diagnostics invoked %q instead of pinned Compose wrapper %q", name, composeWrapper)
		}
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, " run "):
			seen["run"] = true
			_, _ = io.WriteString(stdout, `[{"service":"web-api","version":4,"expectedVersion":4,"dirty":false},{"service":"integration-api","version":1,"expectedVersion":1,"dirty":false},{"service":"endpoint-api","version":1,"expectedVersion":1,"dirty":false},{"service":"assistant-api","version":54,"expectedVersion":54,"dirty":false}]`)
		case strings.Contains(joined, " ps "):
			seen["ps"] = true
			_, _ = io.WriteString(stdout, `{"ExitCode":0,"Health":"healthy"}`)
		case strings.Contains(joined, " logs "):
			seen["logs"] = true
		}
		return nil
	}
	if err := collectDiagnostics(context.Background(), runner, "ci-test", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"run", "ps", "logs"} {
		if !seen[command] {
			t.Fatalf("diagnostics did not invoke Compose %s", command)
		}
	}
}

func TestCollectDiagnosticsFallsBackToRenderedImageMapping(t *testing.T) {
	directory := t.TempDir()
	images := testComposeImageMapping("ci-test")
	writeComposeImageMapping(t, filepath.Join(directory, "compose-images.json"), images)
	if err := os.WriteFile(filepath.Join(directory, "buildx-builder.txt"), []byte("Name: ci-builder\nDriver: docker-container\nStatus: running\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "cache-scope.txt"), []byte("cache-scope\n"), 0600); err != nil {
		t.Fatal(err)
	}
	composeWrapper, err := resolveComposeWrapper()
	if err != nil {
		t.Fatal(err)
	}
	inspected := []string{}
	runner := func(_ context.Context, name string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		if name == composeWrapper {
			return errors.New("stack unavailable")
		}
		if name != "docker" || len(args) != 5 || args[0] != "image" || args[1] != "inspect" || args[2] != "--format" || args[3] != "{{.Id}}" {
			t.Fatalf("unexpected diagnostic command: %s %#v", name, args)
		}
		inspected = append(inspected, args[4])
		if args[4] == images["web-api"] || args[4] == images["ui"] {
			_, _ = fmt.Fprintf(stdout, "sha256:%064x\n", len(inspected)+100)
			return nil
		}
		_, _ = io.WriteString(stdout, "credential=raw-secret-that-must-not-escape\n")
		return errors.New("image unavailable")
	}
	if err := collectDiagnostics(context.Background(), runner, "ci-test", directory); err != nil {
		t.Fatal(err)
	}
	if len(inspected) != len(builtServices) {
		t.Fatalf("inspected %d images, want %d", len(inspected), len(builtServices))
	}
	for index, service := range builtServices {
		if inspected[index] != images[service] {
			t.Fatalf("inspect %d used %q, want %q", index, inspected[index], images[service])
		}
	}
	data, err := os.ReadFile(filepath.Join(directory, "build-diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("raw-secret")) {
		t.Fatal("raw image inspection output escaped into diagnostics")
	}
	var record buildDiagnosticRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.Metadata == nil || len(record.Metadata.Images) != len(builtServices) {
		t.Fatalf("missing fallback image evidence: %#v", record.Metadata)
	}
	for _, image := range record.Metadata.Images {
		if image.Service == "web-api" || image.Service == "ui" {
			if image.Status != "available" || !regexpImageID.MatchString(image.ImageID) {
				t.Fatalf("valid image evidence not retained: %#v", image)
			}
		} else if image.Status != "unavailable" || image.ImageID != "" {
			t.Fatalf("unavailable image evidence not explicit: %#v", image)
		}
	}
}

func TestCollectDiagnosticsSanitizesBuildAndServiceData(t *testing.T) {
	directory := t.TempDir()
	var composeImages strings.Builder
	imageServices := append(append([]string(nil), diagnosticServices...), "test-runner")
	metadata := buildMetadata{Builder: "ci-builder", CacheScope: "cache-scope"}
	for index, service := range imageServices {
		imageID := fmt.Sprintf("sha256:%064x", index+20)
		_, _ = fmt.Fprintf(&composeImages, `{"Service":%q,"ID":%q}`+"\n", service, imageID)
		if contains(builtServices, service) {
			metadata.Images = append(metadata.Images, buildImageMetadata{Service: service, ImageRef: testComposeImageMapping("ci-test")[service], ImageID: imageID})
		}
	}
	metadataData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{
		"buildx-builder.txt":     "Name: ci-builder\nDriver: docker-container\nStatus: running\nBuildkit: v0.16.0\nignored: credential=opaque-builder-secret\n",
		"buildx-du.txt":          "ID RECLAIMABLE SIZE\nTotal: 12.3MB\n",
		"docker-system-df.txt":   "TYPE TOTAL ACTIVE SIZE RECLAIMABLE\nImages 6 6 1GB 0B\nContainers 8 8 10MB 0B\nLocal Volumes 3 3 1MB 0B\nBuild Cache 2 2 20MB 0B\n",
		"cache-scope.txt":        "cache-scope\n",
		"buildkit.log":           "ordinary credential=ignored\nERROR credential=opaque-build-token\n",
		"buildkit-metadata.json": string(metadataData),
		"compose-images.json":    composeImages.String(),
	}
	for name, value := range inputs {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	runner := func(_ context.Context, _ string, args []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "--entrypoint cat"):
			_, _ = io.WriteString(stdout, `[{"service":"web-api","version":4,"expectedVersion":4,"dirty":false},{"service":"integration-api","version":1,"expectedVersion":1,"dirty":false},{"service":"endpoint-api","version":1,"expectedVersion":1,"dirty":false},{"service":"assistant-api","version":54,"expectedVersion":54,"dirty":false}]`)
		case strings.Contains(joined, " ps "):
			_, _ = io.WriteString(stdout, `{"ExitCode":0,"Health":"healthy"}`)
		case strings.Contains(joined, " logs "):
			_, _ = io.WriteString(stdout, "ERROR credential=opaque-service-token\n")
		}
		return nil
	}
	if err := collectDiagnostics(context.Background(), runner, "ci-test", directory); err != nil {
		t.Fatal(err)
	}
	for _, name := range buildDiagnosticSources {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("raw diagnostic source remains: %s", name)
		}
	}
	for _, name := range []string{"diagnostics.json", "build-diagnostics.json"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "opaque-") {
			t.Fatalf("%s contains opaque credential: %s", name, data)
		}
	}
	diagnosticsData, err := os.ReadFile(filepath.Join(directory, "diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []diagnosticRecord
	if err := json.Unmarshal(diagnosticsData, &records); err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		if record.ImageID == "" {
			t.Fatalf("missing image ID for %s", record.Service)
		}
	}
	buildData, err := os.ReadFile(filepath.Join(directory, "build-diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var buildRecord buildDiagnosticRecord
	if err := json.Unmarshal(buildData, &buildRecord); err != nil {
		t.Fatal(err)
	}
	if buildRecord.Metadata == nil || len(buildRecord.Metadata.Images) != len(builtServices) || buildRecord.Builder != "ci-builder" || buildRecord.CacheScope != "cache-scope" {
		t.Fatalf("missing required build metadata: %#v", buildRecord)
	}
}

func TestCollectDiagnosticsFailsClosedOnMalformedBuildMetadata(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "buildkit-metadata.json"), []byte(`{"builder":"ci"}`), 0600); err != nil {
		t.Fatal(err)
	}
	runner := func(_ context.Context, _ string, _ []string, _ io.Reader, _ io.Writer, _ io.Writer, _ []string) error {
		return errors.New("not available")
	}
	if err := collectDiagnostics(context.Background(), runner, "ci-test", directory); err == nil {
		t.Fatal("expected malformed metadata failure")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("diagnostic directory remains after failure: %v", err)
	}
}

func TestSanitizerRedactsKnownSecretsAndTokens(t *testing.T) {
	value := sanitize("password=rapida_db_password authorization: opaque-credential credential=another-opaque-token token=eyJabcdefgh.abcdefgh.abcdefgh postgres://user:db-secret@postgres/db")
	for _, secret := range []string{"rapida_db_password", "opaque-credential", "another-opaque-token", "eyJabcdefgh", "db-secret"} {
		if strings.Contains(value, secret) {
			t.Fatalf("secret %q remains: %s", secret, value)
		}
	}
	if strings.Count(value, "[REDACTED]") < 5 {
		t.Fatalf("secret remains: %s", value)
	}
}

func TestSanitizeDirectoryRejectsOpaqueCredential(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "diagnostics.json"), []byte(`{"logs":["ERROR credential=opaque-value"]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sanitizeDirectory(directory, nil); err == nil {
		t.Fatal("expected opaque credential rejection")
	}
}

func TestSanitizeDirectoryFailsClosed(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "diagnostics.json"), []byte("known-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sanitizeDirectory(directory, []string{"known-secret"}); err == nil {
		t.Fatal("expected sanitizer failure")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("directory remains: %v", err)
	}
}

func TestSanitizeDirectoryRejectsEnvironmentArtifacts(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "newman-environment.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sanitizeDirectory(directory, nil); err == nil {
		t.Fatal("expected unsafe artifact failure")
	}
}

func TestSanitizeDirectoryRejectsInspectArtifacts(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "container-inspect.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sanitizeDirectory(directory, nil); err == nil {
		t.Fatal("expected unsafe artifact failure")
	}
}

func TestSanitizeDirectoryRejectsRawInspectContent(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "diagnostics.json"), []byte(`{"Config":{"Env":["PASSWORD=secret"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sanitizeDirectory(directory, nil); err == nil {
		t.Fatal("expected unsafe artifact failure")
	}
}

func TestSelectedLogsAreAllowlistedAndRedacted(t *testing.T) {
	logs := selectedLogs("ordinary request body secret-value\nERROR authorization: Bearer eyJabcdefgh.abcdefgh.abcdefgh\n")
	if len(logs) != 1 || strings.Contains(logs[0], "eyJabcdefgh") {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestCleanupRetriesUntilProjectResourcesDisappear(t *testing.T) {
	calls := 0
	runner := func(_ context.Context, _ string, _ []string, _ io.Reader, stdout, _ io.Writer, _ []string) error {
		calls++
		if calls <= 3 {
			io.WriteString(stdout, "resource-id\n")
		}
		return nil
	}
	if err := cleanup(context.Background(), runner, "ci-test", 2, time.Millisecond); err != nil {
		t.Fatal(err)
	}
}
