package gorm_types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type DocumentMap map[string]interface{}

// Value Marshal
func (jsonField DocumentMap) Value() (driver.Value, error) {
	return json.Marshal(jsonField)
}

// Scan Unmarshal
func (jsonField *DocumentMap) Scan(value interface{}) error {
	if value == nil {
		*jsonField = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, jsonField)
	case string:
		return json.Unmarshal([]byte(v), jsonField)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

type DocumentStruct struct {
	Source      DocumentSource `json:"source"`
	Type        DocumentType   `json:"type"`
	DocumentUrl string         `json:"documentUrl"`
}

type InternalDocumentStruct struct {
	DocumentStruct
	Size int `json:"size"`
}

type ExternalDocumentStruct struct {
	DocumentStruct
	// keep adding fields depends on what all is needed for document to parse and undersatnd
}
