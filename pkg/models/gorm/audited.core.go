// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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
	Status    type_enums.RecordState `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
	CreatedBy uint64                 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy uint64                 `json:"updatedBy" gorm:"type:bigint;size:20;"`
}

type Organizational struct {
	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;size:20;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:20;not null"`
}
