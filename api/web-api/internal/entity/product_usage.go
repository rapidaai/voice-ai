package internal_entity

import (
	"time"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type ProductUsage struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational

	UsageType  string    `json:"usageType" gorm:"column:usage_type;type:varchar(100);not null"`
	Usages     int64     `json:"usages" gorm:"type:bigint;not null"`
	Unit       string    `json:"unit" gorm:"type:varchar(32);not null"`
	OccurredAt time.Time `json:"occurredAt" gorm:"column:occurred_at;type:timestamp(6);not null"`
}

func (ProductUsage) TableName() string {
	return "product_usages"
}
