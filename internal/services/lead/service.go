package internal_lead_service

import (
	"context"

	internal_gorm "github.com/lexatic/web-backend/internal/gorm"
	internal_services "github.com/lexatic/web-backend/internal/services"
	"github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
)

func NewLeadService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.LeadService {
	return &leadService{
		logger:   logger,
		postgres: postgres,
	}
}

type leadService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func (lS *leadService) Create(ctx context.Context, email string) (*internal_gorm.Lead, error) {
	db := lS.postgres.DB(ctx)
	org := &internal_gorm.Lead{
		Email: email,
	}
	tx := db.Save(org)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return org, nil
	}
}
