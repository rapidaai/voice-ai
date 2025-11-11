package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// IntArray is a custom type to represent an array of integers
type StringArray []string

// Scan converts JSON data into IntArray
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// Value converts IntArray into a format suitable for the database
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// String converts IntArray into a string representation
func (a StringArray) String() string {
	str := make([]string, len(a))
	return fmt.Sprintf("{%s}", strings.Join(str, ","))
}
