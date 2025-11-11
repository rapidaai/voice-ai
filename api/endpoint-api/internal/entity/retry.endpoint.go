package internal_entity

import (
	"database/sql/driver"
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type Retry string

const (
	NEVER_RETRY  Retry = "no-retry"
	STATUS_RETRY Retry = "retry-on-status"
)

func (r Retry) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

func (r Retry) Value() (driver.Value, error) {
	return string(r), nil
}

type EndpointRetry struct {
	gorm_model.Audited
	gorm_model.Mutable
	EndpointId         uint64 `json:"endpointId" gorm:"type:bigint;not null"`
	RetryType          Retry  `json:"retryType" gorm:"type:bigint;size:20;not null"`
	MaxAttempts        uint64 `json:"maxAttempts" gorm:"not null"`
	DelaySeconds       uint64 `json:"delaySeconds" gorm:"not null"`
	ExponentialBackoff bool   `json:"exponentialBackoff" gorm:"not null"`
	// this is depends on retry type
	// in case retry is status then it will be [4XX, 5XX]
	Retryables gorm_types.StringArray `json:"retryables" gorm:"type:text;size:1000;null"`
}
