package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// StringMap is a custom type to represent a map of strings
type StringMap map[string]string

// Scan converts JSON data into StringMap
func (a *StringMap) Scan(value interface{}) error {
	if value == nil {
		*a = nil
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

// Value converts StringMap into a format suitable for the database
func (a StringMap) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// String converts StringMap into a string representation
func (a StringMap) String() string {
	str := make([]string, 0, len(a))
	for k, v := range a {
		str = append(str, fmt.Sprintf("%s:%s", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(str, ","))
}
