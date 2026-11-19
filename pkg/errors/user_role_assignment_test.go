// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestUserRoleAssignmentErrorsRemainClassifiableWhenWrapped(t *testing.T) {
	tests := []error{
		RoleAssignmentSelectionUnavailableError{},
		OrganizationMembershipConflictError{SameOrganization: true},
		ProjectMembershipConflictError{},
		OrganizationOwnerMutationError{},
	}
	for _, target := range tests {
		wrapped := fmt.Errorf("assignment failed: %w", target)
		switch target.(type) {
		case RoleAssignmentSelectionUnavailableError:
			var value RoleAssignmentSelectionUnavailableError
			if !errors.As(wrapped, &value) {
				t.Fatal("selection error is not classifiable")
			}
		case OrganizationMembershipConflictError:
			var value OrganizationMembershipConflictError
			if !errors.As(wrapped, &value) {
				t.Fatal("organization conflict is not classifiable")
			}
		case ProjectMembershipConflictError:
			var value ProjectMembershipConflictError
			if !errors.As(wrapped, &value) {
				t.Fatal("project conflict is not classifiable")
			}
		case OrganizationOwnerMutationError:
			var value OrganizationOwnerMutationError
			if !errors.As(wrapped, &value) {
				t.Fatal("owner error is not classifiable")
			}
		}
	}
}
