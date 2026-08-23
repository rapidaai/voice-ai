// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type TimeWrapper time.Time

type Audited struct {
	Id          uint64      `json:"id" gorm:"type:bigint;primaryKey;<-:create"`
	CreatedDate TimeWrapper `json:"createdDate" gorm:"type:timestamp;not null;default:NOW();<-:create"`
	UpdatedDate TimeWrapper `json:"updatedDate" gorm:"type:timestamp;default:null;onUpdate:NOW()"`
}

func (m *Audited) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedDate = TimeWrapper(time.Now())
	return nil
}

func (m *Audited) BeforeCreate(tx *gorm.DB) (err error) {
	if time.Time(m.CreatedDate).IsZero() {
		m.CreatedDate = TimeWrapper(time.Now())
	}
	if m.Id <= 0 {
		m.Id = gorm_generator.ID()
	}
	return nil
}

func (t TimeWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(timestamppb.New(time.Time(t)))
}

func (t *TimeWrapper) UnmarshalJSON(data []byte) error {
	ts := &timestamppb.Timestamp{}
	if err := json.Unmarshal(data, ts); err != nil {
		return err
	}
	*t = TimeWrapper(ts.AsTime())
	return nil
}

func (t TimeWrapper) Value() (driver.Value, error) {
	return time.Time(t), nil
}

type Mutable struct {
	Status type_enums.RecordState `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
	ActorAudit
}

type ActorAudit struct {
	CreatedActorType *string              `json:"-" gorm:"column:created_actor_type;type:varchar(32);<-:create"`
	CreatedActorID   *uint64              `json:"-" gorm:"column:created_actor_id;type:bigint;<-:create"`
	UpdatedActorType *string              `json:"-" gorm:"column:updated_actor_type;type:varchar(32)"`
	UpdatedActorID   *uint64              `json:"-" gorm:"column:updated_actor_id;type:bigint"`
	CreatedActor     *types.ActorIdentity `json:"createdActor,omitempty" gorm:"-"`
	UpdatedActor     *types.ActorIdentity `json:"updatedActor,omitempty" gorm:"-"`
}

func NewMutable(status type_enums.RecordState, actor types.ActorIdentity) (Mutable, error) {
	mutable := Mutable{Status: status}
	if err := mutable.SetCreatedActor(actor); err != nil {
		return Mutable{}, err
	}
	if err := mutable.SetUpdatedActor(actor); err != nil {
		return Mutable{}, err
	}
	return mutable, nil
}

func ActorUpdateColumns(actor types.ActorIdentity) (map[string]interface{}, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"updated_actor_type": string(actor.Type),
		"updated_actor_id":   actor.ID,
		"updated_date":       time.Now(),
	}, nil
}

func MergeActorUpdateColumns(values map[string]interface{}, actor types.ActorIdentity) (map[string]interface{}, error) {
	actorValues, err := ActorUpdateColumns(actor)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = make(map[string]interface{}, len(actorValues))
	}
	for key, value := range actorValues {
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("actor update column %s is owned by the audit model", key)
		}
		values[key] = value
	}
	return values, nil
}

func (mutable *Mutable) SetCreatedActor(actor types.ActorIdentity) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	actorType := string(actor.Type)
	actorID := actor.ID
	mutable.CreatedActorType = &actorType
	mutable.CreatedActorID = &actorID
	mutable.CreatedActor = &types.ActorIdentity{Type: actor.Type, ID: actor.ID}
	return nil
}

func (mutable *Mutable) SetUpdatedActor(actor types.ActorIdentity) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	actorType := string(actor.Type)
	actorID := actor.ID
	mutable.UpdatedActorType = &actorType
	mutable.UpdatedActorID = &actorID
	mutable.UpdatedActor = &types.ActorIdentity{Type: actor.Type, ID: actor.ID}
	return nil
}

func (mutable *Mutable) AfterFind(tx *gorm.DB) error {
	mutable.ActorAudit.hydrate(tx)
	return nil
}

func (audit *ActorAudit) AfterFind(tx *gorm.DB) error {
	audit.hydrate(tx)
	return nil
}

func (audit *ActorAudit) hydrate(tx *gorm.DB) {
	var createdInvalid, updatedInvalid bool
	audit.CreatedActor, createdInvalid = actorIdentityFromColumns(audit.CreatedActorType, audit.CreatedActorID)
	audit.UpdatedActor, updatedInvalid = actorIdentityFromColumns(audit.UpdatedActorType, audit.UpdatedActorID)
	if createdInvalid {
		recordAuditProjectionFailure(tx, "created")
	}
	if updatedInvalid {
		recordAuditProjectionFailure(tx, "updated")
	}
}

func actorIdentityFromColumns(actorType *string, actorID *uint64) (*types.ActorIdentity, bool) {
	if actorType == nil && actorID == nil {
		return nil, false
	}
	if actorType != nil && *actorType == string(types.ActorTypeUnknown) && actorID == nil {
		return nil, false
	}
	if actorType == nil || actorID == nil {
		return nil, true
	}
	actor := &types.ActorIdentity{Type: types.ActorType(*actorType), ID: *actorID}
	if actor.Validate() != nil {
		return nil, true
	}
	return actor, false
}

type Organizational struct {
	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;size:20;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:20;not null"`
}
