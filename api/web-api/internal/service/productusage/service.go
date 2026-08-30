package internal_productusage_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/connectors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

const MaxBatchSize = 100

var (
	ErrInvalidRequest  = errors.New("invalid product usage request")
	ErrProjectMismatch = errors.New("project does not belong to the authenticated organization")
	ErrUsageConflict   = errors.New("product usage id conflicts with an existing record")
)

type Result struct {
	CreatedCount   uint32
	DuplicateCount uint32
}

type Service interface {
	CreateProductUsages(context.Context, *types.Authentication, []*protos.ProductUsage) (Result, error)
}

type productUsageService struct {
	postgres connectors.PostgresConnector
}

func NewProductUsageService(postgres connectors.PostgresConnector) Service {
	return &productUsageService{postgres: postgres}
}

func (service *productUsageService) CreateProductUsages(ctx context.Context, auth *types.Authentication, inputs []*protos.ProductUsage) (Result, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrProjectMismatch, err)
	}
	if len(inputs) == 0 || len(inputs) > MaxBatchSize {
		return Result{}, fmt.Errorf("%w: usages must contain between 1 and %d records", ErrInvalidRequest, MaxBatchSize)
	}

	usages := make([]internal_entity.ProductUsage, len(inputs))
	for index, input := range inputs {
		usage, validationErr := buildProductUsage(auth, projectContext, input)
		if validationErr != nil {
			return Result{}, fmt.Errorf("%w: usages[%d]: %v", ErrInvalidRequest, index, validationErr)
		}
		usages[index] = usage
	}

	var result Result
	err = service.postgres.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var projectCount int64
		if countErr := tx.Model(&internal_entity.Project{}).
			Where("id = ? AND organization_id = ? AND status = ?", projectContext.ProjectID, projectContext.OrganizationID, type_enums.RECORD_ACTIVE).
			Count(&projectCount).Error; countErr != nil {
			return countErr
		}
		if projectCount != 1 {
			return ErrProjectMismatch
		}

		for index := range usages {
			usage := &usages[index]
			insert := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "organization_id"}, {Name: "project_id"}, {Name: "usage_id"}},
				DoNothing: true,
			}).Create(usage)
			if insert.Error != nil {
				return insert.Error
			}
			if insert.RowsAffected == 1 {
				result.CreatedCount++
				continue
			}

			var existing internal_entity.ProductUsage
			if findErr := tx.Where(
				"organization_id = ? AND project_id = ? AND usage_id = ?",
				projectContext.OrganizationID,
				projectContext.ProjectID,
				usage.UsageID,
			).Take(&existing).Error; findErr != nil {
				return findErr
			}
			if !sameProductUsage(existing, *usage) {
				return fmt.Errorf("%w: %s", ErrUsageConflict, usage.UsageID)
			}
			result.DuplicateCount++
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func buildProductUsage(auth *types.Authentication, projectContext types.ProjectContext, input *protos.ProductUsage) (internal_entity.ProductUsage, error) {
	if input == nil {
		return internal_entity.ProductUsage{}, errors.New("usage is required")
	}
	usageID, err := uuid.Parse(input.GetUsageId())
	if err != nil {
		return internal_entity.ProductUsage{}, errors.New("usageId must be a UUID")
	}
	if input.GetUsages() <= 0 {
		return internal_entity.ProductUsage{}, errors.New("usages must be positive")
	}
	if err = types.ValidateProductUsage(input.GetUsageType(), input.GetUnit()); err != nil {
		return internal_entity.ProductUsage{}, err
	}
	if input.GetOccurredAt() == nil {
		return internal_entity.ProductUsage{}, errors.New("occurredAt is required")
	}
	if err = input.GetOccurredAt().CheckValid(); err != nil {
		return internal_entity.ProductUsage{}, fmt.Errorf("occurredAt is invalid: %w", err)
	}

	actor := auth.Actor()
	return internal_entity.ProductUsage{
		Mutable: gorm_model.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: actor.Type.String(),
			CreatedActorID:   actor.ID,
		},
		Organizational: gorm_model.Organizational{
			OrganizationId: projectContext.OrganizationID,
			ProjectId:      projectContext.ProjectID,
		},
		UsageID:    usageID.String(),
		UsageType:  input.GetUsageType(),
		Usages:     input.GetUsages(),
		Unit:       input.GetUnit(),
		OccurredAt: input.GetOccurredAt().AsTime().UTC().Truncate(time.Microsecond),
	}, nil
}

func sameProductUsage(left, right internal_entity.ProductUsage) bool {
	return left.UsageType == right.UsageType &&
		left.Usages == right.Usages &&
		left.Unit == right.Unit &&
		left.OccurredAt.UTC().Equal(right.OccurredAt.UTC())
}
