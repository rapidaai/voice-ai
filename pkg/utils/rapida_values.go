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
 *  Author: Prashant Srivastav
 *
 */

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func AnyMapToInterfaceMap(anyMap map[string]*anypb.Any) (map[string]interface{}, error) {
	interfaceMap := make(map[string]interface{})
	for key, anyValue := range anyMap {
		value, err := AnyToInterface(anyValue)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Any to interface for key %s: %v", key, err)
		}
		interfaceMap[key] = value
	}
	return interfaceMap, nil
}
func AnyToBool(anyValue *anypb.Any) (bool, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return false, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_BoolValue:
			return v.BoolValue, nil
		case *structpb.Value_StringValue:
			return strconv.ParseBool(v.StringValue)
		default:
			return false, fmt.Errorf("unsupported value type for bool conversion: %T", v)
		}
	}

	boolWrapper := &wrapperspb.BoolValue{}
	err := anyValue.UnmarshalTo(boolWrapper)
	if err != nil {
		return false, err
	}
	return boolWrapper.GetValue(), nil
}

func AnyToBytes(anyValue *anypb.Any) ([]byte, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return nil, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_StringValue:
			return []byte(v.StringValue), nil
		default:
			return nil, fmt.Errorf("unsupported value type for bytes conversion: %T", v)
		}
	}

	bytesWrapper := &wrapperspb.BytesValue{}
	err := anyValue.UnmarshalTo(bytesWrapper)
	if err != nil {
		return nil, err
	}
	return bytesWrapper.GetValue(), nil
}

func AnyToFloat32(anyValue *anypb.Any) (float32, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return float32(v.NumberValue), nil
		case *structpb.Value_StringValue:
			f, err := strconv.ParseFloat(v.StringValue, 32)
			return float32(f), err
		default:
			return 0, fmt.Errorf("unsupported value type for float32 conversion: %T", v)
		}
	}

	floatWrapper := &wrapperspb.FloatValue{}
	err := anyValue.UnmarshalTo(floatWrapper)
	if err != nil {
		return 0, err
	}
	return floatWrapper.GetValue(), nil
}

func AnyToFloat64(anyValue *anypb.Any) (float64, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return v.NumberValue, nil
		case *structpb.Value_StringValue:
			return strconv.ParseFloat(v.StringValue, 64)
		default:
			return 0, fmt.Errorf("unsupported value type for float64 conversion: %T", v)
		}
	}

	floatWrapper := &wrapperspb.DoubleValue{}
	err := anyValue.UnmarshalTo(floatWrapper)
	if err != nil {
		return 0, err
	}
	return floatWrapper.GetValue(), nil
}

func AnyToInt(anyValue *anypb.Any) (int, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return int(v.NumberValue), nil
		case *structpb.Value_StringValue:
			i, err := strconv.Atoi(v.StringValue)
			return i, err
		default:
			return 0, fmt.Errorf("unsupported value type for int conversion: %T", v)
		}
	}

	intWrapper := &wrapperspb.Int32Value{}
	err := anyValue.UnmarshalTo(intWrapper)
	if err != nil {
		return 0, err
	}
	return int(intWrapper.GetValue()), nil
}

func AnyToInt32(anyValue *anypb.Any) (int32, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return int32(v.NumberValue), nil
		case *structpb.Value_StringValue:
			i, err := strconv.ParseInt(v.StringValue, 10, 32)
			return int32(i), err
		default:
			return 0, fmt.Errorf("unsupported value type for int32 conversion: %T", v)
		}
	}

	intWrapper := &wrapperspb.Int32Value{}
	err := anyValue.UnmarshalTo(intWrapper)
	if err != nil {
		return 0, err
	}
	return intWrapper.GetValue(), nil
}

func AnyToInt64(anyValue *anypb.Any) (int64, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return int64(v.NumberValue), nil
		case *structpb.Value_StringValue:
			return strconv.ParseInt(v.StringValue, 10, 64)
		default:
			return 0, fmt.Errorf("unsupported value type for int64 conversion: %T", v)
		}
	}

	intWrapper := &wrapperspb.Int64Value{}
	err := anyValue.UnmarshalTo(intWrapper)
	if err != nil {
		return 0, err
	}
	return intWrapper.GetValue(), nil
}

func AnyToJSON(anyValue *anypb.Any) (map[string]interface{}, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Any to Value: %w", err)
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_StructValue:
			return v.StructValue.AsMap(), nil
		case *structpb.Value_StringValue:
			var result map[string]interface{}
			err := json.Unmarshal([]byte(v.StringValue), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal string value to JSON: %w", err)
			}
			return result, nil
		default:
			return nil, fmt.Errorf("unsupported value type for JSON conversion: %T", v)
		}
	}

	var value map[string]interface{}
	bytesWrapper := &wrapperspb.BytesValue{}
	err := anyValue.UnmarshalTo(bytesWrapper)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytesWrapper.Value, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func AnyToString(anyValue *anypb.Any) (string, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return "", err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_StringValue:
			return v.StringValue, nil
		case *structpb.Value_NumberValue:
			return fmt.Sprintf("%v", v.NumberValue), nil
		case *structpb.Value_BoolValue:
			return strconv.FormatBool(v.BoolValue), nil
		default:
			return "", fmt.Errorf("unsupported value type for string conversion: %T", v)
		}
	}

	stringWrapper := &wrapperspb.StringValue{}
	err := anyValue.UnmarshalTo(stringWrapper)
	if err != nil {
		return "", err
	}
	return stringWrapper.GetValue(), nil
}

func AnyToUInt32(anyValue *anypb.Any) (uint32, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return uint32(v.NumberValue), nil
		case *structpb.Value_StringValue:
			i, err := strconv.ParseUint(v.StringValue, 10, 32)
			return uint32(i), err
		default:
			return 0, fmt.Errorf("unsupported value type for uint32 conversion: %T", v)
		}
	}

	uintWrapper := &wrapperspb.UInt32Value{}
	err := anyValue.UnmarshalTo(uintWrapper)
	if err != nil {
		return 0, err
	}
	return uintWrapper.GetValue(), nil
}

func AnyToUInt64(anyValue *anypb.Any) (uint64, error) {
	if anyValue.TypeUrl == "type.googleapis.com/google.protobuf.Value" {
		value := &structpb.Value{}
		if err := anyValue.UnmarshalTo(value); err != nil {
			return 0, err
		}
		switch v := value.Kind.(type) {
		case *structpb.Value_NumberValue:
			return uint64(v.NumberValue), nil
		case *structpb.Value_StringValue:
			return strconv.ParseUint(v.StringValue, 10, 64)
		default:
			return 0, fmt.Errorf("unsupported value type for uint64 conversion: %T", v)
		}
	}

	uintWrapper := &wrapperspb.UInt64Value{}
	err := anyValue.UnmarshalTo(uintWrapper)
	if err != nil {
		return 0, err
	}
	return uintWrapper.GetValue(), nil
}
func AnyToInterface(anyValue *anypb.Any) (interface{}, error) {
	if anyValue == nil {
		return nil, nil
	}
	var value interface{}
	switch {
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.BoolValue"):
		v := &wrapperspb.BoolValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.BytesValue"):
		v := &wrapperspb.BytesValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.DoubleValue"):
		v := &wrapperspb.DoubleValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Duration"):
		v := &durationpb.Duration{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.AsDuration()
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Empty"):
		v := &emptypb.Empty{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = nil
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.FloatValue"):
		v := &wrapperspb.FloatValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Int32Value"):
		v := &wrapperspb.Int32Value{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Int64Value"):
		v := &wrapperspb.Int64Value{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.StringValue"):
		v := &wrapperspb.StringValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Struct"):
		v := &structpb.Struct{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.AsMap()
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.Timestamp"):
		v := &timestamppb.Timestamp{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.AsTime()
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.UInt32Value"):
		v := &wrapperspb.UInt32Value{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.UInt64Value"):
		v := &wrapperspb.UInt64Value{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.Value
	case strings.HasSuffix(anyValue.TypeUrl, "type.googleapis.com/google.protobuf.ListValue"):
		v := &structpb.ListValue{}
		if err := anyValue.UnmarshalTo(v); err != nil {
			return nil, err
		}
		value = v.AsSlice()
	default:
		jsonBytes, err := protojson.Marshal(anyValue)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal anypb.Any to JSON: %v", err)
		}
		if err := json.Unmarshal(jsonBytes, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON to interface{}: %v", err)
		}
	}

	return value, nil
}

func BoolToAny(value bool) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Bool(value), proto.MarshalOptions{})
	return anyValue, err
}

func BytesToAny(value []byte) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Bytes(value), proto.MarshalOptions{})
	return anyValue, err
}

func Float32ToAny(value float32) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Float(value), proto.MarshalOptions{})
	return anyValue, err
}

func Float64ToAny(value float64) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Double(value), proto.MarshalOptions{})
	return anyValue, err
}

func Int32ToAny(value int32) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Int32(value), proto.MarshalOptions{})
	return anyValue, err
}

func Int64ToAny(value int64) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.Int64(value), proto.MarshalOptions{})
	return anyValue, err
}

func JSONToAny(value map[string]interface{}) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = anypb.MarshalFrom(anyValue, &wrapperspb.BytesValue{Value: jsonData}, proto.MarshalOptions{})
	if err == nil {
		return anyValue, nil
	}
	anyValue.TypeUrl = "type.googleapis.com/google.protobuf.Struct"
	anyValue.Value = jsonData
	return anyValue, nil
}

func StringToAny(value string) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.String(value), proto.MarshalOptions{})
	return anyValue, err
}

func ToIntAny(value int) *anypb.Any {
	v, _ := Int32ToAny(int32(value))
	return v
}

func ToJSONAny(value map[string]interface{}) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = anypb.MarshalFrom(anyValue, &wrapperspb.BytesValue{Value: jsonData}, proto.MarshalOptions{})
	if err == nil {
		return anyValue, nil
	}
	anyValue.TypeUrl = "type.googleapis.com/google.protobuf.Struct"
	anyValue.Value = jsonData
	return anyValue, nil
}

func ToStringAny(value string) *anypb.Any {
	v, _ := StringToAny(value)
	return v
}

func ToUInt64Any(value uint64) *anypb.Any {
	v, _ := UInt64ToAny(value)
	return v
}

func UInt32ToAny(value uint32) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.UInt32(value), proto.MarshalOptions{})
	return anyValue, err
}

func UInt64ToAny(value uint64) (*anypb.Any, error) {
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, wrapperspb.UInt64(value), proto.MarshalOptions{})
	return anyValue, err
}

func InterfaceToAnyValue(v interface{}) (*anypb.Any, error) {
	switch val := v.(type) {
	case bool:
		return BoolToAny(val)
	case []byte:
		return BytesToAny(val)
	case float32:
		return Float32ToAny(val)
	case float64:
		return Float64ToAny(val)
	case int:
		return Int32ToAny(int32(val))
	case int32:
		return Int32ToAny(val)
	case int64:
		return Int64ToAny(val)
	case string:
		return StringToAny(val)
	case uint32:
		return UInt32ToAny(val)
	case uint64:
		return UInt64ToAny(val)
	case map[string]interface{}:
		return JSONToAny(val)
	case []map[string]string:
		return JSONListToAny(val)

	default:
		// For unsupported types, attempt to convert to JSON
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("unsupported type and failed to convert to JSON: %w", err)
		}
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
			return nil, fmt.Errorf("failed to convert JSON to map: %w", err)
		}
		return JSONToAny(jsonMap)
	}
}
func JSONListToAny(values []map[string]string) (*anypb.Any, error) {
	// Convert each map to a structpb.Struct
	structList := make([]*structpb.Struct, len(values))
	for i, value := range values {
		strct, err := structpb.NewStruct(map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("error creating struct at index %d: %w", i, err)
		}
		for k, v := range value {
			strct.Fields[k] = structpb.NewStringValue(v)
		}
		structList[i] = strct
	}

	// Create a ListValue containing the structs
	listValue := &structpb.ListValue{
		Values: make([]*structpb.Value, len(structList)),
	}
	for i, strct := range structList {
		listValue.Values[i] = structpb.NewStructValue(strct)
	}

	// Create Any protobuf
	anyValue := &anypb.Any{}
	err := anypb.MarshalFrom(anyValue, listValue, proto.MarshalOptions{})
	if err != nil {
		return nil, fmt.Errorf("error marshaling to Any: %w", err)
	}

	return anyValue, nil
}
