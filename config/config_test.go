// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rapidaai/config"
	"gopkg.in/yaml.v3"
)

const developmentAuthenticationSecret = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

func TestOrganizationCredentialFingerprintKeyDoesNotValidateConfiguration(t *testing.T) {
	cfg := &config.AppConfig{Secret: "not-base64"}
	if got := base64.StdEncoding.EncodeToString(cfg.OrganizationCredentialFingerprintKey()); got != "QUi9N0LYMibMuExU11xSaICvRvsyVQFWjk2jzEsIjzc=" {
		t.Fatalf("organization key = %q", got)
	}
}

func TestServiceAuthenticationKeyDoesNotValidateConfiguration(t *testing.T) {
	cfg := &config.AppConfig{Secret: "not-base64"}
	if got := base64.StdEncoding.EncodeToString(cfg.ServiceAuthenticationKey()); got != "RgEs6Za3l1oVDyeQhMPsQ+HITHlKvH9WoRT1JT7fFmg=" {
		t.Fatalf("service key = %q", got)
	}
}

func TestAuthenticationSecretDerivesDetachedDomainKeys(t *testing.T) {
	cfg := &config.AppConfig{Secret: developmentAuthenticationSecret}
	serviceKey := cfg.ServiceAuthenticationKey()
	organizationKey := cfg.OrganizationCredentialFingerprintKey()
	if got := base64.StdEncoding.EncodeToString(serviceKey); got != "veUxBjUOPPj8M4ZdOuRNqMuIzamLmE4S2upg4QeQL40=" {
		t.Fatalf("service key = %q", got)
	}
	if got := base64.StdEncoding.EncodeToString(organizationKey); got != "xTPPoHh88NQBGstewC5aEiZ7gr8YCpQYIylE6SElFww=" {
		t.Fatalf("organization key = %q", got)
	}
	if reflect.DeepEqual(serviceKey, organizationKey) {
		t.Fatal("domain keys are equal")
	}
	serviceKey[0] ^= 0xff
	organizationKey[0] ^= 0xff
	secondServiceKey := cfg.ServiceAuthenticationKey()
	secondOrganizationKey := cfg.OrganizationCredentialFingerprintKey()
	if reflect.DeepEqual(serviceKey, secondServiceKey) || reflect.DeepEqual(organizationKey, secondOrganizationKey) {
		t.Fatal("derived key mutation was retained")
	}
}

func TestDeploymentUsesOneDevelopmentAuthenticationSecret(t *testing.T) {
	paths := []string{
		"env/web.yml", "docker/web-api/web.yml", "env/assistant.yaml",
		"docker/assistant-api/assistant.yml", "docker/assistant-api/assistant.knowledge.yml",
		"env/endpoint.yml", "docker/endpoint-api/endpoint.yml", "env/integration.yml",
		"docker/integration-api/integration.yml", "env/document.yaml", "docker/document-api/config.yaml",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", path))
			if err != nil {
				t.Fatal(err)
			}
			var parsed struct {
				Secret           string      `yaml:"secret"`
				ServiceAssertion interface{} `yaml:"service_assertion"`
			}
			if err := yaml.Unmarshal(source, &parsed); err != nil {
				t.Fatal(err)
			}
			if parsed.Secret != developmentAuthenticationSecret || parsed.ServiceAssertion != nil {
				t.Fatalf("authentication config = secret %q, service_assertion %#v", parsed.Secret, parsed.ServiceAssertion)
			}
		})
	}
}

func TestComposeDoesNotMountAuthenticationKeyFiles(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "internal-service-assertion-key") || strings.Contains(string(source), "INTERNAL_SERVICE_ASSERTION_KEY_FILE") {
		t.Fatal("base Compose configuration retains authentication key-file wiring")
	}
	knowledgeSource, err := os.ReadFile(filepath.Join("..", "docker-compose.knowledge.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(knowledgeSource), "internal-service-assertion-key") || strings.Contains(string(knowledgeSource), "INTERNAL_SERVICE_ASSERTION_KEY_FILE") {
		t.Fatal("knowledge Compose configuration retains authentication key-file wiring")
	}
}
