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

// IntArray is a custom type to represent an array of integers
type IntArray []uint64

// Scan converts JSON data into IntArray
func (a *IntArray) Scan(value interface{}) error {
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
func (a IntArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// String converts IntArray into a string representation
func (a IntArray) String() string {
	str := make([]string, len(a))
	for i, v := range a {
		str[i] = fmt.Sprintf("%d", v)
	}
	return fmt.Sprintf("{%s}", strings.Join(str, ","))
}
