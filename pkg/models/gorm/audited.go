package gorm_models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	gorm_generator "github.com/lexatic/web-backend/pkg/models/gorm/generators"
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

func (t TimeWrapper) Value() (driver.Value, error) {
	return time.Time(t), nil
}
