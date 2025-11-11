package internal_factories

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_adapter_request_talking_debugger "github.com/rapidaai/api/assistant-api/internal/adapters/requests/debugger/talking"
	internal_adapter_request_talking_phone "github.com/rapidaai/api/assistant-api/internal/adapters/requests/phone/talking"
	internal_adapter_request_talking_sdk "github.com/rapidaai/api/assistant-api/internal/adapters/requests/sdk/talking"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	internal_adapter_request_talking_web_plugin "github.com/rapidaai/api/assistant-api/internal/adapters/requests/web-plugin/talking"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

func GetTalker(source utils.RapidaSource,
	ctx context.Context,
	cfg *config.AssistantConfig,
	logger commons.Logger,
	postgres connectors.PostgresConnector,
	opensearch connectors.OpenSearchConnector,
	redis connectors.RedisConnector,
	storage storages.Storage,
	streamer internal_adapter_request_streamers.Streamer,
) (internal_adapter_requests.Talking, error) {
	switch source {
	case utils.SDK:
		talker, err := internal_adapter_request_talking_sdk.NewSDKTalking(
			ctx,
			cfg, logger,
			postgres,
			opensearch,
			redis,
			storage, streamer)
		if err != nil {
			logger.Errorf("assistant call talker failed with err %+v", err)
			return nil, err
		}
		return talker, nil

	case utils.Debugger:
		talker, err := internal_adapter_request_talking_debugger.NewTalking(
			ctx,
			cfg,
			logger,
			postgres,
			opensearch,
			redis,
			storage, streamer)
		if err != nil {
			logger.Errorf("assistant call talker failed with err %+v", err)
			return nil, err
		}
		return talker, nil

	case utils.PhoneCall:
		talker, err := internal_adapter_request_talking_phone.NewTalking(
			ctx,
			cfg,
			logger,
			postgres,
			opensearch,
			redis,
			storage,
			streamer,
		)
		if err != nil {
			logger.Errorf("assistant call talker failed with err %+v", err)
		}
		return talker, nil
	case utils.WebPlugin:
		talker, err := internal_adapter_request_talking_web_plugin.NewTalking(
			ctx,
			cfg,
			logger,
			postgres,
			opensearch,
			redis,
			storage, streamer)
		if err != nil {
			logger.Errorf("assistant call talker failed with err %+v", err)
		}
		return talker, nil
	default:
		talker, err := internal_adapter_request_talking_debugger.NewTalking(
			ctx,
			cfg,
			logger,
			postgres,
			opensearch,
			redis,
			storage, streamer)
		if err != nil {
			logger.Errorf("assistant call talker failed with err %+v", err)
			return nil, err
		}
		return talker, nil
	}

}

func Identifier(source utils.RapidaSource, ctx context.Context, auth types.SimplePrinciple, identity string) string {
	switch source {
	case utils.WebPlugin:
		return internal_adapter_requests.WebPluginIdentifier(ctx, auth)
	case utils.Debugger:
		return internal_adapter_requests.DebuggerIdentifier(auth)
	case utils.SDK:
		return internal_adapter_requests.RapidaSDKIdentifier(ctx, auth)
	case utils.PhoneCall:
		return internal_adapter_requests.RapidaCallIdentifier(auth, identity)
	case utils.Whatsapp:
		return internal_adapter_requests.RapidaWhatsappIdentifier(auth, identity)
	default:
		return identity
	}
}
