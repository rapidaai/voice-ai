// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAssistantDeploymentTelephonyOptionPreloadsDeploymentAndAssistant(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(t.TempDir()+"/telephony-option.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE assistants (
		id INTEGER PRIMARY KEY
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE assistant_phone_deployments (
		id INTEGER PRIMARY KEY,
		assistant_id INTEGER
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
	require.NoError(t, database.Exec("INSERT INTO assistants (id) VALUES (?)", 42).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_phone_deployments (id, assistant_id) VALUES (?, ?)",
		100,
		42,
	).Error)
	require.NoError(t, database.Exec(
		"INSERT INTO assistant_deployment_telephony_options (id, assistant_deployment_telephony_id, key, value) VALUES (?, ?, ?, ?)",
		1000,
		100,
		"phone",
		"+15551234567",
	).Error)

	var option AssistantDeploymentTelephonyOption
	err = database.
		Preload("AssistantPhoneDeployment.Assistant").
		First(&option, 1000).Error
	require.NoError(t, err)
	require.NotNil(t, option.AssistantPhoneDeployment)
	require.NotNil(t, option.AssistantPhoneDeployment.Assistant)
	assert.Equal(t, uint64(100), option.AssistantPhoneDeployment.Id)
	assert.Equal(t, uint64(42), option.AssistantPhoneDeployment.Assistant.Id)
}
