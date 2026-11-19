// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	internal_connects "github.com/rapidaai/api/web-api/internal/connect"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"

	"github.com/gin-gonic/gin"

	config "github.com/rapidaai/api/web-api/config"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	"github.com/rapidaai/pkg/authentication"
	authentication_resolver "github.com/rapidaai/pkg/authentication/resolver"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer "github.com/rapidaai/pkg/clients/external/emailer"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	protos "github.com/rapidaai/protos"
)

type webAuthApi struct {
	cfg                 *config.WebAppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	userService         internal_services.UserService
	organizationService internal_services.OrganizationService
	projectService      internal_services.ProjectService
	emailerClient       external_clients.Emailer
	githubConnect       internal_connects.GithubConnect
	linkedinConnect     internal_connects.LinkedinConnect
	googleConnect       internal_connects.GoogleConnect
}

type webAuthRPCApi struct {
	webAuthApi
}

type webAuthGRPCApi struct {
	webAuthApi
	userAuthenticationService         authentication_resolver.UserAuthenticationService
	projectAuthenticationService      authentication_resolver.ProjectAuthenticationService
	organizationAuthenticationService authentication_resolver.OrganizationAuthenticationService
}

var GOOGLE_STATE = "google"

func NewAuthRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, logger commons.Logger, postgres connectors.PostgresConnector) *webAuthRPCApi {
	return &webAuthRPCApi{
		webAuthApi{
			cfg:             config,
			logger:          logger,
			postgres:        postgres,
			userService:     internal_user_service.NewUserService(logger, postgres),
			emailerClient:   external_emailer.NewEmailer(config.EmailerConfig, logger),
			githubConnect:   internal_connects.NewGithubAuthenticationConnect(config, oauthCfg, logger, postgres),
			linkedinConnect: internal_connects.NewLinkedinAuthenticationConnect(config, oauthCfg, logger, postgres),
			googleConnect:   internal_connects.NewGoogleAuthenticationConnect(config, oauthCfg, logger, postgres),
		},
	}
}

func NewAuthGRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, organizationCredentialKey []byte, logger commons.Logger, postgres connectors.PostgresConnector) protos.AuthenticationServiceServer {
	return &webAuthGRPCApi{
		webAuthApi: webAuthApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			userService:         internal_user_service.NewUserService(logger, postgres),
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			emailerClient:       external_emailer.NewEmailer(config.EmailerConfig, logger),
			githubConnect:       internal_connects.NewGithubAuthenticationConnect(config, oauthCfg, logger, postgres),
			linkedinConnect:     internal_connects.NewLinkedinAuthenticationConnect(config, oauthCfg, logger, postgres),
			googleConnect:       internal_connects.NewGoogleAuthenticationConnect(config, oauthCfg, logger, postgres),
		},
		userAuthenticationService:         internal_user_service.NewPrincipalAuthenticationService(logger, postgres),
		projectAuthenticationService:      internal_project_service.NewPrincipalAuthenticationService(logger, postgres),
		organizationAuthenticationService: internal_organization_service.NewPrincipalAuthenticationService(logger, postgres, organizationCredentialKey),
	}
}

func userAuthenticationResponse(result *internal_services.UserAuthentication) *protos.AuthenticateResponse {
	return &protos.AuthenticateResponse{
		Code: 200, Success: true, Authentication: result.Authentication.Proto(),
		Token: &protos.Token{
			Id: result.Token.Id, Token: result.Token.Token,
			TokenType: result.Token.TokenType, IsExpired: !result.Token.ExpireAt.After(time.Now()),
		},
	}
}

// all the rpc handler
func (wAuthApi *webAuthRPCApi) Authenticate(c *gin.Context) {
	wAuthApi.logger.Debugf("Authenticate from rpc with gin context %v", c)
	var irRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := c.Bind(&irRequest)
	if err != nil {
		c.JSON(500, "unable to parse the request, some of the required field missing.")
		return
	}

	result, err := wAuthApi.userService.AuthenticateUser(c, irRequest.Email, irRequest.Password)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}
	c.JSON(http.StatusOK, userAuthenticationResponse(result))
	return
}

func (webAuthApi *webAuthRPCApi) RegisterUser(c *gin.Context) {
	var irRequest struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	err := c.Bind(&irRequest)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}
	source := "rest-api"
	webAuthApi.logger.Debugf("RegisterUser from rpc with gin context %v", irRequest)
	cUser, err := webAuthApi.userService.Get(c, irRequest.Email)
	if err != nil {
		webAuthApi.logger.Debug("registering new user into the system.")
		aUser, err := webAuthApi.userService.Create(c, authentication.Principal{}, irRequest.Name, irRequest.Email, irRequest.Password, type_enums.RECORD_ACTIVE, &source)
		if err != nil {
			webAuthApi.logger.Errorf("registering new user failed with err %v", err)
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}

		err = webAuthApi.emailerClient.EmailRichText(
			c,
			external_clients.Contact{
				Name:  aUser.Name,
				Email: aUser.Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.Name,
			},
		)
		if err != nil {
			webAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		webAuthApi.logger.Debugf("user after creation %v", aUser)
		result, err := webAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, commons.Response{
				Code:    http.StatusInternalServerError,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}
		c.JSON(http.StatusOK, userAuthenticationResponse(result))
		return
	}

	// if it's invited user then
	if cUser.Status == type_enums.RECORD_INVITED {
		// password need to fix
		_, err := webAuthApi.userService.UpdatePassword(c, authentication.Principal{}, cUser.Id, irRequest.Password)
		if err != nil {
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}

		// activate org
		if err = webAuthApi.userService.ActivateAllOrganizationRole(c, cUser.Id); err != nil {
			webAuthApi.logger.Debugf("Error while registering user %v", err)
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}
		// activate project
		if err := webAuthApi.userService.ActivateAllProjectRoles(c, cUser.Id); err != nil {
			webAuthApi.logger.Debugf("Error while registering user %v", err)
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}

		webAuthApi.logger.Debug("activating an invited user.")
		aUser, err := webAuthApi.userService.Activate(c, cUser.Id, irRequest.Name, nil)
		if err != nil {
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}
		result, err := webAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, commons.Response{
				Code:    http.StatusInternalServerError,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}
		c.JSON(http.StatusOK, userAuthenticationResponse(result))
		return
	}

	c.JSON(500, commons.Response{
		Code:    500,
		Success: false,
		Data:    commons.ErrorMessage{Code: 100, Message: errors.New("duplicate email registeration request")},
	})
	return
}

// all grpc handler
/*
	Authneitcate handle for grpc request
	- user.email
	- user.password

	signin -> request to authenticate with valid email and password
*/
func (wAuthApi *webAuthGRPCApi) Authenticate(c context.Context, irRequest *protos.AuthenticateRequest) (*protos.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("Authenticate from grpc with requestPayload %v, %v", irRequest, c)

	result, err := wAuthApi.userService.AuthenticateUser(c, irRequest.Email, irRequest.GetPassword())
	if err != nil {
		wAuthApi.logger.Errorf("unable to process authentication %v", err)
		wAuthApi.logger.Debugf("authentication request failed for user %s", irRequest.Email)
		return &protos.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    401,
				ErrorMessage: err.Error(),
				HumanMessage: "Please provide valid credentials to signin into account.",
			}}, nil
	}

	return userAuthenticationResponse(result), nil
}

/*
Register or activate a user to authenticate into the rapida platform
will be streamlining the code for better managing and explaining later
*/
func (wAuthApi *webAuthGRPCApi) RegisterUser(c context.Context, irRequest *protos.RegisterUserRequest) (*protos.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("RegisterUser from grpc with requestPayload %v, %v", irRequest, c)
	if !utf8.ValidString(irRequest.GetName()) || !validator.NotBlank(irRequest.GetName()) ||
		strings.TrimSpace(irRequest.GetName()) != irRequest.GetName() || strings.ContainsRune(irRequest.GetName(), '\x00') {
		return nil, status.Error(codes.InvalidArgument, "name is invalid")
	}
	cUser, err := wAuthApi.userService.Get(c, irRequest.Email)
	source := "direct"
	if err != nil {
		aUser, err := wAuthApi.userService.Create(c, authentication.Principal{}, irRequest.Name, irRequest.Email, irRequest.GetPassword(), type_enums.RECORD_ACTIVE, &source)
		if err != nil {
			wAuthApi.logger.Errorf("creation user failed with err %v", err)
			return &protos.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create an account, please check and try again.",
				}}, nil
		}
		err = wAuthApi.emailerClient.EmailRichText(
			c,
			external_clients.Contact{
				Name:  aUser.Name,
				Email: aUser.Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.Name,
			},
		)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		result, err := wAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			return nil, err
		}
		return userAuthenticationResponse(result), nil
	}

	// already have an active account
	if cUser.Status == type_enums.RECORD_ACTIVE {
		wAuthApi.logger.Errorf("user is already having account and trying to signup")
		return &protos.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your email is already associated with an existing account, try signin.",
			}}, nil
	}
	// if it's invited user then
	if cUser.Status == type_enums.RECORD_INVITED {
		_, err := wAuthApi.userService.UpdatePassword(c, authentication.Principal{}, cUser.Id, irRequest.GetPassword())
		if err != nil {
			wAuthApi.logger.Errorf("Error while updaing password for user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate org
		if err = wAuthApi.userService.ActivateAllOrganizationRole(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate project
		if err := wAuthApi.userService.ActivateAllProjectRoles(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		// activate user
		aUser, err := wAuthApi.userService.Activate(c, cUser.Id, irRequest.Name, nil)
		if err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		result, err := wAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			return nil, err
		}
		return userAuthenticationResponse(result), nil
	}

	return &protos.AuthenticateResponse{Code: 400, Success: false, Error: &protos.Error{
		ErrorCode:    400,
		ErrorMessage: "illegal state of data",
		HumanMessage: "We are facing issue with account creation, please try again in sometime",
	}}, err
}

func (wAuthApi *webAuthGRPCApi) ForgotPassword(c context.Context, irRequest *protos.ForgotPasswordRequest) (*protos.ForgotPasswordResponse, error) {
	wAuthApi.logger.Debugf("ForgotPassword from grpc with requestPayload %v, %v", irRequest.String(), c)

	aUser, err := wAuthApi.userService.Get(c, irRequest.GetEmail())
	if err != nil {
		wAuthApi.logger.Errorf("getting email for forgot password for user %v failed %v", irRequest.GetEmail(), err)
		return &protos.ForgotPasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Your email is not associated with rapida.ai account, please check and try again",
			}}, nil
	}

	if aUser.Status != type_enums.RECORD_ACTIVE {
		wAuthApi.logger.Errorf("user is changing password for not activated user  %v", aUser.Email)
		return &protos.ForgotPasswordResponse{
			Code:    401,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your account is not activated yet. please activate before signin.",
			}}, nil
	}

	token, err := wAuthApi.userService.CreatePasswordToken(c, aUser.Id)
	if err != nil {
		wAuthApi.logger.Errorf("unable to create password token for user %v failed %v", irRequest.GetEmail(), err)
		return &protos.ForgotPasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create reset password token, please try again in sometime.",
			}}, nil
	}
	// userId uint64, name, email,
	// resetPasswordUrl :=
	err = wAuthApi.emailerClient.EmailRichText(
		c,
		external_clients.Contact{
			Name:  aUser.Name,
			Email: aUser.Email,
		},
		"Reset your Rapida password",
		external_emailer_template.RESET_PASSWORD_TEMPLATE,
		map[string]string{
			"name":       aUser.Name,
			"reset_link": fmt.Sprintf("%s/auth/change-password/%s", wAuthApi.cfg.BaseUrl(), token.Token),
		})
	if err != nil {
		wAuthApi.logger.Errorf("sending forgot password email failed with err %v", err)
	}

	return &protos.ForgotPasswordResponse{
		Code:    200,
		Success: true,
	}, nil
}

func (wAuthApi *webAuthGRPCApi) ChangePassword(c context.Context, irRequest *protos.ChangePasswordRequest) (*protos.ChangePasswordResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	userID := principal.Subject().ID
	user, err := wAuthApi.userService.GetUser(c, userID)
	if err != nil {
		return nil, err
	}
	err = wAuthApi.userService.VerifyPassword(c, user.Email, irRequest.GetOldPassword())
	if err != nil {
		return &protos.ChangePasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to change password, please verify current password and try again.",
			}}, nil
	}

	_, err = wAuthApi.userService.UpdatePassword(c, principal, userID, irRequest.GetPassword())
	if err != nil {
		wAuthApi.logger.Errorf("unable to change password for user failed %v", err)
		return &protos.ChangePasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to change password, please try again in sometime.",
			}}, nil
	}
	return &protos.ChangePasswordResponse{
		Code:    200,
		Success: true,
	}, nil
}

func (wAuthApi *webAuthGRPCApi) CreatePassword(c context.Context, irRequest *protos.CreatePasswordRequest) (*protos.CreatePasswordResponse, error) {
	token, err := wAuthApi.userService.GetToken(c, "password-token", irRequest.GetToken())
	if err != nil {
		wAuthApi.logger.Errorf("unable to verify password token for user %v failed %v", irRequest.GetToken(), err)
		return &protos.CreatePasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to verify reset password token, please try again in sometime.",
			}}, nil
	}

	_, err = wAuthApi.userService.UpdatePassword(c, authentication.Principal{}, token.UserAuthId, irRequest.GetPassword())
	if err != nil {
		wAuthApi.logger.Errorf("unable to change password for user failed %v", err)
		return &protos.CreatePasswordResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create reset password token, please try again in sometime.",
			}}, nil
	}
	return &protos.CreatePasswordResponse{
		Code:    200,
		Success: true,
	}, nil
}

func (wAuthApi *webAuthGRPCApi) Authorize(c context.Context, irRequest *protos.AuthorizeRequest) (*protos.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("Authorize from grpc with requestPayload %v, %v", irRequest, c)
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	result, err := wAuthApi.userService.AuthenticationForPrincipal(c, principal)
	if err != nil {
		wAuthApi.logger.Errorf("unable to authorize the user %v", err)
		return nil, err
	}
	return userAuthenticationResponse(result), nil
}

func (wAuthApi *webAuthGRPCApi) VerifyToken(ctx context.Context, request *protos.VerifyTokenRequest) (*protos.VerifyTokenResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	subject := principal.Subject()
	proof := principal.Proof()
	_, delegated := principal.Actor()
	if subject.Type != authentication.IdentityService || delegated || principal.Tenant() != (authentication.Tenant{}) ||
		proof.Method != authentication.AuthenticationServiceAssertion {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	var value authentication.Authentication
	var err error
	credentialType := "unknown"
	switch credential := request.GetCredential().(type) {
	case *protos.VerifyTokenRequest_User:
		credentialType = "user"
		user := credential.User
		if user == nil || !validator.NotBlank(user.Token) || strings.TrimSpace(user.Token) != user.Token ||
			!validator.Between(user.UserId, uint64(1), uint64(math.MaxInt64)) ||
			user.SelectedOrganizationId > uint64(math.MaxInt64) || user.SelectedProjectId > uint64(math.MaxInt64) {
			return nil, status.Error(codes.Unauthenticated, "valid credential is required")
		}
		value, err = wAuthApi.userAuthenticationService.ResolveUserCredential(
			ctx, user.Token, user.UserId, user.SelectedOrganizationId, user.SelectedProjectId,
		)
	case *protos.VerifyTokenRequest_Project:
		credentialType = "project"
		project := credential.Project
		if project == nil || !validator.NotBlank(project.Token) || strings.TrimSpace(project.Token) != project.Token {
			return nil, status.Error(codes.Unauthenticated, "valid credential is required")
		}
		value, err = wAuthApi.projectAuthenticationService.ResolveProjectCredential(ctx, project.Token)
	case *protos.VerifyTokenRequest_Organization:
		credentialType = "organization"
		organization := credential.Organization
		if organization == nil || !validator.NotBlank(organization.Token) || strings.TrimSpace(organization.Token) != organization.Token {
			return nil, status.Error(codes.Unauthenticated, "valid credential is required")
		}
		value, err = wAuthApi.organizationAuthenticationService.ResolveOrganizationCredential(ctx, organization.Token)
	default:
		return nil, status.Error(codes.Unauthenticated, "valid credential is required")
	}
	if err != nil {
		var invalid pkg_errors.AuthenticationCredentialInvalidError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "authentication resolution canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "authentication resolution deadline exceeded")
		case errors.As(err, &invalid):
			return nil, status.Error(codes.Unauthenticated, "valid credential is required")
		default:
			wAuthApi.logger.Errorf("unable to resolve %s authentication: %v", credentialType, err)
			return nil, status.Error(codes.Internal, "unable to resolve authentication")
		}
	}
	return &protos.VerifyTokenResponse{Authentication: value.Proto()}, nil
}

func (wAuthApi *webAuthApi) GetUser(c context.Context, irRequest *protos.GetUserRequest) (*protos.GetUserResponse, error) {
	wAuthApi.logger.Debugf("GetUser from grpc with requestPayload %v, %v", irRequest, c)
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	user, err := wAuthApi.userService.GetUser(c, principal.Subject().ID)
	if err != nil {
		return nil, err
	}

	aUser := &protos.User{}
	utils.Cast(user, aUser)
	return &protos.GetUserResponse{Code: 200, Success: true, Data: aUser}, nil
}

func (wAuthApi *webAuthApi) UpdateUser(c context.Context, irRequest *protos.UpdateUserRequest) (*protos.UpdateUserResponse, error) {
	wAuthApi.logger.Debugf("UpdateUser from grpc with requestPayload %v, %v", irRequest, c)
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	if !utf8.ValidString(irRequest.GetName()) || !validator.NotBlank(irRequest.GetName()) ||
		strings.TrimSpace(irRequest.GetName()) != irRequest.GetName() || strings.ContainsRune(irRequest.GetName(), '\x00') {
		return nil, status.Error(codes.InvalidArgument, "name is invalid")
	}

	user, err := wAuthApi.userService.UpdateUser(c, principal, principal.Subject().ID, irRequest.Name)
	if err != nil {
		return nil, err
	}

	aUser := &protos.User{}
	utils.Cast(user, aUser)
	return &protos.UpdateUserResponse{Code: 200, Success: true, Data: aUser}, nil
}

/**
Oauth implementation block that will help us quickly login and sign up in our system from multiple social accounts

**/

// common way to create social user
func (wAuthApi *webAuthApi) RegisterSocialUser(c context.Context, inf *internal_connects.OpenID) (*protos.AuthenticateResponse, error) {
	if inf == nil {
		return nil, status.Error(codes.InvalidArgument, "name is invalid")
	}
	cUser, err := wAuthApi.userService.Get(c, inf.Email)
	if err != nil {
		if !utf8.ValidString(inf.Name) || !validator.NotBlank(inf.Name) ||
			strings.TrimSpace(inf.Name) != inf.Name || strings.ContainsRune(inf.Name, '\x00') {
			return nil, status.Error(codes.InvalidArgument, "name is invalid")
		}
		aUser, err := wAuthApi.userService.Create(c, authentication.Principal{}, inf.Name, inf.Email, inf.Token, type_enums.RECORD_ACTIVE, &inf.Source)
		if err != nil {
			return &protos.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create the user, please try again in sometime.",
				}}, nil
		}
		//

		// (ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_entities.UserSocial, error)
		_, err = wAuthApi.userService.CreateSocial(c, aUser.Id, inf.Id, inf.Token, inf.Source, inf.Verified)
		if err != nil {
			wAuthApi.logger.Debugf("failed to persist the user social information %v", err)
			return &protos.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create the user, please try again in sometime.",
				}}, nil
		}

		err = wAuthApi.emailerClient.EmailRichText(
			c,
			external_clients.Contact{
				Name:  aUser.Name,
				Email: aUser.Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.Name,
			},
		)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		result, err := wAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			return nil, err
		}
		return userAuthenticationResponse(result), nil
	}

	// if it's invited user then
	if cUser.Status == type_enums.RECORD_INVITED {
		if !utf8.ValidString(inf.Name) || !validator.NotBlank(inf.Name) ||
			strings.TrimSpace(inf.Name) != inf.Name || strings.ContainsRune(inf.Name, '\x00') {
			return nil, status.Error(codes.InvalidArgument, "name is invalid")
		}
		// activate org
		if err = wAuthApi.userService.ActivateAllOrganizationRole(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate project
		if err := wAuthApi.userService.ActivateAllProjectRoles(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		aUser, err := wAuthApi.userService.Activate(c, cUser.Id, inf.Name, &inf.Source)
		if err != nil {
			wAuthApi.logger.Debugf("failed to activate the user")
			return &protos.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Failed to activate the user, please check with your organization admin.",
				}}, nil
		}

		_, err = wAuthApi.userService.CreateSocial(c, aUser.Id, inf.Id, inf.Token, inf.Source, inf.Verified)
		if err != nil {
			wAuthApi.logger.Debugf("failed to persist the user social information")
			return &protos.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Failed to activate the user, please check with your organization admin.",
				}}, nil
		}

		result, err := wAuthApi.userService.CreateAuthentication(c, aUser.Id)
		if err != nil {
			return nil, err
		}
		return userAuthenticationResponse(result), nil
	}

	// it might be social login
	if cUser.Status == type_enums.RECORD_ACTIVE {
		// check
		_, err := wAuthApi.userService.GetSocial(c, cUser.Id)
		if err != nil {
			_, err = wAuthApi.userService.CreateSocial(c, cUser.Id, inf.Id, inf.Token, inf.Source, inf.Verified)
			if err != nil {
				return &protos.AuthenticateResponse{
					Code:    400,
					Success: false,
					Error: &protos.Error{
						ErrorCode:    400,
						ErrorMessage: err.Error(),
						HumanMessage: "Unable to persist the social informaiton, please try again later.",
					}}, nil
			}
		}
		result, err := wAuthApi.userService.CreateAuthentication(c, cUser.Id)
		if err != nil {
			return nil, err
		}
		return userAuthenticationResponse(result), nil
	}
	return nil, errors.New("you are already registered, please use the existing method to signin")
}

/**

Github
*/

func (wAuthApi *webAuthRPCApi) Github(c *gin.Context) {
	url, err := wAuthApi.githubConnect.AuthCodeURL("github")
	if err != nil {
		c.Error(errors.New("github oauth2 is not enabled"))
		return
	}
	wAuthApi.logger.Debugf("url generated %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (wAuthApi *webAuthGRPCApi) Github(c context.Context, irRequest *protos.SocialAuthenticationRequest) (*protos.AuthenticateResponse, error) {
	inf, err := wAuthApi.githubConnect.GithubUserInfo(c, irRequest.State, irRequest.Code)
	if err != nil {
		wAuthApi.logger.Errorf("github authentication response %v", err)
		return nil, err
	}
	return wAuthApi.RegisterSocialUser(c, inf)
}

func (wAuthApi *webAuthRPCApi) Linkedin(c *gin.Context) {
	url, err := wAuthApi.linkedinConnect.AuthCodeURL("linkedin")
	if err != nil {
		c.Error(errors.New("linkedin oauth2 is not enabled"))
		return
	}
	wAuthApi.logger.Debugf("generated redirect url for linkedin %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}
func (wAuthApi *webAuthGRPCApi) Linkedin(c context.Context, irRequest *protos.SocialAuthenticationRequest) (*protos.AuthenticateResponse, error) {
	inf, err := wAuthApi.linkedinConnect.LinkedinUserInfo(c, irRequest.State, irRequest.Code)
	if err != nil {
		wAuthApi.logger.Errorf("linkedin authentication response %v", err)
		return nil, err
	}
	return wAuthApi.RegisterSocialUser(c, inf)
}

// Google
func (wAuthApi *webAuthRPCApi) Google(c *gin.Context) {
	url, err := wAuthApi.googleConnect.AuthCodeURL(GOOGLE_STATE)
	if err != nil {
		c.Error(errors.New("google oauth2 is not enabled"))
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}
func (wAuthApi *webAuthGRPCApi) Google(c context.Context, irRequest *protos.SocialAuthenticationRequest) (*protos.AuthenticateResponse, error) {
	if GOOGLE_STATE != irRequest.State {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", GOOGLE_STATE, irRequest.State)
		return nil, fmt.Errorf("invalid oauth state")
	}
	inf, err := wAuthApi.googleConnect.GoogleUserInfo(c, irRequest.Code)
	wAuthApi.logger.Debugf("google authenticator respose %v", inf)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication response %v", err)
		return nil, err
	}
	return wAuthApi.RegisterSocialUser(c, inf)
}

func (wAuthApi *webAuthGRPCApi) GetAllUser(c context.Context, irRequest *protos.GetAllUserRequest) (*protos.GetAllUserResponse, error) {
	principal, ok := middleware_authentication.Principal(c)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}

	cnt, allMembers, err := wAuthApi.userService.GetAllOrganizationMember(c,
		principal.Tenant().OrganizationID,
		irRequest.GetCriterias(),
		irRequest.GetPaginate(),
	)
	if err != nil {
		wAuthApi.logger.Errorf("getUsers from grpc with requestPayload %v, %v", irRequest, c)
		return &protos.GetAllUserResponse{
			Code:    400,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to get all the users for the organization, please try again in sometime.",
			}}, nil
	}

	membersByID := make(map[uint64]*protos.User, len(*allMembers))
	memberIDs := make([]uint64, 0, len(*allMembers))
	for _, member := range *allMembers {
		projected, exists := membersByID[member.UserAuthId]
		if !exists {
			projected = &protos.User{
				Name:        member.Member.Name,
				Id:          member.Member.Id,
				Email:       member.Member.Email,
				Status:      member.Status.String(),
				CreatedDate: timestamppb.New(time.Time(member.Member.CreatedDate)),
			}
			membersByID[member.UserAuthId] = projected
			memberIDs = append(memberIDs, member.UserAuthId)
		}
		projected.OrganizationRoles = append(projected.OrganizationRoles, &protos.AssignedRole{
			RoleDefinitionId: member.RoleDefinitionID,
			Key:              member.RoleDefinition.Key,
			Name:             member.RoleDefinition.Name,
		})
	}
	out := make([]*protos.User, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		out = append(out, membersByID[memberID])
	}
	totalItem, err := utils.Int64ToUint32(cnt)
	if err != nil {
		wAuthApi.logger.Errorf("invalid organization member count %d: %v", cnt, err)
		return utils.Error[protos.GetAllUserResponse](
			err,
			"Unable to get all the users for the organization, please try again in sometime.",
		)
	}
	return &protos.GetAllUserResponse{
		Code:    200,
		Success: true,
		Data:    out,
		Paginated: &protos.Paginated{
			TotalItem:   totalItem,
			CurrentPage: irRequest.GetPaginate().GetPage(),
		},
	}, nil
}
