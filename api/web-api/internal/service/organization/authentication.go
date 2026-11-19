// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_organization_service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

// PrincipalAuthenticationService owns canonical Organization credential reads.
type PrincipalAuthenticationService interface {
	resolver.OrganizationAuthenticationService
}

// NewPrincipalAuthenticationService constructs the canonical Organization principal source.
func NewPrincipalAuthenticationService(logger commons.Logger, postgres connectors.PostgresConnector, fingerprintKey []byte) PrincipalAuthenticationService {
	return &organizationService{logger: logger, postgres: postgres, fingerprintKey: append([]byte(nil), fingerprintKey...)}
}

// ResolveOrganizationCredential resolves one exact active Organization credential.
func (service *organizationService) ResolveOrganizationCredential(ctx context.Context, token string) (authentication.Authentication, error) {
	fingerprint, err := service.fingerprint(token)
	if err != nil {
		return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	return service.verifyOrganizationAccess(ctx, fingerprint)
}

func (service *organizationService) verifyOrganizationAccess(ctx context.Context, fingerprint string) (authentication.Authentication, error) {
	var credential struct {
		CredentialID     uint64
		OrganizationID   uint64
		OrganizationName string
	}
	query := service.postgres.DB(ctx).Table("organization_credentials AS credential").
		Select("credential.id AS credential_id, credential.organization_id, organization.name AS organization_name").
		Joins("JOIN organizations AS organization ON organization.id = credential.organization_id AND organization.status = ?", type_enums.RECORD_ACTIVE).
		Where("credential.status = ?", type_enums.RECORD_ACTIVE)
	query = query.Where("credential.key = ?", fingerprint)
	err := query.Take(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return authentication.Authentication{}, pkg_errors.AuthenticationCredentialInvalidError{}
	}
	if err != nil {
		return authentication.Authentication{}, err
	}
	return authentication.NewDirectAuthentication(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: credential.CredentialID},
		authentication.Tenant{OrganizationID: credential.OrganizationID},
		authentication.Proof{
			Method:    authentication.AuthenticationOrganizationCredential,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: strconv.FormatUint(credential.CredentialID, 10)},
		},
		nil,
		[]authentication.Organization{{ID: credential.OrganizationID, Name: strings.TrimSpace(credential.OrganizationName)}},
	)
}
