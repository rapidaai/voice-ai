// Copyright (c) 2023-2025 RapidaAI
// Author: Sarvesh Patil <sarvesh.patil@plivo.com>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_plivo_telephony

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_plivo "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/plivo/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// plivoAPIBaseURL is the Plivo REST API v1 root. The Call resource lives at
// /v1/Account/{auth_id}/Call/ and is authenticated with HTTP Basic (auth_id:auth_token).
const plivoAPIBaseURL = "https://api.plivo.com/v1"

// answerXMLPath is the context-based route that returns the Plivo answer XML.
// Plivo fetches this via answer_url to obtain the <Stream> element, which points
// it at the context WebSocket path (GetContextAnswerPath) for bidirectional media.
func answerXMLPath(contextID string) string {
	return fmt.Sprintf("v1/talk/%s/ctx/%s/answer", internal_plivo.PlivoProvider, contextID)
}

type plivoTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewPlivoTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &plivoTelephony{
		appCfg: config,
		logger: logger,
	}, nil
}

// HangupCall terminates an in-progress Plivo call via the REST API
// (DELETE /v1/Account/{auth_id}/Call/{call_uuid}/, HTTP Basic auth). It is used
// for server-initiated disconnects and the end-conversation tool action.
func (tpc *plivoTelephony) HangupCall(callUUID string, vaultCredential *protos.VaultCredential) error {
	authID, authToken, err := tpc.getCredentials(vaultCredential)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/Account/%s/Call/%s/", plivoAPIBaseURL, authID, url.PathEscape(callUUID))
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(authID, authToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w with status %d: %s", internal_plivo.ErrProviderHangupFailed, resp.StatusCode, string(body))
	}
	return nil
}

// getCredentials extracts the Plivo auth_id and auth_token from the vault credential.
func (tpc *plivoTelephony) getCredentials(vaultCredential *protos.VaultCredential) (string, string, error) {
	if vaultCredential == nil {
		return "", "", internal_plivo.ErrVaultCredentialMissing
	}
	if vaultCredential.GetValue() == nil {
		return "", "", internal_plivo.ErrVaultCredentialValueMissing
	}
	credMap := vaultCredential.GetValue().AsMap()

	authIDVal, ok := credMap["auth_id"]
	if !ok {
		return "", "", internal_plivo.ErrVaultAuthIDMissing
	}
	authTokenVal, ok := credMap["auth_token"]
	if !ok {
		return "", "", internal_plivo.ErrVaultAuthTokenMissing
	}
	authID, ok := authIDVal.(string)
	if !ok {
		return "", "", internal_plivo.ErrVaultAuthIDInvalid
	}
	authToken, ok := authTokenVal.(string)
	if !ok {
		return "", "", internal_plivo.ErrVaultAuthTokenInvalid
	}
	return authID, authToken, nil
}

// xmlAttr escapes a value for safe use inside XML attribute values and element
// text. Both URLs below embed the request-scoped contextID, so an unescaped
// quote or ampersand would otherwise malform the answer XML.
func xmlAttr(v string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(v)); err != nil {
		return ""
	}
	return b.String()
}

// CreateStreamXML builds the Plivo answer XML that opens a bidirectional mu-law
// media stream to the context WebSocket via Plivo's <Stream> element
// (bidirectional/keepCallAlive/contentType/statusCallbackUrl).
func (tpc *plivoTelephony) CreateStreamXML(mediaServer, contextID, eventCallback string) string {
	wsURL := fmt.Sprintf("wss://%s/%s", mediaServer, internal_type.GetContextAnswerPath(internal_plivo.PlivoProvider, contextID))
	return fmt.Sprintf(`<Response>
	<Stream bidirectional="true" keepCallAlive="true" contentType="audio/x-mulaw;rate=8000" statusCallbackUrl="%s" statusCallbackMethod="POST">%s</Stream>
</Response>`, xmlAttr(eventCallback), xmlAttr(wsURL))
}

// OutboundCall places an outbound call using the Plivo Call REST API.
// POST /v1/Account/{auth_id}/Call/ with from, to, answer_url (returns <Stream> XML),
// and hangup_url (status callback). Plivo's create response returns a request_uuid,
// not a CallUUID — the CallUUID arrives later on the stream "start" event and the
// status callbacks, so ChannelUUID is resolved there rather than here.
func (tpc *plivoTelephony) OutboundCall(
	ctx context.Context,
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistant *internal_assistant_entity.Assistant,
	assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	statusReporter internal_type.ProviderCallStatusReporter,
	opts utils.Option,
) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_plivo.PlivoProvider}

	if err := ctx.Err(); err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("request cancelled: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			"request cancelled",
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
			err,
			0,
		)
		return info, err
	}

	authID, authToken, err := tpc.getCredentials(vaultCredential)
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("authentication error: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassAuthentication,
			"authentication error",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")
	answerURL := fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, answerXMLPath(contextID))
	hangupURL := fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_plivo.PlivoProvider, contextID))

	callRequest := map[string]interface{}{
		"from":          fromPhone,
		"to":            toPhone,
		"answer_url":    answerURL,
		"answer_method": "GET",
		"hangup_url":    hangupURL,
		"hangup_method": "POST",
	}

	requestBody, err := json.Marshal(callRequest)
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("failed to marshal request: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestPayload,
			"failed to build provider request payload",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	endpoint := fmt.Sprintf("%s/Account/%s/Call/", plivoAPIBaseURL, authID)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("failed to create request: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCreation,
			"failed to create provider request",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(authID, authToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("API error: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			"provider API error",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("failed to read response: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			"failed to read provider response",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	var callResponse map[string]interface{}
	if err := json.Unmarshal(respBody, &callResponse); err != nil {
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("failed to parse response: %s", err.Error())
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			"failed to decode provider response",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// Plivo returns 201 Created with {message, api_id, request_uuid} on success.
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if m, ok := callResponse["error"].(string); ok && m != "" {
			errMsg = m
		} else if m, ok := callResponse["message"].(string); ok && m != "" {
			errMsg = m
		}
		info.Status = "FAILED"
		info.ErrorMessage = fmt.Sprintf("API error: %s", errMsg)
		err := fmt.Errorf("%w: %s", internal_plivo.ErrProviderAPIError, errMsg)
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			errMsg,
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// request_uuid can be a string (single call) or list (bulk); we only place one.
	requestUUID := ""
	switch v := callResponse["request_uuid"].(type) {
	case string:
		requestUUID = v
	case []interface{}:
		if len(v) > 0 {
			requestUUID, _ = v[0].(string)
		}
	}
	if !validator.NotBlank(requestUUID) {
		err := internal_plivo.ErrOutboundResponseMissingUUID
		info.Status = "FAILED"
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			"provider response missing request_uuid",
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	// The CallUUID is not known at creation — the status callback / stream "start"
	// event carries it. Persist the request_uuid for correlation until then.
	info.Status = "SUCCESS"
	info.Extra = map[string]string{"request_uuid": requestUUID}
	info.StatusInfo = internal_type.StatusInfo{Event: "initiated", Payload: callResponse}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, requestUUID)
	return info, nil
}

// InboundCall answers an incoming Plivo call by returning the <Stream> answer XML.
// Plivo posts the inbound webhook to the number's answer_url; we resolve the
// contextId set upstream and hand back the bidirectional media stream instruction.
func (tpc *plivoTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)

	eventCallback := fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_plivo.PlivoProvider, ctxID))
	c.Data(http.StatusOK, "text/xml", []byte(tpc.CreateStreamXML(tpc.appCfg.Assistant.Public, ctxID, eventCallback)))
	return nil
}

// ReceiveCall parses an inbound Plivo call webhook (form-encoded: From, To, CallUUID).
func (tpc *plivoTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	clientNumber := c.Request.FormValue("From")
	if clientNumber == "" {
		clientNumber = c.Query("From")
	}
	if clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing caller number"})
		return nil, internal_plivo.ErrInboundFromMissing
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		Provider:     internal_plivo.PlivoProvider,
		Status:       "SUCCESS",
		StatusInfo:   internal_type.StatusInfo{Event: "webhook"},
		Extra:        make(map[string]string),
	}
	if v := c.Request.FormValue("To"); v != "" {
		info.FromNumber = v // the DID that received the call (our number)
	} else if v := c.Query("To"); v != "" {
		info.FromNumber = v
	}
	if v := c.Request.FormValue("CallUUID"); v != "" {
		info.ChannelUUID = v
	} else if v := c.Query("CallUUID"); v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}

// StatusCallback parses a Plivo status/hangup webhook for a conversation.
func (tpc *plivoTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	eventDetails, rawPayload, err := tpc.readCallbackParams(c)
	if err != nil {
		return nil, err
	}
	callback, err := internal_plivo.NewStatusCallback(eventDetails, rawPayload)
	if err != nil {
		tpc.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	return callback.StatusInfo(), nil
}

// CatchAllStatusCallback parses a Plivo status webhook without a resolved conversation.
func (tpc *plivoTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails, rawPayload, err := tpc.readCallbackParams(ctx)
	if err != nil {
		return nil, err
	}
	callback, err := internal_plivo.NewStatusCallback(eventDetails, rawPayload)
	if err != nil {
		tpc.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	if !validator.NotBlank(callback.ChannelUUID) {
		tpc.logger.Errorf("call uuid not found or invalid in catch-all payload")
		return nil, internal_plivo.ErrStatusCallbackCallUUIDMissing
	}
	return callback.StatusInfo(), nil
}

// readCallbackParams collects Plivo webhook fields from the query string and,
// when present, the form-encoded body (Plivo posts hangup/answer callbacks as
// application/x-www-form-urlencoded).
func (tpc *plivoTelephony) readCallbackParams(c *gin.Context) (utils.Option, string, error) {
	eventDetails := utils.Option{}
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			eventDetails[key] = values[0]
		}
	}
	rawPayload := c.Request.URL.RawQuery
	if err := c.Request.ParseForm(); err != nil {
		tpc.logger.Warnf("failed to parse Plivo callback form body: %+v", err)
	} else {
		for key, values := range c.Request.PostForm {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			}
		}
		if c.Request.PostForm != nil && len(c.Request.PostForm) > 0 {
			rawPayload = c.Request.PostForm.Encode()
		}
	}
	return eventDetails, rawPayload, nil
}
