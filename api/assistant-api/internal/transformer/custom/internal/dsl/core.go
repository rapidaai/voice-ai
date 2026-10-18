// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_dsl

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	FrameJSON   = "json"
	FrameBinary = "binary"
	FrameText   = "text"
)

type VariableResolver func(name string) (any, error)

type Frame struct {
	Kind   string
	Binary []byte
	JSON   any
	Text   string
}

type When struct {
	Frame  string `json:"frame,omitempty"`
	Path   string `json:"path,omitempty"`
	Equals any    `json:"equals,omitempty"`
}

type RequestWhen struct {
	Packet string `json:"packet"`
}

type Send struct {
	Frame string `json:"frame"`
	Body  any    `json:"body"`
}

type RequestRule struct {
	When RequestWhen `json:"when"`
	Send Send        `json:"send"`
}

type ResponseRule struct {
	When When           `json:"when"`
	Emit map[string]any `json:"emit"`
}

type Contract struct {
	SupportedVariables      []string
	SupportedRequestPackets []string
	SupportedRequestFrames  []string
	SupportedPathRoots      []string
	RequestValidationScopes map[string]any
	SupportedResponseFrames []string
	SupportedEmitKeys       []string
	AllowedFrameSelectors   []string
	AllowDecodeBase64       bool
}

type Core struct {
	errorPrefix string
}

func NewCore(errorPrefix string) *Core {
	return &Core{errorPrefix: strings.TrimSpace(errorPrefix)}
}

func (core *Core) Errorf(format string, args ...any) error {
	if core == nil || core.errorPrefix == "" {
		return fmt.Errorf(format, args...)
	}
	return fmt.Errorf("%s: %s", core.errorPrefix, fmt.Sprintf(format, args...))
}

func (core *Core) BuildConnectionURL(baseURL string, queryParams map[string]any, resolve VariableResolver) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", core.Errorf("invalid base url: %v", err)
	}
	if len(queryParams) == 0 {
		return parsed.String(), nil
	}

	rendered, err := core.RenderObject(queryParams, resolve)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	for key, value := range rendered {
		if value == nil {
			continue
		}
		primitive, err := core.ToPrimitiveString(value)
		if err != nil {
			return "", core.Errorf("query param %q must resolve to primitive: %v", key, err)
		}
		query.Set(key, primitive)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (core *Core) RenderObject(template map[string]any, resolve VariableResolver) (map[string]any, error) {
	rendered, err := core.renderNode(template, resolve)
	if err != nil {
		return nil, err
	}
	object, ok := rendered.(map[string]any)
	if !ok {
		return nil, core.Errorf("rendered request must be an object")
	}
	return object, nil
}

func (core *Core) ValidateRequestObject(template map[string]any, contract Contract, field string) error {
	allowedVariables := toStringSet(contract.SupportedVariables)
	return core.validateRequestNode(template, allowedVariables, field)
}

func (core *Core) ValidateScopedObject(template any, contract Contract, validationScope any, field string) error {
	allowedRoots := toStringSet(contract.SupportedPathRoots)
	if err := core.validateScopedNode(template, allowedRoots, field); err != nil {
		return err
	}
	if validationScope == nil {
		return nil
	}
	rendered, err := core.evalScopedNode(template, validationScope)
	if err != nil {
		return err
	}
	if !isJSONValue(rendered) {
		return core.Errorf("%s must resolve to JSON value", field)
	}
	if _, ok := rendered.(map[string]any); !ok {
		return core.Errorf("%s must resolve to JSON object", field)
	}
	return nil
}

func (core *Core) ValidateQueryParams(template map[string]any, contract Contract, field string) error {
	allowedVariables := toStringSet(contract.SupportedVariables)
	for key, value := range template {
		if strings.HasPrefix(key, "$") {
			return core.Errorf("%s uses unsupported operator %q", field, key)
		}
		if err := core.validateQueryValue(value, allowedVariables, fmt.Sprintf("%s.%s", field, key)); err != nil {
			return err
		}
	}
	return nil
}

func (core *Core) ValidateRequestRules(rules []RequestRule, contract Contract, field string) error {
	if len(rules) == 0 {
		return core.Errorf("%s must contain at least one rule", field)
	}

	allowedPackets := toStringSet(contract.SupportedRequestPackets)
	allowedFrames := toStringSet(contract.SupportedRequestFrames)
	allowedRoots := toStringSet(contract.SupportedPathRoots)

	for index, rule := range rules {
		packet := strings.TrimSpace(rule.When.Packet)
		if packet == "" || !allowedPackets[packet] {
			return core.Errorf(
				"%s[%d].when.packet must be %s",
				field,
				index,
				formatQuotedChoices(contract.SupportedRequestPackets),
			)
		}

		frame := strings.TrimSpace(rule.Send.Frame)
		if frame == "" || !allowedFrames[frame] {
			return core.Errorf(
				"%s[%d].send.frame must be %s",
				field,
				index,
				formatQuotedChoices(contract.SupportedRequestFrames),
			)
		}
		if rule.Send.Body == nil {
			return core.Errorf("%s[%d].send.body is required", field, index)
		}
		if err := core.validateScopedNode(
			rule.Send.Body,
			allowedRoots,
			fmt.Sprintf("%s[%d].send.body", field, index),
		); err != nil {
			return err
		}

		if scope, found := contract.RequestValidationScopes[packet]; found && scope != nil {
			rendered, err := core.evalScopedNode(rule.Send.Body, scope)
			if err != nil {
				return err
			}
			if err := core.validateRenderedRequestBody(
				rendered,
				frame,
				fmt.Sprintf("%s[%d].send.body", field, index),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (core *Core) ValidateResponseRules(rules []ResponseRule, contract Contract, field string) error {
	if len(rules) == 0 {
		return core.Errorf("%s must contain at least one rule", field)
	}

	allowedFrames := toStringSet(contract.SupportedResponseFrames)
	allowedEmitKeys := toStringSet(contract.SupportedEmitKeys)
	allowedFrameSelectors := toStringSet(contract.AllowedFrameSelectors)

	for index, rule := range rules {
		frame := strings.TrimSpace(rule.When.Frame)
		if frame == "" || !allowedFrames[frame] {
			return core.Errorf(
				"%s[%d].when.frame must be %s",
				field,
				index,
				formatQuotedChoices(contract.SupportedResponseFrames),
			)
		}

		switch frame {
		case FrameBinary:
			if strings.TrimSpace(rule.When.Path) != "" || rule.When.Equals != nil {
				return core.Errorf("%s[%d].when.path and when.equals cannot be used with %q", field, index, FrameBinary)
			}
		case FrameJSON:
			hasPath := strings.TrimSpace(rule.When.Path) != ""
			hasEquals := rule.When.Equals != nil
			if hasPath != hasEquals {
				return core.Errorf("%s[%d].when.path and when.equals must be provided together", field, index)
			}
			if hasEquals && !isJSONPrimitive(rule.When.Equals) {
				return core.Errorf("%s[%d].when.equals must be a primitive JSON value", field, index)
			}
		case FrameText:
			if strings.TrimSpace(rule.When.Path) != "" {
				return core.Errorf("%s[%d].when.path cannot be used with %q", field, index, FrameText)
			}
			if rule.When.Equals != nil && !isJSONPrimitive(rule.When.Equals) {
				return core.Errorf("%s[%d].when.equals must be a primitive JSON value", field, index)
			}
		}

		if len(rule.Emit) == 0 {
			return core.Errorf("%s[%d].emit must not be empty", field, index)
		}

		for emitKey, expr := range rule.Emit {
			if !allowedEmitKeys[emitKey] {
				return core.Errorf("%s[%d].emit.%s is not supported", field, index, emitKey)
			}
			if err := core.validateResponseNode(expr, contract, allowedFrameSelectors, field, index, emitKey); err != nil {
				return err
			}
		}
	}

	return nil
}

func (core *Core) MatchRequestWhen(when RequestWhen, packet string) bool {
	return strings.TrimSpace(when.Packet) == strings.TrimSpace(packet)
}

func (core *Core) EvalRequestRuleBody(expr any, scope any) (any, error) {
	return core.evalScopedNode(expr, scope)
}

func (core *Core) renderNode(node any, resolve VariableResolver) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		if rawVar, ok := typed["$var"]; ok {
			if len(typed) != 1 {
				return nil, core.Errorf("$var expression must only include $var")
			}
			varName, ok := rawVar.(string)
			if !ok || varName == "" {
				return nil, core.Errorf("$var must be non-empty string")
			}
			if resolve == nil {
				return nil, core.Errorf("unknown variable %q", varName)
			}
			return resolve(varName)
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return nil, core.Errorf("$cast expression must include only $cast and value")
			}
			castKind, ok := rawCast.(string)
			if !ok || castKind == "" {
				return nil, core.Errorf("$cast must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, core.Errorf("$cast requires value")
			}
			value, err := core.renderNode(valueExpr, resolve)
			if err != nil {
				return nil, err
			}
			return core.CastValue(castKind, value)
		}

		out := make(map[string]any, len(typed))
		for key, value := range typed {
			resolved, err := core.renderNode(value, resolve)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			resolved, err := core.renderNode(item, resolve)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	default:
		return node, nil
	}
}

func (core *Core) ParseFrame(messageType int, payload []byte, isBinary func(messageType int) bool) (Frame, error) {
	if isBinary != nil && isBinary(messageType) {
		return Frame{
			Kind:   FrameBinary,
			Binary: append([]byte(nil), payload...),
		}, nil
	}

	rawText := string(payload)
	var decoded any
	decoder := json.NewDecoder(strings.NewReader(rawText))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err == nil {
		var extra any
		extraErr := decoder.Decode(&extra)
		if extraErr == io.EOF {
			return Frame{Kind: FrameJSON, JSON: decoded, Text: rawText}, nil
		}
		return Frame{Kind: FrameText, Text: rawText}, nil
	}
	return Frame{Kind: FrameText, Text: rawText}, nil
}

func (core *Core) MatchWhen(when When, frame Frame) (bool, error) {
	if when.Frame != "" && when.Frame != frame.Kind {
		return false, nil
	}

	switch frame.Kind {
	case FrameBinary:
		return true, nil
	case FrameText:
		if when.Equals == nil {
			return true, nil
		}
		return core.ValuesEqual(frame.Text, when.Equals), nil
	case FrameJSON:
		if strings.TrimSpace(when.Path) == "" {
			return true, nil
		}

		value, found := core.LookupJSONPath(frame.JSON, when.Path)
		if !found {
			return false, nil
		}
		if when.Equals == nil {
			return true, nil
		}
		return core.ValuesEqual(value, when.Equals), nil
	default:
		return false, core.Errorf("unsupported frame kind %q", frame.Kind)
	}
}

func (core *Core) EvalResponseExpr(expr any, frame Frame, contract Contract) (any, error) {
	allowedFrameSelectors := toStringSet(contract.AllowedFrameSelectors)
	return core.evalResponseNode(expr, frame, contract, allowedFrameSelectors)
}

func (core *Core) LookupJSONPath(root any, path string) (any, bool) {
	current := root
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, found := typed[part]
			if !found {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func (core *Core) CastValue(kind string, value any) (any, error) {
	switch kind {
	case "string":
		return core.ToString(value)
	case "number":
		return core.ToNumber(value)
	case "boolean":
		return core.ToBool(value)
	default:
		return nil, core.Errorf("unsupported cast %q", kind)
	}
}

func (core *Core) ToPrimitiveString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case json.Number:
		return typed.String(), nil
	case int:
		return strconv.Itoa(typed), nil
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", typed), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", typed), nil
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("value type %T is not primitive", value)
	}
}

func (core *Core) ToString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case json.Number:
		return typed.String(), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", typed), nil
	}
}

func (core *Core) ToNumber(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return intValue, nil
		}
		floatValue, err := typed.Float64()
		if err != nil {
			return nil, core.Errorf("invalid number %q", typed.String())
		}
		return floatValue, nil
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(typed).Int(), nil
	case uint, uint8, uint16, uint32, uint64:
		unsignedValue := reflect.ValueOf(typed).Uint()
		if unsignedValue > math.MaxInt64 {
			return nil, core.Errorf("cannot cast %d to number: exceeds maximum signed integer", unsignedValue)
		}
		return int64(unsignedValue), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if intValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return intValue, nil
		}
		floatValue, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, core.Errorf("cannot cast %q to number", trimmed)
		}
		return floatValue, nil
	default:
		return nil, core.Errorf("cannot cast %T to number", value)
	}
}

func (core *Core) ToFloat64(value any) (float64, error) {
	number, err := core.ToNumber(value)
	if err != nil {
		return 0, err
	}
	switch typed := number.(type) {
	case int64:
		return float64(typed), nil
	case float64:
		return typed, nil
	default:
		return 0, core.Errorf("cannot cast %T to float64", value)
	}
}

func (core *Core) ToBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		boolValue, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, core.Errorf("cannot cast %q to boolean", typed)
		}
		return boolValue, nil
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			if intValue == 0 || intValue == 1 {
				return intValue == 1, nil
			}
		}
		if floatValue, err := typed.Float64(); err == nil {
			if floatValue == 0 || floatValue == 1 {
				return floatValue == 1, nil
			}
		}
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(typed).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(typed).Uint() != 0, nil
	case float32:
		return typed != 0, nil
	case float64:
		return typed != 0, nil
	}
	return false, core.Errorf("cannot cast %T to boolean", value)
}

func (core *Core) ToBytes(value any) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...), nil
	case string:
		return []byte(typed), nil
	default:
		return nil, core.Errorf("cannot cast %T to bytes", value)
	}
}

func (core *Core) ValuesEqual(left, right any) bool {
	return reflect.DeepEqual(core.normalize(left), core.normalize(right))
}

func (core *Core) normalize(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return intValue
		}
		if floatValue, err := typed.Float64(); err == nil {
			if math.Trunc(floatValue) == floatValue {
				return int64(floatValue)
			}
			return floatValue
		}
		return typed.String()
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case uint:
		unsignedValue := uint64(typed)
		if unsignedValue > math.MaxInt64 {
			return unsignedValue
		}
		return int64(unsignedValue)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		if typed > math.MaxInt64 {
			return typed
		}
		return int64(typed)
	case float32:
		floatValue := float64(typed)
		if math.Trunc(floatValue) == floatValue {
			return int64(floatValue)
		}
		return floatValue
	case float64:
		if math.Trunc(typed) == typed {
			return int64(typed)
		}
		return typed
	default:
		return value
	}
}

func (core *Core) validateRequestNode(node any, allowedVariables map[string]bool, field string) error {
	switch typed := node.(type) {
	case map[string]any:
		if rawVar, ok := typed["$var"]; ok {
			if len(typed) != 1 {
				return core.Errorf("%s $var expression must only include $var", field)
			}
			varName, ok := rawVar.(string)
			if !ok || strings.TrimSpace(varName) == "" {
				return core.Errorf("%s $var must be non-empty string", field)
			}
			if !allowedVariables[varName] {
				return core.Errorf("%s uses unsupported variable %q", field, varName)
			}
			return nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return core.Errorf("%s $cast expression must include only $cast and value", field)
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return core.Errorf("%s $cast must be non-empty string", field)
			}
			if !isSupportedCast(castKind) {
				return core.Errorf("%s uses unsupported cast %q", field, castKind)
			}
			valueExpr, found := typed["value"]
			if !found {
				return core.Errorf("%s $cast requires value", field)
			}
			return core.validateRequestNode(valueExpr, allowedVariables, field)
		}
		for key, value := range typed {
			if strings.HasPrefix(key, "$") {
				return core.Errorf("%s uses unsupported operator %q", field, key)
			}
			if err := core.validateRequestNode(value, allowedVariables, field); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for _, item := range typed {
			if err := core.validateRequestNode(item, allowedVariables, field); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (core *Core) validateScopedNode(node any, allowedRoots map[string]bool, field string) error {
	switch typed := node.(type) {
	case map[string]any:
		if rawPath, ok := typed["$path"]; ok {
			if len(typed) != 1 {
				return core.Errorf("%s $path expression must only include $path", field)
			}
			path, ok := rawPath.(string)
			if !ok || strings.TrimSpace(path) == "" {
				return core.Errorf("%s $path must be non-empty string", field)
			}
			root := getPathRoot(path)
			if !allowedRoots[root] {
				return core.Errorf(
					"%s $path root must be %s",
					field,
					formatQuotedChoices(sortedKeys(allowedRoots)),
				)
			}
			return nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return core.Errorf("%s $cast expression must include only $cast and value", field)
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return core.Errorf("%s $cast must be non-empty string", field)
			}
			if !isSupportedCast(castKind) {
				return core.Errorf("%s uses unsupported cast %q", field, castKind)
			}
			valueExpr, found := typed["value"]
			if !found {
				return core.Errorf("%s $cast requires value", field)
			}
			return core.validateScopedNode(valueExpr, allowedRoots, field)
		}
		for key, value := range typed {
			if strings.HasPrefix(key, "$") {
				return core.Errorf("%s uses unsupported operator %q", field, key)
			}
			if err := core.validateScopedNode(value, allowedRoots, field); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for _, item := range typed {
			if err := core.validateScopedNode(item, allowedRoots, field); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (core *Core) evalScopedNode(node any, scope any) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		if rawPath, ok := typed["$path"]; ok {
			if len(typed) != 1 {
				return nil, core.Errorf("$path expression must only include $path")
			}
			path, ok := rawPath.(string)
			if !ok || strings.TrimSpace(path) == "" {
				return nil, core.Errorf("$path must be non-empty string")
			}
			value, found := core.LookupJSONPath(scope, path)
			if !found {
				return nil, core.Errorf("request path %q not found", path)
			}
			return value, nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return nil, core.Errorf("$cast expression must include only $cast and value")
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return nil, core.Errorf("$cast must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, core.Errorf("$cast requires value")
			}
			value, err := core.evalScopedNode(valueExpr, scope)
			if err != nil {
				return nil, err
			}
			return core.CastValue(castKind, value)
		}
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			resolved, err := core.evalScopedNode(value, scope)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			resolved, err := core.evalScopedNode(item, scope)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	default:
		return node, nil
	}
}

func (core *Core) validateRenderedRequestBody(rendered any, frame string, field string) error {
	if rendered == nil {
		return core.Errorf("%s must resolve to a value", field)
	}

	switch frame {
	case FrameBinary:
		switch rendered.(type) {
		case []byte, string:
			return nil
		default:
			return core.Errorf("%s must resolve to bytes or string for %q frames", field, FrameBinary)
		}
	case FrameText:
		if _, err := core.ToString(rendered); err != nil {
			return core.Errorf("%s must resolve to string for %q frames: %v", field, FrameText, err)
		}
		return nil
	case FrameJSON:
		if !isJSONValue(rendered) {
			return core.Errorf("%s must resolve to JSON value for %q frames", field, FrameJSON)
		}
		return nil
	default:
		return core.Errorf("%s uses unsupported frame %q", field, frame)
	}
}

func (core *Core) validateQueryValue(node any, allowedVariables map[string]bool, field string) error {
	switch typed := node.(type) {
	case map[string]any:
		if rawVar, ok := typed["$var"]; ok {
			if len(typed) != 1 {
				return core.Errorf("%s $var expression must only include $var", field)
			}
			varName, ok := rawVar.(string)
			if !ok || strings.TrimSpace(varName) == "" {
				return core.Errorf("%s $var must be non-empty string", field)
			}
			if !allowedVariables[varName] {
				return core.Errorf("%s uses unsupported variable %q", field, varName)
			}
			return nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return core.Errorf("%s $cast expression must include only $cast and value", field)
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return core.Errorf("%s $cast must be non-empty string", field)
			}
			if !isSupportedCast(castKind) {
				return core.Errorf("%s uses unsupported cast %q", field, castKind)
			}
			valueExpr, found := typed["value"]
			if !found {
				return core.Errorf("%s $cast requires value", field)
			}
			return core.validateQueryValue(valueExpr, allowedVariables, field)
		}
		for key := range typed {
			if strings.HasPrefix(key, "$") {
				return core.Errorf("%s uses unsupported operator %q", field, key)
			}
		}
		return core.Errorf("%s must resolve to primitive value", field)
	case []any:
		return core.Errorf("%s must resolve to primitive value", field)
	default:
		if !isPrimitiveValue(node) {
			return core.Errorf("%s must resolve to primitive value", field)
		}
		return nil
	}
}

func (core *Core) validateResponseNode(
	node any,
	contract Contract,
	allowedFrameSelectors map[string]bool,
	field string,
	index int,
	emitKey string,
) error {
	switch typed := node.(type) {
	case map[string]any:
		if rawPath, ok := typed["$path"]; ok {
			if len(typed) != 1 {
				return core.Errorf("%s[%d].emit.%s $path expression must only include $path", field, index, emitKey)
			}
			path, ok := rawPath.(string)
			if !ok || strings.TrimSpace(path) == "" {
				return core.Errorf("%s[%d].emit.%s $path must be non-empty string", field, index, emitKey)
			}
			return nil
		}
		if rawFrame, ok := typed["$frame"]; ok {
			if len(typed) != 1 {
				return core.Errorf("%s[%d].emit.%s $frame expression must only include $frame", field, index, emitKey)
			}
			frame, ok := rawFrame.(string)
			if !ok || strings.TrimSpace(frame) == "" {
				return core.Errorf("%s[%d].emit.%s $frame must be non-empty string", field, index, emitKey)
			}
			if !allowedFrameSelectors[frame] {
				return core.Errorf("%s[%d].emit.%s $frame must be %s", field, index, emitKey, formatQuotedChoices(contract.AllowedFrameSelectors))
			}
			return nil
		}
		if rawDecode, ok := typed["$decode"]; ok {
			if len(typed) != 2 {
				return core.Errorf("%s[%d].emit.%s $decode expression must include only $decode and value", field, index, emitKey)
			}
			if !contract.AllowDecodeBase64 {
				return core.Errorf("%s[%d].emit.%s does not support $decode", field, index, emitKey)
			}
			decodeKind, ok := rawDecode.(string)
			if !ok || decodeKind != "base64" {
				return core.Errorf("%s[%d].emit.%s $decode must be %q", field, index, emitKey, "base64")
			}
			valueExpr, found := typed["value"]
			if !found {
				return core.Errorf("%s[%d].emit.%s $decode requires value", field, index, emitKey)
			}
			return core.validateResponseNode(valueExpr, contract, allowedFrameSelectors, field, index, emitKey)
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return core.Errorf("%s[%d].emit.%s $cast expression must include only $cast and value", field, index, emitKey)
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return core.Errorf("%s[%d].emit.%s $cast must be non-empty string", field, index, emitKey)
			}
			if !isSupportedCast(castKind) {
				return core.Errorf("%s[%d].emit.%s uses unsupported cast %q", field, index, emitKey, castKind)
			}
			valueExpr, found := typed["value"]
			if !found {
				return core.Errorf("%s[%d].emit.%s $cast requires value", field, index, emitKey)
			}
			return core.validateResponseNode(valueExpr, contract, allowedFrameSelectors, field, index, emitKey)
		}
		for key, value := range typed {
			if strings.HasPrefix(key, "$") {
				return core.Errorf("%s[%d].emit.%s uses unsupported operator %q", field, index, emitKey, key)
			}
			if err := core.validateResponseNode(value, contract, allowedFrameSelectors, field, index, emitKey); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for _, item := range typed {
			if err := core.validateResponseNode(item, contract, allowedFrameSelectors, field, index, emitKey); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (core *Core) evalResponseNode(node any, frame Frame, contract Contract, allowedFrameSelectors map[string]bool) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		if rawPath, ok := typed["$path"]; ok {
			if len(typed) != 1 {
				return nil, core.Errorf("$path expression must only include $path")
			}
			path, ok := rawPath.(string)
			if !ok || strings.TrimSpace(path) == "" {
				return nil, core.Errorf("$path must be non-empty string")
			}
			if frame.Kind != FrameJSON {
				return nil, core.Errorf("$path can only be used with %q frames", FrameJSON)
			}
			value, found := core.LookupJSONPath(frame.JSON, path)
			if !found {
				return nil, core.Errorf("response path %q not found", path)
			}
			return value, nil
		}
		if rawFrame, ok := typed["$frame"]; ok {
			if len(typed) != 1 {
				return nil, core.Errorf("$frame expression must only include $frame")
			}
			selector, ok := rawFrame.(string)
			if !ok || strings.TrimSpace(selector) == "" {
				return nil, core.Errorf("$frame must be non-empty string")
			}
			if !allowedFrameSelectors[selector] {
				return nil, core.Errorf("unsupported frame selector %q", selector)
			}
			if frame.Kind != selector {
				return nil, core.Errorf("current frame is not %q", selector)
			}
			switch selector {
			case FrameBinary:
				return append([]byte(nil), frame.Binary...), nil
			case FrameJSON:
				return frame.JSON, nil
			case FrameText:
				return frame.Text, nil
			default:
				return nil, core.Errorf("unsupported frame selector %q", selector)
			}
		}
		if rawDecode, ok := typed["$decode"]; ok {
			if len(typed) != 2 {
				return nil, core.Errorf("$decode expression must include only $decode and value")
			}
			decodeKind, ok := rawDecode.(string)
			if !ok || strings.TrimSpace(decodeKind) == "" {
				return nil, core.Errorf("$decode must be non-empty string")
			}
			if !contract.AllowDecodeBase64 || decodeKind != "base64" {
				return nil, core.Errorf("unsupported decode %q", decodeKind)
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, core.Errorf("$decode requires value")
			}
			value, err := core.evalResponseNode(valueExpr, frame, contract, allowedFrameSelectors)
			if err != nil {
				return nil, err
			}
			encoded, err := core.ToString(value)
			if err != nil {
				return nil, err
			}
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, core.Errorf("base64 decode failed: %v", err)
			}
			return decoded, nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return nil, core.Errorf("$cast expression must include only $cast and value")
			}
			castKind, ok := rawCast.(string)
			if !ok || strings.TrimSpace(castKind) == "" {
				return nil, core.Errorf("$cast must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, core.Errorf("$cast requires value")
			}
			value, err := core.evalResponseNode(valueExpr, frame, contract, allowedFrameSelectors)
			if err != nil {
				return nil, err
			}
			return core.CastValue(castKind, value)
		}

		out := make(map[string]any, len(typed))
		for key, value := range typed {
			resolved, err := core.evalResponseNode(value, frame, contract, allowedFrameSelectors)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			resolved, err := core.evalResponseNode(item, frame, contract, allowedFrameSelectors)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	default:
		return node, nil
	}
}

func toStringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func isSupportedCast(kind string) bool {
	switch kind {
	case "string", "number", "boolean":
		return true
	default:
		return false
	}
}

func isJSONPrimitive(value any) bool {
	switch value.(type) {
	case nil, string, bool, float64, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

func isPrimitiveValue(value any) bool {
	switch value.(type) {
	case nil, string, bool, float32, float64, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

func isJSONValue(value any) bool {
	switch typed := value.(type) {
	case nil, string, bool, float32, float64, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	case []byte:
		return false
	case []any:
		for _, item := range typed {
			if !isJSONValue(item) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, item := range typed {
			if !isJSONValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func getPathRoot(path string) string {
	for _, part := range strings.Split(path, ".") {
		if strings.TrimSpace(part) != "" {
			return part
		}
	}
	return ""
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatQuotedChoices(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%q", values[0])
	case 2:
		return fmt.Sprintf("%q or %q", values[0], values[1])
	default:
		parts := make([]string, 0, len(values))
		for _, value := range values[:len(values)-1] {
			parts = append(parts, fmt.Sprintf("%q", value))
		}
		return fmt.Sprintf("%s, or %q", strings.Join(parts, ", "), values[len(values)-1])
	}
}
