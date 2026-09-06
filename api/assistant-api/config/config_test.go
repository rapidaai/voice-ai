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
	"time"

	"github.com/spf13/viper"
)

const baseAssistantYAML = `
service_name: "workflow-api"
service_id: 9007
host: "0.0.0.0"
port: 9007
log_level: "debug"
secret: "rpd_pks"
env: "development"

postgres:
  host: "localhost"
  port: 5432
  db_name: "assistant_db"
  auth:
    user: "rapida_user"
    password: "rapida_db_password"
  max_open_connection: 50
  max_ideal_connection: 25
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
  storage_path_prefix: "/tmp/rapida-data/assets/workflow"

opensearch:
  schema: "http"
  host: "localhost"
  port: 9200
  max_retries: 3
  max_connection: 10

integration:
  host: "localhost:9004"
endpoint:
  host: "localhost:9005"
assistant:
  host: "localhost:9007"
  public: "integral-presently-cub.ngrok-free.app"
web:
  host: "localhost:9001"
document:
  host: "http://localhost:9010"
ui:
  host: "http://localhost:3000"
sip:
  server: "0.0.0.0"
  instance_id: "assistant-test-01"
  port: 5070
  max_concurrent_calls: 42
  call_admission_cps: 7
  call_admission_burst: 11
  inbound:
    answer_mode: "answer_immediately"
    min_ring_duration: 0s
    max_ring_duration: 30s
    ack_timeout: 5s
`

func TestInitConfig(t *testing.T) {
	configPath := filepath.Join(os.TempDir(), "assistant_test.yaml")
	err := os.WriteFile(configPath, []byte(baseAssistantYAML), 0o644)
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
	if appConfig.PostgresConfig.DBName != "assistant_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'assistant_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.Assistant.Public != "integral-presently-cub.ngrok-free.app" {
		t.Errorf("Expected Assistant.Public to be 'integral-presently-cub.ngrok-free.app', but got %v", appConfig.Assistant.Public)
	}
	if appConfig.SIPConfig == nil {
		t.Fatal("Expected SIPConfig to be parsed")
	}
	if appConfig.SIPConfig.Inbound.AnswerMode != "answer_immediately" {
		t.Errorf("Expected nested SIP inbound answer mode, got %q", appConfig.SIPConfig.Inbound.AnswerMode)
	}
	if appConfig.SIPConfig.Inbound.ACKTimeout.String() != "5s" {
		t.Errorf("Expected nested SIP inbound ack timeout 5s, got %s", appConfig.SIPConfig.Inbound.ACKTimeout)
	}
	if appConfig.SIPConfig.MaxConcurrentCalls != 42 {
		t.Errorf("Expected SIP max concurrent calls 42, got %d", appConfig.SIPConfig.MaxConcurrentCalls)
	}
	if appConfig.SIPConfig.CallAdmissionCPS != 7 {
		t.Errorf("Expected SIP call admission cps 7, got %d", appConfig.SIPConfig.CallAdmissionCPS)
	}
	if appConfig.SIPConfig.CallAdmissionBurst != 11 {
		t.Errorf("Expected SIP call admission burst 11, got %d", appConfig.SIPConfig.CallAdmissionBurst)
	}
}

func TestGetApplicationConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(baseAssistantYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(v)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil {
		t.Fatalf("appConfig is nil")
	}
	if appConfig.ServiceID != 9007 {
		t.Fatalf("ServiceID = %d, want 9007", appConfig.ServiceID)
	}

	if appConfig.PostgresConfig.DBName != "assistant_db" {
		t.Errorf("Expected PostgresConfig.DBName to be 'assistant_db', but got %v", appConfig.PostgresConfig.DBName)
	}
	if appConfig.AssetStoreConfig.StorageType != "local" {
		t.Errorf("Expected AssetStoreConfig.StorageType to be 'local', but got %v", appConfig.AssetStoreConfig.StorageType)
	}
	if appConfig.Assistant.Host != "localhost:9007" {
		t.Errorf("Expected Assistant.Host to be 'localhost:9007', but got %v", appConfig.Assistant.Host)
	}
	if appConfig.Assistant.Public != "integral-presently-cub.ngrok-free.app" {
		t.Errorf("Expected Assistant.Public to be 'integral-presently-cub.ngrok-free.app', but got %v", appConfig.Assistant.Public)
	}
}

func TestGetApplicationConfigRejectsMissingServiceID(t *testing.T) {
	vConfig := viper.New()
	vConfig.SetConfigType("yaml")
	configYAML := strings.Replace(baseAssistantYAML, "service_id: 9007\n", "", 1)
	if err := vConfig.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}
	if _, err := GetApplicationConfig(vConfig); err == nil {
		t.Fatal("GetApplicationConfig() error = nil")
	}
}

func TestGetApplicationConfig_ParsesNestedSIPInboundConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")

	inboundYAML := strings.Replace(baseAssistantYAML, `    answer_mode: "answer_immediately"
    min_ring_duration: 0s
    max_ring_duration: 30s
    ack_timeout: 5s`, `    answer_mode: "answer_after_min_ring_ms"
    min_ring_duration: 750ms
    max_ring_duration: 45s
    ack_timeout: 7s`, 1)

	if err := v.ReadConfig(strings.NewReader(inboundYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(v)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig.SIPConfig == nil {
		t.Fatal("Expected SIPConfig to be parsed")
	}

	inboundConfig := appConfig.SIPConfig.Inbound
	if inboundConfig.AnswerMode != "answer_after_min_ring_ms" {
		t.Fatalf("Inbound.AnswerMode = %q, want %q", inboundConfig.AnswerMode, "answer_after_min_ring_ms")
	}
	if inboundConfig.MinRingDuration != 750*time.Millisecond {
		t.Fatalf("Inbound.MinRingDuration = %s, want 750ms", inboundConfig.MinRingDuration)
	}
	if inboundConfig.MaxRingDuration != 45*time.Second {
		t.Fatalf("Inbound.MaxRingDuration = %s, want 45s", inboundConfig.MaxRingDuration)
	}
	if inboundConfig.ACKTimeout != 7*time.Second {
		t.Fatalf("Inbound.ACKTimeout = %s, want 7s", inboundConfig.ACKTimeout)
	}
}

func TestGetApplicationConfig_RequiresSIPInstanceID(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	configYAML := strings.Replace(baseAssistantYAML, `  instance_id: "assistant-test-01"
`, "", 1)

	if err := v.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	if _, err := GetApplicationConfig(v); err == nil {
		t.Fatalf("expected missing SIP instance_id to fail validation")
	}
}

func TestGetApplicationConfig_TelemetryParsing(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	telemetryYAML := baseAssistantYAML + `
telemetry:
  type: "otlp_http"
  otlp_http:
    endpoint: "otel-collector:4318"
    protocol: "http/protobuf"
    headers: "Authorization=Bearer test-token"
    insecure: true
`
	if err := v.ReadConfig(strings.NewReader(telemetryYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	appConfig, err := GetApplicationConfig(v)
	if err != nil {
		t.Fatalf("GetApplicationConfig returned an error: %v", err)
	}
	if appConfig == nil || appConfig.TelemetryConfig == nil {
		t.Fatalf("telemetry config is nil")
	}
	if got := string(appConfig.TelemetryConfig.Type()); got != "otlp_http" {
		t.Fatalf("TelemetryConfig.Type() = %v, want otlp_http", got)
	}
	if appConfig.TelemetryConfig.OTLPHTTP == nil {
		t.Fatalf("TelemetryConfig.OTLPHTTP is nil")
	}
	if appConfig.TelemetryConfig.OTLPHTTP.Endpoint != "otel-collector:4318" {
		t.Fatalf("TelemetryConfig.OTLPHTTP.Endpoint = %q, want %q", appConfig.TelemetryConfig.OTLPHTTP.Endpoint, "otel-collector:4318")
	}
}

func TestGetApplicationConfig_RejectsIncompleteNestedOpenSearchConfigs(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	configYAML := baseAssistantYAML + `
telemetry:
  type: "opensearch"
  opensearch:
    schema: "http"
`
	if err := v.ReadConfig(strings.NewReader(configYAML)); err != nil {
		t.Fatalf("ReadConfig returned an error: %v", err)
	}

	if _, err := GetApplicationConfig(v); err == nil {
		t.Fatalf("expected incomplete telemetry opensearch config to fail validation")
	}
}
