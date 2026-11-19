// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func newAssignmentServiceTest(t *testing.T) (internal_service.UserService, *gorm.DB, authentication.Principal) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	statements := []string{
		`CREATE TABLE organizations (id INTEGER PRIMARY KEY, status TEXT NOT NULL, name TEXT)`,
		`CREATE TABLE projects (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, status TEXT NOT NULL, name TEXT, description TEXT)`,
		`CREATE TABLE user_auths (id INTEGER PRIMARY KEY AUTOINCREMENT, created_date DATETIME, updated_date DATETIME, status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER, updated_actor_type TEXT, updated_actor_id INTEGER, name TEXT, email TEXT UNIQUE, password TEXT, source TEXT)`,
		`CREATE TABLE user_auth_tokens (id INTEGER PRIMARY KEY AUTOINCREMENT, created_date DATETIME, updated_date DATETIME, status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER, updated_actor_type TEXT, updated_actor_id INTEGER, user_auth_id INTEGER, token_type TEXT, token TEXT, expire_at DATETIME)`,
		`CREATE TABLE permission_definitions (id INTEGER PRIMARY KEY, key TEXT NOT NULL, scope_type TEXT NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE role_definitions (id INTEGER PRIMARY KEY, organization_id INTEGER, key TEXT NOT NULL, name TEXT NOT NULL, description TEXT, scope_type TEXT NOT NULL, subject_type TEXT NOT NULL, definition_type TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE role_permissions (id INTEGER PRIMARY KEY AUTOINCREMENT, role_definition_id INTEGER NOT NULL, permission_definition_id INTEGER NOT NULL, scope_type TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE user_organization_roles (id INTEGER PRIMARY KEY AUTOINCREMENT, created_date DATETIME, updated_date DATETIME, status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER, updated_actor_type TEXT, updated_actor_id INTEGER, user_auth_id INTEGER NOT NULL, organization_id INTEGER NOT NULL, role_definition_id INTEGER NOT NULL, role_scope_type TEXT NOT NULL, role_subject_type TEXT NOT NULL)`,
		`CREATE UNIQUE INDEX user_organization_roles_current_assignment_idx ON user_organization_roles(user_auth_id, organization_id, role_definition_id) WHERE status IN ('ACTIVE','INVITED')`,
		`CREATE TABLE user_project_roles (id INTEGER PRIMARY KEY AUTOINCREMENT, created_date DATETIME, updated_date DATETIME, status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER, updated_actor_type TEXT, updated_actor_id INTEGER, user_auth_id INTEGER NOT NULL, project_id INTEGER NOT NULL, role_definition_id INTEGER NOT NULL, role_scope_type TEXT NOT NULL, role_subject_type TEXT NOT NULL)`,
		`CREATE UNIQUE INDEX user_project_roles_current_assignment_idx ON user_project_roles(user_auth_id, project_id, role_definition_id) WHERE status IN ('ACTIVE','INVITED')`,
		`INSERT INTO organizations(id,status,name) VALUES (10,'ACTIVE','Acme'),(20,'ACTIVE','Other')`,
		`INSERT INTO projects(id,organization_id,status,name,description) VALUES (100,10,'ACTIVE','Alpha',''),(101,10,'ACTIVE','Beta',''),(200,20,'ACTIVE','Foreign','')`,
		`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (1,'ACTIVE','Owner','owner@example.com','hash','direct')`,
		`INSERT INTO permission_definitions(id,key,scope_type,name,description,status) VALUES (1,'organization.members.invite','ORGANIZATION','Invite members','','ACTIVE'),(2,'organization.access.manage','ORGANIZATION','Manage access','','ACTIVE')`,
		`INSERT INTO role_definitions(id,organization_id,key,name,description,scope_type,subject_type,definition_type,status) VALUES
		 (101,NULL,'organization_owner','Owner','','ORGANIZATION','USER','BUILT_IN','ACTIVE'),
		 (102,NULL,'organization_admin','Admin','','ORGANIZATION','USER','BUILT_IN','ACTIVE'),
		 (108,NULL,'organization_member','Member','','ORGANIZATION','USER','BUILT_IN','ACTIVE'),
		 (109,NULL,'project_admin','Project admin','','PROJECT','USER','BUILT_IN','ACTIVE'),
		 (110,NULL,'project_developer','Developer','','PROJECT','USER','BUILT_IN','ACTIVE'),
		 (112,NULL,'project_viewer','Viewer','','PROJECT','USER','BUILT_IN','ACTIVE'),
		 (201,10,'custom_org','Custom org','','ORGANIZATION','USER','CUSTOM','ACTIVE'),
		 (202,20,'foreign_org','Foreign org','','ORGANIZATION','USER','CUSTOM','ACTIVE'),
		 (203,10,'custom_project','Custom project','','PROJECT','USER','CUSTOM','ACTIVE'),
		 (204,20,'foreign_project','Foreign project','','PROJECT','USER','CUSTOM','ACTIVE')`,
		`INSERT INTO role_permissions(role_definition_id,permission_definition_id,scope_type,status) VALUES (101,1,'ORGANIZATION','ACTIVE'),(101,2,'ORGANIZATION','ACTIVE'),(102,1,'ORGANIZATION','ACTIVE'),(102,2,'ORGANIZATION','ACTIVE')`,
		`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',1,10,101,'ORGANIZATION','USER')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("execute schema statement: %v", err)
		}
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "assignment-test"}},
		[]authentication.PermissionKey{"organization.members.invite", "organization.access.manage"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return NewUserService(logger, &userServiceTestPostgres{db: db}), db, auth
}

func TestInviteUserToOrganizationPersistsAllAssignmentsAtomically(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	user, projects, err := service.InviteUserToOrganization(context.Background(), auth, "new@example.com", 10, []uint64{102, 201}, []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110, 203}}})
	if err != nil {
		t.Fatal(err)
	}
	if user.Status != type_enums.RECORD_INVITED || len(projects) != 1 || projects[0].Id != 100 {
		t.Fatalf("invite result = %+v, %+v", user, projects)
	}
	var organizationAssignments, projectAssignments int64
	if err := db.Model(&internal_entity.UserOrganizationRole{}).Where("user_auth_id = ? AND status = ?", user.Id, type_enums.RECORD_INVITED).Count(&organizationAssignments).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ? AND status = ?", user.Id, type_enums.RECORD_INVITED).Count(&projectAssignments).Error; err != nil {
		t.Fatal(err)
	}
	if organizationAssignments != 2 || projectAssignments != 2 {
		t.Fatalf("assignment counts = %d/%d", organizationAssignments, projectAssignments)
	}
}

func TestInviteUserToOrganizationRejectsPrincipalTenantMismatchBeforePersistence(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	_, _, err := service.InviteUserToOrganization(context.Background(), auth, "outside@example.com", 20, []uint64{108}, nil)
	var denied pkg_errors.RouteAuthorizationDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("InviteUserToOrganization() error = %v", err)
	}
	var count int64
	if err := db.Model(&internal_entity.UserAuth{}).Where("email = ?", "outside@example.com").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cross-tenant identity count = %d", count)
	}
}

func TestInviteUserToOrganizationRejectsInvalidDuplicateAndCrossTenantSelections(t *testing.T) {
	tests := []struct {
		name         string
		organization []uint64
		projects     []internal_service.ProjectRoleSelection
	}{
		{name: "duplicate organization role", organization: []uint64{108, 108}},
		{name: "cross tenant organization role", organization: []uint64{202}},
		{name: "owner role", organization: []uint64{101}},
		{name: "duplicate project", organization: []uint64{108}, projects: []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110}}, {ProjectID: 100, RoleDefinitionIDs: []uint64{112}}}},
		{name: "duplicate project role", organization: []uint64{108}, projects: []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110, 110}}}},
		{name: "cross tenant project", organization: []uint64{108}, projects: []internal_service.ProjectRoleSelection{{ProjectID: 200, RoleDefinitionIDs: []uint64{110}}}},
		{name: "cross tenant project role", organization: []uint64{108}, projects: []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{204}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, db, auth := newAssignmentServiceTest(t)
			_, _, err := service.InviteUserToOrganization(context.Background(), auth, "rejected@example.com", 10, test.organization, test.projects)
			var unavailable pkg_errors.RoleAssignmentSelectionUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("InviteUserToOrganization() error = %v", err)
			}
			var count int64
			if err := db.Model(&internal_entity.UserAuth{}).Where("email = ?", "rejected@example.com").Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("rejected identity count = %d", count)
			}
		})
	}
}

func TestAddUserToProjectsUsesInvitedIdentityStatus(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	if err := db.Exec(`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (2,'INVITED','Invitee','invitee@example.com','hash','invited-by-other')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('INVITED',2,10,108,'ORGANIZATION','USER')`).Error; err != nil {
		t.Fatal(err)
	}
	projects, err := service.AddUserToProjects(context.Background(), auth, 2, 10, []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110, 112}}})
	if err != nil || len(projects) != 1 {
		t.Fatalf("AddUserToProjects() = %+v, %v", projects, err)
	}
	var count int64
	if err := db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ? AND status = ?", 2, type_enums.RECORD_INVITED).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("invited project assignment count = %d", count)
	}
}

func TestInviteUserToOrganizationRollsBackIdentityOnAssignmentFailure(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	if err := db.Exec(`CREATE TRIGGER fail_viewer_assignment BEFORE INSERT ON user_project_roles WHEN NEW.role_definition_id = 112 BEGIN SELECT RAISE(ABORT, 'injected failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	_, _, err := service.InviteUserToOrganization(context.Background(), auth, "rollback@example.com", 10, []uint64{108}, []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110, 112}}})
	if err == nil {
		t.Fatal("InviteUserToOrganization() error = nil")
	}
	var userCount, organizationCount, projectCount int64
	if err := db.Model(&internal_entity.UserAuth{}).Where("email = ?", "rollback@example.com").Count(&userCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&internal_entity.UserOrganizationRole{}).Where("user_auth_id > 1").Count(&organizationCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&internal_entity.UserProjectRole{}).Count(&projectCount).Error; err != nil {
		t.Fatal(err)
	}
	if userCount != 0 || organizationCount != 0 || projectCount != 0 {
		t.Fatalf("rollback counts = %d/%d/%d", userCount, organizationCount, projectCount)
	}
}

func TestAddUserToProjectsConcurrentRequestFailsWithoutPartialRows(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	if err := db.Exec(`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (2,'ACTIVE','Member','member@example.com','hash','direct')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',2,10,108,'ORGANIZATION','USER')`).Error; err != nil {
		t.Fatal(err)
	}
	selection := []internal_service.ProjectRoleSelection{{ProjectID: 100, RoleDefinitionIDs: []uint64{110, 112}}}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.AddUserToProjects(context.Background(), auth, 2, 10, selection)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	var successes, conflicts int
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		var conflict pkg_errors.ProjectMembershipConflictError
		if errors.As(err, &conflict) {
			conflicts++
			continue
		}
		t.Fatalf("concurrent error = %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent results = %d successes, %d conflicts", successes, conflicts)
	}
	var count int64
	if err := db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 2).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("project assignment count = %d", count)
	}
}

func TestAuthenticationProjectionUnionsActivePermissionsAndExcludesInvitedAssignments(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, db)
	if err := db.Exec(`CREATE TABLE user_feature_permissions (id INTEGER PRIMARY KEY, user_auth_id INTEGER, feature TEXT, is_enabled INTEGER, status TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_auth_tokens(status,user_auth_id,token_type,token,expire_at) VALUES ('ACTIVE',1,'auth-token','token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO permission_definitions(id,key,scope_type,name,description,status) VALUES (4,'organization.custom.active','ORGANIZATION','Custom active','','ACTIVE'),(5,'organization.invited.only','ORGANIZATION','Invited only','','ACTIVE')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO role_permissions(role_definition_id,permission_definition_id,scope_type,status) VALUES (201,4,'ORGANIZATION','ACTIVE'),(202,5,'ORGANIZATION','ACTIVE')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',1,10,201,'ORGANIZATION','USER'),('INVITED',1,10,202,'ORGANIZATION','USER')`).Error; err != nil {
		t.Fatal(err)
	}
	result, err := service.CreateAuthentication(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	organization, ok := result.Authentication.SelectedOrganization()
	if !ok || len(organization.Roles) != 2 {
		t.Fatalf("organization projection = %+v", organization)
	}
	wantPermissions := []string{"organization.access.manage", "organization.custom.active", "organization.members.invite"}
	permissions := result.Authentication.Permissions()
	if len(permissions) != len(wantPermissions) {
		t.Fatalf("effective permissions = %v", permissions)
	}
	for index, permission := range wantPermissions {
		if permissions[index] != authentication.PermissionKey(permission) {
			t.Fatalf("effective permissions = %v", permissions)
		}
	}
	for _, role := range organization.Roles {
		if role.ID == 202 {
			t.Fatal("invited assignment appeared in authentication projection")
		}
	}
}

func TestAuthenticationProjectionUsesCanonicalProjectAssignmentFilters(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, db)
	if err := db.Exec(`INSERT INTO user_auth_tokens(status,user_auth_id,token_type,token,expire_at) VALUES ('ACTIVE',1,'auth-token','token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_project_roles(status,user_auth_id,project_id,role_definition_id,role_scope_type,role_subject_type) VALUES
		('ACTIVE',1,100,109,'PROJECT','USER'),
		('ACTIVE',1,100,110,'ORGANIZATION','USER'),
		('ACTIVE',1,100,204,'PROJECT','USER')`).Error; err != nil {
		t.Fatal(err)
	}

	result, err := service.CreateAuthentication(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	organization, ok := result.Authentication.SelectedOrganization()
	if !ok {
		t.Fatalf("authentication = %+v", result.Authentication)
	}
	projects := organization.Projects
	if len(projects) != 1 || len(projects[0].Roles) != 1 || projects[0].Roles[0].ID != 109 {
		t.Fatalf("canonical project projection = %+v", projects)
	}
}

func TestAuthenticationProjectionFailsClosedOnProjectAssignmentQueryError(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	if err := db.Exec(`INSERT INTO user_auth_tokens(status,user_auth_id,token_type,token,expire_at) VALUES ('ACTIVE',1,'auth-token','token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("project assignment query failed")
	if err := db.Callback().Query().Before("gorm:query").Register("test:fail_project_assignments", func(tx *gorm.DB) {
		if tx.Statement.Table == "user_project_roles" {
			tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := service.CreateAuthentication(context.Background(), 1); !errors.Is(err, wantErr) {
		t.Fatalf("CreateAuthentication() error = %v, want %v", err, wantErr)
	}
}

func TestReplaceOrganizationRolesAppliesCompleteSetAndPreservesInvitedStatus(t *testing.T) {
	service, db, auth := newAssignmentServiceTest(t)
	if err := db.Exec(`INSERT INTO user_auths(id,status,name,email,password,source) VALUES (2,'INVITED','Invitee','replace@example.com','hash','invited-by-other')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',2,10,108,'ORGANIZATION','USER'),('INVITED',2,10,102,'ORGANIZATION','USER')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ReplaceOrganizationRoles(context.Background(), auth, 2, 10, []uint64{102, 201}); err != nil {
		t.Fatal(err)
	}
	var current []internal_entity.UserOrganizationRole
	if err := db.Where("user_auth_id = ? AND status IN ?", 2, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_INVITED}).Order("role_definition_id ASC").Find(&current).Error; err != nil {
		t.Fatal(err)
	}
	if len(current) != 2 || current[0].RoleDefinitionID != 102 || current[1].RoleDefinitionID != 201 {
		t.Fatalf("current assignments = %+v", current)
	}
	for _, assignment := range current {
		if assignment.Status != type_enums.RECORD_INVITED {
			t.Fatalf("assignment %d status = %s", assignment.RoleDefinitionID, assignment.Status)
		}
	}
	var archived int64
	if err := db.Model(&internal_entity.UserOrganizationRole{}).Where("user_auth_id = ? AND role_definition_id = ? AND status = ?", 2, 108, type_enums.RECORD_ARCHIEVE).Count(&archived).Error; err != nil {
		t.Fatal(err)
	}
	if archived != 1 {
		t.Fatalf("archived omitted assignment count = %d", archived)
	}
}

func TestReplaceOrganizationRolesRejectsOwnerMutation(t *testing.T) {
	service, _, auth := newAssignmentServiceTest(t)
	err := service.ReplaceOrganizationRoles(context.Background(), auth, 1, 10, []uint64{108})
	var ownerError pkg_errors.OrganizationOwnerMutationError
	if !errors.As(err, &ownerError) {
		t.Fatalf("ReplaceOrganizationRoles() error = %v", err)
	}
}
