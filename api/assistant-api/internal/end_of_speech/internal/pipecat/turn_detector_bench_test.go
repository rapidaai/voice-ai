//go:build integration && cgo

package internal_pipecat

import "testing"

func BenchmarkPipecatNative(b *testing.B) {
	detector, cases := loadPipecatNativeReference(b)
	for _, fixture := range cases {
		b.Run(fixture.Name, func(b *testing.B) {
			b.Run("features", func(b *testing.B) {
				for range 3 {
					detector.features.extractInto(fixture.audio, detector.scratch.output[:], detector.scratch)
				}
				b.ReportAllocs()
				for b.Loop() {
					detector.features.extractInto(fixture.audio, detector.scratch.output[:], detector.scratch)
				}
			})
			b.Run("inference", func(b *testing.B) {
				for range 3 {
					if _, err := detector.infer(fixture.expectedFeatures); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportAllocs()
				for b.Loop() {
					if _, err := detector.infer(fixture.expectedFeatures); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("predict", func(b *testing.B) {
				for range 3 {
					if _, err := detector.Predict(fixture.audio); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportAllocs()
				for b.Loop() {
					if _, err := detector.Predict(fixture.audio); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
