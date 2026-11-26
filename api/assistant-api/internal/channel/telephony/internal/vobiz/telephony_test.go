// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz_telephony

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_vobiz "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz/internal"
	configs "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiveCall_OutboundAnswerRequestReturnsXMLAndSkipsInboundSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/talk/vobiz/call/42?CustomField=v1%2Ftalk%2Fvobiz%2Fctx%2Fctx-1&StatusCallback=v1%2Ftalk%2Fvobiz%2Fctx%2Fctx-1%2Fevent", nil)
	c.Request = req

	tel := &vobizTelephony{appCfg: &config.AssistantConfig{AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "app.rapida.ai"}}}}
	callInfo, err := tel.ReceiveCall(c)

	require.NoError(t, err)
	assert.Nil(t, callInfo)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<Response><Stream`)
	assert.Contains(t, w.Body.String(), `statusCallbackUrl="https://app.rapida.ai/v1/talk/vobiz/ctx/ctx-1/event"`)
	assert.Contains(t, w.Body.String(), `wss://app.rapida.ai/v1/talk/vobiz/ctx/ctx-1`)
}

func TestReceiveCall_ParsesFormBodyPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := "BillRate=0&CallStatus=ringing&CallUUID=9c5f3046-776f-4d5f-8196-7f4c50e2c74b&CallerName=%2B919902426002&Direction=inbound&Event=StartApp&From=919902426002&To=08071387361"
	req := httptest.NewRequest(http.MethodPost, "/v1/talk/vobiz/call/42?x-api-key=test-key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	tel := &vobizTelephony{}
	callInfo, err := tel.ReceiveCall(c)

	require.NoError(t, err)
	require.NotNil(t, callInfo)
	assert.Equal(t, "919902426002", callInfo.CallerNumber)
	assert.Equal(t, "08071387361", callInfo.FromNumber)
	assert.Equal(t, "9c5f3046-776f-4d5f-8196-7f4c50e2c74b", callInfo.ChannelUUID)
}

func TestStatusCallback_ParsesFormBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("Event=Hangup&CallUUID=call-1&CallStatus=completed"))
	c.Request = req
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	tel := &vobizTelephony{logger: logger}
	statusInfo, err := tel.StatusCallback(c, authentication.Principal{}, 0, 0)

	require.NoError(t, err)
	require.NotNil(t, statusInfo)
	assert.Equal(t, "call-1", statusInfo.ChannelUUID)
	assert.True(t, statusInfo.Completed)
	assert.Equal(t, "Event=Hangup&CallUUID=call-1&CallStatus=completed", statusInfo.RawPayload)
}

func TestCatchAllStatusCallback_RequiresChannelUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?Event=Ring", nil)
	c.Request = req
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	tel := &vobizTelephony{logger: logger}
	_, err = tel.CatchAllStatusCallback(c)

	require.Error(t, err)
	assert.True(t, errors.Is(err, internal_vobiz.ErrCatchAllChannelUUIDMissing))
}
