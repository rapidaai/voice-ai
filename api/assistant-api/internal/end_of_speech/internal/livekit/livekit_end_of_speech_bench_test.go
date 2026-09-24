package internal_livekit

import (
	"path/filepath"
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

func BenchmarkTokenizerEncode(b *testing.B) {
	for _, variant := range []string{"english", "multilingual"} {
		b.Run(variant, func(b *testing.B) {
			path := filepath.Join("testdata", "tokenizer", "tokenizer.json")
			if variant == "multilingual" {
				path = filepath.Join("testdata", "tokenizer", "multilingual", "tokenizer.json")
			}
			tokenizer, err := newTokenizer(path)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if len(tokenizer.Encode("<|im_start|>user\nI'm here 123<|im_end|>")) == 0 {
					b.Fatal("expected token IDs")
				}
			}
		})
	}
}

func BenchmarkPredictTurnCompletion(b *testing.B) {
	endOfSpeech := &livekitEndOfSpeech{
		predictor: testPredictor{predict: func(string) (float64, error) { return 0.9, nil }},
		history: []chatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "How can I help?"},
		},
		maxHistory: defaultMaxHistory,
		modelType:  defaultModelType,
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := endOfSpeech.predictEOU(b.Context(), "Thank you. Goodbye."); err != nil {
			b.Fatal(err)
		}
	}
}
