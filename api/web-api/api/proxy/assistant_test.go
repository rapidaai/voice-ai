package web_proxy_api

import (
	"context"
	"testing"

	web_api "github.com/rapidaai/api/web-api/api"
	web_config "github.com/rapidaai/api/web-api/config"
	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	app_config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	assistant_client "github.com/rapidaai/pkg/clients/workflow"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
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
	getAllAssistantFunc         func(context.Context, authentication.Principal, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error)
	getAssistantFunc            func(context.Context, authentication.Principal, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error)
	createAssistantFunc         func(context.Context, authentication.Principal, *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error)
	createAssistantProviderFunc func(context.Context, authentication.Principal, *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error)
	createPhoneDeploymentFunc   func(context.Context, authentication.Principal, *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error)
}

func (f *fakeAssistantServiceClient) GetAllAssistant(ctx context.Context, auth authentication.Principal, criteria []*protos.Criteria, paginate *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
	return f.getAllAssistantFunc(ctx, auth, criteria, paginate)
}

func (f *fakeAssistantServiceClient) GetAssistant(ctx context.Context, auth authentication.Principal, request *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
	return f.getAssistantFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistant(ctx context.Context, auth authentication.Principal, request *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
	return f.createAssistantFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistantProvider(ctx context.Context, auth authentication.Principal, request *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
	return f.createAssistantProviderFunc(ctx, auth, request)
}

func (f *fakeAssistantServiceClient) CreateAssistantPhoneDeployment(ctx context.Context, auth authentication.Principal, request *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
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
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "assistant-test-session"},
		},
		nil,
	)
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx
}

func assistantProjectProxyContext() (context.Context, authentication.Principal) {
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: 30},
		authentication.Tenant{OrganizationID: 10, ProjectID: 20},
		authentication.Proof{
			Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: "30"},
		},
		nil,
	)
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx, principal
}

func TestGetAllAssistantPreservesCreatedActor(t *testing.T) {
	api := newAssistantProxyTest(t, &fakeAssistantServiceClient{
		getAllAssistantFunc: func(context.Context, authentication.Principal, []*protos.Criteria, *protos.Paginate) (*protos.Paginated, []*protos.Assistant, error) {
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
		getAssistantFunc: func(context.Context, authentication.Principal, *protos.GetAssistantRequest) (*protos.GetAssistantResponse, error) {
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
		createAssistantFunc: func(_ context.Context, auth authentication.Principal, actualRequest *protos.CreateAssistantRequest) (*protos.GetAssistantResponse, error) {
			require.Equal(t, expectedAuth, auth)
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
		createAssistantProviderFunc: func(_ context.Context, auth authentication.Principal, actualRequest *protos.CreateAssistantProviderRequest) (*protos.GetAssistantProviderResponse, error) {
			require.Equal(t, expectedAuth, auth)
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
		createPhoneDeploymentFunc: func(_ context.Context, auth authentication.Principal, actualRequest *protos.CreateAssistantDeploymentRequest) (*protos.GetAssistantPhoneDeploymentResponse, error) {
			require.Equal(t, expectedAuth, auth)
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
