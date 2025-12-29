// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// MapArray is a custom type to represent an array of maps
type MapArray []map[string]string

// Scan converts JSON data into MapArray
func (a *MapArray) Scan(value interface{}) error {
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

// Value converts MapArray into a format suitable for the database
func (a MapArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// String converts MapArray into a string representation
func (a MapArray) String() string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(data)
}

// MapArray is a custom type to represent an array of maps
type MapInterfaceArray []map[string]interface{}

// Scan converts JSON data into MapArray
func (a *MapInterfaceArray) Scan(value interface{}) error {
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

// Value converts MapInterfaceArray into a format suitable for the database
func (a MapInterfaceArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// String converts MapInterfaceArray into a string representation
func (a MapInterfaceArray) String() string {
	data, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(data)
}
