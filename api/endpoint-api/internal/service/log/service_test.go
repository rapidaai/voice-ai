package internal_log_service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Name() string {
	return "test-postgres"
}

func (t *testPostgresConnector) IsConnected(ctx context.Context) bool {
	return true
}

func (t *testPostgresConnector) Disconnect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.DB(ctx).Raw(qry).Scan(dest).Error
}

func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if tx, ok := connectors.PostgresTxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return t.db.WithContext(ctx)
}

func newEndpointLogServiceTest(t *testing.T) (*endpointLogService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, stmt := range []string{
		`CREATE TABLE endpoint_logs (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			project_id integer,
			organization_id integer,
			endpoint_id integer,
			endpoint_provider_model_id integer,
			source text,
			status text,
			time_taken integer
		)`,
		`CREATE TABLE endpoint_log_arguments (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer,
			name text,
			value text,
			endpoint_log_id integer
		)`,
		`CREATE TABLE endpoint_log_metadata (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer,
			key text,
			value text,
			endpoint_log_id integer
		)`,
		`CREATE TABLE endpoint_log_metrics (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_actor_type text,
			created_actor_id integer,
			updated_actor_type text,
			updated_actor_id integer,
			name text,
			value text,
			description text,
			endpoint_log_id integer
		)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}

	testLogger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
	)
	require.NoError(t, err)

	return &endpointLogService{
		logger:   testLogger,
		postgres: &testPostgresConnector{db: db},
	}, db
}

func testAuth(userID, orgID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: userID},
		UserValue:         &types.UserContext{UserID: userID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: orgID},
		ProjectValue:      &types.ProjectContext{OrganizationID: orgID, ProjectID: projectID},
	}
}

func TestEndpointLogServiceRequiresProjectCapability(t *testing.T) {
	service, _ := newEndpointLogServiceTest(t)
	organizationID := uint64(10)

	_, err := service.GetEndpointLog(
		context.Background(),
		&types.Authentication{AuthType: types.AuthTypeOrg, OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID}},
		1,
		2,
	)
	require.Error(t, err)
}

func TestApplyMetadataAllowsProjectAuthWithoutFakeUser(t *testing.T) {
	service, db := newEndpointLogServiceTest(t)
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX idx_endpoint_log_metadata_log_key ON endpoint_log_metadata(endpoint_log_id, key)").Error)
	organizationID := uint64(10)
	projectID := uint64(20)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: projectID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}

	metadata, err := service.ApplyMetadata(context.Background(), auth, 1, map[string]interface{}{"trace_id": "trace-1"})
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.Equal(t, types.AuthTypeProject.String(), metadata[0].CreatedActorType)
	require.Equal(t, projectID, metadata[0].CreatedActorID)
	require.Empty(t, metadata[0].UpdatedActorType)
	require.Zero(t, metadata[0].UpdatedActorID)
}

func TestGetEndpointLogPreloadsMetricsAndContext(t *testing.T) {
	service, db := newEndpointLogServiceTest(t)
	auth := testAuth(1, 10, 20)

	require.NoError(t, db.Create(&internal_gorm.EndpointLog{
		Audited: gorm_models.Audited{Id: 100},
		Organizational: gorm_models.Organizational{
			OrganizationId: 10,
			ProjectId:      20,
		},
		EndpointId:              200,
		EndpointProviderModelId: 300,
		Source:                  "web-app",
		Status:                  type_enums.RECORD_COMPLETE,
	}).Error)
	require.NoError(t, db.Create(&internal_gorm.EndpointLogMetric{
		Audited: gorm_models.Audited{Id: 101},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Metric: gorm_models.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       "42",
			Description: "total token count",
		},
		EndpointLogId: 100,
	}).Error)
	require.NoError(t, db.Create(&internal_gorm.EndpointLogMetadata{
		Audited: gorm_models.Audited{Id: 102},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Metadata:      *gorm_models.NewMetadata("trace_id", "trace-1"),
		EndpointLogId: 100,
	}).Error)
	require.NoError(t, db.Create(&internal_gorm.EndpointLogArgument{
		Audited: gorm_models.Audited{Id: 103},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Argument: gorm_models.Argument{
			Name:  "prompt",
			Value: "hello",
		},
		EndpointLogId: 100,
	}).Error)

	log, err := service.GetEndpointLog(context.Background(), auth, 100, 200)
	require.NoError(t, err)
	require.EqualValues(t, 100, log.Id)
	require.Len(t, log.Metrics, 1)
	require.Equal(t, type_enums.TOTAL_TOKEN.String(), log.Metrics[0].Name)
	require.Equal(t, "42", log.Metrics[0].Value)
	require.Len(t, log.Metadata, 1)
	require.Equal(t, "trace_id", log.Metadata[0].Key)
	require.Len(t, log.Arguments, 1)
	require.Equal(t, "prompt", log.Arguments[0].Name)
}

func TestGetEndpointLogEnforcesProjectOwnership(t *testing.T) {
	service, db := newEndpointLogServiceTest(t)

	require.NoError(t, db.Create(&internal_gorm.EndpointLog{
		Audited: gorm_models.Audited{Id: 101},
		Organizational: gorm_models.Organizational{
			OrganizationId: 10,
			ProjectId:      20,
		},
		EndpointId:              201,
		EndpointProviderModelId: 301,
		Source:                  "web-app",
		Status:                  type_enums.RECORD_COMPLETE,
	}).Error)

	_, err := service.GetEndpointLog(context.Background(), testAuth(1, 10, 21), 101, 201)
	require.Error(t, err)
}
