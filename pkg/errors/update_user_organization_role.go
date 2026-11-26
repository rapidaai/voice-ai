// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import "net/http"

const (
	UpdateUserOrganizationRoleUnauthenticatedCode     ErrorCode = 1025001
	UpdateUserOrganizationRoleMissingOrganizationCode ErrorCode = 1025002
	UpdateUserOrganizationRoleUnauthorizedCode        ErrorCode = 1025003
	UpdateUserOrganizationRoleInvalidUserCode         ErrorCode = 1025004
	UpdateUserOrganizationRoleInvalidRoleCode         ErrorCode = 1025005
	UpdateUserOrganizationRoleUserNotInOrgCode        ErrorCode = 1025006
	UpdateUserOrganizationRoleOwnerCode               ErrorCode = 1025007
	UpdateUserOrganizationRoleUpdateRoleCode          ErrorCode = 1025008
)

var (
	UpdateUserOrganizationRoleUnauthenticated = PlatformError{
		HTTPStatusCode: http.StatusUnauthorized,
		Code:           UpdateUserOrganizationRoleUnauthenticatedCode,
		Error:          "unauthenticated request",
		ErrorMessage:   "Unauthenticated request, please try again with valid authentication.",
	}
	UpdateUserOrganizationRoleMissingOrganization = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           UpdateUserOrganizationRoleMissingOrganizationCode,
		Error:          "missing active organization",
		ErrorMessage:   "Please create or switch to an active organization before updating organization roles.",
	}
	UpdateUserOrganizationRoleUnauthorized = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           UpdateUserOrganizationRoleUnauthorizedCode,
		Error:          "user is not authorized to update organization roles",
		ErrorMessage:   "You do not have permission to update organization roles.",
	}
	UpdateUserOrganizationRoleInvalidUser = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           UpdateUserOrganizationRoleInvalidUserCode,
		Error:          "invalid user id",
		ErrorMessage:   "Please select a valid user.",
	}
	UpdateUserOrganizationRoleInvalidRole = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           UpdateUserOrganizationRoleInvalidRoleCode,
		Error:          "invalid organization role",
		ErrorMessage:   "Please select one or more valid organization roles.",
	}
	UpdateUserOrganizationRoleUserNotInOrg = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           UpdateUserOrganizationRoleUserNotInOrgCode,
		Error:          "user is not part of this organization",
		ErrorMessage:   "Please select a user from this organization.",
	}
	UpdateUserOrganizationRoleOwner = PlatformError{
		HTTPStatusCode: http.StatusForbidden,
		Code:           UpdateUserOrganizationRoleOwnerCode,
		Error:          "cannot update organization owner role",
		ErrorMessage:   "Organization owner role cannot be changed.",
	}
	UpdateUserOrganizationRoleUpdateRole = PlatformError{
		HTTPStatusCode: http.StatusInternalServerError,
		Code:           UpdateUserOrganizationRoleUpdateRoleCode,
		Error:          "unable to update organization role",
		ErrorMessage:   "Unable to update organization role, please try again later.",
	}
)
