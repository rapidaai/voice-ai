package sip_registration

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error    { return nil }
func (t *testPostgresConnector) Name() string                         { return "test-postgres" }
func (t *testPostgresConnector) IsConnected(ctx context.Context) bool { return t.db != nil }
func (t *testPostgresConnector) Disconnect(ctx context.Context) error { return nil }
func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.db.WithContext(ctx).Raw(qry).Scan(dest).Error
}
func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB { return t.db.WithContext(ctx) }

func newTestManager(t *testing.T) (*manager, *gorm.DB, context.Context) {
	t.Helper()
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "registration.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sqlite db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	schema := []string{
		`CREATE TABLE assistants (
			id INTEGER PRIMARY KEY,
			project_id BIGINT,
			organization_id BIGINT
		)`,
		`CREATE TABLE assistant_phone_deployments (
			id INTEGER PRIMARY KEY,
			assistant_id BIGINT,
			status TEXT,
			telephony_provider TEXT
		)`,
		`CREATE TABLE assistant_deployment_telephony_options (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_date DATETIME,
			assistant_deployment_telephony_id BIGINT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			status TEXT,
			created_actor_type TEXT,
			created_actor_id BIGINT,
			updated_actor_type TEXT,
			updated_actor_id BIGINT,
			updated_date DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_adto_deployment_key
			ON assistant_deployment_telephony_options(assistant_deployment_telephony_id, key)`,
	}
	for _, ddl := range schema {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("failed to initialize schema: %v", err)
		}
	}

	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return &manager{
		logger:   logger,
		postgres: &testPostgresConnector{db: db},
	}, db, ctx
}

func insertSIPDeployment(t *testing.T, db *gorm.DB, deploymentID, assistantID uint64, phone string, sipStatus RegistrationStatus) {
	t.Helper()

	options := map[string]string{
		OptKeySIPInbound:   "true",
		OptKeyCredentialID: "101",
	}
	if phone != "" {
		options[OptKeyPhone] = phone
	}
	if sipStatus != "" {
		options[OptKeySIPStatus] = string(sipStatus)
	}
	insertSIPDeploymentWithOptions(t, db, deploymentID, assistantID, options)
}

func insertSIPDeploymentWithOptions(t *testing.T, db *gorm.DB, deploymentID, assistantID uint64, options map[string]string) {
	t.Helper()

	if err := db.Exec(
		`INSERT INTO assistant_phone_deployments (id, assistant_id, status, telephony_provider)
		 VALUES (?, ?, ?, ?)`,
		deploymentID, assistantID, "ACTIVE", "sip",
	).Error; err != nil {
		t.Fatalf("failed creating deployment: %v", err)
	}

	for k, v := range options {
		if err := db.Exec(
			`INSERT INTO assistant_deployment_telephony_options
			 (assistant_deployment_telephony_id, key, value, status, created_actor_type, created_actor_id, updated_actor_type, updated_actor_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			deploymentID, k, v, "ACTIVE", "user", 1, "user", 1,
		).Error; err != nil {
			t.Fatalf("failed creating option %s: %v", k, err)
		}
	}
}

func getOptionValue(t *testing.T, db *gorm.DB, deploymentID uint64, key string) string {
	t.Helper()
	var opt internal_assistant_entity.AssistantDeploymentTelephonyOption
	if err := db.Where("assistant_deployment_telephony_id = ? AND key = ?", deploymentID, key).First(&opt).Error; err != nil {
		t.Fatalf("failed loading option %s for deployment %d: %v", key, deploymentID, err)
	}
	return opt.Value
}

func TestLoadRecords_PrePipelineDedupe_PrefersActiveAndMarksDropped(t *testing.T) {
	m, db, ctx := newTestManager(t)

	// Duplicate DID differing only by '+' formatting.
	insertSIPDeployment(t, db, 1001, 501, "+15551234567", StatusActive)
	insertSIPDeployment(t, db, 1002, 502, "15551234567", "")

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 deduped record, got %d", len(records))
	}
	if records[0].DeploymentID != 1001 {
		t.Fatalf("expected active deployment 1001 to win, got %d", records[0].DeploymentID)
	}

	status := getOptionValue(t, db, 1002, OptKeySIPStatus)
	if status != string(StatusConfigError) {
		t.Fatalf("expected loser status=%s, got %s", StatusConfigError, status)
	}
	failureClass := getOptionValue(t, db, 1002, OptKeySIPFailureClass)
	if failureClass != string(RegistrationFailureClassDuplicate) {
		t.Fatalf("expected loser failure_class=%s, got %s", RegistrationFailureClassDuplicate, failureClass)
	}
	failureReason := getOptionValue(t, db, 1002, OptKeySIPFailureReason)
	if failureReason != string(RegistrationFailureReasonDuplicateDID) {
		t.Fatalf("expected loser failure_reason=%s, got %s", RegistrationFailureReasonDuplicateDID, failureReason)
	}
	reason := getOptionValue(t, db, 1002, OptKeySIPError)
	if !strings.Contains(reason, "Duplicate DID +15551234567") || !strings.Contains(reason, fmt.Sprintf("deployment=%d", uint64(1001))) {
		t.Fatalf("unexpected loser reason: %s", reason)
	}
	retry := getOptionValue(t, db, 1002, OptKeySIPRetry)
	if retry != "0" {
		t.Fatalf("expected loser retry_count=0, got %s", retry)
	}
}

func TestLoadRecords_PrePipelineDedupe_PrefersLatestWhenNoActive(t *testing.T) {
	m, db, ctx := newTestManager(t)

	insertSIPDeployment(t, db, 2001, 601, "+14155550100", "")
	insertSIPDeployment(t, db, 2002, 602, "14155550100", "")

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 deduped record, got %d", len(records))
	}
	if records[0].DeploymentID != 2002 {
		t.Fatalf("expected latest deployment 2002 to win, got %d", records[0].DeploymentID)
	}
}

func TestLoadRecords_MissingPhoneMarksConfigError(t *testing.T) {
	m, db, ctx := newTestManager(t)

	insertSIPDeploymentWithOptions(t, db, 3001, 701, map[string]string{
		OptKeyCredentialID: "101",
		OptKeySIPInbound:   "true",
	})

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no registration record, got %d", len(records))
	}

	if status := getOptionValue(t, db, 3001, OptKeySIPStatus); status != string(StatusConfigError) {
		t.Fatalf("expected status=%s, got %s", StatusConfigError, status)
	}
	if failureClass := getOptionValue(t, db, 3001, OptKeySIPFailureClass); failureClass != string(RegistrationFailureClassConfig) {
		t.Fatalf("expected failure_class=%s, got %s", RegistrationFailureClassConfig, failureClass)
	}
	if failureReason := getOptionValue(t, db, 3001, OptKeySIPFailureReason); failureReason != string(RegistrationFailureReasonMissingDID) {
		t.Fatalf("expected failure_reason=%s, got %s", RegistrationFailureReasonMissingDID, failureReason)
	}
}

func TestLoadRecords_MissingCredentialMarksConfigError(t *testing.T) {
	m, db, ctx := newTestManager(t)

	insertSIPDeploymentWithOptions(t, db, 3002, 702, map[string]string{
		OptKeyPhone:      "+14155550101",
		OptKeySIPInbound: "true",
	})

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no registration record, got %d", len(records))
	}

	if status := getOptionValue(t, db, 3002, OptKeySIPStatus); status != string(StatusConfigError) {
		t.Fatalf("expected status=%s, got %s", StatusConfigError, status)
	}
	if failureClass := getOptionValue(t, db, 3002, OptKeySIPFailureClass); failureClass != string(RegistrationFailureClassConfig) {
		t.Fatalf("expected failure_class=%s, got %s", RegistrationFailureClassConfig, failureClass)
	}
	if failureReason := getOptionValue(t, db, 3002, OptKeySIPFailureReason); failureReason != string(RegistrationFailureReasonMissingCredentialID) {
		t.Fatalf("expected failure_reason=%s, got %s", RegistrationFailureReasonMissingCredentialID, failureReason)
	}
}

func TestLoadRecords_SkipsTerminalStatuses(t *testing.T) {
	m, db, ctx := newTestManager(t)

	terminalStatuses := []RegistrationStatus{
		StatusDisabled,
		StatusRejected,
		StatusConfigError,
		StatusUnreachable,
	}
	for index, status := range terminalStatuses {
		insertSIPDeployment(t, db, uint64(4001+index), uint64(801+index), fmt.Sprintf("+1415555020%d", index), status)
	}
	insertSIPDeployment(t, db, 4010, 810, "+14155550210", StatusFailed)

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected only non-terminal failed status to be retried, got %d records", len(records))
	}
	if records[0].DeploymentID != 4010 {
		t.Fatalf("expected failed deployment 4010 to remain retryable, got %d", records[0].DeploymentID)
	}
}
