package internal_type

import "testing"

func TestInterruptionDecisionExpiredPacket(t *testing.T) {
	packet := InterruptionDecisionExpiredPacket{ContextID: "context-1", Sequence: 7}

	if got := packet.ContextId(); got != "context-1" {
		t.Fatalf("expected context ID %q, got %q", "context-1", got)
	}
	if got := packet.PacketName(); got != PacketNameInterruptionDecisionExpired {
		t.Fatalf("expected packet name %q, got %q", PacketNameInterruptionDecisionExpired, got)
	}
	if packet.Sequence != 7 {
		t.Fatalf("expected sequence 7, got %d", packet.Sequence)
	}
}

func TestTurnChangePacket_InterruptionSequence(t *testing.T) {
	packet := TurnChangePacket{InterruptionSequence: 11}

	if packet.InterruptionSequence != 11 {
		t.Fatalf("expected interruption sequence 11, got %d", packet.InterruptionSequence)
	}
}

func TestSpeechToTextPacket_GetConcat(t *testing.T) {
	tests := []struct {
		name     string
		packet   SpeechToTextPacket
		expected string
	}{
		{
			name:     "nil concat defaults to space",
			packet:   SpeechToTextPacket{},
			expected: " ",
		},
		{
			name: "empty concat pointer",
			packet: SpeechToTextPacket{
				Concat: func() *string {
					value := ""
					return &value
				}(),
			},
			expected: "",
		},
		{
			name: "non empty concat pointer",
			packet: SpeechToTextPacket{
				Concat: func() *string {
					value := "-"
					return &value
				}(),
			},
			expected: "-",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.packet.GetConcat(); got != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}

func TestObservabilityMetricRecordPacket_IsAsync(t *testing.T) {
	var packet any = ObservabilityMetricRecordPacket{}
	asyncPacket, ok := packet.(AsyncPacket)
	if !ok {
		t.Fatal("expected ObservabilityMetricRecordPacket to implement AsyncPacket")
	}
	if !asyncPacket.IsAsync() {
		t.Fatal("expected ObservabilityMetricRecordPacket IsAsync to return true")
	}
}

func TestSpeechToTextAudioPacket_IsSynchronous(t *testing.T) {
	var packet any = SpeechToTextAudioPacket{}
	if _, ok := packet.(AsyncPacket); ok {
		t.Fatal("expected SpeechToTextAudioPacket to preserve ingress ordering")
	}
}
