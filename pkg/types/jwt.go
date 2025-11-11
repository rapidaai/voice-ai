package types

/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CreateJWT creates a JWT token with the provided claims and returns the token string
func CreateServiceScopeToken(principle SimplePrinciple, secretKey string) (string, error) {

	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(), // token expires in 24 hours
	}

	if principle.GetUserId() != nil {
		claims["userId"] = *principle.GetUserId()
	}
	if principle.GetCurrentOrganizationId() != nil {
		claims["organizationId"] = *principle.GetCurrentOrganizationId()
	}
	if principle.GetCurrentProjectId() != nil {
		claims["projectId"] = *principle.GetCurrentProjectId()
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

	ol := &ServiceScope{
		CurrentToken: tokenString,
	}

	if _, exists := claims["userId"]; exists {
		user, ok := toUint64(claims["userId"])
		if ok {
			ol.UserId = &user
		}
	}

	if _, exists := claims["organizationId"]; exists {
		organizationId, ok := toUint64(claims["organizationId"])
		if ok {
			ol.OrganizationId = &organizationId
		}

	}

	if _, exists := claims["projectId"]; exists {
		projectId, ok := toUint64(claims["projectId"])
		if ok {
			ol.ProjectId = &projectId
		}
	}
	return ol, nil
}

func toUint64(value interface{}) (uint64, bool) {
	switch v := value.(type) {
	case float64:
		return uint64(v), true
	case int:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case string:
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			return parsed, true
		}
		// Add more cases as needed
	}
	return 0, false
}
