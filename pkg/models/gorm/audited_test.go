// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type auditedTestRecord struct {
	Audited
	Mutable
}

func TestTimeWrapper_MarshalJSON(t *testing.T) {
	now := time.Now()
	data, err := TimeWrapper(now).MarshalJSON()
	assert.NoError(t, err)

	var timestamp timestamppb.Timestamp
	assert.NoError(t, json.Unmarshal(data, &timestamp))
	assert.True(t, timestamp.AsTime().Sub(now) < time.Second)
}

func TestTimeWrapper_UnmarshalJSON(t *testing.T) {
	now := time.Now()
	data, err := json.Marshal(timestamppb.New(now))
	assert.NoError(t, err)

	var wrapper TimeWrapper
	assert.NoError(t, wrapper.UnmarshalJSON(data))
	assert.True(t, time.Time(wrapper).Sub(now) < time.Second)
}

func TestTimeWrapper_Value(t *testing.T) {
	now := time.Now()
	value, err := TimeWrapper(now).Value()
	assert.NoError(t, err)
	assert.Equal(t, now, value)
}

func TestAudited_BeforeCreate(t *testing.T) {
	audited := &Audited{}
	assert.NoError(t, audited.BeforeCreate(nil))
	assert.False(t, time.Time(audited.CreatedDate).IsZero())
	assert.Positive(t, audited.Id)

	existingTime := time.Now().Add(-time.Hour)
	audited = &Audited{CreatedDate: TimeWrapper(existingTime), Id: 123}
	assert.NoError(t, audited.BeforeCreate(nil))
	assert.Equal(t, TimeWrapper(existingTime), audited.CreatedDate)
	assert.Equal(t, uint64(123), audited.Id)
}

func TestAudited_BeforeUpdate(t *testing.T) {
	audited := &Audited{}
	assert.NoError(t, audited.BeforeUpdate(nil))
	assert.False(t, time.Time(audited.UpdatedDate).IsZero())
}

func TestMutableJSONMarshalingHidesAuditColumns(t *testing.T) {
	mutable := Mutable{
		Status:           "ACTIVE",
		CreatedActorType: "service",
		CreatedActorID:   41,
		UpdatedActorType: "user",
		UpdatedActorID:   42,
	}

	data, err := json.Marshal(mutable)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"status":"ACTIVE"}`, string(data))
}

func TestActorAuditStoresExplicitColumns(t *testing.T) {
	audit := ActorAudit{
		CreatedActorType: "service",
		CreatedActorID:   41,
		UpdatedActorType: "user",
		UpdatedActorID:   42,
	}

	assert.Equal(t, "service", audit.CreatedActorType)
	assert.Equal(t, uint64(41), audit.CreatedActorID)
	assert.Equal(t, "user", audit.UpdatedActorType)
	assert.Equal(t, uint64(42), audit.UpdatedActorID)
}

func TestActorAuditColumnsAreNotSerialized(t *testing.T) {
	encoded, err := json.Marshal(ActorAudit{
		CreatedActorType: "service",
		CreatedActorID:   41,
		UpdatedActorType: "service",
		UpdatedActorID:   41,
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{}`, string(encoded))
}

func TestMutableCreateLeavesUpdatedActorColumnsNull(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, database.Exec(`
		CREATE TABLE audited_test_records (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer
		)
	`).Error)

	record := &auditedTestRecord{Mutable: Mutable{
		Status:           "ACTIVE",
		CreatedActorType: "service",
		CreatedActorID:   41,
	}}
	assert.NoError(t, database.Create(record).Error)

	var createdActorType sql.NullString
	var createdActorID sql.NullInt64
	var updatedActorType sql.NullString
	var updatedActorID sql.NullInt64
	row := database.Raw(
		"SELECT created_actor_type, created_actor_id, updated_actor_type, updated_actor_id FROM audited_test_records WHERE id = ?",
		record.Id,
	).Row()
	assert.NoError(t, row.Scan(&createdActorType, &createdActorID, &updatedActorType, &updatedActorID))
	assert.Equal(t, sql.NullString{String: "service", Valid: true}, createdActorType)
	assert.Equal(t, sql.NullInt64{Int64: 41, Valid: true}, createdActorID)
	assert.False(t, updatedActorType.Valid)
	assert.False(t, updatedActorID.Valid)
}

func TestOrganizational_JSONMarshaling(t *testing.T) {
	data, err := json.Marshal(Organizational{ProjectId: 789, OrganizationId: 101112})
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"projectId":789`)
	assert.Contains(t, string(data), `"organizationId":101112`)
}

func TestTimeWrapper_JSONRoundTrip(t *testing.T) {
	original := TimeWrapper(time.Now())
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	var unmarshaled TimeWrapper
	assert.NoError(t, json.Unmarshal(data, &unmarshaled))
	assert.True(t, time.Time(unmarshaled).Sub(time.Time(original)) < time.Second)
}

func TestAudited_JSONMarshaling(t *testing.T) {
	now := time.Now()
	data, err := json.Marshal(Audited{
		Id:          12345,
		CreatedDate: TimeWrapper(now),
		UpdatedDate: TimeWrapper(now.Add(time.Hour)),
	})
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, float64(12345), result["id"])
	assert.Contains(t, result, "createdDate")
	assert.Contains(t, result, "updatedDate")
}
