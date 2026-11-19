// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package jwt

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	jwtv5 "github.com/golang-jwt/jwt/v5"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/validator"
)

const (
	assertionLifetime        = 55 * time.Second
	assertionMaximumLifetime = 60 * time.Second
	internalServiceID        = uint64(1)
	internalServiceIssuer    = "internal"
)

type assertionIdentity struct {
	Type authentication.IdentityType `json:"type"`
	ID   uint64                      `json:"id"`
}

func (identity *assertionIdentity) UnmarshalJSON(source []byte) error {
	if validateClosedJSONObject(source, []string{"type", "id"}, nil) != nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	var decoded struct {
		Type *authentication.IdentityType `json:"type"`
		ID   *uint64                      `json:"id"`
	}
	if err := json.Unmarshal(source, &decoded); err != nil || decoded.Type == nil || decoded.ID == nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	*identity = assertionIdentity{Type: *decoded.Type, ID: *decoded.ID}
	return nil
}

type assertionTenant struct {
	OrganizationID uint64 `json:"organization_id"`
	ProjectID      uint64 `json:"project_id"`
}

func (tenant *assertionTenant) UnmarshalJSON(source []byte) error {
	if validateClosedJSONObject(source, []string{"organization_id", "project_id"}, nil) != nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	var decoded struct {
		OrganizationID *uint64 `json:"organization_id"`
		ProjectID      *uint64 `json:"project_id"`
	}
	if err := json.Unmarshal(source, &decoded); err != nil || decoded.OrganizationID == nil || decoded.ProjectID == nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	*tenant = assertionTenant{OrganizationID: *decoded.OrganizationID, ProjectID: *decoded.ProjectID}
	return nil
}

type assertionAuthentication struct {
	Subject     *assertionIdentity              `json:"subject,omitempty"`
	Tenant      *assertionTenant                `json:"tenant"`
	Permissions *[]authentication.PermissionKey `json:"permissions"`
}

func (authenticationClaims *assertionAuthentication) UnmarshalJSON(source []byte) error {
	if validateClosedJSONObject(source, []string{"tenant", "permissions"}, []string{"subject"}) != nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	type wireAuthentication assertionAuthentication
	var decoded wireAuthentication
	if err := json.Unmarshal(source, &decoded); err != nil || decoded.Tenant == nil || decoded.Permissions == nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	*authenticationClaims = assertionAuthentication(decoded)
	return nil
}

type assertionClaims struct {
	Audience       string                   `json:"aud"`
	ExpiresAt      *int64                   `json:"exp"`
	Authentication *assertionAuthentication `json:"authentication"`
}

func (claims assertionClaims) GetExpirationTime() (*jwtv5.NumericDate, error) {
	if claims.ExpiresAt == nil {
		return nil, nil
	}
	return jwtv5.NewNumericDate(time.Unix(*claims.ExpiresAt, 0)), nil
}

func (assertionClaims) GetIssuedAt() (*jwtv5.NumericDate, error) { return nil, nil }

func (assertionClaims) GetNotBefore() (*jwtv5.NumericDate, error) { return nil, nil }

func (assertionClaims) GetIssuer() (string, error) { return "", nil }

func (assertionClaims) GetSubject() (string, error) { return "", nil }

func (claims assertionClaims) GetAudience() (jwtv5.ClaimStrings, error) {
	if claims.Audience == "" {
		return nil, nil
	}
	return jwtv5.ClaimStrings{claims.Audience}, nil
}

func (claims *assertionClaims) UnmarshalJSON(source []byte) error {
	if validateClosedJSONObject(source, []string{"aud", "exp", "authentication"}, nil) != nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	type wireClaims assertionClaims
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode((*wireClaims)(claims)); err != nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	return nil
}

func validateClosedJSONObject(source []byte, required []string, optional []string) error {
	if err := validateUniqueJSONMembers(source); err != nil {
		return err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(source, &fields); err != nil || fields == nil {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	allowed := make(map[string]struct{}, len(required)+len(optional))
	for _, name := range required {
		allowed[name] = struct{}{}
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	for _, name := range optional {
		allowed[name] = struct{}{}
		if value, exists := fields[name]; exists && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	for name := range fields {
		if _, exists := allowed[name]; !exists {
			return pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	return nil
}

func validateUniqueJSONMembers(source []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(source))
	var readValue func() error
	readValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return pkg_errors.AuthenticationCredentialInvalidError{}
		}
		delimiter, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}
		switch delimiter {
		case '{':
			members := map[string]struct{}{}
			for decoder.More() {
				member, err := decoder.Token()
				name, isString := member.(string)
				if err != nil || !isString {
					return pkg_errors.AuthenticationCredentialInvalidError{}
				}
				if _, exists := members[name]; exists {
					return pkg_errors.AuthenticationCredentialInvalidError{}
				}
				members[name] = struct{}{}
				if err := readValue(); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim('}') {
				return pkg_errors.AuthenticationCredentialInvalidError{}
			}
		case '[':
			for decoder.More() {
				if err := readValue(); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim(']') {
				return pkg_errors.AuthenticationCredentialInvalidError{}
			}
		default:
			return pkg_errors.AuthenticationCredentialInvalidError{}
		}
		return nil
	}
	if err := readValue(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return pkg_errors.AuthenticationCredentialInvalidError{}
	}
	return nil
}

// Issuer creates destination-bound assertions signed by one service-authentication key.
type Issuer struct {
	audience    string
	key         []byte
	fingerprint string
	now         func() time.Time
}

// NewIssuer constructs an immutable HS256 assertion issuer.
func NewIssuer(audience string, key []byte) (*Issuer, error) {
	if !utf8.ValidString(audience) || !validator.NotBlank(audience) || strings.TrimSpace(audience) != audience ||
		strings.ContainsRune(audience, '\x00') || len(key) != sha256.Size {
		return nil, pkg_errors.JWTIssuerInvalidError{}
	}
	fingerprint := sha256.Sum256(key)
	return &Issuer{
		audience: audience, key: append([]byte(nil), key...),
		fingerprint: base64.RawURLEncoding.EncodeToString(fingerprint[:]), now: time.Now,
	}, nil
}

// Issue signs one direct or delegated request authentication snapshot.
func (issuer *Issuer) Issue(subject authentication.Identity, tenant authentication.Tenant, authorities []authentication.PermissionKey) (string, error) {
	proof := authentication.Proof{
		Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal,
		Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: issuer.fingerprint},
	}
	genericActor := authentication.ServiceActor{ID: internalServiceID, Issuer: internalServiceIssuer}
	var assertedSubject *assertionIdentity
	if subject.Type == authentication.IdentityService {
		if _, err := authentication.NewDirectPrincipal(subject, tenant, proof, authorities); err != nil {
			return "", err
		}
	} else {
		if _, err := authentication.NewDelegatedPrincipal(subject, genericActor, tenant, proof, authorities); err != nil {
			return "", err
		}
		assertedSubject = &assertionIdentity{Type: subject.Type, ID: subject.ID}
	}
	permissions := make([]authentication.PermissionKey, len(authorities))
	copy(permissions, authorities)
	sort.Slice(permissions, func(left int, right int) bool { return permissions[left] < permissions[right] })
	expiresAt := issuer.now().UTC().Truncate(time.Second).Add(assertionLifetime).Unix()
	claims := assertionClaims{
		Audience: issuer.audience, ExpiresAt: &expiresAt,
		Authentication: &assertionAuthentication{
			Subject:     assertedSubject,
			Tenant:      &assertionTenant{OrganizationID: tenant.OrganizationID, ProjectID: tenant.ProjectID},
			Permissions: &permissions,
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	assertion, err := token.SignedString(issuer.key)
	if err != nil {
		return "", pkg_errors.JWTIssueError{}
	}
	return assertion, nil
}

// IssueService signs a direct internal-Service assertion for the selected Tenant.
func (issuer *Issuer) IssueService(tenant authentication.Tenant) (string, error) {
	return issuer.Issue(
		authentication.Identity{Type: authentication.IdentityService, ID: internalServiceID, Issuer: internalServiceIssuer},
		tenant, nil,
	)
}

// Verifier verifies assertions for exactly one destination service.
type Verifier struct {
	audience string
	key      []byte
	now      func() time.Time
}

// NewVerifier constructs an immutable HS256 verifier.
func NewVerifier(audience string, key []byte) Verifier {
	return Verifier{audience: audience, key: append([]byte(nil), key...), now: time.Now}
}

// VerifiedAssertion is an immutable verified request authentication snapshot.
type VerifiedAssertion struct {
	actor          authentication.ServiceActor
	subject        authentication.Identity
	tenant         authentication.Tenant
	authorities    []authentication.PermissionKey
	destination    string
	keyFingerprint string
	expiresAt      time.Time
}

func (assertion VerifiedAssertion) Actor() authentication.ServiceActor { return assertion.actor }

func (assertion VerifiedAssertion) Subject() authentication.Identity { return assertion.subject }

func (assertion VerifiedAssertion) Tenant() authentication.Tenant { return assertion.tenant }

func (assertion VerifiedAssertion) Authorities() []authentication.PermissionKey {
	return append([]authentication.PermissionKey(nil), assertion.authorities...)
}

func (assertion VerifiedAssertion) Destination() string { return assertion.destination }

func (assertion VerifiedAssertion) KeyFingerprint() string { return assertion.keyFingerprint }

func (assertion VerifiedAssertion) ExpiresAt() time.Time { return assertion.expiresAt }

// Verify authenticates one assertion and returns its server-derived caller facts.
func (verifier Verifier) Verify(assertion string) (VerifiedAssertion, error) {
	if verifier.now == nil || !validator.NotBlank(assertion) || strings.TrimSpace(assertion) != assertion {
		return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	now := verifier.now().UTC().Truncate(time.Second)
	segments := strings.Split(assertion, ".")
	if len(segments) != 3 {
		return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	for index, segment := range segments {
		decoded, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != segment {
			return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
		if index < 2 {
			if err := validateUniqueJSONMembers(decoded); err != nil {
				return VerifiedAssertion{}, err
			}
		} else if len(decoded) != sha256.Size {
			return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	verifiedClaims := &assertionClaims{}
	token, err := jwtv5.NewParser(
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
		jwtv5.WithAudience(verifier.audience), jwtv5.WithExpirationRequired(),
		jwtv5.WithStrictDecoding(),
		jwtv5.WithTimeFunc(func() time.Time { return now }),
	).ParseWithClaims(assertion, verifiedClaims, func(*jwtv5.Token) (interface{}, error) { return verifier.key, nil })
	if err != nil || !token.Valid || len(token.Header) != 2 || token.Header["alg"] != jwtv5.SigningMethodHS256.Alg() || token.Header["typ"] != "JWT" ||
		verifiedClaims.ExpiresAt == nil || verifiedClaims.Authentication == nil ||
		verifiedClaims.Authentication.Tenant == nil || verifiedClaims.Authentication.Permissions == nil {
		return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	expiresAt := time.Unix(*verifiedClaims.ExpiresAt, 0).UTC()
	if !expiresAt.After(now) || expiresAt.After(now.Add(assertionMaximumLifetime)) {
		return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	permissions := append([]authentication.PermissionKey(nil), (*verifiedClaims.Authentication.Permissions)...)
	if !sort.SliceIsSorted(permissions, func(left int, right int) bool { return permissions[left] < permissions[right] }) {
		return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	for index := 1; index < len(permissions); index++ {
		if permissions[index-1] == permissions[index] {
			return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	fingerprint := sha256.Sum256(verifier.key)
	keyFingerprint := base64.RawURLEncoding.EncodeToString(fingerprint[:])
	proof := authentication.Proof{
		Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal,
		Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: keyFingerprint},
	}
	actor := authentication.ServiceActor{ID: internalServiceID, Issuer: internalServiceIssuer}
	tenant := authentication.Tenant{
		OrganizationID: verifiedClaims.Authentication.Tenant.OrganizationID,
		ProjectID:      verifiedClaims.Authentication.Tenant.ProjectID,
	}
	subject := authentication.Identity{Type: authentication.IdentityService, ID: internalServiceID, Issuer: internalServiceIssuer}
	if verifiedClaims.Authentication.Subject == nil {
		if _, err := authentication.NewDirectPrincipal(subject, tenant, proof, permissions); err != nil {
			return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	} else {
		subject = authentication.Identity{
			Type: verifiedClaims.Authentication.Subject.Type,
			ID:   verifiedClaims.Authentication.Subject.ID,
		}
		if _, err := authentication.NewDelegatedPrincipal(subject, actor, tenant, proof, permissions); err != nil {
			return VerifiedAssertion{}, pkg_errors.AuthenticationCredentialInvalidError{}
		}
	}
	return VerifiedAssertion{
		actor: actor, subject: subject, tenant: tenant, authorities: permissions,
		destination: verifier.audience, keyFingerprint: keyFingerprint, expiresAt: expiresAt,
	}, nil
}
