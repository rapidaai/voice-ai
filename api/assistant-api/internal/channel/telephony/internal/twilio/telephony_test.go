// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio_telephony

import (
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_twilio "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func testVaultCredential(t *testing.T, values map[string]interface{}) *protos.VaultCredential {
	t.Helper()
	v, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("failed to create vault credential: %v", err)
	}
	return &protos.VaultCredential{Value: v}
}

func TestTwilioClientParams_ValidCredentials(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"account_sid":   "AC1234567890",
		"account_token": "token_abc",
	})

	params, err := twilioClientParams(cred)
	require.NoError(t, err)
	require.NotNil(t, params)
	assert.Equal(t, "AC1234567890", params.Username)
	assert.Equal(t, "token_abc", params.Password)
}

func TestTwilioClientParams_NilVaultValue(t *testing.T) {
	cred := &protos.VaultCredential{Value: nil}

	params, err := twilioClientParams(cred)
	assert.Error(t, err)
	assert.Nil(t, params)
	assert.True(t, errors.Is(err, internal_twilio.ErrVaultCredentialValueMissing))
	assert.Contains(t, err.Error(), "vault credential value is nil")
}

func TestTwilioClientParams_MissingAccountSid(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"account_token": "token_abc",
	})

	params, err := twilioClientParams(cred)
	assert.Error(t, err)
	assert.Nil(t, params)
	assert.True(t, errors.Is(err, internal_twilio.ErrVaultAccountSIDMissing))
	assert.Contains(t, err.Error(), "accountSid")
}

func TestTwilioClientParams_MissingAccountToken(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"account_sid": "AC1234567890",
	})

	params, err := twilioClientParams(cred)
	assert.Error(t, err)
	assert.Nil(t, params)
	assert.True(t, errors.Is(err, internal_twilio.ErrVaultAccountTokenMissing))
	assert.Contains(t, err.Error(), "account_token")
}

func TestTwilioClient_ValidCredentials(t *testing.T) {
	cred := testVaultCredential(t, map[string]interface{}{
		"account_sid":   "AC1234567890",
		"account_token": "token_abc",
	})

	client, err := twilioClient(cred)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestTwilioClient_NilVaultValue(t *testing.T) {
	cred := &protos.VaultCredential{Value: nil}

	client, err := twilioClient(cred)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, errors.Is(err, internal_twilio.ErrVaultCredentialValueMissing))
}

func TestCreateTwinML(t *testing.T) {
	tests := []struct {
		name         string
		mediaServer  string
		streamName   string
		path         string
		callback     string
		clientNumber string
	}{
		{
			name:         "creates valid TwiML",
			mediaServer:  "media.example.com",
			streamName:   "assistant-stream",
			path:         "v1/twilio/answer",
			callback:     "https://api.example.com/v1/twilio/events",
			clientNumber: "+12025550123",
		},
		{
			name:         "escapes attribute delimiters",
			mediaServer:  `media.example.com\" injected=\"true`,
			streamName:   `assistant\" name`,
			path:         `v1/twilio/answer?token=\"quoted\"&mode=test`,
			callback:     `https://api.example.com/events?next=\"quoted\"&mode=test`,
			clientNumber: `+12025550123\" injected=\"true`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			telephony := &twilioTelephony{}
			payload := telephony.CreateTwinML(test.mediaServer, test.streamName, test.path, test.callback, 42, test.clientNumber)

			var response twimlResponse
			require.NoError(t, xml.Unmarshal([]byte(payload), &response))
			assert.Equal(t, "wss://"+test.mediaServer+"/"+test.path, response.Connect.Stream.URL)
			assert.Equal(t, test.streamName, response.Connect.Stream.Name)
			assert.Equal(t, test.callback, response.Connect.Stream.StatusCallback)
			assert.Equal(t, "initiated ringing answered completed", response.Connect.Stream.StatusCallbackEvent)
			require.Len(t, response.Connect.Stream.Parameters, 2)
			assert.Equal(t, twimlParameter{Name: "assistant_id", Value: "42"}, response.Connect.Stream.Parameters[0])
			assert.Equal(t, twimlParameter{Name: "client_number", Value: test.clientNumber}, response.Connect.Stream.Parameters[1])
		})
	}
}

// TestReceiveCall tests the ReceiveCall method with Twilio webhook parameters
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
			name: "Valid Twilio webhook with all parameters",
			queryParams: map[string]string{
				"Called":        "+13345895552",
				"ToState":       "AL",
				"CallerCountry": "US",
				"Direction":     "inbound",
				"CallerState":   "PA",
				"ToZip":         "36303",
				"CallSid":       "CAf64ab88f90f35581dcb16e60f875ea4a",
				"To":            "+13345895552",
				"CallerZip":     "16901",
				"ToCountry":     "US",
				"StirVerstat":   "TN-Validation-Passed-B",
				"CalledZip":     "36303",
				"ApiVersion":    "2010-04-01",
				"CalledCity":    "DOTHAN",
				"CallStatus":    "ringing",
				"From":          "+15703768754",
				"AccountSid":    "546789087657890876DFGHJKASHDFBJK",
				"CalledCountry": "US",
				"CallerCity":    "MIDDLEBURY CENTER",
				"ToCity":        "DOTHAN",
				"FromCountry":   "US",
				"Caller":        "+15703768754",
				"FromCity":      "MIDDLEBURY CENTER",
				"CalledState":   "AL",
				"FromZip":       "16901",
				"FromState":     "PA",
			},
			expectedError: false,
			expectedPhone: "+15703768754",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				require.NotNil(t, info)
				assert.Equal(t, "twilio", info.Provider)
				assert.Equal(t, internal_type.TelephonyStatusSuccess, info.Status)
				assert.Equal(t, "+15703768754", info.CallerNumber)
				assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", info.ChannelUUID)

				// Check StatusInfo
				assert.Equal(t, internal_type.TelephonyEvent("webhook"), info.StatusInfo.Event)
				assert.NotNil(t, info.StatusInfo.Payload)
				payload, ok := info.StatusInfo.Payload.(map[string]string)
				require.True(t, ok, "Payload should be map[string]string")
				assert.Equal(t, "+15703768754", payload["From"])
				assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", payload["CallSid"])
			},
		},
		{
			name: "Valid webhook with minimal parameters",
			queryParams: map[string]string{
				"From":    "+15703768754",
				"To":      "+13345895552",
				"CallSid": "CAf64ab88f90f35581dcb16e60f875ea4a",
			},
			expectedError: false,
			expectedPhone: "+15703768754",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				require.NotNil(t, info)
				assert.Equal(t, "twilio", info.Provider)
				assert.Equal(t, internal_type.TelephonyStatusSuccess, info.Status)
				assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", info.ChannelUUID)
				assert.Equal(t, internal_type.TelephonyEvent("webhook"), info.StatusInfo.Event)
				assert.NotNil(t, info.StatusInfo.Payload)
			},
		},
		{
			name: "Valid webhook without CallSid",
			queryParams: map[string]string{
				"From": "+15703768754",
				"To":   "+13345895552",
			},
			expectedError: false,
			expectedPhone: "+15703768754",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				require.NotNil(t, info)
				assert.Equal(t, "twilio", info.Provider)
				assert.Equal(t, internal_type.TelephonyStatusSuccess, info.Status)
				assert.Empty(t, info.ChannelUUID, "ChannelUUID should be empty without CallSid")
				assert.Equal(t, internal_type.TelephonyEvent("webhook"), info.StatusInfo.Event)
			},
		},
		{
			name: "Missing 'From' parameter",
			queryParams: map[string]string{
				"To":         "+13345895552",
				"CallSid":    "CAf64ab88f90f35581dcb16e60f875ea4a",
				"CallStatus": "ringing",
			},
			expectedError: true,
			expectedPhone: "",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				// CallInfo should be nil on error
			},
		},
		{
			name: "Empty 'From' parameter",
			queryParams: map[string]string{
				"From": "",
				"To":   "+13345895552",
			},
			expectedError: true,
			expectedPhone: "",
			checkCallInfo: func(t *testing.T, info *internal_type.CallInfo) {
				// CallInfo should be nil on error
			},
		},
		{
			name: "Only CallSid without From",
			queryParams: map[string]string{
				"CallSid":    "CAf64ab88f90f35581dcb16e60f875ea4a",
				"To":         "+13345895552",
				"CallStatus": "ringing",
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
			telephony := &twilioTelephony{}

			// Call ReceiveCall
			callInfo, err := telephony.ReceiveCall(c)

			// Verify error expectation
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, callInfo)
				assert.True(t, errors.Is(err, internal_twilio.ErrInboundFromMissing))
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

func TestStatusCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	telephony, err := NewTwilioTelephony(&config.AssistantConfig{}, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		form        map[string]string
		checkStatus func(*testing.T, *internal_type.StatusInfo)
	}{
		{
			name: "completed call captures duration and price",
			form: map[string]string{
				"CallSid":      "CAf64ab88f90f35581dcb16e60f875ea4a",
				"CallStatus":   "completed",
				"CallDuration": "14",
				"Price":        "-0.02000",
			},
			checkStatus: func(t *testing.T, info *internal_type.StatusInfo) {
				require.NotNil(t, info)
				assert.Equal(t, internal_type.TelephonyEventCompleted, info.Event)
				assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", info.ChannelUUID)
				assert.True(t, info.Completed)
				require.NotNil(t, info.Duration)
				assert.Equal(t, 14*time.Second, *info.Duration)
				assert.Equal(t, "-0.02000", info.Price)
				assert.NotEmpty(t, info.RawPayload)
				assert.Nil(t, info.Error)
			},
		},
		{
			name: "busy call maps to failed error reason",
			form: map[string]string{
				"CallSid":    "CAf64ab88f90f35581dcb16e60f875ea4a",
				"CallStatus": "busy",
			},
			checkStatus: func(t *testing.T, info *internal_type.StatusInfo) {
				require.NotNil(t, info)
				assert.Equal(t, internal_type.TelephonyEventCompleted, info.Event)
				assert.False(t, info.Completed)
				require.NotNil(t, info.Error)
				assert.Equal(t, "failed", info.Error.Error)
				assert.Equal(t, "busy", info.Error.Reason)
			},
		},
		{
			name: "error code overrides completed as failed",
			form: map[string]string{
				"CallSid":      "CAf64ab88f90f35581dcb16e60f875ea4a",
				"CallStatus":   "completed",
				"ErrorCode":    "11200",
				"ErrorMessage": "HTTP retrieval failure",
			},
			checkStatus: func(t *testing.T, info *internal_type.StatusInfo) {
				require.NotNil(t, info)
				assert.Equal(t, internal_type.TelephonyEventCompleted, info.Event)
				assert.False(t, info.Completed)
				require.NotNil(t, info.Error)
				assert.Equal(t, "failed", info.Error.Error)
				assert.Equal(t, "HTTP retrieval failure", info.Error.Reason)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			formValues := url.Values{}
			for key, value := range tt.form {
				formValues.Add(key, value)
			}
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formValues.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			c.Request = req

			statusInfo, err := telephony.StatusCallback(c, nil, 1, 1)

			require.NoError(t, err)
			tt.checkStatus(t, statusInfo)
		})
	}
}

func TestCatchAllStatusCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	telephony, err := NewTwilioTelephony(&config.AssistantConfig{}, logger)
	require.NoError(t, err)

	t.Run("valid Twilio global event", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		queryValues := url.Values{}
		queryValues.Add("CallSid", "CAf64ab88f90f35581dcb16e60f875ea4a")
		queryValues.Add("CallStatus", "no-answer")
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)

		statusInfo, err := telephony.CatchAllStatusCallback(c)

		require.NoError(t, err)
		require.NotNil(t, statusInfo)
		assert.Equal(t, internal_type.TelephonyEventCompleted, statusInfo.Event)
		assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", statusInfo.ChannelUUID)
		assert.NotEmpty(t, statusInfo.RawPayload)
		require.NotNil(t, statusInfo.Error)
		assert.Equal(t, "no-answer", statusInfo.Error.Reason)
	})

	t.Run("missing CallSid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?CallStatus=completed", nil)

		statusInfo, err := telephony.CatchAllStatusCallback(c)

		assert.Error(t, err)
		assert.Nil(t, statusInfo)
		assert.True(t, errors.Is(err, internal_twilio.ErrStatusCallbackCallSIDMissing))
	})
}

// TestReceiveCall_QueryParameterExtraction tests that all query parameters are captured in CallInfo payload
func TestReceiveCall_QueryParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	queryParams := map[string]string{
		"Called":        "+13345895552",
		"ToState":       "AL",
		"CallerCountry": "US",
		"Direction":     "inbound",
		"CallerState":   "PA",
		"ToZip":         "36303",
		"CallSid":       "CAf64ab88f90f35581dcb16e60f875ea4a",
		"To":            "+13345895552",
		"CallerZip":     "16901",
		"ToCountry":     "US",
		"StirVerstat":   "TN-Validation-Passed-B",
		"CalledZip":     "36303",
		"ApiVersion":    "2010-04-01",
		"CalledCity":    "DOTHAN",
		"CallStatus":    "ringing",
		"From":          "+15703768754",
		"AccountSid":    "546789087657890876DFGHJKASHDFBJK",
		"CalledCountry": "US",
		"CallerCity":    "MIDDLEBURY CENTER",
		"ToCity":        "DOTHAN",
		"FromCountry":   "US",
		"Caller":        "+15703768754",
		"FromCity":      "MIDDLEBURY CENTER",
		"CalledState":   "AL",
		"FromZip":       "16901",
		"FromState":     "PA",
	}

	queryValues := url.Values{}
	for key, value := range queryParams {
		queryValues.Add(key, value)
	}

	req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
	c.Request = req

	telephony := &twilioTelephony{}
	callInfo, err := telephony.ReceiveCall(c)

	require.NoError(t, err)
	require.NotNil(t, callInfo)

	// Verify StatusInfo contains webhook event with all query parameters as payload
	assert.Equal(t, internal_type.TelephonyEvent("webhook"), callInfo.StatusInfo.Event)
	require.NotNil(t, callInfo.StatusInfo.Payload, "StatusInfo payload should not be nil")

	payloadMap, ok := callInfo.StatusInfo.Payload.(map[string]string)
	require.True(t, ok, "Payload should be map[string]string")

	for key, expectedValue := range queryParams {
		actualValue, exists := payloadMap[key]
		assert.True(t, exists, "Query param '%s' should be in payload", key)
		assert.Equal(t, expectedValue, actualValue, "Value for '%s' should match", key)
	}
}

// TestReceiveCall_PhoneNumberFormats tests various phone number formats
func TestReceiveCall_PhoneNumberFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		phoneNumber   string
		expectedPhone string
	}{
		{
			name:          "E.164 format with plus",
			phoneNumber:   "+15703768754",
			expectedPhone: "+15703768754",
		},
		{
			name:          "10-digit US number",
			phoneNumber:   "5703768754",
			expectedPhone: "5703768754",
		},
		{
			name:          "International format",
			phoneNumber:   "+441234567890",
			expectedPhone: "+441234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			queryValues := url.Values{}
			queryValues.Add("From", tt.phoneNumber)
			queryValues.Add("To", "+13345895552")

			req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
			c.Request = req

			telephony := &twilioTelephony{}
			callInfo, err := telephony.ReceiveCall(c)

			require.NoError(t, err)
			require.NotNil(t, callInfo)
			assert.Equal(t, tt.expectedPhone, callInfo.CallerNumber)
		})
	}
}

// TestReceiveCall_CallInfoStructure tests the structure of CallInfo data
func TestReceiveCall_CallInfoStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	queryValues := url.Values{}
	queryValues.Add("From", "+15703768754")
	queryValues.Add("To", "+13345895552")
	queryValues.Add("CallSid", "CAf64ab88f90f35581dcb16e60f875ea4a")
	queryValues.Add("CallStatus", "ringing")

	req := httptest.NewRequest(http.MethodGet, "/?"+queryValues.Encode(), nil)
	c.Request = req

	telephony := &twilioTelephony{}
	callInfo, err := telephony.ReceiveCall(c)

	require.NoError(t, err)
	require.NotNil(t, callInfo)

	// Verify CallInfo fields
	assert.Equal(t, "twilio", callInfo.Provider)
	assert.Equal(t, internal_type.TelephonyStatusSuccess, callInfo.Status)
	assert.Equal(t, "+15703768754", callInfo.CallerNumber)
	assert.Equal(t, "CAf64ab88f90f35581dcb16e60f875ea4a", callInfo.ChannelUUID)
	assert.Empty(t, callInfo.ErrorMessage)

	// Verify StatusInfo
	assert.Equal(t, internal_type.TelephonyEvent("webhook"), callInfo.StatusInfo.Event)
	assert.NotNil(t, callInfo.StatusInfo.Payload)
}
