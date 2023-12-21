package internal_user_service

import (
	"context"
	"strings"
	"time"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/ciphers"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
	"gorm.io/gorm/clause"
)

const (
	USER_PROJECT_ORG_QUERY = "user_auth_id = ? and status = ?"
)

type userService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewUserService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.UserService {
	return &userService{
		logger:   logger,
		postgres: postgres,
	}
}

func NewAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.Authenticator {
	return &userService{
		logger:   logger,
		postgres: postgres,
	}
}

func (aS *userService) Authenticate(ctx context.Context, email string, password string) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	var aUser internal_gorm.UserAuth
	tx := db.First(&aUser, "email = ? AND password = ?", strings.ToLower(email), ciphers.Hash(password))
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	aToken, err := aS.GetAuthToken(ctx, aUser.Id)
	if err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}

	var rt internal_gorm.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ? AND status = ?", aUser.Id, "active")
	//This fails first request to create org
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		aS.logger.Debugf("organization not found for the user %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, aUser.Id, rt.OrganizationId)
	return &authPrinciple{user: &aUser, userAuthToken: aToken, userOrgRole: &rt, userProjectRoles: prjs}, nil
}

func (aS *userService) getUserProjectRoles(ctx context.Context, userId uint64, organizationId uint64) *[]internal_gorm.UserProjectRole {
	db := aS.postgres.DB(ctx)
	var prjs []internal_gorm.UserProjectRole
	tx := db.Where(&internal_gorm.UserProjectRole{UserAuthId: userId, Status: "active"}).InnerJoins("Project", db.Where(&internal_gorm.Project{OrganizationId: organizationId, Status: "active"})).Find(&prjs)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		aS.logger.Debugf("user project not found not found for the user %v", tx.Error)
		// return nil, tx.Error
	}
	return &prjs
}

func (aS *userService) Create(ctx context.Context, name string, email string, password string, status string, source *string) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_gorm.UserAuth{
		Name:     name,
		Email:    strings.ToLower(email),
		Password: ciphers.Hash(password),
		Status:   status,
	}
	if source != nil {
		user.Source = *source
	}
	tx := db.Save(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, err
	}
	aTh, err := aS.CreateNewAuthToken(ctx, user.Id)
	if err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}

	return &authPrinciple{user: user, userAuthToken: aTh}, nil
}

func (aS *userService) Get(ctx context.Context, email string) (*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var aUser internal_gorm.UserAuth
	tx := db.First(&aUser, "email = ?", strings.ToLower(email))
	if tx.Error != nil {
		aS.logger.Errorf("unable to find the user %s", email)
		return nil, tx.Error
	}
	return &aUser, nil
}

func (aS *userService) Activate(ctx context.Context, userId uint64, name string, source *string) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	ct := internal_gorm.UserAuth{Name: name, Status: "active", UpdatedBy: userId}
	if source != nil {
		ct.Source = *source
	}

	tx := db.Where("id=?", userId).Clauses(clause.Returning{}).Updates(&ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	aS.logger.Debugf("user project not found not found for the user %+v", ct)

	// return ct, nil
	aTh, err := aS.GetAuthToken(ctx, ct.Id)
	if err != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, err
	}

	var rt internal_gorm.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, userId, rt.OrganizationId)
	return &authPrinciple{user: &ct, userAuthToken: aTh, userOrgRole: &rt, userProjectRoles: prjs}, nil

}

func (aS *userService) CreateNewAuthToken(ctx context.Context, userId uint64) (*internal_gorm.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_gorm.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "auth-token", Token: ciphers.Token("at"),
		ExpireAt:  time.Now().Add(200 * time.Hour),
		CreatedBy: userId,
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) CreatePasswordToken(ctx context.Context, userId uint64) (*internal_gorm.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_gorm.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "password-token", Token: ciphers.Token("pt"),
		ExpireAt:  time.Now().Add(200 * time.Hour),
		CreatedBy: userId,
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetAuthToken(ctx context.Context, userId uint64) (*internal_gorm.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserAuthToken
	tx := db.First(&ct, "user_auth_id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) GetToken(ctx context.Context, tokenType string, token string) (*internal_gorm.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserAuthToken
	tx := db.First(&ct, "token_type = ? AND token = ?", tokenType, token)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateOrganizationRole(ctx context.Context, auth types.Principle, role string, userId uint64, orgnizationId uint64, status string) (*internal_gorm.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_gorm.UserOrganizationRole{
		UserAuthId:     userId,
		Role:           role,
		OrganizationId: orgnizationId,
		CreatedBy:      auth.GetUserInfo().Id,
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetProjectRole(ctx context.Context, userId uint64, projectId uint64) (*internal_gorm.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserProjectRole
	tx := db.First(&ct, "user_auth_id = ? AND project_id = ?", userId, projectId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateProjectRole(ctx context.Context, auth types.Principle, userId uint64, role string, projectId uint64, status string) (*internal_gorm.UserProjectRole, error) {
	pr, err := aS.GetProjectRole(ctx, userId, projectId)
	db := aS.postgres.DB(ctx)
	if err != nil {
		projectRole := &internal_gorm.UserProjectRole{
			UserAuthId: userId,
			ProjectId:  projectId,
			Role:       role,
			Status:     status,
			CreatedBy:  auth.GetUserInfo().Id,
		}
		tx := db.Save(projectRole)
		if tx.Error != nil {
			aS.logger.Errorf("exception in DB transaction %v", tx.Error)
			return nil, tx.Error
		}
		return projectRole, nil
	}

	pr.UpdatedBy = auth.GetUserInfo().Id
	pr.Role = role
	pr.Status = status
	tx := db.Save(pr)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return pr, nil
}

func (aS *userService) GetOrganizationRole(ctx context.Context, userId uint64) (*internal_gorm.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserOrganizationRole
	tx := db.Last(&ct, "user_auth_id = ? AND status = ?", userId, "active")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) AuthPrinciple(ctx context.Context, userId uint64) (types.Principle, error) {

	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserAuthToken
	tx := db.Last(&ct, "user_auth_id = ? AND token_type = ? ", userId, "auth-token")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var aUser internal_gorm.UserAuth
	tx = db.First(&aUser, "id = ? AND status = ? ", userId, "active")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var rt internal_gorm.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ? AND status = ?", userId, "active")
	//This fails first request to create org
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		aS.logger.Debugf("organization not found for the user %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, userId, rt.OrganizationId)

	return &authPrinciple{user: &aUser, userAuthToken: &ct, userOrgRole: &rt, userProjectRoles: prjs}, nil
}

func (aS *userService) Authorize(ctx context.Context, token string, userId uint64) (types.Principle, error) {

	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserAuthToken
	tx := db.First(&ct, "user_auth_id = ? AND token = ? AND token_type = ? ", userId, token, "auth-token")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var aUser internal_gorm.UserAuth
	tx = db.First(&aUser, "id = ? AND status = ? ", userId, "active")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var rt internal_gorm.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ? AND status = ?", userId, "active")
	//This fails first request to create org
	if tx.Error != nil {
		aS.logger.Debugf("organization not found for the user %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, userId, rt.OrganizationId)

	return &authPrinciple{user: &aUser, userAuthToken: &ct, userOrgRole: &rt, userProjectRoles: prjs}, nil
}

func (aS *userService) GetUser(ctx context.Context, userId uint64) (*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_gorm.UserAuth
	tx := db.First(&ct, "id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) UpdateUser(ctx context.Context, auth types.Principle, userId uint64, name *string) (*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_gorm.UserAuth{
		UpdatedBy: auth.GetUserInfo().Id,
	}
	if name != nil {
		user.Name = *name
	}
	tx := db.Where("id = ? and status = ?", userId, "active").Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	} else {
		return user, nil
	}
}

func (aS *userService) GetAllUsers(ctx context.Context, uIds []uint64) ([]*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)

	var users []*internal_gorm.UserAuth
	if err := db.Where("id in ? and status = ? ", uIds, "active").Find(&users).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return users, nil
}

func (as *userService) CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_gorm.UserSocial, error) {
	db := as.postgres.DB(ctx)
	ct := &internal_gorm.UserSocial{
		UserAuthId: userId,
		Social:     source,
		Identifier: id,
		Token:      token,
		Verified:   verified,
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		as.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetSocial(ctx context.Context, userId uint64) (*internal_gorm.UserSocial, error) {
	db := aS.postgres.DB(ctx)
	var socialUser *internal_gorm.UserSocial
	if err := db.Where("user_auth_id = ? and status = ? ", userId, "active").Find(&socialUser).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return socialUser, nil
}

func (aS *userService) GetUsers(ctx context.Context, uIds []uint64, limit uint32, offset uint32) ([]*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var u []*internal_gorm.UserAuth
	if err := db.Limit(int(limit)).Offset(int(offset)).Where("id IN ?", uIds).Order("created_date ASC").Find(&u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func (aS *userService) UpdatePassword(ctx context.Context, userId uint64, password string) (*internal_gorm.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_gorm.UserAuth{
		UpdatedBy: userId,
		Password:  ciphers.Hash(password),
	}
	tx := db.Where("id = ? ", userId).Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	} else {
		return user, nil
	}
}

func (aS *userService) ActivateAllProjectRoles(ctx context.Context, userId uint64) error {
	db := aS.postgres.DB(ctx)
	// Update with struct
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, "invited").Updates(&internal_gorm.UserProjectRole{Status: "active", UpdatedBy: userId})
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	} else {
		return nil
	}
}

func (aS *userService) ActivateAllOrganizationRole(ctx context.Context, userId uint64) error {
	db := aS.postgres.DB(ctx)
	// Update with struct
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, "invited").Updates(&internal_gorm.UserOrganizationRole{Status: "active", UpdatedBy: userId})
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	} else {
		return nil
	}
}

func (aS *userService) GetAllActiveProjectMember(ctx context.Context, projectId uint64) (*[]internal_gorm.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []internal_gorm.UserProjectRole
	tx := db.Preload("Member").Where("project_id = ? AND status = ?", projectId, "active").Find(&rt)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return &rt, err
	} else {
		return &rt, nil
	}
}

func (aS *userService) GetAllOrganizationMember(ctx context.Context, organizationId uint64) (*[]internal_gorm.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []internal_gorm.UserOrganizationRole
	tx := db.Preload("Member").Where("organization_id = ? AND status = ?", organizationId, "active").Find(&rt)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return &rt, err
	} else {
		return &rt, nil
	}
}

func (uS *userService) GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_gorm.UserOrganizationRole, error) {
	db := uS.postgres.DB(ctx)
	var roles []*internal_gorm.UserOrganizationRole
	if err := db.Where("organization_id = ? and status = ?", organizationId, "active").Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

func (aS *userService) GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_gorm.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var pr []*internal_gorm.UserProjectRole
	if err := db.Where("project_id IN ? and user_auth_id IN ? and status = ?", pIds, uIds, "active").Find(&pr).Error; err != nil {
		return nil, err
	}
	return pr, nil
}

func (aS *userService) GetAllProjectMembers(ctx context.Context, projectId uint64) (*[]internal_gorm.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []internal_gorm.UserProjectRole
	tx := db.Preload("Member").Where("project_id = ?", projectId).Find(&rt)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return &rt, err
	} else {
		return &rt, nil
	}
}
