package internal_organization_service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generators "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

func NewOrganizationService(logger commons.Logger, postgres connectors.PostgresConnector, fingerprintKey ...string) internal_services.OrganizationService {
	var key string
	if len(fingerprintKey) > 0 {
		key = fingerprintKey[0]
	}
	return &organizationService{
		logger:         logger,
		postgres:       postgres,
		fingerprintKey: []byte(key),
	}
}

func NewOrganizationAuthenticator(logger commons.Logger, postgres connectors.PostgresConnector, fingerprintKey string) types.ClaimAuthenticator[*types.OrganizationScope] {
	return &organizationService{logger: logger, postgres: postgres, fingerprintKey: []byte(fingerprintKey)}
}

type organizationService struct {
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	fingerprintKey []byte
}

func (oS *organizationService) Claim(ctx context.Context, claimToken string) (*types.PlainClaimPrinciple[*types.OrganizationScope], error) {
	fingerprint, err := oS.fingerprint(claimToken)
	if err != nil {
		return nil, err
	}
	var scope types.OrganizationScope
	tx := oS.postgres.DB(ctx).
		Table("organization_credentials").
		Select("id AS credential_id, organization_id, status").
		Where("key = ? AND status = ?", fingerprint, type_enums.RECORD_ACTIVE).
		Take(&scope)
	if tx.Error != nil {
		return nil, tx.Error
	}
	scope.CurrentToken = claimToken
	if !scope.IsAuthenticated() {
		return nil, errors.New("organization credential has invalid identity")
	}
	return &types.PlainClaimPrinciple[*types.OrganizationScope]{Info: &scope}, nil
}

func (oS *organizationService) Create(ctx context.Context, auth *types.Authentication, name string, size string, industry string) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	org := &internal_entity.Organization{
		Name:     name,
		Industry: industry,
		Size:     size,
		Mutable: gorm_models.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
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

func (oS *organizationService) Update(ctx context.Context, auth *types.Authentication, organizationId uint64, name *string, industry *string, email *string) (*internal_entity.Organization, error) {
	db := oS.postgres.DB(ctx)
	updates := map[string]interface{}{
		"status":             type_enums.RECORD_ACTIVE,
		"updated_actor_type": auth.Actor().Type.String(),
		"updated_actor_id":   auth.Actor().ID,
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

func (oS *organizationService) CreateCredential(ctx context.Context, auth *types.Authentication, organizationId uint64, name string) (*internal_entity.OrganizationCredential, string, error) {
	if organizationId == 0 || organizationId > math.MaxInt64 || strings.TrimSpace(name) == "" {
		return nil, "", errors.New("organization credential requires a valid organization and name")
	}
	rawKey := types.ORG_KEY_PREFIX + ciphers.Token("organization")
	fingerprint, err := oS.fingerprint(rawKey)
	if err != nil {
		return nil, "", err
	}
	credentialID := gorm_generators.ID()
	if credentialID == 0 || credentialID > math.MaxInt64 {
		return nil, "", errors.New("organization credential generated an invalid id")
	}
	credential := &internal_entity.OrganizationCredential{
		Audited: gorm_models.Audited{
			Id: credentialID,
		},
		OrganizationId: organizationId,
		Name:           strings.TrimSpace(name),
		Key:            fingerprint,
		Mutable: gorm_models.Mutable{
			Status:           type_enums.RECORD_ACTIVE,
			CreatedActorType: auth.Actor().Type.String(),
			CreatedActorID:   auth.Actor().ID,
		},
	}
	if err := oS.postgres.DB(ctx).Create(credential).Error; err != nil {
		return nil, "", err
	}
	return credential, rawKey, nil
}

func (oS *organizationService) RotateCredential(ctx context.Context, auth *types.Authentication, organizationId, credentialId uint64) (*internal_entity.OrganizationCredential, string, error) {
	if err := validCredentialIdentity(organizationId, credentialId); err != nil {
		return nil, "", err
	}
	rawKey := types.ORG_KEY_PREFIX + ciphers.Token("organization")
	fingerprint, err := oS.fingerprint(rawKey)
	if err != nil {
		return nil, "", err
	}
	credential := &internal_entity.OrganizationCredential{}
	tx := oS.postgres.DB(ctx).Model(credential).
		Where("id = ? AND organization_id = ? AND status = ?", credentialId, organizationId, type_enums.RECORD_ACTIVE).
		Updates(map[string]interface{}{
			"key":                fingerprint,
			"updated_actor_type": auth.Actor().Type.String(),
			"updated_actor_id":   auth.Actor().ID,
			"updated_date":       time.Now(),
		})
	if tx.Error != nil {
		return nil, "", tx.Error
	}
	if tx.RowsAffected != 1 {
		return nil, "", errors.New("active organization credential not found")
	}
	if err := oS.postgres.DB(ctx).Where("id = ?", credentialId).Take(credential).Error; err != nil {
		return nil, "", err
	}
	return credential, rawKey, nil
}

func (oS *organizationService) ArchiveCredential(ctx context.Context, auth *types.Authentication, organizationId, credentialId uint64) (*internal_entity.OrganizationCredential, error) {
	if err := validCredentialIdentity(organizationId, credentialId); err != nil {
		return nil, err
	}
	now := time.Now()
	credential := &internal_entity.OrganizationCredential{}
	tx := oS.postgres.DB(ctx).Model(credential).
		Where("id = ? AND organization_id = ? AND status = ?", credentialId, organizationId, type_enums.RECORD_ACTIVE).
		Updates(map[string]interface{}{
			"status":             type_enums.RECORD_ARCHIEVE,
			"archived_date":      now,
			"updated_actor_type": auth.Actor().Type.String(),
			"updated_actor_id":   auth.Actor().ID,
			"updated_date":       now,
		})
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected != 1 {
		return nil, errors.New("active organization credential not found")
	}
	if err := oS.postgres.DB(ctx).Where("id = ?", credentialId).Take(credential).Error; err != nil {
		return nil, err
	}
	return credential, nil
}

func (oS *organizationService) fingerprint(rawKey string) (string, error) {
	if len(oS.fingerprintKey) == 0 || strings.TrimSpace(rawKey) == "" {
		return "", errors.New("organization credential fingerprint key and credential are required")
	}
	mac := hmac.New(sha256.New, oS.fingerprintKey)
	_, _ = mac.Write([]byte(rawKey))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func validCredentialIdentity(organizationId, credentialId uint64) error {
	if organizationId == 0 || organizationId > math.MaxInt64 || credentialId == 0 || credentialId > math.MaxInt64 {
		return errors.New("organization credential identity is invalid")
	}
	return nil
}
