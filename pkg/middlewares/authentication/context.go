// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	"context"

	principal "github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

type principalContextKey struct{}

// Principal returns the authenticated principal attached to the request.
func Principal(ctx context.Context) (principal.Principal, bool) {
	value, exists := ctx.Value(principalContextKey{}).(principal.Principal)
	return value, exists && validator.NonZero(value.Subject().ID)
}

// ContextWithPrincipal attaches an authenticated principal once per request.
func ContextWithPrincipal(ctx context.Context, value principal.Principal) (context.Context, error) {
	if !validator.NonZero(value.Subject().ID) {
		return ctx, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	if _, exists := ctx.Value(principalContextKey{}).(principal.Principal); exists {
		return ctx, pkg_errors.AuthenticationPrincipalConflictError{}
	}
	return context.WithValue(ctx, principalContextKey{}, value), nil
}

// contextWithoutPrincipal hides authentication facts from public-route handlers.
func contextWithoutPrincipal(ctx context.Context) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal.Principal{})
}
