// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (r *recorder) prepareObservation(scope Scope, record Record) (observation, error) {
	if !validator.NonNil(record) {
		return observation{}, errors.New("observability: record is nil")
	}
	now := r.clock()
	if !validator.NonNil(scope) {
		return observation{}, errors.New("observability: scope is required")
	}
	scope = scope.WithGlobal(r.globalScope)
	if err := ValidateScope(scope); err != nil {
		return observation{}, err
	}

	switch typed := record.(type) {
	case RecordLog:
		if !validator.NotBlank(typed.Message) {
			return observation{}, errors.New("observability: log message is required")
		}
		switch typed.Level {
		case "":
			typed.Level = LevelInfo
		case LevelInfo, LevelError, LevelDebug, LevelCritical:
		default:
			return observation{}, fmt.Errorf("observability: invalid log level %q", typed.Level)
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Attributes = typed.Attributes.Clone()
		return observation{scope: scope, record: typed}, nil
	case RecordEvent:
		if !validator.NotBlank(typed.Event.String()) {
			return observation{}, errors.New("observability: event is required")
		}
		if typed.Component == "" {
			typed.Component = typed.Event.Component()
		}
		if typed.Component == ComponentUnknown {
			return observation{}, fmt.Errorf("observability: component is required for event %q", typed.Event)
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Attributes = typed.Attributes.Clone()
		return observation{scope: scope, record: typed}, nil
	case RecordMetric:
		if len(typed.Metrics) == 0 {
			return observation{}, errors.New("observability: at least one metric is required")
		}
		for i, metric := range typed.Metrics {
			if metric == nil || !validator.NotBlank(metric.GetName()) {
				return observation{}, fmt.Errorf("observability: metric[%d] name is required", i)
			}
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Metrics = cloneMetricRecords(typed.Metrics)
		typed.Attributes = typed.Attributes.Clone()
		return observation{scope: scope, record: typed}, nil
	case RecordMetadata:
		if len(typed.Metadata) == 0 {
			return observation{}, errors.New("observability: at least one metadata entry is required")
		}
		for i, metadata := range typed.Metadata {
			if metadata == nil || !validator.NotBlank(metadata.GetKey()) {
				return observation{}, fmt.Errorf("observability: metadata[%d] key is required", i)
			}
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Metadata = cloneMetadataRecords(typed.Metadata)
		return observation{scope: scope, record: typed}, nil
	case RecordUsage:
		if typed.Duration <= 0 {
			return observation{}, errors.New("observability: usage duration must be greater than zero")
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Attributes = typed.Attributes.Clone()
		return observation{scope: scope, record: typed}, nil
	case RecordWebhook:
		if !validator.NotBlank(typed.Event.String()) {
			return observation{}, errors.New("observability: webhook event is required")
		}
		if typed.Payload == nil {
			typed.Payload = NewV1WebhookPayload(nil)
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.Payload = cloneV1WebhookPayload(typed.Payload)
		return observation{scope: scope, record: typed}, nil
	case RecordToolLog:
		switch typed.Operation {
		case ToolLogOperationCreate, ToolLogOperationUpdate:
		default:
			return observation{}, fmt.Errorf("observability: invalid tool log operation %q", typed.Operation)
		}
		if !validator.NotBlank(typed.ToolCallID) {
			return observation{}, errors.New("observability: tool_call_id is required")
		}
		if typed.Status == "" {
			return observation{}, errors.New("observability: tool log status is required")
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.RequestPayload = cloneBytes(typed.RequestPayload)
		typed.ResponsePayload = cloneBytes(typed.ResponsePayload)
		return observation{scope: scope, record: typed}, nil
	case RecordRequestLog:
		if !validator.NotBlank(typed.Source) {
			return observation{}, errors.New("observability: request log source is required")
		}
		if !validator.NotBlank(typed.SourceEvent) {
			return observation{}, errors.New("observability: request log source_event is required")
		}
		if !validator.NotBlank(typed.ContextID) {
			return observation{}, errors.New("observability: request log context_id is required")
		}
		if typed.Status == "" {
			return observation{}, errors.New("observability: request log status is required")
		}
		if typed.OccurredAt.IsZero() {
			typed.OccurredAt = now
		}
		typed.ErrorMessage = cloneStringPointer(typed.ErrorMessage)
		typed.RequestPayload = cloneBytes(typed.RequestPayload)
		typed.ResponsePayload = cloneBytes(typed.ResponsePayload)
		return observation{scope: scope, record: typed}, nil
	default:
		return observation{}, fmt.Errorf("observability: unsupported record type %T", record)
	}
}

func cloneMetricRecords(metrics []*protos.Metric) []*protos.Metric {
	if len(metrics) == 0 {
		return nil
	}
	clonedMetrics := make([]*protos.Metric, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			clonedMetrics = append(clonedMetrics, nil)
			continue
		}
		clonedMetrics = append(clonedMetrics, &protos.Metric{
			Name:        metric.GetName(),
			Value:       metric.GetValue(),
			Description: metric.GetDescription(),
		})
	}
	return clonedMetrics
}

func cloneMetadataRecords(metadataRecords []*protos.Metadata) []*protos.Metadata {
	if len(metadataRecords) == 0 {
		return nil
	}
	clonedMetadataRecords := make([]*protos.Metadata, 0, len(metadataRecords))
	for _, metadata := range metadataRecords {
		if metadata == nil {
			clonedMetadataRecords = append(clonedMetadataRecords, nil)
			continue
		}
		clonedMetadataRecords = append(clonedMetadataRecords, &protos.Metadata{
			Id:    metadata.GetId(),
			Key:   metadata.GetKey(),
			Value: metadata.GetValue(),
		})
	}
	return clonedMetadataRecords
}

func cloneV1WebhookPayload(payload V1WebhookPayload) V1WebhookPayload {
	if payload == nil {
		return NewV1WebhookPayload(nil)
	}
	if clonedPayload, ok := cloneWebhookPayloadValue(payload).(V1WebhookPayload); ok {
		return clonedPayload
	}
	return payload
}

func cloneWebhookPayload(payload interface{}) interface{} {
	switch typedPayload := payload.(type) {
	case map[string]interface{}:
		if len(typedPayload) == 0 {
			return map[string]interface{}{}
		}
		clonedPayload := make(map[string]interface{}, len(typedPayload))
		for key, value := range typedPayload {
			clonedPayload[key] = cloneWebhookPayloadValue(value)
		}
		return clonedPayload
	default:
		return payload
	}
}

func cloneWebhookPayloadValue(value interface{}) interface{} {
	switch typedValue := value.(type) {
	case map[string]interface{}:
		return cloneWebhookPayload(typedValue)
	case map[string]string:
		clonedMap := make(map[string]string, len(typedValue))
		for key, value := range typedValue {
			clonedMap[key] = value
		}
		return clonedMap
	case []interface{}:
		clonedSlice := make([]interface{}, len(typedValue))
		for index, value := range typedValue {
			clonedSlice[index] = cloneWebhookPayloadValue(value)
		}
		return clonedSlice
	case []map[string]interface{}:
		clonedSlice := make([]map[string]interface{}, len(typedValue))
		for index, value := range typedValue {
			if clonedValue, ok := cloneWebhookPayload(value).(map[string]interface{}); ok {
				clonedSlice[index] = clonedValue
			}
		}
		return clonedSlice
	case []string:
		return append([]string(nil), typedValue...)
	case []byte:
		return cloneBytes(typedValue)
	default:
		reflectedValue := reflect.ValueOf(value)
		if reflectedValue.IsValid() && reflectedValue.Kind() == reflect.Struct {
			return cloneWebhookPayloadStruct(reflectedValue).Interface()
		}
		return value
	}
}

func cloneWebhookPayloadStruct(value reflect.Value) reflect.Value {
	clonedStruct := reflect.New(value.Type()).Elem()
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		clonedField := clonedStruct.Field(index)
		if !field.CanInterface() || !clonedField.CanSet() {
			clonedField.Set(field)
			continue
		}
		clonedValue := cloneWebhookPayloadValue(field.Interface())
		if clonedValue == nil {
			clonedField.Set(reflect.Zero(clonedField.Type()))
			continue
		}
		clonedReflectValue := reflect.ValueOf(clonedValue)
		switch {
		case clonedReflectValue.Type().AssignableTo(clonedField.Type()):
			clonedField.Set(clonedReflectValue)
		case clonedReflectValue.Type().ConvertibleTo(clonedField.Type()):
			clonedField.Set(clonedReflectValue.Convert(clonedField.Type()))
		default:
			clonedField.Set(field)
		}
	}
	return clonedStruct
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	clonedValue := *value
	return &clonedValue
}
