package internal_project_service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type projectTestPostgres struct {
	db *gorm.DB
}

func (p *projectTestPostgres) Connect(context.Context) error    { return nil }
func (p *projectTestPostgres) Name() string                     { return "project-test" }
func (p *projectTestPostgres) IsConnected(context.Context) bool { return true }
func (p *projectTestPostgres) Disconnect(context.Context) error { return nil }
func (p *projectTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *projectTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*projectTestPostgres)(nil)

func TestProjectServiceClaimSelectsCredentialID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE project_credentials (
        id INTEGER PRIMARY KEY,
        project_id INTEGER NOT NULL,
        organization_id INTEGER NOT NULL,
        status TEXT NOT NULL,
        key TEXT NOT NULL,
        created_date DATETIME NOT NULL
    )`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO project_credentials
        (id, project_id, organization_id, status, key, created_date)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`, 42, 7, 9, "ACTIVE", "secret").Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewProjectAuthenticator(logger, &projectTestPostgres{db: db})
	principle, err := service.Claim(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	actor, ok := principle.Info.AuditActor()
	if !ok || actor.ID != "42" {
		t.Fatalf("AuditActor() = %+v, %v", actor, ok)
	}
	projectContext, ok := principle.Info.ProjectContext()
	if !ok {
		t.Fatal("ProjectContext() ok = false")
	}
	if projectContext != (types.ProjectContext{OrganizationID: 9, ProjectID: 7}) {
		t.Fatalf("ProjectContext() = %+v", projectContext)
	}
}
