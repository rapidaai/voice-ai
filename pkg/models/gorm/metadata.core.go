package gorm_models

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Metadata struct {
	Key   string `json:"key" gorm:"type:string;size:200;not null"`
	Value string `json:"value" gorm:"type:string;size:1000;not null"`
}

func (d *Metadata) SetValue(src interface{}) error {
	switch v := src.(type) {
	case string:
		d.Value = v
	case []byte:
		d.Value = string(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		d.Value = fmt.Sprintf("%d", v)
	case float32, float64:
		d.Value = fmt.Sprintf("%f", v)
	case bool:
		d.Value = strconv.FormatBool(v)
	case nil:
		d.Value = ""
	default:
		// Use JSON marshalling for all other types, including maps and complex structures
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		d.Value = string(jsonBytes)
	}
	return nil
}
