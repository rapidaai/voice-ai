// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	InviteUserToOrganizationUnauthenticatedCode           ErrorCode = 1021001
	InviteUserToOrganizationMissingOrganizationCode       ErrorCode = 1021002
	InviteUserToOrganizationUnauthorizedCode              ErrorCode = 1021003
	InviteUserToOrganizationInvalidEmailCode              ErrorCode = 1021004
	InviteUserToOrganizationInvalidOrganizationRoleCode   ErrorCode = 1021005
	InviteUserToOrganizationInvalidProjectRoleCode        ErrorCode = 1021006
	InviteUserToOrganizationInvalidProjectsCode           ErrorCode = 1021007
	InviteUserToOrganizationDuplicateProjectCode          ErrorCode = 1021008
	InviteUserToOrganizationCreateUserCode                ErrorCode = 1021009
	InviteUserToOrganizationCreateOrganizationRoleCode    ErrorCode = 1021010
	InviteUserToOrganizationCreateProjectRolesCode        ErrorCode = 1021011
	InviteUserToOrganizationUserInAnotherOrganizationCode ErrorCode = 1021012
	InviteUserToOrganizationUserAlreadyInOrganizationCode ErrorCode = 1021013
)

var (
	InviteUserToOrganizationUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           InviteUserToOrganizationUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	InviteUserToOrganizationMissingOrganization = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           InviteUserToOrganizationMissingOrganizationCode,
		Error:          "missing active organization",
		ErrorMessage:   "Please create or switch to an active organization before inviting users.",
	}
	InviteUserToOrganizationUnauthorized = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           InviteUserToOrganizationUnauthorizedCode,
		Error:          "user is not authorized to invite users to organization",
		ErrorMessage:   "You do not have permission to invite users to this organization.",
	}
	InviteUserToOrganizationInvalidEmail = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           InviteUserToOrganizationInvalidEmailCode,
		Error:          "invalid email address",
		ErrorMessage:   "Please enter a valid email address.",
	}
	InviteUserToOrganizationInvalidOrganizationRole = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           InviteUserToOrganizationInvalidOrganizationRoleCode,
		Error:          "invalid organization role",
		ErrorMessage:   "Please select one or more valid organization roles.",
	}
	InviteUserToOrganizationInvalidProjectRole = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           InviteUserToOrganizationInvalidProjectRoleCode,
		Error:          "invalid project role",
		ErrorMessage:   "Please select one or more valid roles for every selected project.",
	}
	InviteUserToOrganizationInvalidProjects = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           InviteUserToOrganizationInvalidProjectsCode,
		Error:          "invalid project ids",
		ErrorMessage:   "Please select valid active projects in this organization.",
	}
	InviteUserToOrganizationDuplicateProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           InviteUserToOrganizationDuplicateProjectCode,
		Error:          "duplicate project assignment",
		ErrorMessage:   "Each project can only be selected once.",
	}
	InviteUserToOrganizationCreateUser = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           InviteUserToOrganizationCreateUserCode,
		Error:          "unable to create user for organization invite",
		ErrorMessage:   "Unable to invite user, please try again later.",
	}
	InviteUserToOrganizationCreateOrganizationRole = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           InviteUserToOrganizationCreateOrganizationRoleCode,
		Error:          "unable to create organization role for invite",
		ErrorMessage:   "Unable to invite user, please try again later.",
	}
	InviteUserToOrganizationCreateProjectRoles = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           InviteUserToOrganizationCreateProjectRolesCode,
		Error:          "unable to create project roles for invite",
		ErrorMessage:   "Unable to assign project roles, please try again later.",
	}
	InviteUserToOrganizationUserInAnotherOrganization = PlatformError{
		HTTPStatusCode: http.StatusConflict,
		Code:           InviteUserToOrganizationUserInAnotherOrganizationCode,
		Error:          "user is already part of another organization",
		ErrorMessage:   "User is part of another organization, please ask the user to switch to this organization before inviting.",
	}
	InviteUserToOrganizationUserAlreadyInOrganization = PlatformError{
		HTTPStatusCode: http.StatusConflict,
		Code:           InviteUserToOrganizationUserAlreadyInOrganizationCode,
		Error:          "user is already part of this organization",
		ErrorMessage:   "This user is already part of this organization.",
	}
)
