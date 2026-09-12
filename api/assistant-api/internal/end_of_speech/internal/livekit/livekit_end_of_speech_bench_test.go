package internal_livekit

import (
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func BenchmarkLiveKitExecuteAudio(b *testing.B) {
	endOfSpeech := &livekitEndOfSpeech{}
	packet := internal_type.EndOfSpeechAudioPacket{Audio: make([]byte, 640)}
	b.ReportAllocs()
	for b.Loop() {
		if err := endOfSpeech.Execute(b.Context(), packet); err != nil {
			b.Fatal(err)
		}
	}
}
