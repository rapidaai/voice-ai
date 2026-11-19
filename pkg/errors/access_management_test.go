// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	"net/http"
	"testing"
)

func TestAccessManagementPlatformErrors(t *testing.T) {
	tests := []struct {
		platformError PlatformError
		status        int
		code          ErrorCode
		machine       string
		human         string
	}{
		{AccessManagementInvalidPermissionCriteria, http.StatusBadRequest, 1026301, "invalid permission criteria", "Use a supported permission filter and try again."},
		{AccessManagementInvalidRoleCriteria, http.StatusBadRequest, 1026302, "invalid role criteria", "Use supported role filters and try again."},
		{AccessManagementInvalidPagination, http.StatusBadRequest, 1026303, "invalid pagination", "Choose a page and page size between 1 and 100."},
		{AccessManagementInvalidRoleID, http.StatusBadRequest, 1026304, "invalid role identifier", "Select a valid role and try again."},
		{AccessManagementInvalidRoleScope, http.StatusBadRequest, 1026305, "invalid role scope", "Select an organization or project scope and try again."},
		{AccessManagementInvalidRoleSubject, http.StatusBadRequest, 1026306, "invalid role subject", "Select console or API access and try again."},
		{AccessManagementInvalidAPIRoleScope, http.StatusBadRequest, 1026307, "API access roles require project scope", "Use project scope for API access roles."},
		{AccessManagementInvalidRoleKey, http.StatusBadRequest, 1026308, "invalid role key", "Use 3 to 64 lowercase letters, numbers, or underscores, starting with a letter."},
		{AccessManagementInvalidRoleName, http.StatusBadRequest, 1026309, "invalid role name", "Enter a role name with 200 characters or fewer."},
		{AccessManagementInvalidRoleDescription, http.StatusBadRequest, 1026310, "invalid role description", "Use 400 characters or fewer for the role description."},
		{AccessManagementInvalidPermissionSelection, http.StatusBadRequest, 1026311, "invalid role permission selection", "Select valid permissions for the role scope and try again."},
		{AccessManagementInvalidRoleChangeReason, http.StatusBadRequest, 1026312, "invalid role change reason", "Enter a reason with 400 characters or fewer."},
		{AccessManagementRoleNotFound, http.StatusNotFound, 1026313, "role not found", "The selected role is unavailable."},
		{AccessManagementRoleKeyAlreadyExists, http.StatusConflict, 1026314, "role key already exists", "Choose a different role key and try again."},
		{AccessManagementRoleChangeNotAllowed, http.StatusBadRequest, 1026315, "role change is not allowed", "This role cannot be changed in its current state."},
		{AccessManagementInvalidConfirmationDigest, http.StatusBadRequest, 1026316, "invalid role reduction confirmation", "Preview the role reduction and submit its confirmation digest."},
		{AccessManagementReductionStale, http.StatusPreconditionFailed, 1026317, "role reduction preview is stale", "Preview the role reduction again before confirming it."},
	}
	for _, test := range tests {
		if test.platformError.HTTPStatusCode != test.status || test.platformError.Code != test.code || test.platformError.Error != test.machine || test.platformError.ErrorMessage != test.human {
			t.Fatalf("unexpected platform error: %+v", test.platformError)
		}
	}
}

func TestAccessManagementDomainErrors(t *testing.T) {
	errors := []error{
		AccessManagementPermissionDeniedError{},
		AccessManagementRoleNotFoundError{},
		AccessManagementRoleKeyAlreadyExistsError{},
		AccessManagementRolePermissionSelectionInvalidError{},
		AccessManagementRoleChangeNotAllowedError{},
	}
	for _, err := range errors {
		if err.Error() == "" {
			t.Fatalf("domain error has no message: %T", err)
		}
	}
}
