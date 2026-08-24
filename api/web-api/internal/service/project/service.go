package internal_project_service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generators "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
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
	var prjScope types.ProjectScope
	tx := db.Table("project_credentials").
		Select("id AS credential_id, project_id, organization_id, status").
		Order("created_date DESC").
		Where("key = ?", claimToken).
		Take(&prjScope)
	if tx.Error != nil {
		p.logger.Errorf("Authentication error, illegal key request %v", tx.Error)
		return nil, tx.Error
	}
	p.logger.Debugf("Benchmarking: projectAuthenticator.Claim time taken %v and value %+v", time.Since(start), prjScope)
	prjScope.CurrentToken = claimToken
	return &types.PlainClaimPrinciple[*types.ProjectScope]{
		Info: &prjScope,
	}, nil
}

func (pS *projectService) Create(ctx context.Context, auth *types.Authentication, organizationId uint64, name string, description string) (*internal_entity.Project, error) {

	db := pS.postgres.DB(ctx)
	project := &internal_entity.Project{
		Name:           name,
		OrganizationId: organizationId,
		Description:    description,
		Mutable: gorm_models.Mutable{
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
		},
	}
	tx := db.Save(project)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return project, nil
}
func (pS *projectService) Update(ctx context.Context, auth *types.Authentication, projectId uint64, name *string, description *string) (*internal_entity.Project, error) {

	db := pS.postgres.DB(ctx)
	project := &internal_entity.Project{
		Audited: gorm_models.Audited{
			Id: projectId,
		},
	}
	updates := map[string]interface{}{
		"updated_actor_type": auth.Actor().Type.String(),
		"updated_actor_id":   auth.Actor().ID,
		"updated_date":       time.Now(),
	}

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

func (pS *projectService) GetAll(ctx context.Context, auth *types.Authentication, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.Project, error) {
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

func (pS *projectService) Get(ctx context.Context, auth *types.Authentication, projectId uint64) (*internal_entity.Project, error) {
	db := pS.postgres.DB(ctx)
	var project internal_entity.Project
	tx := db.Where("id = ? AND status = ? ", projectId, type_enums.RECORD_ACTIVE).First(&project)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", projectId)
		return nil, tx.Error
	}
	return &project, nil
}

func (pS *projectService) GetAllByOrganization(ctx context.Context, auth *types.Authentication, organizationId uint64, projectIds []uint64) ([]*internal_entity.Project, error) {
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

func (pS *projectService) Archive(ctx context.Context, auth *types.Authentication, projectId uint64) (*internal_entity.Project, error) {

	db := pS.postgres.DB(ctx)
	ct := &internal_entity.Project{}
	tx := db.Model(ct).Where("id = ?", projectId).Updates(map[string]interface{}{
		"status":             type_enums.RECORD_ARCHIEVE,
		"updated_actor_type": auth.Actor().Type.String(),
		"updated_actor_id":   auth.Actor().ID,
		"updated_date":       time.Now(),
	})
	if tx.Error != nil {
		pS.logger.Debugf("unable to update the project %v", projectId)
		return nil, tx.Error
	}
	return ct, nil
}

func (pS *projectService) CreateCredential(ctx context.Context, auth *types.Authentication, name string, projectId, organizationId uint64) (*internal_entity.ProjectCredential, error) {
	if projectId == 0 || projectId > math.MaxInt64 || organizationId == 0 || organizationId > math.MaxInt64 || strings.TrimSpace(name) == "" {
		return nil, errors.New("project credential requires a valid project, organization, and name")
	}
	credentialID := gorm_generators.ID()
	if credentialID == 0 || credentialID > math.MaxInt64 {
		return nil, errors.New("project credential generated an invalid id")
	}

	db := pS.postgres.DB(ctx)
	key := ciphers.Token("rpx_")
	prc := &internal_entity.ProjectCredential{
		Audited: gorm_models.Audited{Id: credentialID},
		Organizational: gorm_models.Organizational{
			ProjectId:      projectId,
			OrganizationId: organizationId,
		},
		Name: strings.TrimSpace(name),
		Key:  key,
		Mutable: gorm_models.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
		},
	}
	tx := db.Create(prc)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return prc, nil
}

func (pS *projectService) ArchiveCredential(ctx context.Context, auth *types.Authentication, credentialId, projectId, organizationId uint64) (*internal_entity.ProjectCredential, error) {
	if credentialId == 0 || credentialId > math.MaxInt64 || projectId == 0 || projectId > math.MaxInt64 || organizationId == 0 || organizationId > math.MaxInt64 {
		return nil, errors.New("project credential identity is invalid")
	}

	db := pS.postgres.DB(ctx)
	ct := &internal_entity.ProjectCredential{}
	tx := db.Model(ct).
		Where("id = ? AND project_id = ? AND organization_id = ? AND status = ?", credentialId, projectId, organizationId, type_enums.RECORD_ACTIVE).
		Updates(map[string]interface{}{
			"status":             type_enums.RECORD_ARCHIEVE,
			"updated_actor_type": auth.Actor().Type.String(),
			"updated_actor_id":   auth.Actor().ID,
			"updated_date":       time.Now(),
		})
	if tx.Error != nil {
		pS.logger.Debugf("unable to update project credentials %v", credentialId)
		return nil, tx.Error
	}
	if tx.RowsAffected != 1 {
		return nil, errors.New("active project credential not found")
	}
	if err := db.Where("id = ?", credentialId).Take(ct).Error; err != nil {
		return nil, err
	}
	return ct, nil
}

func (pS *projectService) GetAllCredential(ctx context.Context, auth *types.Authentication, projectId, organizationId uint64, criteria []*web_api.Criteria, paginate *web_api.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
	db := pS.postgres.DB(ctx)
	var pcs []*internal_entity.ProjectCredential
	var cnt int64
	qry := db.
		Model(internal_entity.ProjectCredential{}).
		Where("project_id = ? AND organization_id = ? AND status = ? ", projectId, organizationId, type_enums.RECORD_ACTIVE)
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
		Find(&pcs)
	if tx.Error != nil {
		pS.logger.Debugf("unable to find any project %v", organizationId)
		return cnt, nil, tx.Error
	}

	return cnt, pcs, nil
}
