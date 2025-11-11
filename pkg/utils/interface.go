package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// M wraps a map for convenient getters.
type Option map[string]interface{}

func (m Option) GetUint64(key string) (uint64, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("key %q not found or nil", key)
	}

	switch t := v.(type) {
	case uint64:
		return t, nil
	case uint32:
		return uint64(t), nil
	case uint16:
		return uint64(t), nil
	case uint8:
		return uint64(t), nil
	case uint:
		return uint64(t), nil

	case int64:
		if t < 0 {
			return 0, fmt.Errorf("negative int64 for %q", key)
		}
		return uint64(t), nil
	case int32:
		if t < 0 {
			return 0, fmt.Errorf("negative int32 for %q", key)
		}
		return uint64(t), nil
	case int16:
		if t < 0 {
			return 0, fmt.Errorf("negative int16 for %q", key)
		}
		return uint64(t), nil
	case int8:
		if t < 0 {
			return 0, fmt.Errorf("negative int8 for %q", key)
		}
		return uint64(t), nil
	case int:
		if t < 0 {
			return 0, fmt.Errorf("negative int for %q", key)
		}
		return uint64(t), nil

	case float64:
		return floatToUint64(t, key)
	case float32:
		return floatToUint64(float64(t), key)

	case string:
		return parseUintString(t)

	case json.Number:
		// Try as integer first
		if u, err := strconv.ParseUint(t.String(), 10, 64); err == nil {
			return u, nil
		}
		// Fall back to float then check integrality
		f, err := t.Float64()
		if err != nil {
			return 0, fmt.Errorf("json.Number parse err for %q: %w", key, err)
		}
		return floatToUint64(f, key)

	case []byte:
		return parseUintString(string(t))

	case interface{ String() string }: // fmt.Stringer compatible
		return parseUintString(t.String())

	default:
		return 0, fmt.Errorf("unsupported type %T for %q", v, key)
	}
}

func parseUintString(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	// base 0 lets strconv handle 0x..., 0..., etc.
	u, err := strconv.ParseUint(s, 0, 64)
	if err == nil {
		return u, nil
	}

	// If it's a float-like string (e.g., "123.0"), try float then validate integrality.
	if f, ferr := strconv.ParseFloat(s, 64); ferr == nil {
		return floatToUint64(f, "value")
	}

	return 0, fmt.Errorf("cannot parse %q as uint64", s)
}

func floatToUint64(f float64, key string) (uint64, error) {
	if f < 0 {
		return 0, fmt.Errorf("negative float for %q", key)
	}
	if f > math.MaxUint64 {
		return 0, fmt.Errorf("float exceeds uint64 range for %q", key)
	}
	if frac := math.Mod(f, 1.0); frac != 0 {
		return 0, fmt.Errorf("non-integer float for %q", key)
	}
	return uint64(f), nil
}

func (m Option) GetString(key string) (string, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", fmt.Errorf("key %q not found or nil", key)
	}

	switch t := v.(type) {
	case string:
		return t, nil
	case []byte:
		return string(t), nil
	case fmt.Stringer:
		return t.String(), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", t), nil
	case float32, float64:
		return strconv.FormatFloat(float64(t.(float64)), 'f', -1, 64), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func (m Option) GetUint32(key string) (uint32, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("key %q not found or nil", key)
	}

	switch t := v.(type) {
	case uint32:
		return t, nil
	case uint64:
		if t > math.MaxUint32 {
			return 0, fmt.Errorf("uint64 value exceeds uint32 range for %q", key)
		}
		return uint32(t), nil
	case uint16, uint8, uint:
		return uint32(t.(uint)), nil
	case int64, int32, int16, int8, int:
		i := reflect.ValueOf(t).Int()
		if i < 0 || i > math.MaxUint32 {
			return 0, fmt.Errorf("integer value out of uint32 range for %q", key)
		}
		return uint32(i), nil
	case float64:
		return floatToUint32(t, key)
	case float32:
		return floatToUint32(float64(t), key)
	case string:
		return parseUint32String(t)
	case json.Number:
		return parseUint32JsonNumber(t, key)
	case []byte:
		return parseUint32String(string(t))
	case interface{ String() string }:
		return parseUint32String(t.String())
	default:
		return 0, fmt.Errorf("unsupported type %T for %q", v, key)
	}
}

func (m Option) GetFloat64(key string) (float64, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("key %q not found or nil", key)
	}

	switch t := v.(type) {
	case float64:
		return t, nil
	case float32:
		return float64(t), nil
	case int64, int32, int16, int8, int, uint64, uint32, uint16, uint8, uint:
		return float64(reflect.ValueOf(t).Int()), nil
	case string:
		return strconv.ParseFloat(t, 64)
	case json.Number:
		return t.Float64()
	case []byte:
		return strconv.ParseFloat(string(t), 64)
	case interface{ String() string }:
		return strconv.ParseFloat(t.String(), 64)
	default:
		return 0, fmt.Errorf("unsupported type %T for %q", v, key)
	}
}

// Helper functions for GetUint32
func floatToUint32(f float64, key string) (uint32, error) {
	if f < 0 || f > math.MaxUint32 {
		return 0, fmt.Errorf("float out of uint32 range for %q", key)
	}
	if frac := math.Mod(f, 1.0); frac != 0 {
		return 0, fmt.Errorf("non-integer float for %q", key)
	}
	return uint32(f), nil
}

func parseUint32String(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	u, err := strconv.ParseUint(s, 0, 32)
	if err == nil {
		return uint32(u), nil
	}
	if f, ferr := strconv.ParseFloat(s, 64); ferr == nil {
		return floatToUint32(f, "value")
	}
	return 0, fmt.Errorf("cannot parse %q as uint32", s)
}

func parseUint32JsonNumber(n json.Number, key string) (uint32, error) {
	if u, err := n.Int64(); err == nil {
		if u < 0 || u > math.MaxUint32 {
			return 0, fmt.Errorf("json.Number out of uint32 range for %q", key)
		}
		return uint32(u), nil
	}
	f, err := n.Float64()
	if err != nil {
		return 0, fmt.Errorf("json.Number parse error for %q: %w", key, err)
	}
	return floatToUint32(f, key)
}

func (m Option) GetStringMap(key string) (map[string]string, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, fmt.Errorf("key %q not found or nil", key)
	}

	result := make(map[string]string)

	switch rawParams := v.(type) {
	case string:
		if err := json.Unmarshal([]byte(rawParams), &result); err != nil {
			return nil, fmt.Errorf("failed to parse %s as JSON: %v", key, err)
		}
	case []interface{}:
		for _, item := range rawParams {
			if pair, ok := item.(map[string]interface{}); ok {
				for k, v := range pair {
					result[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	case map[string]interface{}:
		for k, v := range rawParams {
			result[k] = fmt.Sprintf("%v", v)
		}
	default:
		return nil, fmt.Errorf("%s is not in the expected format (string or []interface{})", key)
	}

	return result, nil
}

func (m Option) GetBool(key string) (bool, error) {
	v, ok := m[key]
	if !ok || v == nil {
		return false, fmt.Errorf("key %q not found or nil", key)
	}

	switch t := v.(type) {
	case bool:
		return t, nil
	case string:
		// Parse true/false as strings
		b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(t)))
		if err != nil {
			return false, fmt.Errorf("cannot parse string %q as bool for %q", t, key)
		}
		return b, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		i := reflect.ValueOf(t).Int()
		if i == 1 {
			return true, nil
		} else if i == 0 {
			return false, nil
		}
		return false, fmt.Errorf("integer value %d is not convertible to a bool for %q", i, key)
	case float64, float32:
		f := reflect.ValueOf(t).Float()
		if f == 1.0 {
			return true, nil
		} else if f == 0.0 {
			return false, nil
		}
		return false, fmt.Errorf("float value %f is not convertible to a bool for %q", f, key)
	case json.Number:
		// Try as integer first
		if i, err := t.Int64(); err == nil && (i == 0 || i == 1) {
			return i == 1, nil
		}
		// Fall back to float for edge cases
		f, err := t.Float64()
		if err == nil && (f == 0.0 || f == 1.0) {
			return f == 1.0, nil
		}
		return false, fmt.Errorf("json.Number value %q is not convertible to a bool for %q", t.String(), key)
	case []byte:
		return m.GetBool(string(t))
	case interface{ String() string }:
		return m.GetBool(t.String())
	default:
		return false, fmt.Errorf("unsupported type %T for %q", v, key)
	}
}
