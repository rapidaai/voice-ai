package internal_project_service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	internal_accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type projectTestPostgres struct {
	db *gorm.DB
}

func (p *projectTestPostgres) Connect(context.Context) error    { return nil }
func (p *projectTestPostgres) Name() string                     { return "project-test" }
func (p *projectTestPostgres) IsConnected(context.Context) bool { return true }
func (p *projectTestPostgres) Disconnect(context.Context) error { return nil }
func (p *projectTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *projectTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*projectTestPostgres)(nil)

func TestProjectCredentialWritersPersistActorIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE project_credentials (
        id INTEGER PRIMARY KEY,
        project_id INTEGER NOT NULL,
        organization_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        key TEXT NOT NULL,
        status TEXT NOT NULL,
        created_actor_type TEXT,
        created_actor_id INTEGER,
        updated_actor_type TEXT,
        updated_actor_id INTEGER,
        role_definition_id INTEGER NOT NULL,
        role_subject_type TEXT NOT NULL,
        created_date DATETIME NOT NULL,
        updated_date DATETIME
    )`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE projects (id INTEGER PRIMARY KEY, organization_id INTEGER NOT NULL, status TEXT NOT NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE role_definitions (
        id INTEGER PRIMARY KEY, organization_id INTEGER, name TEXT NOT NULL, scope_type TEXT NOT NULL,
        subject_type TEXT NOT NULL, definition_type TEXT NOT NULL, status TEXT NOT NULL
    )`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO projects(id,organization_id,status) VALUES (7,9,'ACTIVE'),(8,10,'ACTIVE'),(9,9,'ARCHIEVE')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO role_definitions(id,organization_id,name,scope_type,subject_type,definition_type,status) VALUES
        (117,NULL,'Runtime','PROJECT','PROJECT_TOKEN','BUILT_IN','ACTIVE'),
        (118,9,'Support','PROJECT','PROJECT_TOKEN','CUSTOM','ACTIVE'),
        (119,10,'Foreign','PROJECT','PROJECT_TOKEN','CUSTOM','ACTIVE'),
        (120,NULL,'Organization','ORGANIZATION','USER','BUILT_IN','ACTIVE'),
        (121,NULL,'Archived','PROJECT','PROJECT_TOKEN','BUILT_IN','ARCHIEVE')`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewProjectService(logger, &projectTestPostgres{db: db})
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.Tenant{OrganizationID: 9, ProjectID: 7},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "project-service-test"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.Get(context.Background(), auth, 7)
	if err != nil || project.Id != 7 || project.OrganizationId != 9 {
		t.Fatalf("Get() = %+v, %v", project, err)
	}
	wrongOrganizationAuth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.Tenant{OrganizationID: 10, ProjectID: 7},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "project-service-wrong-organization"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if project, err = service.Get(context.Background(), wrongOrganizationAuth, 7); !errors.Is(err, gorm.ErrRecordNotFound) || project != nil {
		t.Fatalf("Get() cross-tenant = %+v, %v", project, err)
	}

	credential, err := service.CreateCredential(context.Background(), auth, "primary", 7, 9, 117)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}
	if credential.Id == 0 || credential.Name != "primary" || credential.RoleDefinitionID != 117 || credential.RoleDefinition.Name != "Runtime" || credential.RoleSubjectType != internal_accessmanagement.SubjectProjectToken || credential.CreatedActorType != "user" || credential.CreatedActorID != 11 {
		t.Fatalf("created credential = %+v", credential)
	}

	customCredential, err := service.CreateCredential(context.Background(), auth, "custom", 7, 9, 118)
	if err != nil {
		t.Fatalf("CreateCredential() custom error = %v", err)
	}
	if customCredential.RoleDefinitionID != 118 || customCredential.RoleDefinition.Name != "Support" {
		t.Fatalf("custom credential role = %d", customCredential.RoleDefinitionID)
	}

	count, credentials, err := service.GetAllCredential(context.Background(), auth, 7, 9, []*protos.Criteria{{Key: "role", Logic: "contains", Value: "support"}}, &protos.Paginate{Page: 1, PageSize: 25})
	if err != nil {
		t.Fatalf("GetAllCredential() error = %v", err)
	}
	if count != 1 || len(credentials) != 1 || credentials[0].Id != customCredential.Id || credentials[0].RoleDefinition.Name != "Support" {
		t.Fatalf("GetAllCredential() = %d, %+v", count, credentials)
	}
	count, credentials, err = service.GetAllCredential(context.Background(), auth, 7, 9, []*protos.Criteria{{Key: "name", Logic: "contains", Value: "%"}}, &protos.Paginate{Page: 1, PageSize: 25})
	if err != nil || count != 0 || len(credentials) != 0 {
		t.Fatalf("GetAllCredential() literal wildcard = %d, %+v, %v", count, credentials, err)
	}
	count, credentials, err = service.GetAllCredential(context.Background(), auth, 7, 9, []*protos.Criteria{{Key: "name", Logic: "contains", Value: "rpx_"}}, &protos.Paginate{Page: 1, PageSize: 25})
	if err != nil || count != 0 || len(credentials) != 0 {
		t.Fatalf("GetAllCredential() secret query = %d, %+v, %v", count, credentials, err)
	}
	count, credentials, err = service.GetAllCredential(context.Background(), auth, 7, 9, []*protos.Criteria{{Key: "name", Logic: "contains", Value: "custom"}, {Key: "role", Logic: "contains", Value: "support"}}, &protos.Paginate{Page: 1, PageSize: 25})
	if err != nil || count != 1 || len(credentials) != 1 || credentials[0].Id != customCredential.Id {
		t.Fatalf("GetAllCredential() AND criteria = %d, %+v, %v", count, credentials, err)
	}
	count, credentials, err = service.GetAllCredential(context.Background(), auth, 7, 9, nil, &protos.Paginate{Page: 1, PageSize: 1})
	if err != nil || count != 2 || len(credentials) != 1 || credentials[0].Id != customCredential.Id {
		t.Fatalf("GetAllCredential() page = %d, %+v, %v", count, credentials, err)
	}
	count, credentials, err = service.GetAllCredential(context.Background(), auth, 7, 9, nil, nil)
	if err != nil || count != 2 || len(credentials) != 2 {
		t.Fatalf("GetAllCredential() unbounded = %d, %+v, %v", count, credentials, err)
	}

	blankNameCredential, err := service.CreateCredential(context.Background(), auth, " \t ", 7, 9, 117)
	if err != nil || blankNameCredential.Name != " \t " {
		t.Fatalf("CreateCredential() blank name = %+v, %v", blankNameCredential, err)
	}

	for _, test := range []struct {
		name             string
		projectID        uint64
		organizationID   uint64
		roleDefinitionID uint64
	}{
		{name: "missing project", organizationID: 9, roleDefinitionID: 117},
		{name: "missing organization", projectID: 7, roleDefinitionID: 117},
		{name: "missing role", projectID: 7, organizationID: 9},
		{name: "unknown role", projectID: 7, organizationID: 9, roleDefinitionID: 999},
		{name: "cross organization role", projectID: 7, organizationID: 9, roleDefinitionID: 119},
		{name: "wrong subject", projectID: 7, organizationID: 9, roleDefinitionID: 120},
		{name: "archived role", projectID: 7, organizationID: 9, roleDefinitionID: 121},
		{name: "cross organization project", projectID: 8, organizationID: 9, roleDefinitionID: 117},
		{name: "archived project", projectID: 9, organizationID: 9, roleDefinitionID: 117},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := int64(0)
			if err := db.Table("project_credentials").Count(&before).Error; err != nil {
				t.Fatal(err)
			}
			created, createErr := service.CreateCredential(context.Background(), auth, "rejected", test.projectID, test.organizationID, test.roleDefinitionID)
			isTenantMismatch := test.projectID != 7 || test.organizationID != 9
			var selectionError pkg_errors.CredentialSelectionNotFoundError
			var authorizationError pkg_errors.RouteAuthorizationDeniedError
			if created != nil || (isTenantMismatch && !errors.As(createErr, &authorizationError)) || (!isTenantMismatch && !errors.As(createErr, &selectionError)) {
				t.Fatalf("CreateCredential() = %+v, %v", created, createErr)
			}
			after := int64(0)
			if err := db.Table("project_credentials").Count(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Fatalf("credential count changed from %d to %d", before, after)
			}
		})
	}

	revokedID, err := service.RevokeCredential(context.Background(), auth, credential.Id, 7, 9)
	if err != nil {
		t.Fatalf("RevokeCredential() error = %v", err)
	}
	if revokedID != credential.Id {
		t.Fatalf("RevokeCredential() ID = %d", revokedID)
	}
	if secondID, secondErr := service.RevokeCredential(context.Background(), auth, credential.Id, 7, 9); secondErr != nil || secondID != credential.Id {
		t.Fatalf("RevokeCredential() retry = %d, %v", secondID, secondErr)
	}
	var archived struct {
		Status           type_enums.RecordState
		UpdatedActorType string
		UpdatedActorID   uint64
	}
	if err := db.Table("project_credentials").Where("id = ?", credential.Id).Take(&archived).Error; err != nil {
		t.Fatal(err)
	}
	if archived.Status != type_enums.RECORD_ARCHIEVE || archived.UpdatedActorType != "user" || archived.UpdatedActorID != 11 {
		t.Fatalf("archived credential = %+v", archived)
	}
	for _, hidden := range []struct {
		name           string
		credentialID   uint64
		projectID      uint64
		organizationID uint64
	}{
		{name: "unknown", credentialID: credential.Id + 999, projectID: 7, organizationID: 9},
		{name: "cross project", credentialID: credential.Id, projectID: 8, organizationID: 9},
		{name: "cross organization", credentialID: credential.Id, projectID: 7, organizationID: 10},
	} {
		t.Run(hidden.name, func(t *testing.T) {
			_, hiddenErr := service.RevokeCredential(context.Background(), auth, hidden.credentialID, hidden.projectID, hidden.organizationID)
			isTenantMismatch := hidden.projectID != 7 || hidden.organizationID != 9
			var notFound pkg_errors.ProjectAPIKeyNotFoundError
			var authorizationError pkg_errors.RouteAuthorizationDeniedError
			if (isTenantMismatch && !errors.As(hiddenErr, &authorizationError)) || (!isTenantMismatch && !errors.As(hiddenErr, &notFound)) {
				t.Fatalf("RevokeCredential() hidden error = %v", hiddenErr)
			}
		})
	}
}
