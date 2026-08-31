// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/validator"
)

const defaultIndexPrefix = "rapida-timeline"

type Config struct {
	Logger           commons.Logger
	OpenSearch       connectors.OpenSearchConnector
	OpenSearchConfig *configs.OpenSearchConfig
	IndexPrefix      string
}

type Collector struct {
	logger      commons.Logger
	opensearch  connectors.OpenSearchConnector
	indexPrefix string
}

func New(ctx context.Context, cfg Config) (observability.Collector, error) {
	openSearch, err := openSearchConnector(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if !validator.NonNil(openSearch) {
		return observability.NoopCollector{}, nil
	}
	indexPrefix := strings.TrimSpace(cfg.IndexPrefix)
	if !validator.NotBlank(indexPrefix) {
		indexPrefix = defaultIndexPrefix
	}
	return &Collector{
		logger:      cfg.Logger,
		opensearch:  openSearch,
		indexPrefix: indexPrefix,
	}, nil
}

func (c *Collector) Key() string {
	indexPrefix := strings.TrimSpace(c.indexPrefix)
	if !validator.NotBlank(indexPrefix) {
		return "timeline"
	}
	return "timeline:" + indexPrefix
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, _ observability.Context, record observability.Record) error {
	if !validator.NonNil(c) || !validator.NonNil(c.opensearch) {
		return nil
	}
	organizationID := scope.GlobalScopeValue().OrganizationID
	switch typed := record.(type) {
	case observability.RecordLog:
		doc := newDocument("log", scope, typed.ID, typed.OccurredAt)
		doc.Level = string(typed.Level)
		doc.Title = typed.Message
		doc.Attributes = typed.Attributes.Clone()
		return c.bulk(ctx, c.index(organizationID, doc.OccurredAt), doc)
	case observability.RecordEvent:
		doc := newDocument("event", scope, typed.ID, typed.OccurredAt)
		doc.Name = typed.Event.String()
		doc.Component = typed.Component.String()
		doc.Attributes = typed.Attributes.Clone()
		return c.bulk(ctx, c.index(organizationID, doc.OccurredAt), doc)
	case observability.RecordMetric:
		doc := newDocument("metric", scope, typed.ID, typed.OccurredAt)
		doc.Attributes = typed.Attributes.Clone()
		return c.bulk(ctx, c.index(organizationID, doc.OccurredAt), doc)
	case observability.RecordMetadata:
		doc := newDocument("metadata", scope, typed.ID, typed.OccurredAt)
		return c.bulk(ctx, c.index(organizationID, doc.OccurredAt), doc)
	case observability.RecordUsage:
		doc := newDocument("usage", scope, typed.ID, typed.OccurredAt)
		doc.Component = typed.Component.String()
		doc.Attributes = typed.Attributes.Clone()
		return c.bulk(ctx, c.index(organizationID, doc.OccurredAt), doc)
	default:
		return nil
	}
}

func (c *Collector) Close(ctx context.Context) error {
	if !validator.NonNil(c) || !validator.NonNil(c.opensearch) {
		return nil
	}
	return c.opensearch.Disconnect(ctx)
}

func openSearchConnector(ctx context.Context, cfg Config) (connectors.OpenSearchConnector, error) {
	if validator.NonNil(cfg.OpenSearch) {
		return cfg.OpenSearch, nil
	}
	if !validator.NonNil(cfg.OpenSearchConfig) {
		return nil, nil
	}
	if !cfg.OpenSearchConfig.IsValid() {
		return nil, errors.New("observability timeline: opensearch config is required")
	}
	openSearch := connectors.NewOpenSearchConnector(cfg.OpenSearchConfig, cfg.Logger)
	if err := openSearch.Connect(ctx); err != nil {
		return nil, err
	}
	return openSearch, nil
}

func (c *Collector) index(organizationID uint64, at time.Time) string {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	organization := "global"
	if organizationID != 0 {
		organization = strconv.FormatUint(organizationID, 10)
	}
	return c.indexPrefix + "-" + organization + "-" + at.UTC().Format("20060102")
}

func (c *Collector) bulk(ctx context.Context, index string, doc document) error {
	var sb strings.Builder
	if validator.NotBlank(doc.ID) {
		sb.WriteString(fmt.Sprintf(`{ "index": { "_index": "%s", "_id": "%s" } }`, index, doc.ID))
	} else {
		sb.WriteString(fmt.Sprintf(`{ "index": { "_index": "%s" } }`, index))
	}
	sb.WriteByte('\n')
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	sb.Write(body)
	sb.WriteByte('\n')
	if err := c.opensearch.Bulk(ctx, sb.String()); err != nil {
		if validator.NonNil(c.logger) {
			c.logger.Errorf("observability/timeline/opensearch: bulk index error: %v", err)
		}
		return err
	}
	return nil
}

type document struct {
	ID                      string            `json:"id"`
	Kind                    string            `json:"kind"`
	Name                    string            `json:"name"`
	Component               string            `json:"component"`
	Level                   string            `json:"level"`
	Outcome                 string            `json:"outcome"`
	Title                   string            `json:"title"`
	ProjectID               uint64            `json:"projectId"`
	OrganizationID          uint64            `json:"organizationId"`
	Scope                   string            `json:"scope"`
	AssistantID             uint64            `json:"assistantId"`
	AssistantConversationID uint64            `json:"assistantConversationId"`
	MessageID               string            `json:"messageId,omitempty"`
	MessageRole             string            `json:"messageRole,omitempty"`
	ContextID               string            `json:"contextId"`
	Attributes              map[string]string `json:"attributes,omitempty"`
	OccurredAt              time.Time         `json:"occurredAt"`
}

func newDocument(kind string, scope observability.Scope, id string, occurredAt time.Time) document {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	doc := document{
		ID:             id,
		Kind:           kind,
		ProjectID:      scope.GlobalScopeValue().ProjectID,
		OrganizationID: scope.GlobalScopeValue().OrganizationID,
		Scope:          string(scope.ScopeType()),
		OccurredAt:     occurredAt,
	}
	switch typed := scope.(type) {
	case observability.ProjectScope:
		doc.ContextID = strconv.FormatUint(typed.GlobalScopeValue().ProjectID, 10)
	case observability.AssistantScope:
		doc.AssistantID = typed.AssistantScopeID()
		doc.ContextID = typed.ContextID()
	case observability.ConversationScope:
		doc.AssistantID = typed.AssistantScopeID()
		doc.AssistantConversationID = typed.ConversationScopeID()
		doc.ContextID = typed.ContextID()
	case observability.MessageScope:
		doc.AssistantID = typed.AssistantScopeID()
		doc.AssistantConversationID = typed.ConversationScopeID()
		doc.MessageID = typed.MessageScopeID()
		doc.MessageRole = string(typed.MessageScopeRole())
		doc.ContextID = typed.ContextID()
	}
	return doc
}
