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

func (failingProductUsageService) CreateProductUsages(context.Context, *types.Authentication, []*protos.ProductUsage) (internal_productusage_service.Result, error) {
	return internal_productusage_service.Result{}, errors.New("database unavailable")
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
		id integer primary key, organization_id integer not null, project_id integer not null,
		usage_id text not null, usage_type text not null, usages integer not null, unit text not null,
		occurred_at datetime not null, created_date datetime not null, updated_date datetime,
		status text not null, created_actor_type text not null, created_actor_id integer not null,
		updated_actor_type text, updated_actor_id integer,
		unique (organization_id, project_id, usage_id)
	)`).Error)
	require.NoError(t, database.Exec(`INSERT INTO projects (id, organization_id, status) VALUES (100, 10, 'ACTIVE')`).Error)
	postgres := &productUsageAPITestPostgres{db: database}
	return &webProductUsageGRPCApi{
		logger:              applicationLogger,
		productUsageService: internal_productusage_service.NewProductUsageService(postgres),
	}, database
}

func productUsageAPIContext(authType types.AuthType, includeProject bool) context.Context {
	auth := &types.Authentication{
		AuthType: authType,
		ActorValue: &types.ActorIdentity{
			Type: types.ActorType(authType),
			ID:   1,
		},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 10},
	}
	if authType == types.AuthTypeUser {
		auth.UserValue = &types.UserContext{UserID: 1}
	}
	if includeProject {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: 10, ProjectID: 100}
	}
	return context.WithValue(context.Background(), types.CTX_, auth)
}

func productUsageAPIRequest(usageID string, quantity int64) *protos.CreateProductUsagesRequest {
	return &protos.CreateProductUsagesRequest{Usages: []*protos.ProductUsage{{
		UsageId:    usageID,
		UsageType:  string(types.ProductUsageSTTDuration),
		Usages:     quantity,
		Unit:       string(types.ProductUsageUnitNanosecond),
		OccurredAt: timestamppb.New(time.Date(2026, time.August, 29, 5, 0, 0, 123456789, time.UTC)),
	}}}
}

func TestCreateProductUsagesAcceptsCurrentProjectContexts(t *testing.T) {
	for index, authType := range []types.AuthType{types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService} {
		t.Run(authType.String(), func(t *testing.T) {
			api, database := newProductUsageAPITest(t)
			request := productUsageAPIRequest([]string{
				"550e8400-e29b-41d4-a716-446655440010",
				"550e8400-e29b-41d4-a716-446655440011",
				"550e8400-e29b-41d4-a716-446655440012",
			}[index], 100)

			response, err := api.CreateProductUsages(productUsageAPIContext(authType, true), request)
			require.NoError(t, err)
			require.True(t, response.GetSuccess())
			require.Equal(t, uint32(1), response.GetCreatedCount())

			var usage internal_entity.ProductUsage
			require.NoError(t, database.Take(&usage).Error)
			require.Equal(t, uint64(10), usage.OrganizationId)
			require.Equal(t, uint64(100), usage.ProjectId)
			require.Equal(t, authType.String(), usage.CreatedActorType)
		})
	}
}

func TestCreateProductUsagesRejectsMissingAuthAndProject(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	request := productUsageAPIRequest("550e8400-e29b-41d4-a716-446655440013", 1)

	_, err := api.CreateProductUsages(context.Background(), request)
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	_, err = api.CreateProductUsages(productUsageAPIContext(types.AuthTypeUser, false), request)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreateProductUsagesMapsValidationAndConflictErrors(t *testing.T) {
	api, database := newProductUsageAPITest(t)
	ctx := productUsageAPIContext(types.AuthTypeProject, true)
	usageID := "550e8400-e29b-41d4-a716-446655440014"

	_, err := api.CreateProductUsages(ctx, productUsageAPIRequest(usageID, 0))
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	response, err := api.CreateProductUsages(ctx, productUsageAPIRequest(usageID, 10))
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.GetCreatedCount())

	response, err = api.CreateProductUsages(ctx, productUsageAPIRequest(usageID, 10))
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.GetDuplicateCount())

	_, err = api.CreateProductUsages(ctx, productUsageAPIRequest(usageID, 11))
	require.Equal(t, codes.AlreadyExists, status.Code(err))

	var count int64
	require.NoError(t, database.Table("product_usages").Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestCreateProductUsagesMapsPersistenceErrors(t *testing.T) {
	api, _ := newProductUsageAPITest(t)
	api.productUsageService = failingProductUsageService{}

	_, err := api.CreateProductUsages(
		productUsageAPIContext(types.AuthTypeUser, true),
		productUsageAPIRequest("550e8400-e29b-41d4-a716-446655440015", 1),
	)
	require.Equal(t, codes.Internal, status.Code(err))
}
