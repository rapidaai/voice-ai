// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

// RouteAuthorizationDeniedError reports a generic route-policy denial.
type RouteAuthorizationDeniedError struct{}

func (RouteAuthorizationDeniedError) Error() string {
	return "route authorization denied"
}

// RouteAuthorizationDisabledError reports a route intentionally disabled by policy.
type RouteAuthorizationDisabledError struct{}

func (RouteAuthorizationDisabledError) Error() string {
	return "route is disabled"
}
