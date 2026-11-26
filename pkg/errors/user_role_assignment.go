// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

type RoleAssignmentSelection string

const (
	RoleAssignmentOrganizationSelection RoleAssignmentSelection = "organization"
	RoleAssignmentOrganizationRole      RoleAssignmentSelection = "organization_role"
	RoleAssignmentProject               RoleAssignmentSelection = "project"
	RoleAssignmentProjectRole           RoleAssignmentSelection = "project_role"
	RoleAssignmentMembership            RoleAssignmentSelection = "membership"
)

type RoleAssignmentSelectionUnavailableError struct {
	Selection RoleAssignmentSelection
}

func (RoleAssignmentSelectionUnavailableError) Error() string {
	return "selected role assignment is unavailable"
}

type OrganizationMembershipConflictError struct {
	SameOrganization bool
}

func (e OrganizationMembershipConflictError) Error() string {
	if e.SameOrganization {
		return "user is already assigned to this organization"
	}
	return "user is assigned to another organization"
}

type ProjectMembershipConflictError struct{}

func (ProjectMembershipConflictError) Error() string {
	return "user is already assigned to a selected project"
}

type OrganizationOwnerMutationError struct{}

func (OrganizationOwnerMutationError) Error() string {
	return "organization owner assignments cannot be changed"
}

type OrganizationRolePersistenceError struct{ Cause error }

func (e OrganizationRolePersistenceError) Error() string {
	return "persist organization role assignments: " + e.Cause.Error()
}
func (e OrganizationRolePersistenceError) Unwrap() error { return e.Cause }

type ProjectRolePersistenceError struct{ Cause error }

func (e ProjectRolePersistenceError) Error() string {
	return "persist project role assignments: " + e.Cause.Error()
}
func (e ProjectRolePersistenceError) Unwrap() error { return e.Cause }
