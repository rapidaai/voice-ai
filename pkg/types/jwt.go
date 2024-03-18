package types

import (
	"fmt"
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

	ol := &ServiceScope{}

	fmt.Printf("%+v", claims)
	if _, exists := claims["userId"]; exists {
		user := uint64(claims["userId"].(float64))
		ol.UserId = &user
	}

	if _, exists := claims["organizationId"]; exists {
		organizationId := uint64(claims["organizationId"].(float64))
		ol.OrganizationId = &organizationId
	}

	if _, exists := claims["projectId"]; exists {
		projectId := uint64(claims["projectId"].(float64))
		ol.ProjectId = &projectId
	}

	return ol, nil
}
