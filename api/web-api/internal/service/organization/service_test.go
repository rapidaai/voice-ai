package internal_organization_service

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type organizationTestPostgres struct {
	db *gorm.DB
}

func TestOrganizationUpdateBindsPrincipalTenantAndAuditActor(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TABLE organizations (
		id INTEGER PRIMARY KEY, name TEXT, industry TEXT, email TEXT, status TEXT,
		updated_actor_type TEXT, updated_actor_id INTEGER, updated_date DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("INSERT INTO organizations(id,name,status) VALUES (22,'before','ACTIVE'),(23,'other','ACTIVE')").Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewOrganizationService(logger, &organizationTestPostgres{db: database})
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.Tenant{OrganizationID: 22},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "organization-update-test"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	name := "after"
	if _, err := service.Update(context.Background(), principal, 22, &name, nil, nil); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	var persisted struct {
		Name             string
		UpdatedActorType string
		UpdatedActorID   uint64
	}
	if err := database.Table("organizations").Where("id = ?", 22).Take(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Name != name || persisted.UpdatedActorType != "user" || persisted.UpdatedActorID != 11 {
		t.Fatalf("persisted update = %+v", persisted)
	}
	if _, err := service.Update(context.Background(), principal, 23, &name, nil, nil); err == nil {
		t.Fatal("Update() cross-tenant error = nil")
	} else {
		var denied pkg_errors.RouteAuthorizationDeniedError
		if !errors.As(err, &denied) {
			t.Fatalf("Update() cross-tenant error = %v", err)
		}
	}
}

func (p *organizationTestPostgres) Connect(context.Context) error    { return nil }
func (p *organizationTestPostgres) Name() string                     { return "organization-test" }
func (p *organizationTestPostgres) IsConnected(context.Context) bool { return true }
func (p *organizationTestPostgres) Disconnect(context.Context) error { return nil }
func (p *organizationTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *organizationTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*organizationTestPostgres)(nil)
