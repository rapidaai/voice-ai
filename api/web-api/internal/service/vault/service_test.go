package internal_vault_service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
)

type vaultTestPostgres struct {
	db *gorm.DB
}

func (p *vaultTestPostgres) Connect(context.Context) error    { return nil }
func (p *vaultTestPostgres) Name() string                     { return "vault-test" }
func (p *vaultTestPostgres) IsConnected(context.Context) bool { return true }
func (p *vaultTestPostgres) Disconnect(context.Context) error { return nil }
func (p *vaultTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *vaultTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*vaultTestPostgres)(nil)

func TestVaultServiceUsesNormalizedIdentityContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE vaults (
		id INTEGER PRIMARY KEY,
		created_date DATETIME NOT NULL,
		updated_date DATETIME,
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		created_by INTEGER NOT NULL,
		updated_by INTEGER,
		project_id INTEGER NOT NULL,
		organization_id INTEGER NOT NULL,
		provider TEXT NOT NULL,
		name TEXT NOT NULL,
		value TEXT NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewVaultService(logger, &vaultTestPostgres{db: db})
	projectContext := types.ProjectContext{OrganizationID: 81, ProjectID: 92}

	vault, err := service.Create(context.Background(), 73, projectContext, "provider", "credential", map[string]interface{}{"token": "secret"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if vault.CreatedBy != 73 || vault.OrganizationId != 81 || vault.ProjectId != 92 {
		t.Fatalf("Create() identity context = user:%d organization:%d project:%d", vault.CreatedBy, vault.OrganizationId, vault.ProjectId)
	}

	loaded, err := service.Get(context.Background(), projectContext, vault.Id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.Id != vault.Id {
		t.Fatalf("Get() id = %d, want %d", loaded.Id, vault.Id)
	}

	if _, err := service.Get(context.Background(), types.ProjectContext{OrganizationID: 81, ProjectID: 93}, vault.Id); err == nil {
		t.Fatal("Get() error = nil for mismatched project context")
	}
}
