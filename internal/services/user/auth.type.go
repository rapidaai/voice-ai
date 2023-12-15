package internal_user_service

import (
	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	"github.com/lexatic/web-backend/pkg/types"
)

type authPrinciple struct {
	user             *internal_gorm.UserAuth
	userAuthToken    *internal_gorm.UserAuthToken
	userOrgRole      *internal_gorm.UserOrganizationRole
	userProjectRoles *[]internal_gorm.UserProjectRole
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
	if aP.userOrgRole == nil {
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

func (aP *authPrinciple) GetUserInfo() *types.UserInfo {
	return &types.UserInfo{
		Id:     aP.user.Id,
		Name:   aP.user.Name,
		Email:  aP.user.Email,
		Status: aP.user.Status,
	}
}

func (ap *authPrinciple) PlainAuthPrinciple() types.PlainAuthPrinciple {
	alt := types.PlainAuthPrinciple{
		User:  *ap.GetUserInfo(),
		Token: *ap.GetAuthToken(),
	}
	if ap.userOrgRole != nil {
		alt.OrganizationRole = *ap.GetOrganizationRole()
	}
	if ap.userProjectRoles != nil {
		alt.ProjectRoles = ap.GetProjectRoles()
	}
	return alt

}
