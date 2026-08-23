package gorm_models

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/types"
)

var (
	auditMeter             = otel.Meter("github.com/rapidaai/pkg/models/gorm/audit")
	auditWriteCounter, _   = auditMeter.Int64Counter("rapida.audit.writes")
	missingActorCounter, _ = auditMeter.Int64Counter("rapida.audit.missing_actor")
	unknownActorCounter, _ = auditMeter.Int64Counter("rapida.audit.unknown_actor")
	projectionFailure, _   = auditMeter.Int64Counter("rapida.audit.projection_failures")
)

func recordSuccessfulAuditWrite(tx *gorm.DB, operation string) {
	if tx == nil || tx.Error != nil || auditWriteCounter == nil {
		return
	}
	value, ok := tx.InstanceGet(auditActorInstanceKey)
	actor, ok := value.(types.ActorIdentity)
	if !ok {
		return
	}
	auditWriteCounter.Add(auditContext(tx), 1, metric.WithAttributes(auditAttributes(tx, operation, attribute.String("actor.type", string(actor.Type)))...))
}

func recordMissingAuditActor(tx *gorm.DB, operation string) {
	attributes := auditAttributes(tx, operation)
	slog.ErrorContext(auditContext(tx), "audit write rejected because actor identity is missing", "table", auditTable(tx), "operation", operation)
	if tx != nil && tx.Statement != nil && tx.Statement.Context != nil {
		if authentication, err := types.Authorize(tx.Statement.Context); err == nil && authentication.ActorValue != nil && authentication.ActorValue.Type == types.ActorTypeUnknown {
			if unknownActorCounter != nil {
				unknownActorCounter.Add(tx.Statement.Context, 1, metric.WithAttributes(attributes...))
			}
		}
	}
	if missingActorCounter != nil {
		missingActorCounter.Add(auditContext(tx), 1, metric.WithAttributes(attributes...))
	}
}

func recordAuditProjectionFailure(tx *gorm.DB, field string) {
	slog.ErrorContext(auditContext(tx), "audit actor projection failed", "table", auditTable(tx), "field", field)
	if projectionFailure == nil {
		return
	}
	attributes := auditAttributes(tx, "read", attribute.String("actor.field", field))
	projectionFailure.Add(auditContext(tx), 1, metric.WithAttributes(attributes...))
}

func auditContext(tx *gorm.DB) context.Context {
	if tx != nil && tx.Statement != nil && tx.Statement.Context != nil {
		return tx.Statement.Context
	}
	return context.Background()
}

func auditAttributes(tx *gorm.DB, operation string, additional ...attribute.KeyValue) []attribute.KeyValue {
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = os.Getenv("RAPIDA_SERVICE_NAME")
	}
	if serviceName == "" {
		serviceName = "unknown"
	}
	attributes := []attribute.KeyValue{
		attribute.String("service.name", serviceName),
		attribute.String("db.table", auditTable(tx)),
		attribute.String("db.operation", operation),
	}
	return append(attributes, additional...)
}

func auditTable(tx *gorm.DB) string {
	if tx != nil && tx.Statement != nil && tx.Statement.Table != "" {
		return tx.Statement.Table
	}
	return "unknown"
}
