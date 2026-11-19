// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "net/http"

// CredentialSelectionNotFoundError hides which eligible project credential selection was unavailable.
type CredentialSelectionNotFoundError struct{}

func (CredentialSelectionNotFoundError) Error() string {
	return "eligible project or project-token role not found"
}

// CredentialOrganizationInvalidError identifies an invalid authenticated Organization invariant.
type CredentialOrganizationInvalidError struct{}

func (CredentialOrganizationInvalidError) Error() string {
	return "project credential organization is invalid"
}

const (
	CreateProjectCredentialInvalidProjectCode    ErrorCode = 1026001
	CreateProjectCredentialMissingNameCode       ErrorCode = 1026002
	CreateProjectCredentialInvalidRoleCode       ErrorCode = 1026003
	CreateProjectCredentialSelectionNotFoundCode ErrorCode = 1026004
)

var (
	CreateProjectCredentialInvalidProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateProjectCredentialInvalidProjectCode,
		Error:          "invalid project identifier",
		ErrorMessage:   "Select a valid project and try again.",
	}
	CreateProjectCredentialMissingName = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateProjectCredentialMissingNameCode,
		Error:          "project credential name is required",
		ErrorMessage:   "Enter a project credential name and try again.",
	}
	CreateProjectCredentialInvalidRole = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           CreateProjectCredentialInvalidRoleCode,
		Error:          "invalid project-token role identifier",
		ErrorMessage:   "Select a project-token role and try again.",
	}
	CreateProjectCredentialSelectionNotFound = PlatformError{
		HTTPStatusCode: http.StatusNotFound,
		Code:           CreateProjectCredentialSelectionNotFoundCode,
		Error:          "eligible project or project-token role not found",
		ErrorMessage:   "The selected project or role is unavailable.",
	}
)
