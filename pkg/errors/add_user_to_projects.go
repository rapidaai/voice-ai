// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	AddUserToProjectsUnauthenticatedCode       ErrorCode = 1022001
	AddUserToProjectsMissingOrganizationCode   ErrorCode = 1022002
	AddUserToProjectsUnauthorizedCode          ErrorCode = 1022003
	AddUserToProjectsInvalidUserCode           ErrorCode = 1022004
	AddUserToProjectsUserNotInOrganizationCode ErrorCode = 1022005
	AddUserToProjectsMissingProjectRolesCode   ErrorCode = 1022006
	AddUserToProjectsInvalidProjectRoleCode    ErrorCode = 1022007
	AddUserToProjectsInvalidProjectsCode       ErrorCode = 1022008
	AddUserToProjectsDuplicateProjectCode      ErrorCode = 1022009
	AddUserToProjectsCreateProjectRolesCode    ErrorCode = 1022010
	AddUserToProjectsUserAlreadyInProjectCode  ErrorCode = 1022011
)

var (
	AddUserToProjectsUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           AddUserToProjectsUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	AddUserToProjectsMissingOrganization = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           AddUserToProjectsMissingOrganizationCode,
		Error:          "missing active organization",
		ErrorMessage:   "Please create or switch to an active organization before assigning projects.",
	}
	AddUserToProjectsUnauthorized = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           AddUserToProjectsUnauthorizedCode,
		Error:          "user is not authorized to assign users to projects",
		ErrorMessage:   "You do not have permission to assign users to projects.",
	}
	AddUserToProjectsInvalidUser = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsInvalidUserCode,
		Error:          "invalid user id",
		ErrorMessage:   "Please select a valid user.",
	}
	AddUserToProjectsUserNotInOrganization = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsUserNotInOrganizationCode,
		Error:          "user is not part of this organization",
		ErrorMessage:   "Please select a user from this organization.",
	}
	AddUserToProjectsMissingProjectRoles = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsMissingProjectRolesCode,
		Error:          "project role assignments are required",
		ErrorMessage:   "Please select at least one project and one or more roles.",
	}
	AddUserToProjectsInvalidProjectRole = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsInvalidProjectRoleCode,
		Error:          "invalid project role",
		ErrorMessage:   "Please select one or more valid roles for every selected project.",
	}
	AddUserToProjectsInvalidProjects = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsInvalidProjectsCode,
		Error:          "invalid project ids",
		ErrorMessage:   "Please select valid active projects in this organization.",
	}
	AddUserToProjectsDuplicateProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           AddUserToProjectsDuplicateProjectCode,
		Error:          "duplicate project assignment",
		ErrorMessage:   "Each project can only be selected once.",
	}
	AddUserToProjectsCreateProjectRoles = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           AddUserToProjectsCreateProjectRolesCode,
		Error:          "unable to create project roles",
		ErrorMessage:   "Unable to assign project roles, please try again later.",
	}
	AddUserToProjectsUserAlreadyInProject = PlatformError{
		HTTPStatusCode: http.StatusConflict,
		Code:           AddUserToProjectsUserAlreadyInProjectCode,
		Error:          "user is already assigned to project",
		ErrorMessage:   "This user is already assigned to one of the selected projects.",
	}
)
