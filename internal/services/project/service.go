package internal_project_service

import (
	"context"
	"fmt"
	"time"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/ciphers"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	gorm_models "github.com/lexatic/web-backend/pkg/models/gorm"
	"github.com/lexatic/web-backend/pkg/types"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"gorm.io/gorm/clause"
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

func NewProjectAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector) types.ClaimAuthenticator[*types.ProjectScope] {
	return &projectService{
		logger:   logger,
		postgres: postgres,
	}
}

// Claim implements types.ClaimAuthenticator.
func (p *projectService) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.ProjectScope], error) {
	start := time.Now()
	db := p.postgres.DB(ctx)
	var prjScope *types.ProjectScope
	tx := db.Table("project_credentials").Order("created_date DESC").Where("key = ?", claimToken).First(&prjScope)
	if tx.Error != nil {
		p.logger.Errorf("Authentication error, illegal key request %v", tx.Error)
		return nil, tx.Error
	}
	p.logger.Debugf("Benchmarking: projectAuthenticator.Claim time taken %v and value %+v", time.Since(start), prjScope)
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: prjScope,
	}, nil
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

func (pS *projectService) GetAll(ctx context.Context, auth types.SimplePrinciple, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.Project, error) {
	db := pS.postgres.DB(ctx)
	var projects []internal_gorm.Project
	var cnt int64
	qry := db.Model(internal_gorm.Project{}).
		Where("organization_id = ? AND status = ? ", organizationId, "active")
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
		Find(&projects)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return cnt, nil, tx.Error
	}

	return cnt, &projects, nil
}

func (pS *projectService) Get(ctx context.Context, auth types.SimplePrinciple, projectId uint64) (*internal_gorm.Project, error) {
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

func (pS *projectService) GetAllCredential(ctx context.Context, auth types.Principle, projectId, organizationId uint64, criterias []*web_api.Criteria, paginate *web_api.Paginate) (int64, *[]internal_gorm.ProjectCredential, error) {
	db := pS.postgres.DB(ctx)
	var pcs []internal_gorm.ProjectCredential
	var cnt int64
	qry := db.Model(internal_gorm.ProjectCredential{}).
		Where("project_id = ? AND organization_id = ? AND status = ? ", projectId, organizationId, "active")
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
		Find(&pcs)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return cnt, nil, tx.Error
	}

	return cnt, &pcs, nil

}
