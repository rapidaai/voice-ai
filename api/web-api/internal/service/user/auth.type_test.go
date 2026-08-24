package internal_user_service

import (
	"context"
	"math"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
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

var _ types.Principle = (*authPrinciple)(nil)
var _ types.ProjectContextProvider = (*authPrinciple)(nil)

func TestAuthPrincipleAuditActor(t *testing.T) {
	principle := &authPrinciple{user: &internal_entity.UserAuth{}}
	principle.user.Id = 73

	actor, ok := principle.AuditActor()
	if !ok {
		t.Fatal("AuditActor() ok = false, want true")
	}
	if actor != (types.ActorIdentity{Type: types.ActorTypeUser, ID: 73}) {
		t.Fatalf("AuditActor() = %+v", actor)
	}
}

func TestAuthPrincipleAuditActorRange(t *testing.T) {
	tests := []struct {
		name string
		id   uint64
		ok   bool
	}{
		{name: "zero rejected", id: 0},
		{name: "max bigint accepted", id: math.MaxInt64, ok: true},
		{name: "above max bigint rejected", id: uint64(math.MaxInt64) + 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principle := &authPrinciple{user: &internal_entity.UserAuth{}}
			principle.user.Id = test.id
			actor, ok := principle.AuditActor()
			if ok != test.ok {
				t.Fatalf("AuditActor() ok = %v, want %v", ok, test.ok)
			}
			if ok && actor.ID != test.id {
				t.Fatalf("AuditActor() ID = %d, want %d", actor.ID, test.id)
			}
		})
	}
}

func TestAuthPrincipleCapabilities(t *testing.T) {
	user := &internal_entity.UserAuth{}
	user.Id = 73
	principle := &authPrinciple{
		user: user,
		userOrgRole: &internal_entity.UserOrganizationRole{
			OrganizationId: 81,
		},
		currentProjectRole: &types.ProjectRole{ProjectId: 92},
	}

	userID, ok := principle.UserIdentity()
	if !ok || userID != 73 {
		t.Fatalf("UserIdentity() = %d, %v", userID, ok)
	}
	organizationID, ok := principle.OrganizationContext()
	if !ok || organizationID != 81 {
		t.Fatalf("OrganizationContext() = %d, %v", organizationID, ok)
	}
	projectContext, ok := principle.ProjectContext()
	if !ok {
		t.Fatal("ProjectContext() ok = false")
	}
	if projectContext != (types.ProjectContext{OrganizationID: 81, ProjectID: 92}) {
		t.Fatalf("ProjectContext() = %+v", projectContext)
	}
}

func TestAuthPrincipleProjectContextOptional(t *testing.T) {
	user := &internal_entity.UserAuth{}
	user.Id = 73
	principle := &authPrinciple{
		user: user,
		userOrgRole: &internal_entity.UserOrganizationRole{
			OrganizationId: 81,
		},
	}

	if !principle.IsAuthenticated() {
		t.Fatal("IsAuthenticated() = false without selected project")
	}
	if _, ok := principle.ProjectContext(); ok {
		t.Fatal("ProjectContext() ok = true without selected project")
	}
}

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
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 73},
		UserValue:         &types.UserContext{UserID: 73},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 81},
	}
	source := "invited-by-other"
	if _, err := service.Create(context.Background(), auth, "Invited", "invited@example.com", "password", type_enums.RECORD_INVITED, &source); err != nil {
		t.Fatalf("Create() error = %v", err)
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
