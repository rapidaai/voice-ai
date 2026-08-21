// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type InboundDispatcherOptions struct {
	Config              *config.AssistantConfig
	Logger              commons.Logger
	Store               callcontext.Store
	VaultClient         web_client.VaultClient
	AssistantService    internal_services.AssistantService
	ConversationService internal_services.AssistantConversationService
	TelephonyOption     TelephonyOption
}

type InboundDispatcherFuncOption func(*InboundDispatcherOptions)

func WithConfig(cfg *config.AssistantConfig) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.Config = cfg
	}
}

func WithLogger(logger commons.Logger) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.Logger = logger
	}
}

func WithStore(store callcontext.Store) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.Store = store
	}
}

func WithVaultClient(vaultClient web_client.VaultClient) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.VaultClient = vaultClient
	}
}

func WithAssistantService(assistantService internal_services.AssistantService) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.AssistantService = assistantService
	}
}

func WithConversationService(conversationService internal_services.AssistantConversationService) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.ConversationService = conversationService
	}
}

func WithTelephonyOption(telephonyOpt TelephonyOption) InboundDispatcherFuncOption {
	return func(options *InboundDispatcherOptions) {
		options.TelephonyOption = telephonyOpt
	}
}

// InboundDispatcher handles inbound call processing across all telephony
// channels (SIP, Asterisk, Twilio, Exotel, Vonage). It encapsulates the
// common business logic: provider resolution, call reception, conversation
// creation, call-context persistence, telemetry application, and session resolution.
type InboundDispatcher struct {
	cfg                 *config.AssistantConfig
	store               callcontext.Store
	logger              commons.Logger
	vaultClient         web_client.VaultClient
	assistantService    internal_services.AssistantService
	conversationService internal_services.AssistantConversationService
	telephonyOpt        TelephonyOption
}

// NewInboundDispatcher creates a new inbound call dispatcher.
func NewInboundDispatcher(opts ...InboundDispatcherFuncOption) *InboundDispatcher {
	var options InboundDispatcherOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &InboundDispatcher{
		cfg:                 options.Config,
		store:               options.Store,
		logger:              options.Logger,
		vaultClient:         options.VaultClient,
		assistantService:    options.AssistantService,
		conversationService: options.ConversationService,
		telephonyOpt:        options.TelephonyOption,
	}
}

// ResolveVaultCredential fetches the vault credential for the given assistant.
// This is the only DB round-trip needed — call IDs (assistant, conversation,
// provider) are already in the CallContext from Redis.
func (d *InboundDispatcher) ResolveVaultCredential(ctx context.Context, auth types.SimplePrinciple, assistantId, conversationId uint64) (*protos.VaultCredential, error) {
	assistant, err := d.assistantService.Get(ctx, auth, assistantId, nil, &internal_services.GetAssistantOption{InjectPhoneDeployment: true})
	if err != nil {
		return nil, err
	}
	if !assistant.IsPhoneDeploymentEnable() {
		return nil, fmt.Errorf("phone deployment not enabled for assistant %d", assistantId)
	}
	credentialID, err := assistant.AssistantPhoneDeployment.GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		return nil, err
	}
	vltC, err := d.vaultClient.GetCredential(ctx, auth, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve vault credential: %w", err)
	}
	return vltC, nil
}

// ResolveCallSessionByContext resolves a call context and vault credential using
// a contextId stored in Postgres. The call context is atomically claimed by
// transitioning its status from "pending" to "claimed". Only one media connection
// can claim a given context — subsequent callers get an error.
// The context remains in Postgres so that event/status callbacks can still read it.
// Returns the CallContext (which contains all IDs and auth info) plus the vault
// credential needed for the streamer.
func (d *InboundDispatcher) ResolveCallSessionByContext(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
	cc, err := d.store.Claim(ctx, contextID)
	if err != nil {
		d.logger.Errorf("failed to resolve call context %s: %v", contextID, err)
		return nil, nil, fmt.Errorf("call context not found or already claimed: %w", err)
	}

	auth := cc.ToAuth()
	vaultCred, err := d.ResolveVaultCredential(ctx, auth, cc.AssistantID, cc.ConversationID)
	if err != nil {
		return nil, nil, err
	}
	return cc, vaultCred, nil
}

// ReceiveCall parses the provider webhook and returns CallInfo.
func (d *InboundDispatcher) ReceiveCall(c *gin.Context, provider string) (*internal_type.CallInfo, error) {
	tel, err := GetTelephony(Telephony(provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return nil, fmt.Errorf("telephony provider %s not connected: %w", provider, err)
	}
	return tel.ReceiveCall(c)
}

// SaveCallContext stores the call context in Postgres and returns the contextID.
func (d *InboundDispatcher) SaveCallContext(ctx context.Context, auth types.SimplePrinciple, assistant *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
	direction := callInfo.Direction
	if direction == "" {
		direction = "inbound"
	}
	cc := &callcontext.CallContext{
		AssistantID:         assistant.Id,
		ConversationID:      conversationID,
		AssistantProviderId: assistant.AssistantProviderId,
		AuthToken:           auth.GetCurrentToken(),
		AuthType:            auth.Type().String(),
		Direction:           direction,
		CallerNumber:        callInfo.CallerNumber,
		FromNumber:          callInfo.FromNumber,
		Provider:            provider,
		ChannelUUID:         callInfo.ChannelUUID,
		CallStatus:          callInfo.Status.String(),
	}
	if delegatedContext, err := types.ResolveDelegatedContext(auth); err == nil {
		cc.OrganizationID = delegatedContext.OrganizationID
		if delegatedContext.ProjectID != nil {
			cc.ProjectID = *delegatedContext.ProjectID
		}
	}
	return d.store.Save(ctx, cc)
}

// AnswerProvider instructs the telephony provider to answer the call.
func (d *InboundDispatcher) AnswerProvider(c *gin.Context, auth types.SimplePrinciple, provider string, assistantID uint64, callerNumber string, conversationID uint64) error {
	tel, err := GetTelephony(Telephony(provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return fmt.Errorf("telephony provider %s not connected: %w", provider, err)
	}
	return tel.InboundCall(c, auth, assistantID, callerNumber, conversationID)
}
