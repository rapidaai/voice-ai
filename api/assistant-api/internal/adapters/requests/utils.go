package internal_adapter_requests

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

const (
	VERSION_PREFIX = "vrsn_"
)

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

/*
 * GetVersionDefinition processes the input version string and returns a pointer to uint64.
 *
 * This function handles the following cases:
 * 1. If the version is empty or "latest", it returns nil for both the pointer and error.
 * 2. For other version strings, it removes the VERSION_PREFIX (if present) and converts
 *    the remaining string to a uint64.
 *
 * Parameters:
 * - version: A string representing the version to be processed.
 *
 * Returns:
 * - *uint64: A pointer to the parsed version as a uint64, or nil if version is empty/"latest".
 * - error: An error if the string-to-uint64 conversion fails, or nil if successful.
 *
 * Note: This function assumes that valid version strings (after removing the prefix)
 * are numerical and can be converted to uint64.
 */
func GetVersionDefinition(version string) (*uint64, error) {
	if version == "" || version == "latest" {
		return nil, nil
	}
	_vrsn := strings.Replace(version, VERSION_PREFIX, "", 1)
	_pid, err := strconv.ParseUint(_vrsn, 10, 64)
	return &_pid, err
}
