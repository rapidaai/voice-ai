package internal_assistant_telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

// inMemoryTracer is a thread-safe, in-memory implementation of MessageTracer.
type inMemoryTracer struct {
	logger   commons.Logger
	mu       sync.RWMutex
	stages   map[string]*internal_telemetry.Telemetry
	exporter []internal_telemetry.TraceExporter
}

// NewInMemoryMessageTracer creates a new tracer that exports spans in real-time.
func NewInMemoryTracer(logger commons.Logger, exporter ...internal_telemetry.TraceExporter) internal_telemetry.VoiceAgentTracer {
	return &inMemoryTracer{
		logger:   logger,
		stages:   make(map[string]*internal_telemetry.Telemetry),
		exporter: exporter,
	}
}

// StartSpan begins a new tracing span.
func (t *inMemoryTracer) StartSpan(
	ctx context.Context,
	stage utils.RapidaStage,
	attributes ...internal_telemetry.KV) (context.Context, internal_telemetry.Tracer[utils.RapidaStage], error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	spanID := uuid.New().String()
	var parentID string
	if existingSpan, ok := ctx.Value(internal_telemetry.SpanKey).(string); ok {
		parentID = existingSpan
	}
	newStage := &internal_telemetry.Telemetry{
		StageName:  string(stage),
		StartTime:  time.Now().UTC(),
		Attributes: make(map[string]string),
		SpanID:     spanID,
		ParentID:   parentID,
	}

	for _, attr := range attributes {
		newStage.Attributes[attr.K] = attr.V.String()
	}

	t.stages[spanID] = newStage

	// Return a new context with the new span ID.
	newCtx := context.WithValue(ctx, internal_telemetry.SpanKey, spanID)

	// Return the same tracer instance as it's stateful.
	return newCtx, t, nil
}

// AddAttributes adds key-value attributes to the current span.
func (t *inMemoryTracer) AddAttributes(ctx context.Context, attributes ...internal_telemetry.KV) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	spanID, ok := t.getCurrentSpanID(ctx)
	if !ok {
		return fmt.Errorf("no active span found to add attributes")
	}

	stage, exists := t.stages[spanID]
	if !exists {
		return fmt.Errorf("span with ID %s not found", spanID)
	}

	for _, attr := range attributes {
		stage.Attributes[attr.K] = attr.V.String() // Overwrite without validation
	}

	return nil
}

// Cancel clears all tracing data from the tracer.
func (t *inMemoryTracer) Cancel(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stages = make(map[string]*internal_telemetry.Telemetry) // Replace map with a fresh empty instance.
	return nil
}
func (t *inMemoryTracer) EndSpan(ctx context.Context, stage utils.RapidaStage, attributes ...internal_telemetry.KV) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	spanID, ok := t.getCurrentSpanID(ctx)
	if !ok {
		return fmt.Errorf("no active span to end")
	}

	currentStage, exists := t.stages[spanID]
	if !exists {
		return fmt.Errorf("span with ID %s not found", spanID)
	}

	// Double-check that we are ending the correct stage.
	if currentStage.StageName != string(stage) {
		return fmt.Errorf("mismatched end span: expected %s, got %s", currentStage.StageName, stage)
	}

	currentStage.EndTime = time.Now().UTC()
	currentStage.Duration = currentStage.EndTime.Sub(currentStage.StartTime)

	for _, attr := range attributes {
		currentStage.Attributes[attr.K] = attr.V.String()
	}

	// Keep the ended stage data in `t.stages` without clearing others.
	t.stages[spanID] = currentStage
	return nil
}

func (t *inMemoryTracer) Export(ctx context.Context,
	iauth types.SimplePrinciple,
	opts internal_telemetry.ExportOption,
	exporters ...internal_telemetry.TraceExporter) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Export all spans in `t.stages`.
	builtStages := make([]*internal_telemetry.Telemetry, 0, len(t.stages))
	for _, stage := range t.stages {
		stageCopy := *stage
		stageCopy.Attributes = make(map[string]string, len(stage.Attributes)) // Deep copy attributes
		for k, v := range stage.Attributes {
			stageCopy.Attributes[k] = v
		}
		builtStages = append(builtStages, &stageCopy)
	}

	for _, v := range t.exporter {
		v.Export(ctx, iauth, opts, builtStages)
	}
	for _, v := range exporters {
		v.Export(ctx, iauth, opts, builtStages)
	}
	return nil
}

// getCurrentSpanID safely retrieves the most recent active span ID.
func (t *inMemoryTracer) getCurrentSpanID(ctx context.Context) (string, bool) {
	if existingSpan, ok := ctx.Value(internal_telemetry.SpanKey).(string); ok {
		return existingSpan, true
	}
	return "", false
}
