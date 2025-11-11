package internal_assistant_telemetry_exporters

import (
	"context"

	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	assistant_api "github.com/rapidaai/protos"
)

type postgresExporter struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewPostgresAssistantTraceExporter(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
) internal_telemetry.VoiceAgentTraceExporter {
	return &postgresExporter{
		logger:   logger,
		postgres: postgres,
	}
}

// Persist implements internal_adapter_telemetry.Exporter.
func (pe *postgresExporter) Export(ctx context.Context,
	iauth types.SimplePrinciple,
	options internal_telemetry.ExportOption,
	stages []*internal_telemetry.Telemetry) error {
	psD := make([]*AssistantConversationPostgresTelemetry, 0)
	switch o := options.(type) {
	case *internal_telemetry.VoiceAgentExportOption:
		for _, doc := range stages {
			psD = append(psD, ToPostgresTelemetry(doc,
				o.AssistantId,
				o.AssistantProviderModelId,
				o.AssistantConversationId,
				*iauth.GetCurrentProjectId(),
				*iauth.GetCurrentProjectId(),
			))

		}
		err := pe.postgres.DB(ctx).Create(psD)
		if err != nil {
			pe.logger.Errorf("unable to export telemetry with error %+v", err)
		}
	}

	return nil
}

func (ose *postgresExporter) Get(
	ctx context.Context,
	iauth types.SimplePrinciple,
	criterias []*assistant_api.Criteria,
	paginate *assistant_api.Paginate) (int64, []*internal_telemetry.Telemetry, error) {
	return 0, nil, nil
}
