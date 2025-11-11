package internal_assistant_telemetry_exporters

import (
	"context"

	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

// loggingTraceExporter implements TraceExporter by logging the traces to the console.
type loggingTraceExporter struct {
	logger commons.Logger
}

// NewLoggingTraceExporter creates a new console-based exporter.
func NewLoggingAssistantTraceExporter(logger commons.Logger) internal_telemetry.TraceExporter {
	return &loggingTraceExporter{
		logger: logger,
	}
}

// Export prints the details of each stage to the standard log.
func (e *loggingTraceExporter) Export(ctx context.Context, auth types.SimplePrinciple, opts internal_telemetry.ExportOption, stages []*internal_telemetry.Telemetry) error {
	if len(stages) == 0 {
		return nil
	}
	e.logger.Infof("--- Exporting %d Trace Stage(s) ---", len(stages))
	for i, s := range stages {
		e.logger.Infof("Stage %d: [%s]", i+1, s.StageName)
		e.logger.Infof("  SpanID:   %s", s.SpanID)
		e.logger.Infof("  ParentID: %s", s.ParentID)
		e.logger.Infof("  Duration: %s", s.Duration)
		e.logger.Infof("  Start: %s", s.StartTime)
		e.logger.Infof("  End: %s", s.EndTime)
		e.logger.Infof("  Attributes:")
		for k, v := range s.Attributes {
			e.logger.Infof("    - %s: %s", k, v)
		}
	}
	e.logger.Infof("--- Export Complete ---")
	return nil
}
