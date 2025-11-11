package internal_telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Telemetry struct {
	StageName  string            `json:"stageName"`
	StartTime  time.Time         `json:"startTime"`
	EndTime    time.Time         `json:"endTime"`
	Duration   time.Duration     `json:"duration"`
	Attributes map[string]string `json:"attributes"`
	SpanID     string            `json:"spanID"`
	ParentID   string            `json:"parentID"`
}

func (t *Telemetry) ToProto() *lexatic_backend.Telemetry {
	var startTimeProto *timestamppb.Timestamp
	if !t.StartTime.IsZero() {
		startTimeProto = timestamppb.New(t.StartTime)
	}

	var endTimeProto *timestamppb.Timestamp
	if !t.EndTime.IsZero() {
		endTimeProto = timestamppb.New(t.EndTime)
	}
	return &lexatic_backend.Telemetry{
		StageName:  t.StageName,
		StartTime:  startTimeProto,
		EndTime:    endTimeProto,
		Duration:   uint64(t.Duration), // protobuf stores duration as uint64 ms
		Attributes: t.Attributes,
		SpanID:     t.SpanID,
		ParentID:   t.ParentID,
	}
}

type ExportOption interface{}
type VoiceAgentExportOption struct {
	ExportOption
	AssistantId              uint64
	AssistantProviderModelId uint64
	AssistantConversationId  uint64
}

type JSONValue map[string]interface{}

func (j JSONValue) String() string {
	jsonBytes, err := json.Marshal(j)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

type StringValue string

func (s StringValue) String() string { return string(s) }

// IntValue implements the Value interface for integers.
type IntValue int

func (i IntValue) String() string { return fmt.Sprintf("%d", i) }

// FloatValue implements the Value interface for floats.
type FloatValue float64

func (f FloatValue) String() string { return fmt.Sprintf("%f", f) }

// BoolValue implements the Value interface for booleans.
type BoolValue bool

func (b BoolValue) String() string { return fmt.Sprintf("%t", b) }

// --- Tracer Implementation ---

type contextKey string

const SpanKey contextKey = "current_span_id"

type Value interface {
	String() string
}

type KV struct {
	K string
	V Value
}

func MessageKV(messageID string) KV {
	return KV{
		K: "messageId",
		V: StringValue(messageID),
	}
}

type Tracer[event any] interface {
	// Starts a span with the provided context, stage, and attributes.
	// Conversation IDs should be stored in context via WithConversationIDs.
	StartSpan(
		ctx context.Context,
		stage event,
		attributes ...KV,
	) (context.Context, Tracer[event], error)

	// Adds attributes to an existing span.
	AddAttributes(
		ctx context.Context,
		attributes ...KV,
	) error

	// Ends a span with the provided context, stage, and attributes.
	EndSpan(
		ctx context.Context,
		stage event,
		attributes ...KV,
	) error

	// Cancels any in-flight tracing operations.
	Cancel(ctx context.Context) error

	// export all the span
	Export(ctx context.Context, iauth types.SimplePrinciple, o ExportOption, e ...TraceExporter) error
}

type TraceExporter interface {
	// Exports a slice of stages to an external storage system.
	Export(ctx context.Context, iauth types.SimplePrinciple, opts ExportOption, stages []*Telemetry) error
}

type VoiceAgentTraceExporter interface {
	TraceExporter
	Get(
		ctx context.Context,
		iauth types.SimplePrinciple,
		criterias []*assistant_api.Criteria,
		paginate *assistant_api.Paginate,
	) (int64, []*Telemetry, error)
}
type VoiceAgentTracer interface {
	Tracer[utils.RapidaStage]
}
