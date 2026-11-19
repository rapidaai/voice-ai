// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	internal_connect "github.com/rapidaai/api/web-api/internal/connect"
	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	protos "github.com/rapidaai/protos"
)

type authenticationResponseUserService struct {
	internal_service.UserService
	result *internal_service.UserAuthentication
	user   *internal_entity.UserAuth
	issued bool
}

func (service *authenticationResponseUserService) AuthenticateUser(context.Context, string, string) (*internal_service.UserAuthentication, error) {
	service.issued = true
	return service.result, nil
}

func (service *authenticationResponseUserService) AuthenticationForPrincipal(context.Context, authentication.Principal) (*internal_service.UserAuthentication, error) {
	return service.result, nil
}

func (service *authenticationResponseUserService) CreateAuthentication(context.Context, uint64) (*internal_service.UserAuthentication, error) {
	service.issued = true
	return service.result, nil
}

func (service *authenticationResponseUserService) Get(context.Context, string) (*internal_entity.UserAuth, error) {
	return service.user, nil
}

func (service *authenticationResponseUserService) UpdatePassword(context.Context, authentication.Principal, uint64, string) (*internal_entity.UserAuth, error) {
	return service.user, nil
}

func (*authenticationResponseUserService) ActivateAllOrganizationRole(context.Context, uint64) error {
	return nil
}

func (*authenticationResponseUserService) ActivateAllProjectRoles(context.Context, uint64) error {
	return nil
}

func (service *authenticationResponseUserService) Activate(context.Context, uint64, string, *string) (*internal_entity.UserAuth, error) {
	return service.user, nil
}

func (*authenticationResponseUserService) GetSocial(context.Context, uint64) (*internal_entity.UserSocial, error) {
	return &internal_entity.UserSocial{}, nil
}

func authenticationResponseUser(status type_enums.RecordState) *internal_entity.UserAuth {
	user := &internal_entity.UserAuth{Name: "Ada", Email: "ada@example.com"}
	user.Id = 7
	user.Status = status
	return user
}

func authenticationResponseResult(t *testing.T) *internal_service.UserAuthentication {
	t.Helper()
	value, err := authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: 20},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		&authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
		[]authentication.Organization{{
			ID: 20, Name: "Acme",
			Roles: []authentication.Role{{
				ID: 11, Key: "billing_viewer", Name: "Billing viewer", Scope: authentication.RoleScopeOrganization,
				Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionBuiltIn,
				Permissions: []authentication.Permission{{ID: 1, Key: "organization.billing.read", Scope: authentication.RoleScopeOrganization, Name: "Read billing"}},
			}},
			BillingPlan: &authentication.BillingPlan{
				ID: 31, Name: "Enterprise", Slug: "enterprise", IsActive: true, Currency: "usd",
				Quotas: []authentication.BillingPlanQuota{{ID: 41, ResourceType: "projects", QuotaLimit: -1, Unit: "count"}},
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return &internal_service.UserAuthentication{
		Authentication: value,
		Token: internal_entity.UserAuthToken{
			Audited: gorm_models.Audited{Id: 81}, Token: "raw-token", TokenType: "auth-token", ExpireAt: time.Now().Add(time.Hour),
		},
	}
}

func assertUnifiedAuthenticationResponse(t *testing.T, response *protos.AuthenticateResponse) {
	t.Helper()

	if response.GetAuthentication() == nil || response.GetToken() == nil {
		t.Fatalf("response = %+v", response)
	}
	organization := response.GetAuthentication().GetOrganizations()[0]
	if organization.GetBillingPlan().GetSlug() != "enterprise" || organization.GetBillingPlan().GetQuotas()[0].GetQuotaLimit() != -1 {
		t.Fatalf("organization = %+v", organization)
	}
	if response.GetToken().GetId() != 81 || response.GetAuthentication().GetPrincipal().GetProof().GetReference().GetValue() != "81" {
		t.Fatalf("authentication response = %+v", response)
	}
}

func TestUserAuthenticationResponseIncludesAuthenticationAndToken(t *testing.T) {
	assertUnifiedAuthenticationResponse(t, userAuthenticationResponse(authenticationResponseResult(t)))
}

func TestRPCAuthenticationReturnsFlatAuthenticationResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	result := authenticationResponseResult(t)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := &authenticationResponseUserService{result: result}
	api := &webAuthRPCApi{webAuthApi: webAuthApi{logger: logger, userService: service}}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/authenticate", bytes.NewBufferString(`{"email":"ada@example.com","password":"secret"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	api.Authenticate(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["authentication"] == nil || response["token"] == nil || response["session"] != nil || response["data"] != nil {
		t.Fatalf("response = %v", response)
	}
}

func TestAuthenticationHandlersReturnFlatAuthenticationResult(t *testing.T) {
	result := authenticationResponseResult(t)
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("password", func(t *testing.T) {
		service := &authenticationResponseUserService{result: result}
		api := &webAuthGRPCApi{webAuthApi: webAuthApi{logger: logger, userService: service}}
		response, err := api.Authenticate(context.Background(), &protos.AuthenticateRequest{Email: "ada@example.com", Password: "secret"})
		if err != nil {
			t.Fatal(err)
		}
		assertUnifiedAuthenticationResponse(t, response)
	})

	t.Run("registration", func(t *testing.T) {
		service := &authenticationResponseUserService{
			result: result,
			user:   authenticationResponseUser(type_enums.RECORD_INVITED),
		}
		api := &webAuthGRPCApi{webAuthApi: webAuthApi{logger: logger, userService: service}}
		response, err := api.RegisterUser(context.Background(), &protos.RegisterUserRequest{Email: "ada@example.com", Password: "secret", Name: "Ada"})
		if err != nil {
			t.Fatal(err)
		}
		assertUnifiedAuthenticationResponse(t, response)
	})

	t.Run("authorize", func(t *testing.T) {
		service := &authenticationResponseUserService{result: result}
		api := &webAuthGRPCApi{webAuthApi: webAuthApi{logger: logger, userService: service}}
		ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), result.Authentication.Principal())
		if err != nil {
			t.Fatal(err)
		}
		response, err := api.Authorize(ctx, &protos.AuthorizeRequest{})
		if err != nil {
			t.Fatal(err)
		}
		assertUnifiedAuthenticationResponse(t, response)
	})

	t.Run("social", func(t *testing.T) {
		service := &authenticationResponseUserService{
			result: result,
			user:   authenticationResponseUser(type_enums.RECORD_ACTIVE),
		}
		api := &webAuthApi{logger: logger, userService: service}
		response, err := api.RegisterSocialUser(context.Background(), &internal_connect.OpenID{Email: "ada@example.com", Source: "google"})
		if err != nil {
			t.Fatal(err)
		}
		assertUnifiedAuthenticationResponse(t, response)
		if !service.issued {
			t.Fatal("social authentication did not issue a fresh session")
		}
	})
}
