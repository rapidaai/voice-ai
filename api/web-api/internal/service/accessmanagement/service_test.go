// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package accessmanagement

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testPostgres struct {
	db *gorm.DB
}

func (connector *testPostgres) Connect(context.Context) error { return nil }
func (connector *testPostgres) Name() string                  { return "access-management-test" }
func (connector *testPostgres) IsConnected(context.Context) bool {
	return true
}
func (connector *testPostgres) Disconnect(context.Context) error { return nil }
func (connector *testPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *testPostgres) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

func newTestService(t *testing.T) (internal_services.AccessManagementService, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE organizations (id INTEGER PRIMARY KEY, status TEXT NOT NULL)`,
		`CREATE TABLE user_organization_roles (
			id INTEGER PRIMARY KEY, user_auth_id INTEGER NOT NULL, organization_id INTEGER NOT NULL,
			role TEXT NOT NULL, role_definition_id INTEGER NOT NULL, status TEXT NOT NULL
		)`,
		`CREATE TABLE projects (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE user_project_roles (
			id INTEGER PRIMARY KEY, user_auth_id INTEGER NOT NULL, project_id INTEGER NOT NULL,
			role_definition_id INTEGER NOT NULL, status TEXT NOT NULL
		)`,
		`CREATE TABLE permission_definitions (
			id INTEGER PRIMARY KEY, key TEXT NOT NULL UNIQUE, scope_type TEXT NOT NULL,
			name TEXT NOT NULL, description TEXT NOT NULL,
			created_date DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL, updated_date DATETIME,
			status TEXT NOT NULL, created_actor_type TEXT NOT NULL, created_actor_id INTEGER,
			updated_actor_type TEXT, updated_actor_id INTEGER
		)`,
		`CREATE TABLE role_definitions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, organization_id INTEGER, key TEXT NOT NULL,
			name TEXT NOT NULL, description TEXT NOT NULL, scope_type TEXT NOT NULL,
			subject_type TEXT NOT NULL, definition_type TEXT NOT NULL,
			created_date DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL, updated_date DATETIME,
			status TEXT NOT NULL, created_actor_type TEXT NOT NULL, created_actor_id INTEGER,
			updated_actor_type TEXT, updated_actor_id INTEGER
		)`,
		`CREATE UNIQUE INDEX role_custom_key ON role_definitions(organization_id,key) WHERE definition_type='CUSTOM'`,
		`CREATE TABLE role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, role_definition_id INTEGER NOT NULL,
			permission_definition_id INTEGER NOT NULL, scope_type TEXT NOT NULL,
			created_date DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL, updated_date DATETIME,
			status TEXT NOT NULL, created_actor_type TEXT NOT NULL, created_actor_id INTEGER,
			updated_actor_type TEXT, updated_actor_id INTEGER
		)`,
		`CREATE TABLE project_credentials (
			id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, project_id INTEGER NOT NULL,
			role_definition_id INTEGER NOT NULL, role_subject_type TEXT NOT NULL,
			status TEXT NOT NULL
		)`,
		`INSERT INTO organizations(id,status) VALUES (10,'ACTIVE'),(20,'ACTIVE')`,
		`INSERT INTO projects(id,organization_id,status) VALUES (100,10,'ACTIVE'),(200,20,'ACTIVE')`,
		`INSERT INTO user_organization_roles(id,user_auth_id,organization_id,role,role_definition_id,status) VALUES
			(1,1,10,'owner',90,'ACTIVE'),(2,2,10,'admin',90,'ACTIVE'),(3,3,10,'member',91,'ACTIVE')`,
		`INSERT INTO permission_definitions(id,key,scope_type,name,description,status,created_actor_type) VALUES
			(1,'organization.read','ORGANIZATION','Organization Read','Allows organization.read.','ACTIVE','unknown'),
			(2,'organization.update','ORGANIZATION','Organization Update','Allows organization.update.','ACTIVE','unknown'),
			(3,'organization.roles.read','ORGANIZATION','Organization Roles Read','Allows organization.roles.read.','ACTIVE','unknown'),
			(4,'organization.roles.create','ORGANIZATION','Organization Roles Create','Allows organization.roles.create.','ACTIVE','unknown'),
			(5,'organization.roles.update','ORGANIZATION','Organization Roles Update','Allows organization.roles.update.','ACTIVE','unknown'),
			(6,'organization.roles.archive','ORGANIZATION','Organization Roles Archive','Allows organization.roles.archive.','ACTIVE','unknown'),
			(34,'project.read','PROJECT','Project Read','Allows project.read.','ACTIVE','unknown'),
			(35,'project.update','PROJECT','Project Update','Allows project.update.','ACTIVE','unknown'),
			(36,'project.roles.read','PROJECT','Project Roles Read','Allows project.roles.read.','ACTIVE','unknown')`,
		`INSERT INTO role_definitions(id,organization_id,key,name,description,scope_type,subject_type,definition_type,status,created_actor_type) VALUES
			(90,NULL,'organization_role_admin','Organization Role Admin','Built-in','ORGANIZATION','USER','BUILT_IN','ACTIVE','unknown'),
			(91,NULL,'organization_member','Organization Member','Built-in','ORGANIZATION','USER','BUILT_IN','ACTIVE','unknown'),
			(80,NULL,'runtime_invoker','Runtime Invoker','Built-in','PROJECT','PROJECT_TOKEN','BUILT_IN','ACTIVE','unknown'),
			(81,NULL,'project_viewer','Project Viewer','Built-in','PROJECT','USER','BUILT_IN','ACTIVE','unknown'),
			(101,NULL,'organization_owner','Organization Owner','Built-in','ORGANIZATION','USER','BUILT_IN','ACTIVE','unknown')`,
		`INSERT INTO role_permissions(id,role_definition_id,permission_definition_id,scope_type,status,created_actor_type) VALUES
			(1,90,3,'ORGANIZATION','ACTIVE','unknown'),(2,90,4,'ORGANIZATION','ACTIVE','unknown'),
			(3,90,5,'ORGANIZATION','ACTIVE','unknown'),(4,90,6,'ORGANIZATION','ACTIVE','unknown')`,
	}
	for _, statement := range statements {
		require.NoError(t, database.Exec(statement).Error)
	}
	return NewAccessManagementService(&testPostgres{db: database}), database
}

func testAuth(userID, organizationID uint64) authentication.Principal {
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: organizationID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: "access-management-test-session",
			},
		},
		[]authentication.PermissionKey{
			"organization.roles.read",
			"organization.roles.create",
			"organization.roles.update",
			"organization.roles.archive",
		},
	)
	return principal
}

func testProjectAuth(userID, organizationID, projectID uint64) authentication.Principal {
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceUserSession,
				Value: "access-management-project-test-session",
			},
		},
		[]authentication.PermissionKey{"project.roles.read"},
	)
	return principal
}

func TestCreateCustomRoleUsesDatabaseIDAndSameScopeCatalog(t *testing.T) {
	service, _ := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_updater", "Project Updater", "Updates projects",
		ScopeProject, SubjectUser, []uint64{35},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(102), created.Id)
	require.Len(t, created.RolePermissions, 1)
	require.Equal(t, uint64(35), created.RolePermissions[0].PermissionDefinition.Id)

	tokenRole, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_updater_token", "Project Updater Token", "",
		ScopeProject, SubjectProjectToken, []uint64{35},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(35), tokenRole.RolePermissions[0].PermissionDefinition.Id)

	organizationRole, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"organization_updater", "Organization Updater", "",
		ScopeOrganization, SubjectUser, []uint64{2},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), organizationRole.RolePermissions[0].PermissionDefinition.Id)

	_, err = service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"cross_scope", "Cross Scope", "",
		ScopeOrganization, SubjectUser, []uint64{34},
	)
	require.ErrorIs(t, err, pkg_errors.AccessManagementRolePermissionSelectionInvalidError{})
}

func TestRoleCatalogUsesRouteAuthorizedPrincipal(t *testing.T) {
	service, _ := newTestService(t)
	count, permissions, err := service.ListPermissions(context.Background(), testAuth(3, 10), nil, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(9), count)
	require.Len(t, permissions, 9)

	count, permissions, err = service.ListPermissions(context.Background(), testAuth(2, 10), []*protos.Criteria{{Key: "scope", Logic: "=", Value: ScopeProject}}, &protos.Paginate{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	require.Len(t, permissions, 2)

	count, permissions, err = service.ListPermissions(context.Background(), testAuth(2, 10), []*protos.Criteria{{Key: "scope", Logic: "=", Value: ScopeProject}}, &protos.Paginate{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	require.Len(t, permissions, 1)
}

func TestProjectRoleCatalogCannotReadOrganizationDefinitions(t *testing.T) {
	service, _ := newTestService(t)
	principal := testProjectAuth(3, 10, 100)

	count, roles, err := service.ListRoles(context.Background(), principal, nil, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Len(t, roles, 2)
	for _, role := range roles {
		require.Equal(t, ScopeProject, role.ScopeType)
	}

	_, err = service.GetRole(context.Background(), principal, 101)
	require.ErrorIs(t, err, pkg_errors.AccessManagementRoleNotFoundError{})
}

func TestListRolesAppliesAllowlistedCriteriaWithLiteralContainsAndAND(t *testing.T) {
	service, _ := newTestService(t)
	count, roles, err := service.ListRoles(context.Background(), testAuth(1, 10), []*protos.Criteria{
		{Key: "name", Logic: "contains", Value: "PrOjEcT"},
		{Key: "key", Logic: "contains", Value: "viewer"},
		{Key: "scope", Logic: "=", Value: ScopeProject},
		{Key: "subject", Logic: "=", Value: SubjectUser},
		{Key: "type", Logic: "=", Value: DefinitionBuiltIn},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Len(t, roles, 1)
	require.Equal(t, "project_viewer", roles[0].Key)

	count, roles, err = service.ListRoles(context.Background(), testAuth(1, 10), []*protos.Criteria{
		{Key: "name", Logic: "contains", Value: "%"},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, roles)

	count, roles, err = service.ListRoles(context.Background(), testAuth(1, 10), []*protos.Criteria{
		{Key: "name", Logic: "contains", Value: "project"},
		{Key: "key", Logic: "contains", Value: "runtime"},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, roles)
}

func TestOrganizationCatalogDoesNotRepeatRouteAuthorization(t *testing.T) {
	service, database := newTestService(t)
	custom, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"automation_custom", "Automation Custom", "",
		ScopeProject, SubjectProjectToken, []uint64{34},
	)
	require.NoError(t, err)

	count, roles, err := service.ListRoles(context.Background(), testAuth(3, 10), []*protos.Criteria{
		{Key: "scope", Logic: "=", Value: ScopeProject},
		{Key: "subject", Logic: "=", Value: SubjectProjectToken},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Len(t, roles, 2)

	role, err := service.GetRole(context.Background(), testAuth(3, 10), custom.Id)
	require.NoError(t, err)
	require.Equal(t, custom.Id, role.Id)

	_, _, err = service.ListRoles(context.Background(), testAuth(3, 10), []*protos.Criteria{{Key: "scope", Logic: "=", Value: ScopeProject}}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	_, _, err = service.ListRoles(context.Background(), testAuth(3, 10), []*protos.Criteria{{Key: "scope", Logic: "=", Value: ScopeOrganization}, {Key: "subject", Logic: "=", Value: SubjectUser}}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	_, err = service.GetRole(context.Background(), testAuth(3, 10), 81)
	require.NoError(t, err)
	_, err = service.GetRole(context.Background(), testAuth(3, 10), 101)
	require.NoError(t, err)

	require.NoError(t, database.Model(&internal_entity.RoleDefinition{}).Where("id = ?", custom.Id).Update("status", "ARCHIEVE").Error)
	_, err = service.GetRole(context.Background(), testAuth(3, 10), custom.Id)
	require.ErrorIs(t, err, pkg_errors.AccessManagementRoleNotFoundError{})
}

func TestReducePermissionsArchivesRows(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_runtime", "Project Runtime", "",
		ScopeProject, SubjectUser, []uint64{34},
	)
	require.NoError(t, err)

	_, preview, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Remove read access", "request-123", "", true,
	)
	require.NoError(t, err)
	reduced, _, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Remove read access", "request-123", preview.ConfirmationDigest, false,
	)
	require.NoError(t, err)
	require.Empty(t, reduced.RolePermissions)

	var permission internal_entity.RolePermission
	require.NoError(t, database.Where("role_definition_id = ?", created.Id).First(&permission).Error)
	require.Equal(t, "ARCHIEVE", string(permission.Status))
	require.Equal(t, string(authentication.IdentityUser), permission.UpdatedActorType)
	require.Equal(t, uint64(1), permission.UpdatedActorID)

	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Remove read access", "request-123", preview.ConfirmationDigest, false,
	)
	require.ErrorIs(t, err, pkg_errors.AccessManagementReductionStaleError{})
}

func TestReductionRetryDoesNotArchiveAdditionalPermissions(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_access", "Project Access", "",
		ScopeProject, SubjectUser, []uint64{34, 36},
	)
	require.NoError(t, err)

	permissionDefinitionIDs := []uint64{34, 36}
	_, preview, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, permissionDefinitionIDs, "Remove project access", "request-exact", "", true,
	)
	require.NoError(t, err)
	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, permissionDefinitionIDs, "Remove project access", "request-exact", preview.ConfirmationDigest, false,
	)
	require.NoError(t, err)

	permissionDefinitionIDs = []uint64{34}
	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, permissionDefinitionIDs, "Remove project access", "request-exact", preview.ConfirmationDigest, false,
	)
	require.ErrorIs(t, err, pkg_errors.AccessManagementReductionStaleError{})
	permissionDefinitionIDs = []uint64{35}
	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, permissionDefinitionIDs, "Remove project access", "request-exact", preview.ConfirmationDigest, false,
	)
	require.ErrorIs(t, err, pkg_errors.AccessManagementReductionStaleError{})

	var archived int64
	require.NoError(t, database.Model(&internal_entity.RolePermission{}).
		Where("role_definition_id = ? AND status = ?", created.Id, "ARCHIEVE").Count(&archived).Error)
	require.Equal(t, int64(2), archived)
}

func TestReductionPreviewIncludesCurrentUserImpactsAndRejectsStaleConfirmation(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_support", "Project Support", "", ScopeProject, SubjectUser, []uint64{34, 36})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`INSERT INTO user_project_roles(id,user_auth_id,project_id,role_definition_id,status) VALUES
		(10,7,100,?,'ACTIVE'),(11,8,100,?,'INVITED'),(12,9,100,?,'ARCHIEVE')`, created.Id, created.Id, created.Id).Error)
	require.NoError(t, database.Exec("UPDATE projects SET status='ARCHIEVE' WHERE id=100").Error)

	_, first, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{36}, "Reduce support", "preview-1", "", true)
	require.NoError(t, err)
	_, second, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{36}, "Reduce support", "preview-2", "", true)
	require.NoError(t, err)
	require.Equal(t, first.ConfirmationDigest, second.ConfirmationDigest)
	require.Regexp(t, `^[0-9a-f]{64}$`, first.ConfirmationDigest)
	require.Len(t, first.Impacts, 2)
	require.Equal(t, []string{"ACTIVE", "INVITED"}, []string{first.Impacts[0].AssignmentStatus, first.Impacts[1].AssignmentStatus})
	require.Len(t, first.ResultingRole.RolePermissions, 1)
	require.Equal(t, "project.read", first.ResultingRole.RolePermissions[0].PermissionDefinition.Key)

	require.NoError(t, database.Exec(`INSERT INTO user_project_roles(id,user_auth_id,project_id,role_definition_id,status) VALUES (13,10,100,?,'ACTIVE')`, created.Id).Error)
	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{36}, "Reduce support", "confirm-1", first.ConfirmationDigest, false)
	require.ErrorIs(t, err, pkg_errors.AccessManagementReductionStaleError{})
	var active int64
	require.NoError(t, database.Model(&internal_entity.RolePermission{}).Where("role_definition_id = ? AND status = 'ACTIVE'", created.Id).Count(&active).Error)
	require.Equal(t, int64(2), active)
}

func TestReductionConfirmationRevalidatesCallerPermission(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_observer", "Project Observer", "", ScopeProject, SubjectUser, []uint64{34})
	require.NoError(t, err)
	_, preview, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Reduce observer", "preview", "", true)
	require.NoError(t, err)
	require.NoError(t, database.Exec("UPDATE role_permissions SET status='ARCHIEVE' WHERE role_definition_id=90 AND permission_definition_id=5").Error)
	_, _, err = service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Reduce observer", "confirm", preview.ConfirmationDigest, false)
	require.ErrorIs(t, err, pkg_errors.AccessManagementPermissionDeniedError{})
	var active int64
	require.NoError(t, database.Model(&internal_entity.RolePermission{}).Where("role_definition_id = ? AND status = 'ACTIVE'", created.Id).Count(&active).Error)
	require.Equal(t, int64(1), active)
}

func TestReductionPreviewIncludesOnlyActiveCredentialImpact(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"token_reader", "Token Reader", "", ScopeProject, SubjectProjectToken, []uint64{34})
	require.NoError(t, err)
	require.NoError(t, database.Exec(`INSERT INTO project_credentials(id,organization_id,project_id,role_definition_id,role_subject_type,status) VALUES
		(30,10,100,?,'PROJECT_TOKEN','ACTIVE'),(31,10,100,?,'PROJECT_TOKEN','ARCHIEVE')`, created.Id, created.Id).Error)
	_, preview, err := service.ReduceCustomRolePermissions(context.Background(), testAuth(1, 10),
		created.Id, []uint64{34}, "Reduce token", "preview", "", true)
	require.NoError(t, err)
	require.Len(t, preview.Impacts, 1)
	require.Equal(t, SubjectProjectToken, preview.Impacts[0].SubjectType)
	require.Equal(t, uint64(30), preview.Impacts[0].SubjectID)
}

func TestListGetAndArchiveCustomRole(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_audit", "Project Audit", "",
		ScopeProject, SubjectUser, []uint64{36, 34},
	)
	require.NoError(t, err)

	count, roles, err := service.ListRoles(context.Background(), testAuth(1, 10), []*protos.Criteria{
		{Key: "scope", Logic: "=", Value: ScopeProject},
		{Key: "type", Logic: "=", Value: DefinitionCustom},
	}, &protos.Paginate{Page: 1, PageSize: 25})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, created.Id, roles[0].Id)
	require.Nil(t, roles[0].RolePermissions)

	role, err := service.GetRole(context.Background(), testAuth(1, 10), created.Id)
	require.NoError(t, err)
	require.Len(t, role.RolePermissions, 2)
	require.NotNil(t, role.RolePermissions[0].PermissionDefinition)
	require.Equal(t, uint64(34), role.RolePermissions[0].PermissionDefinition.Id)
	require.Equal(t, uint64(36), role.RolePermissions[1].PermissionDefinition.Id)

	require.NoError(t, service.ArchiveCustomRole(context.Background(), testAuth(1, 10),
		created.Id, "No longer required", "archive-123",
	))
	require.ErrorIs(t, service.ArchiveCustomRole(context.Background(), testAuth(1, 10),
		created.Id, "No longer required", "archive-123",
	), pkg_errors.AccessManagementRoleNotFoundError{})

	var archived internal_entity.RoleDefinition
	require.NoError(t, database.Where("id = ?", created.Id).First(&archived).Error)
	require.Equal(t, "ARCHIEVE", string(archived.Status))
	require.Equal(t, string(authentication.IdentityUser), archived.UpdatedActorType)
	require.Equal(t, uint64(1), archived.UpdatedActorID)
	_, err = service.GetRole(context.Background(), testAuth(1, 10), created.Id)
	require.ErrorIs(t, err, pkg_errors.AccessManagementRoleNotFoundError{})
}

func TestArchiveCustomRoleRejectsActiveProjectCredentialReference(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"assigned_token", "Assigned Token", "",
		ScopeProject, SubjectProjectToken, []uint64{34},
	)
	require.NoError(t, err)
	require.NoError(t, database.Exec(
		"INSERT INTO project_credentials(id,organization_id,project_id,role_definition_id,role_subject_type,status) VALUES (1,10,100,?,'PROJECT_TOKEN','ACTIVE')",
		created.Id,
	).Error)

	require.ErrorIs(t, service.ArchiveCustomRole(context.Background(), testAuth(1, 10),
		created.Id, "Retire", "archive-assigned",
	), pkg_errors.AccessManagementRoleChangeNotAllowedError{})
	require.NoError(t, database.Exec("UPDATE project_credentials SET status='ARCHIEVE' WHERE id=1").Error)
	require.NoError(t, service.ArchiveCustomRole(context.Background(), testAuth(1, 10),
		created.Id, "Retire", "archive-assigned",
	))
}

func TestArchiveCustomRoleRejectsActiveOrInvitedUserAssignment(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"assigned_user", "Assigned User", "", ScopeProject, SubjectUser, []uint64{34})
	require.NoError(t, err)
	require.NoError(t, database.Exec("INSERT INTO user_project_roles(id,user_auth_id,project_id,role_definition_id,status) VALUES (20,7,100,?,'INVITED')", created.Id).Error)
	require.ErrorIs(t, service.ArchiveCustomRole(context.Background(), testAuth(1, 10), created.Id, "Retire", "archive-user"), pkg_errors.AccessManagementRoleChangeNotAllowedError{})
}

func TestBuiltInMutationFailsPrecondition(t *testing.T) {
	service, _ := newTestService(t)
	_, err := service.UpdateCustomRole(context.Background(), testAuth(1, 10),
		101, "Changed Owner", "Not allowed",
	)
	require.ErrorIs(t, err, pkg_errors.AccessManagementRoleChangeNotAllowedError{})
}

func TestUpdateCustomRoleUsesTypedUpdate(t *testing.T) {
	service, database := newTestService(t)
	created, err := service.CreateCustomRole(context.Background(), testAuth(1, 10),
		"project_editor", "Project Editor", "Can edit projects",
		ScopeProject, SubjectUser, []uint64{36, 34},
	)
	require.NoError(t, err)

	updated, err := service.UpdateCustomRole(context.Background(), testAuth(1, 10), created.Id, "Project Maintainer", "")
	require.NoError(t, err)
	require.Equal(t, "Project Maintainer", updated.Name)
	require.Empty(t, updated.Description)
	require.False(t, time.Time(updated.UpdatedDate).IsZero())
	require.Equal(t, string(authentication.IdentityUser), updated.UpdatedActorType)
	require.Equal(t, uint64(1), updated.UpdatedActorID)
	require.Len(t, updated.RolePermissions, 2)
	require.Equal(t, []uint64{34, 36}, []uint64{updated.RolePermissions[0].PermissionDefinition.Id, updated.RolePermissions[1].PermissionDefinition.Id})

	var stored internal_entity.RoleDefinition
	require.NoError(t, database.Where("id = ?", created.Id).First(&stored).Error)
	require.Equal(t, updated.Name, stored.Name)
	require.Empty(t, stored.Description)
	require.False(t, time.Time(stored.UpdatedDate).IsZero())
}

func TestListPermissionsRejectsCanceledContext(t *testing.T) {
	service, _ := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := service.ListPermissions(ctx, testAuth(1, 10), nil, &protos.Paginate{Page: 1, PageSize: 25})
	require.True(t, errors.Is(err, context.Canceled), "error = %v", err)
}
