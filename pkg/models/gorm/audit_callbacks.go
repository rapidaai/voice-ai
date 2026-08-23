package gorm_models

import (
	"errors"

	"gorm.io/gorm"

	"github.com/rapidaai/pkg/types"
)

const (
	createAuditActorCallback  = "rapida:set_create_audit_actor"
	updateAuditActorCallback  = "rapida:set_update_audit_actor"
	recordCreateAuditCallback = "rapida:record_create_audit_actor"
	recordUpdateAuditCallback = "rapida:record_update_audit_actor"
	auditActorInstanceKey     = "rapida:audit_actor"
)

func RegisterAuditActorCallbacks(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is required for audit actor callbacks")
	}
	if err := db.Callback().Create().Before("gorm:create").Register(createAuditActorCallback, setCreateAuditActor); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register(recordCreateAuditCallback, recordCreateAuditActor); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register(updateAuditActorCallback, setUpdateAuditActor); err != nil {
		return err
	}
	return db.Callback().Update().After("gorm:update").Register(recordUpdateAuditCallback, recordUpdateAuditActor)
}

func setCreateAuditActor(tx *gorm.DB) {
	if !hasAuditActorFields(tx) {
		return
	}
	actor, err := requestAuditActor(tx)
	if err != nil {
		recordMissingAuditActor(tx, "create")
		tx.AddError(err)
		return
	}
	tx.InstanceSet(auditActorInstanceKey, actor)
	actorType := string(actor.Type)
	if err := setAuditActorFields(tx, map[string]interface{}{
		"CreatedActorType": &actorType,
		"CreatedActorID":   &actor.ID,
		"UpdatedActorType": &actorType,
		"UpdatedActorID":   &actor.ID,
	}); err != nil {
		tx.AddError(err)
	}
}

func setUpdateAuditActor(tx *gorm.DB) {
	if !hasAuditActorFields(tx) {
		return
	}
	actor, err := requestAuditActor(tx)
	if err != nil {
		recordMissingAuditActor(tx, "update")
		tx.AddError(err)
		return
	}
	tx.InstanceSet(auditActorInstanceKey, actor)
	actorType := string(actor.Type)
	if err := setAuditActorFields(tx, map[string]interface{}{
		"UpdatedActorType": &actorType,
		"UpdatedActorID":   &actor.ID,
	}); err != nil {
		tx.AddError(err)
	}
}

func recordCreateAuditActor(tx *gorm.DB) {
	recordSuccessfulAuditWrite(tx, "create")
}

func recordUpdateAuditActor(tx *gorm.DB) {
	recordSuccessfulAuditWrite(tx, "update")
}

func setAuditActorFields(tx *gorm.DB, values map[string]interface{}) error {
	for name, value := range values {
		tx.Statement.SetColumn(name, value, true)
		if tx.Error != nil {
			return tx.Error
		}
	}
	return nil
}

func hasAuditActorFields(tx *gorm.DB) bool {
	if tx == nil || tx.Statement == nil || tx.Statement.Schema == nil {
		return false
	}
	return tx.Statement.Schema.LookUpField("CreatedActorType") != nil &&
		tx.Statement.Schema.LookUpField("CreatedActorID") != nil &&
		tx.Statement.Schema.LookUpField("UpdatedActorType") != nil &&
		tx.Statement.Schema.LookUpField("UpdatedActorID") != nil
}

func requestAuditActor(tx *gorm.DB) (types.ActorIdentity, error) {
	if tx.Statement.Context == nil {
		return types.ActorIdentity{}, types.ErrActorUnavailable
	}
	auth, err := types.Authorize(tx.Statement.Context)
	if err != nil {
		return types.ActorIdentity{}, types.ErrActorUnavailable
	}
	return auth.Actor()
}
