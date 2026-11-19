package web_api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type onboardingOrganizationService struct {
	internal_service.OrganizationService
	auth authentication.Principal
}

func (service *onboardingOrganizationService) Create(_ context.Context, auth authentication.Principal, name string, size string, industry string) (*internal_entity.Organization, error) {
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
	auth   authentication.Principal
	userID uint64
	roleID uint64
}

func (service *onboardingUserService) CreateOrganizationRole(_ context.Context, auth authentication.Principal, roleDefinitionID uint64, userID uint64, organizationID uint64, state type_enums.RecordState) (*internal_entity.UserOrganizationRole, error) {
	service.auth = auth
	service.userID = userID
	service.roleID = roleDefinitionID
	return &internal_entity.UserOrganizationRole{
		Audited:          gorm_models.Audited{Id: 22},
		UserAuthId:       userID,
		OrganizationId:   organizationID,
		RoleDefinitionID: roleDefinitionID,
		Mutable:          gorm_models.Mutable{Status: state},
	}, nil
}

func (service *onboardingUserService) GetOrganizationRoles(_ context.Context, userID uint64) ([]*internal_entity.UserOrganizationRole, error) {
	return []*internal_entity.UserOrganizationRole{{
		UserAuthId: userID, OrganizationId: 21, RoleDefinitionID: service.roleID,
		RoleDefinition: internal_entity.RoleDefinition{Key: "organization_owner", Name: "Organization Owner"},
	}}, nil
}

func onboardingAuthentication(t *testing.T, userID, organizationID uint64) authentication.Principal {
	t.Helper()
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: organizationID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: "test-session",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
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
	principal := onboardingAuthentication(t, 11, 0)

	t.Run("grpc", func(t *testing.T) {
		api, organizationService, userService := newAPI()
		request := &protos.CreateOrganizationRequest{
			OrganizationName:     "Acme",
			OrganizationSize:     "small",
			OrganizationIndustry: "software",
		}
		ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
		if err != nil {
			t.Fatal(err)
		}
		responseValue, err := (&webOrganizationGRPCApi{webOrganizationApi: api}).CreateOrganization(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		response := responseValue
		if !response.GetSuccess() || response.GetData().GetId() != 21 {
			t.Fatalf("response = %+v", response)
		}
		if organizationService.auth.Subject().ID != 11 || userService.auth.Subject().ID != 11 || userService.userID != 11 {
			t.Fatal("services did not receive authenticated onboarding user")
		}
	})

	t.Run("rest", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		api, organizationService, userService := newAPI()
		engine := gin.New()
		engine.Use(func(c *gin.Context) {
			requestContext, err := middleware_authentication.ContextWithPrincipal(c.Request.Context(), principal)
			if err != nil {
				t.Fatal(err)
			}
			c.Request = c.Request.WithContext(requestContext)
			c.Next()
		})
		engine.POST("/organization", (&webOrganizationRPCApi{webOrganizationApi: api}).CreateOrganization)
		request := httptest.NewRequest(http.MethodPost, "/organization", bytes.NewBufferString(`{"organization_name":"Acme","organization_size":"small","organization_industry":"software"}`))
		request.Header.Set("content-type", "application/json")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if organizationService.auth.Subject().ID != 11 || userService.auth.Subject().ID != 11 || userService.userID != 11 {
			t.Fatal("services did not receive authenticated onboarding user")
		}
	})
}

func TestCreateOrganizationRejectsExistingOrganization(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), onboardingAuthentication(t, 11, 12))
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&webOrganizationGRPCApi{webOrganizationApi: webOrganizationApi{logger: logger}}).CreateOrganization(ctx, &protos.CreateOrganizationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetSuccess() || response.GetCode() != 400 {
		t.Fatalf("response = %+v", response)
	}
}

func TestUpdateBillingInformationRequiresOrganizationPrincipal(t *testing.T) {
	api := &webOrganizationGRPCApi{}
	request := &protos.UpdateBillingInformationRequest{}
	if _, err := api.UpdateBillingInformation(context.Background(), request); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing principal code = %v", status.Code(err))
	}

	withoutOrganization, err := middleware_authentication.ContextWithPrincipal(context.Background(), onboardingAuthentication(t, 11, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateBillingInformation(withoutOrganization, request); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("missing organization code = %v", status.Code(err))
	}

	withOrganization, err := middleware_authentication.ContextWithPrincipal(context.Background(), onboardingAuthentication(t, 11, 12))
	if err != nil {
		t.Fatal(err)
	}
	response, err := api.UpdateBillingInformation(withOrganization, request)
	if err != nil || response != nil {
		t.Fatalf("authorized response = %v, error = %v", response, err)
	}
}
