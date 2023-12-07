package gorm_models

import (
	"time"

	gorm_generator "github.com/lexatic/web-backend/pkg/models/gorm/generators"
	"gorm.io/gorm"
)

type Audited struct {
	Id          uint64    `json:"id" gorm:"type:bigint;primaryKey;<-:create"`
	CreatedDate time.Time `json:"created_date" gorm:"type:timestamp;not null;default:NOW();<-:create"`
	UpdatedDate time.Time `json:"updated_date" gorm:"type:timestamp;default:null;onUpdate:NOW()"`
}

func (m *Audited) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedDate = time.Now()
	return nil
}

func (m *Audited) BeforeCreate(tx *gorm.DB) (err error) {
	if m.CreatedDate.IsZero() {
		m.CreatedDate = time.Now()
	}
	if m.Id <= 0 {
		m.Id = gorm_generator.ID()
	}
	return nil
}
