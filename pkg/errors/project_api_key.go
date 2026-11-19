// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "net/http"

// ProjectAPIKeyNotFoundError hides whether a credential exists outside the requested scope.
type ProjectAPIKeyNotFoundError struct{}

func (ProjectAPIKeyNotFoundError) Error() string {
	return "project API key not found"
}

const (
	ListProjectAPIKeyInvalidProjectCode      ErrorCode = 1026101
	ListProjectAPIKeyInvalidPaginationCode   ErrorCode = 1026102
	ListProjectAPIKeyInvalidCriteriaCode     ErrorCode = 1026103
	RevokeProjectAPIKeyInvalidProjectCode    ErrorCode = 1026201
	RevokeProjectAPIKeyInvalidCredentialCode ErrorCode = 1026202
	RevokeProjectAPIKeyNotFoundCode          ErrorCode = 1026203
)

var (
	ListProjectAPIKeyInvalidProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           ListProjectAPIKeyInvalidProjectCode,
		Error:          "invalid project identifier",
		ErrorMessage:   "Select a valid project and try again.",
	}
	ListProjectAPIKeyInvalidPagination = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           ListProjectAPIKeyInvalidPaginationCode,
		Error:          "invalid project API key pagination",
		ErrorMessage:   "Choose a page and page size between 1 and 100.",
	}
	ListProjectAPIKeyInvalidCriteria = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           ListProjectAPIKeyInvalidCriteriaCode,
		Error:          "invalid project API key criteria",
		ErrorMessage:   "Use supported filters and values, then try again.",
	}
	RevokeProjectAPIKeyInvalidProject = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           RevokeProjectAPIKeyInvalidProjectCode,
		Error:          "invalid project identifier",
		ErrorMessage:   "Select a valid project and try again.",
	}
	RevokeProjectAPIKeyInvalidCredential = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           RevokeProjectAPIKeyInvalidCredentialCode,
		Error:          "invalid project API key identifier",
		ErrorMessage:   "Select a valid API key and try again.",
	}
	RevokeProjectAPIKeyNotFound = PlatformError{
		HTTPStatusCode: http.StatusNotFound,
		Code:           RevokeProjectAPIKeyNotFoundCode,
		Error:          "project API key not found",
		ErrorMessage:   "The selected API key is unavailable.",
	}
)
