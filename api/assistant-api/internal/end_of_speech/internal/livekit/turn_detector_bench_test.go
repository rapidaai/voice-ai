//go:build integration && cgo

package internal_livekit

import (
	"context"
	"testing"
)

func BenchmarkLiveKitNative(b *testing.B) {
	detector, reference := loadLiveKitNativeReference(b)
	for _, fixture := range reference.Cases {
		b.Run(reference.ModelType+"/"+fixture.Name, func(b *testing.B) {
			for _, stage := range []struct {
				name string
				run  func() error
			}{
				{"format", func() error {
					formatChatTemplateFromHistory(fixture.History, fixture.Text, reference.MaxHistoryTurns, reference.ModelType)
					return nil
				}},
				{"tokenize", func() error {
					liveKitBenchmarkTokenSink = liveKitBenchmarkTokenIDs(detector, fixture.Prompt)
					return nil
				}},
				{"inference", func() error {
					_, err := liveKitBenchmarkInfer(detector, fixture.InputIDs)
					return err
				}},
				{"predict", func() error {
					_, err := detector.Predict(fixture.Prompt)
					return err
				}},
				{"pipeline", func() error {
					prompt := formatChatTemplateFromHistory(fixture.History, fixture.Text, reference.MaxHistoryTurns, reference.ModelType)
					_, err := detector.Predict(prompt)
					return err
				}},
				{"production_context", func() error {
					ctx, cancel := context.WithTimeout(b.Context(), predictionTimeout)
					defer cancel()
					prompt := formatChatTemplateFromHistory(fixture.History, fixture.Text, reference.MaxHistoryTurns, reference.ModelType)
					_, err := detector.PredictContext(ctx, prompt)
					return err
				}},
			} {
				b.Run(stage.name, func(b *testing.B) {
					for range 3 {
						if err := stage.run(); err != nil {
							b.Fatal(err)
						}
					}
					b.ReportAllocs()
					for b.Loop() {
						if err := stage.run(); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}

var liveKitBenchmarkTokenSink []int64
