package web_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	internal_clients "github.com/lexatic/web-backend/internal/clients"
	integration_client "github.com/lexatic/web-backend/internal/clients/integration"
	internal_organization_service "github.com/lexatic/web-backend/internal/services/organization"
	internal_project_service "github.com/lexatic/web-backend/internal/services/project"

	"github.com/gin-gonic/gin"
	config "github.com/lexatic/web-backend/config"
	internal_services "github.com/lexatic/web-backend/internal/services"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/linkedin"
)

type webAuthApi struct {
	cfg                 *config.AppConfig
	logger              commons.Logger
	postgres            connectors.PostgresConnector
	userService         internal_services.UserService
	organizationService internal_services.OrganizationService
	projectService      internal_services.ProjectService
	integrationClient   internal_clients.IntegrationServiceClient
	googleOauthConfig   oauth2.Config
	linkedinOauthConfig oauth2.Config
	githubOauthConfig   oauth2.Config
}

type webAuthRPCApi struct {
	webAuthApi
}

type webAuthGRPCApi struct {
	webAuthApi
}

func NewAuthRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) *webAuthRPCApi {
	return &webAuthRPCApi{
		webAuthApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			userService:         internal_user_service.NewUserService(logger, postgres),
			integrationClient:   integration_client.NewIntegrationServiceClientGRPC(config, logger),
			googleOauthConfig:   GoogleOAuth(config),
			linkedinOauthConfig: LinkedinOAuth(config),
			githubOauthConfig:   GithubOAuth(config),
		},
	}
}

func NewAuthGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.AuthenticationServiceServer {
	return &webAuthGRPCApi{
		webAuthApi{
			cfg:                 config,
			logger:              logger,
			postgres:            postgres,
			userService:         internal_user_service.NewUserService(logger, postgres),
			organizationService: internal_organization_service.NewOrganizationService(logger, postgres),
			projectService:      internal_project_service.NewProjectService(logger, postgres),
			integrationClient:   integration_client.NewIntegrationServiceClientGRPC(config, logger),
			googleOauthConfig:   GoogleOAuth(config),
			linkedinOauthConfig: LinkedinOAuth(config),
			githubOauthConfig:   GithubOAuth(config),
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

	aUser, err := wAuthApi.userService.Authenticate(c, irRequest.Email, irRequest.Password)
	if err != nil {
		c.JSON(500, commons.Response{
			Code:    500,
			Success: false,
			Data:    commons.ErrorMessage{Code: 100, Message: err},
		})
		return
	}
	c.JSON(200, commons.Response{
		Code:    200,
		Success: true,
		Data:    aUser.PlainAuthPrinciple(),
	})
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
		aUser, err := webAuthApi.userService.Create(c, irRequest.Name, irRequest.Email, irRequest.Password, "active", &source)
		if err != nil {
			webAuthApi.logger.Errorf("registering new user failed with err %v", err)
			c.JSON(500, commons.Response{
				Code:    500,
				Success: false,
				Data:    commons.ErrorMessage{Code: 100, Message: err},
			})
			return
		}

		_, err = webAuthApi.integrationClient.WelcomeEmail(c, aUser.GetUserInfo().Id, aUser.GetUserInfo().Name, aUser.GetUserInfo().Email)
		if err != nil {
			webAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		webAuthApi.logger.Debugf("user after creation %v", aUser)
		c.JSON(200, commons.Response{
			Code:    200,
			Success: true,
			Data:    aUser.PlainAuthPrinciple(),
		})
		return

	}

	// if it's invited user then
	if cUser.Status == "invited" {

		// password need to fix
		_, err := webAuthApi.userService.UpdatePassword(c, cUser.Id, irRequest.Password)
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
		c.JSON(200, commons.Response{
			Code:    200,
			Success: true,
			Data:    aUser,
		})
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
func (wAuthApi *webAuthGRPCApi) Authenticate(c context.Context, irRequest *web_api.AuthenticateRequest) (*web_api.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("Authenticate from grpc with requestPayload %v, %v", irRequest, c)

	aUser, err := wAuthApi.userService.Authenticate(c, irRequest.Email, irRequest.Password)
	if err != nil {
		wAuthApi.logger.Errorf("unable to process authentication %v", err)
		wAuthApi.logger.Debugf("authentication request failed for user %s", irRequest.Email)
		return &web_api.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    401,
				ErrorMessage: err.Error(),
				HumanMessage: "Please provide valid credentials to signin into account.",
			}}, nil
	}

	/**
	As we support multiple state of user to register
	only active user will able to signin to the rapida account
	*/
	if aUser.GetUserInfo().Status != "active" {
		wAuthApi.logger.Errorf("unable to process authentication because of status of the user status %v", aUser.GetUserInfo().Status)
		return &web_api.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your account is not activated yet. please activate before signin.",
			}}, nil
	}
	auth := &web_api.Authentication{}
	types.Cast(aUser.PlainAuthPrinciple(), auth)
	return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
}

/*
Register or activate a user to authenticate into the rapida platform
will be streamlining the code for better managing and expalining later
*/
func (wAuthApi *webAuthGRPCApi) RegisterUser(c context.Context, irRequest *web_api.RegisterUserRequest) (*web_api.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("RegisterUser from grpc with requestPayload %v, %v", irRequest, c)
	cUser, err := wAuthApi.userService.Get(c, irRequest.Email)
	source := "direct"
	if err != nil {
		aUser, err := wAuthApi.userService.Create(c, irRequest.Name, irRequest.Email, irRequest.Password, "active", &source)
		if err != nil {
			wAuthApi.logger.Errorf("creation user failed with err %v", err)
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create an account, please check and try again.",
				}}, nil
		}
		_, err = wAuthApi.integrationClient.WelcomeEmail(c, aUser.GetUserInfo().Id, aUser.GetUserInfo().Name, aUser.GetUserInfo().Email)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}

		auth := &web_api.Authentication{}
		types.Cast(aUser.PlainAuthPrinciple(), auth)
		return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
	}

	// already have an active account
	if cUser.Status == "active" {
		wAuthApi.logger.Errorf("user is already having account and trying to signup")
		return &web_api.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your email is already associated with an existing account, try signin.",
			}}, nil

	}
	// if it's invited user then
	if cUser.Status == "invited" {
		_, err := wAuthApi.userService.UpdatePassword(c, cUser.Id, irRequest.Password)
		if err != nil {
			wAuthApi.logger.Errorf("Error while updaing password for user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate org
		if err = wAuthApi.userService.ActivateAllOrganizationRole(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate project
		if err := wAuthApi.userService.ActivateAllProjectRoles(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		// activate user
		aUser, err := wAuthApi.userService.Activate(c, cUser.Id, irRequest.Name, nil)
		if err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		auth := &web_api.Authentication{}
		err = types.Cast(aUser.PlainAuthPrinciple(), auth)
		if err != nil {
			wAuthApi.logger.Errorf("Error while unmarshelling user error %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil

		}
		return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
	}

	return &web_api.AuthenticateResponse{Code: 400, Success: false, Error: &web_api.AuthenticationError{
		ErrorCode:    400,
		ErrorMessage: "illegal state of data",
		HumanMessage: "We are facing issue with account creation, please try again in sometime",
	}}, err
}

func (wAuthApi *webAuthGRPCApi) ForgotPassword(c context.Context, irRequest *web_api.ForgotPasswordRequest) (*web_api.ForgotPasswordResponse, error) {
	wAuthApi.logger.Debugf("ForgotPassword from grpc with requestPayload %v, %v", irRequest.String(), c)

	aUser, err := wAuthApi.userService.Get(c, irRequest.GetEmail())
	if err != nil {
		wAuthApi.logger.Errorf("getting email for forgot password for user %v failed %v", irRequest.GetEmail(), err)
		return &web_api.ForgotPasswordResponse{
			Code:    400,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Your email is not associated with rapida.ai account, please check and try again",
			}}, nil
	}

	if aUser.Status != "active" {
		wAuthApi.logger.Errorf("user is changing password for not activated user  %v", aUser.Email)
		return &web_api.ForgotPasswordResponse{
			Code:    401,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your account is not activated yet. please activate before signin.",
			}}, nil
	}

	token, err := wAuthApi.userService.CreatePasswordToken(c, aUser.Id)
	if err != nil {
		wAuthApi.logger.Errorf("unable to create password token for user %v failed %v", irRequest.GetEmail(), err)
		return &web_api.ForgotPasswordResponse{
			Code:    400,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create reset password token, please try again in sometime.",
			}}, nil
	}
	// userId uint64, name, email,
	resetPasswordUrl := fmt.Sprintf("https://rapida.ai/auth/change-password/%s", token.Token)
	_, err = wAuthApi.integrationClient.ResetPasswordEmail(c, aUser.Id,
		aUser.Name, aUser.Email,
		resetPasswordUrl)
	wAuthApi.logger.Debugf("reset password link created %v", resetPasswordUrl)
	if err != nil {
		wAuthApi.logger.Errorf("sending forgot password email failed with err %v", err)
	}

	return &web_api.ForgotPasswordResponse{
		Code:    200,
		Success: true,
	}, nil

}

func (wAuthApi *webAuthGRPCApi) CreatePassword(c context.Context, irRequest *web_api.CreatePasswordRequest) (*web_api.CreatePasswordResponse, error) {
	wAuthApi.logger.Debugf("ChangePassword from grpc with requestPayload %v, %v", irRequest, c)
	// CreateToken(ctx context.Context, userId uint64) (*internal_gorm.UserAuthToken, error)
	// wAuthApi.userService.Get(c, irRe)
	token, err := wAuthApi.userService.GetToken(c, "password-token", irRequest.GetToken())
	if err != nil {
		wAuthApi.logger.Errorf("unable to verify password token for user %v failed %v", irRequest.GetToken(), err)
		return &web_api.CreatePasswordResponse{
			Code:    400,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to verify reset password token, please try again in sometime.",
			}}, nil
	}

	_, err = wAuthApi.userService.UpdatePassword(c, token.UserAuthId, irRequest.Password)
	if err != nil {
		wAuthApi.logger.Errorf("unable to change password for user failed %v", err)
		return &web_api.CreatePasswordResponse{
			Code:    400,
			Success: false,
			Error: &web_api.AuthenticationError{
				ErrorCode:    400,
				ErrorMessage: err.Error(),
				HumanMessage: "Unable to create reset password token, please try again in sometime.",
			}}, nil
	}
	return &web_api.CreatePasswordResponse{
		Code:    200,
		Success: true,
	}, nil

}

func (wAuthApi *webAuthGRPCApi) Authorize(c context.Context, irRequest *web_api.AuthorizeRequest) (*web_api.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("Authorize from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}
	aUser, err := wAuthApi.userService.AuthPrinciple(c, iAuth.GetUserInfo().Id)
	if err != nil {
		wAuthApi.logger.Errorf("unable to authorize the user %v", err)
		return nil, err
	}
	auth := &web_api.Authentication{}
	types.Cast(aUser.PlainAuthPrinciple(), auth)
	return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
}

func (wAuthApi *webAuthApi) VerifyToken(c context.Context, irRequest *web_api.VerifyTokenRequest) (*web_api.VerifyTokenResponse, error) {
	wAuthApi.logger.Debugf("VerifyToken from grpc with requestPayload %v, %v", irRequest, c)
	token, err := wAuthApi.userService.GetToken(c, irRequest.GetTokenType(), irRequest.GetToken())
	if err != nil {
		return nil, err
	}

	aToken := &web_api.Token{}
	types.Cast(token, aToken)
	return &web_api.VerifyTokenResponse{Code: 200, Success: true, Data: aToken}, nil

}

func (wAuthApi *webAuthApi) GetUser(c context.Context, irRequest *web_api.GetUserRequest) (*web_api.GetUserResponse, error) {
	wAuthApi.logger.Debugf("GetUser from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}

	user, err := wAuthApi.userService.GetUser(c, iAuth.GetUserInfo().Id)
	if err != nil {
		return nil, err
	}

	aUser := &web_api.UserInfo{}
	types.Cast(user, aUser)
	return &web_api.GetUserResponse{Code: 200, Success: true, Data: aUser}, nil

}

func (wAuthApi *webAuthApi) UpdateUser(c context.Context, irRequest *web_api.UpdateUserRequest) (*web_api.UpdateUserResponse, error) {
	wAuthApi.logger.Debugf("UpdateUser from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}

	if strings.TrimSpace(irRequest.GetName()) == "" {
		return nil, errors.New("cannot give an empty name")
	}

	user, err := wAuthApi.userService.UpdateUser(c, iAuth, iAuth.GetUserInfo().Id, irRequest.Name)
	if err != nil {
		return nil, err
	}

	aUser := &web_api.UserInfo{}
	types.Cast(user, aUser)
	return &web_api.UpdateUserResponse{Code: 200, Success: true, Data: aUser}, nil
}

/**
Oauth implimentation block that will help us quickly login and sign up in our system from multiple social accounts

**/

func GithubOAuth(cfg *config.AppConfig) oauth2.Config {
	return oauth2.Config{
		RedirectURL:  "https://www.rapida.ai/auth/signin",
		ClientID:     cfg.GithubClientId,
		ClientSecret: cfg.GithubClientSecret,
		Scopes:       []string{"user"},
		Endpoint:     github.Endpoint,
	}
}

func LinkedinOAuth(cfg *config.AppConfig) oauth2.Config {
	return oauth2.Config{
		RedirectURL:  "https://www.rapida.ai/auth/signin",
		ClientID:     cfg.LinkedinClientId,
		ClientSecret: cfg.LinkedinClientSecret,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     linkedin.Endpoint,
	}
}

func GoogleOAuth(cfg *config.AppConfig) oauth2.Config {
	return oauth2.Config{
		RedirectURL:  "https://www.rapida.ai/auth/signin",
		ClientID:     cfg.GoogleClientId,
		ClientSecret: cfg.GoogleClientSecret,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

type OpenID struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Verified bool   `json:"verified_email"`
	Id       string `json:"id"`
	Source   string `json:"source"`
	Token    string `json:"token"`
}

// common way to create social user
func (wAuthApi *webAuthApi) RegisterSocialUser(c context.Context, inf *OpenID) (*web_api.AuthenticateResponse, error) {
	cUser, err := wAuthApi.userService.Get(c, inf.Email)
	if err != nil {
		aUser, err := wAuthApi.userService.Create(c, inf.Name, inf.Email, inf.Token, "active", &inf.Source)
		if err != nil {
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create the user, please try again in sometime.",
				}}, nil
		}
		//

		// (ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_gorm.UserSocial, error)
		_, err = wAuthApi.userService.CreateSocial(c, aUser.GetUserInfo().Id, inf.Id, inf.Token, inf.Source, inf.Verified)
		if err != nil {
			wAuthApi.logger.Debugf("failed to persist the user social information %v", err)
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to create the user, please try again in sometime.",
				}}, nil
		}

		_, err = wAuthApi.integrationClient.WelcomeEmail(c, aUser.GetUserInfo().Id, aUser.GetUserInfo().Name, aUser.GetUserInfo().Email)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}

		auth := &web_api.Authentication{}
		types.Cast(aUser.PlainAuthPrinciple(), auth)
		return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
	}

	// if it's invited user then
	if cUser.Status == "invited" {

		// activate org
		if err = wAuthApi.userService.ActivateAllOrganizationRole(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		// activate project
		if err := wAuthApi.userService.ActivateAllProjectRoles(c, cUser.Id); err != nil {
			wAuthApi.logger.Errorf("Error while registering user %v", err)
			return &web_api.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}

		aUser, err := wAuthApi.userService.Activate(c, cUser.Id, inf.Name, &inf.Source)
		if err != nil {
			wAuthApi.logger.Debugf("failed to activate the user")
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Failed to activate the user, please check with your organization admin.",
				}}, nil
		}

		_, err = wAuthApi.userService.CreateSocial(c, aUser.GetUserInfo().Id, inf.Id, inf.Token, inf.Source, inf.Verified)
		if err != nil {
			wAuthApi.logger.Debugf("failed to persist the user social information")
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Failed to activate the user, please check with your organization admin.",
				}}, nil
		}

		auth := &web_api.Authentication{}
		types.Cast(aUser.PlainAuthPrinciple(), auth)
		return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
	}

	// it might be social login
	if cUser.Status == "active" {
		// check
		currentSocial, err := wAuthApi.userService.GetSocial(c, cUser.Id)
		if err != nil {
			return &web_api.AuthenticateResponse{
				Code:    400,
				Success: false,
				Error: &web_api.AuthenticationError{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "You have already sign up with other source. please retry with existing source.",
				}}, nil
		}
		if currentSocial.Social == inf.Source {
			aUser, err := wAuthApi.userService.AuthPrinciple(c, cUser.Id)
			if err != nil {
				wAuthApi.logger.Debugf("failed to get auth principle %v", err)
				return nil, err
			}
			auth := &web_api.Authentication{}
			types.Cast(aUser.PlainAuthPrinciple(), auth)
			return &web_api.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
		}
	}

	return nil, errors.New("you are already registered, please use the existing method to signin")
}

/*
*

Google implimentation
*/

func (wAuthApi *webAuthRPCApi) Google(c *gin.Context) {
	url := wAuthApi.googleOauthConfig.AuthCodeURL("google")
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}
func (wAuthApi *webAuthGRPCApi) Google(c context.Context, irRequest *web_api.SocialAuthenticationRequest) (*web_api.AuthenticateResponse, error) {
	inf, err := wAuthApi.GoogleUserInfo(c, irRequest.State, irRequest.Code)
	wAuthApi.logger.Debugf("google authenticator respose %v", inf)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication response %v", err)
		return nil, err
	}
	return wAuthApi.RegisterSocialUser(c, inf)
}
func (wAuthApi *webAuthGRPCApi) GoogleUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "google" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "google", state)
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := wAuthApi.googleOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication exchange failed %v", err)
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		wAuthApi.logger.Errorf("unable to get userinfo using the access token %v", err)
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()

	var content OpenID
	err = json.NewDecoder(response.Body).Decode(&content)
	content.Source = "google"
	content.Token = token.AccessToken
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode the response body of the user info %v", err)
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}
	return &content, nil
}

/**

Linkedin oauth
*/

func (wAuthApi *webAuthRPCApi) Linkedin(c *gin.Context) {
	url := wAuthApi.linkedinOauthConfig.AuthCodeURL("linkedin")
	wAuthApi.logger.Debugf("generated redirect url for linkedin %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}
func (wAuthApi *webAuthGRPCApi) Linkedin(c context.Context, irRequest *web_api.SocialAuthenticationRequest) (*web_api.AuthenticateResponse, error) {
	inf, err := wAuthApi.LinkedinUserInfo(c, irRequest.State, irRequest.Code)
	wAuthApi.logger.Debugf("linkedin authenticator respose %v", inf)
	if err != nil {
		wAuthApi.logger.Errorf("google authentication response %v", err)
		return nil, err
	}

	return wAuthApi.RegisterSocialUser(c, inf)
}
func (wAuthApi *webAuthGRPCApi) LinkedinUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "linkedin" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "linkedin", state)
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := wAuthApi.linkedinOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("unable to exchange the token from linkedin %v", err)
		return nil, err
	}

	client := wAuthApi.linkedinOauthConfig.Client(c, token)
	req, err := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		wAuthApi.logger.Errorf("error while creating request %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	response, err := client.Do(req)
	if err != nil {
		wAuthApi.logger.Errorf("error while getting user from linkedin %v", err)
		return nil, err
	}

	defer response.Body.Close()
	// // {"email":"p_srivastav@outlook.com","email_verified":true,"family_name":"Srivastav","given_name":"Prashant","locale":{"country":"US","language":"en"},"name":"Prashant Srivastav","picture":"https://media.licdn.com/dms/image/C5603AQGslsdJ_ZIoMA/profile-displayphoto-shrink_100_100/0/1659118454695?e=1706745600\u0026v=beta\u0026t=8NmYbyO4c6gd3Y1MQjs4LZ3cmh6tYU9zc9Ghlg3FAQ0","sub":"XyBk2_14Uj"}
	var content map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&content)
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode %v", err)
		return nil, err
	}
	return &OpenID{
		Token: token.AccessToken, Source: "linkedin",
		Email:    content["email"].(string),
		Verified: content["email_verified"].(bool),
		Name:     content["name"].(string),
		Id:       content["sub"].(string),
	}, nil
}

/**

Github
*/

func (wAuthApi *webAuthRPCApi) Github(c *gin.Context) {
	url := wAuthApi.githubOauthConfig.AuthCodeURL("github")
	wAuthApi.logger.Debugf("url generated %v", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
	return
}

func (wAuthApi *webAuthGRPCApi) Github(c context.Context, irRequest *web_api.SocialAuthenticationRequest) (*web_api.AuthenticateResponse, error) {
	inf, err := wAuthApi.GithubUserInfo(c, irRequest.State, irRequest.Code)
	wAuthApi.logger.Debugf("github authenticator respose %v", inf)
	if err != nil {
		wAuthApi.logger.Errorf("github authentication response %v", err)
		return nil, err
	}

	return wAuthApi.RegisterSocialUser(c, inf)
}

func (wAuthApi *webAuthGRPCApi) GithubUserInfo(c context.Context, state string, code string) (*OpenID, error) {
	if state != "github" {
		wAuthApi.logger.Errorf("illegal oauth request as auth state is not matching %s %s", "github", state)
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := wAuthApi.githubOauthConfig.Exchange(c, code)
	if err != nil {
		wAuthApi.logger.Errorf("unable to exchange the token from github %v", err)
		return nil, err
	}

	oauthClient := wAuthApi.githubOauthConfig.Client(c, token)
	req, err := http.NewRequest("POST", "https://api.github.com/user", nil)
	if err != nil {
		wAuthApi.logger.Errorf("error while creating request %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	response, err := oauthClient.Do(req)
	if err != nil {
		wAuthApi.logger.Errorf("error while getting user from linkedin %v", err)
		return nil, err
	}

	defer response.Body.Close()
	var content map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&content)
	if err != nil {
		wAuthApi.logger.Errorf("unable to decode %v", err)
		return nil, err
	}
	return &OpenID{
		Token: token.AccessToken, Source: "github",
		Email:    content["email"].(string),
		Verified: true,
		Name:     content["name"].(string),
		Id:       fmt.Sprintf("%f", content["id"].(float64)),
	}, nil
}

func (wAuthApi *webAuthGRPCApi) GetUsers(c context.Context, irRequest *web_api.GetUsersRequest) (*web_api.GetUsersResponse, error) {
	wAuthApi.logger.Debugf("GetUsers from grpc with requestPayload %v, %v", irRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		return nil, errors.New("unauthenticated request")
	}

	allMembers, err := wAuthApi.userService.GetAllOrganizationMember(c, iAuth.GetOrganizationRole().OrganizationId)
	if err != nil {
		wAuthApi.logger.Errorf("getUsers from grpc with requestPayload %v, %v", irRequest, c)
		return nil, err
	}

	out := make([]*web_api.User, len(*allMembers))
	for ix, member := range *allMembers {
		out[ix] = &web_api.User{
			Name:        member.Member.Name,
			Id:          member.Member.Id,
			Email:       member.Member.Email,
			CreatedDate: timestamppb.New(member.Member.CreatedDate),
			OrganizationRole: &web_api.OrganizationRole{
				Id:               iAuth.GetOrganizationRole().Id,
				OrganizationId:   iAuth.GetOrganizationRole().OrganizationId,
				OrganizationName: iAuth.GetOrganizationRole().OrganizationName,
				Role:             member.Role,
			},
		}
	}
	return &web_api.GetUsersResponse{
		Code:       200,
		Success:    true,
		Users:      out,
		TotalCount: uint64(len(*allMembers)),
	}, nil
}
