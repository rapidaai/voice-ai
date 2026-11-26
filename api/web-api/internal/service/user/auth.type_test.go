package internal_user_service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type userServiceTestPostgres struct {
	db *gorm.DB
}

func (p *userServiceTestPostgres) Connect(context.Context) error    { return nil }
func (p *userServiceTestPostgres) Name() string                     { return "user-service-test" }
func (p *userServiceTestPostgres) IsConnected(context.Context) bool { return true }
func (p *userServiceTestPostgres) Disconnect(context.Context) error { return nil }
func (p *userServiceTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *userServiceTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*userServiceTestPostgres)(nil)

func TestUserServicePersistsProvidedAuditActor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_auths (
		id INTEGER PRIMARY KEY,
		created_date DATETIME NOT NULL,
		updated_date DATETIME,
		status TEXT NOT NULL,
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		password TEXT NOT NULL,
		source TEXT NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_auth_tokens (
		id INTEGER PRIMARY KEY,
		created_date DATETIME NOT NULL,
		updated_date DATETIME,
		status TEXT NOT NULL,
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER,
		user_auth_id INTEGER NOT NULL,
		token_type TEXT NOT NULL,
		token TEXT NOT NULL,
		expire_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewUserService(logger, &userServiceTestPostgres{db: db})
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 73},
		authentication.Tenant{OrganizationID: 81},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "user-service-test"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	source := "invited-by-other"
	created, err := service.Create(context.Background(), auth, "Invited", "invited@example.com", "password", type_enums.RECORD_INVITED, &source)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.CreateNewAuthToken(context.Background(), auth, created.Id); err != nil {
		t.Fatalf("CreateNewAuthToken() error = %v", err)
	}

	var user internal_entity.UserAuth
	if err := db.First(&user, "email = ?", "invited@example.com").Error; err != nil {
		t.Fatal(err)
	}
	if user.CreatedActorType != "user" || user.CreatedActorID != 73 {
		t.Fatalf("Create() actor = %q/%d", user.CreatedActorType, user.CreatedActorID)
	}

	var token internal_entity.UserAuthToken
	if err := db.First(&token, "user_auth_id = ?", user.Id).Error; err != nil {
		t.Fatal(err)
	}
	if token.CreatedActorType != "user" || token.CreatedActorID != 73 {
		t.Fatalf("CreateNewAuthToken() actor = %q/%d", token.CreatedActorType, token.CreatedActorID)
	}

	if _, err := service.UpdatePassword(context.Background(), auth, user.Id, "updated-password"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	if err := db.First(&user, "id = ?", user.Id).Error; err != nil {
		t.Fatal(err)
	}
	if user.UpdatedActorType != "user" || user.UpdatedActorID != 73 {
		t.Fatalf("UpdatePassword() actor = %q/%d", user.UpdatedActorType, user.UpdatedActorID)
	}
}
