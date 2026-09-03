// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type SessionOption = internal_core.SessionOption

func WithSessionConfig(config *Config) SessionOption {
	return internal_core.WithSessionConfig(cloneConfig(config))
}

func WithSessionDirection(direction CallDirection) SessionOption {
	return internal_core.WithSessionDirection(direction)
}

func WithSessionCallID(callID string) SessionOption {
	return internal_core.WithSessionCallID(callID)
}

func WithSessionCodec(codec *Codec) SessionOption {
	return internal_core.WithSessionCodec(codecToCore(codec))
}

func WithSessionAuth(auth *types.Authentication) SessionOption {
	return internal_core.WithSessionAuth(auth)
}

func WithSessionAssistant(assistant *internal_assistant_entity.Assistant) SessionOption {
	return internal_core.WithSessionAssistant(assistant)
}

func WithSessionConversationID(conversationID uint64) SessionOption {
	return internal_core.WithSessionConversationID(conversationID)
}

func WithSessionContextID(contextID string) SessionOption {
	return internal_core.WithSessionContextID(contextID)
}

func WithSessionVaultCredential(vaultCredential *protos.VaultCredential) SessionOption {
	return internal_core.WithSessionVaultCredential(vaultCredential)
}

type Session struct {
	inner *internal_core.Session
}
