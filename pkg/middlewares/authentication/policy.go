// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import "github.com/rapidaai/pkg/authentication/authorizer"

// ServicePolicy resolves the authorization policy assigned to each service route.
type ServicePolicy interface {
	HTTPPolicy(route string) (authorizer.RoutePolicy, bool)
	GRPCPolicy(route string) (authorizer.RoutePolicy, bool)
}
