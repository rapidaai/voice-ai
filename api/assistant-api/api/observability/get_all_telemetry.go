// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observability_api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func telemetryTermFilter(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{"term": map[string]interface{}{field: value}}
}

func telemetryExactStringFilter(field string, value string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				telemetryTermFilter(field+".keyword", value),
				telemetryTermFilter(field, value),
			},
			"minimum_should_match": 1,
		},
	}
}

func telemetryAnyExactStringFilter(fields []string, value string) map[string]interface{} {
	should := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		should = append(should, telemetryExactStringFilter(field, value))
	}
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func telemetryTraceIDFilter(value string) map[string]interface{} {
	return telemetryAnyExactStringFilter([]string{
		"context.traceId",
		"context.traceID",
		"context.trace_id",
	}, value)
}

func telemetrySearchFilter(value string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query": value,
						"fields": []string{
							"message",
							"event",
							"event.keyword",
							"name",
							"name.keyword",
							"description",
							"level",
							"id",
							"id.keyword",
						},
					},
				},
				telemetryTraceIDFilter(value),
			},
			"minimum_should_match": 1,
		},
	}
}

type telemetryQueryParts struct {
	filter    []interface{}
	indices   []string
	must      []interface{}
	timeRange map[string]interface{}
}

func newTelemetryQueryParts(organizationID uint64, projectID uint64) telemetryQueryParts {
	return telemetryQueryParts{
		indices: []string{"rapida-logs-*", "rapida-events-*", "rapida-metrics-*"},
		filter: []interface{}{
			telemetryTermFilter("organizationId", organizationID),
			telemetryTermFilter("projectId", projectID),
		},
		must:      []interface{}{},
		timeRange: map[string]interface{}{},
	}
}

func (parts *telemetryQueryParts) applyCriteria(criteriaList []*protos.Criteria) {
	for _, criteria := range criteriaList {
		if criteria == nil {
			continue
		}

		key := strings.TrimSpace(criteria.GetKey())
		value := strings.TrimSpace(criteria.GetValue())
		if key == "" || value == "" {
			continue
		}

		switch key {
		case "kind":
			switch strings.ToLower(value) {
			case "log":
				parts.indices = []string{"rapida-logs-*"}
				parts.filter = append(parts.filter, telemetryExactStringFilter("kind", "log"))
			case "event":
				parts.indices = []string{"rapida-events-*"}
				parts.filter = append(parts.filter, telemetryExactStringFilter("kind", "event"))
			case "metric":
				parts.indices = []string{"rapida-metrics-*"}
				parts.filter = append(parts.filter, telemetryExactStringFilter("kind", "metric"))
			}
		case "id", "scope", "event", "component", "name", "level":
			parts.filter = append(parts.filter, telemetryExactStringFilter(key, value))
		case "assistantId":
			parts.filter = append(parts.filter, telemetryExactStringFilter("scopeAttributes.assistantId", value))
		case "assistantConversationId":
			parts.filter = append(parts.filter, telemetryExactStringFilter("scopeAttributes.assistantConversationId", value))
		case "messageId":
			parts.filter = append(parts.filter, telemetryExactStringFilter("scopeAttributes.messageId", value))
		case "messageRole":
			parts.filter = append(parts.filter, telemetryExactStringFilter("scopeAttributes.messageRole", value))
		case "traceId":
			parts.filter = append(parts.filter, telemetryTraceIDFilter(value))
		case "timestamp":
			switch strings.TrimSpace(criteria.GetLogic()) {
			case ">=":
				parts.timeRange["gte"] = value
			case ">":
				parts.timeRange["gt"] = value
			case "<=":
				parts.timeRange["lte"] = value
			case "<":
				parts.timeRange["lt"] = value
			case "=":
				parts.timeRange["gte"] = value
				if timestamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
					parts.timeRange["lt"] = timestamp.Add(time.Minute).Format(time.RFC3339Nano)
				} else {
					parts.timeRange["lte"] = value
				}
			}
		case "search":
			parts.must = append(parts.must, telemetrySearchFilter(value))
		default:
			if strings.HasPrefix(key, "attributes.") || strings.HasPrefix(key, "scopeAttributes.") || strings.HasPrefix(key, "context.") {
				parts.filter = append(parts.filter, telemetryExactStringFilter(key, value))
			}
		}
	}

	if len(parts.timeRange) > 0 {
		parts.filter = append(parts.filter, map[string]interface{}{"range": map[string]interface{}{"occurredAt": parts.timeRange}})
	}
}

func (api *observabilityGrpcApi) GetAllTelemetry(
	ctx context.Context,
	request *protos.GetAllTelemetryRequest,
) (*protos.GetAllTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	projectContext, authErr := types.RequireProject(iAuth)
	if !isAuthenticated || authErr != nil {
		api.logger.Errorf("unauthenticated request for GetAllTelemetry")
		return exceptions.AuthenticationError[protos.GetAllTelemetryResponse]()
	}

	if api.opensearch == nil {
		return &protos.GetAllTelemetryResponse{Code: 200, Success: true}, nil
	}

	page := int(request.GetPaginate().GetPage())
	if page < 1 {
		page = 1
	}
	size := int(request.GetPaginate().GetPageSize())
	if size < 1 || size > 100 {
		size = 20
	}
	from := (page - 1) * size

	queryParts := newTelemetryQueryParts(
		projectContext.OrganizationID,
		projectContext.ProjectID,
	)
	queryParts.applyCriteria(request.GetCriterias())

	orderColumn := "occurredAt"
	orderDirection := "desc"
	if order := request.GetOrder(); order != nil {
		switch order.GetColumn() {
		case "occurredAt", "kind", "scope", "event", "name", "level":
			orderColumn = order.GetColumn()
		}
		if strings.EqualFold(order.GetOrder(), "asc") {
			orderDirection = "asc"
		}
	}

	boolQuery := map[string]interface{}{"filter": queryParts.filter}
	if len(queryParts.must) > 0 {
		boolQuery["must"] = queryParts.must
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"sort": []interface{}{
			map[string]interface{}{orderColumn: map[string]interface{}{"order": orderDirection}},
		},
		"from": from,
		"size": size,
	}

	body, _ := json.Marshal(query)
	hits := api.opensearch.Search(ctx, queryParts.indices, string(body))
	if hits.Error() != nil {
		api.logger.Errorf("unable to query telemetry: %v", hits.Error())
		return exceptions.BadRequestError[protos.GetAllTelemetryResponse]("Unable to get telemetry.")
	}

	records := make([]*protos.ObservabilityRecord, 0, len(hits.Hits.Hits))
	for _, hit := range hits.Hits.Hits {
		src, _ := hit["_source"].(map[string]interface{})
		if src == nil {
			continue
		}

		strVal := func(v interface{}) string {
			if v == nil {
				return ""
			}
			if f, ok := v.(float64); ok {
				return strconv.FormatUint(uint64(f), 10)
			}
			return fmt.Sprintf("%v", v)
		}
		uintVal := func(v interface{}) uint64 {
			if f, ok := v.(float64); ok {
				return uint64(f)
			}
			if s, ok := v.(string); ok {
				out, _ := utils.StringToUint64(s)
				return out
			}
			return 0
		}
		mapVal := func(v interface{}) map[string]string {
			out := map[string]string{}
			if raw, ok := v.(map[string]interface{}); ok {
				for key, value := range raw {
					out[key] = strVal(value)
				}
			}
			return out
		}
		timestampVal := func(v interface{}) *timestamppb.Timestamp {
			if value, ok := v.(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
					return timestamppb.New(t)
				}
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					return timestamppb.New(t)
				}
			}
			return nil
		}

		switch strVal(src["kind"]) {
		case "log":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Log{
					Log: &protos.ObservabilityLogRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_LOG,
						Level:           strVal(src["level"]),
						Message:         strVal(src["message"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		case "event":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Event{
					Event: &protos.ObservabilityEventRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_EVENT,
						Event:           strVal(src["event"]),
						Component:       strVal(src["component"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		case "metric":
			records = append(records, &protos.ObservabilityRecord{
				Record: &protos.ObservabilityRecord_Metric{
					Metric: &protos.ObservabilityMetricRecord{
						Id:              strVal(src["id"]),
						Kind:            protos.ObservabilityRecordKind_OBSERVABILITY_RECORD_KIND_METRIC,
						Name:            strVal(src["name"]),
						Value:           strVal(src["value"]),
						Description:     strVal(src["description"]),
						ProjectId:       uintVal(src["projectId"]),
						OrganizationId:  uintVal(src["organizationId"]),
						Scope:           strVal(src["scope"]),
						ScopeAttributes: mapVal(src["scopeAttributes"]),
						Attributes:      mapVal(src["attributes"]),
						Context:         mapVal(src["context"]),
						OccurredAt:      timestampVal(src["occurredAt"]),
					},
				},
			})
		}
	}

	return &protos.GetAllTelemetryResponse{
		Code:    200,
		Success: true,
		Data:    records,
		Paginated: &protos.Paginated{
			TotalItem:   uint32(hits.Hits.Total.Value),
			CurrentPage: uint32(page),
		},
	}, nil
}
