// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio_telephony

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_twilio "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type twilioTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

type twimlResponse struct {
	XMLName xml.Name     `xml:"Response"`
	Connect twimlConnect `xml:"Connect"`
}

type twimlConnect struct {
	Stream twimlStream `xml:"Stream"`
}

type twimlStream struct {
	URL                 string           `xml:"url,attr"`
	Name                string           `xml:"name,attr"`
	StatusCallback      string           `xml:"statusCallback,attr"`
	StatusCallbackEvent string           `xml:"statusCallbackEvent,attr"`
	Parameters          []twimlParameter `xml:"Parameter"`
}

type twimlParameter struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func NewTwilioTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &twilioTelephony{
		appCfg: config,
		logger: logger,
	}, nil
}

func twilioClient(vaultCredential *protos.VaultCredential) (*twilio.RestClient, error) {
	clientParams, err := twilioClientParams(vaultCredential)
	if err != nil {
		return nil, err
	}
	return twilio.NewRestClientWithParams(*clientParams), nil
}

func twilioClientParams(vaultCredential *protos.VaultCredential) (*twilio.ClientParams, error) {
	if vaultCredential.GetValue() == nil {
		return nil, internal_twilio.ErrVaultCredentialValueMissing
	}
	vaultMap := vaultCredential.GetValue().AsMap()
	accountSid, ok := vaultMap["account_sid"]
	if !ok {
		return nil, internal_twilio.ErrVaultAccountSIDMissing
	}
	authToken, ok := vaultMap["account_token"]
	if !ok {
		return nil, internal_twilio.ErrVaultAccountTokenMissing
	}
	sid, ok := accountSid.(string)
	if !ok {
		return nil, internal_twilio.ErrVaultAccountSIDInvalid
	}
	token, ok := authToken.(string)
	if !ok {
		return nil, internal_twilio.ErrVaultAccountTokenInvalid
	}
	return &twilio.ClientParams{
		Username: sid,
		Password: token,
	}, nil
}

func (tpc *twilioTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := ctx.Request.URL.RawQuery
	if len(ctx.Request.URL.Query()) > 0 {
		for key, values := range ctx.Request.URL.Query() {
			if len(values) > 0 {
				eventDetails[key] = values[0]
			} else {
				eventDetails[key] = nil
			}
		}
	} else {
		body, err := ctx.GetRawData()
		if err != nil {
			tpc.logger.Errorf("failed to read event body with error %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_twilio.ErrRequestBodyReadFailed, err)
		}
		rawCallbackPayload = string(body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			tpc.logger.Errorf("failed to parse body with error %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_twilio.ErrRequestBodyParseFailed, err)
		}
		for key, value := range values {
			if len(value) > 0 {
				eventDetails[key] = value[0]
			} else {
				eventDetails[key] = nil
			}
		}
	}

	callback, err := internal_twilio.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		tpc.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	if !validator.NotBlank(callback.ChannelUUID) {
		tpc.logger.Errorf("call sid not found or invalid in catch-all payload")
		return nil, internal_twilio.ErrStatusCallbackCallSIDMissing
	}
	return callback.StatusInfo(), nil
}

func (tpc *twilioTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
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
			tpc.logger.Errorf("failed to read event body with error %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_twilio.ErrRequestBodyReadFailed, err)
		}
		rawCallbackPayload = string(body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			tpc.logger.Errorf("failed to parse body with error %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_twilio.ErrRequestBodyParseFailed, err)
		}
		for key, value := range values {
			if len(value) > 0 {
				eventDetails[key] = value[0]
			} else {
				eventDetails[key] = nil
			}
		}
	}

	callback, err := internal_twilio.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		tpc.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	return callback.StatusInfo(), nil
}

func (tpc *twilioTelephony) OutboundCall(ctx context.Context,
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistant *internal_assistant_entity.Assistant,
	assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	statusReporter internal_type.ProviderCallStatusReporter,
	opts utils.Option) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_twilio.TwilioProvider}
	if err := ctx.Err(); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			internal_twilio.OutboundFailureReasonRequestCancelled.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
			err,
			0,
		)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")
	client, err := twilioClient(vaultCredential)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassAuthentication,
			internal_twilio.OutboundFailureReasonAuthenticationFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	callParams := &openapi.CreateCallParams{}
	callParams.SetTo(toPhone)
	callParams.SetFrom(fromPhone)
	callParams.SetStatusCallback(
		fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_twilio.TwilioProvider, contextID)),
	)
	callParams.SetStatusCallbackEvent([]string{
		"initiated", "ringing", "answered", "completed",
	})
	callParams.SetStatusCallbackMethod("POST")
	callParams.SetTwiml(
		tpc.CreateTwinML(
			tpc.appCfg.Assistant.Public,
			fmt.Sprintf("%d__%d", assistant.Id, assistantConversationId),
			internal_type.GetContextAnswerPath(internal_twilio.TwilioProvider, contextID),
			fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_twilio.TwilioProvider, contextID)),
			assistant.Id,
			toPhone),
	)
	resp, err := client.Api.CreateCall(callParams)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_twilio.OutboundFailureReasonProviderAPIError.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	if resp.Status == nil || resp.Sid == nil {
		err := internal_twilio.ErrOutboundResponseMissingStatusSID
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_twilio.OutboundFailureReasonResponseMissingStatusOrSID.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	info.ChannelUUID = *resp.Sid
	info.Status = internal_type.TelephonyStatusSuccess
	info.StatusInfo = internal_type.StatusInfo{Event: internal_twilio.StatusEvent(*resp.Status), Payload: resp}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)
	return info, nil
}

func (tpc *twilioTelephony) CreateTwinML(mediaServer string, name, path string, callback string, assistantId uint64, clientNumber string) string {
	payload, err := xml.Marshal(twimlResponse{
		Connect: twimlConnect{
			Stream: twimlStream{
				URL:                 fmt.Sprintf("wss://%s/%s", mediaServer, path),
				Name:                name,
				StatusCallback:      callback,
				StatusCallbackEvent: "initiated ringing answered completed",
				Parameters: []twimlParameter{
					{Name: "assistant_id", Value: fmt.Sprintf("%d", assistantId)},
					{Name: "client_number", Value: clientNumber},
				},
			},
		},
	})
	if err != nil {
		tpc.logger.Errorf("failed to marshal TwiML with error %+v", err)
		return ""
	}
	return string(payload)
}

func (tpc *twilioTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)

	c.Data(http.StatusOK, "text/xml", []byte(
		tpc.CreateTwinML(
			tpc.appCfg.Assistant.Public,
			fmt.Sprintf("%d__%d", assistantId, assistantConversationId),
			internal_type.GetContextAnswerPath("twilio", ctxID),
			fmt.Sprintf("https://%s/%s", tpc.appCfg.Assistant.Public, internal_type.GetContextEventPath("twilio", ctxID)),
			assistantId, clientNumber),
	))
	return nil
}

func (tpc *twilioTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["From"]
	if !ok || clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return nil, internal_twilio.ErrInboundFromMissing
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		Provider:     internal_twilio.TwilioProvider,
		Status:       internal_type.TelephonyStatusSuccess,
		StatusInfo:   internal_type.StatusInfo{Event: "webhook", Payload: queryParams},
	}
	if v, ok := queryParams["To"]; ok && v != "" {
		info.FromNumber = v // DID that received the call (our number)
	}
	if v, ok := queryParams["CallSid"]; ok && v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}
