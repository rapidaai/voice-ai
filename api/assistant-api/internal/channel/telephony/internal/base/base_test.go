// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telephony_base

import (
	"context"
	"testing"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	scope   observability.Scope
	records []observability.Record
}

func (r *recordingObserver) Record(_ context.Context, scope observability.Scope, records ...observability.Record) error {
	r.scope = scope
	r.records = append(r.records, records...)
	return nil
}

func (r *recordingObserver) AddCollectors(...observability.Collector) error {
	return nil
}

func (r *recordingObserver) Close(context.Context) error {
	return nil
}

func newTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(
		commons.Level("error"),
		commons.Name("base-streamer-test"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return l
}

func metadataString(t *testing.T, md map[string]interface{}, key string) string {
	t.Helper()
	v, ok := md[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	require.Truef(t, ok, "metadata key %q is not a string: %T", key, v)
	return s
}

func TestCreateConnectionRequest_EmitsAllClientKeys(t *testing.T) {
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 42,
		Direction:      "outbound",
		Provider:       "sip",
		CallerNumber:   "15551234567",
		FromNumber:     "15557654321",
		ChannelUUID:    "live-call-id",
		ContextID:      "ctx-uuid-123",
	}
	base := New(newTestLogger(t), cc, nil, nil)

	req := base.CreateConnectionRequest()
	require.NotNil(t, req)

	md, err := utils.AnyMapToInterfaceMap(req.GetMetadata())
	require.NoError(t, err)

	require.Equal(t, "outbound", metadataString(t, md, "client.direction"))
	require.Equal(t, "sip", metadataString(t, md, "client.channel"))
	require.Equal(t, "15551234567", metadataString(t, md, "client.phone"))
	require.Equal(t, "15557654321", metadataString(t, md, "client.assistant_phone"))
	require.Equal(t, "live-call-id", metadataString(t, md, "client.provider_call_id"))
	require.Equal(t, "ctx-uuid-123", metadataString(t, md, "client.context_id"))
}

func TestCreateConnectionRequest_OmitsEmptyOptionalFields(t *testing.T) {
	// CallerNumber / FromNumber empty (defensive — e.g. degraded fallback path).
	cc := &callcontext.CallContext{
		Direction: "inbound",
		Provider:  "sip",
		ContextID: "ctx-1",
	}
	base := New(newTestLogger(t), cc, nil, nil)

	req := base.CreateConnectionRequest()
	md, err := utils.AnyMapToInterfaceMap(req.GetMetadata())
	require.NoError(t, err)

	_, hasPhone := md["client.phone"]
	require.False(t, hasPhone, "client.phone should be omitted when CallerNumber empty")
	_, hasAssistantPhone := md["client.assistant_phone"]
	require.False(t, hasAssistantPhone, "client.assistant_phone should be omitted when FromNumber empty")
	_, hasProviderCallID := md["client.provider_call_id"]
	require.False(t, hasProviderCallID, "client.provider_call_id should be omitted when ChannelUUID empty")

	// Required-when-known fields still emitted.
	require.Equal(t, "inbound", metadataString(t, md, "client.direction"))
	require.Equal(t, "sip", metadataString(t, md, "client.channel"))
	require.Equal(t, "ctx-1", metadataString(t, md, "client.context_id"))
}

func TestCreateConnectionRequest_EmitsResolvedPhoneValuesOnly(t *testing.T) {
	cc := &callcontext.CallContext{
		Direction:    "inbound",
		Provider:     "sip",
		CallerNumber: "07249994778",
		FromNumber:   "+447249994778",
	}
	base := New(newTestLogger(t), cc, nil, nil)

	req := base.CreateConnectionRequest()
	metadata, err := utils.AnyMapToInterfaceMap(req.GetMetadata())
	require.NoError(t, err)

	require.Equal(t, "07249994778", metadataString(t, metadata, "client.phone"))
	require.Equal(t, "+447249994778", metadataString(t, metadata, "client.assistant_phone"))
	for _, value := range metadata {
		if stringValue, ok := value.(string); ok {
			require.NotContains(t, stringValue, "sip:")
			require.NotContains(t, stringValue, "agent-")
		}
	}
}

func TestRecord_PassesRecordsThrough(t *testing.T) {
	cc := &callcontext.CallContext{
		AssistantID:    1,
		ConversationID: 42,
		Direction:      "outbound",
		Provider:       "sip",
		CallerNumber:   "15551234567",
		FromNumber:     "15557654321",
		ChannelUUID:    "provider-call-id",
		ContextID:      "ctx-uuid-123",
	}
	observer := &recordingObserver{}
	base := New(newTestLogger(t), cc, nil, observer)

	err := base.Record(observability.RecordEvent{
		Component: observability.ComponentCall,
		Event:     observability.CallHangup,
		Attributes: observability.Attributes{
			"reason": "remote_hangup",
		},
	})
	require.NoError(t, err)
	require.Len(t, observer.records, 1)

	eventRecord, ok := observer.records[0].(observability.RecordEvent)
	require.True(t, ok)
	require.Equal(t, observability.CallHangup, eventRecord.Event)
	require.Equal(t, "remote_hangup", eventRecord.Attributes["reason"])
}
