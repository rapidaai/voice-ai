package web_authenticators

import (
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

func GetUserAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.Authenticator {
	return internal_user_service.NewAuthenticator(logger, postgres)
}

func GetOrganizationAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector, fingerprintKey string) types.ClaimAuthenticator[*types.OrganizationScope] {
	return internal_organization_service.NewOrganizationAuthenticator(logger, postgres, fingerprintKey)
}

func GetProjectAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.ClaimAuthenticator[*types.ProjectScope] {
	return internal_project_service.NewProjectAuthenticator(logger, postgres)
}
