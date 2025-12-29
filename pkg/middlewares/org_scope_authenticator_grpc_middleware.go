// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package middlewares

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"google.golang.org/grpc"
)

func NewOrganizationAuthenticatorUnaryServerMiddleware(resolver types.ClaimAuthenticator[*types.OrganizationScope], logger commons.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		apiKey := metadata.ExtractIncoming(ctx).Get(types.ORG_SCOPE_KEY)
		if apiKey == "" {
			return handler(ctx, req)
		}
		auth, err := resolver.Claim(ctx, apiKey)
		if err != nil {
			logger.Errorf("unable to resolve given api key for project")
			return handler(ctx, req)
		}
		return handler(context.WithValue(ctx, types.CTX_, auth), req)
	}
}
