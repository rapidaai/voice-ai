package internal_entity

import (
	"database/sql/driver"
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type Cache string

const (
	NEVER_CACHE    Cache = "never"
	STANDARD_CACHE Cache = "standard"
	SEMENTIC_CACHE Cache = "sementic"
)

func (c Cache) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c Cache) Value() (driver.Value, error) {
	return string(c), nil
}

type EndpointCaching struct {
	gorm_model.Audited
	gorm_model.Mutable
	EndpointId     uint64  `json:"endpointId" gorm:"type:bigint;not null"`
	CacheType      Cache   `json:"cacheType" gorm:"type:bigint;size:20;not null"`
	ExpiryInterval uint64  `json:"expiryInterval" gorm:"type:bigint;size:20"`
	MatchThreshold float32 `json:"matchThreshold" gorm:"type:float;size:20"`
	CreatedBy      uint64  `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy      uint64  `json:"updatedBy" gorm:"type:bigint;size:20;"`
}
