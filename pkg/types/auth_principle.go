package types

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lexatic/web-backend/pkg/utils"
)

/*
To support all the principle
*/
type PlainAuthPrinciple struct {
	User               UserInfo          `json:"user"`
	Token              AuthToken         `json:"token"`
	OrganizationRole   *OrganizaitonRole `json:"organizationRole"`
	ProjectRoles       []*ProjectRole    `json:"projectRoles"`
	CurrentProjectRole *ProjectRole      `json:"currentProjectRole"`
}

func (aP *PlainAuthPrinciple) GetAuthToken() *AuthToken {
	return &aP.Token
}

func (aP *PlainAuthPrinciple) GetOrganizationRole() *OrganizaitonRole {
	// do not return empty object
	return aP.OrganizationRole
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

/*
End of service scope
*/

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

func (u *UserInfo) GetId() uint64 {
	return u.Id
}

func (u *UserInfo) GetName() string {
	return u.Name
}

func (u *UserInfo) GetEmail() string {
	return u.Email
}

type ProjectRole struct {
	Id          uint64
	ProjectId   uint64
	Role        string
	ProjectName string
	CreatedDate time.Time
}

func (u *ProjectRole) GetRole() string {
	return u.Role
}

func (u *ProjectRole) GetProjectId() uint64 {
	return u.ProjectId
}

func GetAuthPrincipleGPRC(ctx context.Context) (Principle, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case Principle:
		return md, true
	default:
		return nil, false
	}
}

func GetClaimPrincipleGRPC[T SimplePrinciple](ctx context.Context) (SimplePrinciple, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case *PlainClaimPrinciple[T]:
		return md.Info, md.Info.IsAuthenticated()
	default:
		return nil, false
	}
}

func GetAuthPrinciple(ctx *gin.Context) (Principle, bool) {
	ath, exists := ctx.Get(CTX_)
	if exists {
		return ath.(Principle), true
	}
	return nil, false
}
