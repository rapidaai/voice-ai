// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rapidaai/pkg/authentication/authorizer"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

// NewProjectHTTPMiddleware authenticates one HTTP Project credential.
func NewProjectHTTPMiddleware(projectResolver *resolver.ProjectPrincipalResolver, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !validator.NonNil(projectResolver) {
			logger.Errorf("project authentication middleware is not configured")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}

		headerValues, hasHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.PROJECT_SCOPE_KEY)]
		queryValues, hasQuery := ctx.Request.URL.Query()[types.PROJECT_SCOPE_KEY]
		pathValue, hasPath := ctx.Params.Get(types.PROJECT_SCOPE_KEY)
		if !hasHeader && !hasQuery && !hasPath {
			ctx.Next()
			return
		}
		if _, exists := Principal(ctx.Request.Context()); exists {
			logger.Errorf("project authentication conflicts with an attached principal")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}

		credential := ""
		headerSelected := false
		for _, value := range headerValues {
			if strings.TrimSpace(value) != "" {
				headerSelected = true
				break
			}
		}
		if headerSelected {
			if len(headerValues) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			credential = headerValues[0]
		} else {
			querySelected := false
			for _, value := range queryValues {
				if strings.TrimSpace(value) != "" {
					querySelected = true
					break
				}
			}
			if querySelected {
				if len(queryValues) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				credential = queryValues[0]
			} else if strings.TrimSpace(pathValue) != "" {
				credential = pathValue
			}
		}

		token := strings.TrimPrefix(strings.TrimSpace(credential), types.PROJECT_KEY_PREFIX)
		if !validator.NotBlank(token) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		value, err := projectResolver.Resolve(ctx.Request.Context(), resolver.ProjectCredential{Token: token})
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		if err != nil {
			logger.Errorf("project authentication rejected")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		requestContext, err := ContextWithPrincipal(ctx.Request.Context(), value)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}

// NewOrganizationHTTPMiddleware authenticates one HTTP Organization credential.
func NewOrganizationHTTPMiddleware(organizationResolver *resolver.OrganizationPrincipalResolver, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !validator.NonNil(organizationResolver) {
			logger.Errorf("organization authentication middleware is not configured")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}

		headerValues, hasHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.ORG_SCOPE_KEY)]
		queryValues, hasQuery := ctx.Request.URL.Query()[types.ORG_SCOPE_KEY]
		pathValue, hasPath := ctx.Params.Get(types.ORG_SCOPE_KEY)
		if !hasHeader && !hasQuery && !hasPath {
			ctx.Next()
			return
		}
		if _, exists := Principal(ctx.Request.Context()); exists {
			logger.Errorf("organization authentication conflicts with an attached principal")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}

		credential := ""
		headerSelected := false
		for _, value := range headerValues {
			if strings.TrimSpace(value) != "" {
				headerSelected = true
				break
			}
		}
		if headerSelected {
			if len(headerValues) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			credential = headerValues[0]
		} else {
			querySelected := false
			for _, value := range queryValues {
				if strings.TrimSpace(value) != "" {
					querySelected = true
					break
				}
			}
			if querySelected {
				if len(queryValues) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				credential = queryValues[0]
			} else if strings.TrimSpace(pathValue) != "" {
				credential = pathValue
			}
		}
		credential = strings.TrimSpace(credential)
		if !validator.NotBlank(credential) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		value, err := organizationResolver.Resolve(ctx.Request.Context(), resolver.OrganizationCredential{Token: credential})
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		if err != nil {
			logger.Errorf("organization authentication rejected")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		requestContext, err := ContextWithPrincipal(ctx.Request.Context(), value)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}

// NewServiceHTTPMiddleware authenticates one HTTP Service assertion.
func NewServiceHTTPMiddleware(serviceResolver *resolver.ServicePrincipalResolver, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !validator.NonNil(serviceResolver) {
			logger.Errorf("service authentication middleware is not configured")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}

		headerValues, hasHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.SERVICE_SCOPE_KEY)]
		queryValues, hasQuery := ctx.Request.URL.Query()[types.SERVICE_SCOPE_KEY]
		pathValue, hasPath := ctx.Params.Get(types.SERVICE_SCOPE_KEY)
		if !hasHeader && !hasQuery && !hasPath {
			ctx.Next()
			return
		}
		if _, exists := Principal(ctx.Request.Context()); exists {
			logger.Errorf("service authentication conflicts with an attached principal")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}

		assertion := ""
		headerSelected := false
		for _, value := range headerValues {
			if strings.TrimSpace(value) != "" {
				headerSelected = true
				break
			}
		}
		if headerSelected {
			if len(headerValues) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			assertion = headerValues[0]
		} else {
			querySelected := false
			for _, value := range queryValues {
				if strings.TrimSpace(value) != "" {
					querySelected = true
					break
				}
			}
			if querySelected {
				if len(queryValues) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				assertion = queryValues[0]
			} else if strings.TrimSpace(pathValue) != "" {
				assertion = pathValue
			}
		}
		assertion = strings.TrimSpace(assertion)
		if !validator.NotBlank(assertion) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		value, err := serviceResolver.Resolve(ctx.Request.Context(), resolver.ServiceAssertionCredential{Assertion: assertion})
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		if err != nil {
			logger.Errorf("service authentication rejected")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		requestContext, err := ContextWithPrincipal(ctx.Request.Context(), value)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}

// NewUserHTTPMiddleware authenticates one HTTP User session and tenant selection.
func NewUserHTTPMiddleware(userResolver *resolver.UserPrincipalResolver, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !validator.NonNil(userResolver) {
			logger.Errorf("user authentication middleware is not configured")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}

		authorizationHeaders, hasAuthorizationHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.AUTHORIZATION_KEY)]
		authorizationQuery, hasAuthorizationQuery := ctx.Request.URL.Query()[types.AUTHORIZATION_KEY]
		authorizationPath, hasAuthorizationPath := ctx.Params.Get(types.AUTHORIZATION_KEY)
		userHeaders, hasUserHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.AUTH_KEY)]
		userQuery, hasUserQuery := ctx.Request.URL.Query()[types.AUTH_KEY]
		userPath, hasUserPath := ctx.Params.Get(types.AUTH_KEY)
		organizationHeaders, hasOrganizationHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.ORGANIZATION_KEY)]
		projectHeaders, hasProjectHeader := ctx.Request.Header[http.CanonicalHeaderKey(types.PROJECT_KEY)]
		projectQuery, hasProjectQuery := ctx.Request.URL.Query()[types.PROJECT_KEY]
		projectPath, hasProjectPath := ctx.Params.Get(types.PROJECT_KEY)
		if !hasAuthorizationHeader && !hasAuthorizationQuery && !hasAuthorizationPath &&
			!hasUserHeader && !hasUserQuery && !hasUserPath && !hasOrganizationHeader &&
			!hasProjectHeader && !hasProjectQuery && !hasProjectPath {
			ctx.Next()
			return
		}
		if _, exists := Principal(ctx.Request.Context()); exists {
			logger.Errorf("user authentication conflicts with an attached principal")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}

		token := ""
		if strings.TrimSpace(authorizationPath) != "" {
			token = authorizationPath
		} else {
			headerSelected := false
			for _, value := range authorizationHeaders {
				if strings.TrimSpace(value) != "" {
					headerSelected = true
					break
				}
			}
			if headerSelected {
				if len(authorizationHeaders) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				token = authorizationHeaders[0]
			} else {
				querySelected := false
				for _, value := range authorizationQuery {
					if strings.TrimSpace(value) != "" {
						querySelected = true
						break
					}
				}
				if querySelected {
					if len(authorizationQuery) != 1 {
						ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
						return
					}
					token = authorizationQuery[0]
				}
			}
		}

		userIDValue := ""
		userHeaderSelected := false
		for _, value := range userHeaders {
			if strings.TrimSpace(value) != "" {
				userHeaderSelected = true
				break
			}
		}
		if userHeaderSelected {
			if len(userHeaders) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			userIDValue = userHeaders[0]
		} else if strings.TrimSpace(userPath) != "" {
			userIDValue = userPath
		} else {
			querySelected := false
			for _, value := range userQuery {
				if strings.TrimSpace(value) != "" {
					querySelected = true
					break
				}
			}
			if querySelected {
				if len(userQuery) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				userIDValue = userQuery[0]
			}
		}

		organizationIDValue := ""
		if hasOrganizationHeader {
			if len(organizationHeaders) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			organizationIDValue = strings.TrimSpace(organizationHeaders[0])
			if !validator.NotBlank(organizationIDValue) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
		}

		projectIDValue := ""
		projectHeaderSelected := false
		for _, value := range projectHeaders {
			if strings.TrimSpace(value) != "" {
				projectHeaderSelected = true
				break
			}
		}
		if projectHeaderSelected {
			if len(projectHeaders) != 1 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
			projectIDValue = projectHeaders[0]
		} else if strings.TrimSpace(projectPath) != "" {
			projectIDValue = projectPath
		} else {
			querySelected := false
			for _, value := range projectQuery {
				if strings.TrimSpace(value) != "" {
					querySelected = true
					break
				}
			}
			if querySelected {
				if len(projectQuery) != 1 {
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
					return
				}
				projectIDValue = projectQuery[0]
			}
		}

		token = strings.TrimSpace(token)
		userIDValue = strings.TrimSpace(userIDValue)
		projectIDValue = strings.TrimSpace(projectIDValue)
		if !validator.NotBlank(token) || !validator.NotBlank(userIDValue) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if (hasProjectHeader || hasProjectQuery || hasProjectPath) && !validator.NotBlank(projectIDValue) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		userID, err := utils.StringToUint64(userIDValue)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		organizationID := uint64(0)
		if organizationIDValue != "" {
			organizationID, err = utils.StringToUint64(organizationIDValue)
			if err != nil || !validator.Between(organizationID, uint64(1), uint64(math.MaxInt64)) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
		}
		projectID := uint64(0)
		if projectIDValue != "" {
			projectID, err = utils.StringToUint64(projectIDValue)
			if err != nil || !validator.Between(projectID, uint64(1), uint64(math.MaxInt64)) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
				return
			}
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		value, err := userResolver.Resolve(ctx.Request.Context(), resolver.UserCredential{Token: token, UserID: userID, OrganizationID: organizationID, ProjectID: projectID})
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}
		if err != nil {
			logger.Errorf("user authentication rejected")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		requestContext, err := ContextWithPrincipal(ctx.Request.Context(), value)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		ctx.Request = ctx.Request.WithContext(requestContext)
		ctx.Next()
	}
}

// NewPolicyHTTPMiddleware authorizes the current HTTP route against its attached principal.
func NewPolicyHTTPMiddleware(policy ServicePolicy, logger commons.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !validator.NonNil(policy) {
			logger.Errorf("route authorization middleware is not configured")
			ctx.AbortWithStatusJSON(http.StatusForbidden, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthorizationFailureError{}).Error()})
			return
		}
		if ctx.Request.Context().Err() != nil {
			ctx.Abort()
			return
		}

		route := ctx.Request.Method + " " + ctx.FullPath()
		routePolicy, exists := policy.HTTPPolicy(route)
		if !exists || routePolicy.Validate() != nil {
			logger.Errorf("principal authorization denied for route %s", route)
			ctx.AbortWithStatusJSON(http.StatusForbidden, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthorizationFailureError{}).Error()})
			return
		}
		if routePolicy.Authorization == authorizer.AuthorizationPublic {
			ctx.Request = ctx.Request.WithContext(contextWithoutPrincipal(ctx.Request.Context()))
			ctx.Next()
			return
		}
		if routePolicy.Authorization == authorizer.AuthorizationDisabled {
			logger.Errorf("principal authorization denied for route %s", route)
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthorizationFailureError{}).Error()})
			return
		}
		value, exists := Principal(ctx.Request.Context())
		if !exists {
			logger.Errorf("principal authentication missing for route %s", route)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthenticationFailureError{}).Error()})
			return
		}
		if (authorizer.RouteAuthorizer{}).Authorize(&value, routePolicy) != nil {
			logger.Errorf("principal authorization denied for route %s", route)
			ctx.AbortWithStatusJSON(http.StatusForbidden, pkg_errors.ErrorResponse{Error: (pkg_errors.AuthorizationFailureError{}).Error()})
			return
		}
		ctx.Next()
	}
}
