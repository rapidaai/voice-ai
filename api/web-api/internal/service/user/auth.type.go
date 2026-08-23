package internal_user_service

import (
	"errors"
	"math"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
)

type authPrinciple struct {
	user               *internal_entity.UserAuth
	userAuthToken      *internal_entity.UserAuthToken
	userOrgRole        *internal_entity.UserOrganizationRole
	userProjectRoles   []*internal_entity.UserProjectRole
	currentProjectRole *types.ProjectRole
	featurePermissions []*internal_entity.UserFeaturePermission
}

func (aP *authPrinciple) GetAuthToken() *types.AuthToken {
	return &types.AuthToken{
		Id:        aP.userAuthToken.Id,
		Token:     aP.userAuthToken.Token,
		TokenType: aP.userAuthToken.TokenType,
		IsExpired: aP.userAuthToken.IsExpired(),
	}
}

func (aP *authPrinciple) GetOrganizationRole() *types.OrganizaitonRole {
	// do not return empty object
	if aP.userOrgRole == nil || (*aP.userOrgRole) == (internal_entity.UserOrganizationRole{}) {
		return nil
	}
	return &types.OrganizaitonRole{
		Id:               aP.userOrgRole.Id,
		OrganizationId:   aP.userOrgRole.OrganizationId,
		Role:             aP.userOrgRole.Role,
		OrganizationName: aP.userOrgRole.Organization.Name,
	}
}

func (aP *authPrinciple) GetProjectRoles() []*types.ProjectRole {
	if aP.userProjectRoles == nil {
		return nil
	}

	if aP.userProjectRoles != nil && len(aP.userProjectRoles) == 0 {
		return nil
	}

	prs := make([]*types.ProjectRole, len(aP.userProjectRoles))
	for idx, pr := range aP.userProjectRoles {
		prs[idx] = &types.ProjectRole{
			Id:          pr.Id,
			ProjectId:   pr.ProjectId,
			Role:        pr.Role,
			ProjectName: pr.Project.Name,
		}
	}
	return prs
}

func (aP *authPrinciple) GetFeaturePermission() []*types.FeaturePermission {
	if aP.featurePermissions == nil {
		return nil
	}

	if aP.featurePermissions != nil && len(aP.featurePermissions) == 0 {
		return nil
	}

	prs := make([]*types.FeaturePermission, len(aP.featurePermissions))
	for idx, pr := range aP.featurePermissions {
		prs[idx] = &types.FeaturePermission{
			Id:       pr.Id,
			Feature:  pr.Feature,
			IsEnable: pr.IsEnabled,
		}
	}
	return prs
}

func (aP *authPrinciple) GetUserInfo() *types.UserInfo {
	return &types.UserInfo{
		Id:     aP.user.Id,
		Name:   aP.user.Name,
		Email:  aP.user.Email,
		Status: aP.user.Status.String(),
	}
}

func (ap *authPrinciple) PlainAuthPrinciple() types.PlainAuthPrinciple {
	alt := types.PlainAuthPrinciple{
		User:  *ap.GetUserInfo(),
		Token: *ap.GetAuthToken(),
	}
	alt.OrganizationRole = ap.GetOrganizationRole()
	alt.ProjectRoles = ap.GetProjectRoles()
	alt.FeaturePermissions = ap.GetFeaturePermission()
	return alt
}

func (aP *authPrinciple) SwitchProject(projectId uint64) error {
	prj := aP.GetProjectRoles()
	idx := utils.IndexFunc(prj, func(pRole *types.ProjectRole) bool {
		return pRole.ProjectId == projectId
	})
	if idx == -1 {
		return errors.New("illegal project id for user")
	}
	aP.currentProjectRole = prj[idx]
	return nil
}

func (aP *authPrinciple) UserIdentity() (uint64, bool) {
	if aP.user == nil || aP.user.Id == 0 {
		return 0, false
	}
	return aP.user.Id, true
}

func (aP *authPrinciple) AuditActor() (types.ActorIdentity, bool) {
	userID, ok := aP.UserIdentity()
	if !ok || userID > math.MaxInt64 {
		return types.ActorIdentity{}, false
	}
	return types.ActorIdentity{Type: types.ActorTypeUser, ID: userID}, true
}

func (aP *authPrinciple) OrganizationContext() (uint64, bool) {
	organizationRole := aP.GetOrganizationRole()
	if organizationRole == nil || organizationRole.OrganizationId == 0 {
		return 0, false
	}
	return organizationRole.OrganizationId, true
}

func (aP *authPrinciple) ProjectContext() (types.ProjectContext, bool) {
	organizationID, ok := aP.OrganizationContext()
	if !ok || aP.currentProjectRole == nil || aP.currentProjectRole.ProjectId == 0 {
		return types.ProjectContext{}, false
	}
	return types.ProjectContext{
		OrganizationID: organizationID,
		ProjectID:      aP.currentProjectRole.ProjectId,
	}, true
}

func (aP *authPrinciple) GetCurrentProjectRole() *types.ProjectRole {
	if aP.currentProjectRole == nil {
		return nil
	}
	return aP.currentProjectRole
}

func (aP *authPrinciple) IsAuthenticated() bool {
	_, hasUser := aP.UserIdentity()
	_, hasOrganization := aP.OrganizationContext()
	return hasUser && hasOrganization
}

func (aP *authPrinciple) GetCurrentToken() string {
	tk := aP.GetAuthToken()
	if tk != nil {
		return tk.Token
	}
	return ""
}

func (ap *authPrinciple) Type() types.AuthType {
	return types.AuthTypeUser
}
