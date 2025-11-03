package internal_lead_service

import (
	"context"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

func NewLeadService(logger commons.Logger, postgres connectors.PostgresConnector) internal_service.LeadService {
	return &leadService{
		logger:   logger,
		postgres: postgres,
	}
}

type leadService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
}

func (ls *leadService) Create(ctx context.Context, email, companyName, expectedVolume string) (*internal_entity.Lead, error) {
	db := ls.postgres.DB(ctx)
	lead := &internal_entity.Lead{
		Email:          email,
		CompanyName:    companyName,
		ExpectedVolume: expectedVolume,
	}
	tx := db.Save(lead)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return lead, nil
	}
}
