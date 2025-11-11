package types

import (
	"errors"
	"time"

	"github.com/rapidaai/pkg/utils"
)

/*
To support all the principle
*/
type PlainAuthPrinciple struct {
	User               UserInfo             `json:"user"`
	Token              AuthToken            `json:"token"`
	OrganizationRole   *OrganizaitonRole    `json:"organizationRole"`
	ProjectRoles       []*ProjectRole       `json:"projectRoles"`
	CurrentProjectRole *ProjectRole         `json:"currentProjectRole"`
	FeaturePermissions []*FeaturePermission `json:"featurePermissions"`
	CurrentToken       string               `json:"currentToken"`
}

type FeaturePermission struct {
	Id       uint64
	Feature  string
	IsEnable bool
}

type OrganizaitonRole struct {
	Id               uint64
	OrganizationId   uint64
	Role             string
	OrganizationName string
}

type AuthToken struct {
	Id        uint64
	Token     string
	TokenType string
	IsExpired bool
}

type UserInfo struct {
	Id     uint64
	Name   string
	Email  string
	Status string
}

type ProjectRole struct {
	Id          uint64
	ProjectId   uint64
	Role        string
	ProjectName string
	CreatedDate time.Time
}

func (u *UserInfo) GetId() uint64 {
	return u.Id
}

func (u *UserInfo) GetName() string {
	return u.Name
}

func (u *UserInfo) GetEmail() string {
	return u.Email
}

func (u *ProjectRole) GetRole() string {
	return u.Role
}

func (u *ProjectRole) GetProjectId() uint64 {
	return u.ProjectId
}

func (aP *PlainAuthPrinciple) GetAuthToken() *AuthToken {
	return &aP.Token
}

func (aP *PlainAuthPrinciple) GetCurrentProjectRole() *ProjectRole {
	return aP.CurrentProjectRole
}

func (aP *PlainAuthPrinciple) HasOrganization() bool {
	return aP.OrganizationRole != nil
}

func (aP *PlainAuthPrinciple) PlainAuthPrinciple() PlainAuthPrinciple {
	return *aP
}

func (aP *PlainAuthPrinciple) HasProject() bool {
	return aP.GetCurrentProjectRole() != nil
}

func (aP *PlainAuthPrinciple) GetCurrentToken() string {
	return aP.CurrentToken
}

func (aP *PlainAuthPrinciple) HasUser() bool {
	return aP.GetUserInfo() != nil
}

func (aP *PlainAuthPrinciple) GetOrganizationRole() *OrganizaitonRole {
	// do not return empty object
	return aP.OrganizationRole
}

func (aP *PlainAuthPrinciple) GetFeaturePermission() []*FeaturePermission {
	return aP.FeaturePermissions
}

func (aP *PlainAuthPrinciple) IsFeatureEnabled(featureName string) bool {
	if aP.FeaturePermissions == nil {
		return false
	}

	for _, f := range aP.FeaturePermissions {
		if f.Feature == featureName {
			return true
		}
	}
	return false
}

func (aP *PlainAuthPrinciple) IsAuthenticated() bool {
	return aP.HasOrganization() && aP.HasUser()
}

func (aP *PlainAuthPrinciple) GetProjectRoles() []*ProjectRole {
	return aP.ProjectRoles
}

func (aP *PlainAuthPrinciple) GetUserInfo() *UserInfo {
	return &aP.User
}

func (aP *PlainAuthPrinciple) GetUserId() *uint64 {
	return &aP.User.Id
}

func (aP *PlainAuthPrinciple) GetCurrentOrganizationId() *uint64 {
	return &aP.OrganizationRole.OrganizationId
}

func (aP *PlainAuthPrinciple) GetCurrentProjectId() *uint64 {
	if aP.CurrentProjectRole != nil && aP.CurrentProjectRole.ProjectId > 0 {
		return &aP.CurrentProjectRole.ProjectId
	}
	return nil
}

func (aP *PlainAuthPrinciple) SwitchProject(projectId uint64) error {
	idx := utils.IndexFunc(aP.GetProjectRoles(), func(pRole *ProjectRole) bool {
		return pRole.ProjectId == projectId
	})
	if idx == -1 {
		return errors.New("illegal project id for user")
	}
	aP.CurrentProjectRole = aP.ProjectRoles[idx]
	return nil
}

func (aP *PlainAuthPrinciple) Type() string {
	return "user"
}
