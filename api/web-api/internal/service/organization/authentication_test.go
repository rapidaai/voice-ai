// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_organization_service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/config"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func TestPrincipalAuthenticationServiceRequiresActiveOrganization(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TABLE organizations (id INTEGER PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TABLE organization_credentials (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, status TEXT NOT NULL, key TEXT NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	const rawKey = "organization-secret"
	const rootSecret = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
	fingerprintKey := (&config.AppConfig{Secret: rootSecret}).OrganizationCredentialFingerprintKey()
	mac := hmac.New(sha256.New, fingerprintKey)
	_, _ = mac.Write([]byte(rawKey))
	decodedRoot, err := base64.StdEncoding.Strict().DecodeString(rootSecret)
	if err != nil {
		t.Fatal(err)
	}
	previousKey := hmac.New(sha256.New, decodedRoot)
	_, _ = previousKey.Write([]byte("rapida/organization-credential-fingerprint/v1"))
	previousFingerprint := hmac.New(sha256.New, previousKey.Sum(nil))
	_, _ = previousFingerprint.Write([]byte(rawKey))
	if err := database.Exec(`INSERT INTO organizations VALUES (9,'Organization','ACTIVE')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`INSERT INTO organization_credentials VALUES (11,9,'ACTIVE',?)`, hex.EncodeToString(previousFingerprint.Sum(nil))).Error; err != nil {
		t.Fatal(err)
	}
	service := NewPrincipalAuthenticationService(nil, &organizationTestPostgres{db: database}, fingerprintKey)
	_, err = service.ResolveOrganizationCredential(context.Background(), rawKey)
	var invalid pkg_errors.AuthenticationCredentialInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("previous Organization fingerprint error = %v", err)
	}
	if err := database.Exec(`UPDATE organization_credentials SET key = ? WHERE id = 11`, hex.EncodeToString(mac.Sum(nil))).Error; err != nil {
		t.Fatal(err)
	}
	value, err := service.ResolveOrganizationCredential(context.Background(), rawKey)
	principal := value.Principal()
	if err != nil || principal.Subject().ID != 11 || principal.Tenant().OrganizationID != 9 {
		t.Fatalf("principal = %+v, %v", principal, err)
	}
	if err := database.Exec(`UPDATE organizations SET status = 'ARCHIEVE' WHERE id = 9`).Error; err != nil {
		t.Fatal(err)
	}
	_, err = service.ResolveOrganizationCredential(context.Background(), rawKey)
	if !errors.As(err, &invalid) {
		t.Fatalf("archived organization error = %v", err)
	}
}
