package internal_type

import "testing"

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
