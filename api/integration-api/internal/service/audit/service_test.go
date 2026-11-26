package internal_audit_service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internal_gorm "github.com/rapidaai/api/integration-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (connector *testPostgresConnector) Connect(context.Context) error { return nil }
func (connector *testPostgresConnector) Name() string                  { return "test-postgres" }
func (connector *testPostgresConnector) IsConnected(context.Context) bool {
	return connector.db != nil
}
func (connector *testPostgresConnector) Disconnect(context.Context) error { return nil }
func (connector *testPostgresConnector) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *testPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

func newAuditServiceTest(t *testing.T) (*auditService, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE external_audits (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer,
			integration_name text,
			asset_prefix text,
			response_status integer,
			time_taken integer,
			status text,
			project_id integer,
			organization_id integer,
			credential_id integer,
			metrics text
		)`,
		`CREATE TABLE external_audit_metadata (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer,
			external_audit_id integer,
			key text,
			value text,
			UNIQUE (external_audit_id, key)
		)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}
	applicationLogger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return NewAuditService(applicationLogger, &testPostgresConnector{db: database}).(*auditService), database
}

func auditContext(t *testing.T, actorID uint64) context.Context {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: actorID, Issuer: "integration-test"},
		authentication.Tenant{},
		authentication.Proof{
			Method:   authentication.AuthenticationServiceAssertion,
			Audience: authentication.AudienceRapidaInternal,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceSigningKey,
				Value: "test-key",
			},
		},
		nil,
	)
	require.NoError(t, err)
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	require.NoError(t, err)
	return ctx
}

func TestCreatePersistsAndUpdatesActorIdentity(t *testing.T) {
	service, database := newAuditServiceTest(t)

	created, err := service.Create(auditContext(t, 41), 100, 7, 8, 9, "test", "prefix", nil, type_enums.RECORD_ACTIVE)
	require.NoError(t, err)
	require.Equal(t, uint64(41), created.CreatedActorID)

	_, err = service.Create(auditContext(t, 42), 100, 7, 8, 9, "test", "prefix", nil, type_enums.RECORD_COMPLETE)
	require.NoError(t, err)

	var stored internal_gorm.ExternalAudit
	require.NoError(t, database.First(&stored, "id = ?", 100).Error)
	require.Equal(t, "service", stored.CreatedActorType)
	require.Equal(t, uint64(41), stored.CreatedActorID)
	require.Equal(t, "service", stored.UpdatedActorType)
	require.Equal(t, uint64(42), stored.UpdatedActorID)
}

func TestCreateMetadataPreservesCreationActorOnConflict(t *testing.T) {
	service, database := newAuditServiceTest(t)

	_, err := service.CreateMetadata(auditContext(t, 41), 100, map[string]string{"source": "first"})
	require.NoError(t, err)
	_, err = service.UpdateMetadata(auditContext(t, 42), 100, map[string]string{"source": "second"})
	require.NoError(t, err)

	var stored internal_gorm.ExternalAuditMetadata
	require.NoError(t, database.First(&stored, "external_audit_id = ? AND key = ?", 100, "source").Error)
	require.Equal(t, "second", stored.Value)
	require.Equal(t, "service", stored.CreatedActorType)
	require.Equal(t, uint64(41), stored.CreatedActorID)
	require.Equal(t, "service", stored.UpdatedActorType)
	require.Equal(t, uint64(42), stored.UpdatedActorID)
}
