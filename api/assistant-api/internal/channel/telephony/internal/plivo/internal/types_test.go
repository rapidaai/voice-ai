// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_plivo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Plivo sends sequenceNumber and media.chunk as JSON numbers (not strings). This
// guards against re-typing them as string: that makes every start/media event
// fail to unmarshal and get dropped — no audio reaches the pipeline and the
// streamId/CallUUID are never captured.
func TestPlivoMediaEvent_UnmarshalsNumericWireFormat(t *testing.T) {
	startJSON := []byte(`{"event":"start","sequenceNumber":1,"streamId":"MZ123",` +
		`"start":{"callId":"abc-123","streamId":"MZ123",` +
		`"mediaFormat":{"encoding":"audio/x-mulaw","sampleRate":8000}}}`)
	var start PlivoMediaEvent
	require.NoError(t, json.Unmarshal(startJSON, &start))
	assert.Equal(t, EventTypeStart, start.Event)
	assert.Equal(t, "MZ123", start.StreamID)
	require.NotNil(t, start.Start)
	assert.Equal(t, "abc-123", start.Start.CallID)
	assert.Equal(t, "MZ123", start.Start.StreamID)

	mediaJSON := []byte(`{"event":"media","sequenceNumber":42,"streamId":"MZ123",` +
		`"media":{"track":"inbound","chunk":41,"timestamp":"1705312200000","payload":"AQID"}}`)
	var media PlivoMediaEvent
	require.NoError(t, json.Unmarshal(mediaJSON, &media))
	assert.Equal(t, EventTypeMedia, media.Event)
	require.NotNil(t, media.Media)
	assert.Equal(t, "AQID", media.Media.Payload)
}
