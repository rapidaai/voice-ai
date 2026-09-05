package web_api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/middlewares"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type onboardingOrganizationService struct {
	internal_service.OrganizationService
	auth *types.Authentication
}

func (service *onboardingOrganizationService) Create(_ context.Context, auth *types.Authentication, name string, size string, industry string) (*internal_entity.Organization, error) {
	service.auth = auth
	return &internal_entity.Organization{
		Audited:  gorm_models.Audited{Id: 21},
		Name:     name,
		Size:     size,
		Industry: industry,
	}, nil
}

type onboardingUserService struct {
	internal_service.UserService
	auth   *types.Authentication
	userID uint64
}

type onboardingAuthenticator struct {
	principle types.Principle
}

func (authenticator onboardingAuthenticator) Authorize(_ context.Context, token string, userID uint64) (types.Principle, error) {
	if token != "signup-token" || userID != 11 {
		return nil, types.ErrUnauthenticated
	}
	return authenticator.principle, nil
}

func (authenticator onboardingAuthenticator) AuthPrinciple(_ context.Context, userID uint64) (types.Principle, error) {
	if userID != 11 {
		return nil, types.ErrUnauthenticated
	}
	return authenticator.principle, nil
}

func (service *onboardingUserService) CreateOrganizationRole(_ context.Context, auth *types.Authentication, role string, userID uint64, organizationID uint64, state type_enums.RecordState) (*internal_entity.UserOrganizationRole, error) {
	service.auth = auth
	service.userID = userID
	return &internal_entity.UserOrganizationRole{
		Audited:        gorm_models.Audited{Id: 22},
		UserAuthId:     userID,
		OrganizationId: organizationID,
		Role:           role,
		Mutable:        gorm_models.Mutable{Status: state},
	}, nil
}

func onboardingAuthentication(userID, organizationID uint64) *types.Authentication {
	auth := &types.Authentication{
		AuthType:   types.AuthTypeUser,
		ActorValue: &types.ActorIdentity{Type: types.ActorTypeUser, ID: userID},
		UserValue:  &types.UserContext{UserID: userID},
	}
	if organizationID != 0 {
		auth.OrganizationValue = &types.OrganizationContext{OrganizationID: organizationID}
	}
	return auth
}

func TestCreateOrganizationAcceptsUserWithoutOrganization(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	newAPI := func() (webOrganizationApi, *onboardingOrganizationService, *onboardingUserService) {
		organizationService := &onboardingOrganizationService{}
		userService := &onboardingUserService{}
		return webOrganizationApi{
			logger:              logger,
			organizationService: organizationService,
			userService:         userService,
		}, organizationService, userService
	}
	authenticator := onboardingAuthenticator{principle: &types.PlainAuthPrinciple{
		User:  types.UserInfo{Id: 11},
		Token: types.AuthToken{Token: "signup-token"},
	}}

	t.Run("grpc", func(t *testing.T) {
		api, organizationService, userService := newAPI()
		request := &protos.CreateOrganizationRequest{
			OrganizationName:     "Acme",
			OrganizationSize:     "small",
			OrganizationIndustry: "software",
		}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			types.AUTHORIZATION_KEY, "signup-token",
			types.AUTH_KEY, "11",
		))
		responseValue, err := middlewares.NewAuthenticationUnaryServerMiddleware(authenticator, logger)(
			ctx,
			request,
			nil,
			func(ctx context.Context, request any) (any, error) {
				return (&webOrganizationGRPCApi{webOrganizationApi: api}).CreateOrganization(ctx, request.(*protos.CreateOrganizationRequest))
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		response := responseValue.(*protos.CreateOrganizationResponse)
		if !response.GetSuccess() || response.GetData().GetId() != 21 || response.GetRole().GetId() != 22 {
			t.Fatalf("response = %+v", response)
		}
		if organizationService.auth == nil || userService.auth != organizationService.auth || userService.userID != 11 {
			t.Fatal("services did not receive authenticated onboarding user")
		}
	})

	t.Run("rest", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		api, organizationService, userService := newAPI()
		engine := gin.New()
		engine.Use(middlewares.NewAuthenticationMiddleware(authenticator, logger))
		engine.POST("/organization", (&webOrganizationRPCApi{webOrganizationApi: api}).CreateOrganization)
		request := httptest.NewRequest(http.MethodPost, "/organization", bytes.NewBufferString(`{"organization_name":"Acme","organization_size":"small","organization_industry":"software"}`))
		request.Header.Set("content-type", "application/json")
		request.Header.Set(types.AUTHORIZATION_KEY, "signup-token")
		request.Header.Set(types.AUTH_KEY, "11")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if organizationService.auth == nil || userService.auth != organizationService.auth || userService.userID != 11 {
			t.Fatal("services did not receive authenticated onboarding user")
		}
	})
}

func TestCreateOrganizationRejectsExistingOrganization(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), types.CTX_, onboardingAuthentication(11, 12))
	response, err := (&webOrganizationGRPCApi{webOrganizationApi: webOrganizationApi{logger: logger}}).CreateOrganization(ctx, &protos.CreateOrganizationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetSuccess() || response.GetCode() != 400 {
		t.Fatalf("response = %+v", response)
	}
}

func TestCreateOrganizationRejectsNonUserAuthentication(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 12},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 12},
	})
	_, err := (&webOrganizationGRPCApi{}).CreateOrganization(ctx, &protos.CreateOrganizationRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
}
