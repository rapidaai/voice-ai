package internal_adapter_request_talking_web_plugin

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_adapter_request_generic "github.com/rapidaai/api/assistant-api/internal/adapters/requests/generic"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type webpluginTalking struct {
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
	stream internal_adapter_request_streamers.Streamer,
) (internal_adapter_requests.Talking, error) {
	return &webpluginTalking{
		logger: logger,
		GenericRequestor: internal_adapter_request_generic.NewGenericRequestor(
			ctx,
			config,
			logger,
			utils.WebPlugin,
			postgres,
			opensearch,
			redis,
			storage, stream),
	}, nil
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
func (talking *webpluginTalking) Talk(
	ctx context.Context,
	auth types.SimplePrinciple,
	identifier string) error {

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
				talking.Input(req.GetMessage())
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
