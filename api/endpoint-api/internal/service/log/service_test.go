package internal_log_service

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
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

func testAuth(t *testing.T, userID, orgID, projectID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: orgID, ProjectID: projectID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: "test-session",
			},
		},
		nil,
	)
	require.NoError(t, err)
	return principal
}

func testOrganizationAuth(t *testing.T, organizationID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: 1},
		authentication.Tenant{OrganizationID: organizationID},
		authentication.Proof{
			Method: authentication.AuthenticationOrganizationCredential,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceCredential,
				Value: "1",
			},
		},
		nil,
	)
	require.NoError(t, err)
	return principal
}

func testProjectAuth(t *testing.T, credentialID, organizationID, projectID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: credentialID},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method:   authentication.AuthenticationProjectCredential,
			Audience: authentication.AudienceManagement,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceCredential,
				Value: strconv.FormatUint(credentialID, 10),
			},
		},
		nil,
	)
	require.NoError(t, err)
	return principal
}

func TestEndpointLogServiceRequiresProjectCapability(t *testing.T) {
	service, _ := newEndpointLogServiceTest(t)
	organizationID := uint64(10)

	_, err := service.GetEndpointLog(
		context.Background(),
		testOrganizationAuth(t, organizationID),
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
	auth := testProjectAuth(t, projectID, organizationID, projectID)

	metadata, err := service.ApplyMetadata(context.Background(), auth, 1, map[string]interface{}{"trace_id": "trace-1"})
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.Equal(t, string(authentication.IdentityProjectCredential), metadata[0].CreatedActorType)
	require.Equal(t, projectID, metadata[0].CreatedActorID)
	require.Empty(t, metadata[0].UpdatedActorType)
	require.Zero(t, metadata[0].UpdatedActorID)
}

func TestGetEndpointLogPreloadsMetricsAndContext(t *testing.T) {
	service, db := newEndpointLogServiceTest(t)
	auth := testAuth(t, 1, 10, 20)

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

	_, err := service.GetEndpointLog(context.Background(), testAuth(t, 1, 10, 21), 101, 201)
	require.Error(t, err)
}
