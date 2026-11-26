// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "net/http"

type AccessManagementPermissionDeniedError struct{}

func (AccessManagementPermissionDeniedError) Error() string {
	return "access management permission denied"
}

type AccessManagementRoleNotFoundError struct{}

func (AccessManagementRoleNotFoundError) Error() string {
	return "role not found"
}

type AccessManagementRoleKeyAlreadyExistsError struct{}

func (AccessManagementRoleKeyAlreadyExistsError) Error() string {
	return "role key already exists"
}

type AccessManagementRolePermissionSelectionInvalidError struct{}

func (AccessManagementRolePermissionSelectionInvalidError) Error() string {
	return "invalid role permission selection"
}

type AccessManagementRoleChangeNotAllowedError struct{}

func (AccessManagementRoleChangeNotAllowedError) Error() string {
	return "role change is not allowed"
}

type AccessManagementReductionStaleError struct{}

func (AccessManagementReductionStaleError) Error() string {
	return "role reduction preview is stale"
}

const (
	AccessManagementInvalidPermissionCriteriaCode  ErrorCode = 1026301
	AccessManagementInvalidRoleCriteriaCode        ErrorCode = 1026302
	AccessManagementInvalidPaginationCode          ErrorCode = 1026303
	AccessManagementInvalidRoleIDCode              ErrorCode = 1026304
	AccessManagementInvalidRoleScopeCode           ErrorCode = 1026305
	AccessManagementInvalidRoleSubjectCode         ErrorCode = 1026306
	AccessManagementInvalidAPIRoleScopeCode        ErrorCode = 1026307
	AccessManagementInvalidRoleKeyCode             ErrorCode = 1026308
	AccessManagementInvalidRoleNameCode            ErrorCode = 1026309
	AccessManagementInvalidRoleDescriptionCode     ErrorCode = 1026310
	AccessManagementInvalidPermissionSelectionCode ErrorCode = 1026311
	AccessManagementInvalidRoleChangeReasonCode    ErrorCode = 1026312
	AccessManagementRoleNotFoundCode               ErrorCode = 1026313
	AccessManagementRoleKeyAlreadyExistsCode       ErrorCode = 1026314
	AccessManagementRoleChangeNotAllowedCode       ErrorCode = 1026315
	AccessManagementInvalidConfirmationDigestCode  ErrorCode = 1026316
	AccessManagementReductionStaleCode             ErrorCode = 1026317
)

var (
	AccessManagementInvalidPermissionCriteria  = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidPermissionCriteriaCode, Error: "invalid permission criteria", ErrorMessage: "Use a supported permission filter and try again."}
	AccessManagementInvalidRoleCriteria        = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleCriteriaCode, Error: "invalid role criteria", ErrorMessage: "Use supported role filters and try again."}
	AccessManagementInvalidPagination          = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidPaginationCode, Error: "invalid pagination", ErrorMessage: "Choose a page and page size between 1 and 100."}
	AccessManagementInvalidRoleID              = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleIDCode, Error: "invalid role identifier", ErrorMessage: "Select a valid role and try again."}
	AccessManagementInvalidRoleScope           = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleScopeCode, Error: "invalid role scope", ErrorMessage: "Select an organization or project scope and try again."}
	AccessManagementInvalidRoleSubject         = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleSubjectCode, Error: "invalid role subject", ErrorMessage: "Select console or API access and try again."}
	AccessManagementInvalidAPIRoleScope        = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidAPIRoleScopeCode, Error: "API access roles require project scope", ErrorMessage: "Use project scope for API access roles."}
	AccessManagementInvalidRoleKey             = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleKeyCode, Error: "invalid role key", ErrorMessage: "Use 3 to 64 lowercase letters, numbers, or underscores, starting with a letter."}
	AccessManagementInvalidRoleName            = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleNameCode, Error: "invalid role name", ErrorMessage: "Enter a role name with 200 characters or fewer."}
	AccessManagementInvalidRoleDescription     = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleDescriptionCode, Error: "invalid role description", ErrorMessage: "Use 400 characters or fewer for the role description."}
	AccessManagementInvalidPermissionSelection = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidPermissionSelectionCode, Error: "invalid role permission selection", ErrorMessage: "Select valid permissions for the role scope and try again."}
	AccessManagementInvalidRoleChangeReason    = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidRoleChangeReasonCode, Error: "invalid role change reason", ErrorMessage: "Enter a reason with 400 characters or fewer."}
	AccessManagementRoleNotFound               = PlatformError{HTTPStatusCode: http.StatusNotFound, Code: AccessManagementRoleNotFoundCode, Error: "role not found", ErrorMessage: "The selected role is unavailable."}
	AccessManagementRoleKeyAlreadyExists       = PlatformError{HTTPStatusCode: http.StatusConflict, Code: AccessManagementRoleKeyAlreadyExistsCode, Error: "role key already exists", ErrorMessage: "Choose a different role key and try again."}
	AccessManagementRoleChangeNotAllowed       = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementRoleChangeNotAllowedCode, Error: "role change is not allowed", ErrorMessage: "This role cannot be changed in its current state."}
	AccessManagementInvalidConfirmationDigest  = PlatformError{HTTPStatusCode: http.StatusBadRequest, Code: AccessManagementInvalidConfirmationDigestCode, Error: "invalid role reduction confirmation", ErrorMessage: "Preview the role reduction and submit its confirmation digest."}
	AccessManagementReductionStale             = PlatformError{HTTPStatusCode: http.StatusPreconditionFailed, Code: AccessManagementReductionStaleCode, Error: "role reduction preview is stale", ErrorMessage: "Preview the role reduction again before confirming it."}
)
