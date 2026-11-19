// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"context"
	"math"
	"strconv"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rapidaai/pkg/authentication/authorizer"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

// NewProjectUnaryMiddleware authenticates Project credentials for unary gRPC requests.
func NewProjectUnaryMiddleware(principalResolver *resolver.ProjectPrincipalResolver, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !validator.NonNil(principalResolver) {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.PROJECT_SCOPE_KEY]
		if !presented {
			return handler(ctx, req)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimPrefix(strings.TrimSpace(values[0]), types.PROJECT_KEY_PREFIX)
		if !validator.NotBlank(token) {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.ProjectCredential{Token: token})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return nil, status.Error(codes.Canceled, ctxErr.Error())
				}
				return nil, status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		return handler(resolvedContext, req)
	}
}

// NewProjectStreamMiddleware authenticates Project credentials for streaming gRPC requests.
func NewProjectStreamMiddleware(principalResolver *resolver.ProjectPrincipalResolver, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if !validator.NonNil(principalResolver) {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.PROJECT_SCOPE_KEY]
		if !presented {
			return handler(srv, stream)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimPrefix(strings.TrimSpace(values[0]), types.PROJECT_KEY_PREFIX)
		if !validator.NotBlank(token) {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.ProjectCredential{Token: token})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return status.Error(codes.Canceled, ctxErr.Error())
				}
				return status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("project authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = resolvedContext
		return handler(srv, wrapped)
	}
}

// NewOrganizationUnaryMiddleware authenticates Organization credentials for unary gRPC requests.
func NewOrganizationUnaryMiddleware(principalResolver *resolver.OrganizationPrincipalResolver, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !validator.NonNil(principalResolver) {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.ORG_SCOPE_KEY]
		if !presented {
			return handler(ctx, req)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimSpace(values[0])
		if !validator.NotBlank(token) {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.OrganizationCredential{Token: token})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return nil, status.Error(codes.Canceled, ctxErr.Error())
				}
				return nil, status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		return handler(resolvedContext, req)
	}
}

// NewOrganizationStreamMiddleware authenticates Organization credentials for streaming gRPC requests.
func NewOrganizationStreamMiddleware(principalResolver *resolver.OrganizationPrincipalResolver, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if !validator.NonNil(principalResolver) {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.ORG_SCOPE_KEY]
		if !presented {
			return handler(srv, stream)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimSpace(values[0])
		if !validator.NotBlank(token) {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.OrganizationCredential{Token: token})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return status.Error(codes.Canceled, ctxErr.Error())
				}
				return status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("organization authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = resolvedContext
		return handler(srv, wrapped)
	}
}

// NewServiceUnaryMiddleware authenticates Service assertions for unary gRPC requests.
func NewServiceUnaryMiddleware(principalResolver *resolver.ServicePrincipalResolver, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !validator.NonNil(principalResolver) {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.SERVICE_SCOPE_KEY]
		if !presented {
			return handler(ctx, req)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		assertion := strings.TrimSpace(values[0])
		if !validator.NotBlank(assertion) {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.ServiceAssertionCredential{Assertion: assertion})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return nil, status.Error(codes.Canceled, ctxErr.Error())
				}
				return nil, status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		return handler(resolvedContext, req)
	}
}

// NewServiceStreamMiddleware authenticates Service assertions for streaming gRPC requests.
func NewServiceStreamMiddleware(principalResolver *resolver.ServicePrincipalResolver, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if !validator.NonNil(principalResolver) {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		values, presented := metadata[types.SERVICE_SCOPE_KEY]
		if !presented {
			return handler(srv, stream)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || len(values) != 1 {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		assertion := strings.TrimSpace(values[0])
		if !validator.NotBlank(assertion) {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		value, err := principalResolver.Resolve(ctx, resolver.ServiceAssertionCredential{Assertion: assertion})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return status.Error(codes.Canceled, ctxErr.Error())
				}
				return status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("service authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = resolvedContext
		return handler(srv, wrapped)
	}
}

// NewUserUnaryMiddleware authenticates User credentials for unary gRPC requests.
func NewUserUnaryMiddleware(principalResolver *resolver.UserPrincipalResolver, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !validator.NonNil(principalResolver) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		tokenValues, tokenPresented := metadata[types.AUTHORIZATION_KEY]
		userIDValues, userIDPresented := metadata[types.AUTH_KEY]
		organizationIDValues, organizationIDPresented := metadata[types.ORGANIZATION_KEY]
		projectIDValues, projectIDPresented := metadata[types.PROJECT_KEY]
		if !tokenPresented && !userIDPresented && !organizationIDPresented && !projectIDPresented {
			return handler(ctx, req)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || !tokenPresented || !userIDPresented ||
			len(tokenValues) != 1 || len(userIDValues) != 1 ||
			(organizationIDPresented && len(organizationIDValues) != 1) || (projectIDPresented && len(projectIDValues) != 1) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimSpace(tokenValues[0])
		userIDValue := strings.TrimSpace(userIDValues[0])
		if !validator.NotBlank(token) || !validator.NotBlank(userIDValue) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		userID, err := strconv.ParseUint(userIDValue, 10, 64)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		organizationID := uint64(0)
		if organizationIDPresented {
			organizationIDValue := strings.TrimSpace(organizationIDValues[0])
			if !validator.NotBlank(organizationIDValue) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
			organizationID, err = strconv.ParseUint(organizationIDValue, 10, 64)
			if err != nil || !validator.Between(organizationID, uint64(1), uint64(math.MaxInt64)) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
		}
		projectID := uint64(0)
		if projectIDPresented {
			projectIDValue := strings.TrimSpace(projectIDValues[0])
			if !validator.NotBlank(projectIDValue) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
			projectID, err = strconv.ParseUint(projectIDValue, 10, 64)
			if err != nil || !validator.Between(projectID, uint64(1), uint64(math.MaxInt64)) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
		}
		value, err := principalResolver.Resolve(ctx, resolver.UserCredential{Token: token, UserID: userID, OrganizationID: organizationID, ProjectID: projectID})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return nil, status.Error(codes.Canceled, ctxErr.Error())
				}
				return nil, status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		return handler(resolvedContext, req)
	}
}

// NewUserStreamMiddleware authenticates User credentials for streaming gRPC requests.
func NewUserStreamMiddleware(principalResolver *resolver.UserPrincipalResolver, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if !validator.NonNil(principalResolver) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		metadata, _ := grpc_metadata.FromIncomingContext(ctx)
		tokenValues, tokenPresented := metadata[types.AUTHORIZATION_KEY]
		userIDValues, userIDPresented := metadata[types.AUTH_KEY]
		organizationIDValues, organizationIDPresented := metadata[types.ORGANIZATION_KEY]
		projectIDValues, projectIDPresented := metadata[types.PROJECT_KEY]
		if !tokenPresented && !userIDPresented && !organizationIDPresented && !projectIDPresented {
			return handler(srv, stream)
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		if _, exists := Principal(ctx); exists || !tokenPresented || !userIDPresented ||
			len(tokenValues) != 1 || len(userIDValues) != 1 ||
			(organizationIDPresented && len(organizationIDValues) != 1) || (projectIDPresented && len(projectIDValues) != 1) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		token := strings.TrimSpace(tokenValues[0])
		userIDValue := strings.TrimSpace(userIDValues[0])
		if !validator.NotBlank(token) || !validator.NotBlank(userIDValue) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		userID, err := strconv.ParseUint(userIDValue, 10, 64)
		if err != nil || !validator.Between(userID, uint64(1), uint64(math.MaxInt64)) {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		organizationID := uint64(0)
		if organizationIDPresented {
			organizationIDValue := strings.TrimSpace(organizationIDValues[0])
			if !validator.NotBlank(organizationIDValue) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
			organizationID, err = strconv.ParseUint(organizationIDValue, 10, 64)
			if err != nil || !validator.Between(organizationID, uint64(1), uint64(math.MaxInt64)) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
		}
		projectID := uint64(0)
		if projectIDPresented {
			projectIDValue := strings.TrimSpace(projectIDValues[0])
			if !validator.NotBlank(projectIDValue) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
			projectID, err = strconv.ParseUint(projectIDValue, 10, 64)
			if err != nil || !validator.Between(projectID, uint64(1), uint64(math.MaxInt64)) {
				logger.Errorf("user authentication rejected for route %s", info.FullMethod)
				return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
			}
		}
		value, err := principalResolver.Resolve(ctx, resolver.UserCredential{Token: token, UserID: userID, OrganizationID: organizationID, ProjectID: projectID})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				if ctxErr == context.Canceled {
					return status.Error(codes.Canceled, ctxErr.Error())
				}
				return status.Error(codes.DeadlineExceeded, ctxErr.Error())
			}
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		resolvedContext, err := ContextWithPrincipal(ctx, value)
		if err != nil {
			logger.Errorf("user authentication rejected for route %s", info.FullMethod)
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = resolvedContext
		return handler(srv, wrapped)
	}
}

// NewPolicyUnaryMiddleware authorizes one unary gRPC request against its route policy.
func NewPolicyUnaryMiddleware(policy ServicePolicy, logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !validator.NonNil(policy) {
			return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if err := ctx.Err(); err != nil {
			if err == context.Canceled {
				return nil, status.Error(codes.Canceled, err.Error())
			}
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		}
		routePolicy, exists := policy.GRPCPolicy(info.FullMethod)
		if !exists || routePolicy.Validate() != nil {
			logger.Errorf("principal authorization denied for route %s", info.FullMethod)
			return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if routePolicy.Authorization == authorizer.AuthorizationDisabled {
			return nil, status.Error(codes.Unavailable, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if routePolicy.Authorization == authorizer.AuthorizationPublic {
			return handler(contextWithoutPrincipal(ctx), req)
		}
		value, exists := Principal(ctx)
		if !exists {
			return nil, status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if (authorizer.RouteAuthorizer{}).Authorize(&value, routePolicy) != nil {
			logger.Errorf("principal authorization denied for route %s", info.FullMethod)
			return nil, status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		return handler(ctx, req)
	}
}

// NewPolicyStreamMiddleware authorizes one streaming gRPC request against its route policy.
func NewPolicyStreamMiddleware(policy ServicePolicy, logger commons.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !validator.NonNil(policy) {
			return status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if err := stream.Context().Err(); err != nil {
			if err == context.Canceled {
				return status.Error(codes.Canceled, err.Error())
			}
			return status.Error(codes.DeadlineExceeded, err.Error())
		}
		routePolicy, exists := policy.GRPCPolicy(info.FullMethod)
		if !exists || routePolicy.Validate() != nil {
			logger.Errorf("principal authorization denied for route %s", info.FullMethod)
			return status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if routePolicy.Authorization == authorizer.AuthorizationDisabled {
			return status.Error(codes.Unavailable, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		if routePolicy.Authorization == authorizer.AuthorizationPublic {
			wrapped := grpc_middleware.WrapServerStream(stream)
			wrapped.WrappedContext = contextWithoutPrincipal(stream.Context())
			return handler(srv, wrapped)
		}
		value, exists := Principal(stream.Context())
		if !exists {
			return status.Error(codes.Unauthenticated, (pkg_errors.AuthenticationFailureError{}).Error())
		}
		if (authorizer.RouteAuthorizer{}).Authorize(&value, routePolicy) != nil {
			logger.Errorf("principal authorization denied for route %s", info.FullMethod)
			return status.Error(codes.PermissionDenied, (pkg_errors.AuthorizationFailureError{}).Error())
		}
		return handler(srv, stream)
	}
}
