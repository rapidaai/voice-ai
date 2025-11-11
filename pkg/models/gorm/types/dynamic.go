package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Dynamic struct {
	Data interface{}
}

func NewDynamic(in interface{}) Dynamic {
	return Dynamic{
		Data: in,
	}
}

// Value converts the data to a format suitable for database storage
func (d Dynamic) Value() (driver.Value, error) {
	switch v := d.Data.(type) {
	case string, int, float64, bool:
		return v, nil
	default:
		return json.Marshal(d.Data)
	}
}

// Scan reads the data from the database and converts it to interface{}
func (d *Dynamic) Scan(src interface{}) error {
	if src == nil {
		d.Data = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		// Try to parse JSON
		var jsonData interface{}
		if err := json.Unmarshal(v, &jsonData); err == nil {
			d.Data = jsonData
			return nil
		}
		d.Data = string(v) // Fallback to string if unmarshalling fails
	case string:
		d.Data = v
	case int64:
		d.Data = int(v)
	case float64:
		d.Data = v
	default:
		return errors.New("unsupported data type")
	}
	return nil
}

// Get retrieves the value with type assertion
func (d Dynamic) Get() interface{} {
	return d.Data
}

// GetString safely returns the value as a string
func (d Dynamic) GetString() (string, bool) {
	str, ok := d.Data.(string)
	return str, ok
}

// GetInt safely returns the value as an int
func (d Dynamic) GetInt() (int, bool) {
	num, ok := d.Data.(int)
	return num, ok
}

// GetMap safely returns the value as map[string]interface{}
func (d Dynamic) GetMap() (map[string]interface{}, bool) {
	m, ok := d.Data.(map[string]interface{})
	return m, ok
}
