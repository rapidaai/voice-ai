// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"golang.org/x/sync/errgroup"
)

type websocketExecutor struct {
	logger     commons.Logger
	connection *websocket.Conn // WebSocket connection
}

// Init implements internal_executors.AssistantExecutor.
func (executor *websocketExecutor) Initialize(ctx context.Context,
	communication internal_adapter_requests.Communication) error {
	ctx, span, _ := communication.Tracer().StartSpan(
		ctx,
		utils.AssistantAgentConnectStage,
		internal_adapter_telemetry.KV{
			K: "executor",
			V: internal_adapter_telemetry.StringValue(executor.Name()),
		},
	)
	defer span.EndSpan(ctx, utils.AssistantAgentConnectStage)
	g, ctx := errgroup.WithContext(ctx)

	providerDefinition := communication.
		Assistant().
		AssistantProviderWebsocket

	g.Go(func() error {

		// Prepare HTTP headers
		headers := http.Header{}
		if providerDefinition.Headers != nil {
			for key, value := range providerDefinition.Headers { // Assuming communication.Headers is map[string]string
				headers.Set(key, value)
			}
		}
		wsURL, err := url.Parse(providerDefinition.Url)
		if err != nil {
			return err
		}

		// Add query parameters to the WebSocket URL
		query := wsURL.Query()
		if providerDefinition.Parameters != nil {
			for key, value := range providerDefinition.Parameters { // Assuming communication.Params is map[string]string
				query.Set(key, value)
			}
			wsURL.RawQuery = query.Encode()
		}

		conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
		span.AddAttributes(ctx, internal_adapter_telemetry.KV{
			K: "url",
			V: internal_adapter_telemetry.StringValue(wsURL.String()),
		})
		if err != nil {
			executor.logger.Errorf("Error while getting provider model credentials: %v", err)
			return fmt.Errorf("failed to get provider credential: %w", err)
		}
		executor.connection = conn
		return err

	})
	// Persist WebSocket connection
	return nil
}

// Name implements internal_executors.AssistantExecutor.
func (a *websocketExecutor) Name() string {
	return "websocket"
}

// Talk implements internal_executors.AssistantExecutor.
func (a *websocketExecutor) Talk(ctx context.Context, messageid string, msg *types.Message, communcation internal_adapter_requests.Communication) error {
	panic("unimplemented")
}

func (a *websocketExecutor) Close(
	ctx context.Context,
	communication internal_adapter_requests.Communication,
) error {
	return nil
}

func NewWebsocketAssistantExecutor(
	logger commons.Logger,
) AssistantExecutor {
	return &websocketExecutor{
		logger: logger,
	}

}
