// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software and is not open source.
// Unauthorized copying, modification, or redistribution is strictly prohibited.

package internal_transformer_deepgram

import (
	msginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/speak/v1/websocket/interfaces"
	"github.com/rapidaai/pkg/commons"
)

// Implement the SpeakMessageCallback interface
type deepgramSpeakCallback struct {
	logger     commons.Logger
	onSpeech   func([]byte) error
	onComplete func() error
}

func NewDeepgramSpeakCallback(logger commons.Logger,
	onSpeech func([]byte) error, onComplete func() error) msginterfaces.SpeakMessageCallback {
	return &deepgramSpeakCallback{
		logger:     logger,
		onSpeech:   onSpeech,
		onComplete: onComplete,
	}
}

// Handle when the WebSocket is opened
func (d *deepgramSpeakCallback) Open(or *msginterfaces.OpenResponse) error {
	// d.logger.Debugf("Deepgram Speak WebSocket opened")
	return nil

}

// Handle metadata
func (d *deepgramSpeakCallback) Metadata(md *msginterfaces.MetadataResponse) error {
	// d.logger.Debugf("Speak Metadata received")
	return nil
}

// Handle flush event
func (d *deepgramSpeakCallback) Flush(fl *msginterfaces.FlushedResponse) error {
	// d.logger.Debugf("Speak Flush event received")
	d.onComplete()
	return nil
}

// Handle clear event
func (d *deepgramSpeakCallback) Clear(cl *msginterfaces.ClearedResponse) error {
	// d.logger.Debugf("Speak Clear event received")
	return nil
}

// Handle when the WebSocket is closed
func (d *deepgramSpeakCallback) Close(cr *msginterfaces.CloseResponse) error {
	// d.logger.Debugf("Deepgram Speak WebSocket closed")
	return nil
}

// Handle warnings
func (d *deepgramSpeakCallback) Warning(wr *msginterfaces.WarningResponse) error {
	d.logger.Warnf("Speak Warning: %+v", wr)
	return nil
}

// Handle errors
func (d *deepgramSpeakCallback) Error(er *msginterfaces.ErrorResponse) error {
	d.logger.Errorf("Speak Error: %+v", er)
	return nil
}

// Handle unhandled events
func (d *deepgramSpeakCallback) UnhandledEvent(byMsg []byte) error {
	d.logger.Warnf("Speak Unhandled Event: %s", string(byMsg))
	return nil
}

// Handle binary messages
func (d *deepgramSpeakCallback) Binary(byMsg []byte) error {
	d.onSpeech(byMsg)
	return nil
}
