package internal_organization_service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type organizationTestPostgres struct {
	db *gorm.DB
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

func TestOrganizationCredentialLifecycleAndAuthentication(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE organization_credentials (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		key TEXT NOT NULL UNIQUE,
		created_date DATETIME NOT NULL,
		updated_date DATETIME,
		status TEXT NOT NULL,
		created_actor_type TEXT,
		created_actor_id INTEGER,
		updated_actor_type TEXT,
		updated_actor_id INTEGER,
		archived_date DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	const fingerprintKey = "test-fingerprint-key"
	postgres := &organizationTestPostgres{db: db}
	service := NewOrganizationService(logger, postgres, fingerprintKey)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: 11},
		UserValue:         &types.UserContext{UserID: 11},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 22},
	}

	first, firstKey, err := service.CreateCredential(context.Background(), auth, 22, "primary")
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}
	second, _, err := service.CreateCredential(context.Background(), auth, 22, "secondary")
	if err != nil {
		t.Fatalf("CreateCredential() second error = %v", err)
	}
	if first.Id == 0 || first.Id == second.Id {
		t.Fatalf("credential IDs = %d and %d", first.Id, second.Id)
	}
	mac := hmac.New(sha256.New, []byte(fingerprintKey))
	_, _ = mac.Write([]byte(firstKey))
	wantFingerprint := hex.EncodeToString(mac.Sum(nil))
	var stored internal_entity.OrganizationCredential
	if err := db.Where("id = ?", first.Id).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Key != wantFingerprint || stored.Key == firstKey {
		t.Fatalf("stored key = %q, want HMAC fingerprint", stored.Key)
	}
	var createdActor struct {
		Type string
		ID   uint64
	}
	if err := db.Table("organization_credentials").
		Select("created_actor_type AS type, created_actor_id AS id").
		Where("id = ?", first.Id).
		Scan(&createdActor).Error; err != nil {
		t.Fatal(err)
	}
	if createdActor.Type != "user" || createdActor.ID != 11 {
		t.Fatalf("created actor = %q/%d", createdActor.Type, createdActor.ID)
	}
	var updatedActorType sql.NullString
	var updatedActorID sql.NullInt64
	if err := db.Table("organization_credentials").
		Select("updated_actor_type, updated_actor_id").
		Where("id = ?", first.Id).
		Row().Scan(&updatedActorType, &updatedActorID); err != nil {
		t.Fatal(err)
	}
	if updatedActorType.Valid || updatedActorID.Valid {
		t.Fatalf("created credential update actor = %v/%v, want null", updatedActorType, updatedActorID)
	}

	authenticator := NewOrganizationAuthenticator(logger, postgres, fingerprintKey)
	principle, err := authenticator.Claim(context.Background(), firstKey)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	actor, err := types.ResolveAuditActor(principle.Info)
	if err != nil || actor.ID != first.Id || actor.Type != types.ActorTypeOrganization {
		t.Fatalf("ResolveAuditActor() = %+v, %v", actor, err)
	}
	organizationID, ok := principle.Info.OrganizationContext()
	if !ok || organizationID != 22 || actor.ID == organizationID {
		t.Fatalf("organization scope = %d, %v; actor = %d", organizationID, ok, actor.ID)
	}

	rotated, rotatedKey, err := service.RotateCredential(context.Background(), auth, 22, first.Id)
	if err != nil {
		t.Fatalf("RotateCredential() error = %v", err)
	}
	if rotated.Id != first.Id || rotatedKey == firstKey {
		t.Fatalf("rotated credential = %d/%q", rotated.Id, rotatedKey)
	}
	if _, err := authenticator.Claim(context.Background(), firstKey); err == nil {
		t.Fatal("old credential authenticated after rotation")
	}
	if _, err := authenticator.Claim(context.Background(), rotatedKey); err != nil {
		t.Fatalf("rotated credential failed authentication: %v", err)
	}

	archived, err := service.ArchiveCredential(context.Background(), auth, 22, first.Id)
	if err != nil {
		t.Fatalf("ArchiveCredential() error = %v", err)
	}
	if archived.Status != type_enums.RECORD_ARCHIEVE {
		t.Fatalf("archived status = %q", archived.Status)
	}
	var updatedActor struct {
		Type string
		ID   uint64
	}
	if err := db.Table("organization_credentials").
		Select("updated_actor_type AS type, updated_actor_id AS id").
		Where("id = ?", first.Id).
		Scan(&updatedActor).Error; err != nil {
		t.Fatal(err)
	}
	if updatedActor.Type != "user" || updatedActor.ID != 11 {
		t.Fatalf("updated actor = %q/%d", updatedActor.Type, updatedActor.ID)
	}
	var archivedDate sql.NullTime
	if err := db.Table("organization_credentials").Select("archived_date").Where("id = ?", first.Id).Scan(&archivedDate).Error; err != nil {
		t.Fatal(err)
	}
	if !archivedDate.Valid {
		t.Fatal("archived_date is nil")
	}
	if _, err := authenticator.Claim(context.Background(), rotatedKey); err == nil {
		t.Fatal("archived credential authenticated")
	}
}
