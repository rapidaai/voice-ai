// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_project_service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/authentication"
)

func TestPrincipalAuthenticationServiceResolvesActiveProjectCredential(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE organizations (id INTEGER PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE projects (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, name TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE project_credentials (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, project_id INTEGER NOT NULL, role_definition_id INTEGER NOT NULL, role_subject_type TEXT NOT NULL, status TEXT NOT NULL, key TEXT NOT NULL)`,
		`CREATE TABLE role_definitions (id INTEGER PRIMARY KEY, organization_id INTEGER, key TEXT NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL, scope_type TEXT NOT NULL, subject_type TEXT NOT NULL, definition_type TEXT NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE role_permissions (id INTEGER PRIMARY KEY, role_definition_id INTEGER NOT NULL, permission_definition_id INTEGER NOT NULL, status TEXT NOT NULL)`,
		`CREATE TABLE permission_definitions (id INTEGER PRIMARY KEY, key TEXT NOT NULL, scope_type TEXT NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL, status TEXT NOT NULL)`,
		`INSERT INTO organizations VALUES (9,'Organization','ACTIVE')`,
		`INSERT INTO projects VALUES (10,9,'Project','ACTIVE')`,
		`INSERT INTO role_definitions VALUES (12,NULL,'project_token','Project Token','Token role','PROJECT','PROJECT_TOKEN','BUILT_IN','ACTIVE')`,
		`INSERT INTO project_credentials VALUES (11,9,10,12,'PROJECT_TOKEN','ACTIVE','project-secret')`,
		`INSERT INTO permission_definitions VALUES (1,'project.resources.read','PROJECT','Read','Read resources','ACTIVE'),(2,'project.resources.update','PROJECT','Update','Update resources','ACTIVE')`,
		`INSERT INTO role_permissions VALUES (1,12,2,'ACTIVE'),(2,12,1,'ACTIVE')`,
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := NewPrincipalAuthenticationService(nil, &projectTestPostgres{db: database})
	value, err := service.ResolveProjectCredential(context.Background(), "project-secret")
	if err != nil {
		t.Fatal(err)
	}
	principal := value.Principal()
	if principal.Subject().ID != 11 || principal.Tenant().OrganizationID != 9 || principal.Tenant().ProjectID != 10 ||
		len(principal.Authorities()) != 2 || principal.Authorities()[0] != authentication.PermissionKey("project.resources.read") {
		t.Fatalf("principal = %+v", principal)
	}
}
