// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage_telephony

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_vonage "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vonage/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"github.com/vonage/vonage-go-sdk"
	"github.com/vonage/vonage-go-sdk/ncco"
)

type vonageTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewVonageTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &vonageTelephony{
		logger: logger,
		appCfg: config,
	}, nil
}

func vonageAuth(vaultCredential *protos.VaultCredential) (vonage.Auth, error) {
	if vaultCredential.GetValue() == nil {
		return nil, internal_vonage.ErrVaultCredentialValueMissing
	}
	vaultMap := vaultCredential.GetValue().AsMap()
	privateKey, ok := vaultMap["private_key"]
	if !ok {
		return nil, internal_vonage.ErrVaultPrivateKeyMissing
	}
	applicationId, ok := vaultMap["application_id"]
	if !ok {
		return nil, internal_vonage.ErrVaultApplicationIDMissing
	}
	pk, ok := privateKey.(string)
	if !ok {
		return nil, internal_vonage.ErrVaultPrivateKeyInvalid
	}
	appID, ok := applicationId.(string)
	if !ok {
		return nil, internal_vonage.ErrVaultApplicationIDInvalid
	}
	clientAuth, err := vonage.CreateAuthFromAppPrivateKey(appID, []byte(pk))
	if err != nil {
		return nil, err
	}
	return clientAuth, nil
}

func (vng *vonageTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	eventDetails := utils.Option{}
	rawCallbackPayload := ctx.Request.URL.RawQuery
	for key, values := range ctx.Request.URL.Query() {
		if len(values) > 0 {
			eventDetails[key] = values[0]
		} else {
			eventDetails[key] = nil
		}
	}

	callback, err := internal_vonage.NewStatusCallback(eventDetails, rawCallbackPayload)
	if err != nil {
		vng.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	if !validator.NotBlank(callback.ChannelUUID) {
		vng.logger.Errorf("uuid not found or invalid in catch-all payload")
		return nil, internal_vonage.ErrCatchAllChannelUUIDMissing
	}

	return callback.StatusInfo(), nil
}

func (vng *vonageTelephony) StatusCallback(c *gin.Context, auth *types.Authentication, assistantId uint64, assistantConversationId uint64) (*internal_type.StatusInfo, error) {
	var payload utils.Option
	rawCallbackPayload := c.Request.URL.RawQuery
	if len(c.Request.URL.Query()) > 0 {
		payload = utils.Option{}
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				payload[key] = values[0]
			} else {
				payload[key] = nil
			}
		}
	} else {
		body, err := c.GetRawData()
		if err != nil {
			vng.logger.Errorf("failed to read request body with error %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_vonage.ErrRequestBodyReadFailed, err)
		}
		rawCallbackPayload = string(body)
		if err := json.Unmarshal(body, &payload); err != nil {
			vng.logger.Errorf("failed to parse request body: %+v", err)
			return nil, fmt.Errorf("%w: %w", internal_vonage.ErrRequestBodyParseFailed, err)
		}
	}

	callback, err := internal_vonage.NewStatusCallback(payload, rawCallbackPayload)
	if err != nil {
		vng.logger.Errorf("failed to parse status callback: %+v", err)
		return nil, err
	}
	vng.logger.Debugf("event processed | status: %s, payload: %+v", callback.Status, payload)
	return callback.StatusInfo(), nil
}

func (vng *vonageTelephony) OutboundCall(
	ctx context.Context,
	auth *types.Authentication,
	toPhone string,
	fromPhone string,
	assistant *internal_assistant_entity.Assistant, assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	statusReporter internal_type.ProviderCallStatusReporter,
	opts utils.Option,
) (*internal_type.CallInfo, error) {
	info := &internal_type.CallInfo{Provider: internal_vonage.Provider}

	if err := ctx.Err(); err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassRequestCancelled,
			internal_vonage.OutboundFailureReasonRequestCancelled.String(),
			internal_telephony_base.OutboundDisconnectReasonRequestCancelled,
			err,
			0,
		)
		return info, err
	}

	cAuth, err := vonageAuth(vaultCredential)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassAuthentication,
			internal_vonage.OutboundFailureReasonAuthenticationFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}
	ct := vonage.NewVoiceClient(cAuth)

	contextID, _ := opts.GetString("rapida.context_id")

	connectAction := ncco.Ncco{}
	nccoConnect := ncco.ConnectAction{
		EventType: internal_vonage.NCCOEventTypeSync,
		EventUrl:  []string{fmt.Sprintf("https://%s/%s", vng.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_vonage.Provider, contextID))},
		Endpoint: []ncco.Endpoint{ncco.WebSocketEndpoint{
			Uri: fmt.Sprintf("wss://%s/%s",
				vng.appCfg.Assistant.Public,
				internal_type.GetContextAnswerPath(internal_vonage.Provider, contextID)),
			ContentType: internal_vonage.WebSocketContentType,
		}},
	}
	connectAction.AddAction(nccoConnect)
	result, vErr, apiError := ct.CreateCall(
		vonage.CreateCallOpts{
			From:        vonage.CallFrom{Type: "phone", Number: fromPhone},
			To:          vonage.CallTo{Type: "phone", Number: toPhone},
			Ncco:        connectAction,
			EventUrl:    []string{fmt.Sprintf("https://%s/%s", vng.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_vonage.Provider, contextID))},
			EventMethod: "GET",
		})

	if apiError != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = apiError.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderAPI,
			internal_vonage.OutboundFailureReasonProviderAPIError.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			apiError,
			0,
		)
		return info, apiError
	}

	if vErr.Error != nil {
		err := internal_vonage.ErrProviderCallCreateFailed
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = fmt.Sprint(vErr.Error)
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassProviderResponse,
			internal_vonage.OutboundFailureReasonProviderCallCreate.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	info.ChannelUUID = result.Uuid
	info.Status = internal_type.TelephonyStatusSuccess
	info.StatusInfo = internal_type.StatusInfo{Event: internal_vonage.StatusEvent(result.Status), Payload: result}
	info.Extra = map[string]string{
		"conversation_uuid": result.ConversationUuid,
	}
	internal_telephony_base.ReportOutboundInitiated(statusReporter, info.ChannelUUID)
	return info, nil
}

func (vng *vonageTelephony) InboundCall(c *gin.Context, auth *types.Authentication, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	contextID, _ := c.Get("contextId")
	ctxID := fmt.Sprintf("%v", contextID)

	c.JSON(http.StatusOK, []gin.H{
		{
			"action":    "connect",
			"eventType": internal_vonage.NCCOEventTypeSync,
			"eventUrl":  []string{fmt.Sprintf("https://%s/%s", vng.appCfg.Assistant.Public, internal_type.GetContextEventPath(internal_vonage.Provider, ctxID))},
			"endpoint": []gin.H{
				{
					"type": internal_vonage.WebSocketEndpointType,
					"uri": fmt.Sprintf("wss://%s/%s",
						vng.appCfg.Assistant.Public,
						internal_type.GetContextAnswerPath(internal_vonage.Provider, ctxID)),
					"content-type": internal_vonage.WebSocketContentType,
				},
			},
		},
	})
	return nil
}

func (vng *vonageTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["from"]
	if !ok || clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return nil, internal_vonage.ErrInboundFromMissing
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		Provider:     internal_vonage.Provider,
		Status:       internal_type.TelephonyStatusSuccess,
		StatusInfo:   internal_type.StatusInfo{Event: internal_vonage.WebhookEvent, Payload: queryParams},
		Extra:        make(map[string]string),
	}

	if v, ok := queryParams["to"]; ok && v != "" {
		info.FromNumber = v
	}
	if v, ok := queryParams["conversation_uuid"]; ok && v != "" {
		info.Extra["conversation_uuid"] = v
	}
	if v, ok := queryParams["uuid"]; ok && v != "" {
		info.ChannelUUID = v
	}
	return info, nil
}
