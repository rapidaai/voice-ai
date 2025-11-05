package internal_analyzer_factories

import (
	"errors"

	internal_analyzers "github.com/rapidaai/api/assistant-api/internal/analyzers"
	internal_voices "github.com/rapidaai/api/assistant-api/internal/voices"
	"github.com/rapidaai/pkg/commons"
)

type AnalyzerIdentifier string

const (
	UtteranceStartAnalyzer      AnalyzerIdentifier = "utterance-start-analyzer"
	CerebrasEndOfSpeechAnalyzer AnalyzerIdentifier = "cerebras-end-of-speech-analyzer"
	UtteranceEndAnalyzer        AnalyzerIdentifier = "utterance-end-analyzer"
)

func GetVoiceAnalyzer(aa AnalyzerIdentifier, logger commons.Logger, audioConfig *internal_voices.AudioConfig, opts *internal_analyzers.VoiceAnalyzerOptions) (internal_analyzers.AudioAnalyzer, error) {
	switch aa {
	default:
		return internal_analyzers.NewSileroVadAnalyzer(logger, audioConfig, opts)
	}
}

func GetTextAnalyzer(aa AnalyzerIdentifier, logger commons.Logger, opts *internal_analyzers.TextAnalyzerOptions) (internal_analyzers.TextAnalyzer, error) {
	switch aa {
	case UtteranceEndAnalyzer:
		return internal_analyzers.NewTextUtteranceEndAnalyzer(logger, opts)
	default:
		return nil, errors.New("illegal analyzer")
	}
}
