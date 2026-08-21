package web_proxy_api

import (
	"context"
	"testing"

	web_api "github.com/rapidaai/api/web-api/api"
	web_config "github.com/rapidaai/api/web-api/config"
	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	app_config "github.com/rapidaai/config"
	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type assistantProxyPostgresConnector struct {
	db *gorm.DB
}

func (t *assistantProxyPostgresConnector) Connect(ctx context.Context) error {
	return nil
}

func (t *assistantProxyPostgresConnector) Name() string {
	return "assistant-proxy-test-postgres"
}

func (t *assistantProxyPostgresConnector) IsConnected(ctx context.Context) bool {
	return true
}

func (t *assistantProxyPostgresConnector) Disconnect(ctx context.Context) error {
	return nil
}

func (t *assistantProxyPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.DB(ctx).Raw(qry).Scan(dest).Error
}

func (t *assistantProxyPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if tx, ok := connectors.PostgresTxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return t.db.WithContext(ctx)
}

type fakeAssistantServiceClient struct {
	assistant_client.AssistantServiceClient
	getAllAssistantFunc func(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error)
	getAssistantFunc    func(context.Context, *types.Authentication, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error)
}

func (f *fakeAssistantServiceClient) GetAllAssistant(ctx context.Context, auth *types.Authentication, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
	return f.getAllAssistantFunc(ctx, auth, criteria, paginate)
}

func (f *fakeAssistantServiceClient) GetAssistant(ctx context.Context, auth *types.Authentication, request *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	return f.getAssistantFunc(ctx, auth, request)
}

func newAssistantProxyTest(t *testing.T, client assistant_client.AssistantServiceClient) *webAssistantGRPCApi {
	t.Helper()
	appLogger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE user_auths (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, name text, email text, password text, source text)`).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 42},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Assistant Owner",
		Email:    "assistant-owner@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)

	postgres := &assistantProxyPostgresConnector{db: db}
	cfg := &web_config.WebAppConfig{
		AppConfig: app_config.AppConfig{
			Ui: app_config.ServiceHostConfig{Host: "http://ui.test"},
		},
	}

	return &webAssistantGRPCApi{
		webAssistantApi: webAssistantApi{
			WebApi:          web_api.NewWebApi(cfg, appLogger, postgres, nil),
			cfg:             cfg,
			logger:          appLogger,
			postgres:        postgres,
			assistantClient: client,
		},
	}
}

func assistantProxyContext() context.Context {
	return context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: 1},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 10},
	})
}

func TestGetAllAssistantHydratesTopLevelCreatedUser(t *testing.T) {
	api := newAssistantProxyTest(t, &fakeAssistantServiceClient{
		getAllAssistantFunc: func(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
			return &protos.Paginated{TotalItem: 1, CurrentPage: 1}, []*protos.Assistant{
				{Id: 100, CreatedBy: 42, Name: "Support assistant"},
			}, nil
		},
	})

	res, err := api.GetAllAssistant(assistantProxyContext(), &protos.GetAllAssistantRequest{
		Paginate: &protos.Paginate{Page: 1, PageSize: 10},
	})

	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Len(t, res.GetData(), 1)
	require.Equal(t, "Assistant Owner", res.GetData()[0].GetCreatedUser().GetName())
}

func TestGetAssistantHydratesTopLevelCreatedUser(t *testing.T) {
	api := newAssistantProxyTest(t, &fakeAssistantServiceClient{
		getAssistantFunc: func(context.Context, *types.Authentication, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
			return &protos.GetAssistantResponse{
				Code:    200,
				Success: true,
				Data:    &protos.Assistant{Id: 100, CreatedBy: 42, Name: "Support assistant"},
			}, nil
		},
	})

	res, err := api.GetAssistant(assistantProxyContext(), &protos.GetAssistantRequest{})

	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, "Assistant Owner", res.GetData().GetCreatedUser().GetName())
}
