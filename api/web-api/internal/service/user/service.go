package internal_user_service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	internal_entity "github.com/lexatic/web-backend/api/web-api/internal/entity"
	internal_services "github.com/lexatic/web-backend/api/web-api/internal/service"
	"github.com/lexatic/web-backend/pkg/ciphers"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	"github.com/lexatic/web-backend/pkg/types"
	type_enums "github.com/lexatic/web-backend/pkg/types/enums"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm/clause"
)

var DEFAULT_USER_FEATURE_PERMISSION = []string{"/deployment/.*", "/knowledge/.*", "/observability/.*"}

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
	var aUser internal_entity.UserAuth
	tx := db.First(&aUser, "email = ? AND password = ?", strings.ToLower(email), ciphers.Hash(password))
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var wg sync.WaitGroup
	var aToken *internal_entity.UserAuthToken
	var rt internal_entity.UserOrganizationRole
	var prjs *[]internal_entity.UserProjectRole
	var permissions []*internal_entity.UserFeaturePermission

	var errChan = make(chan error, 3)

	// all is done in goroutine
	wg.Add(4)

	go func() {
		defer wg.Done()
		var err error
		aToken, err = aS.GetAuthToken(ctx, aUser.Id)
		if err != nil {
			errChan <- fmt.Errorf("exception in DB transaction: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		tx := db.Preload("Organization").First(&rt, "user_auth_id = ? AND status = ?", aUser.Id, type_enums.RECORD_ACTIVE.String())
		if tx.Error != nil {
			aS.logger.Debugf("organization not found for the user: %v", tx.Error)
			// Uncomment the following line if you want to treat this as an error
			// errChan <- fmt.Errorf("exception in DB transaction: %v", tx.Error)
		}
	}()

	go func() {
		defer wg.Done()
		prjs = aS.getUserProjectRoles(ctx, aUser.Id, rt.OrganizationId)
	}()

	go func() {
		defer wg.Done()
		permissions, _ = aS.GetAllUserFeaturePermission(ctx, aUser.Id)
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			aS.logger.Errorf("error on one of the routine of authentication %v", err)
			return nil, err
		}
	}
	return &authPrinciple{user: &aUser, userAuthToken: aToken, userOrgRole: &rt, userProjectRoles: prjs, featurePermissions: permissions}, nil
}

func (aS *userService) getUserProjectRoles(ctx context.Context, userId uint64, organizationId uint64) *[]internal_entity.UserProjectRole {
	db := aS.postgres.DB(ctx)
	var prjs []internal_entity.UserProjectRole
	tx := db.Where(&internal_entity.UserProjectRole{
		UserAuthId: userId,
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	}).InnerJoins("Project", db.Where(&internal_entity.Project{
		OrganizationId: organizationId,
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		}})).Find(&prjs)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		aS.logger.Debugf("user project not found not found for the user %v", tx.Error)
		// return nil, tx.Error
	}
	return &prjs
}

func (aS *userService) Create(ctx context.Context, name string, email string, password string, status type_enums.RecordState, source *string) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Name:     name,
		Email:    strings.ToLower(email),
		Password: ciphers.Hash(password),
		Mutable: gorm_models.Mutable{
			Status: status,
		},
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

func (aS *userService) Get(ctx context.Context, email string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var aUser internal_entity.UserAuth
	tx := db.First(&aUser, "email = ?", strings.ToLower(email))
	if tx.Error != nil {
		aS.logger.Errorf("unable to find the user %s", email)
		return nil, tx.Error
	}
	return &aUser, nil
}

func (aS *userService) Activate(ctx context.Context, userId uint64, name string, source *string) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	ct := internal_entity.UserAuth{
		Name: name,
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			UpdatedBy: userId,
		},
	}
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

	var rt internal_entity.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, userId, rt.OrganizationId)
	return &authPrinciple{user: &ct, userAuthToken: aTh, userOrgRole: &rt, userProjectRoles: prjs}, nil

}

func (aS *userService) CreateNewAuthToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_entity.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "auth-token", Token: ciphers.Token("at"),
		ExpireAt: time.Now().Add(200 * time.Hour),
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: userId,
		},
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) CreatePasswordToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_entity.UserAuthToken{
		UserAuthId: userId,
		TokenType:  "password-token", Token: ciphers.Token("pt"),
		ExpireAt: time.Now().Add(200 * time.Hour),
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: userId,
		},
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetAuthToken(ctx context.Context, userId uint64) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuthToken
	tx := db.First(&ct, "user_auth_id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) GetToken(ctx context.Context, tokenType string, token string) (*internal_entity.UserAuthToken, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuthToken
	tx := db.First(&ct, "token_type = ? AND token = ?", tokenType, token)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateOrganizationRole(ctx context.Context, auth types.Principle, role string, userId uint64, orgnizationId uint64, status type_enums.RecordState) (*internal_entity.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	ct := &internal_entity.UserOrganizationRole{
		UserAuthId:     userId,
		Role:           role,
		OrganizationId: orgnizationId,
		Mutable: gorm_models.Mutable{
			Status:    status,
			CreatedBy: auth.GetUserInfo().Id,
		},
	}
	tx := db.Save(ct)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return ct, nil
}

func (aS *userService) GetProjectRole(ctx context.Context, userId uint64, projectId uint64) (*internal_entity.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserProjectRole
	tx := db.First(&ct, "user_auth_id = ? AND project_id = ?", userId, projectId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) CreateProjectRole(ctx context.Context, auth types.Principle, userId uint64, role string, projectId uint64, status type_enums.RecordState) (*internal_entity.UserProjectRole, error) {
	pr, err := aS.GetProjectRole(ctx, userId, projectId)
	db := aS.postgres.DB(ctx)
	if err != nil {
		projectRole := &internal_entity.UserProjectRole{
			UserAuthId: userId,
			ProjectId:  projectId,
			Role:       role,
			Mutable: gorm_models.Mutable{
				Status:    status,
				CreatedBy: auth.GetUserInfo().Id,
			},
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

func (aS *userService) GetOrganizationRole(ctx context.Context, userId uint64) (*internal_entity.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserOrganizationRole
	tx := db.Last(&ct, "user_auth_id = ? AND status = ?", userId, type_enums.RECORD_ACTIVE.String())
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) AuthPrinciple(ctx context.Context, userId uint64) (types.Principle, error) {
	db := aS.postgres.DB(ctx)
	var authToken internal_entity.UserAuthToken
	var userAuth internal_entity.UserAuth
	var orgRole internal_entity.UserOrganizationRole
	var prjs *[]internal_entity.UserProjectRole

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		tx := db.Last(&authToken, "user_auth_id = ? AND token_type = ? ", userId, "auth-token")
		if tx.Error != nil {
			aS.logger.Errorf("exception in DB transaction %v", tx.Error)
			return tx.Error
		}
		return nil
	})

	g.Go(func() error {
		tx := db.First(&userAuth, "id = ? AND status = ? ", userId, type_enums.RECORD_ACTIVE.String())
		if tx.Error != nil {
			aS.logger.Errorf("exception in DB transaction %v", tx.Error)
			return tx.Error
		}
		return nil
	})

	g.Go(func() error {
		tx := db.Preload("Organization").First(&orgRole, "user_auth_id = ? AND status = ?", userId, type_enums.RECORD_ACTIVE.String())
		if tx.Error != nil {
			aS.logger.Errorf("exception in DB transaction %v", tx.Error)
			aS.logger.Debugf("organization not found for the user %v", tx.Error)
			// Note: We're not returning an error here as per the original code
		}
		return nil
	})

	g.Go(func() error {
		prjs = aS.getUserProjectRoles(ctx, userId, orgRole.OrganizationId)
		return nil
	})

	var permissions []*internal_entity.UserFeaturePermission
	g.Go(func() error {
		permissions, _ = aS.GetAllUserFeaturePermission(ctx, userId)
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return &authPrinciple{user: &userAuth, userAuthToken: &authToken, userOrgRole: &orgRole, userProjectRoles: prjs, featurePermissions: permissions}, nil
}

func (aS *userService) Authorize(ctx context.Context, token string, userId uint64) (types.Principle, error) {

	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuthToken
	tx := db.First(&ct, "user_auth_id = ? AND token = ? AND token_type = ? ", userId, token, "auth-token")
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var aUser internal_entity.UserAuth
	tx = db.First(&aUser, "id = ? AND status = ? ", userId, type_enums.RECORD_ACTIVE.String())
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}

	var rt internal_entity.UserOrganizationRole
	tx = db.Preload("Organization").First(&rt, "user_auth_id = ? AND status = ?", userId, type_enums.RECORD_ACTIVE.String())
	//This fails first request to create org
	if tx.Error != nil {
		aS.logger.Debugf("organization not found for the user %v", tx.Error)
		// return nil, tx.Error
	}

	prjs := aS.getUserProjectRoles(ctx, userId, rt.OrganizationId)

	return &authPrinciple{user: &aUser, userAuthToken: &ct, userOrgRole: &rt, userProjectRoles: prjs}, nil
}

func (aS *userService) GetUser(ctx context.Context, userId uint64) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var ct internal_entity.UserAuth
	tx := db.First(&ct, "id = ?", userId)
	if tx.Error != nil {
		aS.logger.Errorf("exception in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return &ct, nil
}

func (aS *userService) UpdateUser(ctx context.Context, auth types.Principle, userId uint64, name *string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Mutable: gorm_models.Mutable{
			UpdatedBy: auth.GetUserInfo().Id,
		},
	}
	if name != nil {
		user.Name = *name
	}
	tx := db.Where("id = ? and status = ?", userId, type_enums.RECORD_ACTIVE.String()).Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	} else {
		return user, nil
	}
}

func (as *userService) CreateSocial(ctx context.Context, userId uint64, id string, token string, source string, verified bool) (*internal_entity.UserSocial, error) {
	db := as.postgres.DB(ctx)
	ct := &internal_entity.UserSocial{
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

func (aS *userService) GetSocial(ctx context.Context, userId uint64) (*internal_entity.UserSocial, error) {
	db := aS.postgres.DB(ctx)
	var socialUser *internal_entity.UserSocial
	if err := db.Where("user_auth_id = ? and status = ? ", userId, type_enums.RECORD_ACTIVE.String()).Find(&socialUser).Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return socialUser, nil
}

func (aS *userService) GetUsers(ctx context.Context, uIds []uint64, limit uint32, offset uint32) ([]*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	var u []*internal_entity.UserAuth
	if err := db.Limit(int(limit)).Offset(int(offset)).Where("id IN ?", uIds).Order("created_date ASC").Find(&u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func (aS *userService) UpdatePassword(ctx context.Context, userId uint64, password string) (*internal_entity.UserAuth, error) {
	db := aS.postgres.DB(ctx)
	user := &internal_entity.UserAuth{
		Mutable: gorm_models.Mutable{
			UpdatedBy: userId,
		},
		Password: ciphers.Hash(password),
	}
	tx := db.Where("id = ? ", userId).Updates(user)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return nil, err
	}
	return user, nil
}

func (aS *userService) ActivateAllProjectRoles(ctx context.Context, userId uint64) error {
	db := aS.postgres.DB(ctx)
	// Update with struct
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, "INVITED").
		Updates(&internal_entity.UserProjectRole{
			Mutable: gorm_models.Mutable{
				Status: type_enums.RECORD_ACTIVE, UpdatedBy: userId,
			},
		})
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
	tx := db.Where("user_auth_id = ? AND status  = ?", userId, "INVITED").Updates(&internal_entity.UserOrganizationRole{
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE, UpdatedBy: userId,
		},
	})
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return err
	}
	return nil
}

func (aS *userService) GetAllActiveProjectMember(ctx context.Context, projectId uint64) (*[]internal_entity.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []internal_entity.UserProjectRole
	tx := db.Preload("Member").Where("project_id = ? AND status = ?", projectId, type_enums.RECORD_ACTIVE.String()).Find(&rt)
	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return &rt, err
	}
	return &rt, nil
}

func (aS *userService) GetAllOrganizationMember(ctx context.Context, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_entity.UserOrganizationRole, error) {
	db := aS.postgres.DB(ctx)
	var rt []internal_entity.UserOrganizationRole
	var cnt int64

	qry := db.Model(internal_entity.UserOrganizationRole{}).
		Preload("Member").
		Where("organization_id = ? AND status = ?", organizationId, type_enums.RECORD_ACTIVE.String())
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&rt)

	if err := tx.Error; err != nil {
		aS.logger.Errorf("exception in DB transaction %v", err)
		return cnt, &rt, err
	}
	return cnt, &rt, nil
}

func (uS *userService) GetAllUserRolesForOrg(ctx context.Context, organizationId uint64) ([]*internal_entity.UserOrganizationRole, error) {
	db := uS.postgres.DB(ctx)
	var roles []*internal_entity.UserOrganizationRole
	if err := db.Where("organization_id = ? and status = ?", organizationId, type_enums.RECORD_ACTIVE.String()).Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

func (aS *userService) GetProjectRolesForUsers(ctx context.Context, pIds []uint64, uIds []uint64) ([]*internal_entity.UserProjectRole, error) {
	db := aS.postgres.DB(ctx)
	var pr []*internal_entity.UserProjectRole
	if err := db.Where("project_id IN ? and user_auth_id IN ? and status = ?", pIds, uIds, type_enums.RECORD_ACTIVE.String()).Find(&pr).Error; err != nil {
		return nil, err
	}
	return pr, nil
}

func (service *userService) GetAllUserFeaturePermission(ctx context.Context, userId uint64) ([]*internal_entity.UserFeaturePermission, error) {
	db := service.postgres.DB(ctx)
	var permissions []*internal_entity.UserFeaturePermission
	if err := db.Where("user_auth_id = ? and status = ?", userId, type_enums.RECORD_ACTIVE.String()).Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (service *userService) EnableAllDefaultUserFeaturePermission(ctx context.Context, userId uint64) ([]*internal_entity.UserFeaturePermission, error) {
	db := service.postgres.DB(ctx)
	allPermission := make([]*internal_entity.UserFeaturePermission, 0)
	for _, prm := range DEFAULT_USER_FEATURE_PERMISSION {
		allPermission = append(allPermission, &internal_entity.UserFeaturePermission{
			UserAuthId: userId,
			Feature:    prm,
			IsEnabled:  true,
			Mutable: gorm_models.Mutable{
				Status:    type_enums.RECORD_ACTIVE,
				UpdatedBy: userId,
			},
		})
	}

	if len(allPermission) == 0 {
		return allPermission, nil
	}

	tx := db.Save(allPermission)
	if tx.Error != nil {
		service.logger.Errorf("exception while adding permission in DB transaction %v", tx.Error)
		return nil, tx.Error
	}
	return allPermission, nil

}
