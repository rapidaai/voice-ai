package internal_organization_service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func NewOrganizationService(logger commons.Logger, postgres connectors.PostgresConnector) internal_services.OrganizationService {
	return &organizationService{
		logger:   logger,
		postgres: postgres,
	}
}

type organizationService struct {
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	fingerprintKey []byte
}

func (oS *organizationService) Create(ctx context.Context, auth authentication.Principal, name string, size string, industry string) (*internal_entity.Organization, error) {
	caller := auth.Caller()
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Name:     name,
		Industry: industry,
		Size:     size,
		Mutable: gorm_models.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: string(caller.Type),
			CreatedActorID:   caller.ID,
		},
	}
	tx := db.Save(org)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return org, nil
	}
}

func (oS *organizationService) Get(ctx context.Context, organizationId uint64) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	var ct internal_entity.Organization
	tx := db.Last(&ct, "id = ?", organizationId)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &ct, nil
}

func (oS *organizationService) Update(ctx context.Context, auth authentication.Principal, organizationId uint64, name *string, industry *string, email *string) (*internal_entity.Organization, error) {
	tenant := auth.Tenant()
	if tenant.OrganizationID == 0 || organizationId != tenant.OrganizationID {
		return nil, pkg_errors.RouteAuthorizationDeniedError{}
	}
	caller := auth.Caller()
	db := oS.postgres.DB(ctx)
	updates := map[string]interface{}{
		"status":             type_enums.RECORD_ACTIVE,
		"updated_actor_type": string(caller.Type),
		"updated_actor_id":   caller.ID,
		"updated_date":       time.Now(),
	}

	if name != nil {
		updates["name"] = *name
	}
	if industry != nil {
		updates["industry"] = *industry
	}
	if email != nil {
		updates["contact"] = *email
	}
	org := &internal_entity.Organization{}
	tx := db.Model(org).Where("id = ? ", organizationId).Updates(updates)
	if err := tx.Error; err != nil {
		return nil, err
	} else {
		return org, nil
	}
}

func (oS *organizationService) fingerprint(rawKey string) (string, error) {
	if len(oS.fingerprintKey) == 0 || strings.TrimSpace(rawKey) == "" {
		return "", errors.New("organization credential fingerprint key and credential are required")
	}
	mac := hmac.New(sha256.New, oS.fingerprintKey)
	_, _ = mac.Write([]byte(rawKey))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
