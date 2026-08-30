package internal_entity

import (
	"reflect"
	"testing"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

func TestProductUsageUsesAuditedGeneratedID(t *testing.T) {
	entityType := reflect.TypeOf(ProductUsage{})
	audited, ok := entityType.FieldByName("Audited")
	if !ok {
		t.Fatal("ProductUsage.Audited is missing")
	}
	if !audited.Anonymous || audited.Type != reflect.TypeOf(gorm_model.Audited{}) {
		t.Fatalf("ProductUsage.Audited = %#v, want embedded Audited", audited)
	}
	if _, ok = entityType.FieldByName("UsageID"); ok {
		t.Fatal("ProductUsage must not contain UsageID")
	}
	if table := (ProductUsage{}).TableName(); table != "product_usages" {
		t.Fatalf("TableName()=%q, want product_usages", table)
	}
}
