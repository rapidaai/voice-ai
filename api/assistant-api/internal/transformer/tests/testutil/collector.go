// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transformer_testutil

import (
	"sync"
	"testing"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// PacketCollector is a thread-safe collector for packets emitted by transformers.
// It provides typed accessor methods and wait helpers for integration tests.
type PacketCollector struct {
	mu      sync.Mutex
	packets []internal_type.Packet
}

// NewPacketCollector creates a new empty PacketCollector.
func NewPacketCollector() *PacketCollector {
	return &PacketCollector{}
}

// OnPacket is the callback to wire into transformer constructors.
func (pc *PacketCollector) OnPacket(pkts ...internal_type.Packet) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.packets = append(pc.packets, pkts...)
	return nil
}

// GetPackets returns a snapshot of all collected packets.
func (pc *PacketCollector) GetPackets() []internal_type.Packet {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	out := make([]internal_type.Packet, len(pc.packets))
	copy(out, pc.packets)
	return out
}

// Clear removes all collected packets.
func (pc *PacketCollector) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.packets = nil
}

// AudioPackets returns only TextToSpeechAudioPacket packets.
func (pc *PacketCollector) AudioPackets() []internal_type.TextToSpeechAudioPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.TextToSpeechAudioPacket
	for _, p := range pc.packets {
		if a, ok := p.(internal_type.TextToSpeechAudioPacket); ok {
			out = append(out, a)
		}
	}
	return out
}

// EndPackets returns only TextToSpeechEndPacket packets.
func (pc *PacketCollector) EndPackets() []internal_type.TextToSpeechEndPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.TextToSpeechEndPacket
	for _, p := range pc.packets {
		if e, ok := p.(internal_type.TextToSpeechEndPacket); ok {
			out = append(out, e)
		}
	}
	return out
}

// TranscriptPackets returns only SpeechToTextPacket packets.
func (pc *PacketCollector) TranscriptPackets() []internal_type.SpeechToTextPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.SpeechToTextPacket
	for _, p := range pc.packets {
		if s, ok := p.(internal_type.SpeechToTextPacket); ok {
			out = append(out, s)
		}
	}
	return out
}

// FinalTranscripts returns SpeechToTextPacket packets where Interim == false.
func (pc *PacketCollector) FinalTranscripts() []internal_type.SpeechToTextPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.SpeechToTextPacket
	for _, p := range pc.packets {
		if s, ok := p.(internal_type.SpeechToTextPacket); ok && !s.Interim {
			out = append(out, s)
		}
	}
	return out
}

// InterimTranscripts returns SpeechToTextPacket packets where Interim == true.
func (pc *PacketCollector) InterimTranscripts() []internal_type.SpeechToTextPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.SpeechToTextPacket
	for _, p := range pc.packets {
		if s, ok := p.(internal_type.SpeechToTextPacket); ok && s.Interim {
			out = append(out, s)
		}
	}
	return out
}

// EventPackets returns only ObservabilityEventRecordPacket packets.
func (pc *PacketCollector) EventPackets() []internal_type.ObservabilityEventRecordPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.ObservabilityEventRecordPacket
	for _, p := range pc.packets {
		if e, ok := p.(internal_type.ObservabilityEventRecordPacket); ok {
			out = append(out, e)
		}
	}
	return out
}

// MetricPackets returns only ObservabilityMetricRecordPacket packets.
func (pc *PacketCollector) MetricPackets() []internal_type.ObservabilityMetricRecordPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.ObservabilityMetricRecordPacket
	for _, p := range pc.packets {
		if m, ok := p.(internal_type.ObservabilityMetricRecordPacket); ok {
			out = append(out, m)
		}
	}
	return out
}

// InterruptionDetectedPackets returns only InterruptionDetectedPacket packets.
func (pc *PacketCollector) InterruptionDetectedPackets() []internal_type.InterruptionDetectedPacket {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	var out []internal_type.InterruptionDetectedPacket
	for _, p := range pc.packets {
		if i, ok := p.(internal_type.InterruptionDetectedPacket); ok {
			out = append(out, i)
		}
	}
	return out
}

// WaitFor polls until the predicate returns true or the timeout expires.
func (pc *PacketCollector) WaitFor(t *testing.T, timeout time.Duration, desc string, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s after %v (collected %d packets)", desc, timeout, len(pc.GetPackets()))
}

// WaitForTTSEnd waits until at least one TextToSpeechEndPacket is collected.
func (pc *PacketCollector) WaitForTTSEnd(t *testing.T, timeout time.Duration) {
	t.Helper()
	pc.WaitFor(t, timeout, "TTS end packet", func() bool {
		return len(pc.EndPackets()) > 0
	})
}

// WaitForFinalTranscript waits until at least one final SpeechToTextPacket is collected.
func (pc *PacketCollector) WaitForFinalTranscript(t *testing.T, timeout time.Duration) {
	t.Helper()
	pc.WaitFor(t, timeout, "final transcript", func() bool {
		return len(pc.FinalTranscripts()) > 0
	})
}

// WaitForAudio waits until at least one TextToSpeechAudioPacket is collected.
func (pc *PacketCollector) WaitForAudio(t *testing.T, timeout time.Duration) {
	t.Helper()
	pc.WaitFor(t, timeout, "TTS audio packet", func() bool {
		return len(pc.AudioPackets()) > 0
	})
}

// WaitForAnyTranscript waits until at least one SpeechToTextPacket (interim or final) is collected.
func (pc *PacketCollector) WaitForAnyTranscript(t *testing.T, timeout time.Duration) {
	t.Helper()
	pc.WaitFor(t, timeout, "any transcript", func() bool {
		return len(pc.TranscriptPackets()) > 0
	})
}
