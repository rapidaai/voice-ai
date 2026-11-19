// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	"net/http"
	"testing"
)

func TestProviderCredentialPlatformErrors(t *testing.T) {
	tests := []struct {
		value PlatformError
		code  ErrorCode
	}{
		{ProviderCredentialInvalidCriteria, ProviderCredentialInvalidCriteriaCode},
		{ProviderCredentialInvalidPagination, ProviderCredentialInvalidPaginationCode},
	}
	for _, test := range tests {
		if test.value.HTTPStatusCode != http.StatusBadRequest || test.value.Code != test.code || test.value.CodeString() == "" {
			t.Fatalf("PlatformError = %+v", test.value)
		}
	}
}
