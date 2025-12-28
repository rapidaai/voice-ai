// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_talking_debugger

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_adapter_request_generic "github.com/rapidaai/api/assistant-api/internal/adapters/generic"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type debuggerTalking struct {
	internal_adapter_request_generic.GenericRequestor
	logger commons.Logger
}

// GetMessage implements internal_adapter_requests.Talking.
func NewTalking(
	ctx context.Context,
	config *config.AssistantConfig,
	logger commons.Logger,
	postgres connectors.PostgresConnector,
	opensearch connectors.OpenSearchConnector,
	redis connectors.RedisConnector,
	storage storages.Storage,
	streamer internal_streamers.Streamer,
) (internal_adapter_requests.Talking, error) {

	debuggerTalking := &debuggerTalking{
		logger: logger,
		GenericRequestor: internal_adapter_request_generic.
			NewGenericRequestor(
				context.Background(),
				config,
				logger,
				utils.Debugger,
				postgres,
				opensearch,
				redis,
				storage,
				streamer,
			),
	}
	return debuggerTalking, nil
}

/*
* startlistening starts a goroutine that listens for incoming messages on the stream.//+
* It initializes the transformer, processes incoming requests, and handles different content types.//+
* The function continues to listen until an EOF or a Canceled error is received.//+
* //+
* This method doesn't take any parameters as it operates on the debuggerTalking struct.//+
* //+
* The function doesn't return any value. It runs asynchronously in a separate goroutine.//+
 */
func (talking *debuggerTalking) Talk(
	ctx context.Context,
	auth types.SimplePrinciple,
	identifier string,
) error {
	talking.StartedAt = time.Now()
	var initialized = false
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			if initialized {
				talking.Disconnect()
			}
			return ctx.Err()
		default:
			// Continue processing
		}

		req, err := talking.Streamer().Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				if initialized {
					talking.Disconnect()
				}
				break
			}
			// Log and return unrecoverable errors
			return fmt.Errorf("stream.Recv error: %w", err)
		}
		switch msg := req.GetRequest().(type) {
		case *protos.AssistantMessagingRequest_Message:
			if initialized {
				// talking.logger.Benchmark("accepting input after", time.Since(talking.StartedAt))
				if err := talking.Input(req.GetMessage()); err != nil {
					talking.logger.Errorf("error while accepting input %v", err)
				}
			}
		case *protos.AssistantMessagingRequest_Configuration:
			if err := talking.Connect(ctx, auth, identifier, msg.Configuration); err != nil {
				talking.logger.Errorf("unexpected error while connect assistant, might be problem in configuration %+v", err)
				return fmt.Errorf("talking.Connect error: %w", err)
			}
			initialized = true
		}
	}
	return nil
}
