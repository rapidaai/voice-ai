// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

const (
	RoleScopeOrganization = "ORGANIZATION"
	RoleScopeProject      = "PROJECT"

	RoleSubjectUser         = "USER"
	RoleSubjectProjectToken = "PROJECT_TOKEN"

	RoleDefinitionBuiltIn = "BUILT_IN"
	RoleDefinitionCustom  = "CUSTOM"
)

// Permission is one platform authorization capability assigned through a Role.
type Permission struct {
	ID          uint64
	Key         string
	Scope       string
	Name        string
	Description string
}

// Role is one assigned role and its authoritative Permission records.
type Role struct {
	ID             uint64
	Key            string
	Name           string
	Description    string
	Scope          string
	Subject        string
	DefinitionType string
	Permissions    []Permission
}

// BillingPlanQuota is one Organization billing-plan allowance.
type BillingPlanQuota struct {
	ID           uint64
	ResourceType string
	QuotaLimit   int64
	Unit         string
}

// BillingPlan is the plan owned by one Organization authentication projection.
type BillingPlan struct {
	ID           uint64
	Name         string
	Slug         string
	Description  string
	IsDefault    bool
	IsActive     bool
	SortOrder    int32
	PriceMonthly int64
	PriceYearly  int64
	Currency     string
	StripeURL    string
	Quotas       []BillingPlanQuota
}

// Project is one Project membership and its assigned Roles.
type Project struct {
	ID    uint64
	Name  string
	Roles []Role
}

// Organization is one Organization membership, billing plan, and Project graph.
type Organization struct {
	ID          uint64
	Name        string
	Roles       []Role
	BillingPlan *BillingPlan
	Projects    []Project
}

// User is the profile associated with a User subject.
type User struct {
	ID     uint64
	Name   string
	Email  string
	Status string
}

// Authentication is an immutable identity, membership, role, permission, and billing snapshot.
type Authentication struct {
	principal     Principal
	user          *User
	organizations []Organization
}

// NewDirectAuthentication validates a direct subject and derives its Principal authorities from the Role graph.
func NewDirectAuthentication(subject Identity, tenant Tenant, proof Proof, user *User, organizations []Organization) (Authentication, error) {
	principal, err := NewDirectPrincipal(subject, tenant, proof, nil)
	if err != nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	return newAuthentication(principal, user, organizations)
}

// NewDelegatedAuthentication validates a Service-delegated subject and derives its Principal authorities from the Role graph.
func NewDelegatedAuthentication(subject Identity, actor ServiceActor, tenant Tenant, proof Proof, user *User, organizations []Organization) (Authentication, error) {
	principal, err := NewDelegatedPrincipal(subject, actor, tenant, proof, nil)
	if err != nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	return newAuthentication(principal, user, organizations)
}

func newAuthentication(principal Principal, user *User, organizations []Organization) (Authentication, error) {
	subject := principal.Subject()
	if !validator.Between(subject.ID, uint64(1), uint64(math.MaxInt64)) {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	if subject.Type == IdentityUser {
		if user == nil || user.ID != subject.ID || !validator.Between(user.ID, uint64(1), uint64(math.MaxInt64)) ||
			!utf8.ValidString(user.Name) || !validator.NotBlank(user.Name) || strings.TrimSpace(user.Name) != user.Name || strings.ContainsRune(user.Name, '\x00') ||
			!utf8.ValidString(user.Email) || !validator.NotBlank(user.Email) || strings.TrimSpace(user.Email) != user.Email || strings.ContainsRune(user.Email, '\x00') ||
			!utf8.ValidString(user.Status) || !validator.NotBlank(user.Status) || strings.TrimSpace(user.Status) != user.Status || strings.ContainsRune(user.Status, '\x00') {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
	} else if user != nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}

	roleSubject := ""
	switch subject.Type {
	case IdentityUser:
		roleSubject = RoleSubjectUser
	case IdentityProjectCredential:
		roleSubject = RoleSubjectProjectToken
	case IdentityOrganizationCredential, IdentityService:
	default:
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}

	permissionIDs := make(map[uint64]string)
	permissionKeys := make(map[string]uint64)
	organizationIDs := make(map[uint64]struct{}, len(organizations))
	value := Authentication{principal: principal}
	if user != nil {
		userCopy := *user
		value.user = &userCopy
	}
	value.organizations = make([]Organization, 0, len(organizations))

	for _, inputOrganization := range organizations {
		if !validator.Between(inputOrganization.ID, uint64(1), uint64(math.MaxInt64)) ||
			!utf8.ValidString(inputOrganization.Name) || !validator.NotBlank(inputOrganization.Name) ||
			strings.TrimSpace(inputOrganization.Name) != inputOrganization.Name || strings.ContainsRune(inputOrganization.Name, '\x00') {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
		if _, exists := organizationIDs[inputOrganization.ID]; exists {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
		organizationIDs[inputOrganization.ID] = struct{}{}
		organization := Organization{ID: inputOrganization.ID, Name: inputOrganization.Name}

		roleIDs := make(map[uint64]struct{}, len(inputOrganization.Roles))
		organization.Roles = make([]Role, 0, len(inputOrganization.Roles))
		for _, inputRole := range inputOrganization.Roles {
			if subject.Type != IdentityUser || !validator.Between(inputRole.ID, uint64(1), uint64(math.MaxInt64)) ||
				inputRole.Scope != RoleScopeOrganization || inputRole.Subject != roleSubject ||
				!validator.OneOf(inputRole.DefinitionType, RoleDefinitionBuiltIn, RoleDefinitionCustom) ||
				!utf8.ValidString(inputRole.Key) || !validator.NotBlank(inputRole.Key) || strings.TrimSpace(inputRole.Key) != inputRole.Key || strings.ContainsRune(inputRole.Key, '\x00') ||
				!utf8.ValidString(inputRole.Name) || !validator.NotBlank(inputRole.Name) || strings.TrimSpace(inputRole.Name) != inputRole.Name || strings.ContainsRune(inputRole.Name, '\x00') ||
				!utf8.ValidString(inputRole.Description) || strings.TrimSpace(inputRole.Description) != inputRole.Description || strings.ContainsRune(inputRole.Description, '\x00') {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			if _, exists := roleIDs[inputRole.ID]; exists {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			roleIDs[inputRole.ID] = struct{}{}
			role := Role{ID: inputRole.ID, Key: inputRole.Key, Name: inputRole.Name, Description: inputRole.Description, Scope: inputRole.Scope, Subject: inputRole.Subject, DefinitionType: inputRole.DefinitionType}
			role.Permissions = make([]Permission, 0, len(inputRole.Permissions))
			rolePermissionIDs := make(map[uint64]struct{}, len(inputRole.Permissions))
			rolePermissionKeys := make(map[string]struct{}, len(inputRole.Permissions))
			for _, permission := range inputRole.Permissions {
				if !validator.Between(permission.ID, uint64(1), uint64(math.MaxInt64)) || permission.Scope != RoleScopeOrganization ||
					!strings.HasPrefix(permission.Key, "organization.") || !utf8.ValidString(permission.Key) || !validator.NotBlank(permission.Key) || strings.TrimSpace(permission.Key) != permission.Key || strings.ContainsRune(permission.Key, '\x00') ||
					!utf8.ValidString(permission.Name) || !validator.NotBlank(permission.Name) || strings.TrimSpace(permission.Name) != permission.Name || strings.ContainsRune(permission.Name, '\x00') ||
					!utf8.ValidString(permission.Description) || strings.TrimSpace(permission.Description) != permission.Description || strings.ContainsRune(permission.Description, '\x00') {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if _, exists := rolePermissionIDs[permission.ID]; exists {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if _, exists := rolePermissionKeys[permission.Key]; exists {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if key, exists := permissionIDs[permission.ID]; exists && key != permission.Key {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if id, exists := permissionKeys[permission.Key]; exists && id != permission.ID {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				rolePermissionIDs[permission.ID] = struct{}{}
				rolePermissionKeys[permission.Key] = struct{}{}
				permissionIDs[permission.ID] = permission.Key
				permissionKeys[permission.Key] = permission.ID
				role.Permissions = append(role.Permissions, permission)
			}
			sort.Slice(role.Permissions, func(left int, right int) bool {
				if role.Permissions[left].Key == role.Permissions[right].Key {
					return role.Permissions[left].ID < role.Permissions[right].ID
				}
				return role.Permissions[left].Key < role.Permissions[right].Key
			})
			organization.Roles = append(organization.Roles, role)
		}
		sort.Slice(organization.Roles, func(left int, right int) bool { return organization.Roles[left].ID < organization.Roles[right].ID })

		if inputOrganization.BillingPlan != nil {
			plan := *inputOrganization.BillingPlan
			if !validator.Between(plan.ID, uint64(1), uint64(math.MaxInt64)) || plan.SortOrder < 0 || plan.PriceMonthly < 0 || plan.PriceYearly < 0 ||
				!utf8.ValidString(plan.Name) || !validator.NotBlank(plan.Name) || strings.TrimSpace(plan.Name) != plan.Name || strings.ContainsRune(plan.Name, '\x00') ||
				!utf8.ValidString(plan.Slug) || !validator.NotBlank(plan.Slug) || strings.TrimSpace(plan.Slug) != plan.Slug || strings.ContainsRune(plan.Slug, '\x00') ||
				!utf8.ValidString(plan.Currency) || !validator.NotBlank(plan.Currency) || strings.TrimSpace(plan.Currency) != plan.Currency || strings.ContainsRune(plan.Currency, '\x00') ||
				!utf8.ValidString(plan.Description) || strings.TrimSpace(plan.Description) != plan.Description || strings.ContainsRune(plan.Description, '\x00') ||
				!utf8.ValidString(plan.StripeURL) || strings.TrimSpace(plan.StripeURL) != plan.StripeURL || strings.ContainsRune(plan.StripeURL, '\x00') {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			quotaIDs := make(map[uint64]struct{}, len(plan.Quotas))
			resourceTypes := make(map[string]struct{}, len(plan.Quotas))
			plan.Quotas = append([]BillingPlanQuota(nil), plan.Quotas...)
			for _, quota := range plan.Quotas {
				if !validator.Between(quota.ID, uint64(1), uint64(math.MaxInt64)) || quota.QuotaLimit < -1 ||
					!utf8.ValidString(quota.ResourceType) || !validator.NotBlank(quota.ResourceType) || strings.TrimSpace(quota.ResourceType) != quota.ResourceType || strings.ContainsRune(quota.ResourceType, '\x00') ||
					!utf8.ValidString(quota.Unit) || !validator.NotBlank(quota.Unit) || strings.TrimSpace(quota.Unit) != quota.Unit || strings.ContainsRune(quota.Unit, '\x00') {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if _, exists := quotaIDs[quota.ID]; exists {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if _, exists := resourceTypes[quota.ResourceType]; exists {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				quotaIDs[quota.ID] = struct{}{}
				resourceTypes[quota.ResourceType] = struct{}{}
			}
			sort.Slice(plan.Quotas, func(left int, right int) bool { return plan.Quotas[left].ID < plan.Quotas[right].ID })
			organization.BillingPlan = &plan
		}

		projectIDs := make(map[uint64]struct{}, len(inputOrganization.Projects))
		organization.Projects = make([]Project, 0, len(inputOrganization.Projects))
		for _, inputProject := range inputOrganization.Projects {
			if !validator.Between(inputProject.ID, uint64(1), uint64(math.MaxInt64)) ||
				!utf8.ValidString(inputProject.Name) || !validator.NotBlank(inputProject.Name) || strings.TrimSpace(inputProject.Name) != inputProject.Name || strings.ContainsRune(inputProject.Name, '\x00') {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			if _, exists := projectIDs[inputProject.ID]; exists {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			projectIDs[inputProject.ID] = struct{}{}
			project := Project{ID: inputProject.ID, Name: inputProject.Name}
			projectRoleIDs := make(map[uint64]struct{}, len(inputProject.Roles))
			project.Roles = make([]Role, 0, len(inputProject.Roles))
			for _, inputRole := range inputProject.Roles {
				if !validator.Between(inputRole.ID, uint64(1), uint64(math.MaxInt64)) || inputRole.Scope != RoleScopeProject ||
					inputRole.Subject != roleSubject || !validator.OneOf(inputRole.DefinitionType, RoleDefinitionBuiltIn, RoleDefinitionCustom) ||
					(subject.Type != IdentityUser && subject.Type != IdentityProjectCredential) ||
					!utf8.ValidString(inputRole.Key) || !validator.NotBlank(inputRole.Key) || strings.TrimSpace(inputRole.Key) != inputRole.Key || strings.ContainsRune(inputRole.Key, '\x00') ||
					!utf8.ValidString(inputRole.Name) || !validator.NotBlank(inputRole.Name) || strings.TrimSpace(inputRole.Name) != inputRole.Name || strings.ContainsRune(inputRole.Name, '\x00') ||
					!utf8.ValidString(inputRole.Description) || strings.TrimSpace(inputRole.Description) != inputRole.Description || strings.ContainsRune(inputRole.Description, '\x00') {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				if _, exists := projectRoleIDs[inputRole.ID]; exists {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				projectRoleIDs[inputRole.ID] = struct{}{}
				role := Role{ID: inputRole.ID, Key: inputRole.Key, Name: inputRole.Name, Description: inputRole.Description, Scope: inputRole.Scope, Subject: inputRole.Subject, DefinitionType: inputRole.DefinitionType}
				role.Permissions = make([]Permission, 0, len(inputRole.Permissions))
				rolePermissionIDs := make(map[uint64]struct{}, len(inputRole.Permissions))
				rolePermissionKeys := make(map[string]struct{}, len(inputRole.Permissions))
				for _, permission := range inputRole.Permissions {
					if !validator.Between(permission.ID, uint64(1), uint64(math.MaxInt64)) || permission.Scope != RoleScopeProject ||
						!strings.HasPrefix(permission.Key, "project.") || !utf8.ValidString(permission.Key) || !validator.NotBlank(permission.Key) || strings.TrimSpace(permission.Key) != permission.Key || strings.ContainsRune(permission.Key, '\x00') ||
						!utf8.ValidString(permission.Name) || !validator.NotBlank(permission.Name) || strings.TrimSpace(permission.Name) != permission.Name || strings.ContainsRune(permission.Name, '\x00') ||
						!utf8.ValidString(permission.Description) || strings.TrimSpace(permission.Description) != permission.Description || strings.ContainsRune(permission.Description, '\x00') {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					if _, exists := rolePermissionIDs[permission.ID]; exists {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					if _, exists := rolePermissionKeys[permission.Key]; exists {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					if key, exists := permissionIDs[permission.ID]; exists && key != permission.Key {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					if id, exists := permissionKeys[permission.Key]; exists && id != permission.ID {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					rolePermissionIDs[permission.ID] = struct{}{}
					rolePermissionKeys[permission.Key] = struct{}{}
					permissionIDs[permission.ID] = permission.Key
					permissionKeys[permission.Key] = permission.ID
					role.Permissions = append(role.Permissions, permission)
				}
				sort.Slice(role.Permissions, func(left int, right int) bool {
					if role.Permissions[left].Key == role.Permissions[right].Key {
						return role.Permissions[left].ID < role.Permissions[right].ID
					}
					return role.Permissions[left].Key < role.Permissions[right].Key
				})
				project.Roles = append(project.Roles, role)
			}
			sort.Slice(project.Roles, func(left int, right int) bool { return project.Roles[left].ID < project.Roles[right].ID })
			organization.Projects = append(organization.Projects, project)
		}
		sort.Slice(organization.Projects, func(left int, right int) bool {
			return organization.Projects[left].ID < organization.Projects[right].ID
		})
		value.organizations = append(value.organizations, organization)
	}
	sort.Slice(value.organizations, func(left int, right int) bool { return value.organizations[left].ID < value.organizations[right].ID })

	tenant := principal.Tenant()
	var selectedOrganization *Organization
	var selectedProject *Project
	for organizationIndex := range value.organizations {
		if value.organizations[organizationIndex].ID != tenant.OrganizationID {
			continue
		}
		selectedOrganization = &value.organizations[organizationIndex]
		for projectIndex := range selectedOrganization.Projects {
			if selectedOrganization.Projects[projectIndex].ID == tenant.ProjectID {
				selectedProject = &selectedOrganization.Projects[projectIndex]
				break
			}
		}
		break
	}
	if tenant.OrganizationID != 0 && selectedOrganization == nil || tenant.ProjectID != 0 && selectedProject == nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	switch subject.Type {
	case IdentityUser:
	case IdentityProjectCredential:
		if len(value.organizations) != 1 || selectedOrganization == nil || selectedProject == nil || len(selectedOrganization.Projects) != 1 || len(selectedOrganization.Roles) != 0 {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
	case IdentityOrganizationCredential:
		if len(value.organizations) != 1 || selectedOrganization == nil {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
	case IdentityService:
		if tenant.OrganizationID == 0 && len(value.organizations) != 0 || tenant.OrganizationID != 0 && len(value.organizations) != 1 {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
		if tenant.ProjectID != 0 && (selectedOrganization == nil || selectedProject == nil || len(selectedOrganization.Projects) != 1) {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
	}

	permissions := value.selectedPermissions()
	actor, delegated := principal.Actor()
	if delegated {
		principal, err := NewDelegatedPrincipal(subject, actor, tenant, principal.Proof(), permissions)
		if err != nil {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
		value.principal = principal
		return value, nil
	}
	principal, err := NewDirectPrincipal(subject, tenant, principal.Proof(), permissions)
	if err != nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	value.principal = principal
	return value, nil
}

// Principal returns the request Principal represented by this aggregate.
func (authentication Authentication) Principal() Principal { return authentication.principal }

// User returns a copy of the User profile when the subject is a User.
func (authentication Authentication) User() (User, bool) {
	if authentication.user == nil {
		return User{}, false
	}
	return *authentication.user, true
}

// Organizations returns a deep copy of every Organization membership.
func (authentication Authentication) Organizations() []Organization {
	organizations := make([]Organization, 0, len(authentication.organizations))
	for _, organization := range authentication.organizations {
		organizations = append(organizations, organization.copy())
	}
	return organizations
}

// Organization returns one Organization membership by ID.
func (authentication Authentication) Organization(id uint64) (Organization, bool) {
	for _, organization := range authentication.organizations {
		if organization.ID == id {
			return organization.copy(), true
		}
	}
	return Organization{}, false
}

// SelectedOrganization returns the Principal's selected Organization.
func (authentication Authentication) SelectedOrganization() (Organization, bool) {
	return authentication.Organization(authentication.principal.Tenant().OrganizationID)
}

// SelectedProject returns the Principal's selected Project.
func (authentication Authentication) SelectedProject() (Project, bool) {
	organization, ok := authentication.SelectedOrganization()
	if !ok {
		return Project{}, false
	}
	projectID := authentication.principal.Tenant().ProjectID
	for _, project := range organization.Projects {
		if project.ID == projectID {
			return project.copy(), true
		}
	}
	return Project{}, false
}

// Permissions returns the selected tenant's sorted effective Permission keys.
func (authentication Authentication) Permissions() []PermissionKey {
	return authentication.selectedPermissions()
}

func (authentication Authentication) selectedPermissions() []PermissionKey {
	tenant := authentication.principal.Tenant()
	var selectedOrganization *Organization
	var selectedProject *Project
	for organizationIndex := range authentication.organizations {
		if authentication.organizations[organizationIndex].ID != tenant.OrganizationID {
			continue
		}
		selectedOrganization = &authentication.organizations[organizationIndex]
		for projectIndex := range selectedOrganization.Projects {
			if selectedOrganization.Projects[projectIndex].ID == tenant.ProjectID {
				selectedProject = &selectedOrganization.Projects[projectIndex]
				break
			}
		}
		break
	}
	subject := authentication.principal.Subject()
	permissions := make(map[PermissionKey]struct{})
	if selectedOrganization != nil && subject.Type == IdentityUser {
		for _, role := range selectedOrganization.Roles {
			for _, permission := range role.Permissions {
				permissions[PermissionKey(permission.Key)] = struct{}{}
			}
		}
	}
	if selectedProject != nil && (subject.Type == IdentityUser || subject.Type == IdentityProjectCredential) {
		for _, role := range selectedProject.Roles {
			for _, permission := range role.Permissions {
				permissions[PermissionKey(permission.Key)] = struct{}{}
			}
		}
	}
	result := make([]PermissionKey, 0, len(permissions))
	for permission := range permissions {
		result = append(result, permission)
	}
	sort.Slice(result, func(left int, right int) bool { return result[left] < result[right] })
	return result
}

func (organization Organization) copy() Organization {
	value := Organization{ID: organization.ID, Name: organization.Name}
	value.Roles = make([]Role, 0, len(organization.Roles))
	for _, role := range organization.Roles {
		value.Roles = append(value.Roles, role.copy())
	}
	if organization.BillingPlan != nil {
		plan := *organization.BillingPlan
		plan.Quotas = append([]BillingPlanQuota(nil), plan.Quotas...)
		value.BillingPlan = &plan
	}
	value.Projects = make([]Project, 0, len(organization.Projects))
	for _, project := range organization.Projects {
		value.Projects = append(value.Projects, project.copy())
	}
	return value
}

func (project Project) copy() Project {
	value := Project{ID: project.ID, Name: project.Name, Roles: make([]Role, 0, len(project.Roles))}
	for _, role := range project.Roles {
		value.Roles = append(value.Roles, role.copy())
	}
	return value
}

func (role Role) copy() Role {
	value := role
	value.Permissions = append([]Permission(nil), role.Permissions...)
	return value
}
