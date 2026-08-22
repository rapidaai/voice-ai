package systemcheck

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type migrationSpec struct {
	Service  string
	Database string
	Head     int
}

var migrations = []migrationSpec{
	{Service: "web-api", Database: "web_db", Head: 4},
	{Service: "integration-api", Database: "integration_db", Head: 1},
	{Service: "endpoint-api", Database: "endpoint_db", Head: 1},
	{Service: "assistant-api", Database: "assistant_db", Head: 54},
}

var services = []struct {
	Name string
	URL  string
}{
	{Name: "web-api", URL: "http://web-api:9001"},
	{Name: "integration-api", URL: "http://integration-api:9004"},
	{Name: "endpoint-api", URL: "http://endpoint-api:9005"},
	{Name: "assistant-api", URL: "http://assistant-api:9007"},
}

var diagnosticServices = []string{
	"postgres",
	"redis",
	"web-api",
	"integration-api",
	"endpoint-api",
	"assistant-api",
	"ui",
	"nginx",
}

const composeWrapperRelativePath = "tests/system/bin/docker-compose"

type commandRunner func(context.Context, string, []string, io.Reader, io.Writer, io.Writer, []string) error

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: systemcheck <compose-contract|service-image-digests|openapi-parse|build-metadata|migrations|health|ui-nginx|assistant-smoke|collect-diagnostics|sanitize-artifacts|cleanup>")
	}
	switch args[0] {
	case "compose-contract":
		fs := flag.NewFlagSet("compose-contract", flag.ContinueOnError)
		fs.SetOutput(stderr)
		base := fs.String("base-rendered", "", "base Compose JSON")
		override := fs.String("override", "", "Compose override YAML")
		merged := fs.String("merged-rendered", "", "merged Compose JSON")
		version := fs.String("compose-version", "", "Compose version")
		forbidPath := fs.String("forbid-path", "", "forbidden path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return validateComposeContract(*base, *override, *merged, *version, *forbidPath)
	case "service-image-digests":
		fs := flag.NewFlagSet("service-image-digests", flag.ContinueOnError)
		fs.SetOutput(stderr)
		compose := fs.String("compose", "", "authoritative Compose file")
		lock := fs.String("lock", "", "service image lock")
		baseline := fs.String("baseline", "", "baseline Compose file")
		forbidMajor := fs.Bool("forbid-major-change", false, "reject changed image tracks")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return verifyServiceImages(*compose, *lock, *baseline, *forbidMajor)
	case "openapi-parse":
		if len(args) != 2 {
			return errors.New("usage: systemcheck openapi-parse <directory>")
		}
		return parseOpenAPIDirectory(args[1], stdout)
	case "build-metadata":
		fs := flag.NewFlagSet("build-metadata", flag.ContinueOnError)
		fs.SetOutput(stderr)
		composeImages := fs.String("compose-images", "", "rendered service-to-image JSON mapping")
		output := fs.String("output", "", "sanitized build metadata output")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("usage: systemcheck build-metadata --compose-images <path> --output <path>")
		}
		return writeBuildMetadata(ctx, runCommand, *composeImages, *output, os.Getenv("BUILDX_BUILDER"), os.Getenv("SYSTEM_CACHE_SCOPE"))
	case "migrations":
		fs := flag.NewFlagSet("migrations", flag.ContinueOnError)
		fs.SetOutput(stderr)
		requireClean := fs.Bool("require-clean", false, "require every migration state to be clean")
		requireHead := fs.Bool("require-head", false, "require every migration state to match repository head")
		report := fs.String("report", "/reports/migrations.json", "migration report path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || !*requireClean || !*requireHead {
			return errors.New("usage: systemcheck migrations --require-clean --require-head --report <path>")
		}
		return checkMigrations(ctx, runCommand, stdout, *report, *requireClean, *requireHead)
	case "health":
		fs := flag.NewFlagSet("health", flag.ContinueOnError)
		fs.SetOutput(stderr)
		timeoutPerService := fs.Duration("timeout-per-service", 60*time.Second, "health timeout for each service")
		interval := fs.Duration("interval", time.Second, "retry interval")
		readinessKey := fs.String("readiness-key", readinessPostgresKey, "required readiness dependency key")
		rejectArbitraryTrue := fs.Bool("reject-arbitrary-true-fallback", false, "reject fallback to unrelated true readiness fields")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || *timeoutPerService <= 0 || *interval <= 0 || *readinessKey != readinessPostgresKey || !*rejectArbitraryTrue {
			return errors.New("usage: systemcheck health --timeout-per-service <duration> --interval <duration> --readiness-key 'PSQL psql://postgres:5432' --reject-arbitrary-true-fallback")
		}
		return checkHealth(ctx, http.DefaultClient, *timeoutPerService, *interval, *readinessKey, stdout)
	case "ui-nginx":
		fs := flag.NewFlagSet("ui-nginx", flag.ContinueOnError)
		fs.SetOutput(stderr)
		baseURL := fs.String("base-url", "http://nginx:8080", "nginx base URL")
		assetPattern := fs.String("asset-pattern", defaultAssetPattern, "regular expression matching a hashed UI asset")
		requireSPARoot := fs.Bool("require-spa-root", false, "require the SPA entry document")
		requireHashedAsset := fs.Bool("require-hashed-asset", false, "require and fetch a content-hashed asset")
		var proxyRoutes stringListFlag
		var httpRoutes stringListFlag
		fs.Var(&proxyRoutes, "proxy-route", "proxy route mapping PATH=HOST:PORT; may be repeated")
		fs.Var(&httpRoutes, "http-route", "HTTP route mapping PATH=HOST:PORT; may be repeated")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || !*requireSPARoot || !*requireHashedAsset || len(proxyRoutes) == 0 || len(httpRoutes) == 0 {
			return errors.New("usage: systemcheck ui-nginx --base-url <url> --asset-pattern <regexp> --require-spa-root --require-hashed-asset --proxy-route <path=host:port> [--proxy-route <path=host:port>] --http-route <path=host:port> [--http-route <path=host:port>]")
		}
		return checkUI(ctx, http.DefaultClient, *baseURL, *assetPattern, proxyRoutes, httpRoutes)
	case "assistant-smoke":
		fs := flag.NewFlagSet("assistant-smoke", flag.ContinueOnError)
		fs.SetOutput(stderr)
		collection := fs.String("collection", "openapi/postman/assistant-api/assistant-api.smoke.postman_collection.json", "collection path")
		baseURL := fs.String("base-url", "http://assistant-api:9007", "assistant URL")
		tmpfs := fs.String("tmpfs", "/run/secrets", "tmpfs directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return assistantSmoke(ctx, runCommand, *collection, *baseURL, *tmpfs, stdout, stderr)
	case "collect-diagnostics":
		fs := flag.NewFlagSet("collect-diagnostics", flag.ContinueOnError)
		fs.SetOutput(stderr)
		project := fs.String("compose-project", "", "Compose project name")
		directory := fs.String("directory", "", "artifact staging directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || *project == "" || *directory == "" {
			return errors.New("usage: systemcheck collect-diagnostics --compose-project <name> --directory <path>")
		}
		return collectDiagnostics(ctx, runCommand, *project, *directory)
	case "sanitize-artifacts":
		fs := flag.NewFlagSet("sanitize-artifacts", flag.ContinueOnError)
		fs.SetOutput(stderr)
		directory := fs.String("directory", "", "artifact staging directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || *directory == "" {
			return errors.New("usage: systemcheck sanitize-artifacts --directory <path>")
		}
		return sanitizeDirectory(*directory, configuredSecrets())
	case "cleanup":
		fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
		fs.SetOutput(stderr)
		project := fs.String("compose-project", "", "Compose project name")
		retries := fs.Int("retries", 10, "resource checks")
		interval := fs.Duration("interval", time.Second, "retry interval")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 || *project == "" || *retries < 1 || *interval <= 0 {
			return errors.New("usage: systemcheck cleanup --compose-project <name> --retries <positive integer> --interval <positive duration>")
		}
		return cleanup(ctx, runCommand, *project, *retries, *interval)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parseOpenAPIDirectory(directory string, stdout io.Writer) error {
	root, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	for _, name := range []string{"assistant-api.yaml", "talk-api.yaml"} {
		path := filepath.Join(root, name)
		if err := validateOpenAPIFile(root, path, map[string]bool{}); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Fprintf(stdout, "%s parsed with local references\n", name)
	}
	return nil
}

func validateOpenAPIFile(root, path string, visited map[string]bool) error {
	canonical, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if canonical != root && !strings.HasPrefix(canonical, root+string(os.PathSeparator)) {
		return fmt.Errorf("reference escapes directory: %s", path)
	}
	if visited[canonical] {
		return nil
	}
	data, err := os.ReadFile(canonical)
	if err != nil {
		return err
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if filepath.Base(canonical) != "common.yaml" {
		if version, ok := document["openapi"].(string); !ok || strings.TrimSpace(version) == "" {
			return errors.New("missing OpenAPI version")
		}
	}
	visited[canonical] = true
	return walkReferences(root, canonical, document, visited)
}

func walkReferences(root, current string, value any, visited map[string]bool) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok {
					return fmt.Errorf("non-string $ref in %s", current)
				}
				if err := validateReference(root, current, reference, visited); err != nil {
					return err
				}
				continue
			}
			if err := walkReferences(root, current, child, visited); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := walkReferences(root, current, child, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateReference(root, current, reference string, visited map[string]bool) error {
	parsed, err := url.Parse(reference)
	if err != nil {
		return fmt.Errorf("invalid $ref %q: %w", reference, err)
	}
	if parsed.IsAbs() || parsed.Host != "" {
		return fmt.Errorf("remote $ref is not hermetic: %s", reference)
	}
	target := current
	if parsed.Path != "" {
		target = filepath.Join(filepath.Dir(current), filepath.FromSlash(parsed.Path))
	}
	canonical, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve $ref %q: %w", reference, err)
	}
	if canonical != root && !strings.HasPrefix(canonical, root+string(os.PathSeparator)) {
		return fmt.Errorf("reference escapes directory: %s", target)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("resolve $ref %q: %w", reference, err)
	}
	var document any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse $ref %q: %w", reference, err)
	}
	if parsed.Fragment != "" {
		fragment, err := url.PathUnescape(parsed.Fragment)
		if err != nil {
			return fmt.Errorf("decode $ref %q: %w", reference, err)
		}
		if _, err := resolveJSONPointer(document, fragment); err != nil {
			return fmt.Errorf("resolve $ref %q: %w", reference, err)
		}
	}
	return validateOpenAPIFile(root, target, visited)
}

func resolveJSONPointer(document any, pointer string) (any, error) {
	if pointer == "" {
		return document, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON pointer %q", pointer)
	}
	current := document
	for _, part := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pointer %q traverses non-object", pointer)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("pointer %q does not exist", pointer)
		}
	}
	return current, nil
}

func runCommand(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	cmd.Env = mergeEnvironment(os.Environ(), env)
	return cmd.Run()
}

func mergeEnvironment(base, overrides []string) []string {
	values := map[string]string{}
	order := make([]string, 0, len(base)+len(overrides))
	for _, entry := range append(append([]string(nil), base...), overrides...) {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}
	result := make([]string, 0, len(order))
	for _, key := range order {
		result = append(result, key+"="+values[key])
	}
	return result
}

type migrationRecord struct {
	Service         string `json:"service"`
	Version         int    `json:"version"`
	ExpectedVersion int    `json:"expectedVersion"`
	Dirty           bool   `json:"dirty"`
}

func checkMigrations(ctx context.Context, run commandRunner, stdout io.Writer, reportPath string, requireClean, requireHead bool) error {
	records := make([]migrationRecord, 0, len(migrations))
	var failures []string
	for _, spec := range migrations {
		record := migrationRecord{Service: spec.Service, ExpectedVersion: spec.Head, Dirty: true}
		var output bytes.Buffer
		args := []string{"-h", "postgres", "-U", "rapida_user", "-d", spec.Database, "-Atc", "SELECT version || '|' || dirty FROM schema_migrations LIMIT 1"}
		if err := run(ctx, "psql", args, nil, &output, io.Discard, nil); err != nil {
			records = append(records, record)
			failures = append(failures, fmt.Sprintf("%s migration query failed: %v", spec.Service, err))
			continue
		}
		version, dirty, err := parseMigrationState(output.String())
		if err != nil {
			records = append(records, record)
			failures = append(failures, fmt.Sprintf("%s: %v", spec.Service, err))
			continue
		}
		record.Version = version
		record.Dirty = dirty
		records = append(records, record)
		if requireClean && dirty {
			failures = append(failures, fmt.Sprintf("%s migration state version=%d dirty=%t, require clean", spec.Service, version, dirty))
		}
		if requireHead && version != spec.Head {
			failures = append(failures, fmt.Sprintf("%s migration state version=%d dirty=%t, want version=%d dirty=false", spec.Service, version, dirty, spec.Head))
		}
		fmt.Fprintf(stdout, "%s migration version=%d expected=%d dirty=%t\n", spec.Service, version, spec.Head, dirty)
	}
	if err := writeMigrationReport(reportPath, records); err != nil {
		failures = append(failures, fmt.Sprintf("write migration report: %v", err))
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func writeMigrationReport(path string, records []migrationRecord) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func parseMigrationState(value string) (int, bool, error) {
	parts := strings.Split(strings.TrimSpace(value), "|")
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("invalid migration state %q", strings.TrimSpace(value))
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false, fmt.Errorf("invalid migration version: %w", err)
	}
	dirty, err := strconv.ParseBool(parts[1])
	if err != nil {
		return 0, false, fmt.Errorf("invalid migration dirty flag: %w", err)
	}
	return version, dirty, nil
}

type healthResponse struct {
	Code    int             `json:"code"`
	Success bool            `json:"success"`
	Data    map[string]bool `json:"data"`
}

const readinessPostgresKey = "PSQL psql://postgres:5432"

func checkHealth(ctx context.Context, client *http.Client, timeoutPerService, interval time.Duration, readinessKey string, stdout io.Writer) error {
	for _, service := range services {
		serviceCtx, cancel := context.WithTimeout(ctx, timeoutPerService)
		if err := retry(serviceCtx, interval, func() error { return checkEndpoint(serviceCtx, client, service.URL+"/healthz/", "healthy") }); err != nil {
			cancel()
			return fmt.Errorf("%s liveness: %w", service.Name, err)
		}
		if err := retry(serviceCtx, interval, func() error {
			return checkEndpoint(serviceCtx, client, service.URL+"/readiness/", readinessKey)
		}); err != nil {
			cancel()
			return fmt.Errorf("%s readiness: %w", service.Name, err)
		}
		cancel()
		fmt.Fprintf(stdout, "%s healthy and ready\n", service.Name)
	}
	return nil
}

func retry(ctx context.Context, interval time.Duration, check func() error) error {
	var last error
	for {
		if err := check(); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("deadline exceeded; last response: %w", last)
		case <-time.After(interval):
		}
	}
}

func checkEndpoint(ctx context.Context, client *http.Client, url, requiredKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return err
	}
	var payload healthResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("HTTP %d invalid JSON %q", response.StatusCode, sanitize(string(body)))
	}
	value := payload.Data[requiredKey]
	if response.StatusCode != http.StatusOK || payload.Code != 200 || !payload.Success || !value {
		return fmt.Errorf("HTTP %d body=%s", response.StatusCode, sanitize(string(body)))
	}
	return nil
}

const defaultAssetPattern = `(?:src|href)=["']([^"']*/static/[^"']+\.[a-f0-9]{8,}\.[^"']*)["']`

type stringListFlag []string

func (values *stringListFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *stringListFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type proxyRoute struct {
	RPCPath  string
	Upstream string
}

type httpRoute struct {
	Path        string
	Upstream    string
	StatusCode  int
	ContentType string
	BodyCheck   func([]byte) error
}

const (
	talkProxyRPCPath  = "/talk_api.TalkService/GetAllAssistantConversation"
	webProxyRPCPath   = "/web_api.AuthenticationService/ForgotPassword"
	talkProxyUpstream = "assistant-api:9007"
	webProxyUpstream  = "web-api:9001"
)

var proxyRPCUpstreams = map[string]string{
	talkProxyRPCPath: talkProxyUpstream,
	webProxyRPCPath:  webProxyUpstream,
}

var httpRouteContracts = map[string]httpRoute{
	"/v1/__systemcheck__": {
		Path:        "/v1/__systemcheck__",
		Upstream:    webProxyUpstream,
		StatusCode:  http.StatusNotFound,
		ContentType: "text/plain",
		BodyCheck:   requireNonHTMLBody,
	},
	"/oauth/__systemcheck__": {
		Path:        "/oauth/__systemcheck__",
		Upstream:    webProxyUpstream,
		StatusCode:  http.StatusNotFound,
		ContentType: "text/plain",
		BodyCheck:   requireNonHTMLBody,
	},
	"/readiness/": {
		Path:        "/readiness/",
		Upstream:    webProxyUpstream,
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		BodyCheck: func(body []byte) error {
			return requireHealthResponse(body, readinessPostgresKey)
		},
	},
	"/healthz/": {
		Path:        "/healthz/",
		Upstream:    webProxyUpstream,
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		BodyCheck: func(body []byte) error {
			return requireHealthResponse(body, "healthy")
		},
	},
}

func checkUI(ctx context.Context, client *http.Client, baseURL, assetPattern string, proxyRouteValues, httpRouteValues []string) error {
	parsedProxyRoutes, err := parseProxyRoutes(proxyRouteValues)
	if err != nil {
		return err
	}
	parsedHTTPRoutes, err := parseHTTPRoutes(httpRouteValues)
	if err != nil {
		return err
	}
	body, err := getOK(ctx, client, baseURL+"/")
	if err != nil {
		return fmt.Errorf("nginx UI entry: %w", err)
	}
	pattern, err := regexp.Compile(assetPattern)
	if err != nil {
		return fmt.Errorf("invalid asset pattern: %w", err)
	}
	match := pattern.FindSubmatch(body)
	if len(match) == 0 {
		return errors.New("nginx UI entry contains no content-hashed static asset")
	}
	asset := match[0]
	if len(match) > 1 {
		asset = match[1]
	}
	_, err = getOK(ctx, client, baseURL+string(asset))
	if err != nil {
		return err
	}
	for _, route := range parsedProxyRoutes {
		if err := checkProxyRoute(ctx, client, baseURL, route); err != nil {
			return err
		}
	}
	for _, route := range parsedHTTPRoutes {
		if err := checkHTTPRoute(ctx, client, baseURL, route); err != nil {
			return err
		}
	}
	return nil
}

func parseHTTPRoutes(values []string) ([]httpRoute, error) {
	if len(values) != len(httpRouteContracts) {
		return nil, fmt.Errorf("HTTP routes must contain exactly %d registered paths", len(httpRouteContracts))
	}
	routes := make([]httpRoute, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		path, upstream, ok := strings.Cut(value, "=")
		contract, registered := httpRouteContracts[path]
		if !ok || !registered {
			return nil, fmt.Errorf("invalid HTTP route %q; path is not registered", value)
		}
		if upstream != contract.Upstream {
			return nil, fmt.Errorf("HTTP path %s requires upstream %s, got %s", path, contract.Upstream, upstream)
		}
		if seen[path] {
			return nil, fmt.Errorf("duplicate HTTP path %q", path)
		}
		seen[path] = true
		routes = append(routes, contract)
	}
	return routes, nil
}

func parseProxyRoute(value string) (proxyRoute, error) {
	rpcPath, upstream, ok := strings.Cut(value, "=")
	requiredUpstream, registered := proxyRPCUpstreams[rpcPath]
	if !ok || !registered {
		return proxyRoute{}, fmt.Errorf("invalid proxy route %q; RPC path is not registered", value)
	}
	parsedPath, err := url.ParseRequestURI(rpcPath)
	if err != nil || parsedPath.Path != rpcPath || parsedPath.RawQuery != "" || parsedPath.Fragment != "" {
		return proxyRoute{}, fmt.Errorf("invalid proxy RPC path %q", rpcPath)
	}
	if upstream != requiredUpstream {
		return proxyRoute{}, fmt.Errorf("proxy RPC path %s requires upstream %s, got %s", rpcPath, requiredUpstream, upstream)
	}
	return proxyRoute{RPCPath: rpcPath, Upstream: upstream}, nil
}

func parseProxyRoutes(values []string) ([]proxyRoute, error) {
	if len(values) != len(proxyRPCUpstreams) {
		return nil, fmt.Errorf("proxy routes must contain exactly %d registered RPC paths", len(proxyRPCUpstreams))
	}
	routes := make([]proxyRoute, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		route, err := parseProxyRoute(value)
		if err != nil {
			return nil, err
		}
		if seen[route.RPCPath] {
			return nil, fmt.Errorf("duplicate proxy RPC path %q", route.RPCPath)
		}
		seen[route.RPCPath] = true
		routes = append(routes, route)
	}
	return routes, nil
}

type proxyResponse struct {
	StatusCode  int
	ContentType string
	GRPCStatus  string
	Body        []byte
}

func checkProxyRoute(ctx context.Context, client *http.Client, baseURL string, route proxyRoute) error {
	proxied, err := probeGRPCWeb(ctx, client, strings.TrimRight(baseURL, "/")+route.RPCPath)
	if err != nil {
		return err
	}
	upstream, err := probeGRPCWeb(ctx, client, "http://"+route.Upstream+route.RPCPath)
	if err != nil {
		return err
	}
	if proxied.GRPCStatus == "12" || upstream.GRPCStatus == "12" {
		return fmt.Errorf("proxy route %s did not reach a registered RPC", route.RPCPath)
	}
	if proxied.StatusCode != upstream.StatusCode || proxied.ContentType != upstream.ContentType || proxied.GRPCStatus != upstream.GRPCStatus || !bytes.Equal(proxied.Body, upstream.Body) {
		return fmt.Errorf("proxy route %s does not match expected upstream %s", route.RPCPath, route.Upstream)
	}
	return nil
}

func checkHTTPRoute(ctx context.Context, client *http.Client, baseURL string, route httpRoute) error {
	proxied, err := probeHTTP(ctx, client, strings.TrimRight(baseURL, "/")+route.Path)
	if err != nil {
		return err
	}
	upstream, err := probeHTTP(ctx, client, "http://"+route.Upstream+route.Path)
	if err != nil {
		return err
	}
	if upstream.StatusCode != route.StatusCode || upstream.ContentType != route.ContentType {
		return fmt.Errorf("HTTP route %s upstream contract changed: HTTP %d content-type %q", route.Path, upstream.StatusCode, upstream.ContentType)
	}
	if err := route.BodyCheck(upstream.Body); err != nil {
		return fmt.Errorf("HTTP route %s upstream contract changed: %w", route.Path, err)
	}
	if proxied.StatusCode != upstream.StatusCode || proxied.ContentType != upstream.ContentType || !bytes.Equal(proxied.Body, upstream.Body) {
		return fmt.Errorf("HTTP route %s does not match expected upstream %s", route.Path, route.Upstream)
	}
	return nil
}

type httpResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func probeHTTP(ctx context.Context, client *http.Client, endpoint string) (httpResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return httpResponse{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return httpResponse{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return httpResponse{}, err
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	return httpResponse{StatusCode: response.StatusCode, ContentType: contentType, Body: body}, nil
}

func requireNonHTMLBody(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return errors.New("empty response body")
	}
	if bytes.Contains(bytes.ToLower(body), []byte("<html")) {
		return errors.New("response body is HTML")
	}
	return nil
}

func requireHealthResponse(body []byte, key string) error {
	var payload struct {
		Code    int             `json:"code"`
		Success bool            `json:"success"`
		Data    map[string]bool `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if payload.Code != http.StatusOK || !payload.Success || !payload.Data[key] {
		return fmt.Errorf("missing successful %q status", key)
	}
	return nil
}

func probeGRPCWeb(ctx context.Context, client *http.Client, endpoint string) (proxyResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte{0, 0, 0, 0, 0}))
	if err != nil {
		return proxyResponse{}, err
	}
	request.Header.Set("Content-Type", "application/grpc-web+proto")
	request.Header.Set("X-Grpc-Web", "1")
	response, err := client.Do(request)
	if err != nil {
		return proxyResponse{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return proxyResponse{}, err
	}
	if response.StatusCode != http.StatusOK {
		return proxyResponse{}, fmt.Errorf("%s returned HTTP %d", endpoint, response.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if !strings.HasPrefix(contentType, "application/grpc-web") {
		return proxyResponse{}, fmt.Errorf("%s returned non-gRPC-Web content type %q", endpoint, contentType)
	}
	grpcStatus := response.Header.Get("Grpc-Status")
	if grpcStatus == "" {
		grpcStatus = response.Trailer.Get("Grpc-Status")
	}
	if grpcStatus == "" {
		grpcStatus = grpcWebTrailerStatus(body)
	}
	if grpcStatus == "" {
		return proxyResponse{}, fmt.Errorf("%s returned no gRPC status", endpoint)
	}
	return proxyResponse{StatusCode: response.StatusCode, ContentType: contentType, GRPCStatus: grpcStatus, Body: body}, nil
}

func grpcWebTrailerStatus(body []byte) string {
	for len(body) >= 5 {
		flags := body[0]
		length := int(binary.BigEndian.Uint32(body[1:5]))
		body = body[5:]
		if length > len(body) {
			return ""
		}
		payload := body[:length]
		body = body[length:]
		if flags&0x80 == 0 {
			continue
		}
		for _, line := range strings.Split(string(payload), "\r\n") {
			key, value, ok := strings.Cut(line, ":")
			if ok && strings.EqualFold(strings.TrimSpace(key), "grpc-status") {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func getOK(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return body, nil
}

func assistantSmoke(ctx context.Context, run commandRunner, collection, baseURL, tmpfs string, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(tmpfs, 0700); err != nil {
		return err
	}
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	token := hex.EncodeToString(random)
	sql := fmt.Sprintf(`INSERT INTO organizations (id,name,description,size,industry,contact,status,created_by,updated_by) VALUES (1,'CI','CI','1','CI','ci@example.invalid','ACTIVE',1,1) ON CONFLICT (id) DO NOTHING;
INSERT INTO projects (id,organization_id,name,description,status,created_by,updated_by) VALUES (1,1,'CI','CI','ACTIVE',1,1) ON CONFLICT (id) DO NOTHING;
INSERT INTO user_auths (id,name,email,password,status,created_by,updated_by,source) VALUES (1,'CI','ci@example.invalid','unused','ACTIVE',1,1,'direct') ON CONFLICT (id) DO NOTHING;
INSERT INTO user_auth_tokens (id,user_auth_id,token_type,token,expire_at,status,created_by,updated_by) VALUES (1,1,'auth-token','%s',now()+interval '1 hour','ACTIVE',1,1) ON CONFLICT (id) DO UPDATE SET token=EXCLUDED.token,expire_at=EXCLUDED.expire_at,status='ACTIVE';
INSERT INTO user_organization_roles (id,user_auth_id,organization_id,role,status,created_by,updated_by) VALUES (1,1,1,'owner','ACTIVE',1,1) ON CONFLICT (id) DO NOTHING;
INSERT INTO user_project_roles (id,project_id,user_auth_id,role,status,created_by,updated_by) VALUES (1,1,1,'owner','ACTIVE',1,1) ON CONFLICT (id) DO NOTHING;`, token)
	var seedStderr bytes.Buffer
	outputIsolation := []string{"GITHUB_OUTPUT=", "GITHUB_STEP_SUMMARY="}
	if err := run(ctx, "psql", []string{"-v", "ON_ERROR_STOP=1", "-h", "postgres", "-U", "rapida_user", "-d", "web_db"}, strings.NewReader(sql), io.Discard, &seedStderr, outputIsolation); err != nil {
		writeSanitized(stderr, seedStderr.String(), token)
		return fmt.Errorf("seed assistant credentials: %w", err)
	}
	environmentPath := filepath.Join(tmpfs, "smoke.postman_environment.json")
	defer os.Remove(environmentPath)
	payload := map[string]any{"name": "assistant-system-smoke", "values": []map[string]any{
		{"key": "baseUrl", "value": baseURL, "enabled": true},
		{"key": "authToken", "value": token, "enabled": true},
		{"key": "authId", "value": "1", "enabled": true},
		{"key": "projectId", "value": "1", "enabled": true},
	}}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(environmentPath, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(environmentPath, 0600); err != nil {
		return err
	}
	args := []string{"run", collection, "--folder", "Smoke Flow", "--bail", "--environment", environmentPath}
	var newmanStdout, newmanStderr bytes.Buffer
	if err := run(ctx, "./node_modules/.bin/newman", args, nil, &newmanStdout, &newmanStderr, outputIsolation); err != nil {
		writeSanitized(stdout, newmanStdout.String(), token)
		writeSanitized(stderr, newmanStderr.String(), token)
		return fmt.Errorf("newman smoke failed: %w", err)
	}
	writeSanitized(stdout, newmanStdout.String(), token)
	writeSanitized(stderr, newmanStderr.String(), token)
	return nil
}

func writeSanitized(writer io.Writer, value string, secrets ...string) {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	_, _ = io.WriteString(writer, sanitize(value))
}

type diagnosticRecord struct {
	Service                  string          `json:"service"`
	Phase                    string          `json:"phase"`
	Timestamp                string          `json:"timestamp"`
	ExitCode                 int             `json:"exitCode"`
	Health                   map[string]bool `json:"health,omitempty"`
	ImageID                  string          `json:"imageId,omitempty"`
	MigrationVersion         int             `json:"migrationVersion,omitempty"`
	MigrationExpectedVersion int             `json:"migrationExpectedVersion,omitempty"`
	MigrationDirty           *bool           `json:"migrationDirty,omitempty"`
	Logs                     []string        `json:"logs,omitempty"`
}

type buildDiagnosticRecord struct {
	Timestamp  string         `json:"timestamp"`
	Builder    string         `json:"builder,omitempty"`
	Driver     string         `json:"driver,omitempty"`
	Status     string         `json:"status,omitempty"`
	Buildkit   string         `json:"buildkit,omitempty"`
	CacheScope string         `json:"cacheScope,omitempty"`
	DiskUsage  []string       `json:"diskUsage,omitempty"`
	Logs       []string       `json:"logs,omitempty"`
	Metadata   *buildMetadata `json:"metadata,omitempty"`
}

func collectDiagnostics(ctx context.Context, run commandRunner, project, directory string) error {
	if project == "" || directory == "" {
		return errors.New("compose project and directory are required")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	safe := false
	defer func() {
		if !safe {
			_ = os.RemoveAll(directory)
		}
	}()
	composeWrapper, err := resolveComposeWrapper()
	if err != nil {
		return err
	}
	records := make([]diagnosticRecord, 0, len(diagnosticServices))
	migrationRecords, err := collectMigrationReport(ctx, run, composeWrapper, project)
	if err != nil {
		return err
	}
	imageIDs, imageEvidencePresent, err := composeImageIDs(filepath.Join(directory, "compose-images.json"))
	if err != nil {
		return fmt.Errorf("parse Compose image diagnostics: %w", err)
	}
	for _, service := range diagnosticServices {
		record := diagnosticRecord{Service: service, Phase: "failure", Timestamp: time.Now().UTC().Format(time.RFC3339)}
		if imageEvidencePresent {
			record.ImageID = imageIDs[service]
			if record.ImageID == "" {
				return fmt.Errorf("Compose image diagnostics missing %s", service)
			}
		}
		if migration, ok := migrationRecords[service]; ok {
			record.MigrationVersion = migration.Version
			record.MigrationExpectedVersion = migration.ExpectedVersion
			dirty := migration.Dirty
			record.MigrationDirty = &dirty
		}
		var status bytes.Buffer
		if err := run(ctx, composeWrapper, []string{"-p", project, "-f", "docker-compose.yml", "-f", "docker-compose.ci.yml", "ps", "-a", "--format", "json", service}, nil, &status, io.Discard, nil); err == nil && status.Len() > 0 {
			if err := applyComposeStatus(&record, status.Bytes()); err != nil {
				return fmt.Errorf("parse %s Compose status: %w", service, err)
			}
		}
		var logs bytes.Buffer
		_ = run(ctx, composeWrapper, []string{"-p", project, "-f", "docker-compose.yml", "-f", "docker-compose.ci.yml", "logs", "--no-color", "--tail", "100", service}, nil, &logs, io.Discard, nil)
		record.Logs = selectedLogs(logs.String())
		records = append(records, record)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "diagnostics.json"), data, 0600); err != nil {
		return err
	}
	if err := collectBuildDiagnostics(ctx, run, directory); err != nil {
		return err
	}
	if err := sanitizeDirectory(directory, configuredSecrets()); err != nil {
		return err
	}
	safe = true
	return nil
}

func collectMigrationReport(ctx context.Context, run commandRunner, composeWrapper, project string) (map[string]migrationRecord, error) {
	var output bytes.Buffer
	args := []string{"-p", project, "-f", "docker-compose.yml", "-f", "docker-compose.ci.yml", "run", "--rm", "--no-deps", "--entrypoint", "cat", "test-runner", "/reports/migrations.json"}
	if err := run(ctx, composeWrapper, args, nil, &output, io.Discard, nil); err != nil {
		return nil, nil
	}
	var records []migrationRecord
	if err := json.Unmarshal(output.Bytes(), &records); err != nil {
		return nil, fmt.Errorf("parse migration report: %w", err)
	}
	result := map[string]migrationRecord{}
	for _, record := range records {
		if !containsMigrationService(record.Service) {
			return nil, fmt.Errorf("migration report contains unexpected service %q", record.Service)
		}
		if _, exists := result[record.Service]; exists {
			return nil, fmt.Errorf("migration report contains duplicate service %q", record.Service)
		}
		result[record.Service] = record
	}
	for _, spec := range migrations {
		if _, ok := result[spec.Service]; !ok {
			return nil, fmt.Errorf("migration report missing %s", spec.Service)
		}
	}
	return result, nil
}

func resolveComposeWrapper() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(directory, composeWrapperRelativePath)
		info, statErr := os.Lstat(candidate)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
				return "", fmt.Errorf("pinned Compose wrapper is not a regular executable: %s", candidate)
			}
			return candidate, nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("pinned Compose wrapper not found at %s", composeWrapperRelativePath)
		}
		directory = parent
	}
}

func containsMigrationService(service string) bool {
	for _, spec := range migrations {
		if spec.Service == service {
			return true
		}
	}
	return false
}

func applyComposeStatus(record *diagnosticRecord, raw []byte) error {
	var statuses []struct {
		ExitCode int    `json:"ExitCode"`
		Health   string `json:"Health"`
	}
	if err := json.Unmarshal(raw, &statuses); err != nil {
		var status struct {
			ExitCode int    `json:"ExitCode"`
			Health   string `json:"Health"`
		}
		if err := json.Unmarshal(bytes.TrimSpace(raw), &status); err != nil {
			return err
		}
		statuses = append(statuses, status)
	}
	if len(statuses) == 0 {
		return errors.New("empty Compose status")
	}
	record.ExitCode = statuses[0].ExitCode
	if statuses[0].Health != "" {
		record.Health = map[string]bool{"container": statuses[0].Health == "healthy"}
	}
	return nil
}

func selectedLogs(raw string) []string {
	interesting := regexp.MustCompile(`(?i)(error|fatal|panic|health|readiness|migration|failed)`)
	lines := strings.Split(raw, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if !interesting.MatchString(line) {
			continue
		}
		line = strings.TrimSpace(sanitize(line))
		if line == "" {
			continue
		}
		result = append(result, line)
		if len(result) == 100 {
			break
		}
	}
	return result
}

var tokenPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)\b(authorization|proxy-authorization|x-api-key|api[-_]?key|access[-_]?token|refresh[-_]?token|auth[-_]?token|token|password|passwd|secret|credential|client[-_]?secret|private[-_]?key|cookie|set-cookie)(["'=:\s]+)(?:Bearer\s+|Basic\s+)?([^\s"',;}\]]+)`), `${1}${2}[REDACTED]`},
	{regexp.MustCompile(`(?i)(postgres(?:ql)?://[^:/\s]+:)([^@\s]+)(@)`), `${1}[REDACTED]${3}`},
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`), `[REDACTED]`},
	{regexp.MustCompile(`\brpd-(?:prj|org)-[A-Za-z0-9_-]+\b`), `[REDACTED]`},
}

func sanitize(value string) string {
	for _, tokenPattern := range tokenPatterns {
		value = tokenPattern.pattern.ReplaceAllString(value, tokenPattern.replacement)
	}
	for _, secret := range configuredSecrets() {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	return value
}

func configuredSecrets() []string {
	values := []string{"rapida_db_password"}
	for _, value := range strings.Split(os.Getenv("SYSTEMCHECK_SECRETS"), ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func sanitizeDirectory(directory string, secrets []string) error {
	if directory == "" {
		return errors.New("directory is required")
	}
	unsafe := false
	unsafeReason := ""
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if !contains([]string{"diagnostics.json", "build-diagnostics.json", "buildkit-metadata.json"}, name) {
			unsafe = true
			unsafeReason = "unallowlisted artifact " + entry.Name()
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := strings.ReplaceAll(string(data), "[REDACTED]", "")
		if strings.Contains(text, `"Config":{"Env"`) || strings.Contains(text, `"Config": {"Env"`) || strings.Contains(text, `"values":[{"key":"authToken"`) {
			unsafe = true
			unsafeReason = "raw environment content in " + entry.Name()
		}
		for _, secret := range secrets {
			if secret != "" && strings.Contains(text, secret) {
				unsafe = true
				unsafeReason = "configured secret in " + entry.Name()
			}
		}
		for index, tokenPattern := range tokenPatterns {
			if match := tokenPattern.pattern.FindStringSubmatch(text); match != nil {
				unsafe = true
				label := "standalone"
				if len(match) > 1 {
					label = match[1]
				}
				unsafeReason = fmt.Sprintf("token-shaped value pattern %d (%s) in %s", index, label, entry.Name())
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if unsafe {
		if err := os.RemoveAll(directory); err != nil {
			return fmt.Errorf("unsafe diagnostics found and deletion failed: %w", err)
		}
		return fmt.Errorf("unsafe diagnostics found (%s); staging directory deleted", unsafeReason)
	}
	return nil
}

func cleanup(ctx context.Context, run commandRunner, project string, retries int, interval time.Duration) error {
	if project == "" {
		return errors.New("compose project is required")
	}
	if retries < 1 {
		return errors.New("retries must be positive")
	}
	filters := [][]string{
		{"ps", "-aq", "--filter", "label=com.docker.compose.project=" + project},
		{"network", "ls", "-q", "--filter", "label=com.docker.compose.project=" + project},
		{"volume", "ls", "-q", "--filter", "label=com.docker.compose.project=" + project},
	}
	for attempt := 0; attempt < retries; attempt++ {
		remaining := []string{}
		for _, args := range filters {
			var output bytes.Buffer
			if err := run(ctx, "docker", args, nil, &output, io.Discard, nil); err != nil {
				return err
			}
			if value := strings.TrimSpace(output.String()); value != "" {
				remaining = append(remaining, value)
			}
		}
		if len(remaining) == 0 {
			return nil
		}
		if attempt+1 < retries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval):
			}
		}
	}
	return fmt.Errorf("Compose resources remain for project %q", project)
}
