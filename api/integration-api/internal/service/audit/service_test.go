package internal_audit_service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internal_gorm "github.com/rapidaai/api/integration-api/internal/entity"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
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
	require.NoError(t, gorm_models.RegisterAuditActorCallbacks(database))
	applicationLogger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	return NewAuditService(applicationLogger, &testPostgresConnector{db: database}).(*auditService), database
}

func auditContext(actorType types.AuthType, actorID uint64) context.Context {
	authentication := &types.Authentication{
		AuthType:   actorType,
		ActorValue: &types.ActorIdentity{Type: types.ActorType(actorType), ID: actorID},
	}
	if actorType != types.AuthTypeSystem {
		authentication.OrganizationValue = &types.OrganizationContext{OrganizationID: 7}
	}
	return context.WithValue(context.Background(), types.CTX_, authentication)
}

func TestCreatePersistsAndUpdatesActorIdentity(t *testing.T) {
	service, database := newAuditServiceTest(t)

	created, err := service.Create(auditContext(types.AuthTypeService, 41), 100, 7, 8, 9, "test", "prefix", nil, type_enums.RECORD_ACTIVE)
	require.NoError(t, err)
	require.Equal(t, uint64(41), *created.CreatedActorID)

	_, err = service.Create(auditContext(types.AuthTypeSystem, 42), 100, 7, 8, 9, "test", "prefix", nil, type_enums.RECORD_COMPLETE)
	require.NoError(t, err)

	var stored internal_gorm.ExternalAudit
	require.NoError(t, database.First(&stored, "id = ?", 100).Error)
	require.Equal(t, "service", *stored.CreatedActorType)
	require.Equal(t, uint64(41), *stored.CreatedActorID)
	require.Equal(t, "system", *stored.UpdatedActorType)
	require.Equal(t, uint64(42), *stored.UpdatedActorID)
}

func TestCreateFailsWithoutActorBeforePersistence(t *testing.T) {
	service, database := newAuditServiceTest(t)

	_, err := service.Create(context.Background(), 100, 7, 8, 9, "test", "prefix", nil, type_enums.RECORD_ACTIVE)
	require.ErrorIs(t, err, types.ErrActorUnavailable)

	var count int64
	require.NoError(t, database.Model(&internal_gorm.ExternalAudit{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateMetadataPreservesCreationActorOnConflict(t *testing.T) {
	service, database := newAuditServiceTest(t)

	_, err := service.CreateMetadata(auditContext(types.AuthTypeService, 41), 100, map[string]string{"source": "first"})
	require.NoError(t, err)
	_, err = service.UpdateMetadata(auditContext(types.AuthTypeSystem, 42), 100, map[string]string{"source": "second"})
	require.NoError(t, err)

	var stored internal_gorm.ExternalAuditMetadata
	require.NoError(t, database.First(&stored, "external_audit_id = ? AND key = ?", 100, "source").Error)
	require.Equal(t, "second", stored.Value)
	require.Equal(t, "service", *stored.CreatedActorType)
	require.Equal(t, uint64(41), *stored.CreatedActorID)
	require.Equal(t, "system", *stored.UpdatedActorType)
	require.Equal(t, uint64(42), *stored.UpdatedActorID)
}
