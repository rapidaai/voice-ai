// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_authenticators

import (
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

// NewUserResolver constructs the Web API User principal resolver.
func NewUserResolver(logger commons.Logger, postgres connectors.PostgresConnector) *resolver.UserPrincipalResolver {
	return resolver.NewUserPrincipalResolver(internal_user_service.NewPrincipalAuthenticationService(logger, postgres))
}

// NewProjectResolver constructs the Web API Project principal resolver.
func NewProjectResolver(logger commons.Logger, postgres connectors.PostgresConnector) *resolver.ProjectPrincipalResolver {
	return resolver.NewProjectPrincipalResolver(internal_project_service.NewPrincipalAuthenticationService(logger, postgres))
}

// NewOrganizationResolver constructs the Web API Organization principal resolver.
func NewOrganizationResolver(fingerprintKey []byte, logger commons.Logger, postgres connectors.PostgresConnector) *resolver.OrganizationPrincipalResolver {
	return resolver.NewOrganizationPrincipalResolver(internal_organization_service.NewPrincipalAuthenticationService(logger, postgres, fingerprintKey))
}
