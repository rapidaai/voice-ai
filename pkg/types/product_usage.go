package types

import (
	"errors"
	"fmt"
)

type ProductUsageType string

type ProductUsageUnit string

const (
	ProductUsageSTTDuration     ProductUsageType = "stt_duration"
	ProductUsageTTSDuration     ProductUsageType = "tts_duration"
	ProductUsageVADDuration     ProductUsageType = "vad_duration"
	ProductUsageEOSDuration     ProductUsageType = "eos_duration"
	ProductUsageDenoiseDuration ProductUsageType = "denoise_duration"
	ProductUsageLLMDuration     ProductUsageType = "llm_duration"

	ProductUsageUnitNanosecond ProductUsageUnit = "nanosecond"
)

var ErrInvalidProductUsage = errors.New("invalid product usage")

var productUsageUnits = map[ProductUsageType]ProductUsageUnit{
	ProductUsageSTTDuration:     ProductUsageUnitNanosecond,
	ProductUsageTTSDuration:     ProductUsageUnitNanosecond,
	ProductUsageVADDuration:     ProductUsageUnitNanosecond,
	ProductUsageEOSDuration:     ProductUsageUnitNanosecond,
	ProductUsageDenoiseDuration: ProductUsageUnitNanosecond,
	ProductUsageLLMDuration:     ProductUsageUnitNanosecond,
}

func ProductUsageUnitFor(usageType string) (string, bool) {
	unit, ok := productUsageUnits[ProductUsageType(usageType)]
	return string(unit), ok
}

func ValidateProductUsage(usageType, unit string) error {
	expectedUnit, ok := ProductUsageUnitFor(usageType)
	if !ok {
		return fmt.Errorf("%w: unsupported usage type %q", ErrInvalidProductUsage, usageType)
	}
	if unit != expectedUnit {
		return fmt.Errorf("%w: usage type %q requires unit %q", ErrInvalidProductUsage, usageType, expectedUnit)
	}
	return nil
}
