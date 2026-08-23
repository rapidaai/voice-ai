// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx_telephony

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

const (
	// Telnyx API base URL
	telnyxAPIBaseURL = "https://api.telnyx.com/v2"
)

type telnyxTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewTelnyxTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &telnyxTelephony{
		appCfg: config,
		logger: logger,
	}, nil
}

// CatchAllStatusCallback handles catch-all event callbacks.
func (tpc *telnyxTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	statusInfo, err := tpc.StatusCallback(ctx, nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if statusInfo == nil || !validator.NotBlank(statusInfo.ChannelUUID) {
		tpc.logger.Errorf("call control id not found or invalid in catch-all payload")
		return nil, internal_telnyx.ErrCatchAllCallControlIDMissing
	}
	return statusInfo, nil
}

// StatusCallback handles a status/event callback for a conversation.
// Telnyx sends webhooks for call events like call.answered, call.hangup, etc.
func (tpc *telnyxTelephony) StatusCallback(c *gin.Context, auth *types.Authentication, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	body, err := c.GetRawData()
	if err != nil {
		tpc.logger.Errorf("failed to read request body with error %+v", err)
		return nil, fmt.Errorf("%w: %w", internal_telnyx.ErrRequestBodyReadFailed, err)
	}

	var payload utils.Option
	if err := json.Unmarshal(body, &payload); err != nil {
		tpc.logger.Errorf("failed to parse request body: %+v", err)
		return nil, fmt.Errorf("%w: %w", internal_telnyx.ErrRequestBodyParseFailed, err)
	}

	callback, err := internal_telnyx.NewStatusCallback(payload, string(body))
	if err != nil {
		tpc.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}

	tpc.logger.Debugf("event processed | event_type: %s, payload: %+v", callback.EventType, payload)
	return callback.StatusInfo(), nil
}

// ReceiveCall processes an incoming call webhook and returns structured call info.
// Telnyx sends a webhook with call.answered event when an inbound call is received.
func (tpc *telnyxTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	// Parse query parameters
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Telnyx sends from/to in query params or in the webhook payload
	clientNumber := queryParams["from"]
	if clientNumber == "" {
		clientNumber = queryParams["caller_id"]
	}
	if clientNumber == "" {
		// Try from request body
		body, err := c.GetRawData()
		if err != nil {
			tpc.logger.Warnf("failed to read request body for caller number: %v", err)
		} else {
			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err == nil {
				if data, ok := payload["data"].(map[string]interface{}); ok {
					if payloadData, ok := data["payload"].(map[string]interface{}); ok {
						if from, ok := payloadData["from"].(string); ok {
							clientNumber = from
						}
					}
				}
			}
		}
	}

	if clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing caller number"})
		return nil, internal_telnyx.ErrInboundFromMissing
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		Provider:     internal_telnyx.Provider,
		Status:       internal_type.TelephonyStatusSuccess,
		StatusInfo:   internal_type.StatusInfo{Event: internal_type.TelephonyEvent(internal_telnyx.WebhookEvent), Payload: queryParams},
		Extra:        make(map[string]string),
	}

	// Extract call_control_id if present
	if v, ok := queryParams["call_control_id"]; ok && v != "" {
		info.ChannelUUID = v
		info.Extra["call_control_id"] = v
	}

	return info, nil
}

// OutboundCall places an outbound call using Telnyx Call Control API.
// POST to /v2/calls with connection_id, to, from, and stream_url.
func (tpc *telnyxTelephony) OutboundCall(
	ctx context.Context,
	auth *types.Authentication,
	toPhone string,
	fromPhone string,
	assistant *internal_assistant_entity.Assistant,
	assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	statusReporter internal_type.ProviderCallStatusReporter,
	opts utils.Option,
) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_telnyx.Provider}

	if err := ctx.Err(); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			internal_telnyx.OutboundFailureReasonRequestCancelled.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
			err,
			0,
		)
		return info, err
	}

	apiKey, connectionID, err := tpc.getCredentials(vaultCredential)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassAuthentication,
			internal_telnyx.OutboundFailureReasonAuthenticationFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")

	// Build the WebSocket stream URL for bidirectional audio
	streamURL := fmt.Sprintf("wss://%s/%s",
		tpc.appCfg.Assistant.Public,
		internal_type.GetContextAnswerPath(internal_telnyx.Provider, contextID))

	// Build the request body for Telnyx Call Control API
	callRequest := map[string]interface{}{
		"connection_id": connectionID,
		"to":            toPhone,
		"from":          fromPhone,
		"stream_url":    streamURL,
		"stream_track":  internal_telnyx.StreamTrackBoth,
	}

	requestBody, err := json.Marshal(callRequest)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestPayload,
			internal_telnyx.OutboundFailureReasonRequestPayloadFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", telnyxAPIBaseURL+"/calls", bytes.NewReader(requestBody))
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCreation,
			internal_telnyx.OutboundFailureReasonRequestCreateFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	req.Header.Set(internal_telnyx.HTTPHeaderContentType, internal_telnyx.HTTPContentTypeApplication)
	req.Header.Set(internal_telnyx.HTTPHeaderAuthorization, "Bearer "+apiKey)

	// Send the request with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_telnyx.OutboundFailureReasonProviderAPIError.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_telnyx.OutboundFailureReasonResponseReadFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// Parse the response
	var callResponse map[string]interface{}
	if err := json.Unmarshal(respBody, &callResponse); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_telnyx.OutboundFailureReasonResponseDecodeFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if errors, ok := callResponse["errors"].([]interface{}); ok && len(errors) > 0 {
			if errMap, ok := errors[0].(map[string]interface{}); ok {
				if detail, ok := errMap["detail"].(string); ok {
					errMsg = detail
				}
			}
		}
		err := fmt.Errorf("%w: %s", internal_telnyx.ErrProviderAPIError, errMsg)
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_telnyx.OutboundFailureReasonHTTPStatusFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// Extract call information from response
	var callData map[string]interface{}
	if data, ok := callResponse["data"].(map[string]interface{}); ok {
		callData = data
	} else {
		callData = callResponse
	}

	// Get call_control_id
	callControlID := ""
	if id, ok := callData["call_control_id"].(string); ok {
		callControlID = id
	} else if id, ok := callData["id"].(string); ok {
		callControlID = id
	}

	info.ChannelUUID = callControlID
	info.Status = internal_type.TelephonyStatusSuccess

	// Extract call session ID if present
	callSessionID := ""
	if sid, ok := callData["call_session_id"].(string); ok {
		callSessionID = sid
	}

	info.Extra = map[string]string{
		"call_control_id": callControlID,
		"call_session_id": callSessionID,
	}

	// Determine event name
	eventName := "initiated"
	if callStatus, ok := callData["status"].(string); ok {
		eventName = callStatus
	}

	info.StatusInfo = internal_type.StatusInfo{Event: internal_telnyx.StatusEvent(eventName), Payload: callResponse}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)

	return info, nil
}

// InboundCall instructs Telnyx to answer the inbound call and connect to our WebSocket.
// Returns JSON response for Telnyx to execute streaming.start command.
func (tpc *telnyxTelephony) InboundCall(c *gin.Context, auth *types.Authentication, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)

	// Return JSON to tell Telnyx to start streaming
	c.JSON(http.StatusOK, gin.H{
		"result": internal_telnyx.InboundStreamingStart,
		"params": gin.H{
			"stream_url": fmt.Sprintf("wss://%s/%s",
				tpc.appCfg.Assistant.Public,
				internal_type.GetContextAnswerPath(internal_telnyx.Provider, ctxID)),
			"stream_track": internal_telnyx.StreamTrackBoth,
		},
	})

	return nil
}

// Auth extracts the API key from vault credential.
func (tpc *telnyxTelephony) Auth(vaultCredential *protos.VaultCredential) (string, error) {
	apiKey, _, err := tpc.getCredentials(vaultCredential)
	return apiKey, err
}

// getCredentials extracts API key and connection ID from vault credential.
func (tpc *telnyxTelephony) getCredentials(vaultCredential *protos.VaultCredential) (string, string, error) {
	if vaultCredential == nil {
		return "", "", internal_telnyx.ErrVaultCredentialMissing
	}
	if vaultCredential.GetValue() == nil {
		return "", "", internal_telnyx.ErrVaultCredentialValueMissing
	}

	credMap := vaultCredential.GetValue().AsMap()

	apiKey, ok := credMap["api_key"]
	if !ok {
		return "", "", internal_telnyx.ErrVaultAPIKeyMissing
	}

	connectionID, ok := credMap["connection_id"]
	if !ok {
		return "", "", internal_telnyx.ErrVaultConnectionIDMissing
	}

	return fmt.Sprintf("%v", apiKey), fmt.Sprintf("%v", connectionID), nil
}

// HangupCall hangs up a call using Telnyx Call Control API.
func (tpc *telnyxTelephony) HangupCall(callControlID string, vaultCredential *protos.VaultCredential) error {
	apiKey, _, err := tpc.getCredentials(vaultCredential)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/calls/%s/actions/hangup", telnyxAPIBaseURL, url.PathEscape(callControlID)),
		nil)
	if err != nil {
		return err
	}

	req.Header.Set(internal_telnyx.HTTPHeaderAuthorization, "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w with status %d: %s", internal_telnyx.ErrProviderHangupFailed, resp.StatusCode, string(body))
	}

	return nil
}
