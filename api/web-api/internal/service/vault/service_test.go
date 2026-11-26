package internal_vault_service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
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
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER,
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
	projectContext := authentication.Tenant{OrganizationID: 81, ProjectID: 92}
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 73},
		authentication.Tenant{OrganizationID: 81, ProjectID: 92},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "vault-test"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	vault, err := service.Create(context.Background(), auth, "provider", "credential", map[string]interface{}{"token": "secret"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if vault.OrganizationId != 81 || vault.ProjectId != 92 {
		t.Fatalf("Create() identity context = organization:%d project:%d", vault.OrganizationId, vault.ProjectId)
	}
	if vault.CreatedActorType != "user" || vault.CreatedActorID != 73 {
		t.Fatalf("Create() actor = %v/%v", vault.CreatedActorType, vault.CreatedActorID)
	}

	loaded, err := service.Get(context.Background(), projectContext, vault.Id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.Id != vault.Id {
		t.Fatalf("Get() id = %d, want %d", loaded.Id, vault.Id)
	}

	if _, err := service.Get(context.Background(), authentication.Tenant{OrganizationID: 81, ProjectID: 93}, vault.Id); err == nil {
		t.Fatal("Get() error = nil for mismatched project context")
	}
}

func TestVaultServiceDeleteReturnsDatabaseError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewVaultService(logger, &vaultTestPostgres{db: db})
	auth, principalErr := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 73},
		authentication.Tenant{OrganizationID: 81, ProjectID: 92},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "vault-delete-test"}}, nil,
	)
	if principalErr != nil {
		t.Fatal(principalErr)
	}
	if _, err := service.Delete(context.Background(), auth, 1); err == nil {
		t.Fatal("Delete() error = nil for missing vault table")
	}
}

func TestVaultMetadataListUsesFixedANDCriteriaTenantBindingAndStableOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE projects (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, status TEXT NOT NULL)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE vaults (
		id INTEGER PRIMARY KEY, created_date DATETIME NOT NULL, updated_date DATETIME,
		status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER,
		updated_actor_type TEXT, updated_actor_id INTEGER, project_id INTEGER NOT NULL,
		organization_id INTEGER NOT NULL, provider TEXT NOT NULL, name TEXT NOT NULL, value TEXT NOT NULL
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO projects(id,organization_id,status) VALUES (92,81,'ACTIVE'),(93,81,'ACTIVE'),(94,82,'ACTIVE')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO vaults(id,created_date,status,project_id,organization_id,provider,name,value) VALUES
		(1,'2026-01-01T00:00:00Z','ACTIVE',92,81,'openai','Alpha%_\Key','{"token":"secret-1"}'),
		(2,'2026-01-01T00:00:00Z','ACTIVE',92,81,'openai','Alpha%_\Key second','{"token":"secret-2"}'),
		(3,'2026-01-02T00:00:00Z','ACTIVE',92,81,'anthropic','Alpha%_\Key','{"token":"secret-3"}'),
		(4,'2026-01-03T00:00:00Z','ACTIVE',93,81,'openai','Alpha%_\Key','{"token":"other-project"}'),
		(5,'2026-01-03T00:00:00Z','ACTIVE',94,82,'openai','Alpha%_\Key','{"token":"other-org"}'),
		(6,'2026-01-04T00:00:00Z','ARCHIEVE',92,81,'openai','Alpha%_\Key','{"token":"archived"}')`).Error)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	require.NoError(t, err)
	service := NewVaultService(logger, &vaultTestPostgres{db: db})
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 73},
		authentication.Tenant{OrganizationID: 81, ProjectID: 92},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "vault-list-test"}}, nil,
	)
	require.NoError(t, err)
	count, rows, err := service.GetAllOrganizationCredential(context.Background(), auth, []*protos.Criteria{
		{Key: "name", Logic: "contains", Value: `%_\`},
		{Key: "provider", Logic: "=", Value: "openai"},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Len(t, rows, 2)
	require.Equal(t, []uint64{2, 1}, []uint64{rows[0].Id, rows[1].Id})

	count, rows, err = service.GetAllOrganizationCredential(context.Background(), auth, nil, &protos.Paginate{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	require.Equal(t, []uint64{3, 2}, []uint64{rows[0].Id, rows[1].Id})
}
