package smallest_internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextToSpeechInputContextJSON(t *testing.T) {
	for _, continuation := range []bool{true, false} {
		input := TextToSpeechInput{ContextID: "turn-123", Continue: continuation, Flush: !continuation}
		encoded, err := json.Marshal(input)
		require.NoError(t, err)
		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal(encoded, &payload))
		require.Equal(t, "turn-123", payload["context_id"])
		require.Equal(t, continuation, payload["continue"])
		require.NotContains(t, payload, "session_id")
		if !continuation {
			require.Equal(t, true, payload["flush"])
		}
	}
	encoded, err := json.Marshal(TextToSpeechInput{})
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.NotContains(t, payload, "context_id")
	require.NotContains(t, payload, "session_id")
}

func TestTextToSpeechOutputSessionJSONUnchanged(t *testing.T) {
	var response TextToSpeechOutput
	require.NoError(t, json.Unmarshal([]byte(`{"session_id":"provider-session","request_id":"request-1","status":"chunk","data":{"audio":"AQI="}}`), &response))
	require.Equal(t, "provider-session", response.SessionID)
	require.Equal(t, "request-1", response.RequestID)
	require.Equal(t, "AQI=", response.Data.Audio)
}
