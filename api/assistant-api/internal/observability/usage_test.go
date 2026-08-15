// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"testing"
	"time"
)

func TestNewUsageRecord(t *testing.T) {
	record := NewUsageRecord(ComponentAgent, "openai", 123*time.Millisecond)

	if record.Component != ComponentAgent {
		t.Fatalf("expected component %q, got %q", ComponentAgent, record.Component)
	}
	if record.Provider != "openai" {
		t.Fatalf("expected provider %q, got %q", "openai", record.Provider)
	}
	if record.Duration != 123*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 123*time.Millisecond, record.Duration)
	}
	if record.Attributes != nil {
		t.Fatalf("expected nil attributes, got %v", record.Attributes)
	}
}

func TestNewSTTDurationUsageRecord(t *testing.T) {
	record := NewSTTDurationUsageRecord("deepgram-speech-to-text", 456*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationSTTDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationSTTDuration, record.Component)
	}
	if record.Provider != "deepgram-speech-to-text" {
		t.Fatalf("expected provider %q, got %q", "deepgram-speech-to-text", record.Provider)
	}
	if record.Duration != 456*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 456*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}

func TestNewTTSDurationUsageRecord(t *testing.T) {
	record := NewTTSDurationUsageRecord("deepgram-text-to-speech", 789*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationTTSDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationTTSDuration, record.Component)
	}
	if record.Provider != "deepgram-text-to-speech" {
		t.Fatalf("expected provider %q, got %q", "deepgram-text-to-speech", record.Provider)
	}
	if record.Duration != 789*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 789*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}

func TestNewVADDurationUsageRecord(t *testing.T) {
	record := NewVADDurationUsageRecord("silero_vad", 321*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationVADDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationVADDuration, record.Component)
	}
	if record.Provider != "silero_vad" {
		t.Fatalf("expected provider %q, got %q", "silero_vad", record.Provider)
	}
	if record.Duration != 321*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 321*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}

func TestNewEOSDurationUsageRecord(t *testing.T) {
	record := NewEOSDurationUsageRecord("silenceBasedEndOfSpeech", 654*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationEOSDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationEOSDuration, record.Component)
	}
	if record.Provider != "silenceBasedEndOfSpeech" {
		t.Fatalf("expected provider %q, got %q", "silenceBasedEndOfSpeech", record.Provider)
	}
	if record.Duration != 654*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 654*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}

func TestNewDenoiseDurationUsageRecord(t *testing.T) {
	record := NewDenoiseDurationUsageRecord("rn_noise", 987*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationDenoiseDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationDenoiseDuration, record.Component)
	}
	if record.Provider != "rn_noise" {
		t.Fatalf("expected provider %q, got %q", "rn_noise", record.Provider)
	}
	if record.Duration != 987*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 987*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}

func TestNewLLMDurationUsageRecord(t *testing.T) {
	record := NewLLMDurationUsageRecord("openai", 123*time.Millisecond, Attributes{"context_id": "ctx-1"})

	if record.Component != ComponentName(UsageConversationLLMDuration) {
		t.Fatalf("expected component %q, got %q", UsageConversationLLMDuration, record.Component)
	}
	if record.Provider != "openai" {
		t.Fatalf("expected provider %q, got %q", "openai", record.Provider)
	}
	if record.Duration != 123*time.Millisecond {
		t.Fatalf("expected duration %v, got %v", 123*time.Millisecond, record.Duration)
	}
	if record.Attributes["context_id"] != "ctx-1" {
		t.Fatalf("expected context_id %q, got %q", "ctx-1", record.Attributes["context_id"])
	}
}
