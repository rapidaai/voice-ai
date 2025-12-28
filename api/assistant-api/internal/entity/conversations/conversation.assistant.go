// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_conversation_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

type AssistantConversation struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	Identifier               string `json:"identifier" gorm:"type:bigint;not null"`
	AssistantId              uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantProviderModelId uint64 `json:"assistantProviderModelId" gorm:"type:bigint;not null"`
	Name                     string `json:"name" gorm:"type:text"`

	Source    utils.RapidaSource               `json:"source" gorm:"type:string;size:50;not null;default:web-app"`
	Direction type_enums.ConversationDirection `json:"direction" gorm:"type:string;size:20;not null;default:inbound"`

	Arguments       []*AssistantConversationArgument       `json:"arguments" gorm:"foreignKey:AssistantConversationId"`
	Metadatas       []*AssistantConversationMetadata       `json:"metadata" gorm:"foreignKey:AssistantConversationId"`
	Metrics         []*AssistantConversationMetric         `json:"metrics" gorm:"foreignKey:AssistantConversationId"`
	Options         []*AssistantConversationOption         `json:"options" gorm:"foreignKey:AssistantConversationId"`
	Recordings      []*AssistantConversationRecording      `json:"recordings" gorm:"foreignKey:AssistantConversationId"`
	TelephonyEvents []*AssistantConversationTelephonyEvent `json:"telephonyEvents" gorm:"foreignKey:AssistantConversationId"`
}

func (ac *AssistantConversation) GetArugments() map[string]interface{} {
	args := make(map[string]interface{})
	if len(ac.Arguments) > 0 {
		for _, ar := range ac.Arguments {
			args[ar.Name] = ar.Argument.Value
		}
	}
	return args
}

func (ac *AssistantConversation) GetMetadatas() map[string]interface{} {
	mt := make(map[string]interface{})
	if len(ac.Metadatas) > 0 {
		for _, ar := range ac.Metadatas {
			mt[ar.Key] = ar.Value
		}
	}
	return mt
}

func (ac *AssistantConversation) GetOptions() utils.Option {
	mt := make(map[string]interface{})
	if len(ac.Options) > 0 {
		for _, ar := range ac.Options {
			mt[ar.Key] = ar.Value
		}
	}
	return mt
}

type AssistantConversationRecording struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	AssistantId             uint64 `json:"assistantId" gorm:"type:bigint;not null"`
	AssistantConversationId uint64 `json:"assistantConversationId" gorm:"type:bigint;not null"`
	RecordingUrl            string `json:"recordingUrl" gorm:"type:string;not null"`
}
