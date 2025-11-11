// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_deepgram

import (
	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/websocket/interfaces"
	"github.com/rapidaai/pkg/commons"
)

// Implement the LiveMessageCallback interface
type deepgramSttCallback struct {
	logger       commons.Logger
	onTranscript func(
		transcript string,
		confidence float64,
		language string,
		isFinal bool,
	) error
}

func NewDeepgramSttCallback(
	logger commons.Logger,
	onTranscript func(
		transcript string,
		confidence float64,
		language string,
		isFinal bool) error,
) msginterfaces.LiveMessageCallback {
	return &deepgramSttCallback{
		logger:       logger,
		onTranscript: onTranscript,
	}

}

// Handle when the WebSocket is opened
func (d *deepgramSttCallback) Open(or *msginterfaces.OpenResponse) error {
	return nil
}

// Handle incoming transcription messages from Deepgram
func (d *deepgramSttCallback) Message(mr *msginterfaces.MessageResponse) error {
	for _, alternative := range mr.Channel.Alternatives {
		if alternative.Transcript != "" {
			return d.onTranscript(
				alternative.Transcript,
				alternative.Confidence,
				d.GetMostUsedLanguage(alternative.Languages),
				mr.IsFinal,
			)
		}
	}
	return nil
}

// Handle utterance end event - this signals the end of a sentence
func (d *deepgramSttCallback) UtteranceEnd(ur *msginterfaces.UtteranceEndResponse) error {
	return nil
}

// Handle metadata (optional, can be left empty)
func (d *deepgramSttCallback) Metadata(md *msginterfaces.MetadataResponse) error {
	return nil
}

// Handle speech started event
func (d *deepgramSttCallback) SpeechStarted(ssr *msginterfaces.SpeechStartedResponse) error {
	return nil
}

// Handle when the WebSocket is closed
func (d *deepgramSttCallback) Close(cr *msginterfaces.CloseResponse) error {
	// d.logger.Debugf("Deepgram WebSocket closed")
	return nil
}

// Handle errors from Deepgram
func (d *deepgramSttCallback) Error(er *msginterfaces.ErrorResponse) error {
	d.logger.Errorf("Error %+v", er)
	return nil
}

// Handle unhandled events (optional, can be left empty)
func (d *deepgramSttCallback) UnhandledEvent(byData []byte) error {
	d.logger.Errorf("UnhandledEvent %+v", byData)
	return nil
}

func (d *deepgramSttCallback) GetMostUsedLanguage(languages []string) string {
	if len(languages) == 0 {
		return "en"
	}

	languageCount := make(map[string]int)
	for _, lang := range languages {
		languageCount[lang]++
	}

	mostUsedLang := ""
	maxCount := 0
	for lang, count := range languageCount {
		if count > maxCount {
			maxCount = count
			mostUsedLang = lang
		}
	}
	return mostUsedLang
}
