package internal_endpoint_service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rapidaai/api/endpoint-api/config"
	internal_gorm "github.com/rapidaai/api/endpoint-api/internal/entity"
	internal_service "github.com/rapidaai/api/endpoint-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
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

func newEndpointServiceTest(t *testing.T) (*endpointService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, stmt := range []string{
		`CREATE TABLE endpoints (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			project_id integer,
			organization_id integer,
			name text,
			description text,
			endpoint_provider_model_id integer,
			visibility text,
			source text,
			source_identifier integer,
			cache_enable boolean,
			retry_enable boolean
		)`,
		`CREATE TABLE endpoint_provider_models (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			endpoint_id integer,
			description text,
			request text,
			model_provider_name text
		)`,
		`CREATE TABLE endpoint_provider_model_options (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			key text,
			value text,
			endpoint_provider_model_id integer
		)`,
		`CREATE TABLE endpoint_retries (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			endpoint_id integer,
			retry_type text,
			max_attempts integer,
			delay_seconds integer,
			exponential_backoff boolean,
			retryables text
		)`,
		`CREATE TABLE endpoint_cachings (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			endpoint_id integer,
			cache_type text,
			expiry_interval integer,
			match_threshold real
		)`,
		`CREATE TABLE endpoint_tags (
			id integer primary key,
			created_date datetime,
			updated_date datetime,
			status text,
			created_by integer,
			updated_by integer,
			endpoint_id integer,
			tag text
		)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX idx_endpoint_cachings_endpoint_id ON endpoint_cachings(endpoint_id)").Error)
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX idx_endpoint_retries_endpoint_id_retry_type ON endpoint_retries(endpoint_id, retry_type)").Error)
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX idx_endpoint_tags_endpoint_id ON endpoint_tags(endpoint_id)").Error)

	testLogger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return &endpointService{
		cfg:      &config.EndpointConfig{},
		logger:   testLogger,
		postgres: &testPostgresConnector{db: db},
	}, db
}

func testAuth(userID, orgID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: userID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: orgID},
		ProjectValue:      &types.ProjectContext{OrganizationID: orgID, ProjectID: projectID},
	}
}

func TestEndpointServiceRequiresCapabilities(t *testing.T) {
	service, _ := newEndpointServiceTest(t)
	organizationID := uint64(10)
	projectID := uint64(20)
	projectlessUser := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: 1},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}

	_, err := service.Get(
		context.Background(),
		projectlessUser,
		1,
		nil,
		internal_service.NewDefaultGetEndpointOption(),
	)
	require.Error(t, err)

	_, _, err = service.GetAll(
		context.Background(),
		&types.Authentication{AuthType: types.AuthTypeOrg, OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID}},
		nil,
		&protos.Paginate{},
	)
	require.Error(t, err)

	_, err = service.CreateEndpoint(
		context.Background(),
		&types.Authentication{AuthType: types.AuthTypeProject, OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID}, ProjectValue: &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}},
		"endpoint",
		nil,
		nil,
		nil,
		nil,
	)
	require.Error(t, err)
}

func TestEndpointServiceInvokeLookupAllowsOrganizationForPublicEndpoint(t *testing.T) {
	service, db := newEndpointServiceTest(t)
	organizationID := uint64(10)
	endpoint := insertEndpoint(t, db, 104, organizationID, 20, "public")
	providerModel := &internal_gorm.EndpointProviderModel{
		Audited:    gorm_models.Audited{Id: 204},
		EndpointId: endpoint.Id,
	}
	require.NoError(t, db.Create(providerModel).Error)
	require.NoError(t, db.Model(endpoint).Update("endpoint_provider_model_id", providerModel.Id).Error)

	result, err := service.Get(
		context.Background(),
		&types.Authentication{AuthType: types.AuthTypeOrg, OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID}},
		endpoint.Id,
		nil,
		internal_service.NewInvokeGetEndpointOption(),
	)
	require.NoError(t, err)
	require.Equal(t, endpoint.Id, result.Id)
}

func TestEndpointServiceInvokeLookupRejectsOrganizationForPrivateEndpoint(t *testing.T) {
	service, db := newEndpointServiceTest(t)
	organizationID := uint64(10)
	endpoint := insertEndpoint(t, db, 105, organizationID, 20, "private")
	providerModel := &internal_gorm.EndpointProviderModel{
		Audited:    gorm_models.Audited{Id: 205},
		EndpointId: endpoint.Id,
	}
	require.NoError(t, db.Create(providerModel).Error)
	require.NoError(t, db.Model(endpoint).Update("endpoint_provider_model_id", providerModel.Id).Error)

	_, err := service.Get(
		context.Background(),
		&types.Authentication{AuthType: types.AuthTypeOrg, OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID}},
		endpoint.Id,
		nil,
		internal_service.NewInvokeGetEndpointOption(),
	)
	require.Error(t, err)
}

func insertEndpoint(t *testing.T, db *gorm.DB, id, orgID, projectID uint64, visibility string) *internal_gorm.Endpoint {
	t.Helper()
	endpoint := &internal_gorm.Endpoint{
		Audited: gorm_models.Audited{Id: id},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Organizational: gorm_models.Organizational{
			OrganizationId: orgID,
			ProjectId:      projectID,
		},
		Name:       "endpoint",
		Visibility: &visibility,
	}
	require.NoError(t, db.Create(endpoint).Error)
	return endpoint
}

func requireAccessDenied(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "access to the endpoint"), "unexpected error: %v", err)
}

func TestGetAllEndpointProviderModelEnforcesEndpointOwnership(t *testing.T) {
	service, db := newEndpointServiceTest(t)
	endpoint := insertEndpoint(t, db, 101, 10, 20, "private")
	otherProjectAuth := testAuth(2, 11, 21)
	ownerAuth := testAuth(1, endpoint.OrganizationId, endpoint.ProjectId)
	require.NoError(t, db.Create(&internal_gorm.EndpointProviderModel{
		Audited: gorm_models.Audited{Id: 201},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		EndpointId:                   endpoint.Id,
		Description:                  "primary",
		ModelProviderName:            "openai",
		EndpointProviderModelOptions: []*internal_gorm.EndpointProviderModelOption{},
	}).Error)

	count, models, err := service.GetAllEndpointProviderModel(context.Background(), ownerAuth, endpoint.Id, nil, &protos.Paginate{})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, models, 1)

	count, models, err = service.GetAllEndpointProviderModel(context.Background(), otherProjectAuth, endpoint.Id, nil, &protos.Paginate{})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Len(t, models, 0)
}

func TestEndpointSubresourceWritesEnforceEndpointOwnership(t *testing.T) {
	service, db := newEndpointServiceTest(t)
	endpoint := insertEndpoint(t, db, 102, 10, 20, "private")
	otherProjectAuth := testAuth(2, 11, 21)
	ownerAuth := testAuth(1, endpoint.OrganizationId, endpoint.ProjectId)

	_, err := service.ConfigureEndpointCaching(context.Background(), ownerAuth, endpoint.Id, internal_gorm.STANDARD_CACHE, 3600, 0.8)
	require.NoError(t, err)

	_, err = service.ConfigureEndpointCaching(context.Background(), otherProjectAuth, endpoint.Id, internal_gorm.SEMENTIC_CACHE, 7200, 0.7)
	requireAccessDenied(t, err)
	var cache internal_gorm.EndpointCaching
	require.NoError(t, db.Where("endpoint_id = ?", endpoint.Id).First(&cache).Error)
	require.Equal(t, internal_gorm.STANDARD_CACHE, cache.CacheType)

	_, err = service.ConfigureEndpointRetry(context.Background(), ownerAuth, endpoint.Id, internal_gorm.STATUS_RETRY, 3, 2, true, []string{"500"})
	require.NoError(t, err)

	_, err = service.ConfigureEndpointRetry(context.Background(), otherProjectAuth, endpoint.Id, internal_gorm.STATUS_RETRY, 9, 9, false, []string{"400"})
	requireAccessDenied(t, err)
	var retry internal_gorm.EndpointRetry
	require.NoError(t, db.Where("endpoint_id = ? AND retry_type = ?", endpoint.Id, internal_gorm.STATUS_RETRY).First(&retry).Error)
	require.EqualValues(t, 3, retry.MaxAttempts)

	_, err = service.CreateOrUpdateEndpointTag(context.Background(), ownerAuth, endpoint.Id, []string{"owner"})
	require.NoError(t, err)

	_, err = service.CreateOrUpdateEndpointTag(context.Background(), otherProjectAuth, endpoint.Id, []string{"other"})
	require.Error(t, err)
	var tag internal_gorm.EndpointTag
	require.NoError(t, db.Where("endpoint_id = ?", endpoint.Id).First(&tag).Error)
	require.Equal(t, []string{"owner"}, []string(tag.Tag))
}

func TestCreateEndpointProviderModelEnforcesOwnershipForPublicEndpoint(t *testing.T) {
	service, db := newEndpointServiceTest(t)
	endpoint := insertEndpoint(t, db, 103, 10, 20, "public")
	otherProjectAuth := testAuth(2, 11, 21)
	ownerAuth := testAuth(1, endpoint.OrganizationId, endpoint.ProjectId)

	_, err := service.CreateEndpointProviderModel(
		context.Background(),
		ownerAuth,
		endpoint.Id,
		"owner model",
		"openai",
		`{"messages":[]}`,
		nil,
	)
	require.NoError(t, err)

	_, err = service.CreateEndpointProviderModel(
		context.Background(),
		otherProjectAuth,
		endpoint.Id,
		"other model",
		"openai",
		`{"messages":[]}`,
		nil,
	)
	require.Error(t, err)

	var count int64
	require.NoError(t, db.Model(&internal_gorm.EndpointProviderModel{}).Where("endpoint_id = ?", endpoint.Id).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
