package internal_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
)

type LeadService interface {
	Create(ctx context.Context, email, company, expectedVolume string) (*internal_entity.Lead, error)
}
