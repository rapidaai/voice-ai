package type_enums

import (
	"database/sql/driver"
	"encoding/json"
)

/**
* A state representation of table row
* it can have state depends on logic but the record will only have these status
 */
type RecordVisibility string

// keep adding but not good idea the status of record and status of transaction should be very different
// status of record is immutable and common for all table but state of transaction should vary depends on what table is
// you know what will refactoir this in future
const (
	RECORD_PUBLIC  RecordVisibility = "PUBLIC"
	RECORD_PRIVATE RecordVisibility = "PRIVATE"
)

func (m RecordVisibility) String() string {
	return string(m)
}

func (c RecordVisibility) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c RecordVisibility) Value() (driver.Value, error) {
	return string(c), nil
}

func ToRecordVisibility(s string) RecordVisibility {
	switch s {
	case "public":
		return RECORD_PUBLIC
	default:
		return RECORD_PRIVATE // or any other default status you prefer
	}
}
