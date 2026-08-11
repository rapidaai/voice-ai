// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz_telephony

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_vobiz "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

const (
	vobizCallReceiverPathFmt     = "v1/talk/%s/call/%d"
	vobizCustomFieldParam        = "CustomField"
	vobizStatusCallbackPathParam = "StatusCallback"
)

type vobizTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewVobizTelephony(cfg *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &vobizTelephony{appCfg: cfg, logger: logger}, nil
}

// AnswerXML returns the <Stream> XML that points vobiz at our WebSocket.
func (v *vobizTelephony) AnswerXML(streamPath, statusCallbackPath string) (string, error) {
	if !validator.NotBlank(streamPath) {
		return "", fmt.Errorf("invalid stream path %q", streamPath)
	}
	if !validator.NotBlank(statusCallbackPath) {
		return "", fmt.Errorf("invalid status callback path %q", statusCallbackPath)
	}
	return fmt.Sprintf(`<Response><Stream bidirectional="true" audioTrack="inbound" contentType="audio/x-mulaw;rate=8000" keepCallAlive="true" statusCallbackUrl="%s">%s</Stream></Response>`, fmt.Sprintf("https://%s/%s", v.appCfg.Assistant.Public, statusCallbackPath), fmt.Sprintf("wss://%s/%s", v.appCfg.Assistant.Public, streamPath)), nil
}

func (v *vobizTelephony) OutboundCall(ctx context.Context, auth types.SimplePrinciple, toPhone string, fromPhone string, assistant *internal_assistant_entity.Assistant, assistantConversationId uint64, vaultCredential *protos.VaultCredential, statusReporter internal_type.ProviderCallStatusReporter, opts utils.Option) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_vobiz.VobizProvider}
	if err := ctx.Err(); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			internal_vobiz.OutboundFailureReasonRequestCancelled.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled, err, 0)
		return info, err
	}

	if !validator.NonNil(vaultCredential.GetValue()) {
		err := internal_vobiz.ErrVaultCredentialValueMissing
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_vobiz.OutboundFailureReasonMissingVaultCredential.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled, err, 0)
		return info, err
	}

	vobizCredential := vaultCredential.GetValue().AsMap()
	authID, ok := vobizCredential["auth_id"].(string)
	if !ok {
		err := internal_vobiz.ErrVaultAuthIDMissing
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_vobiz.OutboundFailureReasonAuthIDMissing.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}
	authToken, ok := vobizCredential["auth_token"].(string)
	if !ok {
		err := internal_vobiz.ErrVaultAuthTokenMissing
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_vobiz.OutboundFailureReasonAuthTokenMissing.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")
	if contextID == "" {
		err := fmt.Errorf("missing rapida.context_id; cannot build answer/event callback URLs")
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_vobiz.OutboundFailureReasonContextIDMissing.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}

	answerQuery := url.Values{}
	answerQuery.Set(vobizCustomFieldParam, internal_type.GetContextAnswerPath(internal_vobiz.VobizProvider, contextID))
	answerQuery.Set(vobizStatusCallbackPathParam, internal_type.GetContextEventPath(internal_vobiz.VobizProvider, contextID))
	answerPath := fmt.Sprintf(vobizCallReceiverPathFmt, internal_vobiz.VobizProvider, assistant.Id)
	resp, err := internal_vobiz.New().MakeCall(ctx, authID, authToken, internal_vobiz.MakeCallRequest{
		From:         fromPhone,
		To:           toPhone,
		AnswerURL:    fmt.Sprintf("https://%s/%s?%s", v.appCfg.Assistant.Public, answerPath, answerQuery.Encode()),
		AnswerMethod: "POST",
		RingURL:      fmt.Sprintf("https://%s/%s", v.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_vobiz.VobizProvider, contextID)),
		RingMethod:   "POST",
		HangupURL:    fmt.Sprintf("https://%s/%s", v.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_vobiz.VobizProvider, contextID)),
		HangupMethod: "POST",
	})
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_vobiz.OutboundFailureReasonProviderAPIError.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}
	if resp.RequestUUID == "" {
		err := internal_vobiz.ErrOutboundResponseMissingUUID
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_vobiz.OutboundFailureReasonResponseMissingRequestUUID.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed, err, 0)
		return info, err
	}

	info.ChannelUUID = resp.RequestUUID
	info.Status = internal_type.TelephonyStatusSuccess
	info.StatusInfo = internal_type.StatusInfo{Event: internal_type.TelephonyEvent(resp.Message), Payload: resp}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)
	return info, nil
}

// InboundCall answers an incoming call by returning the <Stream> XML.
func (v *vobizTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)
	xml, err := v.AnswerXML(
		internal_type.GetContextAnswerPath(internal_vobiz.VobizProvider, ctxID),
		internal_type.GetContextEventPath(internal_vobiz.VobizProvider, ctxID),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build answer"})
		return err
	}
	c.Data(http.StatusOK, "text/xml", []byte(xml))
	return nil
}

func (v *vobizTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	if c.Request.Body != nil {
		body, err := c.GetRawData()
		if err != nil {
			v.logger.Errorf("failed to read vobiz incoming call payload: %+v", err)
		} else {
			values, err := url.ParseQuery(string(body))
			if err != nil {
				v.logger.Errorf("failed to parse vobiz incoming call payload: %+v", err)
			} else {
				for key, values := range values {
					if len(values) > 0 {
						queryParams[key] = values[0]
					}
				}
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}
	}
	streamPath, ok := queryParams[vobizCustomFieldParam]
	if ok {
		statusCallbackPath := queryParams[vobizStatusCallbackPathParam]
		xml, err := v.AnswerXML(streamPath, statusCallbackPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build answer"})
			return nil, err
		}
		c.Data(http.StatusOK, "text/xml", []byte(xml))
		return nil, nil
	}
	info := &internal_type.CallInfo{
		Provider:   internal_vobiz.VobizProvider,
		Status:     internal_type.TelephonyStatusSuccess,
		StatusInfo: internal_type.StatusInfo{Event: internal_type.TelephonyEvent(internal_vobiz.WebhookEvent), Payload: queryParams},
	}
	if v := queryParams["From"]; v != "" {
		info.CallerNumber = v
	}
	if v := queryParams["To"]; v != "" {
		info.FromNumber = v
	}
	if v := queryParams["CallUUID"]; v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}

func (v *vobizTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := c.Request.URL.RawQuery
	if len(c.Request.URL.Query()) > 0 {
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
	} else {
		body, err := c.GetRawData()
		if err != nil {
			v.logger.Errorf("failed to read callback body with error %+v", err)
			return nil, err
		}
		rawCallbackPayload = string(body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			v.logger.Errorf("failed to parse callback body with error %+v", err)
			return nil, err
		}
		for key, values := range values {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
	}

	callback, err := internal_vobiz.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		v.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	return callback.StatusInfo(), nil
}

func (v *vobizTelephony) CatchAllStatusCallback(c *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := c.Request.URL.RawQuery
	if len(c.Request.URL.Query()) > 0 {
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
	} else {
		body, err := c.GetRawData()
		if err != nil {
			v.logger.Errorf("failed to read callback body with error %+v", err)
			return nil, err
		}
		rawCallbackPayload = string(body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			v.logger.Errorf("failed to parse callback body with error %+v", err)
			return nil, err
		}
		for key, values := range values {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
	}

	callback, err := internal_vobiz.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		v.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	if !validator.NotBlank(callback.ChannelUUID) {
		v.logger.Errorf("call uuid not found or invalid in catch-all payload")
		return nil, internal_vobiz.ErrCatchAllChannelUUIDMissing
	}
	return callback.StatusInfo(), nil
}
