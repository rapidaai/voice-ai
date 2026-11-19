package assistant_deployment_api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/authentication"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
)

func attachTestAuthentication(ginContext *gin.Context, auth authentication.Principal) {
	ctx, err := middleware_authentication.ContextWithPrincipal(ginContext.Request.Context(), auth)
	if err != nil {
		panic(err)
	}
	ginContext.Request = ginContext.Request.WithContext(ctx)
}

func testUserAuthentication(userID, organizationID, projectID uint64) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: strconv.FormatUint(userID, 10)}},
		nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}

func testProjectAuthentication(organizationID, projectID uint64) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: projectID},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: strconv.FormatUint(projectID, 10)}},
		nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}
