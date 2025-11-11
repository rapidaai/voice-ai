package utils

/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
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
	return IntToString(uint64(v))
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
