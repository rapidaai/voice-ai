package internal_productusage_service

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/connectors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type productUsageTestPostgres struct {
	db *gorm.DB
}

func (connector *productUsageTestPostgres) Connect(context.Context) error { return nil }
func (connector *productUsageTestPostgres) Name() string                  { return "product-usage-test" }
func (connector *productUsageTestPostgres) IsConnected(context.Context) bool {
	return true
}
func (connector *productUsageTestPostgres) Disconnect(context.Context) error { return nil }
func (connector *productUsageTestPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *productUsageTestPostgres) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

func newProductUsageServiceTest(t *testing.T) (Service, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`CREATE TABLE projects (
		id integer primary key,
		organization_id integer not null,
		status text not null
	)`).Error)
	require.NoError(t, database.Exec(`CREATE TABLE product_usages (
		id integer primary key autoincrement,
		organization_id integer not null,
		project_id integer not null,
		usage_type text not null,
		usages integer not null,
		unit text not null,
		occurred_at datetime not null,
		created_date datetime default current_timestamp not null,
		updated_date datetime,
		status text default 'ACTIVE' not null,
		created_actor_type text not null,
		created_actor_id integer not null,
		updated_actor_type text,
		updated_actor_id integer
	)`).Error)
	require.NoError(t, database.Exec(`INSERT INTO projects (id, organization_id, status) VALUES
		(100, 10, 'ACTIVE'),
		(101, 10, 'ACTIVE'),
		(200, 20, 'ACTIVE'),
		(300, 10, 'ARCHIEVE')`).Error)
	return NewProductUsageService(&productUsageTestPostgres{db: database}), database
}

func productUsageAuth(authType types.AuthType, organizationID, projectID, actorID uint64) *types.Authentication {
	auth := &types.Authentication{
		AuthType: authType,
		ActorValue: &types.ActorIdentity{
			Type: types.ActorType(authType),
			ID:   actorID,
		},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}
	if projectID > 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}
	}
	if authType == types.AuthTypeUser {
		auth.UserValue = &types.UserContext{UserID: actorID}
	}
	return auth
}

func productUsageInput(quantity int64, occurredAt time.Time) *protos.CreateProductUsageRequest {
	return &protos.CreateProductUsageRequest{
		UsageType:  string(types.ProductUsageSTTDuration),
		Usages:     quantity,
		Unit:       string(types.ProductUsageUnitNanosecond),
		OccurredAt: timestamppb.New(occurredAt),
	}
}

func insertProductUsage(t *testing.T, database *gorm.DB, organizationID, projectID uint64, usageType string, usages int64, occurredAt time.Time) internal_entity.ProductUsage {
	t.Helper()
	usage := internal_entity.ProductUsage{
		Mutable: gorm_model.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: types.AuthTypeUser.String(),
			CreatedActorID:   1,
		},
		Organizational: gorm_model.Organizational{
			OrganizationId: organizationID,
			ProjectId:      projectID,
		},
		UsageType:  usageType,
		Usages:     usages,
		Unit:       string(types.ProductUsageUnitNanosecond),
		OccurredAt: occurredAt,
	}
	require.NoError(t, database.Create(&usage).Error)
	return usage
}

func TestCreateProductUsageGeneratesIDAndAllowsRepeatedEvents(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeUser, 10, 100, 1)
	request := productUsageInput(25, time.Date(2026, time.August, 29, 10, 30, 0, 123456789, time.FixedZone("test", 5*60*60+30*60)))

	first, err := service.CreateProductUsage(context.Background(), auth, request)
	require.NoError(t, err)
	second, err := service.CreateProductUsage(context.Background(), auth, request)
	require.NoError(t, err)

	require.NotZero(t, first.Id)
	require.NotZero(t, second.Id)
	require.NotEqual(t, first.Id, second.Id)
	require.Equal(t, uint64(10), first.OrganizationId)
	require.Equal(t, uint64(100), first.ProjectId)
	require.Equal(t, int64(25), first.Usages)
	require.Equal(t, types.AuthTypeUser.String(), first.CreatedActorType)
	require.Equal(t, uint64(1), first.CreatedActorID)
	require.Equal(t, request.GetOccurredAt().AsTime().UTC().Truncate(time.Microsecond), first.OccurredAt)

	var count int64
	require.NoError(t, database.Table("product_usages").Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestCreateProductUsageValidatesInputAndProjectOwnership(t *testing.T) {
	service, _ := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeService, 10, 100, 601)
	occurredAt := time.Date(2026, time.August, 29, 5, 0, 0, 0, time.UTC)

	_, err := service.CreateProductUsage(context.Background(), auth, productUsageInput(0, occurredAt))
	require.ErrorIs(t, err, ErrInvalidRequest)

	invalidUnit := productUsageInput(1, occurredAt)
	invalidUnit.Unit = "second"
	_, err = service.CreateProductUsage(context.Background(), auth, invalidUnit)
	require.ErrorIs(t, err, ErrInvalidRequest)

	_, err = service.CreateProductUsage(context.Background(), productUsageAuth(types.AuthTypeUser, 20, 100, 1), productUsageInput(1, occurredAt))
	require.ErrorIs(t, err, ErrProjectMismatch)

	_, err = service.CreateProductUsage(context.Background(), auth, nil)
	require.True(t, errors.Is(err, ErrInvalidRequest))
}

func TestGetProductUsagesScopesTypeTenantCriteriaAndPagination(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	base := time.Date(2026, time.August, 29, 5, 0, 0, 0, time.UTC)
	first := insertProductUsage(t, database, 10, 100, string(types.ProductUsageSTTDuration), 10, base)
	second := insertProductUsage(t, database, 10, 100, string(types.ProductUsageSTTDuration), 20, base.Add(time.Minute))
	insertProductUsage(t, database, 10, 100, string(types.ProductUsageLLMDuration), 30, base.Add(2*time.Minute))
	insertProductUsage(t, database, 10, 101, string(types.ProductUsageSTTDuration), 40, base.Add(3*time.Minute))
	insertProductUsage(t, database, 20, 200, string(types.ProductUsageSTTDuration), 50, base.Add(4*time.Minute))

	count, usages, err := service.GetProductUsages(
		context.Background(),
		productUsageAuth(types.AuthTypeUser, 10, 100, 1),
		string(types.ProductUsageSTTDuration),
		[]*protos.Criteria{{Key: "usages", Logic: ">=", Value: "10"}},
		&protos.Paginate{Page: 2, PageSize: 1},
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Len(t, usages, 1)
	require.Equal(t, first.Id, usages[0].Id)
	require.NotEqual(t, second.Id, usages[0].Id)
}

func TestGetProductUsagesRequiresSupportedUsageType(t *testing.T) {
	service, _ := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeProject, 10, 100, 501)

	_, _, err := service.GetProductUsages(context.Background(), auth, "", nil, nil)
	require.ErrorIs(t, err, ErrInvalidRequest)
	_, _, err = service.GetProductUsages(context.Background(), auth, "unsupported", nil, nil)
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestGetOrganizationUsagesSpansProjectsWithoutCrossingTenant(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	base := time.Date(2026, time.August, 29, 5, 0, 0, 0, time.UTC)
	insertProductUsage(t, database, 10, 100, string(types.ProductUsageSTTDuration), 10, base)
	wanted := insertProductUsage(t, database, 10, 101, string(types.ProductUsageLLMDuration), 20, base.Add(time.Minute))
	insertProductUsage(t, database, 20, 200, string(types.ProductUsageLLMDuration), 30, base.Add(2*time.Minute))

	count, usages, err := service.GetOrganizationUsages(
		context.Background(),
		productUsageAuth(types.AuthTypeOrg, 10, 0, 700),
		[]*protos.Criteria{{Key: "usageType", Value: string(types.ProductUsageLLMDuration)}},
		&protos.Paginate{Page: 1, PageSize: 10},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Len(t, usages, 1)
	require.Equal(t, wanted.Id, usages[0].Id)
	require.Equal(t, uint64(101), usages[0].ProjectId)
}

func TestGetOrganizationUsagesRejectsUnsafeCriteria(t *testing.T) {
	service, _ := newProductUsageServiceTest(t)
	_, _, err := service.GetOrganizationUsages(
		context.Background(),
		productUsageAuth(types.AuthTypeUser, 10, 0, 1),
		[]*protos.Criteria{{Key: "organization_id OR 1=1", Value: "10"}},
		nil,
	)
	require.ErrorIs(t, err, ErrInvalidRequest)
}
