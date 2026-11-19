// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	"net/http"
	"testing"
)

func TestProjectAPIKeyPlatformErrors(t *testing.T) {
	tests := []struct {
		name          string
		platformError PlatformError
		httpStatus    int
		code          ErrorCode
		machine       string
		human         string
	}{
		{name: "list invalid project", platformError: ListProjectAPIKeyInvalidProject, httpStatus: http.StatusBadRequest, code: 1026101, machine: "invalid project identifier", human: "Select a valid project and try again."},
		{name: "list invalid pagination", platformError: ListProjectAPIKeyInvalidPagination, httpStatus: http.StatusBadRequest, code: 1026102, machine: "invalid project API key pagination", human: "Choose a page and page size between 1 and 100."},
		{name: "list invalid criteria", platformError: ListProjectAPIKeyInvalidCriteria, httpStatus: http.StatusBadRequest, code: 1026103, machine: "invalid project API key criteria", human: "Use supported filters and values, then try again."},
		{name: "revoke invalid project", platformError: RevokeProjectAPIKeyInvalidProject, httpStatus: http.StatusBadRequest, code: 1026201, machine: "invalid project identifier", human: "Select a valid project and try again."},
		{name: "revoke invalid credential", platformError: RevokeProjectAPIKeyInvalidCredential, httpStatus: http.StatusBadRequest, code: 1026202, machine: "invalid project API key identifier", human: "Select a valid API key and try again."},
		{name: "revoke unavailable", platformError: RevokeProjectAPIKeyNotFound, httpStatus: http.StatusNotFound, code: 1026203, machine: "project API key not found", human: "The selected API key is unavailable."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.platformError.HTTPStatusCode != test.httpStatus || test.platformError.Code != test.code || test.platformError.Error != test.machine || test.platformError.ErrorMessage != test.human {
				t.Fatalf("platform error = %+v", test.platformError)
			}
		})
	}
}

func TestProjectAPIKeyNotFoundError(t *testing.T) {
	if got := (ProjectAPIKeyNotFoundError{}).Error(); got != "project API key not found" {
		t.Fatalf("Error() = %q", got)
	}
}
