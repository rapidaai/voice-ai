package internal_productusage_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
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
		id integer primary key,
		organization_id integer not null,
		project_id integer not null,
		usage_id text not null,
		usage_type text not null,
		usages integer not null,
		unit text not null,
		occurred_at datetime not null,
		created_date datetime not null,
		updated_date datetime,
		status text not null,
		created_actor_type text not null,
		created_actor_id integer not null,
		updated_actor_type text,
		updated_actor_id integer,
		unique (organization_id, project_id, usage_id)
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
		ProjectValue: &types.ProjectContext{
			OrganizationID: organizationID,
			ProjectID:      projectID,
		},
	}
	if authType == types.AuthTypeUser {
		auth.UserValue = &types.UserContext{UserID: actorID}
	}
	return auth
}

func productUsageInput(usageID string, quantity int64) *protos.ProductUsage {
	return &protos.ProductUsage{
		UsageId:    usageID,
		UsageType:  string(types.ProductUsageSTTDuration),
		Usages:     quantity,
		Unit:       string(types.ProductUsageUnitNanosecond),
		OccurredAt: timestamppb.New(time.Date(2026, time.August, 29, 10, 30, 0, 123456789, time.FixedZone("test", 5*60*60+30*60))),
	}
}

func TestCreateProductUsagesCreatesAndCountsExactDuplicates(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeUser, 10, 100, 1)
	usage := productUsageInput("550e8400-e29b-41d4-a716-446655440000", 25)

	result, err := service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{usage, usage})
	require.NoError(t, err)
	require.Equal(t, Result{CreatedCount: 1, DuplicateCount: 1}, result)

	var stored struct {
		OrganizationID   uint64
		ProjectID        uint64
		UsageID          string
		Usages           int64
		CreatedActorType string
		CreatedActorID   uint64
		OccurredAt       time.Time
	}
	require.NoError(t, database.Table("product_usages").Take(&stored).Error)
	require.Equal(t, uint64(10), stored.OrganizationID)
	require.Equal(t, uint64(100), stored.ProjectID)
	require.Equal(t, usage.GetUsageId(), stored.UsageID)
	require.Equal(t, int64(25), stored.Usages)
	require.Equal(t, types.AuthTypeUser.String(), stored.CreatedActorType)
	require.Equal(t, uint64(1), stored.CreatedActorID)
	require.Equal(t, stored.OccurredAt.Truncate(time.Microsecond), stored.OccurredAt)

	result, err = service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{usage})
	require.NoError(t, err)
	require.Equal(t, Result{DuplicateCount: 1}, result)
}

func TestCreateProductUsagesRollsBackConflictingBatch(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeProject, 10, 100, 501)
	existing := productUsageInput("550e8400-e29b-41d4-a716-446655440001", 10)
	_, err := service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{existing})
	require.NoError(t, err)

	newUsage := productUsageInput("550e8400-e29b-41d4-a716-446655440002", 20)
	conflicting := productUsageInput(existing.GetUsageId(), 11)
	_, err = service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{newUsage, conflicting})
	require.ErrorIs(t, err, ErrUsageConflict)

	var count int64
	require.NoError(t, database.Table("product_usages").Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestCreateProductUsagesScopesUsageIDToProject(t *testing.T) {
	service, database := newProductUsageServiceTest(t)
	usage := productUsageInput("550e8400-e29b-41d4-a716-446655440003", 10)

	_, err := service.CreateProductUsages(context.Background(), productUsageAuth(types.AuthTypeUser, 10, 100, 1), []*protos.ProductUsage{usage})
	require.NoError(t, err)
	_, err = service.CreateProductUsages(context.Background(), productUsageAuth(types.AuthTypeProject, 10, 101, 502), []*protos.ProductUsage{usage})
	require.NoError(t, err)

	var count int64
	require.NoError(t, database.Table("product_usages").Where("usage_id = ?", usage.GetUsageId()).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

func TestCreateProductUsagesValidatesInputAndProjectOwnership(t *testing.T) {
	service, _ := newProductUsageServiceTest(t)
	auth := productUsageAuth(types.AuthTypeService, 10, 100, 601)

	invalidQuantity := productUsageInput("550e8400-e29b-41d4-a716-446655440004", 0)
	_, err := service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{invalidQuantity})
	require.ErrorIs(t, err, ErrInvalidRequest)

	invalidUnit := productUsageInput("550e8400-e29b-41d4-a716-446655440005", 1)
	invalidUnit.Unit = "second"
	_, err = service.CreateProductUsages(context.Background(), auth, []*protos.ProductUsage{invalidUnit})
	require.ErrorIs(t, err, ErrInvalidRequest)

	_, err = service.CreateProductUsages(context.Background(), productUsageAuth(types.AuthTypeUser, 20, 100, 1), []*protos.ProductUsage{productUsageInput("550e8400-e29b-41d4-a716-446655440006", 1)})
	require.ErrorIs(t, err, ErrProjectMismatch)

	_, err = service.CreateProductUsages(context.Background(), auth, nil)
	require.True(t, errors.Is(err, ErrInvalidRequest))
}
