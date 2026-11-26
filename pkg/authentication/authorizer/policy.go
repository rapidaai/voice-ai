// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authorizer

import (
	"math"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

// Authorization identifies one route admission mode.
type Authorization string

const (
	AuthorizationPublic        Authorization = "public"
	AuthorizationAuthenticated Authorization = "authenticated"
	AuthorizationPermission    Authorization = "permission"
	AuthorizationAnyOf         Authorization = "any_of"
	AuthorizationDisabled      Authorization = "disabled"
)

// Context identifies the tenant shape required by a policy decision.
type Context string

const (
	ContextUser         Context = "user"
	ContextOrganization Context = "organization"
	ContextProject      Context = "project"
	ContextService      Context = "service"
)

// ProjectTokenAudience identifies the minimum Project-token audience.
type ProjectTokenAudience string

const (
	ProjectTokenAudienceManagement ProjectTokenAudience = "management"
	ProjectTokenAudienceRuntime    ProjectTokenAudience = "runtime"
)

// ProjectTokenBinding identifies the tenant binding required for a Project token.
type ProjectTokenBinding string

// ProjectTokenBindingProject requires a credential bound to the selected Project.
const ProjectTokenBindingProject ProjectTokenBinding = "project"

// CallerServicePolicy identifies one allowed immediate service caller.
type CallerServicePolicy struct {
	Issuer  string `yaml:"issuer"`
	ActorID uint64 `yaml:"actor_id"`
}

// ProjectTokenPolicy declares the minimum audience and binding for Project subjects.
type ProjectTokenPolicy struct {
	Audience ProjectTokenAudience `yaml:"audience"`
	Binding  ProjectTokenBinding  `yaml:"binding"`
}

// AlternativePolicy declares one complete route authorization branch.
type AlternativePolicy struct {
	Actors         []authentication.IdentityType  `yaml:"actors"`
	CallerServices []CallerServicePolicy          `yaml:"caller_services,omitempty"`
	Context        Context                        `yaml:"context"`
	ProjectToken   *ProjectTokenPolicy            `yaml:"project_token,omitempty"`
	Authenticated  bool                           `yaml:"authenticated,omitempty"`
	Permissions    []authentication.PermissionKey `yaml:"permissions,omitempty"`
}

// RoutePolicy is one typed route-policy entry.
type RoutePolicy struct {
	Authorization  Authorization                  `yaml:"authorization"`
	Actors         []authentication.IdentityType  `yaml:"actors,omitempty"`
	CallerServices []CallerServicePolicy          `yaml:"caller_services,omitempty"`
	Context        Context                        `yaml:"context,omitempty"`
	ProjectToken   *ProjectTokenPolicy            `yaml:"project_token,omitempty"`
	Permissions    []authentication.PermissionKey `yaml:"permissions,omitempty"`
	AnyOf          []AlternativePolicy            `yaml:"any_of,omitempty"`
}

// Validate rejects policy shapes that cannot be evaluated safely.
func (policy RoutePolicy) Validate() error {
	if !policy.isValid() {
		return pkg_errors.AuthenticationPolicyInvalidError{}
	}
	return nil
}

// Snapshot returns a detached copy suitable for a service policy lookup.
func (policy RoutePolicy) Snapshot() RoutePolicy {
	policy.Actors = append([]authentication.IdentityType(nil), policy.Actors...)
	policy.CallerServices = append([]CallerServicePolicy(nil), policy.CallerServices...)
	policy.Permissions = append([]authentication.PermissionKey(nil), policy.Permissions...)
	if policy.ProjectToken != nil {
		projectToken := *policy.ProjectToken
		policy.ProjectToken = &projectToken
	}
	if policy.AnyOf != nil {
		alternatives := make([]AlternativePolicy, len(policy.AnyOf))
		for index, alternative := range policy.AnyOf {
			alternative.Actors = append([]authentication.IdentityType(nil), alternative.Actors...)
			alternative.CallerServices = append([]CallerServicePolicy(nil), alternative.CallerServices...)
			alternative.Permissions = append([]authentication.PermissionKey(nil), alternative.Permissions...)
			if alternative.ProjectToken != nil {
				projectToken := *alternative.ProjectToken
				alternative.ProjectToken = &projectToken
			}
			alternatives[index] = alternative
		}
		policy.AnyOf = alternatives
	}
	return policy
}

func (policy RoutePolicy) isValid() bool {
	if policy.Authorization == AuthorizationPublic || policy.Authorization == AuthorizationDisabled {
		return len(policy.Actors) == 0 && len(policy.CallerServices) == 0 && policy.Context == "" &&
			policy.ProjectToken == nil && len(policy.Permissions) == 0 && len(policy.AnyOf) == 0
	}
	if policy.Authorization == AuthorizationAnyOf {
		if len(policy.Actors) != 0 || len(policy.CallerServices) != 0 || policy.Context != "" ||
			policy.ProjectToken != nil || len(policy.Permissions) != 0 || len(policy.AnyOf) == 0 {
			return false
		}
		for index, alternative := range policy.AnyOf {
			if !alternative.isValid() {
				return false
			}
			for previous := 0; previous < index; previous++ {
				if reflect.DeepEqual(policy.AnyOf[previous], alternative) {
					return false
				}
			}
		}
		return true
	}
	if policy.Authorization != AuthorizationAuthenticated && policy.Authorization != AuthorizationPermission {
		return false
	}
	if len(policy.Actors) == 0 {
		return false
	}
	hasProjectActor := false
	actors := make(map[authentication.IdentityType]struct{}, len(policy.Actors))
	for _, actor := range policy.Actors {
		switch actor {
		case authentication.IdentityUser, authentication.IdentityProjectCredential,
			authentication.IdentityOrganizationCredential, authentication.IdentityService:
		default:
			return false
		}
		if _, exists := actors[actor]; exists {
			return false
		}
		actors[actor] = struct{}{}
		if actor == authentication.IdentityProjectCredential {
			hasProjectActor = true
		}
	}
	if hasProjectActor != (policy.ProjectToken != nil) {
		return false
	}
	if policy.ProjectToken != nil && !policy.ProjectToken.isValid() {
		return false
	}
	if policy.Context == ContextService {
		if len(policy.Actors) != 1 || policy.Actors[0] != authentication.IdentityService || len(policy.CallerServices) == 0 {
			return false
		}
	}

	callers := make(map[CallerServicePolicy]struct{}, len(policy.CallerServices))
	for _, caller := range policy.CallerServices {
		if !caller.isValid() {
			return false
		}
		if _, exists := callers[caller]; exists {
			return false
		}
		callers[caller] = struct{}{}
	}

	switch policy.Authorization {
	case AuthorizationAuthenticated:
		return policy.Context.isConcrete() && len(policy.Permissions) == 0 && len(policy.AnyOf) == 0
	case AuthorizationPermission:
		return policy.Context.isConcrete() && len(policy.AnyOf) == 0 && policy.hasValidPermissions()
	default:
		return false
	}
}

func (alternative AlternativePolicy) isValid() bool {
	if len(alternative.Actors) == 0 || !alternative.Context.isConcrete() ||
		alternative.Authenticated == (len(alternative.Permissions) > 0) {
		return false
	}
	authorization := AuthorizationPermission
	if alternative.Authenticated {
		authorization = AuthorizationAuthenticated
	}
	return RoutePolicy{
		Authorization:  authorization,
		Actors:         alternative.Actors,
		CallerServices: alternative.CallerServices,
		Context:        alternative.Context,
		ProjectToken:   alternative.ProjectToken,
		Permissions:    alternative.Permissions,
	}.isValid()
}

func (policy RoutePolicy) hasValidPermissions() bool {
	if len(policy.Permissions) == 0 {
		return false
	}
	permissions := make(map[authentication.PermissionKey]struct{}, len(policy.Permissions))
	for _, permission := range policy.Permissions {
		value := string(permission)
		if !utf8.ValidString(value) || !validator.NotBlank(value) || strings.TrimSpace(value) != value ||
			strings.ContainsRune(value, '\x00') {
			return false
		}
		switch policy.Context {
		case ContextOrganization:
			if !strings.HasPrefix(value, "organization.") {
				return false
			}
		case ContextProject:
			if !strings.HasPrefix(value, "organization.") && !strings.HasPrefix(value, "project.") {
				return false
			}
		default:
			return false
		}
		if _, exists := permissions[permission]; exists {
			return false
		}
		permissions[permission] = struct{}{}
	}
	return true
}

func (caller CallerServicePolicy) isValid() bool {
	return utf8.ValidString(caller.Issuer) && validator.NotBlank(caller.Issuer) &&
		strings.TrimSpace(caller.Issuer) == caller.Issuer && !strings.ContainsRune(caller.Issuer, '\x00') &&
		validator.Between(caller.ActorID, uint64(1), uint64(math.MaxInt64))
}

func (policy ProjectTokenPolicy) isValid() bool {
	return (policy.Audience == ProjectTokenAudienceManagement || policy.Audience == ProjectTokenAudienceRuntime) &&
		policy.Binding == ProjectTokenBindingProject
}

func (context Context) isConcrete() bool {
	return context == ContextUser || context == ContextOrganization || context == ContextProject || context == ContextService
}
