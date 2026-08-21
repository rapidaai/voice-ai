package types

import (
	"errors"
	"reflect"
)

type ProjectContext struct {
	OrganizationID uint64
	ProjectID      uint64
}

type DelegatedContext struct {
	UserID         *uint64
	OrganizationID uint64
	ProjectID      *uint64
}

// RequireUser resolves a user from a legacy authentication principle.
// Deprecated: use Authentication.UserContext.
func RequireUser(auth AuthenticationPrinciple) (uint64, error) {
	if err := requireAuthenticated(auth); err != nil {
		return 0, err
	}
	provider, ok := auth.(UserIdentityProvider)
	if !ok {
		return 0, errors.New("authenticated principle does not provide user identity")
	}
	userID, ok := provider.UserIdentity()
	if !ok || userID == 0 {
		return 0, errors.New("authenticated principle has no valid user identity")
	}
	return userID, nil
}

// RequireOrganization resolves an organization from a legacy authentication principle.
// Deprecated: use Authentication.OrganizationContext.
func RequireOrganization(auth AuthenticationPrinciple) (uint64, error) {
	if err := requireAuthenticated(auth); err != nil {
		return 0, err
	}
	if provider, ok := auth.(OrganizationContextProvider); ok {
		organizationID, valid := provider.OrganizationContext()
		if !valid || organizationID == 0 {
			return 0, errors.New("authenticated principle has no valid organization context")
		}
		return organizationID, nil
	}
	delegatedContext, ok := delegatedContextFromProvider(auth)
	if !ok {
		return 0, errors.New("authenticated principle does not provide organization context")
	}
	return delegatedContext.OrganizationID, nil
}

// RequireProject resolves a project from a legacy authentication principle.
// Deprecated: use Authentication.ProjectContext.
func RequireProject(auth AuthenticationPrinciple) (ProjectContext, error) {
	if err := requireAuthenticated(auth); err != nil {
		return ProjectContext{}, err
	}
	if provider, ok := auth.(ProjectContextProvider); ok {
		projectContext, valid := provider.ProjectContext()
		if !valid || projectContext.OrganizationID == 0 || projectContext.ProjectID == 0 {
			return ProjectContext{}, errors.New("authenticated principle has no valid project context")
		}
		return projectContext, nil
	}
	delegatedContext, ok := delegatedContextFromProvider(auth)
	if !ok || delegatedContext.ProjectID == nil {
		return ProjectContext{}, errors.New("authenticated principle does not provide project context")
	}
	return ProjectContext{
		OrganizationID: delegatedContext.OrganizationID,
		ProjectID:      *delegatedContext.ProjectID,
	}, nil
}

// ResolveDelegatedContext resolves contexts from a legacy authentication principle.
// Deprecated: use Authentication context methods directly.
func ResolveDelegatedContext(auth AuthenticationPrinciple) (DelegatedContext, error) {
	if err := requireAuthenticated(auth); err != nil {
		return DelegatedContext{}, err
	}
	if _, ok := auth.(DelegatedContextProvider); ok {
		delegatedContext, valid := delegatedContextFromProvider(auth)
		if !valid {
			return DelegatedContext{}, errors.New("authenticated principle has no valid delegated context")
		}
		return delegatedContext, nil
	}

	organizationID, err := RequireOrganization(auth)
	if err != nil {
		return DelegatedContext{}, err
	}
	derived := DelegatedContext{OrganizationID: organizationID}
	if provider, ok := auth.(UserIdentityProvider); ok {
		if userID, provided := provider.UserIdentity(); provided {
			if userID == 0 {
				return DelegatedContext{}, errors.New("authenticated principle has malformed delegated user identity")
			}
			derived.UserID = &userID
		}
	}
	if provider, ok := auth.(ProjectContextProvider); ok {
		if projectContext, provided := provider.ProjectContext(); provided {
			if projectContext.OrganizationID == 0 || projectContext.ProjectID == 0 {
				return DelegatedContext{}, errors.New("authenticated principle has malformed delegated project context")
			}
			if projectContext.OrganizationID != organizationID {
				return DelegatedContext{}, errors.New("delegated project context does not match organization context")
			}
			derived.ProjectID = &projectContext.ProjectID
		}
	}
	delegatedContext, ok := normalizeDelegatedContext(derived, true)
	if !ok {
		return DelegatedContext{}, errors.New("authenticated principle has no valid derived delegated context")
	}
	return delegatedContext, nil
}

func delegatedContextFromProvider(auth AuthenticationPrinciple) (DelegatedContext, bool) {
	provider, ok := auth.(DelegatedContextProvider)
	if !ok {
		return DelegatedContext{}, false
	}
	return normalizeDelegatedContext(provider.DelegatedContext())
}

func requireAuthenticated(auth AuthenticationPrinciple) error {
	if auth == nil || isNilAuthenticationPrinciple(auth) || !auth.IsAuthenticated() {
		return errors.New("authenticated principle is required")
	}
	return nil
}

func isNilAuthenticationPrinciple(auth AuthenticationPrinciple) bool {
	value := reflect.ValueOf(auth)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func normalizeDelegatedContext(context DelegatedContext, provided bool) (DelegatedContext, bool) {
	if !provided || context.OrganizationID == 0 {
		return DelegatedContext{}, false
	}
	userID, ok := normalizeOptionalID(context.UserID)
	if !ok {
		return DelegatedContext{}, false
	}
	projectID, ok := normalizeOptionalID(context.ProjectID)
	if !ok {
		return DelegatedContext{}, false
	}
	return DelegatedContext{
		UserID:         userID,
		OrganizationID: context.OrganizationID,
		ProjectID:      projectID,
	}, true
}

func normalizeOptionalID(id *uint64) (*uint64, bool) {
	if id == nil {
		return nil, true
	}
	if *id == 0 {
		return nil, false
	}
	value := *id
	return &value, true
}
