// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_base

import (
	"encoding/base64"
	"strings"

	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// BaseTelephonyStreamer embeds channel_base.BaseStreamer for channel lifecycle
// and adds telephony-specific call context, base64 encoding, and credentials.
//
// Concrete telephony streamers (Twilio, Exotel, Vonage, SIP, Asterisk) embed
// this struct and only implement transport-specific I/O logic.
type BaseTelephonyStreamer struct {
	channel_base.BaseStreamer

	// callCtx holds IDs and metadata from the call setup phase (Redis).
	// Replaces separate assistant/conversation entity references. The
	// streamer only needs IDs, not full DB entities.
	callCtx *callcontext.CallContext

	resampler       internal_type.AudioResampler
	encoder         *base64.Encoding
	vaultCredential *protos.VaultCredential
	observer        observability.Recorder

	// ChannelUUID is the provider-specific call identifier, propagated from
	// CallContext so concrete streamers can use it for call control.
	ChannelUUID string
}

// New creates a new BaseTelephonyStreamer from call context
// and vault credentials.
func New(
	logger commons.Logger,
	cc *callcontext.CallContext,
	vaultCred *protos.VaultCredential,
	observer observability.Recorder,
) BaseTelephonyStreamer {
	resampler := resampler_soxr.New(
		resampler_soxr.WithLogger(logger),
		resampler_soxr.WithHighQuality(),
	)
	return BaseTelephonyStreamer{
		BaseStreamer:    channel_base.New(channel_base.WithLogger(logger)),
		callCtx:         cc,
		resampler:       resampler,
		encoder:         base64.StdEncoding,
		vaultCredential: vaultCred,
		observer:        observer,
		ChannelUUID:     cc.ChannelUUID,
	}
}

// GetAssistantDefinition returns the protobuf assistant definition.
func (base *BaseTelephonyStreamer) GetAssistantDefinition() *protos.AssistantDefinition {
	return &protos.AssistantDefinition{
		AssistantId: base.callCtx.AssistantID,
		Version:     utils.GetVersionString(base.callCtx.AssistantProviderId),
	}
}

// GetConversationId returns the conversation ID.
func (base *BaseTelephonyStreamer) GetConversationId() uint64 {
	return base.callCtx.ConversationID
}

// Encoder returns the base64 encoder used by the streamer.
func (base *BaseTelephonyStreamer) Encoder() *base64.Encoding {
	return base.encoder
}

// VaultCredential returns the vault credential associated with the streamer.
func (base *BaseTelephonyStreamer) VaultCredential() *protos.VaultCredential {
	return base.vaultCredential
}

// Resampler returns the audio resampler.
func (base *BaseTelephonyStreamer) Resampler() internal_type.AudioResampler {
	return base.resampler
}

func (base *BaseTelephonyStreamer) Observer() observability.Recorder {
	return base.observer
}

func (base *BaseTelephonyStreamer) Record(records ...observability.Record) error {
	if base.observer == nil {
		return nil
	}
	return base.observer.Record(base.Ctx, observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: base.GetAssistantDefinition().AssistantId},
		ConversationID: base.GetConversationId(),
	}, records...)
}

// CreateConnectionRequest builds the initial ConversationInitialization message.
// Carries non-empty client.* metadata in the init payload so the requestor's
// in-memory state is populated at connect time. This avoids races with downstream
// metadata writes that only persist to DB. Empty fields are omitted so they
// can't overwrite previously-stored values during resume.
func (base *BaseTelephonyStreamer) CreateConnectionRequest() *protos.ConversationInitialization {
	clientMetadata := map[string]interface{}{
		"client.direction": base.callCtx.Direction,
		"client.channel":   base.callCtx.Provider,
	}
	if v := base.callCtx.CallerNumber; v != "" {
		clientMetadata["client.phone"] = v
	}
	if v := base.callCtx.FromNumber; v != "" {
		clientMetadata["client.assistant_phone"] = v
	}
	if v := base.callCtx.ChannelUUID; v != "" {
		clientMetadata["client.provider_call_id"] = v
	}
	if v := base.callCtx.ContextID; v != "" {
		clientMetadata["client.context_id"] = v
	}
	metadata, _ := utils.InterfaceMapToAnyMap(clientMetadata)
	return &protos.ConversationInitialization{
		AssistantConversationId: base.GetConversationId(),
		Assistant:               base.GetAssistantDefinition(),
		StreamMode:              protos.StreamMode_STREAM_MODE_AUDIO,
		Metadata:                metadata,
	}
}

// SplitTransferTargets parses a multi-target transfer_to argument into an
// ordered list of targets joined by commons.SEPARATOR. Empty/whitespace
// entries are dropped. If parsing yields nothing, the original raw string is
// returned as a single-element slice so callers always have at least one
// candidate to act on.
func (base *BaseTelephonyStreamer) SplitTransferTargets(raw string) []string {
	parts := strings.Split(raw, commons.SEPARATOR)
	targets := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			targets = append(targets, t)
		}
	}
	if len(targets) == 0 {
		return []string{raw}
	}
	return targets
}
