package internal_entity

import (
	"reflect"
	"testing"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

func TestProductUsageUsesProjectScopedAuditModels(t *testing.T) {
	entityType := reflect.TypeOf(ProductUsage{})
	for _, field := range []struct {
		name   string
		typeOf reflect.Type
	}{
		{name: "Audited", typeOf: reflect.TypeOf(gorm_model.Audited{})},
		{name: "Mutable", typeOf: reflect.TypeOf(gorm_model.Mutable{})},
		{name: "Organizational", typeOf: reflect.TypeOf(gorm_model.Organizational{})},
	} {
		actual, ok := entityType.FieldByName(field.name)
		if !ok || actual.Type != field.typeOf || !actual.Anonymous {
			t.Fatalf("embedded field %s=%v, want %v", field.name, actual.Type, field.typeOf)
		}
	}
	if table := (ProductUsage{}).TableName(); table != "product_usages" {
		t.Fatalf("TableName()=%q, want product_usages", table)
	}
}
