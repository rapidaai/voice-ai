package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewOrganizationAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.OrganizationScope], logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authToken, ok := c.GetQuery(types.ORG_SCOPE_KEY)
		if !ok {
			c.Next()
			return
		}

		auth, err := resolver.Claim(c, authToken)
		if err != nil {
			c.Next()
			return
		}

		// Attach the user information to the context
		c.Set(string(types.CTX_), auth)
		c.Next()
	}
}
