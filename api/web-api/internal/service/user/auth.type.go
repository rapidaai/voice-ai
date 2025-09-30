package internal_user_service

import (
	"errors"

	internal_entity "github.com/lexatic/web-backend/api/web-api/internal/entity"
	"github.com/lexatic/web-backend/pkg/types"
	"github.com/lexatic/web-backend/pkg/utils"
)

type authPrinciple struct {
	user               *internal_entity.UserAuth
	userAuthToken      *internal_entity.UserAuthToken
	userOrgRole        *internal_entity.UserOrganizationRole
	userProjectRoles   *[]internal_entity.UserProjectRole
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

	if aP.userProjectRoles != nil && len(*aP.userProjectRoles) == 0 {
		return nil
	}

	prs := make([]*types.ProjectRole, len(*aP.userProjectRoles))
	for idx, pr := range *aP.userProjectRoles {
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

func (aP *authPrinciple) GetUserId() *uint64 {
	return &aP.user.Id
}

func (aP *authPrinciple) GetCurrentOrganizationId() *uint64 {
	if aP.GetOrganizationRole() != nil {
		return &aP.GetOrganizationRole().OrganizationId
	}

	return nil
}

func (aP *authPrinciple) GetCurrentProjectId() *uint64 {
	if aP.currentProjectRole == nil {
		return nil
	}
	return &aP.currentProjectRole.ProjectId
}

func (aP *authPrinciple) GetCurrentProjectRole() *types.ProjectRole {
	panic("illegal of calling current project role when not selected")
}

// has an user
func (aP *authPrinciple) HasUser() bool {
	return aP.GetUserId() != nil
}

// has an org
func (aP *authPrinciple) HasOrganization() bool {
	return aP.GetCurrentOrganizationId() != nil
}

// has an project
func (aP *authPrinciple) HasProject() bool {
	return aP.GetCurrentProjectId() != nil
}

func (aP *authPrinciple) IsAuthenticated() bool {
	return aP.HasOrganization() && aP.HasUser()
}

func (aP *authPrinciple) GetCurrentToken() string {
	tk := aP.GetAuthToken()
	if tk != nil {
		return tk.Token
	}
	return ""
}

func (ap *authPrinciple) Type() string {
	return "user"
}
