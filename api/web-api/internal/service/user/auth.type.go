// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import internal_entity "github.com/rapidaai/api/web-api/internal/entity"

// authenticationSnapshot owns the database records used to build one Authentication aggregate.
type authenticationSnapshot struct {
	user             *internal_entity.UserAuth
	userAuthToken    *internal_entity.UserAuthToken
	userOrgRoles     []*internal_entity.UserOrganizationRole
	userProjectRoles []*internal_entity.UserProjectRole
}
