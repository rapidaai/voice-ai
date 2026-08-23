// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const baseWebYAML = `
service_name: "web-api"
service_id: 9001
host: "0.0.0.0"
port: 9001
log_level: "debug"
secret: "rpd_pks"
env: "development"

postgres:
  host: "localhost"
  port: 5432
  db_name: "web_db"
  auth:
    user: "rapida_user"
    password: "rapida_db_password"
  max_open_connection: 10
  max_ideal_connection: 10
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  db: 0
  max_connection: 5
  auth:
    user: ""
    password: ""

asset_store:
  storage_type: "local"
  storage_path_prefix: "/tmp/rapida-data/assets/web"

oauth2:
  google_client_id: "google-client-id"
  google_client_secret: "google-client-secret"

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

func TestInitConfig(t *testing.T) {
	configPath := filepath.Join(os.TempDir(), "web_test.yaml")
	err := os.WriteFile(configPath, []byte(baseWebYAML), 0o644)
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
	if appConfig.PostgresConfig.DBName != "web_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'web_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.Assistant.Host != "localhost:9007" {
		t.Errorf("Expected Assistant.Host to be 'localhost:9007', but got %v", appConfig.Assistant.Host)
	}
}

func TestGetApplicationConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(baseWebYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(v)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil {
		t.Fatalf("appConfig is nil")
	}
	if appConfig.ServiceID != 9001 {
		t.Fatalf("ServiceID = %d, want 9001", appConfig.ServiceID)
	}

	if appConfig.Name != "web-api" {
		t.Errorf("Expected ServiceName to be 'web-api', but got %v", appConfig.Name)
	}
	if appConfig.PostgresConfig.Host != "localhost" {
		t.Errorf("Expected PostgresConfig.Host to be 'localhost', but got %v", appConfig.PostgresConfig.Host)
	}
	if appConfig.PostgresConfig.DBName != "web_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'web_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.RedisConfig.Host != "localhost" || appConfig.RedisConfig.Port != 6379 {
		t.Errorf("Redis Config mismatch: Host=%v, Port=%v", appConfig.RedisConfig.Host, appConfig.RedisConfig.Port)
	}
	if appConfig.AssetStoreConfig.StoragePathPrefix != "/tmp/rapida-data/assets/web" {
		t.Errorf("Expected AssetStoreConfig.StoragePathPrefix to be /tmp/rapida-data/assets/web, but got %v", appConfig.AssetStoreConfig.StoragePathPrefix)
	}

	if appConfig.Document.Host != "http://localhost:9010" {
		t.Errorf("Expected Document.Host to be 'http://localhost:9010', but got %v", appConfig.Document.Host)
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
	if appConfig.Ui.Host != "http://localhost:3000" {
		t.Errorf("Expected Ui.Host to be 'http://localhost:3000', but got %v", appConfig.Ui.Host)
	}
}

func TestGetApplicationConfigRejectsMissingServiceID(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	configYAML := strings.Replace(baseWebYAML, "service_id: 9001\n", "", 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}
	if _, err := GetApplicationConfig(vConfig); err == nil {
		t.Fatal("GetApplicationConfig() error = nil")
	}
}
