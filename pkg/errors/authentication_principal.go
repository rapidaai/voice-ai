// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package errors

// AuthenticationInvalidError reports an invalid authentication aggregate without exposing its input.
type AuthenticationInvalidError struct{}

func (AuthenticationInvalidError) Error() string {
	return "authentication is invalid"
}

// AuthenticationPrincipalInvalidError reports an invalid principal without exposing its input.
type AuthenticationPrincipalInvalidError struct{}

func (AuthenticationPrincipalInvalidError) Error() string {
	return "authentication principal is invalid"
}

// AuthenticationPrincipalConflictError reports an occupied request principal context.
type AuthenticationPrincipalConflictError struct{}

func (AuthenticationPrincipalConflictError) Error() string {
	return "authentication principal conflicts with request context"
}

// AuthenticationResolverInvalidError reports an incomplete resolver configuration.
type AuthenticationResolverInvalidError struct{}

func (AuthenticationResolverInvalidError) Error() string {
	return "authentication resolver is invalid"
}

// AuthenticationPolicyInvalidError reports an invalid policy without exposing configured values.
type AuthenticationPolicyInvalidError struct{}

func (AuthenticationPolicyInvalidError) Error() string {
	return "authentication policy is invalid"
}

// AuthenticationCredentialInvalidError reports a rejected credential without exposing its value.
type AuthenticationCredentialInvalidError struct{}

func (AuthenticationCredentialInvalidError) Error() string {
	return "authentication credential is invalid"
}

// AuthenticationSecretInvalidError reports invalid root-secret configuration.
type AuthenticationSecretInvalidError struct{}

func (AuthenticationSecretInvalidError) Error() string {
	return "authentication secret is invalid"
}

// JWTIssuerInvalidError reports unsafe authentication JWT issuer configuration.
type JWTIssuerInvalidError struct{}

func (JWTIssuerInvalidError) Error() string {
	return "authentication JWT issuer is invalid"
}

// JWTVerifierInvalidError reports unsafe authentication JWT verifier configuration.
type JWTVerifierInvalidError struct{}

func (JWTVerifierInvalidError) Error() string {
	return "authentication JWT verifier is invalid"
}

// JWTIssueError reports a failure to create an authentication JWT.
type JWTIssueError struct{}

func (JWTIssueError) Error() string {
	return "authentication JWT could not be issued"
}

// ApplicationConfigurationInvalidError reports invalid configuration without exposing configured values.
type ApplicationConfigurationInvalidError struct{}

func (ApplicationConfigurationInvalidError) Error() string {
	return "application configuration is invalid"
}
