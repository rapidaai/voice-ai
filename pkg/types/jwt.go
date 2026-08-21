// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CreateJWT creates a JWT token with the provided claims and returns the token string
func CreateServiceScopeToken(delegatedContext DelegatedContext, secretKey string) (string, error) {
	normalizedContext, ok := normalizeDelegatedContext(delegatedContext, true)
	if !ok {
		return "", fmt.Errorf("delegated context must contain a valid organization and non-zero optional identities")
	}

	claims := jwt.MapClaims{
		"exp":            time.Now().Add(time.Hour * 24).Unix(),
		"organizationId": normalizedContext.OrganizationID,
	}

	if normalizedContext.UserID != nil {
		claims["userId"] = *normalizedContext.UserID
	}
	if normalizedContext.ProjectID != nil {
		claims["projectId"] = *normalizedContext.ProjectID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("error creating token: %v", err)
	}
	return tokenString, nil
}

// ExtractJWT extracts the claims from the provided JWT token string and returns the decoded PlainAuthPrinciple
func ExtractServiceScope(tokenString string, secretKey string) (*ServiceScope, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	organizationID, ok := requiredUint64Claim(claims, "organizationId")
	if !ok {
		return nil, fmt.Errorf("service scope token requires a valid organizationId claim")
	}
	userID, ok := optionalUint64Claim(claims, "userId")
	if !ok {
		return nil, fmt.Errorf("service scope token contains an invalid userId claim")
	}
	projectID, ok := optionalUint64Claim(claims, "projectId")
	if !ok {
		return nil, fmt.Errorf("service scope token contains an invalid projectId claim")
	}
	normalizedContext, ok := normalizeDelegatedContext(DelegatedContext{
		UserID:         userID,
		OrganizationID: organizationID,
		ProjectID:      projectID,
	}, true)
	if !ok {
		return nil, fmt.Errorf("service scope token contains malformed delegated context")
	}
	return &ServiceScope{
		UserId:         normalizedContext.UserID,
		OrganizationId: &normalizedContext.OrganizationID,
		ProjectId:      normalizedContext.ProjectID,
		CurrentToken:   tokenString,
	}, nil
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
	switch v := value.(type) {
	case float64:
		if v <= 0 || math.Trunc(v) != v || v >= math.Exp2(64) {
			return 0, false
		}
		return uint64(v), true
	case int:
		if v <= 0 {
			return 0, false
		}
		return uint64(v), true
	case int64:
		if v <= 0 {
			return 0, false
		}
		return uint64(v), true
	case uint64:
		return v, v != 0
	case uint:
		return uint64(v), v != 0
	case string:
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil && parsed != 0 {
			return parsed, true
		}
	}
	return 0, false
}
