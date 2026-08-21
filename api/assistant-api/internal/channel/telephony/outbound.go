// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
)

const defaultOutboundConnectTimeout = 2 * time.Minute

type OutboundDispatcherOptions struct {
	Config               *config.AssistantConfig
	Logger               commons.Logger
	Store                callcontext.Store
	VaultClient          web_client.VaultClient
	AssistantService     internal_services.AssistantService
	ConversationService  internal_services.AssistantConversationService
	ConfigurationService internal_services.AssistantConfigurationService
	HTTPLogService       internal_services.AssistantHTTPLogService
	TelephonyOption      TelephonyOption
}

type OutboundDispatcherFuncOption func(*OutboundDispatcherOptions)

func WithOutboundConfig(cfg *config.AssistantConfig) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.Config = cfg
	}
}

func WithOutboundLogger(logger commons.Logger) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.Logger = logger
	}
}

func WithOutboundStore(store callcontext.Store) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.Store = store
	}
}

func WithOutboundVaultClient(vaultClient web_client.VaultClient) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.VaultClient = vaultClient
	}
}

func WithOutboundAssistantService(assistantService internal_services.AssistantService) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.AssistantService = assistantService
	}
}

func WithOutboundConversationService(conversationService internal_services.AssistantConversationService) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.ConversationService = conversationService
	}
}

func WithOutboundAssistantConfigurationService(configurationService internal_services.AssistantConfigurationService) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.ConfigurationService = configurationService
	}
}

func WithOutboundHTTPLogService(httpLogService internal_services.AssistantHTTPLogService) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.HTTPLogService = httpLogService
	}
}

func WithOutboundTelephonyOption(telephonyOpt TelephonyOption) OutboundDispatcherFuncOption {
	return func(options *OutboundDispatcherOptions) {
		options.TelephonyOption = telephonyOpt
	}
}

type OutboundDispatcher struct {
	cfg                    *config.AssistantConfig
	store                  callcontext.Store
	logger                 commons.Logger
	vaultClient            web_client.VaultClient
	assistantService       internal_services.AssistantService
	conversationService    internal_services.AssistantConversationService
	configurationService   internal_services.AssistantConfigurationService
	httpLogService         internal_services.AssistantHTTPLogService
	telephonyOpt           TelephonyOption
	outboundConnectTimeout time.Duration
}

func NewOutboundDispatcher(opts ...OutboundDispatcherFuncOption) *OutboundDispatcher {
	var options OutboundDispatcherOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &OutboundDispatcher{
		cfg:                    options.Config,
		store:                  options.Store,
		logger:                 options.Logger,
		vaultClient:            options.VaultClient,
		assistantService:       options.AssistantService,
		conversationService:    options.ConversationService,
		configurationService:   options.ConfigurationService,
		httpLogService:         options.HTTPLogService,
		telephonyOpt:           options.TelephonyOption,
		outboundConnectTimeout: defaultOutboundConnectTimeout,
	}
}

func (d *OutboundDispatcher) Dispatch(ctx context.Context, contextID string) (*internal_type.CallInfo, error) {
	callContext, err := d.store.Get(ctx, contextID)
	if err != nil {
		d.logger.Errorf("outbound dispatcher: failed to resolve call context %s: %v", contextID, err)
		return nil, err
	}

	d.logger.Infof("outbound dispatcher[%s]: processing contextId=%s, assistant=%d, conversation=%d",
		callContext.Provider, callContext.ContextID, callContext.AssistantID, callContext.ConversationID)

	callInfo, err := d.performOutbound(ctx, callContext)
	if err != nil {
		d.logger.Errorf("outbound dispatcher[%s]: call failed for contextId=%s: %v", callContext.Provider, contextID, err)
		currentCallContext, getCallContextErr := d.store.Get(ctx, callContext.ContextID)
		if getCallContextErr != nil ||
			(currentCallContext.Status != callcontext.StatusFailed &&
				currentCallContext.Status != callcontext.StatusCompleted &&
				currentCallContext.CallStatus != callcontext.CallStatusFailed &&
				currentCallContext.CallStatus != callcontext.CallStatusCancelled) {
			if updateCallStatusErr := d.store.UpdateCallStatus(ctx, callContext.ContextID, callcontext.CallStatusUpdate{
				CallStatus:       callcontext.CallStatusFailed,
				CallError:        err.Error(),
				FailureClass:     internal_telephony_base.OutboundFailureClassSetup,
				FailureReason:    "outbound setup failed",
				DisconnectReason: internal_telephony_base.OutboundDisconnectReasonSetupFailed,
			}); updateCallStatusErr != nil {
				d.logger.Warnw("Failed to persist outbound setup failure status",
					"contextId", callContext.ContextID,
					"call_status", callcontext.CallStatusFailed,
					"failure_class", internal_telephony_base.OutboundFailureClassSetup,
					"error", updateCallStatusErr)
			}
		}
		return callInfo, err
	}

	d.logger.Infof("outbound dispatcher[%s]: call initiated for contextId=%s", callContext.Provider, contextID)

	// The answer monitor must outlive the API request that initiated the call.
	go d.monitorCallConnect(context.WithoutCancel(ctx), contextID, callContext)
	return callInfo, nil
}

// monitorCallConnect marks unclaimed outbound calls as no-answer after the provider timeout.
func (d *OutboundDispatcher) monitorCallConnect(ctx context.Context, contextID string, initialCallContext *callcontext.CallContext) {
	providerConnectTimeout := d.providerOutboundConnectTimeout(initialCallContext.Provider)
	select {
	case <-ctx.Done():
		return
	case <-time.After(providerConnectTimeout):
	}

	currentCallContext, err := d.store.Get(ctx, contextID)
	if err != nil {
		return
	}
	if currentCallContext.Status != callcontext.StatusPending {
		return // Already claimed or processed
	}
	if currentCallContext.CallStatus != callcontext.CallStatusNew {
		return // Provider callback already moved call_status forward
	}

	d.NewStatusReporter(currentCallContext.ContextID)(internal_type.ProviderCallStatusUpdate{
		ExpectedCallStatus: callcontext.CallStatusNew,
		CallStatus:         callcontext.CallStatusFailed,
		ErrorMessage:       "Provider callback was not received before outbound connect timeout: " + providerConnectTimeout.String(),
		FailureClass:       internal_telephony_base.OutboundFailureClassNoAnswer,
		FailureReason:      internal_telephony_base.OutboundFailureReasonNoAnswer,
		DisconnectReason:   internal_telephony_base.OutboundDisconnectReasonNoAnswer,
		Retryable:          true,
	})
}

func (d *OutboundDispatcher) providerOutboundConnectTimeout(provider string) time.Duration {
	timeout := d.outboundConnectTimeout
	if timeout <= 0 {
		timeout = defaultOutboundConnectTimeout
	}
	if provider == SIP.String() && d.cfg != nil && d.cfg.SIPConfig != nil && d.cfg.SIPConfig.InviteTimeout > 0 {
		return d.cfg.SIPConfig.InviteTimeout + 15*time.Second
	}
	return timeout
}

func (d *OutboundDispatcher) performOutbound(ctx context.Context, callContext *callcontext.CallContext) (*internal_type.CallInfo, error) {
	telephonyProvider, err := GetTelephony(Telephony(callContext.Provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return nil, fmt.Errorf("telephony provider %s not available: %w", callContext.Provider, err)
	}

	callAuth, err := callContext.ToAuthentication()
	if err != nil {
		return nil, fmt.Errorf("invalid call context authentication: %w", err)
	}

	assistant, err := d.assistantService.Get(ctx, callAuth, callContext.AssistantID, nil, &internal_services.GetAssistantOption{InjectPhoneDeployment: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load assistant %d: %w", callContext.AssistantID, err)
	}
	if !assistant.IsPhoneDeploymentEnable() {
		return nil, fmt.Errorf("phone deployment not enabled for assistant %d", callContext.AssistantID)
	}

	credentialID, err := assistant.AssistantPhoneDeployment.GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get credential ID: %w", err)
	}

	vaultCredential, err := d.vaultClient.GetCredential(ctx, callAuth, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault credential: %w", err)
	}

	phoneDeploymentOptions := assistant.AssistantPhoneDeployment.GetOptions()
	phoneDeploymentOptions["rapida.context_id"] = callContext.ContextID

	callInfo, outboundCallErr := telephonyProvider.OutboundCall(ctx, callAuth, callContext.CallerNumber, callContext.FromNumber, assistant, callContext.ConversationID, vaultCredential, d.NewStatusReporter(callContext.ContextID), phoneDeploymentOptions)
	if outboundCallErr != nil {
		d.logger.Errorf("outbound dispatcher[%s]: telephony call failed for contextId=%s: %v", callContext.Provider, callContext.ContextID, outboundCallErr)
	}
	if callInfo == nil {
		return nil, outboundCallErr
	}

	if callInfo.ChannelUUID != "" {
		if updateErr := d.store.UpdateField(ctx, callContext.ContextID, "channel_uuid", callInfo.ChannelUUID); updateErr != nil {
			d.logger.Warnf("outbound dispatcher[%s]: failed to store channel UUID: %v", callContext.Provider, updateErr)
		}
	}

	return callInfo, outboundCallErr
}
