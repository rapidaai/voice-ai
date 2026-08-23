package types

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testServiceSecret = "shared-service-secret"

func TestCreateAndExtractServiceScopeToken(t *testing.T) {
	projectID := uint64(3)
	token, err := CreateServiceScopeToken(DelegatedContext{OrganizationID: 2, ProjectID: &projectID}, testServiceAssertion(), testServiceSecret)
	if err != nil {
		t.Fatalf("CreateServiceScopeToken() error = %v", err)
	}
	scope, err := ExtractServiceScope(token, testServiceSecret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	if scope.ActorId != 41 || scope.Issuer != "assistant-api" || scope.Audience != ServiceAssertionAudience {
		t.Fatalf("ExtractServiceScope() = %+v", scope)
	}
	delegatedContext, ok := scope.DelegatedContext()
	if !ok || delegatedContext.OrganizationID != 2 || delegatedContext.UserID != nil || delegatedContext.ProjectID == nil || *delegatedContext.ProjectID != projectID {
		t.Fatalf("DelegatedContext() = %+v, %v", delegatedContext, ok)
	}
}

func TestCreateAndExtractServiceScopeTokenWithDelegatedActors(t *testing.T) {
	projectID := uint64(3)
	tests := []struct {
		name    string
		actor   ActorIdentity
		context DelegatedContext
	}{
		{name: "user", actor: ActorIdentity{Type: ActorTypeUser, ID: 5}, context: DelegatedContext{OrganizationID: 2, ProjectID: &projectID}},
		{name: "project", actor: ActorIdentity{Type: ActorTypeProject, ID: 6}, context: DelegatedContext{OrganizationID: 2, ProjectID: &projectID}},
		{name: "organization", actor: ActorIdentity{Type: ActorTypeOrganization, ID: 7}, context: DelegatedContext{OrganizationID: 2}},
		{name: "service", actor: ActorIdentity{Type: ActorTypeService, ID: 8}, context: DelegatedContext{OrganizationID: 2}},
		{name: "system", actor: ActorIdentity{Type: ActorTypeSystem, ID: 9}, context: DelegatedContext{OrganizationID: 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertion := testServiceAssertion()
			delegatedActor := test.actor
			assertion.DelegatedActor = &delegatedActor
			token, err := CreateServiceScopeToken(test.context, assertion, testServiceSecret)
			if err != nil {
				t.Fatalf("CreateServiceScopeToken() error = %v", err)
			}
			scope, err := ExtractServiceScope(token, testServiceSecret)
			if err != nil {
				t.Fatalf("ExtractServiceScope() error = %v", err)
			}
			auth, err := scope.Authentication()
			if err != nil {
				t.Fatalf("Authentication() error = %v", err)
			}
			actor, err := auth.Actor()
			if err != nil || actor != test.actor {
				t.Fatalf("Actor() = %+v, %v; want %+v", actor, err, test.actor)
			}
			caller, err := auth.Caller()
			if err != nil || caller != (ActorIdentity{Type: ActorTypeService, ID: 41}) {
				t.Fatalf("Caller() = %+v, %v", caller, err)
			}
		})
	}
}

func TestCreateAndExtractServiceScopeTokenPreservesBigintActor(t *testing.T) {
	const actorID = uint64(9007199254740993)
	token, err := CreateServiceScopeToken(DelegatedContext{OrganizationID: actorID}, ServiceAssertion{
		ActorID: actorID,
		Issuer:  "assistant-api",
		TTL:     time.Minute,
	}, testServiceSecret)
	if err != nil {
		t.Fatalf("CreateServiceScopeToken() error = %v", err)
	}
	scope, err := ExtractServiceScope(token, testServiceSecret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	if scope.ActorId != actorID || scope.OrganizationId == nil || *scope.OrganizationId != actorID {
		t.Fatalf("ExtractServiceScope() lost bigint precision: %+v", scope)
	}
}

func TestCreateServiceScopeTokenRejectsLegacyUserClaim(t *testing.T) {
	userID := uint64(5)
	if _, err := CreateServiceScopeToken(DelegatedContext{OrganizationID: 2, UserID: &userID}, testServiceAssertion(), testServiceSecret); err == nil {
		t.Fatal("CreateServiceScopeToken() error = nil")
	}
}

func TestCreateServiceScopeTokenRejectsInvalidInputs(t *testing.T) {
	zero := uint64(0)
	projectActor := ActorIdentity{Type: ActorTypeProject, ID: 5}
	organizationActor := ActorIdentity{Type: ActorTypeOrganization, ID: 5}
	projectID := uint64(3)
	tests := []struct {
		name      string
		context   DelegatedContext
		assertion ServiceAssertion
		secret    string
		want      error
	}{
		{name: "missing organization", context: DelegatedContext{}, assertion: testServiceAssertion(), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "zero project", context: DelegatedContext{OrganizationID: 2, ProjectID: &zero}, assertion: testServiceAssertion(), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "zero actor", context: DelegatedContext{OrganizationID: 2}, assertion: ServiceAssertion{Issuer: "assistant-api", TTL: time.Minute}, secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "missing issuer", context: DelegatedContext{OrganizationID: 2}, assertion: ServiceAssertion{ActorID: 41, TTL: time.Minute}, secret: testServiceSecret, want: ErrServiceNameUnavailable},
		{name: "missing secret", context: DelegatedContext{OrganizationID: 2}, assertion: testServiceAssertion(), want: ErrServiceSecretUnavailable},
		{name: "long ttl", context: DelegatedContext{OrganizationID: 2}, assertion: ServiceAssertion{ActorID: 41, Issuer: "assistant-api", TTL: 6 * time.Minute}, secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "project without project context", context: DelegatedContext{OrganizationID: 2}, assertion: ServiceAssertion{ActorID: 41, Issuer: "assistant-api", TTL: time.Minute, DelegatedActor: &projectActor}, secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "organization with project context", context: DelegatedContext{OrganizationID: 2, ProjectID: &projectID}, assertion: ServiceAssertion{ActorID: 41, Issuer: "assistant-api", TTL: time.Minute, DelegatedActor: &organizationActor}, secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CreateServiceScopeToken(test.context, test.assertion, test.secret); !errors.Is(err, test.want) {
				t.Fatalf("CreateServiceScopeToken() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestExtractServiceScopeRejectsInvalidTokens(t *testing.T) {
	now := time.Now()
	validClaims := jwt.MapClaims{
		"actor_type":     "service",
		"actor_id":       41,
		"iss":            "assistant-api",
		"aud":            ServiceAssertionAudience,
		"iat":            now.Unix(),
		"exp":            now.Add(time.Minute).Unix(),
		"organizationId": 2,
	}
	validToken := signServiceTestToken(t, jwt.SigningMethodHS256, validClaims, testServiceSecret)
	longLivedClaims := cloneClaims(validClaims)
	longLivedClaims["exp"] = now.Add(6 * time.Minute).Unix()
	expiredClaims := cloneClaims(validClaims)
	expiredClaims["iat"] = now.Add(-2 * time.Minute).Unix()
	expiredClaims["exp"] = now.Add(-time.Minute).Unix()
	invalidActorClaims := cloneClaims(validClaims)
	invalidActorClaims["actor_id"] = 0
	forwardedUserClaims := cloneClaims(validClaims)
	forwardedUserClaims["userId"] = 9
	partialDelegatedTypeClaims := cloneClaims(validClaims)
	partialDelegatedTypeClaims["delegated_auth_type"] = "user"
	partialDelegatedIDClaims := cloneClaims(validClaims)
	partialDelegatedIDClaims["delegated_actor_id"] = 9
	unsupportedDelegatedClaims := cloneClaims(validClaims)
	unsupportedDelegatedClaims["delegated_auth_type"] = "unsupported"
	unsupportedDelegatedClaims["delegated_actor_id"] = 9
	projectWithoutContextClaims := cloneClaims(validClaims)
	projectWithoutContextClaims["delegated_auth_type"] = "project"
	projectWithoutContextClaims["delegated_actor_id"] = 9
	organizationWithProjectClaims := cloneClaims(validClaims)
	organizationWithProjectClaims["delegated_auth_type"] = "organization"
	organizationWithProjectClaims["delegated_actor_id"] = 9
	organizationWithProjectClaims["projectId"] = 3

	tests := []struct {
		name   string
		token  string
		secret string
		want   error
	}{
		{name: "invalid", token: "invalid.token.here", secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "wrong secret", token: validToken, secret: "wrong-secret", want: ErrInvalidServiceAssertion},
		{name: "wrong algorithm", token: signServiceTestToken(t, jwt.SigningMethodHS384, validClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "expired", token: signServiceTestToken(t, jwt.SigningMethodHS256, expiredClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "long lived", token: signServiceTestToken(t, jwt.SigningMethodHS256, longLivedClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "invalid actor", token: signServiceTestToken(t, jwt.SigningMethodHS256, invalidActorClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidServiceAssertion},
		{name: "forwarded user", token: signServiceTestToken(t, jwt.SigningMethodHS256, forwardedUserClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "delegated type only", token: signServiceTestToken(t, jwt.SigningMethodHS256, partialDelegatedTypeClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "delegated id only", token: signServiceTestToken(t, jwt.SigningMethodHS256, partialDelegatedIDClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "unsupported delegated actor", token: signServiceTestToken(t, jwt.SigningMethodHS256, unsupportedDelegatedClaims, testServiceSecret), secret: testServiceSecret, want: ErrUnsupportedDelegatedAuthentication},
		{name: "project without context", token: signServiceTestToken(t, jwt.SigningMethodHS256, projectWithoutContextClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "organization with project", token: signServiceTestToken(t, jwt.SigningMethodHS256, organizationWithProjectClaims, testServiceSecret), secret: testServiceSecret, want: ErrInvalidDelegatedIdentity},
		{name: "empty secret", token: validToken, want: ErrServiceSecretUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ExtractServiceScope(test.token, test.secret); !errors.Is(err, test.want) {
				t.Fatalf("ExtractServiceScope() error = %v, want %v", err, test.want)
			}
		})
	}
}

func signServiceTestToken(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return token
}

func cloneClaims(claims jwt.MapClaims) jwt.MapClaims {
	result := make(jwt.MapClaims, len(claims))
	for key, value := range claims {
		result[key] = value
	}
	return result
}

func testServiceAssertion() ServiceAssertion {
	return ServiceAssertion{ActorID: 41, Issuer: "assistant-api", TTL: time.Minute}
}

func TestToUint64(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  uint64
		ok    bool
	}{
		{name: "float64", value: float64(123), want: 123, ok: true},
		{name: "int", value: 456, want: 456, ok: true},
		{name: "int64", value: int64(789), want: 789, ok: true},
		{name: "uint64", value: uint64(10), want: 10, ok: true},
		{name: "string valid", value: "101112", want: 101112, ok: true},
		{name: "zero", value: 0, ok: false},
		{name: "negative", value: -1, ok: false},
		{name: "fractional", value: 1.5, ok: false},
		{name: "string invalid", value: "not-a-number", ok: false},
		{name: "unsupported type", value: true, ok: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := toUint64(test.value)
			if ok != test.ok || ok && got != test.want {
				t.Fatalf("toUint64() = %d, %v; want %d, %v", got, ok, test.want, test.ok)
			}
		})
	}
}
