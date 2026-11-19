// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package jwt

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"

	"github.com/rapidaai/pkg/authentication"
)

func assertionTestKey(value byte) []byte {
	return bytes.Repeat([]byte{value}, sha256.Size)
}

func TestAssertionRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	key := assertionTestKey(1)
	issuer, err := NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	issuer.now = func() time.Time { return now }
	verifier := NewVerifier("web-api", key)
	verifier.now = func() time.Time { return now }
	tests := []struct {
		name            string
		subject         authentication.Identity
		tenant          authentication.Tenant
		authorities     []authentication.PermissionKey
		wantSubject     authentication.Identity
		wantAuthorities []authentication.PermissionKey
		wantActor       bool
	}{
		{
			name: "tenantless service", subject: authentication.Identity{Type: authentication.IdentityService, ID: 9007, Issuer: "workflow-api"},
			wantSubject: authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"},
		},
		{
			name: "tenant service", subject: authentication.Identity{Type: authentication.IdentityService, ID: 9007, Issuer: "workflow-api"},
			tenant:      authentication.Tenant{OrganizationID: 8, ProjectID: 9},
			wantSubject: authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"},
		},
		{
			name: "user", subject: authentication.Identity{Type: authentication.IdentityUser, ID: 11},
			tenant:          authentication.Tenant{OrganizationID: 8, ProjectID: 9},
			authorities:     []authentication.PermissionKey{"project.assistant.read", "organization.user.read"},
			wantSubject:     authentication.Identity{Type: authentication.IdentityUser, ID: 11},
			wantAuthorities: []authentication.PermissionKey{"organization.user.read", "project.assistant.read"}, wantActor: true,
		},
		{
			name: "project", subject: authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
			tenant:          authentication.Tenant{OrganizationID: 8, ProjectID: 9},
			authorities:     []authentication.PermissionKey{"project.assistant.read"},
			wantSubject:     authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 12},
			wantAuthorities: []authentication.PermissionKey{"project.assistant.read"}, wantActor: true,
		},
		{
			name: "organization", subject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13},
			tenant:      authentication.Tenant{OrganizationID: 8},
			wantSubject: authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 13}, wantActor: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertion, err := issuer.Issue(test.subject, test.tenant, test.authorities)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := verifier.Verify(assertion)
			if err != nil {
				t.Fatal(err)
			}
			if verified.Subject() != test.wantSubject || verified.Tenant() != test.tenant || verified.Destination() != "web-api" ||
				!verified.ExpiresAt().Equal(now.Add(55*time.Second)) {
				t.Fatalf("verified assertion = %+v", verified)
			}
			fingerprint := sha256.Sum256(key)
			if verified.KeyFingerprint() != base64.RawURLEncoding.EncodeToString(fingerprint[:]) {
				t.Fatalf("KeyFingerprint() = %q", verified.KeyFingerprint())
			}
			if verified.Actor() != (authentication.ServiceActor{ID: 1, Issuer: "internal"}) {
				t.Fatalf("Actor() = %+v", verified.Actor())
			}
			if !reflect.DeepEqual(verified.Authorities(), test.wantAuthorities) {
				t.Fatalf("Authorities() = %v, want %v", verified.Authorities(), test.wantAuthorities)
			}
			if _, err := verifier.Verify(assertion); err != nil {
				t.Fatalf("bounded replay rejected: %v", err)
			}
		})
	}
}

func TestAssertionContainsOnlyAcceptedHeaderAndClaims(t *testing.T) {
	key := assertionTestKey(2)
	issuer, err := NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	assertion, err := issuer.IssueService(authentication.Tenant{})
	if err != nil {
		t.Fatal(err)
	}
	parts := bytes.Split([]byte(assertion), []byte("."))
	if len(parts) != 3 {
		t.Fatalf("JWT parts = %d", len(parts))
	}
	decode := func(source []byte) map[string]json.RawMessage {
		decoded, err := base64.RawURLEncoding.DecodeString(string(source))
		if err != nil {
			t.Fatal(err)
		}
		value := map[string]json.RawMessage{}
		if err := json.Unmarshal(decoded, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if header := decode(parts[0]); len(header) != 2 || header["alg"] == nil || header["typ"] == nil {
		t.Fatalf("header = %v", header)
	}
	if claims := decode(parts[1]); len(claims) != 3 || claims["aud"] == nil || claims["exp"] == nil || claims["authentication"] == nil {
		t.Fatalf("claims = %v", claims)
	} else {
		authenticationClaims := map[string]json.RawMessage{}
		if err := json.Unmarshal(claims["authentication"], &authenticationClaims); err != nil {
			t.Fatal(err)
		}
		if _, exists := authenticationClaims["subject"]; exists {
			t.Fatalf("direct authentication claims contain subject: %v", authenticationClaims)
		}
	}
}

func TestVerifierAcceptsPythonGoldenAssertion(t *testing.T) {
	key, err := base64.StdEncoding.DecodeString("veUxBjUOPPj8M4ZdOuRNqMuIzamLmE4S2upg4QeQL40=")
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewVerifier("document-api", key)
	verifier.now = func() time.Time { return time.Unix(1794528000, 0).UTC() }
	verified, err := verifier.Verify("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJkb2N1bWVudC1hcGkiLCJleHAiOjE3OTQ1MjgwNTUsImF1dGhlbnRpY2F0aW9uIjp7InRlbmFudCI6eyJvcmdhbml6YXRpb25faWQiOjcsInByb2plY3RfaWQiOjExfSwicGVybWlzc2lvbnMiOlsib3JnYW5pemF0aW9uLm1lbWJlcnMucmVhZCIsInByb2plY3Qua25vd2xlZGdlLnJlYWQiXSwic3ViamVjdCI6eyJ0eXBlIjoidXNlciIsImlkIjo0Mn19fQ.WoHmCKOSMEraI__2IDKR2kzvc_MB3Wi0zv9RP15BppM")
	if err != nil {
		t.Fatal(err)
	}
	if verified.Subject() != (authentication.Identity{Type: authentication.IdentityUser, ID: 42}) ||
		verified.Actor() != (authentication.ServiceActor{ID: 1, Issuer: "internal"}) ||
		verified.KeyFingerprint() != "kT0c-JnsGE4x4pLFAL7pVVX4h2L42piDVcwmK2zPWbg" {
		t.Fatalf("verified assertion = %+v", verified)
	}
}

func TestVerifierRejectsSharedMalformedAssertions(t *testing.T) {
	key, err := base64.StdEncoding.DecodeString("veUxBjUOPPj8M4ZdOuRNqMuIzamLmE4S2upg4QeQL40=")
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewVerifier("document-api", key)
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return time.Unix(1794528000, 0).UTC() }
	source, err := os.ReadFile("../../../api/document-api/tests/fixtures/service_assertion_malformed.json")
	if err != nil {
		t.Fatal(err)
	}
	assertions := map[string]string{}
	if err := json.Unmarshal(source, &assertions); err != nil {
		t.Fatal(err)
	}
	for name, assertion := range assertions {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Verify(assertion); err == nil {
				t.Fatal("malformed assertion accepted")
			}
		})
	}
}

func TestAssertionRejectsInvalidTokens(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	key := assertionTestKey(3)
	verifier := NewVerifier("web-api", key)
	verifier.now = func() time.Time { return now }
	sign := func(t *testing.T, claims jwtv5.Claims, header func(map[string]interface{})) string {
		t.Helper()
		token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
		if header != nil {
			header(token.Header)
		}
		assertion, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return assertion
	}
	validAt := func(offset time.Duration) assertionClaims {
		permissions := []authentication.PermissionKey{"organization.user.read"}
		expiresAt := now.Add(offset).Unix()
		return assertionClaims{
			Audience:  "web-api",
			ExpiresAt: &expiresAt,
			Authentication: &assertionAuthentication{
				Subject: &assertionIdentity{Type: authentication.IdentityUser, ID: 11},
				Tenant:  &assertionTenant{OrganizationID: 8}, Permissions: &permissions,
			},
		}
	}
	tests := []struct {
		name      string
		assertion func(*testing.T) string
	}{
		{name: "expired", assertion: func(t *testing.T) string { return sign(t, validAt(0), nil) }},
		{name: "lifetime too long", assertion: func(t *testing.T) string { return sign(t, validAt(61*time.Second), nil) }},
		{name: "wrong audience", assertion: func(t *testing.T) string {
			claims := validAt(time.Minute)
			claims.Audience = "endpoint-api"
			return sign(t, claims, nil)
		}},
		{name: "missing authentication", assertion: func(t *testing.T) string {
			claims := validAt(time.Minute)
			claims.Authentication = nil
			return sign(t, claims, nil)
		}},
		{name: "service subject in payload", assertion: func(t *testing.T) string {
			claims := validAt(time.Minute)
			claims.Authentication.Subject.Type = authentication.IdentityService
			return sign(t, claims, nil)
		}},
		{name: "unsorted permissions", assertion: func(t *testing.T) string {
			claims := validAt(time.Minute)
			values := []authentication.PermissionKey{"project.read", "organization.read"}
			auth := *claims.Authentication
			auth.Permissions = &values
			claims.Authentication = &auth
			return sign(t, claims, nil)
		}},
		{name: "additional header", assertion: func(t *testing.T) string {
			return sign(t, validAt(time.Minute), func(header map[string]interface{}) { header["kid"] = "removed" })
		}},
		{name: "old format", assertion: func(t *testing.T) string {
			return sign(t, jwtv5.MapClaims{"iss": "workflow-api", "aud": "web-api", "exp": now.Add(time.Minute).Unix()}, nil)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := verifier.Verify(test.assertion(t)); err == nil {
				t.Fatal("invalid assertion accepted")
			}
		})
	}
	wrongToken := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, validAt(time.Minute))
	wrongAssertion, err := wrongToken.SignedString(assertionTestKey(4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(wrongAssertion); err == nil {
		t.Fatal("wrong signing key accepted")
	}
	edDSAToken := jwtv5.NewWithClaims(jwtv5.SigningMethodEdDSA, validAt(time.Minute))
	edDSAAssertion, err := edDSAToken.SignedString(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(edDSAAssertion); err == nil {
		t.Fatal("EdDSA assertion accepted")
	}
}

func TestAssertionClockSkewBudget(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	key := assertionTestKey(5)
	verifier := NewVerifier("web-api", key)
	verifier.now = func() time.Time { return now }
	for _, offset := range []time.Duration{-5 * time.Second, 5 * time.Second} {
		issuer, err := NewIssuer("web-api", key)
		if err != nil {
			t.Fatal(err)
		}
		issuer.now = func() time.Time { return now.Add(offset) }
		assertion, err := issuer.IssueService(authentication.Tenant{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := verifier.Verify(assertion); err != nil {
			t.Fatalf("clock offset %s rejected: %v", offset, err)
		}
	}
	issuer, err := NewIssuer("web-api", key)
	if err != nil {
		t.Fatal(err)
	}
	issuer.now = func() time.Time { return now.Add(6 * time.Second) }
	assertion, err := issuer.IssueService(authentication.Tenant{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(assertion); err == nil {
		t.Fatal("clock offset beyond budget accepted")
	}
}

func TestAssertionConfigurationRejectsUnsafeKeys(t *testing.T) {
	key := assertionTestKey(6)
	if _, err := NewIssuer("", key); err == nil {
		t.Fatal("blank audience accepted")
	}
	if _, err := NewIssuer("web-api", nil); err == nil {
		t.Fatal("missing private key accepted")
	}

}
