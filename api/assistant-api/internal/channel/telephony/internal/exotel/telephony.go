// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_exotel_telephony

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_exotel "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

type exotelTelephony struct {
	logger commons.Logger
	appCfg *config.AssistantConfig
}

func NewExotelTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &exotelTelephony{
		logger: logger,
		appCfg: config,
	}, nil
}

func (exo *exotelTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := ctx.Request.URL.RawQuery

	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			eventDetails[key] = values[0]
		} else {
			eventDetails[key] = nil
		}
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
	}
	if len(body) > 0 {
		rawCallbackPayload = string(body)
		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
		if strings.HasPrefix(strings.ToLower(ctx.Request.Header.Get("Content-Type")), "multipart/form-data") {
			form, err := ctx.MultipartForm()
			if err != nil {
				return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
			}
			for key, values := range form.Value {
				if len(values) > 0 {
					eventDetails[key] = values[0]
				} else {
					eventDetails[key] = nil
				}
			}
		} else {
			if err := ctx.Request.ParseForm(); err != nil {
				return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
			}
			for key, values := range ctx.Request.PostForm {
				if len(values) > 0 {
					eventDetails[key] = values[0]
				} else {
					eventDetails[key] = nil
				}
			}
		}
	}

	callback, err := internal_exotel.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		return nil, err
	}
	if !validator.NotBlank(callback.ChannelUUID) {
		return nil, internal_exotel.ErrCatchAllCallSIDMissing
	}
	return callback.StatusInfo(), nil
}

func (exo *exotelTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := c.Request.URL.RawQuery

	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			eventDetails[key] = values[0]
		} else {
			eventDetails[key] = nil
		}
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
	}
	if len(body) > 0 {
		rawCallbackPayload = string(body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		if strings.HasPrefix(strings.ToLower(c.Request.Header.Get("Content-Type")), "multipart/form-data") {
			form, err := c.MultipartForm()
			if err != nil {
				return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
			}
			for key, values := range form.Value {
				if len(values) > 0 {
					eventDetails[key] = values[0]
				} else {
					eventDetails[key] = nil
				}
			}
		} else {
			if err := c.Request.ParseForm(); err != nil {
				return nil, fmt.Errorf("%w: %w", internal_exotel.ErrCallbackFormParseFailed, err)
			}
			for key, values := range c.Request.PostForm {
				if len(values) > 0 {
					eventDetails[key] = values[0]
				} else {
					eventDetails[key] = nil
				}
			}
		}
	}
	callback, err := internal_exotel.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		return nil, err
	}
	return callback.StatusInfo(), nil
}

func (exo *exotelTelephony) ClientUrl(vaultCredential *protos.VaultCredential, opts utils.Option) (*string, error) {
	if vaultCredential.GetValue() == nil {
		return nil, internal_exotel.ErrVaultCredentialValueMissing
	}
	vaultMap := vaultCredential.GetValue().AsMap()
	accountSid, ok := vaultMap["account_sid"]
	if !ok {
		return nil, internal_exotel.ErrVaultAccountSIDMissing
	}
	clientId, ok := vaultMap["client_id"]
	if !ok {
		return nil, internal_exotel.ErrVaultClientIDMissing
	}
	authToken, ok := vaultMap["client_secret"]
	if !ok {
		return nil, internal_exotel.ErrVaultClientSecretMissing
	}
	sid, _ := accountSid.(string)
	cid, _ := clientId.(string)
	token, _ := authToken.(string)
	if sid == "" || cid == "" || token == "" {
		return nil, internal_exotel.ErrVaultCredentialInvalid
	}
	return utils.Ptr(fmt.Sprintf("https://%s:%s@api.exotel.com/v1/Accounts/%s/Calls/connect.json",
		cid, token, sid)), nil
}

func (exo *exotelTelephony) AppUrl(vaultCredential *protos.VaultCredential, opts utils.Option) (*string, error) {
	if vaultCredential.GetValue() == nil {
		return nil, internal_exotel.ErrVaultCredentialValueMissing
	}
	vaultMap := vaultCredential.GetValue().AsMap()
	accountSid, ok := vaultMap["account_sid"]
	if !ok {
		return nil, internal_exotel.ErrVaultAccountSIDMissing
	}
	app_id, err := opts.GetString("app_id")
	if err != nil {
		return nil, internal_exotel.ErrAppIDMissing
	}
	sid, _ := accountSid.(string)
	if sid == "" {
		return nil, internal_exotel.ErrVaultAccountSIDInvalid
	}
	return utils.Ptr(fmt.Sprintf("http://my.exotel.com/%s/exoml/start_voice/%s", sid, app_id)), nil
}

func (exo *exotelTelephony) OutboundCall(
	ctx context.Context,
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistant *internal_assistant_entity.Assistant, assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	statusReporter internal_type.ProviderCallStatusReporter,
	opts utils.Option) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_exotel.Provider}

	if err := ctx.Err(); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			internal_exotel.OutboundFailureReasonRequestCancelled.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
			err,
			0,
		)
		return info, err
	}

	clientUrl, err := exo.ClientUrl(vaultCredential, opts)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_exotel.OutboundFailureReasonProviderURLBuildFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	appUrl, err := exo.AppUrl(vaultCredential, opts)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_exotel.OutboundFailureReasonAppURLBuildFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")

	formData := url.Values{}
	formData.Set("From", toPhone)
	formData.Set("CallerId", fromPhone)
	formData.Set("To", fromPhone)
	formData.Set("Url", *appUrl)
	formData.Set("StatusCallback", fmt.Sprintf("https://%s/%s", exo.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_exotel.Provider, contextID)))
	formData.Set("CustomField", internal_type.GetContextAnswerPath(internal_exotel.Provider, contextID))

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", *clientUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCreation,
			internal_exotel.OutboundFailureReasonRequestCreateFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_exotel.OutboundFailureReasonProviderAPIError.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_exotel.OutboundFailureReasonResponseReadFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("status code %d: %s", resp.StatusCode, string(bodyBytes))
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		info.StatusInfo = internal_type.StatusInfo{Event: internal_type.TelephonyEventCompleted, Payload: string(bodyBytes)}
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_exotel.OutboundFailureReasonHTTPStatusFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	var jsonResponse internal_exotel.MakeCallResponse
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		info.StatusInfo = internal_type.StatusInfo{Event: internal_exotel.StatusEvent(jsonResponse.Call.Status), Payload: "Failed to decode response"}
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_exotel.OutboundFailureReasonResponseDecodeFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			resp.StatusCode,
		)
		return info, err
	}

	info.ChannelUUID = jsonResponse.Call.Sid
	info.Status = internal_type.TelephonyStatusSuccess
	info.StatusInfo = internal_type.StatusInfo{Event: internal_exotel.StatusEvent(jsonResponse.Call.Status), Payload: jsonResponse}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)
	return info, nil
}

func (exo *exotelTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)

	response := map[string]string{
		"url": fmt.Sprintf("wss://%s/%s",
			exo.appCfg.Assistant.Public,
			internal_type.GetContextAnswerPath(internal_exotel.Provider, ctxID)),
	}
	c.JSON(http.StatusOK, response)
	return nil
}

func (exo *exotelTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Exotel outbound redirect: when CustomField is present, this is the
	// callback from Exotel's outbound flow asking for the WebSocket URL.
	// Respond with the URL and return nil CallInfo so the pipeline skips
	// normal inbound call setup.
	socketUrl, ok := queryParams["CustomField"]
	if ok {
		response := map[string]string{"url": fmt.Sprintf("wss://%s/%s", exo.appCfg.Assistant.Public, socketUrl)}
		c.JSON(http.StatusOK, response)
		return nil, nil
	}

	clientNumber, ok := queryParams["CallFrom"]
	if !ok || clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid caller"})
		return nil, internal_exotel.ErrInboundFromMissing
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		Provider:     internal_exotel.Provider,
		Status:       internal_type.TelephonyStatusSuccess,
		StatusInfo:   internal_type.StatusInfo{Event: internal_exotel.WebhookEvent, Payload: queryParams},
	}
	if v, ok := queryParams["CallTo"]; ok && v != "" {
		info.FromNumber = v
	}
	if v, ok := queryParams["CallSid"]; ok && v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}
