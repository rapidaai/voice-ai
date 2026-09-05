// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"time"

	"github.com/rapidaai/pkg/types"
)

const (
	UsageConversationSTTDuration     = string(types.ProductUsageSTTDuration)
	UsageConversationTTSDuration     = string(types.ProductUsageTTSDuration)
	UsageConversationVADDuration     = string(types.ProductUsageVADDuration)
	UsageConversationEOSDuration     = string(types.ProductUsageEOSDuration)
	UsageConversationDenoiseDuration = string(types.ProductUsageDenoiseDuration)
	UsageConversationLLMDuration     = string(types.ProductUsageLLMDuration)
)

func NewSTTDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationSTTDuration), provider, duration)
	record.Attributes = attr
	return record
}

func NewTTSDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationTTSDuration), provider, duration)
	record.Attributes = attr
	return record
}

func NewVADDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationVADDuration), provider, duration)
	record.Attributes = attr
	return record
}

func NewEOSDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationEOSDuration), provider, duration)
	record.Attributes = attr
	return record
}

func NewDenoiseDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationDenoiseDuration), provider, duration)
	record.Attributes = attr
	return record
}

func NewLLMDurationUsageRecord(provider string, duration time.Duration, attr Attributes) RecordUsage {
	record := NewUsageRecord(ComponentName(UsageConversationLLMDuration), provider, duration)
	record.Attributes = attr
	return record
}
