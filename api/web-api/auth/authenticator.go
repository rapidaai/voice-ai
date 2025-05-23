package web_authenticators

import (
	internal_project_service "github.com/lexatic/web-backend/api/web-api/internal/services/project"
	internal_user_service "github.com/lexatic/web-backend/api/web-api/internal/services/user"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

func GetUserAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.Authenticator {
	return internal_user_service.NewAuthenticator(logger, postgres)
}

func GetProjectAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.ClaimAuthenticator[*types.ProjectScope] {
	return internal_project_service.NewProjectAuthenticator(logger, postgres)
}
