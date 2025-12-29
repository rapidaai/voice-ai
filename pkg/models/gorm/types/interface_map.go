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
	"strings"
)

// InterfaceMap is a custom type to represent a map of strings
type InterfaceMap map[string]interface{}

// Scan converts JSON data into InterfaceMap
func (a *InterfaceMap) Scan(value interface{}) error {
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

// Value converts InterfaceMap into a format suitable for the database
func (a InterfaceMap) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil // Return an empty JSON object instead of NULL
	}
	return json.Marshal(a)
}

// String converts InterfaceMap into a string representation
func (a InterfaceMap) String() string {
	str := make([]string, 0, len(a))
	for k, v := range a {
		str = append(str, fmt.Sprintf("%s:%s", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(str, ","))
}
