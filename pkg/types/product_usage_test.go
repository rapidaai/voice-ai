package types

import (
	"errors"
	"testing"
)

func TestValidateProductUsage(t *testing.T) {
	for _, usageType := range []ProductUsageType{
		ProductUsageSTTDuration,
		ProductUsageTTSDuration,
		ProductUsageVADDuration,
		ProductUsageEOSDuration,
		ProductUsageDenoiseDuration,
		ProductUsageLLMDuration,
	} {
		t.Run(string(usageType), func(t *testing.T) {
			unit, ok := ProductUsageUnitFor(string(usageType))
			if !ok || unit != string(ProductUsageUnitNanosecond) {
				t.Fatalf("ProductUsageUnitFor() = %q, %v", unit, ok)
			}
			if err := ValidateProductUsage(string(usageType), unit); err != nil {
				t.Fatalf("ValidateProductUsage() error = %v", err)
			}
		})
	}
}

func TestValidateProductUsageRejectsUnknownTypeAndUnit(t *testing.T) {
	for _, test := range []struct {
		usageType string
		unit      string
	}{
		{usageType: "unknown", unit: string(ProductUsageUnitNanosecond)},
		{usageType: string(ProductUsageSTTDuration), unit: "second"},
	} {
		if err := ValidateProductUsage(test.usageType, test.unit); !errors.Is(err, ErrInvalidProductUsage) {
			t.Fatalf("ValidateProductUsage(%q, %q) error = %v", test.usageType, test.unit, err)
		}
	}
}
