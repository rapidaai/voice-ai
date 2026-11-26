package internal_project_service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	internal_accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generators "github.com/rapidaai/pkg/models/gorm/generators"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	web_api "github.com/rapidaai/protos"
)

type projectService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewProjectService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.ProjectService {
	return &projectService{
		logger:   logger,
		postgres: postgres,
	}
}

func (pS *projectService) Create(ctx context.Context, auth authentication.Principal, organizationId uint64, name string, description string) (*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || organizationId != tenant.OrganizationID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := pS.postgres.DB(ctx)
	project := &internal_entity.Project{
		Name:           name,
		OrganizationId: organizationId,
		Description:    description,
		Mutable: gorm_models.Mutable{
			CreatedActorType: string(caller.Type),
			CreatedActorID:   caller.ID,
		},
	}
	tx := db.Save(project)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return project, nil
}
func (pS *projectService) Update(ctx context.Context, auth authentication.Principal, projectId uint64, name *string, description *string) (*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 || projectId != tenant.ProjectID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := pS.postgres.DB(ctx)
	project := &internal_entity.Project{
		Audited: gorm_models.Audited{
			Id: projectId,
		},
	}
	updates := map[string]interface{}{
		"updated_actor_type": string(caller.Type),
		"updated_actor_id":   caller.ID,
		"updated_date":       time.Now(),
	}

	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	tx := db.Model(&project).Where("organization_id = ? AND status = ?", tenant.OrganizationID, type_enums.RECORD_ACTIVE).Updates(updates)
	if err := tx.Error; err != nil {
		return nil, err
	}
	if tx.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return project, nil
}

func (pS *projectService) GetAll(ctx context.Context, auth authentication.Principal, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || organizationId != tenant.OrganizationID {
		return 0, nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := pS.postgres.DB(ctx)
	var projects []*internal_entity.Project
	var cnt int64
	qry := db.Model(internal_entity.Project{}).
		Where("organization_id = ? AND status = ? ", organizationId, type_enums.RECORD_ACTIVE)
	for _, ct := range criteria {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					paginate.GetPage(),
					paginate.GetPageSize(),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&projects)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return cnt, nil, tx.Error
	}

	return cnt, projects, nil
}

func (pS *projectService) Get(ctx context.Context, auth authentication.Principal, projectId uint64) (*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 || projectId != tenant.ProjectID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := pS.postgres.DB(ctx)
	var project internal_entity.Project
	tx := db.Where("id = ? AND organization_id = ? AND status = ? ", projectId, tenant.OrganizationID, type_enums.RECORD_ACTIVE).First(&project)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", projectId)
		return nil, tx.Error
	}
	return &project, nil
}

func (pS *projectService) GetAllByOrganization(ctx context.Context, auth authentication.Principal, organizationId uint64, projectIds []uint64) ([]*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || organizationId != tenant.OrganizationID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := pS.postgres.DB(ctx)
	uniqueProjectIds := utils.Unique(projectIds)

	var projects []*internal_entity.Project
	tx := db.
		Where("organization_id = ? AND id IN ? AND status = ?", organizationId, uniqueProjectIds, type_enums.RECORD_ACTIVE).
		Find(&projects)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find projects %v for organization %v", projectIds, organizationId)
		return nil, tx.Error
	}
	if len(projects) != len(uniqueProjectIds) {
		return nil, fmt.Errorf("one or more requested projects are missing, archived, or outside the organization")
	}

	return projects, nil
}

func (pS *projectService) Archive(ctx context.Context, auth authentication.Principal, projectId uint64) (*internal_entity.Project, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || tenant.ProjectID == 0 || projectId != tenant.ProjectID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := pS.postgres.DB(ctx)
	ct := &internal_entity.Project{}
	tx := db.Model(ct).Where("id = ? AND organization_id = ? AND status = ?", projectId, tenant.OrganizationID, type_enums.RECORD_ACTIVE).Updates(map[string]interface{}{
		"status":             type_enums.RECORD_ARCHIEVE,
		"updated_actor_type": string(caller.Type),
		"updated_actor_id":   caller.ID,
		"updated_date":       time.Now(),
	})
	if tx.Error != nil {
		pS.logger.Debugf("unable to update the project %v", projectId)
		return nil, tx.Error
	}
	if tx.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return ct, nil
}

func (pS *projectService) CreateCredential(ctx context.Context, auth authentication.Principal, name string, projectId, organizationId, roleDefinitionId uint64) (*internal_entity.ProjectCredential, error) {
	tenant := auth.Tenant()
	if tenant.ProjectID == 0 || projectId != tenant.ProjectID || organizationId != tenant.OrganizationID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	credentialID := gorm_generators.ID()
	if credentialID == 0 || credentialID > math.MaxInt64 {
		return nil, errors.New("project credential generated an invalid id")
	}

	credential := &internal_entity.ProjectCredential{
		Audited: gorm_models.Audited{Id: credentialID},
		Organizational: gorm_models.Organizational{
			ProjectId:      projectId,
			OrganizationId: organizationId,
		},
		Name:             name,
		Key:              ciphers.Token("rpx_"),
		RoleDefinitionID: roleDefinitionId,
		RoleSubjectType:  internal_accessmanagement.SubjectProjectToken,
		Mutable: gorm_models.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: string(caller.Type),
			CreatedActorID:   caller.ID,
		},
	}
	role := internal_entity.RoleDefinition{}
	err := pS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var project struct{ Id uint64 }
		projectErr := tx.Table("projects").Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND organization_id = ? AND status = ?", projectId, organizationId, type_enums.RECORD_ACTIVE).
			Take(&project).Error
		if errors.Is(projectErr, gorm.ErrRecordNotFound) {
			return pkg_errors.CredentialSelectionNotFoundError{}
		}
		if projectErr != nil {
			return projectErr
		}

		roleErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ? AND scope_type = ? AND subject_type = ?", roleDefinitionId, type_enums.RECORD_ACTIVE, internal_accessmanagement.ScopeProject, internal_accessmanagement.SubjectProjectToken).
			Where("(definition_type = ? AND organization_id IS NULL) OR (definition_type = ? AND organization_id = ?)", internal_accessmanagement.DefinitionBuiltIn, internal_accessmanagement.DefinitionCustom, organizationId).
			Take(&role).Error
		if errors.Is(roleErr, gorm.ErrRecordNotFound) {
			return pkg_errors.CredentialSelectionNotFoundError{}
		}
		if roleErr != nil {
			return roleErr
		}
		return tx.Create(credential).Error
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	credential.RoleDefinition = &role
	return credential, nil
}

func (pS *projectService) RevokeCredential(ctx context.Context, auth authentication.Principal, credentialId, projectId, organizationId uint64) (uint64, error) {
	tenant := auth.Tenant()
	if tenant.ProjectID == 0 || projectId != tenant.ProjectID || organizationId != tenant.OrganizationID {
		return 0, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	err := pS.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		credential := &internal_entity.ProjectCredential{}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND project_id = ? AND organization_id = ? AND status IN ?", credentialId, projectId, organizationId, []type_enums.RecordState{type_enums.RECORD_ACTIVE, type_enums.RECORD_ARCHIEVE}).
			Take(credential).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkg_errors.ProjectAPIKeyNotFoundError{}
		}
		if err != nil {
			return err
		}
		if credential.Status == type_enums.RECORD_ARCHIEVE {
			return nil
		}
		result := tx.Model(&internal_entity.ProjectCredential{}).
			Where("id = ? AND status = ?", credentialId, type_enums.RECORD_ACTIVE).
			Updates(map[string]interface{}{
				"status":             type_enums.RECORD_ARCHIEVE,
				"updated_actor_type": string(caller.Type),
				"updated_actor_id":   caller.ID,
				"updated_date":       time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return pkg_errors.ProjectAPIKeyNotFoundError{}
		}
		return nil
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, ctxErr
		}
		return 0, err
	}
	return credentialId, nil
}

func (pS *projectService) GetAllCredential(ctx context.Context, auth authentication.Principal, projectId, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
	tenant := auth.Tenant()
	if tenant.ProjectID == 0 || projectId != tenant.ProjectID || organizationId != tenant.OrganizationID {
		return 0, nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	db := pS.postgres.DB(ctx)
	var credentials []*internal_entity.ProjectCredential
	var cnt int64
	qry := db.
		Table("project_credentials").
		Joins("JOIN role_definitions ON role_definitions.id = project_credentials.role_definition_id").
		Where("project_credentials.project_id = ? AND project_credentials.organization_id = ? AND project_credentials.status = ?", projectId, organizationId, type_enums.RECORD_ACTIVE)
	for _, criteria := range criterias {
		pattern := strings.ReplaceAll(criteria.GetValue(), `\`, `\\`)
		pattern = strings.ReplaceAll(pattern, `%`, `\%`)
		pattern = strings.ReplaceAll(pattern, `_`, `\_`)
		pattern = "%" + pattern + "%"
		switch criteria.GetKey() {
		case "name":
			qry = qry.Where(`LOWER(project_credentials.name) LIKE LOWER(?) ESCAPE '\'`, pattern)
		case "role":
			qry = qry.Where(`LOWER(role_definitions.name) LIKE LOWER(?) ESCAPE '\'`, pattern)
		}
	}
	page, pageSize := uint32(0), uint32(0)
	if paginate != nil {
		page, pageSize = paginate.GetPage(), paginate.GetPageSize()
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					page,
					pageSize,
					&cnt,
					qry))).
		Select("project_credentials.*").
		Preload("RoleDefinition").
		Order("project_credentials.created_date DESC, project_credentials.id DESC").
		Find(&credentials)
	if tx.Error != nil {
		pS.logger.Debugf("unable to list project credentials for project %v", projectId)
		return cnt, nil, tx.Error
	}

	return cnt, credentials, nil
}
