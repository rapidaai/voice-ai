// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCreateProjectCredentialDomainErrors(t *testing.T) {
	selectionErr := fmt.Errorf("wrapped: %w", CredentialSelectionNotFoundError{})
	var selectionTarget CredentialSelectionNotFoundError
	if !stderrors.As(selectionErr, &selectionTarget) || selectionTarget.Error() != "eligible project or project-token role not found" {
		t.Fatalf("selection error = %v", selectionErr)
	}

	organizationErr := fmt.Errorf("wrapped: %w", CredentialOrganizationInvalidError{})
	var organizationTarget CredentialOrganizationInvalidError
	if !stderrors.As(organizationErr, &organizationTarget) || organizationTarget.Error() != "project credential organization is invalid" {
		t.Fatalf("organization error = %v", organizationErr)
	}
}

func TestCreateProjectCredentialPlatformErrors(t *testing.T) {
	tests := []struct {
		name          string
		platformError PlatformError
		httpStatus    int
		code          ErrorCode
		codeString    string
		machine       string
		human         string
	}{
		{name: "invalid project", platformError: CreateProjectCredentialInvalidProject, httpStatus: http.StatusBadRequest, code: 1026001, codeString: "1026001", machine: "invalid project identifier", human: "Select a valid project and try again."},
		{name: "missing name", platformError: CreateProjectCredentialMissingName, httpStatus: http.StatusBadRequest, code: 1026002, codeString: "1026002", machine: "project credential name is required", human: "Enter a project credential name and try again."},
		{name: "invalid role", platformError: CreateProjectCredentialInvalidRole, httpStatus: http.StatusBadRequest, code: 1026003, codeString: "1026003", machine: "invalid project-token role identifier", human: "Select a project-token role and try again."},
		{name: "selection unavailable", platformError: CreateProjectCredentialSelectionNotFound, httpStatus: http.StatusNotFound, code: 1026004, codeString: "1026004", machine: "eligible project or project-token role not found", human: "The selected project or role is unavailable."},
	}
	seen := make(map[ErrorCode]string, len(tests))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.platformError.HTTPStatusCode != test.httpStatus || test.platformError.Code != test.code || test.platformError.CodeString() != test.codeString || test.platformError.Error != test.machine || test.platformError.ErrorMessage != test.human {
				t.Fatalf("PlatformError = %+v", test.platformError)
			}
			if other, exists := seen[test.platformError.Code]; exists {
				t.Fatalf("error code %d is shared by %q and %q", test.platformError.Code, other, test.name)
			}
			seen[test.platformError.Code] = test.name
		})
	}
}
