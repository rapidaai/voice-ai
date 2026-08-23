package systemcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNginxConfigRoutesWebAPIEndpointsAndUIRoot(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", "docker", "nginx", "nginx.conf")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read nginx config: %v", err)
	}
	config := string(data)

	for _, path := range []string{"/v1/", "/oauth/", "/readiness/", "/healthz/"} {
		block := nginxLocationBlock(t, config, "location "+path+" {")
		for _, directive := range []string{
			"proxy_pass http://web-api:9001;",
			"proxy_set_header Host $http_host;",
			"proxy_set_header X-Real-IP $remote_addr;",
			"proxy_set_header X-Forwarded-For $remote_addr;",
			"proxy_set_header X-Forwarded-Proto $scheme;",
		} {
			if !strings.Contains(block, directive) {
				t.Errorf("nginx location %s missing %q", path, directive)
			}
		}
	}

	rootBlock := nginxLocationBlock(t, config, "location / {")
	if !strings.Contains(rootBlock, "proxy_pass http://ui:3000;") {
		t.Error("nginx root location does not proxy to UI")
	}
	if strings.Contains(rootBlock, "web-api:9001") {
		t.Error("nginx root location must not proxy to web-api")
	}
}

func nginxLocationBlock(t *testing.T, config, declaration string) string {
	t.Helper()

	start := strings.Index(config, declaration)
	if start == -1 {
		t.Fatalf("nginx config missing %q", declaration)
	}
	open := start + strings.Index(config[start:], "{")
	depth := 0
	for index := open; index < len(config); index++ {
		switch config[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return config[start : index+1]
			}
		}
	}

	t.Fatalf("nginx location %q has no closing brace", declaration)
	return ""
}
