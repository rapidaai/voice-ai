package internal_project_service

import (
	"context"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/ciphers"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	"github.com/lexatic/web-backend/pkg/types"
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

func (pS *projectService) Create(ctx context.Context, auth types.Principle, organizationId uint64, name string, description string) (*internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	project := &internal_gorm.Project{
		Name:           name,
		OrganizationId: organizationId,
		Description:    description,
		CreatedBy:      auth.GetUserInfo().Id,
	}
	tx := db.Save(project)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return project, nil
}
func (pS *projectService) Update(ctx context.Context, auth types.Principle, projectId uint64, name *string, description *string) (*internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	project := &internal_gorm.Project{
		Audited: gorm_models.Audited{
			Id: projectId,
		},
	}
	updates := map[string]interface{}{"updated_by": auth.GetUserInfo().Id}

	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	tx := db.Model(&project).Updates(updates)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return project, nil
}

func (pS *projectService) GetAll(ctx context.Context, auth types.Principle, organizationId uint64) (*[]internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	var projects []internal_gorm.Project
	tx := db.Where("organization_id = ? AND status = ? ", organizationId, "active").Find(&projects)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return nil, tx.Error
	}
	return &projects, nil
}

func (pS *projectService) Get(ctx context.Context, auth types.Principle, projectId uint64) (*internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	var project internal_gorm.Project
	tx := db.Where("id = ? AND status = ? ", projectId, "active").First(&project)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", projectId)
		return nil, tx.Error
	}
	return &project, nil
}

func (pS *projectService) Archive(ctx context.Context, auth types.Principle, projectId uint64) (*internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	ct := &internal_gorm.Project{Status: "archive", UpdatedBy: auth.GetUserInfo().Id}
	tx := db.Where("id=?", projectId).Updates(&ct)
	if tx.Error != nil {
		pS.logger.Debugf("unable to update the project %v", projectId)
		return nil, tx.Error
	}
	return ct, nil
}

func (pS *projectService) CreateCredential(ctx context.Context, auth types.Principle, name string, projectId, organizationId uint64) (*internal_gorm.ProjectCredential, error) {
	db := pS.postgres.DB(ctx)
	key := ciphers.Token("rpx_")
	prc := &internal_gorm.ProjectCredential{
		ProjectId:      projectId,
		OrganizationId: organizationId,
		Name:           name,
		Key:            key,
		Status:         "active",
		CreatedBy:      auth.GetUserInfo().Id,
	}
	tx := db.Save(prc)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return prc, nil
}

func (pS *projectService) ArchiveCredential(ctx context.Context, auth types.Principle, credentialId, projectId, organizationId uint64) (*internal_gorm.ProjectCredential, error) {
	db := pS.postgres.DB(ctx)
	ct := &internal_gorm.ProjectCredential{Status: "archive", UpdatedBy: auth.GetUserInfo().Id}
	tx := db.Where("id=? AND project_id = ? AND organization_id = ?", credentialId, projectId, organizationId).Updates(&ct)
	if tx.Error != nil {
		pS.logger.Debugf("unable to update project credentials %v", credentialId)
		return nil, tx.Error
	}
	return ct, nil
}

func (pS *projectService) GetAllCredential(ctx context.Context, auth types.Principle, projectId, organizationId uint64) (*[]internal_gorm.ProjectCredential, error) {
	db := pS.postgres.DB(ctx)
	var pcs []internal_gorm.ProjectCredential
	tx := db.Where("project_id = ? AND organization_id = ? AND status = ? ", projectId, organizationId, "active").Find(&pcs)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return nil, tx.Error
	}
	return &pcs, nil

}
