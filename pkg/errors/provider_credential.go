// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import "net/http"

const (
	ProviderCredentialInvalidCriteriaCode   ErrorCode = 1026401
	ProviderCredentialInvalidPaginationCode ErrorCode = 1026402
)

var (
	ProviderCredentialInvalidCriteria = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           ProviderCredentialInvalidCriteriaCode,
		Error:          "invalid provider credential criteria",
		ErrorMessage:   "Use supported provider credential filters and try again.",
	}
	ProviderCredentialInvalidPagination = PlatformError{
		HTTPStatusCode: http.StatusBadRequest,
		Code:           ProviderCredentialInvalidPaginationCode,
		Error:          "invalid provider credential pagination",
		ErrorMessage:   "Choose a page and page size between 1 and 100.",
	}
)
