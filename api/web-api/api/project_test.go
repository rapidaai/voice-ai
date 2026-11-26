//go:build integration && cgo
// +build integration,cgo

package web_api

import (
	"context"
	"testing"
	"time"

	web_config "github.com/rapidaai/api/web-api/config"
	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_organization_service "github.com/rapidaai/api/web-api/internal/service/organization"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	app_config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Name() string {
	return "test-postgres"
}

func (t *testPostgresConnector) IsConnected(ctx context.Context) bool {
	return true
}

func (t *testPostgresConnector) Disconnect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.DB(ctx).Raw(qry).Scan(dest).Error
}

func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if tx, ok := connectors.PostgresTxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return t.db.WithContext(ctx)
}

type testEmailer struct {
	err   error
	calls int
	to    external_clients.Contact
	args  map[string]string
}

func (t *testEmailer) EmailText(ctx context.Context, to external_clients.Contact, subject string, content string) error {
	return nil
}

func (t *testEmailer) EmailRichText(ctx context.Context, to external_clients.Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error {
	t.calls++
	t.to = to
	t.args = args
	return t.err
}

func (t *testEmailer) EmailTemplate(ctx context.Context, to external_clients.Contact, subject string, templateId string, args map[string]string) error {
	return nil
}

func newProjectAPITest(t *testing.T) (*webProjectGRPCApi, *gorm.DB, *testEmailer) {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger2Discard()})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE organizations (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, name text, description text, size text, industry text, contact text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE projects (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, organization_id integer, name text, description text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_auths (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, name text, email text, password text, source text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_auth_tokens (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, user_auth_id integer, token_type text, token text, expire_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE permission_definitions (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, key text, scope_type text, name text, description text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE role_definitions (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, organization_id integer, key text, name text, description text, scope_type text, subject_type text, definition_type text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE role_permissions (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, role_definition_id integer, permission_definition_id integer, scope_type text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_organization_roles (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, user_auth_id integer, organization_id integer, role_definition_id integer, role_scope_type text, role_subject_type text)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX user_organization_roles_current_assignment_idx ON user_organization_roles(user_auth_id, organization_id, role_definition_id) WHERE status IN ('ACTIVE','INVITED')`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_project_roles (id integer primary key, created_date datetime, updated_date datetime, status text, created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer, user_auth_id integer, project_id integer, role_definition_id integer, role_scope_type text, role_subject_type text)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX user_project_roles_current_assignment_idx ON user_project_roles(user_auth_id, project_id, role_definition_id) WHERE status IN ('ACTIVE','INVITED')`).Error)
	require.NoError(t, db.Create(&internal_entity.Organization{
		Audited: gorm_models.Audited{Id: 10},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:        "Acme",
		Description: "Acme org",
		Industry:    "software",
		Contact:     "admin@example.com",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Organization{
		Audited: gorm_models.Audited{Id: 20},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:        "Other",
		Description: "Other org",
		Industry:    "software",
		Contact:     "admin@example.com",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 100},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		OrganizationId: 10,
		Name:           "Alpha",
		Description:    "Alpha project",
	}).Error)
	require.NoError(t, db.Exec(`INSERT INTO permission_definitions(id,status,key,scope_type,name,description) VALUES (1,'ACTIVE','organization.members.invite','ORGANIZATION','Invite',''),(2,'ACTIVE','organization.access.manage','ORGANIZATION','Manage',''),(3,'ACTIVE','organization.project_access_tokens.revoke','ORGANIZATION','Revoke','')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO role_definitions(id,status,organization_id,key,name,description,scope_type,subject_type,definition_type) VALUES
		(101,'ACTIVE',NULL,'organization_owner','Organization Owner','','ORGANIZATION','USER','BUILT_IN'),
		(102,'ACTIVE',NULL,'organization_admin','Organization Admin','','ORGANIZATION','USER','BUILT_IN'),
		(108,'ACTIVE',NULL,'organization_member','Organization Member','','ORGANIZATION','USER','BUILT_IN'),
		(109,'ACTIVE',NULL,'project_admin','Project Admin','','PROJECT','USER','BUILT_IN'),
		(110,'ACTIVE',NULL,'project_developer','Project Developer','','PROJECT','USER','BUILT_IN'),
		(112,'ACTIVE',NULL,'project_viewer','Project Viewer','','PROJECT','USER','BUILT_IN')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO role_permissions(status,role_definition_id,permission_definition_id,scope_type) VALUES ('ACTIVE',101,1,'ORGANIZATION'),('ACTIVE',101,2,'ORGANIZATION'),('ACTIVE',101,3,'ORGANIZATION'),('ACTIVE',102,1,'ORGANIZATION'),('ACTIVE',102,2,'ORGANIZATION'),('ACTIVE',102,3,'ORGANIZATION')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (1,'ACTIVE','Owner','owner@example.com','hash','direct')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_organization_roles(id,status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES (1,'ACTIVE',1,10,101,'ORGANIZATION','USER')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (2,'ACTIVE','Member','member@example.com','hash','direct')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO user_organization_roles(id,status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES (2,'ACTIVE',2,10,108,'ORGANIZATION','USER')`).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 101},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		OrganizationId: 10,
		Name:           "Beta",
		Description:    "Beta project",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 200},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		OrganizationId: 20,
		Name:           "Cross",
		Description:    "Cross project",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 300},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		OrganizationId: 10,
		Name:           "Archived",
		Description:    "Archived project",
	}).Error)

	emailer := &testEmailer{}
	postgres := &testPostgresConnector{db: db}
	return &webProjectGRPCApi{
		webProjectApi: webProjectApi{
			cfg: &web_config.WebAppConfig{
				AppConfig: app_config.AppConfig{
					Ui: app_config.ServiceHostConfig{Host: "http://ui.test"},
				},
			},
			logger:         logger,
			postgres:       postgres,
			projectService: internal_project_service.NewProjectService(logger, postgres),
			userService:    internal_user_service.NewUserService(logger, postgres),
			emailerClient:  emailer,
		},
	}, db, emailer
}

func newOrganizationAPITest(t *testing.T) (*webOrganizationGRPCApi, *gorm.DB, *testEmailer) {
	t.Helper()
	projectApi, db, emailer := newProjectAPITest(t)
	return &webOrganizationGRPCApi{
		webOrganizationApi: webOrganizationApi{
			cfg:                 projectApi.cfg,
			logger:              projectApi.logger,
			postgres:            projectApi.postgres,
			organizationService: internal_organization_service.NewOrganizationService(projectApi.logger, projectApi.postgres),
			projectService:      projectApi.projectService,
			userService:         projectApi.userService,
			emailerClient:       emailer,
		},
	}, db, emailer
}

func logger2Discard() logger.Interface {
	return logger.Discard.LogMode(logger.Silent)
}

func ownerContext(roleDefinitionID uint64) context.Context {
	userID := uint64(1)
	if roleDefinitionID == 108 {
		userID = 2
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "test-session"},
		},
		nil,
	)
	if err != nil {
		panic(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx
}

func TestOnboardingCreatesOrganizationBeforeProject(t *testing.T) {
	projectAPI, db, _ := newProjectAPITest(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	userID := uint64(500)
	token := "onboarding-token"
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited:  gorm_models.Audited{Id: userID},
		Mutable:  gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
		Name:     "Onboarding User",
		Email:    "onboarding@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuthToken{
		Audited:    gorm_models.Audited{Id: 501},
		Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
		UserAuthId: userID,
		TokenType:  "auth-token",
		Token:      token,
		ExpireAt:   time.Now().Add(time.Hour),
	}).Error)

	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: token},
		},
		nil,
	)
	require.NoError(t, err)
	requestContext, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	require.NoError(t, err)

	_, err = projectAPI.CreateProject(requestContext, &protos.CreateProjectRequest{ProjectName: "First Project"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	organizationAPI := &webOrganizationGRPCApi{webOrganizationApi: webOrganizationApi{
		logger:              projectAPI.logger,
		postgres:            projectAPI.postgres,
		organizationService: internal_organization_service.NewOrganizationService(projectAPI.logger, projectAPI.postgres),
		userService:         projectAPI.userService,
	}}
	organizationResponse, err := organizationAPI.CreateOrganization(requestContext, &protos.CreateOrganizationRequest{
		OrganizationName:     "Onboarding Organization",
		OrganizationSize:     "small",
		OrganizationIndustry: "software",
	})
	require.NoError(t, err)
	require.True(t, organizationResponse.GetSuccess())

	principal, err = authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: organizationResponse.GetData().GetId()},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: token},
		},
		nil,
	)
	require.NoError(t, err)
	requestContext, err = middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	require.NoError(t, err)

	projectDescription := "Created after onboarding"
	projectResponse, err := projectAPI.CreateProject(requestContext, &protos.CreateProjectRequest{
		ProjectName:        "First Project",
		ProjectDescription: &projectDescription,
	})
	require.NoError(t, err)
	require.True(t, projectResponse.GetSuccess())
}

func TestAddUserToProjectsAssignsExistingOrganizationUserProjectRoles(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 90},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Existing",
		Email:    "existing-projects@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 91},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       90,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(uint64(102)), &protos.AddUserToProjectsRequest{
		UserId: 90,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, RoleDefinitionIds: []uint64{109}},
			{ProjectId: 101, RoleDefinitionIds: []uint64{112}},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Len(t, res.GetData(), 2)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", 90).Error)
	require.Len(t, projectRoles, 2)
	roles := map[uint64]uint64{}
	for _, projectRole := range projectRoles {
		roles[projectRole.ProjectId] = projectRole.RoleDefinitionID
		require.Equal(t, type_enums.RECORD_ACTIVE, projectRole.Status)
	}
	require.Equal(t, map[uint64]uint64{
		100: uint64(109),
		101: uint64(112),
	}, roles)
}

func TestAddUserToProjectsRejectsExistingProjectMembership(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 88},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Existing Project Member",
		Email:    "existing-project-member@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 89},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       88,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 87},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       88,
		ProjectId:        100,
		RoleDefinitionID: 112,
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(uint64(102)), &protos.AddUserToProjectsRequest{
		UserId: 88,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, RoleDefinitionIds: []uint64{109}},
			{ProjectId: 101, RoleDefinitionIds: []uint64{112}},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.ErrorMessage, res.GetError().GetHumanMessage())

	var projectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&projectRole, "user_auth_id = ? AND project_id = ?", 88, 100).Error)
	require.Equal(t, uint64(112), projectRole.RoleDefinitionID)
}

func TestAddUserToProjectsRejectsUserOutsideOrganization(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 92},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Outside",
		Email:    "outside@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 93},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       92,
		OrganizationId:   20,
		RoleDefinitionID: 108,
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(uint64(101)), &protos.AddUserToProjectsRequest{
		UserId: 92,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, RoleDefinitionIds: []uint64{112}},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage, res.GetError().GetHumanMessage())
}

func TestAddUserToProjectsRejectsArchivedOrganizationUser(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 96},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		Name:     "Archived Org User",
		Email:    "archived-org-user@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 97},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		UserAuthId:       96,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(uint64(101)), &protos.AddUserToProjectsRequest{
		UserId: 96,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, RoleDefinitionIds: []uint64{112}},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 96).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestAddUserToProjectsRejectsInvalidAssignmentsBeforeWrites(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 94},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Invalid Assignments",
		Email:    "invalid-assignments@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 95},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       94,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)

	tests := []struct {
		name          string
		req           *protos.AddUserToProjectsRequest
		platformError pkg_errors.PlatformError
	}{
		{
			name:          "zero user id",
			platformError: pkg_errors.AddUserToProjectsInvalidUser,
			req: &protos.AddUserToProjectsRequest{
				UserId: 0,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, RoleDefinitionIds: []uint64{112}},
				},
			},
		},
		{
			name:          "missing project roles",
			platformError: pkg_errors.AddUserToProjectsMissingProjectRoles,
			req:           &protos.AddUserToProjectsRequest{UserId: 94},
		},
		{
			name:          "empty project id",
			platformError: pkg_errors.AddUserToProjectsInvalidProjects,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 0, RoleDefinitionIds: []uint64{112}},
				},
			},
		},
		{
			name:          "duplicate project",
			platformError: pkg_errors.AddUserToProjectsDuplicateProject,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, RoleDefinitionIds: []uint64{112}},
					{ProjectId: 100, RoleDefinitionIds: []uint64{110}},
				},
			},
		},
		{
			name:          "invalid project role",
			platformError: pkg_errors.AddUserToProjectsInvalidProjectRole,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, RoleDefinitionIds: []uint64{0}},
				},
			},
		},
		{
			name:          "empty project roles",
			platformError: pkg_errors.AddUserToProjectsInvalidProjectRole,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100},
				},
			},
		},
		{
			name:          "duplicate project role",
			platformError: pkg_errors.AddUserToProjectsInvalidProjectRole,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, RoleDefinitionIds: []uint64{112, 112}},
				},
			},
		},
		{
			name:          "project outside organization",
			platformError: pkg_errors.AddUserToProjectsInvalidProjects,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 200, RoleDefinitionIds: []uint64{112}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.AddUserToProjects(ownerContext(uint64(102)), tt.req)
			require.NoError(t, err)
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
			var count int64
			require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestAddUserToProjectsProjectRoleFailureReturnsPayloadError(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 96},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Failure",
		Email:    "failure@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 97},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       96,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)
	require.NoError(t, db.Exec(`DROP TABLE user_project_roles`).Error)

	res, err := api.AddUserToProjects(ownerContext(uint64(101)), &protos.AddUserToProjectsRequest{
		UserId: 96,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, RoleDefinitionIds: []uint64{112}},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsCreateProjectRoles.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.ErrorMessage, res.GetError().GetHumanMessage())
}

func TestDeleteUserFromProjectArchivesOnlySelectedProjectRole(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 130},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Delete Project",
		Email:    "delete-project@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 131},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       130,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 132},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       130,
		ProjectId:        100,
		RoleDefinitionID: 112,
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 133},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       130,
		ProjectId:        101,
		RoleDefinitionID: 110,
	}).Error)

	res, err := api.DeleteUserFromProject(ownerContext(uint64(101)), &protos.DeleteUserFromProjectRequest{
		UserId:    130,
		ProjectId: 100,
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 130, res.GetId())

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 130).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, user.Status)

	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ? AND organization_id = ?", 130, 10).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, orgRole.Status)

	var deletedProjectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&deletedProjectRole, "user_auth_id = ? AND project_id = ?", 130, 100).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, deletedProjectRole.Status)

	var activeProjectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&activeProjectRole, "user_auth_id = ? AND project_id = ?", 130, 101).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, activeProjectRole.Status)
}

func TestDeleteUserFromProjectRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 140},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Project Delete Validation",
		Email:    "project-delete-validation@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 141},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:       140,
		OrganizationId:   10,
		RoleDefinitionID: 108,
	}).Error)

	res, err := api.DeleteUserFromProject(context.Background(), &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.DeleteUserFromProjectUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.DeleteUserFromProjectRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(uint64(108)),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectUnauthorized,
		},
		{
			name:          "zero user id",
			ctx:           ownerContext(uint64(102)),
			req:           &protos.DeleteUserFromProjectRequest{ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectInvalidUser,
		},
		{
			name:          "zero project id",
			ctx:           ownerContext(uint64(102)),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140},
			platformError: pkg_errors.DeleteUserFromProjectInvalidProject,
		},
		{
			name:          "project outside organization",
			ctx:           ownerContext(uint64(102)),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 200},
			platformError: pkg_errors.DeleteUserFromProjectInvalidProject,
		},
		{
			name:          "user not in project",
			ctx:           ownerContext(uint64(102)),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectUserNotInProject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.DeleteUserFromProject(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
		})
	}
}
