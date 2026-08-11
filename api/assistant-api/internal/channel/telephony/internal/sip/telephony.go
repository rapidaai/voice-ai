// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_sip "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip/internal"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type sipTelephony struct {
	appCfg       *config.AssistantConfig
	logger       commons.Logger
	sharedServer *sip_infra.Server
}

func NewSIPTelephony(cfg *config.AssistantConfig, logger commons.Logger, sipServer *sip_infra.Server) (internal_type.Telephony, error) {
	return &sipTelephony{
		appCfg:       cfg,
		logger:       logger,
		sharedServer: sipServer,
	}, nil
}

func (t *sipTelephony) parseConfig(vaultCredential *protos.VaultCredential) (*sip_infra.Config, error) {
	cfg, err := sip_infra.ParseConfigFromVault(vaultCredential)
	if err != nil {
		return nil, err
	}
	if cfg.Port <= 0 {
		cfg.Port = internal_sip.DefaultOutboundSIPPort
	}
	if t.appCfg.SIPConfig != nil {
		cfg.ApplyOperationalDefaults(
			t.appCfg.SIPConfig.Port,
			sip_infra.Transport(t.appCfg.SIPConfig.Transport),
			t.appCfg.SIPConfig.RTPPortRangeStart,
			t.appCfg.SIPConfig.RTPPortRangeEnd,
		)
		cfg.ApplyTimeoutDefaults(
			t.appCfg.SIPConfig.RegisterTimeout,
			t.appCfg.SIPConfig.InviteTimeout,
			t.appCfg.SIPConfig.SessionTimeout,
		)
		cfg.ApplyMediaTimeoutDefaults(
			t.appCfg.SIPConfig.MediaTimeoutInitial,
			t.appCfg.SIPConfig.MediaTimeout,
		)
		cfg.ApplyInboundAnswerDefaults(
			sip_infra.InboundAnswerMode(t.appCfg.SIPConfig.Inbound.AnswerMode),
			t.appCfg.SIPConfig.Inbound.MinRingDuration,
			t.appCfg.SIPConfig.Inbound.MaxRingDuration,
			t.appCfg.SIPConfig.Inbound.ACKTimeout,
		)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (t *sipTelephony) StatusCallback(
	c *gin.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	assistantConversationId uint64,
) (*internal_type.StatusInfo, error) {
	payload := utils.Option{}
	rawPayload := ""
	if body, err := c.GetRawData(); err == nil && len(body) > 0 {
		rawPayload = string(body)
		if json.Unmarshal(body, &payload) != nil {
			if formValues, formErr := url.ParseQuery(rawPayload); formErr == nil {
				for k, v := range formValues {
					if len(v) == 0 {
						continue
					}
					payload[k] = v[0]
				}
			}
		}
	}
	if len(payload) == 0 {
		rawPayload = c.Request.URL.RawQuery
		for k, v := range c.Request.URL.Query() {
			if len(v) == 0 {
				continue
			}
			payload[k] = v[0]
		}
	}

	callback, err := internal_sip.NewStatusCallback(payload, rawPayload)
	if err != nil {
		return nil, err
	}

	return callback.StatusInfo(), nil
}

func (t *sipTelephony) CatchAllStatusCallback(ctx *gin.Context) (*internal_type.StatusInfo, error) {
	return nil, nil
}

func (t *sipTelephony) OutboundCall(
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
	info := &internal_type.CallInfo{Provider: internal_sip.Provider}
	cfg, err := t.parseConfig(vaultCredential)
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassConfiguration,
			internal_sip.OutboundFailureReasonInvalidConfiguration.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	contextID, _ := opts.GetString("rapida.context_id")
	fromUser := strings.TrimSpace(fromPhone)
	if t.sharedServer == nil {
		err := internal_sip.ErrSIPServerNotInitialized
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassHealthGate,
			internal_sip.OutboundFailureReasonServerNotInitialized.String(),
			internal_telephony_base.OutboundDisconnectReasonHealthGate,
			err,
			0,
		)
		return info, err
	}
	if !t.sharedServer.IsRunning() {
		err := internal_sip.ErrSIPServerNotRunning
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassHealthGate,
			internal_sip.OutboundFailureReasonServerNotRunning.String(),
			internal_telephony_base.OutboundDisconnectReasonHealthGate,
			err,
			0,
		)
		return info, err
	}

	if t.outboundHealthGateEnabled(t.appCfg) {
		healthSnapshot := t.sharedServer.HealthSnapshot()
		if !healthSnapshot.Ready {
			err := fmt.Errorf("SIP outbound health gate failed: %s", healthSnapshot.Reason)
			info.Status = internal_type.TelephonyStatusFailed
			info.ErrorMessage = err.Error()
			internal_telephony_base.ReportOutboundFailure(
				statusReporter,
				internal_telephony_base.OutboundFailureClassHealthGate,
				internal_sip.OutboundFailureReasonHealthGateFailed.String(),
				internal_telephony_base.OutboundDisconnectReasonHealthGate,
				err,
				0,
			)
			return info, err
		}
	}

	session, err := t.sharedServer.MakeCall(ctx, cfg, toPhone, fromUser, sip_infra.MakeCallOptions{
		Auth:               auth,
		Assistant:          assistant,
		ConversationID:     assistantConversationId,
		ContextID:          contextID,
		VaultCredential:    vaultCredential,
		CallStatusObserver: statusReporter,
	})
	if err != nil {
		info.Status = internal_type.TelephonyStatusFailed
		info.ErrorMessage = err.Error()
		internal_telephony_base.ReportOutboundFailure(
			statusReporter,
			internal_telephony_base.OutboundFailureClassSetup,
			internal_sip.OutboundFailureReasonSetupFailed.String(),
			internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			err,
			0,
		)
		return info, err
	}

	return &internal_type.CallInfo{
		Provider:    internal_sip.Provider,
		ChannelUUID: session.GetCallID(),
		Status:      internal_type.TelephonyStatusSuccess,
		StatusInfo: internal_type.StatusInfo{
			Event: internal_sip.StatusEvent(string(sip_infra.OutboundCallStatusInitiated)),
			Payload: map[string]interface{}{
				"to":              toPhone,
				"from":            fromUser,
				"call_id":         session.GetCallID(),
				"assistant_id":    assistant.Id,
				"conversation_id": assistantConversationId,
			},
		},
		Extra: map[string]string{
			observability.MetricCallStatus: string(sip_infra.OutboundCallStatusInitiated),
		},
	}, nil
}

func (t *sipTelephony) outboundHealthGateEnabled(appCfg *config.AssistantConfig) bool {
	if appCfg.SIPConfig.OutboundHealthGate == nil {
		return true
	}
	return *appCfg.SIPConfig.OutboundHealthGate
}

func (t *sipTelephony) InboundCall(
	c *gin.Context,
	auth types.SimplePrinciple,
	assistantId uint64,
	clientNumber string,
	assistantConversationId uint64,
) error {
	c.JSON(http.StatusOK, gin.H{
		"status":          "ready",
		"assistant_id":    assistantId,
		"conversation_id": assistantConversationId,
		"client_number":   clientNumber,
		"message":         "SIP inbound call ready - connect via SIP signaling",
	})
	return nil
}

func (t *sipTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
	clientNumber := c.Query("from")
	if clientNumber == "" {
		clientNumber = c.Query("caller")
	}
	if clientNumber == "" {
		return nil, internal_sip.ErrInboundCallerMissing
	}

	dialedNumber := c.Query("to")
	if dialedNumber == "" {
		dialedNumber = c.Query("called")
	}
	if dialedNumber == "" {
		dialedNumber = c.Query("destination")
	}

	queryParams := make(map[string]string, len(c.Request.URL.Query()))
	for key, values := range c.Request.URL.Query() {
		queryParams[key] = values[0]
	}

	info := &internal_type.CallInfo{
		CallerNumber: clientNumber,
		FromNumber:   dialedNumber,
		Provider:     internal_sip.Provider,
		Status:       internal_type.TelephonyStatusSuccess,
		StatusInfo:   internal_type.StatusInfo{Event: internal_type.TelephonyEvent(internal_sip.WebhookEvent), Payload: queryParams},
	}
	if callID := c.Query("call_id"); callID != "" {
		info.ChannelUUID = callID
	}
	return info, nil
}
