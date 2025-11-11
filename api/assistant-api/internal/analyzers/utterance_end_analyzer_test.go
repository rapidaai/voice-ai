package internal_analyzers

// import (
// 	"context"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/rapidaai/pkg/commons"
// 	"github.com/stretchr/testify/assert"
// )

// func TestNewTextUtteranceEndAnalyzer(t *testing.T) {
// 	onAnalyze := func(ctx context.Context, seg Activity) error { return nil }

// 	tests := []struct {
// 		name    string
// 		mode    string
// 		wantErr bool
// 	}{
// 		{"Valid relaxed mode", "relaxed", false},
// 		{"Valid normal mode", "normal", false},
// 		{"Valid fast mode", "fast", false},
// 		{"Invalid mode", "invalid", true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			analyzer, err := NewTextUtteranceEndAnalyzer(commons.NewApplicationLogger(), &TextAnalyzerOptions{
// 				OnAnalyze: onAnalyze,
// 				AnalyzeOptions: AnalyzeOptions{opts: map[string]interface{}{
// 					"speaker_mode": tt.mode,
// 				}},
// 			})
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 				assert.Nil(t, analyzer)
// 			} else {
// 				assert.NoError(t, err)
// 				assert.NotNil(t, analyzer)
// 			}
// 		})
// 	}
// }

// func TestUtteranceEndAnalyzer_Name(t *testing.T) {

// 	onAnalyze := func(ctx context.Context, seg Activity) error { return nil }
// 	analyzer, _ := NewTextUtteranceEndAnalyzer(commons.NewApplicationLogger(), &TextAnalyzerOptions{
// 		OnAnalyze: onAnalyze,
// 		AnalyzeOptions: AnalyzeOptions{opts: map[string]interface{}{
// 			"speaker_mode": "normal",
// 		}},
// 	})

// 	assert.Equal(t, "utterance-end-analyzer", analyzer.Name())
// }

// func TestUtteranceEndAnalyzer_Analyze(t *testing.T) {
// 	logger := commons.NewApplicationLogger()
// 	var mu sync.Mutex
// 	var segments []Activity
// 	onAnalyze := func(ctx context.Context, seg Activity) error {
// 		mu.Lock()
// 		defer mu.Unlock()
// 		segments = append(segments, seg)
// 		return nil
// 	}

// 	analyzer, _ := NewTextUtteranceEndAnalyzer(logger, &TextAnalyzerOptions{
// 		OnAnalyze: onAnalyze,
// 		AnalyzeOptions: AnalyzeOptions{opts: map[string]interface{}{
// 			"speaker_mode": "normal",
// 		}},
// 	})
// 	defer analyzer.Close()

// 	ctx := context.Background()

// 	// Test soft silence
// 	analyzer.Analyze(ctx, "Hello")
// 	time.Sleep(1100 * time.Millisecond) // Just over soft silence threshold

// 	// Test hard silence
// 	analyzer.Analyze(ctx, "World")
// 	time.Sleep(2100 * time.Millisecond) // Just over hard silence threshold

// 	// Allow some time for the analyzer to process
// 	time.Sleep(100 * time.Millisecond)

// 	mu.Lock()
// 	defer mu.Unlock()
// 	assert.Len(t, segments, 2)
// 	// assert.InDelta(t, 1.0, segments[0].GetDuration(), 0.1)
// 	// assert.InDelta(t, 2.0, segments[1].GetDuration(), 0.1)
// }
