// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

// AuthenticationFailureError is the safe client-facing authentication failure.
type AuthenticationFailureError struct{}

func (AuthenticationFailureError) Error() string {
	return "Invalid authentication credentials"
}

// AuthorizationFailureError is the safe client-facing authorization failure.
type AuthorizationFailureError struct{}

func (AuthorizationFailureError) Error() string {
	return "Access denied"
}

// ErrorResponse is the shared HTTP error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}
