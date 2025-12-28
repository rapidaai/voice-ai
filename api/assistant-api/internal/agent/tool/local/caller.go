// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_local_tool

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type ToolCaller interface {
	// tool call id
	Id() uint64

	//
	Name() string

	//
	Definition() (*protos.FunctionDefinition, error)

	//
	ExecutionMethod() string

	//
	Call(
		ctx context.Context,
		messageId string,
		args string,
		communication internal_adapter_requests.Communication,
	) (map[string]interface{}, []*types.Metric)
}
