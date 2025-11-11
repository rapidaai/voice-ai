package integration_api

import (
	"context"
	"fmt"

	config "github.com/rapidaai/api/integration-api/config"
	internal_emailers "github.com/rapidaai/api/integration-api/internal/emailers"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	integration_api "github.com/rapidaai/protos"
)

type sendgridIntegrationApi struct {
	cfg     *config.IntegrationConfig
	logger  commons.Logger
	emailer internal_emailers.Emailer
}

type sendgridIntegrationRPCApi struct {
	sendgridIntegrationApi
}

type sendgridIntegrationGRPCApi struct {
	sendgridIntegrationApi
}

var (
	welcomeTemplateId       = "d-501a158ede1e49629c1d23715f566db5"
	resetPasswordTemplateId = "d-5030619110b34006b1694a6da2de30d3"
	inviteUserTemplateId    = "d-4e99105f827443718202631bf0887743"
)

func NewSendgridRPC(config *config.IntegrationConfig, logger commons.Logger) *sendgridIntegrationRPCApi {
	return &sendgridIntegrationRPCApi{
		sendgridIntegrationApi{
			cfg:     config,
			logger:  logger,
			emailer: internal_emailers.NewSendgridEmailer(logger, config),
		},
	}
}

func NewSendgridGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.SendgridServiceServer {
	return &sendgridIntegrationGRPCApi{
		sendgridIntegrationApi{
			cfg:     config,
			logger:  logger,
			emailer: internal_emailers.NewSendgridEmailer(logger, config),
		},
	}
}

func (sG *sendgridIntegrationGRPCApi) WelcomeEmail(c context.Context, iaRequest *integration_api.WelcomeEmailRequest) (*integration_api.WelcomeEmailResponse, error) {
	err := sG.emailer.EmailTemplate(
		c,
		internal_emailers.Contact{Name: iaRequest.To.Name, Email: iaRequest.To.Email},
		"Welcome to RapidaAI!",
		welcomeTemplateId,
		map[string]string{
			"name": iaRequest.To.Name,
		},
	)
	if err != nil {
		sG.logger.Errorf("error while sending welcome email %v", err)
		return nil, err
	}
	return &integration_api.WelcomeEmailResponse{
		Code:    200,
		Success: true,
	}, nil

}
func (sG *sendgridIntegrationGRPCApi) ResetPasswordEmail(c context.Context, iaRequest *integration_api.ResetPasswordEmailRequest) (*integration_api.ResetPasswordEmailResponse, error) {
	err := sG.emailer.EmailTemplate(
		c,
		internal_emailers.Contact{Name: iaRequest.To.Name, Email: iaRequest.To.Email},
		"[RapidaAI] Reset your password",
		resetPasswordTemplateId,
		map[string]string{
			"name":              iaRequest.To.Name,
			"passwordResetLink": iaRequest.ResetPasswordLink,
		},
	)
	if err != nil {
		sG.logger.Errorf("error while sending reset password email %v", err)
		return nil, err
	}
	return &integration_api.ResetPasswordEmailResponse{
		Code:    200,
		Success: true,
	}, nil
}
func (sG *sendgridIntegrationGRPCApi) InviteMemberEmail(c context.Context, iaRequest *integration_api.InviteMemeberEmailRequest) (*integration_api.InviteMemeberEmailResponse, error) {
	err := sG.emailer.EmailTemplate(
		c,
		internal_emailers.Contact{Name: iaRequest.To.Name, Email: iaRequest.To.Email},
		fmt.Sprintf("[RapidaAI] %s has invited you to join the %s organization", iaRequest.InviterName, iaRequest.OrganizationName),
		inviteUserTemplateId,
		map[string]string{
			"name":             iaRequest.To.Name,
			"inviterName":      iaRequest.InviterName,
			"organizationName": iaRequest.OrganizationName,
			"projectName":      iaRequest.ProjectName,
		},
	)

	if err != nil {
		sG.logger.Errorf("error while sending invite member email %v", err)
		return nil, err
	}
	return &integration_api.InviteMemeberEmailResponse{
		Code:    200,
		Success: true,
	}, nil
}
