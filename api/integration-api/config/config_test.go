// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const baseIntegrationYAML = `
service_name: "integration-api"
service_id: 9004
host: "0.0.0.0"
port: 9004
secret: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
log_level: "debug"
env: "development"

postgres:
  host: "localhost"
  port: 5432
  db_name: "integration_db"
  auth:
    user: "rapida_user"
    password: "rapida_db_password"
  max_open_connection: 10
  max_ideal_connection: 10
  ssl_mode: "disable"

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0
  max_connection: 10
  auth:
    user: ""
    password: ""

asset_store:
  storage_type: "local"
  storage_path_prefix: "/tmp/rapida-data/assets/integration"

integration:
  host: "localhost:9004"
endpoint:
  host: "localhost:9005"
assistant:
  host: "localhost:9007"
web:
  host: "localhost:9001"
document:
  host: "http://localhost:9010"
ui:
  host: "http://localhost:3000"
`

func TestInitConfigDoesNotExposeMalformedAuthenticationSecret(t *testing.T) {
	const secret = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"
	path := filepath.Join(t.TempDir(), "integration.yaml")
	if err := os.WriteFile(path, []byte("secret: *"+secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENV_PATH", path)
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	_, err := InitConfig()
	if err == nil {
		t.Fatal("InitConfig() error = nil")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(output.String(), secret) {
		t.Fatal("InitConfig() exposed the authentication secret")
	}
}

func TestInitConfig(t *testing.T) {
	configPath := filepath.Join(os.TempDir(), "integration_test.yaml")
	err := os.WriteFile(configPath, []byte(baseIntegrationYAML), 0o644)
	if err != nil {
		t.Fatalf("Failed to create mock config file: %v", err)
	}
	defer os.Remove(configPath)

	os.Setenv("ENV_PATH", configPath)
	defer os.Unsetenv("ENV_PATH")

	vConfig, err := InitConfig()
	if err != nil {
		t.Fatalf("InitConfig returned an error: %v", err)
	}
	if vConfig == nil {
		t.Fatalf("vConfig is nil")
	}
	if vConfig.ConfigFileUsed() != configPath {
		t.Errorf("Expected config file used to be %v, but got %v", configPath, vConfig.ConfigFileUsed())
	}

	appConfig, err := GetApplicationConfig(vConfig)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig.PostgresConfig.DBName != "integration_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'integration_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.Assistant.Host != "localhost:9007" {
		t.Errorf("Expected Assistant.Host to be 'localhost:9007', but got %v", appConfig.Assistant.Host)
	}
}

func TestGetApplicationConfig(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	if err := vConfig.ReadConfig(strings.NewReader(baseIntegrationYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(vConfig)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil {
		t.Fatalf("appConfig is nil")
	}
	if appConfig.ServiceID != 9004 {
		t.Fatalf("ServiceID = %d, want 9004", appConfig.ServiceID)
	}

	if appConfig.PostgresConfig.DBName != "integration_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'integration_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.AssetStoreConfig.StorageType != "local" {
		t.Errorf("Expected AssetStoreConfig.StorageType to be 'local', but got %v", appConfig.AssetStoreConfig.StorageType)
	}
	if appConfig.RedisConfig.Host != "127.0.0.1" || appConfig.RedisConfig.Port != 6379 {
		t.Errorf("Redis Config mismatch: Host=%v, Port=%v", appConfig.RedisConfig.Host, appConfig.RedisConfig.Port)
	}
	if appConfig.Integration.Host != "localhost:9004" {
		t.Errorf("Expected Integration.Host to be 'localhost:9004', but got %v", appConfig.Integration.Host)
	}
	if appConfig.Endpoint.Host != "localhost:9005" {
		t.Errorf("Expected Endpoint.Host to be 'localhost:9005', but got %v", appConfig.Endpoint.Host)
	}
	if appConfig.Assistant.Host != "localhost:9007" {
		t.Errorf("Expected Assistant.Host to be 'localhost:9007', but got %v", appConfig.Assistant.Host)
	}
	if appConfig.Web.Host != "localhost:9001" {
		t.Errorf("Expected Web.Host to be 'localhost:9001', but got %v", appConfig.Web.Host)
	}
	if appConfig.Document.Host != "http://localhost:9010" {
		t.Errorf("Expected Document.Host to be 'http://localhost:9010', but got %v", appConfig.Document.Host)
	}
	if appConfig.Ui.Host != "http://localhost:3000" {
		t.Errorf("Expected Ui.Host to be 'http://localhost:3000', but got %v", appConfig.Ui.Host)
	}
}

func TestGetApplicationConfigRejectsMissingServiceID(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	configYAML := strings.Replace(baseIntegrationYAML, "service_id: 9004\n", "", 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}
	if _, err := GetApplicationConfig(vConfig); err == nil {
		t.Fatal("GetApplicationConfig() error = nil")
	}
}

func TestGetApplicationConfigRejectsInvalidAuthenticationSecret(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	configYAML := strings.Replace(baseIntegrationYAML, `secret: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="`, `secret: "rpd_pks"`, 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatal(err)
	}
	if _, err := GetApplicationConfig(vConfig); err == nil {
		t.Fatal("GetApplicationConfig() accepted an invalid authentication secret")
	}
}

func TestGetApplicationConfigDoesNotExposeMalformedAuthenticationSecret(t *testing.T) {
	const secret = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	configYAML := strings.Replace(baseIntegrationYAML, `secret: "`+secret+`"`, "secret:\n  - \""+secret+"\"", 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	_, err := GetApplicationConfig(vConfig)
	if err == nil {
		t.Fatal("GetApplicationConfig() error = nil")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(output.String(), secret) {
		t.Fatal("GetApplicationConfig() exposed the authentication secret")
	}
}

func TestGetApplicationConfigRejectsLegacyServiceAssertionConfig(t *testing.T) {
	for name, serviceAssertion := range map[string]string{
		"empty": `service_assertion: {}`,
		"null":  `service_assertion: null`,
		"old only": `service_assertion:
  private_key_file: "/run/secrets/integration-service-assertion-key"
  trusted_public_keys: ["legacy"]`,
		"mixed": `service_assertion:
  key_file: "/run/secrets/internal-service-assertion-key"
  trusted_public_keys: ["legacy"]`,
		"unknown": `service_assertion:
  key_file: "/run/secrets/internal-service-assertion-key"
  unexpected: true`,
	} {
		t.Run(name, func(t *testing.T) {
			vConfig := viper.New()
			vConfig.SetConfigType("yaml")
			configYAML := baseIntegrationYAML + "\n" + serviceAssertion
			if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
				t.Fatal(err)
			}
			if _, err := GetApplicationConfig(vConfig); err == nil {
				t.Fatal("GetApplicationConfig() accepted legacy service assertion configuration")
			}
		})
	}
}
