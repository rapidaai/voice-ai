package middlewares

import (
	"context"
	"math"
	"strconv"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

type userCredential struct {
	token     string
	authID    string
	projectID string
}

func (credential userCredential) isEmpty() bool {
	return !validator.NotBlank(credential.token) && !validator.NotBlank(credential.authID)
}

func (credential userCredential) authenticate(ctx context.Context, resolver types.Authenticator) (*types.Authentication, error) {
	if !validator.NonNil(resolver) {
		return nil, errUserAuthNotSupported
	}
	if !validator.NotBlank(credential.token) || !validator.NotBlank(credential.authID) {
		return nil, errUserAuthIncomplete
	}

	expectedUserID, err := strconv.ParseUint(credential.authID, 0, 64)
	if err != nil || !validator.Between(expectedUserID, uint64(1), uint64(math.MaxInt64)) {
		return nil, errUserAuthInvalidID
	}

	principle, err := resolver.Authorize(ctx, credential.token, expectedUserID)
	if err != nil || !validator.NonNil(principle) {
		return nil, errUserAuthRejected
	}

	if validator.NotBlank(credential.projectID) {
		projectID, err := strconv.ParseUint(credential.projectID, 0, 64)
		if err != nil || !validator.NonZero(projectID) {
			return nil, errUserAuthInvalidProjectID
		}
		if err := principle.SwitchProject(projectID); err != nil {
			return nil, errUserAuthProjectSelectionRejected
		}
	}

	userID, hasUser := principle.UserIdentity()
	if !hasUser || userID != expectedUserID || principle.Type() != types.AuthTypeUser {
		return nil, errUserAuthInvalidAuditActor
	}
	actorProvider, hasActorProvider := principle.(types.ActorIdentityProvider)
	if !hasActorProvider {
		return nil, errUserAuthInvalidAuditActor
	}
	actor, hasActor := actorProvider.AuditActor()
	if !hasActor || actor.Validate() != nil || actor.Type != types.ActorTypeUser || actor.ID != userID {
		return nil, errUserAuthInvalidAuditActor
	}

	auth := &types.Authentication{
		AuthType:   types.AuthTypeUser,
		ActorValue: &actor,
		UserValue:  &types.UserContext{UserID: userID},
	}
	organizationID, hasOrganization := principle.OrganizationContext()
	projectRole := principle.GetCurrentProjectRole()
	if !hasOrganization {
		if projectRole != nil {
			return nil, errUserAuthRejected
		}
		return auth, nil
	}

	auth.OrganizationValue = &types.OrganizationContext{OrganizationID: organizationID}
	if projectRole != nil && projectRole.ProjectId != 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectRole.ProjectId}
	}
	return auth, nil
}
