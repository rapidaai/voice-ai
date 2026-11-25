// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo_telephony

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_plivo "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/plivo/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	configs "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newPlivoTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

func TestNewPlivoTelephony(t *testing.T) {
	cfg := &config.AssistantConfig{
		AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "test.example.com"}},
	}
	logger := newPlivoTestLogger(t)

	telephony, err := NewPlivoTelephony(cfg, logger)
	if err != nil {
		t.Fatalf("NewPlivoTelephony returned error: %v", err)
	}
	if telephony == nil {
		t.Fatal("NewPlivoTelephony returned nil")
	}
}

func TestTelephonyInterfaceCompliance(t *testing.T) {
	var _ internal_type.Telephony = (*plivoTelephony)(nil)
}

func TestGetCredentials(t *testing.T) {
	telephony := &plivoTelephony{}

	tests := []struct {
		name        string
		credMap     map[string]interface{}
		expectErr   bool
		expectID    string
		expectToken string
	}{
		{
			name:        "valid credentials",
			credMap:     map[string]interface{}{"auth_id": "MA123", "auth_token": "secret"},
			expectErr:   false,
			expectID:    "MA123",
			expectToken: "secret",
		},
		{
			name:      "missing auth_id",
			credMap:   map[string]interface{}{"auth_token": "secret"},
			expectErr: true,
		},
		{
			name:      "missing auth_token",
			credMap:   map[string]interface{}{"auth_id": "MA123"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structValue, _ := structpb.NewStruct(tt.credMap)
			vaultCred := &protos.VaultCredential{Value: structValue}

			authID, authToken, err := telephony.getCredentials(vaultCred)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				switch tt.name {
				case "missing auth_id":
					if !errors.Is(err, internal_plivo.ErrVaultAuthIDMissing) {
						t.Errorf("expected ErrVaultAuthIDMissing, got %v", err)
					}
				case "missing auth_token":
					if !errors.Is(err, internal_plivo.ErrVaultAuthTokenMissing) {
						t.Errorf("expected ErrVaultAuthTokenMissing, got %v", err)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if authID != tt.expectID {
				t.Errorf("expected auth_id %s, got %s", tt.expectID, authID)
			}
			if authToken != tt.expectToken {
				t.Errorf("expected auth_token %s, got %s", tt.expectToken, authToken)
			}
		})
	}
}

func TestGetCredentials_NilVault(t *testing.T) {
	telephony := &plivoTelephony{}
	_, _, err := telephony.getCredentials(nil)
	if err == nil {
		t.Error("expected error for nil vault credential, got nil")
	}
	if !errors.Is(err, internal_plivo.ErrVaultCredentialMissing) {
		t.Errorf("expected ErrVaultCredentialMissing, got %v", err)
	}
}

func TestOutboundCall_MissingCredentials(t *testing.T) {
	cfg := &config.AssistantConfig{
		AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "test.example.com"}},
	}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)
	var statusUpdate internal_type.ProviderCallStatusUpdate

	info, err := telephony.OutboundCall(context.Background(), nil, "+15551234567", "+15559876543", nil, 1, nil, func(update internal_type.ProviderCallStatusUpdate) {
		statusUpdate = update
	}, utils.Option{})
	if err == nil {
		t.Error("expected error for nil vault credential")
	}
	if info.Status != "FAILED" {
		t.Errorf("expected FAILED status, got %s", info.Status)
	}
	if statusUpdate.CallStatus != callcontext.CallStatusFailed {
		t.Errorf("expected outbound status failed, got %s", statusUpdate.CallStatus)
	}
	if statusUpdate.FailureClass != internal_telephony_base.OutboundFailureClassAuthentication {
		t.Errorf("expected authentication failure class, got %s", statusUpdate.FailureClass)
	}
}

func TestOutboundCall_RequestCancelled(t *testing.T) {
	cfg := &config.AssistantConfig{
		AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "test.example.com"}},
	}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	info, err := telephony.OutboundCall(ctx, nil, "+15551234567", "+15559876543", nil, 1, nil, nil, utils.Option{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
	if info.Status != "FAILED" {
		t.Errorf("expected FAILED status, got %s", info.Status)
	}
}

func TestHangupCall_MissingCredentials(t *testing.T) {
	telephony := &plivoTelephony{}
	err := telephony.HangupCall("call-123", nil)
	if err == nil {
		t.Error("expected error for nil vault credential")
	}
}

func TestReceiveCall(t *testing.T) {
	cfg := &config.AssistantConfig{}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)

	tests := []struct {
		name         string
		formValues   map[string]string
		expectErr    bool
		expectNumber string
		expectUUID   string
	}{
		{
			name:         "valid From/To/CallUUID",
			formValues:   map[string]string{"From": "+15551234567", "To": "+15559876543", "CallUUID": "call-uuid-1"},
			expectErr:    false,
			expectNumber: "+15551234567",
			expectUUID:   "call-uuid-1",
		},
		{
			name:       "missing From",
			formValues: map[string]string{"To": "+15559876543"},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			for k, v := range tt.formValues {
				form.Set(k, v)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/plivo/call/1", strings.NewReader(form.Encode()))
			c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			callInfo, err := telephony.ReceiveCall(c)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if callInfo.CallerNumber != tt.expectNumber {
				t.Errorf("expected caller number %s, got %s", tt.expectNumber, callInfo.CallerNumber)
			}
			if callInfo.Provider != internal_plivo.PlivoProvider {
				t.Errorf("expected provider %s, got %s", internal_plivo.PlivoProvider, callInfo.Provider)
			}
			if callInfo.ChannelUUID != tt.expectUUID {
				t.Errorf("expected channel uuid %s, got %s", tt.expectUUID, callInfo.ChannelUUID)
			}
		})
	}
}

func TestStatusCallback(t *testing.T) {
	cfg := &config.AssistantConfig{}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)

	form := url.Values{}
	form.Set("CallUUID", "call-uuid-9")
	form.Set("CallStatus", "completed")
	form.Set("HangupCause", "NORMAL_CLEARING")
	form.Set("Duration", "12")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/plivo/ctx/ctx-1/event", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	statusInfo, err := telephony.StatusCallback(c, nil, 1, 1)
	if err != nil {
		t.Fatalf("StatusCallback returned error: %v", err)
	}
	if statusInfo.ChannelUUID != "call-uuid-9" {
		t.Errorf("expected call-uuid-9, got %s", statusInfo.ChannelUUID)
	}
	if !statusInfo.Completed {
		t.Error("expected completed=true for NORMAL_CLEARING hangup")
	}
	if statusInfo.Error != nil {
		t.Errorf("expected no error, got %+v", statusInfo.Error)
	}
}

func TestCatchAllStatusCallback_MissingUUID(t *testing.T) {
	cfg := &config.AssistantConfig{}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)

	form := url.Values{}
	form.Set("CallStatus", "completed")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/plivo/event/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	statusInfo, err := telephony.CatchAllStatusCallback(c)
	if err == nil {
		t.Error("expected error for missing CallUUID")
	}
	if statusInfo != nil {
		t.Errorf("expected nil StatusInfo, got %+v", statusInfo)
	}
}

func TestInboundCall_ReturnsStreamXML(t *testing.T) {
	cfg := &config.AssistantConfig{
		AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "test.example.com"}},
	}
	logger := newPlivoTestLogger(t)
	telephony, _ := NewPlivoTelephony(cfg, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("contextId", "ctx-42")
	c.Request = httptest.NewRequest("POST", "/plivo/call/1", nil)

	if err := telephony.InboundCall(c, nil, 1, "+15551234567", 1); err != nil {
		t.Fatalf("InboundCall returned error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<Stream") {
		t.Errorf("expected <Stream> element, got %s", body)
	}
	if !strings.Contains(body, "ctx-42") {
		t.Errorf("expected context id in stream url, got %s", body)
	}
	if !strings.Contains(body, "audio/x-mulaw;rate=8000") {
		t.Errorf("expected mu-law content type, got %s", body)
	}
}
