// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/node"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/state"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

const maxNodeStepsPerTurn = 32

type orchestrator struct {
	graph        *graph
	runtimeState *state.RuntimeState
	nodeRegistry node.Registry
	turnMu       sync.Mutex
}

func newOrchestrator(
	graph *graph,
	runtimeState *state.RuntimeState,
	nodeRegistry node.Registry,
) *orchestrator {
	return &orchestrator{
		graph:        graph,
		runtimeState: runtimeState,
		nodeRegistry: nodeRegistry,
	}
}

func (orchestrator *orchestrator) HandleUserInput(ctx context.Context, communication internal_type.Communication, packet internal_type.UserInputPacket) error {
	orchestrator.turnMu.Lock()
	defer orchestrator.turnMu.Unlock()

	currentNodeID := orchestrator.runtimeState.CurrentNodeID()
	return orchestrator.runFromNode(ctx, communication, packet.ContextID, currentNodeID, packet.Text, "")
}

func (orchestrator *orchestrator) HandleToolResult(packet internal_type.LLMToolResultPacket) {
	orchestrator.runtimeState.RecordToolResult(packet.Name, packet.Result)
}

func (orchestrator *orchestrator) runFromNode(
	ctx context.Context,
	communication internal_type.Communication,
	contextID string,
	startNodeID string,
	inputText string,
	continuationText string,
) error {
	currentNodeID := startNodeID
	currentInputText := inputText
	currentContinuationText := continuationText

	for nodeStep := 0; nodeStep < maxNodeStepsPerTurn; nodeStep++ {
		currentNode, exists := orchestrator.graph.node(currentNodeID)
		if !exists {
			return fmt.Errorf("agentflow: node %q does not exist", currentNodeID)
		}

		handler, exists := orchestrator.nodeRegistry.HandlerForNode(currentNode)
		if !exists {
			return fmt.Errorf("agentflow: unsupported node type %q", currentNode.Type)
		}

		orchestrator.runtimeState.SetCurrentNodeID(currentNode.ID)
		result, err := handler.Execute(ctx, node.Request{
			ContextID:        contextID,
			InputText:        currentInputText,
			ContinuationText: currentContinuationText,
			Node:             currentNode,
			RuntimeState:     orchestrator.runtimeState,
			Communication:    communication,
		})
		if err != nil {
			return err
		}
		if result.Terminal {
			return nil
		}
		if result.WaitForNextInput {
			return nil
		}

		nextNodeID, foundNextNode := "", false
		if len(result.RouteHandles) > 0 {
			nextNodeID, foundNextNode = orchestrator.graph.targetByHandle(currentNode.ID, result.RouteHandles...)
		} else {
			nextNodeID, foundNextNode = orchestrator.graph.firstTarget(currentNode.ID)
		}
		if !foundNextNode {
			if len(result.RouteHandles) > 0 {
				_ = communication.OnPacket(ctx, internal_type.ObservabilityEventRecordPacket{
					ContextID: contextID,
					Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
					Record: observability.RecordEvent{
						Component: observability.ComponentAgent,
						Event:     observability.AgentTransitionMissingEdge,
						Attributes: observability.Attributes{
							"context_id":      contextID,
							"from_node_id":    currentNode.ID,
							"from_node_label": currentNode.Label,
							"from_node_type":  currentNode.Type,
							"route_handles":   strings.Join(result.RouteHandles, ","),
							"result":          "missing_edge",
						},
						OccurredAt: time.Now(),
					},
				})
			}
			return nil
		}
		nextNode, _ := orchestrator.graph.node(nextNodeID)
		if len(result.RouteHandles) > 0 {
			_ = communication.OnPacket(ctx, internal_type.ObservabilityEventRecordPacket{
				ContextID: contextID,
				Scope:     internal_type.ObservabilityRecordScopeAssistantMessage,
				Record: observability.RecordEvent{
					Component: observability.ComponentAgent,
					Event:     observability.AgentTransitionMatched,
					Attributes: observability.Attributes{
						"context_id":      contextID,
						"from_node_id":    currentNode.ID,
						"from_node_label": currentNode.Label,
						"from_node_type":  currentNode.Type,
						"route_handles":   strings.Join(result.RouteHandles, ","),
						"to_node_id":      nextNode.ID,
						"to_node_label":   nextNode.Label,
						"to_node_type":    nextNode.Type,
						"result":          "matched_edge",
					},
					OccurredAt: time.Now(),
				},
			})
		}

		currentNodeID = nextNodeID
		currentInputText = ""
		currentContinuationText = result.ContinuationText
	}

	return fmt.Errorf("agentflow: maximum node steps exceeded")
}
