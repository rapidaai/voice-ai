// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_connect

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

type externalConnectTestPostgres struct {
	db *gorm.DB
}

func (postgres *externalConnectTestPostgres) Connect(context.Context) error    { return nil }
func (postgres *externalConnectTestPostgres) Name() string                     { return "external-connect-test" }
func (postgres *externalConnectTestPostgres) IsConnected(context.Context) bool { return true }
func (postgres *externalConnectTestPostgres) Disconnect(context.Context) error { return nil }
func (postgres *externalConnectTestPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return postgres.DB(ctx).Raw(query).Scan(destination).Error
}
func (postgres *externalConnectTestPostgres) DB(ctx context.Context) *gorm.DB {
	return postgres.db.WithContext(ctx)
}

var _ connectors.PostgresConnector = (*externalConnectTestPostgres)(nil)

func externalConnectPrincipal(t *testing.T, organizationID, projectID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "test-session"},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func TestDecodeStateRequiresExactPrincipalTenant(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&internal_entity.OAuthExternalConnect{}); err != nil {
		t.Fatal(err)
	}
	wanted := &internal_entity.OAuthExternalConnect{
		Identifier: "state-token",
		Provider:   "hubspot",
		RedirectTo: "/settings",
	}
	wanted.OrganizationId = 10
	wanted.ProjectId = 20
	if err := database.Create(wanted).Error; err != nil {
		t.Fatal(err)
	}
	connect := ExternalConnect{postgres: &externalConnectTestPostgres{db: database}}

	decoded, err := connect.DecodeState(context.Background(), externalConnectPrincipal(t, 10, 20), "state-token")
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Identifier != wanted.Identifier || decoded.OrganizationId != 10 || decoded.ProjectId != 20 {
		t.Fatalf("DecodeState() = %+v", decoded)
	}

	if _, err := connect.DecodeState(context.Background(), externalConnectPrincipal(t, 10, 21), "state-token"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("DecodeState() cross-tenant error = %v, want record not found", err)
	}
}

func TestDecodeStateRejectsPrincipalWithoutProjectTenant(t *testing.T) {
	connect := ExternalConnect{}
	_, err := connect.DecodeState(context.Background(), externalConnectPrincipal(t, 10, 0), "state-token")
	var denied pkg_errors.RouteAuthorizationDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("DecodeState() error = %v, want route authorization denied", err)
	}
}
