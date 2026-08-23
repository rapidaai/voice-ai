// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type assistantConfigurationService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func NewAssistantConfigurationService(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
) internal_services.AssistantConfigurationService {
	return &assistantConfigurationService{
		logger:   logger,
		postgres: postgres,
	}
}

func (s *assistantConfigurationService) Get(
	ctx context.Context,
	auth *types.Authentication,
	configurationId uint64,
	assistantId uint64,
) (*internal_assistant_entity.AssistantConfiguration, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := s.postgres.DB(ctx)

	var out *internal_assistant_entity.AssistantConfiguration
	tx := db.Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ? AND status = ?",
			configurationId,
			assistantId,
			projectContext.OrganizationID,
			projectContext.ProjectID,
			type_enums.RECORD_ACTIVE,
		).
		First(&out)
	if tx.Error != nil {
		s.logger.Benchmark("AssistantConfigurationService.Get", time.Since(start))
		return nil, tx.Error
	}

	s.logger.Benchmark("AssistantConfigurationService.Get", time.Since(start))
	return out, nil
}

func (s *assistantConfigurationService) GetAll(
	ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	configurationType string,
	provider string,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
) (int64, []*internal_assistant_entity.AssistantConfiguration, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return 0, nil, err
	}

	start := time.Now()
	db := s.postgres.DB(ctx)

	var (
		out []*internal_assistant_entity.AssistantConfiguration
		cnt int64
	)

	qry := db.Model(internal_assistant_entity.AssistantConfiguration{}).
		Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
		Where("assistant_id = ? AND organization_id = ? AND project_id = ? AND status = ?",
			assistantId,
			projectContext.OrganizationID,
			projectContext.ProjectID,
			type_enums.RECORD_ACTIVE,
		)
	if strings.TrimSpace(configurationType) != "" {
		qry = qry.Where("configuration_type = ?", strings.TrimSpace(configurationType))
	}
	if strings.TrimSpace(provider) != "" {
		qry = qry.Where("provider = ?", strings.TrimSpace(provider))
	}
	for _, ct := range criterias {
		qry = qry.Where(fmt.Sprintf("%s %s ?", ct.GetKey(), ct.GetLogic()), ct.GetValue())
	}

	page := uint32(0)
	pageSize := uint32(0)
	if paginate != nil {
		page = paginate.GetPage()
		pageSize = paginate.GetPageSize()
	}

	tx := qry.Scopes(
		gorm_models.Paginate(
			gorm_models.NewPaginated(
				page,
				pageSize,
				&cnt,
				qry,
			),
		),
	).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_date"},
		Desc:   true,
	}).Find(&out)
	if tx.Error != nil {
		s.logger.Benchmark("AssistantConfigurationService.GetAll", time.Since(start))
		return cnt, nil, tx.Error
	}

	s.logger.Benchmark("AssistantConfigurationService.GetAll", time.Since(start))
	return cnt, out, nil
}

func (s *assistantConfigurationService) Create(
	ctx context.Context,
	auth *types.Authentication,
	assistantId uint64,
	configurationType string,
	provider string,
	enabled bool,
	options []*protos.Metadata,
) (*internal_assistant_entity.AssistantConfiguration, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := s.postgres.DB(ctx)

	configurationType = strings.TrimSpace(configurationType)
	provider = strings.TrimSpace(provider)

	out := &internal_assistant_entity.AssistantConfiguration{
		AssistantId:       assistantId,
		ConfigurationType: internal_assistant_entity.AssistantConfigurationType(configurationType),
		Provider:          provider,
		Enabled:           enabled,
		Organizational: gorm_models.Organizational{
			ProjectId:      projectContext.ProjectID,
			OrganizationId: projectContext.OrganizationID,
		},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(out).Error; err != nil {
			return err
		}
		if _, err := s.createOptions(ctx, tx, auth, out.Id, options); err != nil {
			return err
		}
		return tx.WithContext(ctx).
			Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
			Where("id = ?", out.Id).
			First(&out).Error
	})
	if err != nil {
		s.logger.Benchmark("AssistantConfigurationService.Create", time.Since(start))
		return nil, err
	}

	s.logger.Benchmark("AssistantConfigurationService.Create", time.Since(start))
	return out, nil
}

func (s *assistantConfigurationService) Update(
	ctx context.Context,
	auth *types.Authentication,
	configurationId uint64,
	assistantId uint64,
	configurationType string,
	provider string,
	enabled bool,
	options []*protos.Metadata,
) (*internal_assistant_entity.AssistantConfiguration, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := s.postgres.DB(ctx)

	configurationType = strings.TrimSpace(configurationType)
	provider = strings.TrimSpace(provider)

	var out *internal_assistant_entity.AssistantConfiguration
	err = db.Transaction(func(tx *gorm.DB) error {
		patch := map[string]interface{}{
			"configuration_type": internal_assistant_entity.AssistantConfigurationType(configurationType),
			"provider":           provider,
			"enabled":            enabled,
		}
		query := tx.WithContext(ctx).
			Model(&internal_assistant_entity.AssistantConfiguration{}).
			Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ? AND status = ?",
				configurationId,
				assistantId,
				projectContext.OrganizationID,
				projectContext.ProjectID,
				type_enums.RECORD_ACTIVE,
			).
			Updates(patch)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			return errors.New("assistant configuration not found")
		}
		if err := s.archiveOptions(ctx, tx, auth, configurationId); err != nil {
			return err
		}
		if _, err := s.createOptions(ctx, tx, auth, configurationId, options); err != nil {
			return err
		}
		return tx.WithContext(ctx).
			Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
			Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ?",
				configurationId,
				assistantId,
				projectContext.OrganizationID,
				projectContext.ProjectID,
			).
			First(&out).Error
	})
	if err != nil {
		s.logger.Benchmark("AssistantConfigurationService.Update", time.Since(start))
		return nil, err
	}

	s.logger.Benchmark("AssistantConfigurationService.Update", time.Since(start))
	return out, nil
}

func (s *assistantConfigurationService) Delete(
	ctx context.Context,
	auth *types.Authentication,
	configurationId uint64,
	assistantId uint64,
) (*internal_assistant_entity.AssistantConfiguration, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	db := s.postgres.DB(ctx)

	var out *internal_assistant_entity.AssistantConfiguration
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Preload("Options", "status = ?", type_enums.RECORD_ACTIVE).
			Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ? AND status = ?",
				configurationId,
				assistantId,
				projectContext.OrganizationID,
				projectContext.ProjectID,
				type_enums.RECORD_ACTIVE,
			).
			First(&out).Error; err != nil {
			return err
		}
		query := tx.WithContext(ctx).
			Model(&internal_assistant_entity.AssistantConfiguration{}).
			Where("id = ? AND assistant_id = ? AND organization_id = ? AND project_id = ? AND status = ?",
				configurationId,
				assistantId,
				projectContext.OrganizationID,
				projectContext.ProjectID,
				type_enums.RECORD_ACTIVE,
			).
			Updates(map[string]interface{}{
				"status": type_enums.RECORD_ARCHIEVE,
			})
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			return errors.New("assistant configuration not found")
		}
		return s.archiveOptions(ctx, tx, auth, configurationId)
	})
	if err != nil {
		s.logger.Benchmark("AssistantConfigurationService.Delete", time.Since(start))
		return nil, err
	}

	s.logger.Benchmark("AssistantConfigurationService.Delete", time.Since(start))
	return out, nil
}

func (s *assistantConfigurationService) archiveOptions(
	ctx context.Context,
	tx *gorm.DB,
	auth *types.Authentication,
	configurationId uint64,
) error {
	return tx.WithContext(ctx).
		Model(&internal_assistant_entity.AssistantConfigurationOption{}).
		Where("assistant_configuration_id = ? AND status = ?", configurationId, type_enums.RECORD_ACTIVE).
		Updates(map[string]interface{}{
			"status": type_enums.RECORD_ARCHIEVE,
		}).Error
}

func (s *assistantConfigurationService) createOptions(
	ctx context.Context,
	tx *gorm.DB,
	auth *types.Authentication,
	configurationId uint64,
	options []*protos.Metadata,
) ([]*internal_assistant_entity.AssistantConfigurationOption, error) {

	if len(options) == 0 {
		return []*internal_assistant_entity.AssistantConfigurationOption{}, nil
	}
	out := make([]*internal_assistant_entity.AssistantConfigurationOption, 0, len(options))
	for _, opt := range options {
		out = append(out, &internal_assistant_entity.AssistantConfigurationOption{
			AssistantConfigurationId: configurationId,
			Metadata: gorm_models.Metadata{
				Key:   opt.GetKey(),
				Value: opt.GetValue(),
			},
			Mutable: gorm_models.Mutable{
				Status: type_enums.RECORD_ACTIVE,
			},
		})
	}

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "key"},
			{Name: "assistant_configuration_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"status",
			"updated_date",
		}),
	}).Create(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
