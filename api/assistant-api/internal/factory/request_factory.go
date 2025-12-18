package internal_factories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rapidaai/api/assistant-api/config"
	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_adapter_request_talking_debugger "github.com/rapidaai/api/assistant-api/internal/adapters/requests/debugger/talking"
	internal_adapter_request_talking_phone "github.com/rapidaai/api/assistant-api/internal/adapters/requests/phone/talking"
	internal_adapter_request_talking_sdk "github.com/rapidaai/api/assistant-api/internal/adapters/requests/sdk/talking"
	internal_adapter_request_talking_web_plugin "github.com/rapidaai/api/assistant-api/internal/adapters/requests/web-plugin/talking"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
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
	streamer internal_streamers.Streamer,
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
		return WebPluginIdentifier(ctx, auth)
	case utils.Debugger:
		return DebuggerIdentifier(auth)
	case utils.SDK:
		return RapidaSDKIdentifier(ctx, auth)
	case utils.PhoneCall:
		return RapidaCallIdentifier(auth, identity)
	case utils.Whatsapp:
		return RapidaWhatsappIdentifier(auth, identity)
	default:
		return identity
	}
}

/*
* DebuggerIdentifier generates a unique identifier for a debugger session.
* It combines the user ID, current project ID, and current organization ID
* from the provided SimplePrinciple authentication object.
* The format of the identifier is: "rapida-debugger-<userId>-<projectId>-<orgId>"
 */
func DebuggerIdentifier(auth types.SimplePrinciple) string {
	return fmt.Sprintf("rapida-debugger-%d-%d-%d",
		*auth.GetUserId(),
		*auth.GetCurrentProjectId(),
		*auth.GetCurrentOrganizationId())
}

/*
WebPluginIdentifier generates a unique identifier for web plugin requests.
It combines information from the SimplePrinciple, client source, client environment,
and authentication ID to create a lowercase string identifier.

Parameters:
  - auth: SimplePrinciple containing project and organization IDs
  - ctx: Context containing client information and auth ID

Returns:
  - A string identifier in the format: "source-environment-authId-projectId-organizationId"
*/
func WebPluginIdentifier(ctx context.Context, auth types.SimplePrinciple) string {
	source, _ := utils.GetClientSource(ctx)
	environment, _ := utils.GetClientEnvironment(ctx)
	authId, exists := utils.GetAuthId(ctx)

	if !exists {
		authId = utils.Ptr[string](uuid.New().String())
	}
	vl := strings.ToLower(
		fmt.Sprintf(`%s-%s-%s-%d-%d`,
			source,
			environment,
			*authId,
			*auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId()))
	return vl
}

// RapidaSDKIdentifier generates a unique identifier for the Rapida SDK.
//
// This function creates a standardized identifier string used in the Rapida SDK
// by combining various pieces of information:
//
// 1. Client source: Obtained from the context
// 2. Client environment: Obtained from the context
// 3. Authentication ID: Retrieved from the context or generated if not present
// 4. Current project ID: Obtained from the SimplePrinciple
// 5. Current organization ID: Obtained from the SimplePrinciple
//
// The resulting identifier is a lowercase string with the format:
// "source-environment-authid-projectid-organizationid"
//
// Parameters:
//   - auth: A types.SimplePrinciple object containing project and organization IDs
//   - ctx: The context.Context object
//
// Returns:
//   - A string representing the unique Rapida SDK identifier

func RapidaSDKIdentifier(ctx context.Context, auth types.SimplePrinciple) string {
	source, _ := utils.GetClientSource(ctx)
	environment, _ := utils.GetClientEnvironment(ctx)
	authId, exists := utils.GetAuthId(ctx)

	if !exists {
		authId = utils.Ptr[string](uuid.New().String())
	}
	vl := strings.ToLower(
		fmt.Sprintf(`%s-%s-%s-%d-%d`,
			source,
			environment,
			*authId,
			*auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId()))
	return vl
}

// RapidaTwilioCallIdentifier generates a unique identifier for Twilio calls in the Rapida system.
//
// This function creates a standardized, lowercase string that combines several pieces of information:
// 1. A fixed prefix "twilio-call" to indicate the type of identifier
// 2. The environment, hardcoded as "production"
// 3. The provided identity string
// 4. The current project ID from the authentication principle
// 5. The current organization ID from the authentication principle
//
// Parameters:
//   - auth: A SimplePrinciple object containing authentication information
//   - identity: A string representing the specific identity for this call
//
// Returns:
//   - A string containing the formatted and lowercase identifier
//
// Note: This identifier can be used for logging, tracking, or associating Twilio calls
// with specific projects and organizations within the Rapida system.

func RapidaCallIdentifier(auth types.SimplePrinciple, identity string) string {
	vl := strings.ToLower(
		fmt.Sprintf(`%s-%s-%s-%d-%d`,
			"phone-call",
			"production",
			identity,
			*auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId()))
	return vl
}

// RapidaTwilioWhatsappIdentifier generates a unique identifier for Twilio WhatsApp integration.
//
// This function creates a standardized, lowercase string that combines several pieces of information:
// - The string "twilio-whatsapp" to indicate the service
// - The environment, which is set to "production"
// - The provided identity string
// - The current project ID
// - The current organization ID
//
// The resulting identifier is useful for tracking and managing Twilio WhatsApp integrations
// across different projects and organizations within the Rapida system.
//
// Parameters:
//   - auth: A SimplePrinciple object containing authentication information
//   - identity: A string representing the specific identity for this WhatsApp integration
//
// Returns:
//   - A string containing the formatted, lowercase identifier

func RapidaWhatsappIdentifier(auth types.SimplePrinciple, identity string) string {
	vl := strings.ToLower(
		fmt.Sprintf(`%s-%s-%s-%d-%d`,
			"twilio-whatsapp",
			"production",
			identity,
			*auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId()))
	return vl
}
