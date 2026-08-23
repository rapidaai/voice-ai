// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ServiceAssertionAudience = "rapida-internal"

type ServiceAssertion struct {
	ActorID uint64
	Issuer  string
	TTL     time.Duration
}

func CreateServiceScopeToken(delegatedContext DelegatedContext, assertion ServiceAssertion, secret string) (string, error) {
	if delegatedContext.UserID != nil {
		return "", fmt.Errorf("service assertions must not forward an originating user identity")
	}
	normalizedContext, ok := normalizeDelegatedContext(delegatedContext, true)
	if !ok {
		return "", fmt.Errorf("delegated context must contain a valid organization and non-zero optional identities")
	}
	actor := ActorIdentity{Type: ActorTypeService, ID: assertion.ActorID}
	if err := actor.Validate(); err != nil {
		return "", fmt.Errorf("service assertion actor is invalid: %w", err)
	}
	issuer := strings.TrimSpace(assertion.Issuer)
	if issuer == "" {
		return "", fmt.Errorf("service assertion issuer is required")
	}
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("service assertion secret is required")
	}
	if assertion.TTL <= 0 || assertion.TTL > 5*time.Minute {
		return "", fmt.Errorf("service assertion ttl must be between zero and five minutes")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"actor_type":     string(ActorTypeService),
		"actor_id":       assertion.ActorID,
		"iss":            issuer,
		"aud":            ServiceAssertionAudience,
		"iat":            now.Unix(),
		"exp":            now.Add(assertion.TTL).Unix(),
		"organizationId": normalizedContext.OrganizationID,
	}
	if normalizedContext.ProjectID != nil {
		claims["projectId"] = *normalizedContext.ProjectID
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("error creating token: %v", err)
	}
	return tokenString, nil
}

func ExtractServiceScope(tokenString string, secret string) (*ServiceScope, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("service assertion secret is required")
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithAudience(ServiceAssertionAudience),
		jwt.WithJSONNumber(),
	)
	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return serviceScopeFromToken(token, tokenString)
}

func serviceScopeFromToken(token *jwt.Token, tokenString string) (*ServiceScope, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}
	actorType, ok := requiredStringClaim(claims, "actor_type")
	if !ok || actorType != string(ActorTypeService) {
		return nil, fmt.Errorf("service scope token requires actor_type=service")
	}
	actorID, ok := requiredUint64Claim(claims, "actor_id")
	if !ok || actorID > math.MaxInt64 {
		return nil, fmt.Errorf("service scope token requires a valid actor_id claim")
	}
	issuer, ok := requiredStringClaim(claims, "iss")
	if !ok {
		return nil, fmt.Errorf("service scope token requires an issuer")
	}
	audience, ok := requiredStringClaim(claims, "aud")
	if !ok || audience != ServiceAssertionAudience {
		return nil, fmt.Errorf("service scope token has an invalid audience")
	}
	if _, exists := claims["userId"]; exists {
		return nil, fmt.Errorf("service scope token must not contain a userId claim")
	}
	issuedAt, err := claims.GetIssuedAt()
	if err != nil || issuedAt == nil {
		return nil, fmt.Errorf("service scope token requires a valid issued-at claim")
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil || !expiresAt.After(issuedAt.Time) || expiresAt.Sub(issuedAt.Time) > 5*time.Minute {
		return nil, fmt.Errorf("service scope token lifetime must not exceed five minutes")
	}

	organizationID, ok := requiredUint64Claim(claims, "organizationId")
	if !ok {
		return nil, fmt.Errorf("service scope token requires a valid organizationId claim")
	}
	projectID, ok := optionalUint64Claim(claims, "projectId")
	if !ok {
		return nil, fmt.Errorf("service scope token contains an invalid projectId claim")
	}
	normalizedContext, ok := normalizeDelegatedContext(DelegatedContext{
		OrganizationID: organizationID,
		ProjectID:      projectID,
	}, true)
	if !ok {
		return nil, fmt.Errorf("service scope token contains malformed delegated context")
	}
	return &ServiceScope{
		ActorId:        actorID,
		Issuer:         issuer,
		Audience:       audience,
		OrganizationId: &normalizedContext.OrganizationID,
		ProjectId:      normalizedContext.ProjectID,
		CurrentToken:   tokenString,
	}, nil
}

func requiredStringClaim(claims jwt.MapClaims, name string) (string, bool) {
	value, exists := claims[name]
	if !exists {
		return "", false
	}
	result, ok := value.(string)
	result = strings.TrimSpace(result)
	return result, ok && result != ""
}

func requiredUint64Claim(claims jwt.MapClaims, name string) (uint64, bool) {
	value, exists := claims[name]
	if !exists {
		return 0, false
	}
	result, ok := toUint64(value)
	return result, ok && result != 0
}

func optionalUint64Claim(claims jwt.MapClaims, name string) (*uint64, bool) {
	value, exists := claims[name]
	if !exists {
		return nil, true
	}
	result, ok := toUint64(value)
	if !ok || result == 0 {
		return nil, false
	}
	return &result, true
}

func toUint64(value interface{}) (uint64, bool) {
	switch value := value.(type) {
	case float64:
		if value <= 0 || math.Trunc(value) != value || value >= math.Exp2(64) {
			return 0, false
		}
		return uint64(value), true
	case int:
		if value <= 0 {
			return 0, false
		}
		return uint64(value), true
	case int64:
		if value <= 0 {
			return 0, false
		}
		return uint64(value), true
	case uint64:
		return value, value != 0
	case uint:
		return uint64(value), value != 0
	case string:
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil && parsed != 0 {
			return parsed, true
		}
	case json.Number:
		if parsed, err := strconv.ParseUint(value.String(), 10, 64); err == nil && parsed != 0 {
			return parsed, true
		}
	}
	return 0, false
}
