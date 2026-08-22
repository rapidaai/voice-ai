// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/types"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return db
}

func TestTimeWrapper_MarshalJSON(t *testing.T) {
	now := time.Now()
	tw := TimeWrapper(now)

	data, err := tw.MarshalJSON()
	assert.NoError(t, err)

	var ts timestamppb.Timestamp
	err = json.Unmarshal(data, &ts)
	assert.NoError(t, err)

	assert.True(t, ts.AsTime().Sub(now) < time.Second) // Close enough
}

func TestTimeWrapper_UnmarshalJSON(t *testing.T) {
	now := time.Now()
	ts := timestamppb.New(now)
	data, err := json.Marshal(ts)
	assert.NoError(t, err)

	var tw TimeWrapper
	err = tw.UnmarshalJSON(data)
	assert.NoError(t, err)

	assert.True(t, time.Time(tw).Sub(now) < time.Second)
}

func TestTimeWrapper_Value(t *testing.T) {
	now := time.Now()
	tw := TimeWrapper(now)

	value, err := tw.Value()
	assert.NoError(t, err)
	assert.Equal(t, now, value)
}

func TestAudited_BeforeCreate(t *testing.T) {
	// Test with zero CreatedDate and zero Id
	audited := &Audited{}
	err := audited.BeforeCreate(nil) // DB not used in method
	assert.NoError(t, err)
	assert.False(t, time.Time(audited.CreatedDate).IsZero())
	assert.True(t, audited.Id > 0)

	// Test with existing CreatedDate
	existingTime := time.Now().Add(-time.Hour)
	audited2 := &Audited{
		CreatedDate: TimeWrapper(existingTime),
		Id:          123,
	}
	err = audited2.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.Equal(t, TimeWrapper(existingTime), audited2.CreatedDate)
	assert.Equal(t, uint64(123), audited2.Id)
}

func TestAudited_BeforeUpdate(t *testing.T) {
	audited := &Audited{}
	err := audited.BeforeUpdate(nil)
	assert.NoError(t, err)
	assert.False(t, time.Time(audited.UpdatedDate).IsZero())
}

func TestMutable_JSONMarshaling(t *testing.T) {
	createdActorType := "user"
	createdActorID := uint64(123)
	updatedActorType := "project"
	updatedActorID := uint64(789)
	createdActor := &types.ActorIdentity{Type: types.ActorTypeUser, ID: 123}
	updatedActor := &types.ActorIdentity{Type: types.ActorTypeProject, ID: 789}
	mutable := Mutable{
		Status: "ACTIVE",
		ActorAudit: ActorAudit{
			CreatedActorType: &createdActorType,
			CreatedActorID:   &createdActorID,
			UpdatedActorType: &updatedActorType,
			UpdatedActorID:   &updatedActorID,
			CreatedActor:     createdActor,
			UpdatedActor:     updatedActor,
		},
	}

	data, err := json.Marshal(mutable)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"status":"ACTIVE"`)
	assert.NotContains(t, string(data), `"createdActorType"`)
	assert.NotContains(t, string(data), `"createdActorId"`)
	assert.NotContains(t, string(data), `"createdBy"`)
	assert.NotContains(t, string(data), `"updatedBy"`)
	assert.Contains(t, string(data), `"createdActor":{"type":"user","id":123}`)
	assert.Contains(t, string(data), `"updatedActor":{"type":"project","id":789}`)
}

func TestMutableAfterFindHydratesActorProjection(t *testing.T) {
	createdType := "service"
	createdID := uint64(41)
	updatedType := "system"
	updatedID := uint64(99)
	mutable := Mutable{
		ActorAudit: ActorAudit{
			CreatedActorType: &createdType,
			CreatedActorID:   &createdID,
			UpdatedActorType: &updatedType,
			UpdatedActorID:   &updatedID,
		},
	}
	if err := mutable.AfterFind(nil); err != nil {
		t.Fatal(err)
	}
	if mutable.CreatedActor == nil || *mutable.CreatedActor != (types.ActorIdentity{Type: types.ActorTypeService, ID: 41}) {
		t.Fatalf("created actor = %+v", mutable.CreatedActor)
	}
	if mutable.UpdatedActor == nil || *mutable.UpdatedActor != (types.ActorIdentity{Type: types.ActorTypeSystem, ID: 99}) {
		t.Fatalf("updated actor = %+v", mutable.UpdatedActor)
	}
}

func TestMutableActorAssignment(t *testing.T) {
	mutable := Mutable{}
	if err := mutable.SetCreatedActor(types.ActorIdentity{Type: types.ActorTypeProject, ID: 123}); err != nil {
		t.Fatal(err)
	}
	if mutable.CreatedActorType == nil || *mutable.CreatedActorType != "project" || mutable.CreatedActorID == nil || *mutable.CreatedActorID != 123 {
		t.Fatalf("created actor fields = %+v", mutable)
	}
	if err := mutable.SetUpdatedActor(types.ActorIdentity{Type: types.ActorTypeUser, ID: 456}); err != nil {
		t.Fatal(err)
	}
	if mutable.UpdatedActorType == nil || *mutable.UpdatedActorType != "user" || mutable.UpdatedActorID == nil || *mutable.UpdatedActorID != 456 {
		t.Fatalf("updated actor fields = %+v", mutable)
	}
	if err := mutable.SetCreatedActor(types.ActorIdentity{Type: types.ActorTypeUnknown}); err == nil {
		t.Fatal("SetCreatedActor() error = nil for unknown actor")
	}
}

func TestNewMutableAndActorUpdateColumns(t *testing.T) {
	actor := types.ActorIdentity{Type: types.ActorTypeService, ID: 42}
	mutable, err := NewMutable("ACTIVE", actor)
	if err != nil {
		t.Fatal(err)
	}
	if mutable.CreatedActorID == nil || *mutable.CreatedActorID != 42 || mutable.UpdatedActorID == nil || *mutable.UpdatedActorID != 42 {
		t.Fatalf("NewMutable() = %+v", mutable)
	}
	columns, err := MergeActorUpdateColumns(map[string]interface{}{"status": "ARCHIEVE"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := columns["updated_by"]; exists {
		t.Fatalf("MergeActorUpdateColumns() retained legacy column: %+v", columns)
	}
	if columns["updated_actor_type"] != "service" || columns["updated_actor_id"] != uint64(42) {
		t.Fatalf("MergeActorUpdateColumns() = %+v", columns)
	}
	if _, err := MergeActorUpdateColumns(map[string]interface{}{"updated_actor_id": uint64(1)}, actor); err == nil {
		t.Fatal("MergeActorUpdateColumns() error = nil for owned column")
	}
}

func TestOrganizational_JSONMarshaling(t *testing.T) {
	org := Organizational{
		ProjectId:      789,
		OrganizationId: 101112,
	}

	data, err := json.Marshal(org)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"projectId":789`)
	assert.Contains(t, string(data), `"organizationId":101112`)
}

func TestTimeWrapper_JSONRoundTrip(t *testing.T) {
	original := TimeWrapper(time.Now())

	// Marshal
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Unmarshal
	var unmarshaled TimeWrapper
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	// Should be approximately equal
	assert.True(t, time.Time(unmarshaled).Sub(time.Time(original)) < time.Second)
}

func TestAudited_JSONMarshaling(t *testing.T) {
	now := time.Now()
	audited := Audited{
		Id:          12345,
		CreatedDate: TimeWrapper(now),
		UpdatedDate: TimeWrapper(now.Add(time.Hour)),
	}

	data, err := json.Marshal(audited)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)

	assert.Equal(t, float64(12345), result["id"])
	assert.Contains(t, result, "createdDate")
	assert.Contains(t, result, "updatedDate")
}
