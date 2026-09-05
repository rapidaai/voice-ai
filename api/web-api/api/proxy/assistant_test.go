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
	"google.golang.org/protobuf/proto"
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
	getAllAssistantFunc         func(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error)
	getAssistantFunc            func(context.Context, *types.Authentication, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error)
	createAssistantFunc         func(context.Context, *types.Authentication, *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error)
	createAssistantProviderFunc func(context.Context, *types.Authentication, *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error)
	createPhoneDeploymentFunc   func(context.Context, *types.Authentication, *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error)
}

func (f *fakeAssistantServiceClient) GetAllAssistant(ctx context.Context, auth *types.Authentication, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
	return f.getAllAssistantFunc(ctx, auth, criteria, paginate)
}

func (f *fakeAssistantServiceClient) GetAssistant(ctx context.Context, auth *types.Authentication, request *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	return f.getAssistantFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistant(ctx context.Context, auth *types.Authentication, request *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
	return f.createAssistantFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistantProvider(ctx context.Context, auth *types.Authentication, request *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
	return f.createAssistantProviderFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistantPhoneDeployment(ctx context.Context, auth *types.Authentication, request *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
	return f.createPhoneDeploymentFunc(ctx, auth, request)
}

func newAssistantProxyTest(t *testing.T, client assistant_client.AssistantServiceClient) *webAssistantGRPCApi {
	t.Helper()
	appLogger, err := commons.NewApplicationLogger()
	require.NoError(t, err)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE user_auths (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, name text, email text, password text, source text)`).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 42},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
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
	actor := types.ActorIdentity{Type: types.ActorTypeUser, ID: 1}
	return context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &actor,
		UserValue:         &types.UserContext{UserID: 1},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 10},
	})
}

func assistantProjectProxyContext() (context.Context, *types.Authentication) {
	actor := types.ActorIdentity{Type: types.ActorTypeProject, ID: 30}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 10},
		ProjectValue:      &types.ProjectContext{OrganizationID: 10, ProjectID: 20},
	}
	return context.WithValue(context.Background(), types.CTX_, auth), auth
}

func TestGetAllAssistantPreservesCreatedActor(t *testing.T) {
	api := newAssistantProxyTest(t, &fakeAssistantServiceClient{
		getAllAssistantFunc: func(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
			return &protos.Paginated{TotalItem: 1, CurrentPage: 1}, []*protos.Assistant{
				{Id: 100, CreatedActor: &protos.AuditActor{Type: "user", Id: 42, DisplayName: proto.String("Assistant Owner")}, Name: "Support assistant"},
			}, nil
		},
	})

	res, err := api.GetAllAssistant(assistantProxyContext(), &protos.GetAllAssistantRequest{
		Paginate: &protos.Paginate{Page: 1, PageSize: 10},
	})

	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Len(t, res.GetData(), 1)
	require.Equal(t, "Assistant Owner", res.GetData()[0].GetCreatedActor().GetDisplayName())
}

func TestGetAssistantPreservesCreatedActor(t *testing.T) {
	api := newAssistantProxyTest(t, &fakeAssistantServiceClient{
		getAssistantFunc: func(context.Context, *types.Authentication, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
			return &protos.GetAssistantResponse{
				Code:    200,
				Success: true,
				Data:    &protos.Assistant{Id: 100, CreatedActor: &protos.AuditActor{Type: "user", Id: 42, DisplayName: proto.String("Assistant Owner")}, Name: "Support assistant"},
			}, nil
		},
	})

	res, err := api.GetAssistant(assistantProxyContext(), &protos.GetAssistantRequest{})

	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, "Assistant Owner", res.GetData().GetCreatedActor().GetDisplayName())
}

func TestCreateAssistantAcceptsProjectScope(t *testing.T) {
	ctx, expectedAuth := assistantProjectProxyContext()
	request := &protos.CreateAssistantRequest{Name: "Project assistant"}
	client := &fakeAssistantServiceClient{
		createAssistantFunc: func(_ context.Context, auth *types.Authentication, actualRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
			require.Same(t, expectedAuth, auth)
			require.Same(t, request, actualRequest)
			return &protos.GetAssistantResponse{Code: 200, Success: true}, nil
		},
	}

	response, err := newAssistantProxyTest(t, client).CreateAssistant(ctx, request)

	require.NoError(t, err)
	require.True(t, response.GetSuccess())
}

func TestCreateAssistantProviderAcceptsProjectScope(t *testing.T) {
	ctx, expectedAuth := assistantProjectProxyContext()
	request := &protos.CreateAssistantProviderRequest{AssistantId: 100}
	client := &fakeAssistantServiceClient{
		createAssistantProviderFunc: func(_ context.Context, auth *types.Authentication, actualRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
			require.Same(t, expectedAuth, auth)
			require.Same(t, request, actualRequest)
			return &protos.GetAssistantProviderResponse{Code: 200, Success: true}, nil
		},
	}

	response, err := newAssistantProxyTest(t, client).CreateAssistantProvider(ctx, request)

	require.NoError(t, err)
	require.True(t, response.GetSuccess())
}

func TestCreateAssistantPhoneDeploymentAcceptsProjectScope(t *testing.T) {
	ctx, expectedAuth := assistantProjectProxyContext()
	request := &protos.CreateAssistantDeploymentRequest{}
	client := &fakeAssistantServiceClient{
		createPhoneDeploymentFunc: func(_ context.Context, auth *types.Authentication, actualRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
			require.Same(t, expectedAuth, auth)
			require.Same(t, request, actualRequest)
			return &protos.GetAssistantPhoneDeploymentResponse{Code: 200, Success: true}, nil
		},
	}
	api := &webAssistantDeploymentGRPCApi{
		webAssistantDeploymentApi: webAssistantDeploymentApi{assistantClient: client},
	}

	response, err := api.CreateAssistantPhoneDeployment(ctx, request)

	require.NoError(t, err)
	require.True(t, response.GetSuccess())
}
