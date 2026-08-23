// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import (
	"encoding/json"
	"fmt"
	"time"
)

// Collection of knowledge for given organization
func OrganizationKnowledgeCollection(orgId, projectId, knowledgeId uint64) string {
	return fmt.Sprintf("%d__%d__%d", orgId, projectId, knowledgeId)
}

// object prefix for given org
// object key
func OrganizationObjectPrefix(orgId, projectId uint64, prefix string) string {
	return fmt.Sprintf("%d/%d/%s", orgId, projectId, prefix)
}

func Ptr[T any](v T) *T {
	return &v
}

func UnPtr[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

func IntToString(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func DurationToString(v time.Duration) string {
	return Int64ToString(int64(v))
}

func ToJson(obj interface{}) map[string]interface{} {
	// Marshal the struct to JSON
	var result map[string]interface{}

	jsonData, err := json.Marshal(obj)
	if err != nil {
		return result
	}

	// Unmarshal the JSON to a map
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result
	}

	return result
}

func Serialize(data map[string]interface{}) ([]byte, error) {
	serializableData := make(map[string]interface{})
	for k, v := range data {
		switch v := v.(type) {
		case error:
			serializableData[k] = v.Error()
		default:
			_, err := json.Marshal(v)
			if err == nil {
				serializableData[k] = v
			}
		}
	}
	return json.Marshal(serializableData)
}
