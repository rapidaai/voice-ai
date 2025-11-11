package internal_assistant_telemetry_exporters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type opensearchExporter struct {
	config              *config.AppConfig
	logger              commons.Logger
	opensearchConnector connectors.OpenSearchConnector
	prefix              string
}

func NewOpensearchAssistantTraceExporter(
	logger commons.Logger,
	config *config.AppConfig,
	opensearchConnector connectors.OpenSearchConnector,
) telemetry.VoiceAgentTraceExporter {
	return &opensearchExporter{
		logger:              logger,
		config:              config,
		opensearchConnector: opensearchConnector,
		prefix:              "assistant_conversation_",
	}
}

// Persist implements telemetry.Exporter.
func (ose *opensearchExporter) Export(
	ctx context.Context,
	iauth types.SimplePrinciple,
	options telemetry.ExportOption,
	stages []*telemetry.Telemetry) error {
	switch opts := options.(type) {
	case *telemetry.VoiceAgentExportOption:
		var bulkRequestBody strings.Builder
		for _, doc := range stages {

			// Prepare metadata (index and ID) for this document
			meta := fmt.Sprintf(`{ "index": { "_index": "%s", "_id": "%s" } }`, commons.TelemetryIndex(ose.config.IsDevelopment()), fmt.Sprintf("%s__%s", ose.prefix, uuid.NewString()))
			_, _ = bulkRequestBody.WriteString(meta + "\n")

			// Prepare document and append to buffer
			opensearchAssistantObject, err := json.Marshal(ToOpensearchTelemetry(
				doc,
				opts.AssistantId,
				opts.AssistantProviderModelId,
				opts.AssistantConversationId,
				*iauth.GetCurrentProjectId(),
				*iauth.GetCurrentOrganizationId(),
			))
			if err != nil {
				ose.logger.Errorf("unable to marshal document with error %+v", err)
				continue
			}
			_, _ = bulkRequestBody.WriteString(string(opensearchAssistantObject) + "\n")
		}

		// Execute bulk operation
		err := ose.opensearchConnector.Bulk(ctx, bulkRequestBody.String())
		if err != nil {
			ose.logger.Errorf("unable to execute bulk persist with error %+v", err)
			return err
		}
	}
	return nil
}

func (ose *opensearchExporter) Get(
	ctx context.Context,
	iauth types.SimplePrinciple,
	criterias []*protos.Criteria,
	paginate *protos.Paginate) (int64, []*telemetry.Telemetry, error) {
	var (
		deployments []*telemetry.Telemetry
	)
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"match": map[string]interface{}{
						"projectId": *iauth.GetCurrentProjectId(),
					},
				},
				map[string]interface{}{
					"match": map[string]interface{}{
						"organizationId": *iauth.GetCurrentOrganizationId(),
					},
				},
			},
		},
	}

	for _, ct := range criterias {
		switch ct.GetLogic() {
		case "oneOf":
			values := strings.Split(ct.GetValue(), ",")
			shouldQueries := make([]interface{}, len(values))
			for i, v := range values {
				shouldQueries[i] = map[string]interface{}{
					"term": map[string]interface{}{
						ct.GetKey(): v,
					},
				}
			}
			query["bool"].(map[string]interface{})["must"] = append(query["bool"].(map[string]interface{})["must"].([]interface{}), map[string]interface{}{
				"bool": map[string]interface{}{
					"should": shouldQueries,
				},
			})
		case "term", "match", "range": // Add supported logic types explicitly here
			query["bool"].(map[string]interface{})["must"] = append(query["bool"].(map[string]interface{})["must"].([]interface{}), map[string]interface{}{
				ct.GetLogic(): map[string]interface{}{
					ct.GetKey(): ct.GetValue(),
				},
			})
		}
	}

	searchBody := map[string]interface{}{
		"query": query,
		"from":  (paginate.GetPage() - 1) * paginate.GetPageSize(),
		"size":  paginate.GetPageSize(),
		"sort": []map[string]interface{}{
			{"startTime": map[string]string{"order": "asc"}},
		},
	}

	searchBodyJSON, err := json.Marshal(searchBody)
	if err != nil {
		log.Fatalf("Error marshaling search body: %s", err)
	}

	resp := ose.opensearchConnector.SearchWithCount(ctx, []string{
		commons.TelemetryIndex(ose.config.IsDevelopment())}, string(searchBodyJSON))
	if resp.Err != nil {
		ose.logger.Errorf("unable to get results from the given opensearch index %+v", resp.Err)
		return 0, nil, resp.Err
	}
	// Handle hits
	for _, hit := range resp.Hits.Hits {
		var deployment telemetry.Telemetry
		source := hit["_source"]
		sourceBytes, _ := json.Marshal(source)
		json.Unmarshal(sourceBytes, &deployment)
		deployments = append(deployments, &deployment)
	}

	// Total hits
	return int64(resp.Hits.Total), deployments, nil
}
