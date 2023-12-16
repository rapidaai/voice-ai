package types

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

var CTX_ string = "__auth"

type Authenticator interface {
	Authorize(ctx context.Context, authToken string, userId uint64) (Principle, error)
	AuthPrinciple(ctx context.Context, userId uint64) (Principle, error)
}

type PlainAuthPrinciple struct {
	User             UserInfo          `json:"user"`
	Token            AuthToken         `json:"token"`
	OrganizationRole *OrganizaitonRole `json:"organizationRole"`
	ProjectRoles     []*ProjectRole    `json:"projectRoles"`
}

type Principle interface {
	GetAuthToken() *AuthToken
	GetOrganizationRole() *OrganizaitonRole
	GetUserInfo() *UserInfo
	GetProjectRoles() []*ProjectRole
	PlainAuthPrinciple() PlainAuthPrinciple
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

func GetAuthPrincipleGPRC(ctx context.Context) (Principle, bool) {
	ath := ctx.Value(CTX_)
	switch md := ath.(type) {
	case Principle:
		return md, true
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
