// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage_telephony

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_vonage "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vonage/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type recordingObserver struct {
	records []observability.Record
}

func (r *recordingObserver) Record(_ context.Context, _ observability.Scope, records ...observability.Record) error {
	r.records = append(r.records, records...)
	return nil
}

func (r *recordingObserver) AddCollectors(...observability.Collector) error {
	return nil
}

func (r *recordingObserver) Close(context.Context) error {
	return nil
}

func testVaultCredential(t *testing.T, values map[string]interface{}) *protos.VaultCredential {
	t.Helper()
	v, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("failed to create vault credential: %v", err)
	}
	return &protos.VaultCredential{Value: v}
}

func TestVonageAuth_ValidCredentials(t *testing.T) {
	// vonage.CreateAuthFromAppPrivateKey requires a valid PEM key, so this
	// test verifies the vault parsing path up to the SDK call. We expect an
	// error from the SDK due to the non-PEM key, but NOT a vault-parsing error.
	cred := testVaultCredential(t, map[string]interface{}{
		"private_key":    "not-a-real-pem-key",
		"application_id": "app-123",
	})

	_, err := vonageAuth(cred)
	// The SDK will reject the fake key, but the vault parsing succeeded.
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "vault credential")
	assert.NotContains(t, err.Error(), "illegal vault config")
}

func TestVonageAuth_NilVaultValue(t *testing.T) {
	cred := &protos.VaultCredential{Value: nil}

	_, err := vonageAuth(cred)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, internal_vonage.ErrVaultCredentialValueMissing))
	assert.Contains(t, err.Error(), "vault credential value is nil")
}

func TestVonageAuth_MissingPrivateKey(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"application_id": "app-123",
	})

	_, err := vonageAuth(cred)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, internal_vonage.ErrVaultPrivateKeyMissing))
	assert.Contains(t, err.Error(), "privateKey")
}

func TestVonageAuth_MissingApplicationId(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"private_key": "some-key",
	})

	_, err := vonageAuth(cred)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, internal_vonage.ErrVaultApplicationIDMissing))
	assert.Contains(t, err.Error(), "application_id")
}

func TestCatchAllStatusCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	telephony, err := NewVonageTelephony(&config.AssistantConfig{}, logger)
	require.NoError(t, err)

	tests := []struct {
		name          string
		queryParams   map[string]string
		expectedError bool
		checkStatus   func(*testing.T, *internal_type.StatusInfo)
	}{
		{
			name: "valid Vonage global event",
			queryParams: map[string]string{
				"status":            "completed",
				"uuid":              "6fbb257e-75e5-4c68-a5f2-bc560fa200d3",
				"conversation_uuid": "CON-dbde628c-2039-4fe3-83f2-9e757d3e9e13",
				"direction":         "outbound",
				"disconnected_by":   "user",
				"detail":            "remote_busy",
				"sip_code":          "486",
			},
			checkStatus: func(t *testing.T, statusInfo *internal_type.StatusInfo) {
				require.NotNil(t, statusInfo)
				assert.Equal(t, internal_type.TelephonyEventCompleted, statusInfo.Event)
				assert.Equal(t, "6fbb257e-75e5-4c68-a5f2-bc560fa200d3", statusInfo.ChannelUUID)
				assert.Nil(t, statusInfo.Error)
				payload, ok := statusInfo.Payload.(utils.Option)
				require.True(t, ok, "Payload should be utils.Option")
				assert.Equal(t, "6fbb257e-75e5-4c68-a5f2-bc560fa200d3", payload["uuid"])
				assert.Equal(t, "CON-dbde628c-2039-4fe3-83f2-9e757d3e9e13", payload["conversation_uuid"])
				assert.Equal(t, "user", payload["disconnected_by"])
				assert.Equal(t, "486", payload["sip_code"])
			},
		},
		{
			name: "failed completed event captures reason duration and price",
			queryParams: map[string]string{
				"status":   "completed",
				"uuid":     "f9abbc8a-457a-40b2-a8bf-3717c0abc918",
				"detail":   "remote_busy",
				"duration": "0",
				"price":    "0.00000000",
				"sip_code": "486",
			},
			checkStatus: func(t *testing.T, statusInfo *internal_type.StatusInfo) {
				require.NotNil(t, statusInfo)
				assert.Equal(t, internal_type.TelephonyEventCompleted, statusInfo.Event)
				assert.Equal(t, "f9abbc8a-457a-40b2-a8bf-3717c0abc918", statusInfo.ChannelUUID)
				require.NotNil(t, statusInfo.Duration)
				assert.Equal(t, time.Duration(0), *statusInfo.Duration)
				assert.Equal(t, "0.00000000", statusInfo.Price)
				require.NotNil(t, statusInfo.Error)
				assert.Equal(t, "failed", statusInfo.Error.Error)
				assert.Equal(t, "remote_busy", statusInfo.Error.Reason)
			},
		},
		{
			name: "missing status",
			queryParams: map[string]string{
				"uuid":              "6fbb257e-75e5-4c68-a5f2-bc560fa200d3",
				"conversation_uuid": "CON-dbde628c-2039-4fe3-83f2-9e757d3e9e13",
			},
			expectedError: true,
		},
		{
			name: "empty status",
			queryParams: map[string]string{
				"status": "",
				"uuid":   "6fbb257e-75e5-4c68-a5f2-bc560fa200d3",
			},
			expectedError: true,
		},
		{
			name: "missing uuid",
			queryParams: map[string]string{
				"status": "completed",
			},
			expectedError: true,
		},
		{
			name: "empty uuid",
			queryParams: map[string]string{
				"status": "completed",
				"uuid":   "",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			queryValues := url.Values{}
			for key, value := range tt.queryParams {
				queryValues.Add(key, value)
			}
			c.Request = httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)

			statusInfo, err := telephony.CatchAllStatusCallback(c)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, statusInfo)
				return
			}
			assert.NoError(t, err)
			if tt.checkStatus != nil {
				tt.checkStatus(t, statusInfo)
			}
		})
	}
}

func TestStatusCallback_QueryPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	telephony, err := NewVonageTelephony(&config.AssistantConfig{}, logger)
	require.NoError(t, err)

	queryValues := url.Values{}
	queryValues.Add("status", "completed")
	queryValues.Add("uuid", "f9abbc8a-457a-40b2-a8bf-3717c0abc918")
	queryValues.Add("detail", "remote_busy")
	queryValues.Add("duration", "0")
	queryValues.Add("price", "0.00000000")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)

	statusInfo, err := telephony.StatusCallback(c, nil, 1, 1)

	require.NoError(t, err)
	require.NotNil(t, statusInfo)
	assert.Equal(t, internal_type.TelephonyEventCompleted, statusInfo.Event)
	assert.Equal(t, "f9abbc8a-457a-40b2-a8bf-3717c0abc918", statusInfo.ChannelUUID)
	require.NotNil(t, statusInfo.Duration)
	assert.Equal(t, time.Duration(0), *statusInfo.Duration)
	assert.Equal(t, "0.00000000", statusInfo.Price)
	require.NotNil(t, statusInfo.Error)
	assert.Equal(t, "failed", statusInfo.Error.Error)
	assert.Equal(t, "remote_busy", statusInfo.Error.Reason)
}

func TestStatusCallback_JSONNumericZeroDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	telephony, err := NewVonageTelephony(&config.AssistantConfig{}, logger)
	require.NoError(t, err)

	body, err := json.Marshal(map[string]interface{}{
		"status":   "completed",
		"uuid":     "f9abbc8a-457a-40b2-a8bf-3717c0abc918",
		"detail":   "remote_busy",
		"duration": float64(0),
		"price":    "0.00000000",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	statusInfo, err := telephony.StatusCallback(c, nil, 1, 1)

	require.NoError(t, err)
	require.NotNil(t, statusInfo)
	require.NotNil(t, statusInfo.Duration)
	assert.Equal(t, time.Duration(0), *statusInfo.Duration)
	require.NotNil(t, statusInfo.Error)
	assert.Equal(t, "failed", statusInfo.Error.Error)
	assert.Equal(t, "remote_busy", statusInfo.Error.Reason)
}

// TestReceiveCall tests the ReceiveCall method with Vonage webhook parameters
func TestReceiveCall(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		queryParams   map[string]string
		expectedError bool
		expectedPhone string
		checkCallInfo func(*testing.T, *internal_type.CallInfo)
	}{
		{
			name: "Valid Vonage webhook with all parameters",
			queryParams: map[string]string{
				"from":              "15703768754",
				"to":                "12019868532",
				"endpoint_type":     "phone",
				"conversation_uuid": "CON-3d4ae1dd-5e14-4131-be3d-0247cb19a28a",
				"uuid":              "bccbc3faaf864e1641fe0cdb1921b6aa",
				"region_url":        "https://api-ap-3.vonage.com",
			},
			expectedError: false,
			expectedPhone: "15703768754",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				require.NotNil(t, info)
				assert.Equal(t, internal_vonage.Provider, info.Provider)
				assert.Equal(t, internal_type.TelephonyStatusSuccess, info.Status)
				assert.Equal(t, "15703768754", info.CallerNumber)
				assert.Equal(t, "bccbc3faaf864e1641fe0cdb1921b6aa", info.ChannelUUID)

				// Check StatusInfo
				assert.Equal(t, internal_type.TelephonyEvent(internal_vonage.WebhookEvent), info.StatusInfo.Event)
				assert.NotNil(t, info.StatusInfo.Payload)
				payload, ok := info.StatusInfo.Payload.(map[string]string)
				require.True(t, ok, "Payload should be map[string]string")
				assert.Equal(t, "15703768754", payload["from"])

				// Check Extra for conversation_uuid
				require.NotNil(t, info.Extra)
				assert.Equal(t, "CON-3d4ae1dd-5e14-4131-be3d-0247cb19a28a", info.Extra["conversation_uuid"])
			},
		},
		{
			name: "Valid webhook with minimal parameters",
			queryParams: map[string]string{
				"from": "15703768754",
				"to":   "12019868532",
			},
			expectedError: false,
			expectedPhone: "15703768754",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				require.NotNil(t, info)
				assert.Equal(t, internal_vonage.Provider, info.Provider)
				assert.Equal(t, internal_type.TelephonyStatusSuccess, info.Status)
				assert.Equal(t, internal_type.TelephonyEvent(internal_vonage.WebhookEvent), info.StatusInfo.Event)
				assert.NotNil(t, info.StatusInfo.Payload)
				assert.Empty(t, info.ChannelUUID, "ChannelUUID should be empty without uuid param")
			},
		},
		{
			name: "Missing 'from' parameter",
			queryParams: map[string]string{
				"to":                "12019868532",
				"conversation_uuid": "CON-3d4ae1dd-5e14-4131-be3d-0247cb19a28a",
			},
			expectedError: true,
			expectedPhone: "",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				// CallInfo should be nil on error
			},
		},
		{
			name: "Empty 'from' parameter",
			queryParams: map[string]string{
				"from": "",
				"to":   "12019868532",
			},
			expectedError: true,
			expectedPhone: "",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				// CallInfo should be nil on error
			},
		},
		{
			name: "Only conversation_uuid without phone",
			queryParams: map[string]string{
				"conversation_uuid": "CON-3d4ae1dd-5e14-4131-be3d-0247cb19a28a",
				"uuid":              "bccbc3faaf864e1641fe0cdb1921b6aa",
			},
			expectedError: true,
			expectedPhone: "",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				// CallInfo should be nil on error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Build query string
			queryValues := url.Values{}
			for key, value := range tt.queryParams {
				queryValues.Add(key, value)
			}

			// Create request with query parameters
			req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
			c.Request = req

			// Create telephony instance
			telephony := &vonageTelephony{}

			// Call ReceiveCall
			callInfo, err := telephony.ReceiveCall(c)

			// Verify error expectation
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, callInfo)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, callInfo)
				assert.Equal(t, tt.expectedPhone, callInfo.CallerNumber)
			}

			// Check CallInfo
			if tt.checkCallInfo != nil {
				tt.checkCallInfo(t, callInfo)
			}
		})
	}
}

// TestSend_EndConversation_PushesToolCallResult verifies that the END_CONVERSATION
// tool-call action pushes a ConversationToolCallResult with status "completed"
// into the critical channel before cancelling the streamer.
func TestSend_EndConversation_PushesToolCallResult(t *testing.T) {
	// Set up a minimal WebSocket server so the streamer has a non-nil connection.
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("test server upgrade error: %v", err)
			return
		}
		defer conn.Close()
		// Keep the server alive until the test finishes.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	// Dial the test server.
	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Build a vonageWebsocketStreamer with empty ChannelUUID so the Vonage API
	// call is skipped (the if-block on GetConversationUuid() != "" is false).
	cc := &callcontext.CallContext{} // empty ChannelUUID
	logger, _ := commons.NewApplicationLogger()
	vng := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, cc, nil, nil),
		connection:            conn,
	}

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-123",
		ToolId: "tool-456",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err = vng.Send(toolCall)
	assert.NoError(t, err)

	// The Input call routes ConversationToolCallResult to CriticalCh.
	select {
	case msg := <-vng.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected *protos.ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-123", result.GetId())
		assert.Equal(t, "tool-456", result.GetToolId())
		assert.Equal(t, "end_conversation", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION, result.GetAction())
		assert.Equal(t, map[string]string{"status": "completed"}, result.GetResult())
	default:
		t.Fatal("expected a ConversationToolCallResult on CriticalCh, but channel was empty")
	}

	// Context should remain open; disconnect is owned by handleToolResult in adapter layer.
	select {
	case <-vng.Ctx.Done():
		t.Fatal("expected streamer context to remain open after END_CONVERSATION")
	default:
		// expected
	}
}

// TestSend_EndConversation_NilConnection verifies that Send returns early
// without panicking when the connection is nil (e.g. already closed).
func TestSend_EndConversation_NilConnection(t *testing.T) {
	cc := &callcontext.CallContext{}
	logger, _ := commons.NewApplicationLogger()
	vng := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, cc, nil, nil),
		connection:            nil,
	}

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-999",
		ToolId: "tool-888",
		Name:   "end_conversation",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}

	err := vng.Send(toolCall)
	assert.NoError(t, err)

	// CriticalCh should be empty since Send returned early.
	select {
	case msg := <-vng.CriticalCh:
		t.Fatalf("expected empty CriticalCh, but got %T", msg)
	default:
		// expected
	}
}

func TestSend_Disconnection_LogsVonageAuthError(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	logger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	observer := &recordingObserver{}

	vng := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			logger,
			&callcontext.CallContext{
				AssistantID:    1,
				ConversationID: 2,
				ChannelUUID:    "vonage-call-id",
			},
			nil,
			observer,
		),
		connection: conn,
	}

	err = vng.Send(&protos.ConversationDisconnection{
		Type: protos.ConversationDisconnection_DISCONNECTION_TYPE_USER,
	})
	require.NoError(t, err)

	require.Len(t, observer.records, 2)
	logRecord, ok := observer.records[0].(observability.RecordLog)
	require.True(t, ok)
	assert.Equal(t, "Failed to create Vonage client for server-side disconnect", logRecord.Message)
	assert.Equal(t, "vonage-call-id", logRecord.Attributes["conversation_uuid"])
	assert.Equal(t, protos.ConversationDisconnection_DISCONNECTION_TYPE_USER.String(), logRecord.Attributes["disconnection_type"])

	metricRecord, ok := observer.records[1].(observability.RecordMetric)
	require.True(t, ok)
	require.Len(t, metricRecord.Metrics, 1)
	assert.Equal(t, observability.MetricCallStatus, metricRecord.Metrics[0].Name)
	assert.Equal(t, "FAILED", metricRecord.Metrics[0].Value)
}

// TestSend_TransferConversation_PushesFailedResult verifies that the
// TRANSFER_CONVERSATION tool-call action pushes a "failed" result.
func TestSend_TransferConversation_PushesFailedResult(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	cc := &callcontext.CallContext{}
	logger, _ := commons.NewApplicationLogger()
	vng := &vonageWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(logger, cc, nil, nil),
		connection:            conn,
	}

	toolCall := &protos.ConversationToolCall{
		Id:     "tc-transfer",
		ToolId: "tool-xfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err = vng.Send(toolCall)
	assert.NoError(t, err)

	select {
	case msg := <-vng.CriticalCh:
		result, ok := msg.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected *protos.ConversationToolCallResult, got %T", msg)
		assert.Equal(t, "tc-transfer", result.GetId())
		assert.Equal(t, "tool-xfer", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported for Vonage")
	default:
		t.Fatal("expected a ConversationToolCallResult on CriticalCh, but channel was empty")
	}
}

// TestReceiveCall_QueryParameterExtraction tests that all query parameters are captured in CallInfo payload
func TestReceiveCall_QueryParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	queryParams := map[string]string{
		"from":              "15703768754",
		"to":                "12019868532",
		"endpoint_type":     "phone",
		"conversation_uuid": "CON-3d4ae1dd-5e14-4131-be3d-0247cb19a28a",
		"uuid":              "bccbc3faaf864e1641fe0cdb1921b6aa",
		"region_url":        "https://api-ap-3.vonage.com",
		"x-api-key":         "3dd5c2eef53d27942bccd892750fda23ea0b92965d4699e73d8e754ab882955f",
	}

	queryValues := url.Values{}
	for key, value := range queryParams {
		queryValues.Add(key, value)
	}

	req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
	c.Request = req

	telephony := &vonageTelephony{}
	callInfo, err := telephony.ReceiveCall(c)

	require.NoError(t, err)
	require.NotNil(t, callInfo)

	// Verify StatusInfo contains webhook event with all query parameters as payload
	assert.Equal(t, internal_type.TelephonyEvent(internal_vonage.WebhookEvent), callInfo.StatusInfo.Event)
	require.NotNil(t, callInfo.StatusInfo.Payload, "StatusInfo payload should not be nil")

	payloadMap, ok := callInfo.StatusInfo.Payload.(map[string]string)
	require.True(t, ok, "Payload should be map[string]string")

	for key, expectedValue := range queryParams {
		actualValue, exists := payloadMap[key]
		assert.True(t, exists, "Query param '%s' should be in payload", key)
		assert.Equal(t, expectedValue, actualValue, "Value for '%s' should match", key)
	}
}

func TestReceiveCall_MissingFromUsesTypedError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?to=12019868532", nil)

	callInfo, err := (&vonageTelephony{}).ReceiveCall(c)

	require.Nil(t, callInfo)
	require.Error(t, err)
	assert.True(t, errors.Is(err, internal_vonage.ErrInboundFromMissing))
}
