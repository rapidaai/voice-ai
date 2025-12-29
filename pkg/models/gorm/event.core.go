// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package gorm_models

import (
	"encoding/json"
	"fmt"
	"strconv"

	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type Event struct {
	EventType string                  `json:"eventType" gorm:"type:string;size:200;not null"`
	Payload   gorm_types.InterfaceMap `json:"payload" gorm:"type:string;size:1000;not null"`
}

func (d *Event) SetValue(src interface{}) error {
	switch v := src.(type) {
	case string:
		d.Payload = gorm_types.InterfaceMap{"value": v}
	case []byte:
		d.Payload = gorm_types.InterfaceMap{"value": string(v)}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		d.Payload = gorm_types.InterfaceMap{"value": fmt.Sprintf("%d", v)}
	case float32, float64:
		d.Payload = gorm_types.InterfaceMap{"value": fmt.Sprintf("%f", v)}
	case bool:
		d.Payload = gorm_types.InterfaceMap{"value": strconv.FormatBool(v)}
	case nil:
		d.Payload = gorm_types.InterfaceMap{}
	default:
		// Use JSON marshalling for all other types
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal Payload: %w", err)
		}
		var result gorm_types.InterfaceMap
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return fmt.Errorf("failed to unmarshal into InterfaceMap: %w", err)
		}
		d.Payload = result
	}
	return nil
}

func NewEvent(k string, v interface{}) *Event {
	md := &Event{
		EventType: k,
	}
	md.SetValue(v)
	return md
}
