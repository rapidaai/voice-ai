// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/state"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type Handler interface {
	Type() string
	Execute(ctx context.Context, request Request) (Result, error)
	Close(ctx context.Context) error
}

type Request struct {
	ContextID        string
	InputText        string
	ContinuationText string
	Node             schema.Node
	RuntimeState     *state.RuntimeState
	Communication    internal_type.Communication
}

type Result struct {
	RouteHandles     []string
	ContinuationText string
	ResponseText     string
	WaitForNextInput bool
	Terminal         bool
}

type Registry struct {
	NodeHandlers map[string]Handler
	Handlers     []Handler
}

func (registry Registry) HandlerForNode(node schema.Node) (Handler, bool) {
	handler, exists := registry.NodeHandlers[node.ID]
	return handler, exists
}
