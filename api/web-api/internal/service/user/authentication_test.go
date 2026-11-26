// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/ciphers"
	pkg_errors "github.com/rapidaai/pkg/errors"
)

func createAuthenticationBillingTables(t *testing.T, database *gorm.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE billing_plans (id INTEGER PRIMARY KEY, created_date DATETIME, updated_date DATETIME, name TEXT NOT NULL, slug TEXT NOT NULL, description TEXT NOT NULL, is_default INTEGER NOT NULL, is_active INTEGER NOT NULL, sort_order INTEGER NOT NULL, price_monthly INTEGER NOT NULL, price_yearly INTEGER NOT NULL, currency TEXT NOT NULL, stripe_url TEXT NOT NULL)`,
		`CREATE TABLE billing_plan_quotas (id INTEGER PRIMARY KEY, created_date DATETIME, updated_date DATETIME, billing_plan_id INTEGER NOT NULL, resource_type TEXT NOT NULL, quota_limit INTEGER NOT NULL, unit TEXT NOT NULL)`,
		`CREATE TABLE billing_subscriptions (id INTEGER PRIMARY KEY, created_date DATETIME, updated_date DATETIME, status TEXT NOT NULL, created_actor_type TEXT, created_actor_id INTEGER, updated_actor_type TEXT, updated_actor_id INTEGER, organization_id INTEGER NOT NULL, billing_plan_id INTEGER NOT NULL, billing_interval TEXT NOT NULL, current_period_start DATETIME NOT NULL, current_period_end DATETIME, metadata TEXT NOT NULL)`,
	} {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrincipalAuthenticationServiceBindsExactActiveSession(t *testing.T) {
	_, database, _ := newAssignmentServiceTest(t)
	if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (81,'ACTIVE',1,'auth-token','exact-token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	createAuthenticationBillingTables(t, database)
	service := NewPrincipalAuthenticationService(nil, &userServiceTestPostgres{db: database})
	value, err := service.ResolveUserCredential(context.Background(), "exact-token", 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	principal := value.Principal()
	if principal.Subject().ID != 1 || principal.Proof().Reference.Value != "81" || principal.Tenant().OrganizationID != 10 || len(principal.Authorities()) != 2 {
		t.Fatalf("principal = %+v", principal)
	}

	for _, token := range []string{"wrong-token", ""} {
		_, err := service.ResolveUserCredential(context.Background(), token, 1, 0, 0)
		var invalid pkg_errors.AuthenticationCredentialInvalidError
		if !errors.As(err, &invalid) {
			t.Fatalf("ResolveUserCredential(%q) error = %v", token, err)
		}
	}
	if err := database.Exec(`UPDATE user_auth_tokens SET expire_at = ? WHERE id = 81`, time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	_, err = service.ResolveUserCredential(context.Background(), "exact-token", 1, 0, 0)
	var invalid pkg_errors.AuthenticationCredentialInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("expired session error = %v", err)
	}
}

func TestPrincipalAuthenticationServiceSelectsTenantAcrossMultipleOrganizations(t *testing.T) {
	_, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	statements := []string{
		`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (83,'ACTIVE',1,'auth-token','multi-token',?)`,
		`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',1,20,102,'ORGANIZATION','USER')`,
		`INSERT INTO user_project_roles(status,user_auth_id,project_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',1,200,109,'PROJECT','USER')`,
	}
	for index, statement := range statements {
		arguments := []any(nil)
		if index == 0 {
			arguments = []any{time.Now().Add(time.Hour)}
		}
		if err := database.Exec(statement, arguments...).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := NewPrincipalAuthenticationService(nil, &userServiceTestPostgres{db: database})

	tests := []struct {
		name                   string
		selectedOrganizationID uint64
		selectedProjectID      uint64
		wantTenant             authentication.Tenant
		wantInvalid            bool
	}{
		{name: "no selection remains tenantless", wantTenant: authentication.Tenant{}},
		{name: "explicit organization", selectedOrganizationID: 20, wantTenant: authentication.Tenant{OrganizationID: 20}},
		{name: "project infers organization", selectedProjectID: 200, wantTenant: authentication.Tenant{OrganizationID: 20, ProjectID: 200}},
		{name: "matching organization and project", selectedOrganizationID: 20, selectedProjectID: 200, wantTenant: authentication.Tenant{OrganizationID: 20, ProjectID: 200}},
		{name: "unknown organization", selectedOrganizationID: 30, wantInvalid: true},
		{name: "mismatched organization and project", selectedOrganizationID: 10, selectedProjectID: 200, wantInvalid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := service.ResolveUserCredential(context.Background(), "multi-token", 1, test.selectedOrganizationID, test.selectedProjectID)
			if test.wantInvalid {
				var invalid pkg_errors.AuthenticationCredentialInvalidError
				if !errors.As(err, &invalid) {
					t.Fatalf("ResolveUserCredential() error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if value.Principal().Tenant() != test.wantTenant || len(value.Organizations()) != 2 {
				t.Fatalf("tenant = %+v, organizations = %d", value.Principal().Tenant(), len(value.Organizations()))
			}
		})
	}
}

func TestSessionProducersKeepMultipleOrganizationsTenantless(t *testing.T) {
	_, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	if err := database.Exec(`INSERT INTO user_organization_roles(status,user_auth_id,organization_id,role_definition_id,role_scope_type,role_subject_type) VALUES ('ACTIVE',1,20,102,'ORGANIZATION','USER')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`UPDATE user_auths SET password = ? WHERE id = 1`, ciphers.Hash("secret")).Error; err != nil {
		t.Fatal(err)
	}
	service := NewUserService(nil, &userServiceTestPostgres{db: database})
	created, err := service.CreateAuthentication(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := service.AuthenticationForPrincipal(context.Background(), created.Authentication.Principal())
	if err != nil {
		t.Fatal(err)
	}
	password, err := service.AuthenticateUser(context.Background(), "owner@example.com", "secret")
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]*internal_service.UserAuthentication{
		"create": created, "refresh": refreshed, "password": password,
	} {
		if value.Authentication.Principal().Tenant() != (authentication.Tenant{}) || len(value.Authentication.Organizations()) != 2 {
			t.Fatalf("%s tenant = %+v, organizations = %d", name, value.Authentication.Principal().Tenant(), len(value.Authentication.Organizations()))
		}
	}
}

func TestPrincipalAuthenticationServicePropagatesAuthorityQueryFailure(t *testing.T) {
	_, database, _ := newAssignmentServiceTest(t)
	if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (82,'ACTIVE',1,'auth-token','exact-token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`DROP TABLE role_permissions`).Error; err != nil {
		t.Fatal(err)
	}
	service := NewPrincipalAuthenticationService(nil, &userServiceTestPostgres{db: database})
	_, err := service.ResolveUserCredential(context.Background(), "exact-token", 1, 0, 0)
	if err == nil {
		t.Fatal("authority query failure was swallowed")
	}
	var invalid pkg_errors.AuthenticationCredentialInvalidError
	if errors.As(err, &invalid) {
		t.Fatalf("database failure was converted to credential denial: %v", err)
	}
}

func TestUserAuthenticationIncludesBillingPlanFromCanonicalSnapshot(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	statements := []string{
		`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (81,'ACTIVE',1,'auth-token','exact-token',?)`,
		`INSERT INTO billing_plans(id,name,slug,description,is_default,is_active,sort_order,price_monthly,price_yearly,currency,stripe_url) VALUES (201,'Enterprise','enterprise','',0,1,3,0,0,'usd','')`,
		`INSERT INTO billing_plan_quotas(id,billing_plan_id,resource_type,quota_limit,unit) VALUES (301,201,'projects',-1,'count')`,
		`INSERT INTO billing_subscriptions(id,organization_id,billing_plan_id,billing_interval,current_period_start,metadata,status) VALUES (401,10,201,'monthly',CURRENT_TIMESTAMP,'{}','ACTIVE')`,
	}
	for index, statement := range statements {
		var err error
		if index == 0 {
			err = database.Exec(statement, time.Now().Add(time.Hour)).Error
		} else {
			err = database.Exec(statement).Error
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.AuthenticationForPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	organization, ok := result.Authentication.SelectedOrganization()
	if !ok || organization.BillingPlan == nil || organization.BillingPlan.Slug != "enterprise" ||
		len(organization.BillingPlan.Quotas) != 1 || organization.BillingPlan.Quotas[0].QuotaLimit != -1 {
		t.Fatalf("organization = %+v", organization)
	}
	if result.Token.Id != 81 || result.Authentication.Principal().Proof().Reference.Value != "81" {
		t.Fatalf("authentication result = %+v, proof = %+v", result.Token, result.Authentication.Principal().Proof())
	}
}

func TestUserAuthenticationForPrincipalPreservesExactSession(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	for _, session := range []struct {
		id    uint64
		token string
	}{{id: 81, token: "requested"}, {id: 82, token: "newer"}} {
		if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (?,'ACTIVE',1,'auth-token',?,?)`, session.id, session.token, time.Now().Add(time.Hour)).Error; err != nil {
			t.Fatal(err)
		}
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		[]authentication.PermissionKey{"organization.access.manage", "organization.members.invite"},
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.AuthenticationForPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	if result.Token.Id != 81 || result.Token.Token != "requested" || result.Authentication.Principal().Proof().Reference.Value != "81" {
		t.Fatalf("result = %+v", result)
	}
}

func TestUserAuthenticationPropagatesBillingFailure(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (81,'ACTIVE',1,'auth-token','exact-token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticationForPrincipal(context.Background(), principal); err == nil {
		t.Fatal("missing billing schema was ignored")
	}
}

func TestPasswordAuthenticationIssuesFreshSession(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	if err := database.Exec(`UPDATE user_auths SET password = ? WHERE id = 1`, ciphers.Hash("secret")).Error; err != nil {
		t.Fatal(err)
	}
	result, err := service.AuthenticateUser(context.Background(), "owner@example.com", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token.Id == 0 || result.Token.Token == "" || result.Authentication.Principal().Proof().Reference.Value == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestPasswordAuthenticationDoesNotPersistSessionWhenAggregateFails(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	if err := database.Exec(`UPDATE user_auths SET password = ? WHERE id = 1`, ciphers.Hash("secret")).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateUser(context.Background(), "owner@example.com", "secret"); err == nil {
		t.Fatal("missing billing schema was ignored")
	}
	var sessions int64
	if err := database.Table("user_auth_tokens").Where("user_auth_id = ?", 1).Count(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	if sessions != 0 {
		t.Fatalf("persisted sessions = %d", sessions)
	}
}

func TestCreateAuthenticationReturnsItsExactPersistedSession(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	result, err := service.CreateAuthentication(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	var session internal_entity.UserAuthToken
	if err := database.Where("id = ? AND user_auth_id = ?", result.Token.Id, 1).Take(&session).Error; err != nil {
		t.Fatal(err)
	}
	if session.Token != result.Token.Token || result.Authentication.Principal().Proof().Reference.Value != strconv.FormatUint(session.Id, 10) {
		t.Fatalf("session = %+v, result = %+v", session, result)
	}
}

func TestCreateAuthenticationKeepsConcurrentSessionsDistinct(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	results := make(chan *internal_service.UserAuthentication, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := service.CreateAuthentication(context.Background(), 1)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
	seen := make(map[uint64]struct{}, 2)
	for result := range results {
		if result.Authentication.Principal().Proof().Reference.Value != strconv.FormatUint(result.Token.Id, 10) {
			t.Fatalf("result = %+v", result)
		}
		seen[result.Token.Id] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("session IDs = %v", seen)
	}
}

func TestUserAuthenticationExcludesArchivedPermissionDefinitions(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (81,'ACTIVE',1,'auth-token','exact-token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`UPDATE permission_definitions SET status = 'ARCHIEVE' WHERE id = 1`).Error; err != nil {
		t.Fatal(err)
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.AuthenticationForPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	permissions := result.Authentication.Permissions()
	if len(permissions) != 1 || permissions[0] != "organization.access.manage" {
		t.Fatalf("permissions = %v", permissions)
	}
}

func TestUserAuthenticationCanonicalizesLegacyDisplayText(t *testing.T) {
	service, database, _ := newAssignmentServiceTest(t)
	createAuthenticationBillingTables(t, database)
	if err := database.Exec(`INSERT INTO user_auth_tokens(id,status,user_auth_id,token_type,token,expire_at) VALUES (81,'ACTIVE',1,'auth-token','exact-token',?)`, time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`UPDATE user_auths SET name = ' Owner ', email = ' owner@example.com ' WHERE id = 1`,
		`UPDATE organizations SET name = ' Acme ' WHERE id = 10`,
		`UPDATE role_definitions SET name = ' Owner ', description = ' Full access ' WHERE id = 101`,
		`UPDATE permission_definitions SET name = ' Invite members ', description = ' Invite users ' WHERE id = 1`,
	} {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "81"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.AuthenticationForPrincipal(context.Background(), principal)
	if err != nil {
		t.Fatal(err)
	}
	user, ok := result.Authentication.User()
	organization, organizationOK := result.Authentication.SelectedOrganization()
	if !ok || !organizationOK || user.Name != "Owner" || user.Email != "owner@example.com" || organization.Name != "Acme" ||
		organization.Roles[0].Name != "Owner" || organization.Roles[0].Description != "Full access" {
		t.Fatalf("user = %+v, organization = %+v", user, organization)
	}
}
