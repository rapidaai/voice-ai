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

const baseEndpointYAML = `
service_name: "endpoint-api"
service_id: 9005
host: "0.0.0.0"
port: 9005
log_level: "debug"
secret: "rpd_pks"
env: "development"

postgres:
  host: "localhost"
  port: 5432
  db_name: "endpoint_db"
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
  storage_path_prefix: "/tmp/rapida-data/assets/endpoint"

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
	configPath := filepath.Join(os.TempDir(), "endpoint_test.yaml")
	err := os.WriteFile(configPath, []byte(baseEndpointYAML), 0o644)
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
	if appConfig.PostgresConfig.DBName != "endpoint_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'endpoint_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.Assistant.Host != "localhost:9007" {
		t.Errorf("Expected Assistant.Host to be 'localhost:9007', but got %v", appConfig.Assistant.Host)
	}
}

func TestGetApplicationConfig(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	if err := vConfig.ReadConfig(strings.NewReader(baseEndpointYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(vConfig)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil {
		t.Fatalf("appConfig is nil")
	}
	if appConfig.ServiceID != 9005 {
		t.Fatalf("ServiceID = %d, want 9005", appConfig.ServiceID)
	}

	if appConfig.PostgresConfig.DBName != "endpoint_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'endpoint_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.AssetStoreConfig.StorageType != "local" {
		t.Errorf("Expected AssetStoreConfig.StorageType to be 'local', but got %v", appConfig.AssetStoreConfig.StorageType)
	}
	if appConfig.RedisConfig.Host != "localhost" || appConfig.RedisConfig.Port != 6379 {
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
	configYAML := strings.Replace(baseEndpointYAML, "service_id: 9005\n", "", 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}
	if _, err := GetApplicationConfig(vConfig); err == nil {
		t.Fatal("GetApplicationConfig() error = nil")
	}
}
