package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func NewProjectAuthenticatorMiddleware(resolver types.ClaimAuthenticator[*types.ProjectScope], logger commons.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var authToken string
		authToken, ok := c.GetQuery(types.PROJECT_SCOPE_KEY)
		if !ok || authToken == "" {
			authToken = c.Param(types.PROJECT_SCOPE_KEY)
		}
		if authToken == "" {
			c.Next()
			return
		}
		auth, err := resolver.Claim(c, authToken)
		if err != nil {
			c.Next()
			return
		}
		c.Set(string(types.CTX_), auth)
		c.Next()
	}
}
