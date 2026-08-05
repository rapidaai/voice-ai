// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package middleware

import (
	"context"
	"strconv"
	"strings"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

type middlewareOption struct {
	ctx                    context.Context
	logger                 commons.Logger
	postgres               connectors.PostgresConnector
	assistantService       internal_services.AssistantService
	vaultClient            web_client.VaultClient
	applySIPConfigDefaults func(*sip_infra.Config)
}

func WithContext(ctx context.Context) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.logger = logger
	}
}

func WithPostgres(postgres connectors.PostgresConnector) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.postgres = postgres
	}
}

func WithAssistantService(assistantService internal_services.AssistantService) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.assistantService = assistantService
	}
}

func WithVaultClient(vaultClient web_client.VaultClient) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.vaultClient = vaultClient
	}
}

func WithApplySIPConfigDefaults(applySIPConfigDefaults func(*sip_infra.Config)) func(*middlewareOption) {
	return func(m *middlewareOption) {
		m.applySIPConfigDefaults = applySIPConfigDefaults
	}
}

func NewRouteMiddleware(options ...func(*middlewareOption)) sip_infra.Middleware {
	m := &middlewareOption{ctx: context.Background()}
	for _, option := range options {
		if validator.NonNil(option) {
			option(m)
		}
	}
	return func(ctx *sip_infra.SIPRequestContext) error {
		user := ""
		for _, uri := range []string{ctx.ToURI, ctx.FromURI} {
			raw := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(uri), "sip:"), "sips:")
			parts := strings.SplitN(raw, "@", 2)
			if len(parts) == 0 || !validator.NotBlank(parts[0]) {
				continue
			}
			user = strings.TrimSpace(parts[0])
			if idx := strings.IndexByte(user, ';'); idx >= 0 {
				user = strings.TrimSpace(user[:idx])
			}
			if validator.NotBlank(user) {
				break
			}
		}
		if !validator.NotBlank(user) {
			return &sip_infra.SIPError{Code: 404, Message: "No routable SIP user found in SIP URI", Err: sip_infra.ErrAuthRequired}
		}

		routeKind := "did"
		routeValue := user
		if strings.HasPrefix(user, "agent-") {
			routeKind = "agent"
			routeValue = strings.TrimSpace(user[len("agent-"):])
		} else if strings.HasPrefix(user, "did-") {
			routeValue = strings.TrimSpace(user[len("did-"):])
		}
		if !validator.NotBlank(routeValue) || strings.Contains(routeValue, ":") {
			return &sip_infra.SIPError{Code: 404, Message: "Invalid SIP route user", Err: sip_infra.ErrAuthRequired}
		}
		if !validator.NonNil(m.postgres) {
			return &sip_infra.SIPError{Code: 500, Message: "SIP route resolver not configured", Err: sip_infra.ErrInvalidConfig}
		}

		db := m.postgres.DB(m.ctx)
		var assistantID uint64
		var projectID uint64
		var organizationID uint64
		if routeKind == "agent" {
			parsedAssistantID, err := strconv.ParseUint(routeValue, 10, 64)
			if err != nil {
				m.logger.Warnw("SIP: invalid agent route",
					"call_id", ctx.CallID,
					"route_value", routeValue,
					"error", err)
				return &sip_infra.SIPError{Code: 404, Message: "Invalid assistant route", Err: sip_infra.ErrAuthRequired}
			}
			if !validator.NonZero(parsedAssistantID) {
				m.logger.Warnw("SIP: invalid agent route",
					"call_id", ctx.CallID,
					"route_value", routeValue,
					"error", "assistant id is zero")
				return &sip_infra.SIPError{Code: 404, Message: "Invalid assistant route", Err: sip_infra.ErrAuthRequired}
			}

			type assistantRouteResult struct {
				ProjectID      uint64
				OrganizationID uint64
			}
			var result assistantRouteResult
			tx := db.Model(&internal_assistant_entity.Assistant{}).
				Select("project_id, organization_id").
				Where("id = ?", parsedAssistantID).
				First(&result)
			if tx.Error != nil {
				m.logger.Warnw("SIP: assistant route lookup failed",
					"call_id", ctx.CallID,
					"assistant_id", parsedAssistantID,
					"error", tx.Error)
				return &sip_infra.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_infra.ErrAuthRequired}
			}
			assistantID = parsedAssistantID
			projectID = result.ProjectID
			organizationID = result.OrganizationID
		} else {
			type didLookupResult struct {
				AssistantID    uint64
				ProjectID      uint64
				OrganizationID uint64
			}
			// Match the INVITE's user part against every inbound identifier a
			// deployment can be reached by: the DID, the extension the PBX dials,
			// and the AOR the trunk registered under. Providers differ in which one
			// they put in the Request-URI, and a "+"-prefixed DID must still match
			// an unprefixed one.
			var result didLookupResult
			tx := db.Model(&internal_assistant_entity.Assistant{}).
				Select("assistants.id AS assistant_id, assistants.project_id, assistants.organization_id").
				Joins("JOIN assistant_phone_deployments apd ON apd.assistant_id = assistants.id").
				Joins("JOIN assistant_deployment_telephony_options o ON o.assistant_deployment_telephony_id = apd.id").
				Where("apd.telephony_provider = ? AND apd.status = ?", "sip", type_enums.RECORD_ACTIVE).
				Where("o.key IN ?", inboundRouteOptionKeys).
				Where("o.value IN ?", inboundRouteCandidates(routeValue)).
				First(&result)
			if tx.Error != nil {
				m.logger.Warnw("SIP: DID route lookup failed",
					"call_id", ctx.CallID,
					"did", routeValue,
					"error", tx.Error)
				return &sip_infra.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_infra.ErrAuthRequired}
			}
			assistantID = result.AssistantID
			projectID = result.ProjectID
			organizationID = result.OrganizationID
		}
		if !validator.AllNonZero(assistantID, projectID, organizationID) {
			m.logger.Warnw("SIP: route returned incomplete scope",
				"call_id", ctx.CallID,
				"route_kind", routeKind,
				"route_value", routeValue,
				"assistant_id", assistantID,
				"project_id", projectID,
				"organization_id", organizationID)
			return &sip_infra.SIPError{Code: 404, Message: "No assistant found for this SIP route", Err: sip_infra.ErrAuthRequired}
		}

		ctx.AssistantID = strconv.FormatUint(assistantID, 10)
		ctx.Auth = &types.ProjectScope{
			ProjectId:      &projectID,
			OrganizationId: &organizationID,
		}
		if !validator.NonNil(m.assistantService) {
			return &sip_infra.SIPError{Code: 500, Message: "SIP assistant resolver not configured", Err: sip_infra.ErrInvalidConfig}
		}

		assistant, err := m.assistantService.Get(m.ctx, ctx.Auth, assistantID, utils.GetVersionDefinition("latest"),
			&internal_services.GetAssistantOption{InjectPhoneDeployment: true})
		if err != nil {
			m.logger.Error("SIP: assistant not found",
				"call_id", ctx.CallID,
				"method", ctx.Method,
				"assistant_id", assistantID,
				"error", err)
			return &sip_infra.SIPError{Code: 404, Message: "Assistant not found", Err: sip_infra.ErrAuthRequired}
		}
		if !validator.NonNil(ctx.Auth.GetCurrentProjectId()) ||
			!validator.NonZero(assistant.ProjectId) ||
			*ctx.Auth.GetCurrentProjectId() != assistant.ProjectId {
			return &sip_infra.SIPError{Code: 403, Message: "API key does not have access to this assistant", Err: sip_infra.ErrAuthRequired}
		}
		ctx.Assistant = assistant

		m.logger.Infow("SIP: routed inbound call",
			"call_id", ctx.CallID,
			"route_kind", routeKind,
			"route_value", routeValue,
			"assistant_id", assistantID)

		return nil
	}
}

// inboundRouteOptionKeys are the deployment option keys an inbound INVITE user
// part may match. "phone" is the DID, "extension" is the identifier the PBX
// dials internally, and "sip_aor_user" is the registered AOR — providers put
// different ones in the Request-URI depending on how the trunk is provisioned.
var inboundRouteOptionKeys = []string{"phone", "extension", "sip_aor_user"}

// inboundRouteCandidates expands a SIP user part into the equivalent stored
// values, so a "+"-prefixed DID matches an unprefixed one and vice versa.
func inboundRouteCandidates(routeValue string) []string {
	value := strings.TrimSpace(routeValue)
	if value == "" {
		return []string{""}
	}
	candidates := []string{value}
	if trimmed := strings.TrimPrefix(value, "+"); trimmed != value {
		candidates = append(candidates, trimmed)
	} else {
		candidates = append(candidates, "+"+value)
	}
	return candidates
}
