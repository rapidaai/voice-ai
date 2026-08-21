package web_api

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	internal_connects "github.com/rapidaai/api/web-api/internal/connect"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"

	"github.com/gin-gonic/gin"

	config "github.com/rapidaai/api/web-api/config"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer "github.com/rapidaai/pkg/clients/external/emailer"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
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
}

var (
	GOOGLE_STATE = "google"
)

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

func NewAuthGRPC(config *config.WebAppConfig, oauthCfg *config.OAuth2Config, logger commons.Logger, postgres connectors.PostgresConnector) protos.AuthenticationServiceServer {
	return &webAuthGRPCApi{
		webAuthApi{
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
		aUser, err := webAuthApi.userService.Create(c, irRequest.Name, irRequest.Email, irRequest.Password, type_enums.RECORD_ACTIVE, &source)
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
				Name:  aUser.GetUserInfo().Name,
				Email: aUser.GetUserInfo().Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.GetUserInfo().Name,
			},
		)
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
	if cUser.Status == type_enums.RECORD_INVITED {
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
func (wAuthApi *webAuthGRPCApi) Authenticate(c context.Context, irRequest *protos.AuthenticateRequest) (*protos.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("Authenticate from grpc with requestPayload %v, %v", irRequest, c)

	aUser, err := wAuthApi.userService.Authenticate(c, irRequest.Email, irRequest.GetPassword())
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

	/**
	As we support multiple state of user to register
	only active user will able to signin to the rapida account
	*/
	if aUser.GetUserInfo().Status != string(type_enums.RECORD_ACTIVE) {
		wAuthApi.logger.Errorf("unable to process authentication because of status of the user status %v", aUser.GetUserInfo().Status)
		return &protos.AuthenticateResponse{
			Code:    401,
			Success: false,
			Error: &protos.Error{
				ErrorCode:    400,
				ErrorMessage: "illegal user status",
				HumanMessage: "Your account is not activated yet. please activate before signin.",
			}}, nil
	}
	responseAuth := &protos.Authentication{}
	utils.Cast(aUser.PlainAuthPrinciple(), responseAuth)
	return &protos.AuthenticateResponse{Code: 200, Success: true, Data: responseAuth}, nil
}

/*
Register or activate a user to authenticate into the rapida platform
will be streamlining the code for better managing and explaining later
*/
func (wAuthApi *webAuthGRPCApi) RegisterUser(c context.Context, irRequest *protos.RegisterUserRequest) (*protos.AuthenticateResponse, error) {
	wAuthApi.logger.Debugf("RegisterUser from grpc with requestPayload %v, %v", irRequest, c)
	cUser, err := wAuthApi.userService.Get(c, irRequest.Email)
	source := "direct"
	if err != nil {
		aUser, err := wAuthApi.userService.Create(c, irRequest.Name, irRequest.Email, irRequest.GetPassword(), type_enums.RECORD_ACTIVE, &source)
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
				Name:  aUser.GetUserInfo().Name,
				Email: aUser.GetUserInfo().Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.GetUserInfo().Name,
			},
		)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		auth := &protos.Authentication{}
		utils.Cast(aUser.PlainAuthPrinciple(), auth)
		return &protos.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
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
		_, err := wAuthApi.userService.UpdatePassword(c, cUser.Id, irRequest.GetPassword())
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

		auth := &protos.Authentication{}
		err = utils.Cast(aUser.PlainAuthPrinciple(), auth)
		if err != nil {
			wAuthApi.logger.Errorf("Error while unmarshelling user error %v", err)
			return &protos.AuthenticateResponse{
				Code:    401,
				Success: false,
				Error: &protos.Error{
					ErrorCode:    400,
					ErrorMessage: err.Error(),
					HumanMessage: "Unable to activate your account, please try again later.",
				}}, nil
		}
		return &protos.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
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
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	user, err := wAuthApi.userService.GetUser(c, userContext.UserID)
	if err != nil {
		return nil, err
	}
	_, err = wAuthApi.userService.Authenticate(c, user.Email, irRequest.GetOldPassword())
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

	_, err = wAuthApi.userService.UpdatePassword(c, userContext.UserID, irRequest.GetPassword())
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

	_, err = wAuthApi.userService.UpdatePassword(c, token.UserAuthId, irRequest.GetPassword())
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
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	aUser, err := wAuthApi.userService.AuthPrinciple(c, userContext.UserID)
	if err != nil {
		wAuthApi.logger.Errorf("unable to authorize the user %v", err)
		return nil, err
	}
	responseAuth := &protos.Authentication{}
	utils.Cast(aUser.PlainAuthPrinciple(), responseAuth)
	return &protos.AuthenticateResponse{Code: 200, Success: true, Data: responseAuth}, nil
}

func (wAuthApi *webAuthGRPCApi) ScopeAuthorize(c context.Context, irRequest *protos.ScopeAuthorizeRequest) (*protos.ScopedAuthenticationResponse, error) {
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	response := &protos.ScopedAuthentication{Status: type_enums.RECORD_ACTIVE.String()}
	var iAuth *types.Authentication
	var scopeErr error
	if irRequest.GetScope() == "project" {
		iAuth, scopeErr = auth.Scope(types.AuthTypeProject)
	} else {
		iAuth, scopeErr = auth.Scope(types.AuthTypeOrg)
	}
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	if irRequest.GetScope() == "project" {
		projectContext, err := iAuth.ProjectContext()
		if err != nil {
			return nil, err
		}
		response.OrganizationId = projectContext.OrganizationID
		response.ProjectId = projectContext.ProjectID
	} else {
		organizationContext, err := iAuth.OrganizationContext()
		if err != nil {
			return nil, err
		}
		response.OrganizationId = organizationContext.OrganizationID
	}
	if actor, actorErr := iAuth.Actor(); actorErr == nil {
		actorType := string(actor.Type)
		response.ActorType = &actorType
		response.ActorId = &actor.ID
	}
	return &protos.ScopedAuthenticationResponse{Code: 200, Success: true, Data: response}, nil
}

func (wAuthApi *webAuthApi) VerifyToken(c context.Context, irRequest *protos.VerifyTokenRequest) (*protos.VerifyTokenResponse, error) {
	wAuthApi.logger.Debugf("VerifyToken from grpc with requestPayload %v, %v", irRequest, c)
	token, err := wAuthApi.userService.GetToken(c, irRequest.GetTokenType(), irRequest.GetToken())
	if err != nil {
		return nil, err
	}

	aToken := &protos.Token{}
	utils.Cast(token, aToken)
	return &protos.VerifyTokenResponse{Code: 200, Success: true, Data: aToken}, nil
}

func (wAuthApi *webAuthApi) GetUser(c context.Context, irRequest *protos.GetUserRequest) (*protos.GetUserResponse, error) {
	wAuthApi.logger.Debugf("GetUser from grpc with requestPayload %v, %v", irRequest, c)
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	user, err := wAuthApi.userService.GetUser(c, userContext.UserID)
	if err != nil {
		return nil, err
	}

	aUser := &protos.User{}
	utils.Cast(user, aUser)
	return &protos.GetUserResponse{Code: 200, Success: true, Data: aUser}, nil
}

func (wAuthApi *webAuthApi) UpdateUser(c context.Context, irRequest *protos.UpdateUserRequest) (*protos.UpdateUserResponse, error) {
	wAuthApi.logger.Debugf("UpdateUser from grpc with requestPayload %v, %v", irRequest, c)
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	if strings.TrimSpace(irRequest.GetName()) == "" {
		return nil, errors.New("cannot give an empty name")
	}

	userContext, err := iAuth.UserContext()
	if err != nil {
		return nil, err
	}
	user, err := wAuthApi.userService.UpdateUser(c, iAuth, userContext.UserID, irRequest.Name)
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
	cUser, err := wAuthApi.userService.Get(c, inf.Email)
	if err != nil {
		aUser, err := wAuthApi.userService.Create(c, inf.Name, inf.Email, inf.Token, type_enums.RECORD_ACTIVE, &inf.Source)
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
		_, err = wAuthApi.userService.CreateSocial(c, aUser.GetUserInfo().Id, inf.Id, inf.Token, inf.Source, inf.Verified)
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
				Name:  aUser.GetUserInfo().Name,
				Email: aUser.GetUserInfo().Email,
			},
			"Welcome to Rapida!",
			external_emailer_template.WELCOME_MEMBER_TEMPLATE,
			map[string]string{
				"name": aUser.GetUserInfo().Name,
			},
		)
		if err != nil {
			wAuthApi.logger.Errorf("sending welcome email failed with err %v", err)
		}
		auth := &protos.Authentication{}
		utils.Cast(aUser.PlainAuthPrinciple(), auth)
		return &protos.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
	}

	// if it's invited user then
	if cUser.Status == type_enums.RECORD_INVITED {
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

		_, err = wAuthApi.userService.CreateSocial(c, aUser.GetUserInfo().Id, inf.Id, inf.Token, inf.Source, inf.Verified)
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

		auth := &protos.Authentication{}
		utils.Cast(aUser.PlainAuthPrinciple(), auth)
		return &protos.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
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
		aUser, err := wAuthApi.userService.AuthPrinciple(c, cUser.Id)
		if err != nil {
			wAuthApi.logger.Debugf("failed to get auth principle %v", err)
			return nil, err
		}
		auth := &protos.Authentication{}
		utils.Cast(aUser.PlainAuthPrinciple(), auth)
		return &protos.AuthenticateResponse{Code: 200, Success: true, Data: auth}, nil
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
	auth, authErr := types.Authorize(c)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}

	organizationContext, err := iAuth.OrganizationContext()
	if err != nil {
		return nil, err
	}
	cnt, allMembers, err := wAuthApi.userService.GetAllOrganizationMember(c,
		organizationContext.OrganizationID,
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

	out := make([]*protos.User, len(*allMembers))
	for ix, member := range *allMembers {
		out[ix] = &protos.User{
			Name:        member.Member.Name,
			Id:          member.Member.Id,
			Email:       member.Member.Email,
			Role:        member.Role,
			Status:      member.Status.String(),
			CreatedDate: timestamppb.New(time.Time(member.Member.CreatedDate)),
		}
	}
	return &protos.GetAllUserResponse{
		Code:    200,
		Success: true,
		Data:    out,
		Paginated: &protos.Paginated{
			TotalItem:   uint32(cnt),
			CurrentPage: irRequest.GetPaginate().GetPage(),
		},
	}, nil
}
