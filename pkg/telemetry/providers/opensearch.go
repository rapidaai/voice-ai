// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/telemetry"
)

// OpenSearchExporter indexes logs, events, and metrics to dedicated OpenSearch indices.
type OpenSearchExporter struct {
	logger    commons.Logger
	config    OpenSearchConfig
	connector connectors.OpenSearchConnector
}

func NewOpenSearchExporter(
	logger commons.Logger,
	config OpenSearchConfig,
	connector connectors.OpenSearchConnector,
) *OpenSearchExporter {
	return &OpenSearchExporter{logger: logger, config: config, connector: connector}
}

func NewOpenSearchExporterFromOptions(
	ctx context.Context,
	logger commons.Logger,
	opts map[string]interface{},
) (telemetry.Exporter, error) {
	connectorConfig := &configs.OpenSearchConfig{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           connectorConfig,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(opts); err != nil {
		return nil, err
	}
	if !connectorConfig.IsValid() {
		return nil, fmt.Errorf("telemetry/opensearch: opensearch config is required")
	}
	exporterConfig, err := OpenSearchConfigFromOptions(opts)
	if err != nil {
		return nil, err
	}
	connector := connectors.NewOpenSearchConnector(connectorConfig, logger)
	if err := connector.Connect(ctx); err != nil {
		return nil, err
	}
	return &OpenSearchExporter{
		logger:    logger,
		config:    exporterConfig,
		connector: connector,
	}, nil
}

func (e *OpenSearchExporter) index(kind string, organizationID uint64, occurredAt time.Time) string {
	prefix := "rapida"
	if strings.TrimSpace(e.config.IndexPrefix) != "" {
		prefix = strings.TrimSpace(e.config.IndexPrefix)
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	organization := "global"
	if organizationID != 0 {
		organization = strconv.FormatUint(organizationID, 10)
	}
	return prefix + "-" + kind + "-" + organization + "-" + occurredAt.UTC().Format("20060102")
}

func (e *OpenSearchExporter) logIndex(organizationID uint64, occurredAt time.Time) string {
	return e.index("logs", organizationID, occurredAt)
}

func (e *OpenSearchExporter) eventIndex(organizationID uint64, occurredAt time.Time) string {
	return e.index("events", organizationID, occurredAt)
}

func (e *OpenSearchExporter) metricIndex(organizationID uint64, occurredAt time.Time) string {
	return e.index("metrics", organizationID, occurredAt)
}

type opensearchEventDoc struct {
	ID              string            `json:"id,omitempty"`
	Kind            string            `json:"kind"`
	Context         map[string]string `json:"context"`
	Event           string            `json:"event"`
	Component       string            `json:"component,omitempty"`
	ProjectID       uint64            `json:"projectId"`
	OrganizationID  uint64            `json:"organizationId"`
	Scope           string            `json:"scope,omitempty"`
	ScopeAttributes map[string]string `json:"scopeAttributes,omitempty"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	OccurredAt      time.Time         `json:"occurredAt"`
}

type opensearchLogDoc struct {
	ID              string            `json:"id,omitempty"`
	Kind            string            `json:"kind"`
	Context         map[string]string `json:"context"`
	Level           string            `json:"level"`
	Message         string            `json:"message"`
	ProjectID       uint64            `json:"projectId"`
	OrganizationID  uint64            `json:"organizationId"`
	Scope           string            `json:"scope,omitempty"`
	ScopeAttributes map[string]string `json:"scopeAttributes,omitempty"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	OccurredAt      time.Time         `json:"occurredAt"`
}

type opensearchMetricDoc struct {
	ID              string            `json:"id,omitempty"`
	Kind            string            `json:"kind"`
	Context         map[string]string `json:"context"`
	Name            string            `json:"name"`
	Value           string            `json:"value"`
	Description     string            `json:"description,omitempty"`
	ProjectID       uint64            `json:"projectId"`
	OrganizationID  uint64            `json:"organizationId"`
	Scope           string            `json:"scope,omitempty"`
	ScopeAttributes map[string]string `json:"scopeAttributes,omitempty"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	OccurredAt      time.Time         `json:"occurredAt"`
}

func (e *OpenSearchExporter) Export(ctx context.Context, scope telemetry.Scope, rec telemetry.Record) error {
	switch typed := rec.(type) {
	case telemetry.LogRecord:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		id := typed.ID
		if strings.TrimSpace(id) == "" {
			id = uuid.NewString()
		}
		doc := opensearchLogDoc{
			ID:              id,
			Kind:            "log",
			Context:         typed.Context,
			Level:           typed.Level,
			Message:         typed.Message,
			ProjectID:       scope.ProjectID,
			OrganizationID:  scope.OrganizationID,
			Scope:           scope.Name,
			ScopeAttributes: scope.ScopeAttributes,
			Attributes:      typed.Attributes,
			OccurredAt:      occurredAt,
		}
		return e.bulk(ctx, e.logIndex(doc.OrganizationID, doc.OccurredAt), doc.ID, doc)
	case telemetry.EventRecord:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		id := typed.ID
		if strings.TrimSpace(id) == "" {
			id = uuid.NewString()
		}
		doc := opensearchEventDoc{
			ID:              id,
			Kind:            "event",
			Context:         typed.Context,
			Event:           typed.Event,
			Component:       typed.Component,
			ProjectID:       scope.ProjectID,
			OrganizationID:  scope.OrganizationID,
			Scope:           scope.Name,
			ScopeAttributes: scope.ScopeAttributes,
			Attributes:      typed.Attributes,
			OccurredAt:      occurredAt,
		}
		return e.bulk(ctx, e.eventIndex(doc.OrganizationID, doc.OccurredAt), doc.ID, doc)
	case telemetry.MetricRecord:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		id := typed.ID
		if strings.TrimSpace(id) == "" {
			id = uuid.NewString()
		}
		doc := opensearchMetricDoc{
			ID:              id,
			Kind:            "metric",
			Context:         typed.Context,
			Name:            typed.Name,
			Value:           typed.Value,
			Description:     typed.Description,
			ProjectID:       scope.ProjectID,
			OrganizationID:  scope.OrganizationID,
			Scope:           scope.Name,
			ScopeAttributes: scope.ScopeAttributes,
			Attributes:      typed.Attributes,
			OccurredAt:      occurredAt,
		}
		return e.bulk(ctx, e.metricIndex(doc.OrganizationID, doc.OccurredAt), doc.ID, doc)
	default:
		return nil
	}
}

func (e *OpenSearchExporter) Close(ctx context.Context) error {
	if e.connector != nil {
		return e.connector.Disconnect(ctx)
	}
	return nil
}

func (e *OpenSearchExporter) bulk(ctx context.Context, index string, id string, doc interface{}) error {
	var sb strings.Builder
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}
	meta := fmt.Sprintf(`{ "index": { "_index": "%s", "_id": "%s" } }`, index, id)
	sb.WriteString(meta + "\n")
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	sb.WriteString(string(b) + "\n")
	if err := e.connector.Bulk(ctx, sb.String()); err != nil {
		e.logger.Errorf("telemetry/opensearch: bulk index error: %v", err)
		return err
	}
	return nil
}
