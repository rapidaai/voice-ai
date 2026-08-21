// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

type AssistantConfigurationService interface {
	Get(
		ctx context.Context,
		auth *types.Authentication,
		configurationId uint64,
		assistantId uint64,
	) (*internal_assistant_entity.AssistantConfiguration, error)

	GetAll(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		configurationType string,
		provider string,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
	) (int64, []*internal_assistant_entity.AssistantConfiguration, error)

	Create(
		ctx context.Context,
		auth *types.Authentication,
		assistantId uint64,
		configurationType string,
		provider string,
		enabled bool,
		options []*protos.Metadata,
	) (*internal_assistant_entity.AssistantConfiguration, error)

	Update(
		ctx context.Context,
		auth *types.Authentication,
		configurationId uint64,
		assistantId uint64,
		configurationType string,
		provider string,
		enabled bool,
		options []*protos.Metadata,
	) (*internal_assistant_entity.AssistantConfiguration, error)

	Delete(
		ctx context.Context,
		auth *types.Authentication,
		configurationId uint64,
		assistantId uint64,
	) (*internal_assistant_entity.AssistantConfiguration, error)
}
