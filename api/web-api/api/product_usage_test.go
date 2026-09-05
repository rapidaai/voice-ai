package web_api

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type failingProductUsageService struct{}

func (failingProductUsageService) CreateProductUsage(context.Context, *types.Authentication, *protos.CreateProductUsageRequest) (*internal_entity.ProductUsage, error) {
	return nil, errors.New("database unavailable")
}

func (failingProductUsageService) GetProductUsages(context.Context, *types.Authentication, string, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProductUsage, error) {
	return 0, nil, errors.New("database unavailable")
}

func (failingProductUsageService) GetOrganizationUsages(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProductUsage, error) {
	return 0, nil, errors.New("database unavailable")
}

type productUsageAPITestPostgres struct {
	db *gorm.DB
}

func (connector *productUsageAPITestPostgres) Connect(context.Context) error { return nil }
func (connector *productUsageAPITestPostgres) Name() string                  { return "product-usage-api-test" }
func (connector *productUsageAPITestPostgres) IsConnected(context.Context) bool {
	return true
}
func (connector *productUsageAPITestPostgres) Disconnect(context.Context) error { return nil }
func (connector *productUsageAPITestPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *productUsageAPITestPostgres) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

func newProductUsageAPITest(t *testing.T) (*webProductUsageGRPCApi, *gorm.DB) {
	t.Helper()
	applicationLogger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE projects (id integer primary key, organization_id integer not null, status text not null)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE product_usages (
		id integer primary key autoincrement, organization_id integer not null, project_id integer not null,
		usage_type text not null, usages integer not null, unit text not null,
		occurred_at datetime not null, created_date datetime default current_timestamp not null, updated_date datetime,
		status text default 'ACTIVE' not null, created_actor_type text not null, created_actor_id integer not null,
		updated_actor_type text, updated_actor_id integer
	)`).Error)
	require.NoError(t, database.Exec(`INSERT INTO projects (id, organization_id, status) VALUES
		(100, 10, 'ACTIVE'), (101, 10, 'ACTIVE'), (200, 20, 'ACTIVE')`).Error)
	postgres := &productUsageAPITestPostgres{db: database}
	return &webProductUsageGRPCApi{
		logger:              applicationLogger,
		productUsageService: internal_productusage_service.NewProductUsageService(postgres),
	}, database
}

func productUsageAPIContext(authType types.AuthType, organizationID, projectID uint64) context.Context {
	auth := &types.Authentication{
		AuthType: authType,
		ActorValue: &types.ActorIdentity{
			Type: types.ActorType(authType),
			ID:   1,
		},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}
	if authType == types.AuthTypeUser {
		auth.UserValue = &types.UserContext{UserID: 1}
	}
	if projectID > 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}
	}
	return context.WithValue(context.Background(), types.CTX_, auth)
}

func productUsageAPIRequest(usageType string, quantity int64, occurredAt time.Time) *protos.CreateProductUsageRequest {
	return &protos.CreateProductUsageRequest{
		UsageType:  usageType,
		Usages:     quantity,
		Unit:       string(types.ProductUsageUnitNanosecond),
		OccurredAt: timestamppb.New(occurredAt),
	}
}

func TestCreateProductUsageReturnsPersistedUsage(t *testing.T) {
	for _, authType := range []types.AuthType{types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService} {
		t.Run(authType.String(), func(t *testing.T) {
			api, database := newProductUsageAPITest(t)
			occurredAt := time.Date(2026, time.August, 29, 5, 0, 0, 123456789, time.UTC)

			response, err := api.CreateProductUsage(
				productUsageAPIContext(authType, 10, 100),
				productUsageAPIRequest(string(types.ProductUsageSTTDuration), 100, occurredAt),
			)
			require.NoError(t, err)
			require.True(t, response.GetSuccess())
			require.NotZero(t, response.GetData().GetId())
			require.Equal(t, uint64(100), response.GetData().GetProjectId())
			require.Equal(t, occurredAt.Truncate(time.Microsecond), response.GetData().GetOccurredAt().AsTime())

			var usage internal_entity.ProductUsage
			require.NoError(t, database.Take(&usage).Error)
			require.Equal(t, response.GetData().GetId(), usage.Id)
			require.Equal(t, authType.String(), usage.CreatedActorType)
		})
	}
}

func TestCreateProductUsageRejectsMissingAuthProjectAndInvalidInput(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	request := productUsageAPIRequest(string(types.ProductUsageSTTDuration), 1, time.Now())

	_, err := api.CreateProductUsage(context.Background(), request)
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	_, err = api.CreateProductUsage(productUsageAPIContext(types.AuthTypeUser, 10, 0), request)
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	request.Usages = 0
	_, err = api.CreateProductUsage(productUsageAPIContext(types.AuthTypeUser, 10, 100), request)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetProductUsagesReturnsOnlyAuthenticatedProjectType(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	base := time.Date(2026, time.August, 29, 5, 0, 0, 0, time.UTC)
	projectContext := productUsageAPIContext(types.AuthTypeUser, 10, 100)

	first, err := api.CreateProductUsage(projectContext, productUsageAPIRequest(string(types.ProductUsageSTTDuration), 10, base))
	require.NoError(t, err)
	second, err := api.CreateProductUsage(projectContext, productUsageAPIRequest(string(types.ProductUsageSTTDuration), 20, base.Add(time.Minute)))
	require.NoError(t, err)
	_, err = api.CreateProductUsage(projectContext, productUsageAPIRequest(string(types.ProductUsageLLMDuration), 30, base.Add(2*time.Minute)))
	require.NoError(t, err)
	_, err = api.CreateProductUsage(productUsageAPIContext(types.AuthTypeProject, 10, 101), productUsageAPIRequest(string(types.ProductUsageSTTDuration), 40, base.Add(3*time.Minute)))
	require.NoError(t, err)

	response, err := api.GetProductUsages(projectContext, &protos.GetProductUsagesRequest{
		UsageType: string(types.ProductUsageSTTDuration),
		Criterias: []*protos.Criteria{{Key: "usages", Logic: ">=", Value: "10"}},
		Paginate:  &protos.Paginate{Page: 2, PageSize: 1},
	})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, uint32(2), response.GetPaginated().GetTotalItem())
	require.Equal(t, uint32(2), response.GetPaginated().GetCurrentPage())
	require.Len(t, response.GetData(), 1)
	require.Equal(t, first.GetData().GetId(), response.GetData()[0].GetId())
	require.NotEqual(t, second.GetData().GetId(), response.GetData()[0].GetId())

	_, err = api.GetProductUsages(projectContext, &protos.GetProductUsagesRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetOrganizationUsagesReturnsProjectsForAuthenticatedOrganization(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	base := time.Date(2026, time.August, 29, 5, 0, 0, 0, time.UTC)
	_, err := api.CreateProductUsage(productUsageAPIContext(types.AuthTypeUser, 10, 100), productUsageAPIRequest(string(types.ProductUsageSTTDuration), 10, base))
	require.NoError(t, err)
	wanted, err := api.CreateProductUsage(productUsageAPIContext(types.AuthTypeProject, 10, 101), productUsageAPIRequest(string(types.ProductUsageLLMDuration), 20, base.Add(time.Minute)))
	require.NoError(t, err)
	_, err = api.CreateProductUsage(productUsageAPIContext(types.AuthTypeProject, 20, 200), productUsageAPIRequest(string(types.ProductUsageLLMDuration), 30, base.Add(2*time.Minute)))
	require.NoError(t, err)

	response, err := api.GetOrganizationUsages(
		productUsageAPIContext(types.AuthTypeOrg, 10, 0),
		&protos.GetOrganizationUsagesRequest{
			Criterias: []*protos.Criteria{{Key: "usageType", Value: string(types.ProductUsageLLMDuration)}},
			Paginate:  &protos.Paginate{Page: 1, PageSize: 10},
		},
	)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.GetPaginated().GetTotalItem())
	require.Len(t, response.GetData(), 1)
	require.Equal(t, wanted.GetData().GetId(), response.GetData()[0].GetId())
	require.Equal(t, uint64(101), response.GetData()[0].GetProjectId())

	_, err = api.GetOrganizationUsages(productUsageAPIContext(types.AuthTypeProject, 10, 100), &protos.GetOrganizationUsagesRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestProductUsageMethodsMapPersistenceErrors(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	api.productUsageService = failingProductUsageService{}
	projectContext := productUsageAPIContext(types.AuthTypeUser, 10, 100)

	_, err := api.CreateProductUsage(projectContext, productUsageAPIRequest(string(types.ProductUsageSTTDuration), 1, time.Now()))
	require.Equal(t, codes.Internal, status.Code(err))
	_, err = api.GetProductUsages(projectContext, &protos.GetProductUsagesRequest{UsageType: string(types.ProductUsageSTTDuration)})
	require.Equal(t, codes.Internal, status.Code(err))
	_, err = api.GetOrganizationUsages(productUsageAPIContext(types.AuthTypeUser, 10, 0), &protos.GetOrganizationUsagesRequest{})
	require.Equal(t, codes.Internal, status.Code(err))
}
