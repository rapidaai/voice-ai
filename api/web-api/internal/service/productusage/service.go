package internal_productusage_service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	"github.com/rapidaai/pkg/connectors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidRequest  = errors.New("invalid product usage request")
	ErrProjectMismatch = errors.New("project does not belong to the authenticated organization")
)

type Service interface {
	CreateProductUsage(context.Context, *types.Authentication, *protos.CreateProductUsageRequest) (*internal_entity.ProductUsage, error)
	GetProductUsages(context.Context, *types.Authentication, string, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProductUsage, error)
	GetOrganizationUsages(context.Context, *types.Authentication, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProductUsage, error)
}

type productUsageService struct {
	postgres connectors.PostgresConnector
}

func NewProductUsageService(postgres connectors.PostgresConnector) Service {
	return &productUsageService{postgres: postgres}
}

func (service *productUsageService) CreateProductUsage(ctx context.Context, auth *types.Authentication, input *protos.CreateProductUsageRequest) (*internal_entity.ProductUsage, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProjectMismatch, err)
	}

	usage, err := buildProductUsage(auth, projectContext, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

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
		return tx.Create(&usage).Error
	})
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

func (service *productUsageService) GetProductUsages(ctx context.Context, auth *types.Authentication, usageType string, criteria []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.ProductUsage, error) {
	projectContext, err := auth.ProjectContext()
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrProjectMismatch, err)
	}
	if _, ok := types.ProductUsageUnitFor(usageType); !ok {
		return 0, nil, fmt.Errorf("%w: usageType is required and must be supported", ErrInvalidRequest)
	}

	query := service.postgres.DB(ctx).
		Model(&internal_entity.ProductUsage{}).
		Where(
			"organization_id = ? AND project_id = ? AND usage_type = ? AND status = ?",
			projectContext.OrganizationID,
			projectContext.ProjectID,
			usageType,
			type_enums.RECORD_ACTIVE,
		)
	for index, criterion := range criteria {
		if criterion == nil || strings.TrimSpace(criterion.GetKey()) == "" || strings.TrimSpace(criterion.GetValue()) == "" {
			continue
		}

		switch strings.TrimSpace(criterion.GetKey()) {
		case "id":
			if logic := strings.TrimSpace(criterion.GetLogic()); logic != "" && logic != "=" {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: id only supports equality", ErrInvalidRequest, index)
			}
			value, parseErr := strconv.ParseUint(criterion.GetValue(), 10, 64)
			if parseErr != nil || value == 0 {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: id must be a positive integer", ErrInvalidRequest, index)
			}
			query = query.Where("id = ?", value)
		case "usages":
			value, parseErr := strconv.ParseInt(criterion.GetValue(), 10, 64)
			if parseErr != nil {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: usages must be an integer", ErrInvalidRequest, index)
			}
			switch strings.TrimSpace(criterion.GetLogic()) {
			case ">":
				query = query.Where("usages > ?", value)
			case ">=":
				query = query.Where("usages >= ?", value)
			case "<":
				query = query.Where("usages < ?", value)
			case "<=":
				query = query.Where("usages <= ?", value)
			case "", "=":
				query = query.Where("usages = ?", value)
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: usages has an unsupported comparison", ErrInvalidRequest, index)
			}
		case "unit":
			switch strings.TrimSpace(criterion.GetLogic()) {
			case "contains":
				query = query.Where("unit LIKE ?", "%"+criterion.GetValue()+"%")
			case "", "=":
				query = query.Where("unit = ?", criterion.GetValue())
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: unit has an unsupported comparison", ErrInvalidRequest, index)
			}
		case "occurredAt":
			value, parseErr := time.Parse(time.RFC3339Nano, criterion.GetValue())
			if parseErr != nil {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: occurredAt must be RFC3339", ErrInvalidRequest, index)
			}
			switch strings.TrimSpace(criterion.GetLogic()) {
			case ">":
				query = query.Where("occurred_at > ?", value.UTC())
			case ">=":
				query = query.Where("occurred_at >= ?", value.UTC())
			case "<":
				query = query.Where("occurred_at < ?", value.UTC())
			case "<=":
				query = query.Where("occurred_at <= ?", value.UTC())
			case "", "=":
				query = query.Where("occurred_at = ?", value.UTC())
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: occurredAt has an unsupported comparison", ErrInvalidRequest, index)
			}
		default:
			return 0, nil, fmt.Errorf("%w: criterias[%d] has unsupported key %q", ErrInvalidRequest, index, criterion.GetKey())
		}
	}

	var count int64
	usages := make([]*internal_entity.ProductUsage, 0)
	result := query.
		Scopes(gorm_model.Paginate(gorm_model.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&count,
			query,
		))).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "occurred_at"}, Desc: true}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true}).
		Find(&usages)
	if result.Error != nil {
		return count, nil, result.Error
	}
	return count, usages, nil
}

func (service *productUsageService) GetOrganizationUsages(ctx context.Context, auth *types.Authentication, criteria []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.ProductUsage, error) {
	organizationContext, err := auth.OrganizationContext()
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}

	query := service.postgres.DB(ctx).
		Model(&internal_entity.ProductUsage{}).
		Where("organization_id = ? AND status = ?", organizationContext.OrganizationID, type_enums.RECORD_ACTIVE)
	for index, criterion := range criteria {
		if criterion == nil || strings.TrimSpace(criterion.GetKey()) == "" || strings.TrimSpace(criterion.GetValue()) == "" {
			continue
		}

		switch strings.TrimSpace(criterion.GetKey()) {
		case "id", "projectId":
			if logic := strings.TrimSpace(criterion.GetLogic()); logic != "" && logic != "=" {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: %s only supports equality", ErrInvalidRequest, index, criterion.GetKey())
			}
			value, parseErr := strconv.ParseUint(criterion.GetValue(), 10, 64)
			if parseErr != nil || value == 0 {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: %s must be a positive integer", ErrInvalidRequest, index, criterion.GetKey())
			}
			if criterion.GetKey() == "id" {
				query = query.Where("id = ?", value)
			} else {
				query = query.Where("project_id = ?", value)
			}
		case "usageType", "unit":
			column := "usage_type"
			if criterion.GetKey() == "unit" {
				column = "unit"
			}
			switch strings.TrimSpace(criterion.GetLogic()) {
			case "contains":
				query = query.Where(column+" LIKE ?", "%"+criterion.GetValue()+"%")
			case "", "=":
				query = query.Where(column+" = ?", criterion.GetValue())
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: %s has an unsupported comparison", ErrInvalidRequest, index, criterion.GetKey())
			}
		case "usages":
			value, parseErr := strconv.ParseInt(criterion.GetValue(), 10, 64)
			if parseErr != nil {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: usages must be an integer", ErrInvalidRequest, index)
			}
			switch strings.TrimSpace(criterion.GetLogic()) {
			case ">":
				query = query.Where("usages > ?", value)
			case ">=":
				query = query.Where("usages >= ?", value)
			case "<":
				query = query.Where("usages < ?", value)
			case "<=":
				query = query.Where("usages <= ?", value)
			case "", "=":
				query = query.Where("usages = ?", value)
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: usages has an unsupported comparison", ErrInvalidRequest, index)
			}
		case "occurredAt":
			value, parseErr := time.Parse(time.RFC3339Nano, criterion.GetValue())
			if parseErr != nil {
				return 0, nil, fmt.Errorf("%w: criterias[%d]: occurredAt must be RFC3339", ErrInvalidRequest, index)
			}
			switch strings.TrimSpace(criterion.GetLogic()) {
			case ">":
				query = query.Where("occurred_at > ?", value.UTC())
			case ">=":
				query = query.Where("occurred_at >= ?", value.UTC())
			case "<":
				query = query.Where("occurred_at < ?", value.UTC())
			case "<=":
				query = query.Where("occurred_at <= ?", value.UTC())
			case "", "=":
				query = query.Where("occurred_at = ?", value.UTC())
			default:
				return 0, nil, fmt.Errorf("%w: criterias[%d]: occurredAt has an unsupported comparison", ErrInvalidRequest, index)
			}
		default:
			return 0, nil, fmt.Errorf("%w: criterias[%d] has unsupported key %q", ErrInvalidRequest, index, criterion.GetKey())
		}
	}

	var count int64
	usages := make([]*internal_entity.ProductUsage, 0)
	result := query.
		Scopes(gorm_model.Paginate(gorm_model.NewPaginated(
			paginate.GetPage(),
			paginate.GetPageSize(),
			&count,
			query,
		))).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "occurred_at"}, Desc: true}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true}).
		Find(&usages)
	if result.Error != nil {
		return count, nil, result.Error
	}
	return count, usages, nil
}

func buildProductUsage(auth *types.Authentication, projectContext types.ProjectContext, input *protos.CreateProductUsageRequest) (internal_entity.ProductUsage, error) {
	if input == nil {
		return internal_entity.ProductUsage{}, errors.New("usage is required")
	}
	if input.GetUsages() <= 0 {
		return internal_entity.ProductUsage{}, errors.New("usages must be positive")
	}
	if err := types.ValidateProductUsage(input.GetUsageType(), input.GetUnit()); err != nil {
		return internal_entity.ProductUsage{}, err
	}
	if input.GetOccurredAt() == nil {
		return internal_entity.ProductUsage{}, errors.New("occurredAt is required")
	}
	if err := input.GetOccurredAt().CheckValid(); err != nil {
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
		UsageType:  input.GetUsageType(),
		Usages:     input.GetUsages(),
		Unit:       input.GetUnit(),
		OccurredAt: input.GetOccurredAt().AsTime().UTC().Truncate(time.Microsecond),
	}, nil
}
