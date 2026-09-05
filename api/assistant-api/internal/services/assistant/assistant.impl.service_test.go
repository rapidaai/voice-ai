package internal_assistant_service

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAssistantServiceGetAssistantWithPhoneDeploymentById(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/assistant-service.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE assistants (
		id INTEGER PRIMARY KEY,
		project_id INTEGER,
		organization_id INTEGER,
		visibility TEXT,
		status TEXT
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_phone_deployments (
		id INTEGER PRIMARY KEY,
		assistant_id INTEGER,
		telephony_provider TEXT,
		status TEXT,
		created_date DATETIME
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_deployment_telephony_options (
		id INTEGER PRIMARY KEY,
		created_date DATETIME,
		updated_date DATETIME,
		status TEXT,
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER,
		key TEXT,
		value TEXT,
		assistant_deployment_telephony_id INTEGER
	)`).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistants (id, project_id, organization_id, visibility, status) VALUES (?, ?, ?, ?, ?)",
		42,
		7,
		8,
		"private",
		type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec("INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status, created_date) VALUES (100, 42, 'sip', ?, '2026-09-04 10:00:00'), (101, 42, 'sip', ?, '2026-09-05 10:00:00'), (102, 43, 'sip', ?, '2026-09-05 10:00:00')", type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_ACTIVE.String(), type_enums.RECORD_ACTIVE.String()).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistants (id, project_id, organization_id, visibility, status) VALUES (?, ?, ?, ?, ?)",
		43,
		7,
		8,
		"private",
		type_enums.RECORD_INACTIVE.String(),
	).Error)

	applicationLogger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
		commons.Level("error"),
	)
	require.NoError(t, err)
	service := &assistantService{
		logger:   applicationLogger,
		postgres: &auditTestPostgresConnector{db: database},
	}

	assistant, err := service.GetAssistantWithPhoneDeploymentById(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), assistant.Id)
	assert.Equal(t, uint64(7), assistant.ProjectId)
	assert.Equal(t, uint64(8), assistant.OrganizationId)
	require.NotNil(t, assistant.AssistantPhoneDeployment)
	assert.Equal(t, uint64(101), assistant.AssistantPhoneDeployment.Id)

	_, err = service.GetAssistantWithPhoneDeploymentById(context.Background(), 43)
	require.Error(t, err)
}

func TestAssistantServiceGetAssistantWithPhoneDeploymentByDID(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/assistant-did-service.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE assistants (
		id INTEGER PRIMARY KEY,
		project_id INTEGER,
		organization_id INTEGER,
		visibility TEXT,
		status TEXT
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_phone_deployments (
		id INTEGER PRIMARY KEY,
		assistant_id INTEGER,
		telephony_provider TEXT,
		status TEXT,
		created_date DATETIME
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_deployment_telephony_options (
		id INTEGER PRIMARY KEY,
		assistant_deployment_telephony_id INTEGER,
		key TEXT,
		value TEXT,
		created_date DATETIME,
		updated_date DATETIME,
		status TEXT,
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_deployment_audios (
		id INTEGER PRIMARY KEY,
		assistant_deployment_id INTEGER,
		audio_type TEXT
	)`).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistants (id, project_id, organization_id, visibility, status) VALUES (?, ?, ?, ?, ?)",
		42, 7, 8, "private", type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status, created_date) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		100, 42, "sip", type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (id, assistant_deployment_telephony_id, key, value, created_date) VALUES (?, ?, ?, ?, ?)",
		1000, 100, "phone", "+15551234567", "2026-09-04 10:00:00",
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (id, assistant_deployment_telephony_id, key, value, created_date) VALUES (?, ?, ?, ?, ?)",
		1002, 100, "rapida.credential_id", "76", "2026-09-04 10:00:00",
	).Error)

	applicationLogger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
		commons.Level("error"),
	)
	require.NoError(t, err)
	service := &assistantService{
		logger:   applicationLogger,
		postgres: &auditTestPostgresConnector{db: database},
	}

	assistant, err := service.GetAssistantWithPhoneDeploymentByDID(context.Background(), "+15551234567")
	require.NoError(t, err)
	assert.Equal(t, uint64(42), assistant.Id)
	require.NotNil(t, assistant.AssistantPhoneDeployment)
	assert.Equal(t, uint64(100), assistant.AssistantPhoneDeployment.Id)
	phone, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
	require.NoError(t, err)
	assert.Equal(t, "+15551234567", phone)

	require.NoError(t, database.Exec(
		"INSERT INTO assistants (id, project_id, organization_id, visibility, status) VALUES (?, ?, ?, ?, ?)",
		43, 9, 10, "private", type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_phone_deployments (id, assistant_id, telephony_provider, status, created_date) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		101, 43, "sip", type_enums.RECORD_ACTIVE.String(),
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (id, assistant_deployment_telephony_id, key, value, created_date) VALUES (?, ?, ?, ?, ?)",
		1001, 101, "phone", "+15551234567", "2026-09-05 10:00:00",
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (id, assistant_deployment_telephony_id, key, value, created_date) VALUES (?, ?, ?, ?, ?)",
		1003, 101, "rapida.credential_id", "77", "2026-09-05 10:00:00",
	).Error)

	assistant, err = service.GetAssistantWithPhoneDeploymentByDID(context.Background(), "+15551234567")
	require.NoError(t, err)
	assert.Equal(t, uint64(43), assistant.Id)
	require.NotNil(t, assistant.AssistantPhoneDeployment)
	assert.Equal(t, uint64(101), assistant.AssistantPhoneDeployment.Id)
	credentialID, err := assistant.AssistantPhoneDeployment.GetOptions().GetUint64("rapida.credential_id")
	require.NoError(t, err)
	assert.Equal(t, uint64(77), credentialID)
}
