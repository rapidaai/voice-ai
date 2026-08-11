// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk_telephony

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

func newAsteriskTelephonyForTest(t *testing.T) *asteriskTelephony {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return &asteriskTelephony{
		appCfg: &config.AssistantConfig{},
		logger: logger,
	}
}

func TestStatusCallback_NormalizesChannelDestroyedCompleted(t *testing.T) {
	tel := newAsteriskTelephonyForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"type":"ChannelDestroyed","channel":{"id":"ast-chan-1"},"cause":16,"cause_txt":"NORMAL_CLEARING","duration_ms":2450}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	info, err := tel.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Event != internal_type.TelephonyEventCompleted {
		t.Fatalf("expected completed event, got %q", info.Event)
	}
	if !info.Completed {
		t.Fatal("expected completed callback")
	}
	if info.Error != nil {
		t.Fatalf("expected no error, got %+v", info.Error)
	}
	if info.ChannelUUID != "ast-chan-1" {
		t.Fatalf("expected ChannelUUID ast-chan-1, got %q", info.ChannelUUID)
	}
	if info.Duration == nil || info.Duration.Milliseconds() != 2450 {
		t.Fatalf("expected duration 2450ms, got %v", info.Duration)
	}
	if info.RawPayload != body {
		t.Fatalf("expected raw payload %q, got %q", body, info.RawPayload)
	}
}

func TestStatusCallback_NormalizesDialFailure(t *testing.T) {
	tel := newAsteriskTelephonyForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"type":"Dial","dialstatus":"BUSY","channel":{"id":"ast-chan-2"}}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	info, err := tel.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Error == nil {
		t.Fatal("expected failed status error")
	}
	if info.Error.Reason != "BUSY" {
		t.Fatalf("expected failure reason BUSY, got %q", info.Error.Reason)
	}
	if info.ChannelUUID != "ast-chan-2" {
		t.Fatalf("expected ChannelUUID ast-chan-2, got %q", info.ChannelUUID)
	}
}

func TestStatusCallback_NormalizesRingingStateChange(t *testing.T) {
	tel := newAsteriskTelephonyForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"type":"ChannelStateChange","channel":{"id":"ast-chan-3","state":"Ring"}}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	info, err := tel.StatusCallback(c, nil, 42, 99)
	if err != nil {
		t.Fatalf("StatusCallback() error = %v", err)
	}

	if info.Event != internal_type.TelephonyEventRinging {
		t.Fatalf("expected ringing event, got %q", info.Event)
	}
	if info.Completed || info.Error != nil {
		t.Fatalf("ringing callback must not be terminal: completed=%v error=%+v", info.Completed, info.Error)
	}
	if info.ChannelUUID != "ast-chan-3" {
		t.Fatalf("expected ChannelUUID ast-chan-3, got %q", info.ChannelUUID)
	}
}

func TestReceiveCall_PopulatesDialedNumberFromFallbackParams(t *testing.T) {
	tel := newAsteriskTelephonyForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/?callerid=15551234567&dnid=18005550100&channel_id=ast-chan-1", nil)
	c.Request = req

	info, err := tel.ReceiveCall(c)
	if err != nil {
		t.Fatalf("ReceiveCall() error = %v", err)
	}

	if info.CallerNumber != "15551234567" {
		t.Fatalf("expected CallerNumber 15551234567, got %q", info.CallerNumber)
	}
	if info.FromNumber != "18005550100" {
		t.Fatalf("expected FromNumber from dnid fallback, got %q", info.FromNumber)
	}
	if info.ChannelUUID != "ast-chan-1" {
		t.Fatalf("expected ChannelUUID ast-chan-1, got %q", info.ChannelUUID)
	}
	payload, ok := info.StatusInfo.Payload.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string payload, got %T", info.StatusInfo.Payload)
	}
	if got := payload["to"]; got != "18005550100" {
		t.Fatalf("expected status payload to=18005550100, got %q", got)
	}
}

func TestReceiveCall_PopulatesCallerNumberFromCallerParam(t *testing.T) {
	tel := newAsteriskTelephonyForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/?caller=15557654321&to=18005550101", nil)
	c.Request = req

	info, err := tel.ReceiveCall(c)
	if err != nil {
		t.Fatalf("ReceiveCall() error = %v", err)
	}

	if info.CallerNumber != "15557654321" {
		t.Fatalf("expected CallerNumber 15557654321, got %q", info.CallerNumber)
	}
	if info.FromNumber != "18005550101" {
		t.Fatalf("expected FromNumber 18005550101, got %q", info.FromNumber)
	}
	payload, ok := info.StatusInfo.Payload.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string payload, got %T", info.StatusInfo.Payload)
	}
	if got := payload["caller"]; got != "15557654321" {
		t.Fatalf("expected status payload caller=15557654321, got %q", got)
	}
}
