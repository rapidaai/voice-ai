package internal_project_service

import (
	"context"
	"math"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
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
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`, int64(math.MaxInt64), 7, 9, "ACTIVE", "secret").Error; err != nil {
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
	if !ok || actor.ID != math.MaxInt64 {
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

func TestProjectCredentialWritersPersistActorIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE project_credentials (
        id INTEGER PRIMARY KEY,
        project_id INTEGER NOT NULL,
        organization_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        key TEXT NOT NULL,
        status TEXT NOT NULL,
        created_actor_type TEXT,
        created_actor_id INTEGER,
        updated_actor_type TEXT,
        updated_actor_id INTEGER,
        created_date DATETIME NOT NULL,
        updated_date DATETIME
    )`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewProjectService(logger, &projectTestPostgres{db: db})
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 11},
		UserValue:         &types.UserContext{UserID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 9},
	}

	credential, err := service.CreateCredential(context.Background(), auth, " primary ", 7, 9)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}
	if credential.Id == 0 || credential.Name != "primary" || credential.CreatedActorType == nil || *credential.CreatedActorType != "user" || credential.CreatedActorID == nil || *credential.CreatedActorID != 11 {
		t.Fatalf("created credential = %+v", credential)
	}

	archived, err := service.ArchiveCredential(context.Background(), auth, credential.Id, 7, 9)
	if err != nil {
		t.Fatalf("ArchiveCredential() error = %v", err)
	}
	if archived.Status != type_enums.RECORD_ARCHIEVE || archived.UpdatedActorType == nil || *archived.UpdatedActorType != "user" || archived.UpdatedActorID == nil || *archived.UpdatedActorID != 11 {
		t.Fatalf("archived credential = %+v", archived)
	}
}

func TestProjectCredentialWritersRejectMismatchedOrganization(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewProjectService(logger, &projectTestPostgres{})
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 11},
		UserValue:         &types.UserContext{UserID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 10},
	}
	if _, err := service.CreateCredential(context.Background(), auth, "primary", 7, 9); err == nil {
		t.Fatal("CreateCredential() error = nil for mismatched organization")
	}
}
