package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"strings"
)

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

func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			if nestedMap, ok := v.(map[string]interface{}); ok {
				if existingNestedMap, exists := result[k].(map[string]interface{}); exists {
					result[k] = MergeMaps(existingNestedMap, nestedMap)
				} else {
					result[k] = MergeMaps(nestedMap)
				}
			} else {
				result[k] = v
			}
		}
	}
	return result
}

func GetCaseInsensitiveKeyValue(cfg map[string]string, key string) (string, bool) {
	if value, ok := cfg[key]; ok {
		return value, true
	}
	if value, ok := cfg[strings.ToUpper(key)]; ok {
		return value, true
	}
	return "", false
}

func EmbeddingToFloat64[T float32 | float64](embedding []T) []float64 {
	float64Embedding := make([]float64, len(embedding))
	for i, val := range embedding {
		float64Embedding[i] = float64(val)
	}
	return float64Embedding
}

func EmbeddingToFloat32[T float32 | float64](embedding []T) []float32 {
	float32Embedding := make([]float32, len(embedding))
	for i, val := range embedding {
		float32Embedding[i] = float32(val)
	}
	return float32Embedding
}

// Convert a slice of float32 to a byte array
func Float64SliceToByteArray(data []float64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EmbeddingToBase64(embedding []float64) string {
	byteArray, err := Float64SliceToByteArray(embedding)
	if err != nil {
		return ""
	}
	base64Str := base64.StdEncoding.EncodeToString(byteArray)
	return base64Str
}
