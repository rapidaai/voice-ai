package gorm_models

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type callbackAuditedRecord struct {
	Audited
	Mutable
	Name string
}

func TestAuditActorCallbacksCreateAndUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := createCallbackAuditTable(db); err != nil {
		t.Fatal(err)
	}
	if err := RegisterAuditActorCallbacks(db); err != nil {
		t.Fatal(err)
	}

	serviceAuth := &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: 41},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, serviceAuth)
	record := &callbackAuditedRecord{Mutable: Mutable{Status: type_enums.RECORD_ACTIVE}, Name: "first"}
	if err := db.WithContext(ctx).Create(record).Error; err != nil {
		t.Fatal(err)
	}
	if record.CreatedActorType == nil || *record.CreatedActorType != "service" || record.CreatedActorID == nil || *record.CreatedActorID != 41 {
		t.Fatalf("created audit actor = %+v", record.Mutable)
	}

	userAuth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 9},
		UserValue:         &types.UserContext{UserID: 9},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 7},
	}
	ctx = context.WithValue(context.Background(), types.CTX_, userAuth)
	if err := db.WithContext(ctx).Model(record).Update("name", "second").Error; err != nil {
		t.Fatal(err)
	}
	if record.UpdatedActorType == nil || *record.UpdatedActorType != "user" || record.UpdatedActorID == nil || *record.UpdatedActorID != 9 {
		t.Fatalf("updated audit actor = %+v", record.Mutable)
	}
}

func TestAuditActorCallbacksRejectMissingActor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := createCallbackAuditTable(db); err != nil {
		t.Fatal(err)
	}
	if err := RegisterAuditActorCallbacks(db); err != nil {
		t.Fatal(err)
	}
	result := db.Create(&callbackAuditedRecord{Mutable: Mutable{Status: type_enums.RECORD_ACTIVE}, Name: "missing"})
	if !errors.Is(result.Error, types.ErrActorUnavailable) {
		t.Fatalf("Create() error = %v, want %v", result.Error, types.ErrActorUnavailable)
	}
}

func createCallbackAuditTable(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE callback_audited_records (
		id integer primary key,
		created_date datetime,
		updated_date datetime,
		status text,
		created_actor_type text,
		created_actor_id integer,
		updated_actor_type text,
		updated_actor_id integer,
		name text
	)`).Error
}
