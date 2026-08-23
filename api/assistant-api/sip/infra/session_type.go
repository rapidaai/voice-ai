// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type SessionConfig struct {
	Config          *Config
	Direction       CallDirection
	CallID          string
	Codec           *Codec
	Logger          commons.Logger
	Auth            *types.Authentication
	Assistant       *internal_assistant_entity.Assistant
	ConversationID  uint64
	ContextID       string
	VaultCredential *protos.VaultCredential
}

func (cfg *SessionConfig) toCore() *internal_core.SessionConfig {
	if cfg == nil {
		return nil
	}
	return &internal_core.SessionConfig{
		Config:          cfg.Config.toCore(),
		Direction:       cfg.Direction.toCore(),
		CallID:          cfg.CallID,
		Codec:           cfg.Codec.toCore(),
		Logger:          cfg.Logger,
		Auth:            cfg.Auth,
		Assistant:       cfg.Assistant,
		ConversationID:  cfg.ConversationID,
		ContextID:       cfg.ContextID,
		VaultCredential: cfg.VaultCredential,
	}
}

type Session struct {
	inner *internal_core.Session
}
