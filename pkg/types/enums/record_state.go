package type_enums

import (
	"database/sql/driver"
	"encoding/json"
)

/**
* A state representation of table row
* it can have state depends on logic but the record will only have these status
 */
type RecordState string

// keep adding but not good idea the status of record and status of transaction should be very different
// status of record is immutable and common for all table but state of transaction should vary depends on what table is
// you know what will refactoir this in future
const (
	RECORD_ACTIVE RecordState = "ACTIVE"

	RECORD_QUEUED    RecordState = "QUEUED"
	RECORD_CONNECTED RecordState = "CONNECTED"

	RECORD_IN_PROGRESS RecordState = "IN_PROGRESS"
	RECORD_SUCCESS     RecordState = "SUCCESS"
	RECORD_COMPLETE    RecordState = "COMPLETE"
	RECORD_INACTIVE    RecordState = "INACTIVE"
	RECORD_ARCHIEVE    RecordState = "ARCHIEVE"
	RECORD_FAILED      RecordState = "FAILED"
)

func (m RecordState) String() string {
	return string(m)
}

func (c RecordState) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c RecordState) Value() (driver.Value, error) {
	return string(c), nil
}

func ToRecordState(s string) RecordState {
	switch s {
	case "ACTIVE":
		return RECORD_ACTIVE
	default:
		return RECORD_INACTIVE // or any other default status you prefer
	}
}
