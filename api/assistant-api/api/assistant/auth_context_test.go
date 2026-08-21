package assistant_api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/types"
)

func attachTestAuthentication(ginContext *gin.Context, auth *types.Authentication) {
	ginContext.Request = ginContext.Request.WithContext(context.WithValue(ginContext.Request.Context(), types.CTX_, auth))
}

func testUserAuthentication(userID, organizationID, projectID uint64) *types.Authentication {
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: userID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}
	if projectID != 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}
	}
	return auth
}
