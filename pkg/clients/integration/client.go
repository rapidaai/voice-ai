package integration

import (
	"context"
	"math"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	"github.com/lexatic/web-backend/pkg/commons"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type integrationServiceClient struct {
	cfg                *config.AppConfig
	logger             commons.Logger
	sendgridClient     integration_api.SendgridServiceClient
	auditLoggingClient integration_api.AuditLoggingServiceClient
}

func NewIntegrationServiceClientGRPC(config *config.AppConfig, logger commons.Logger) clients.IntegrationServiceClient {
	logger.Debugf("conntecting to integration client with %s", config.IntegrationHost)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.MaxCallSendMsgSize(math.MaxInt64),
		),
	}
	conn, err := grpc.Dial(config.IntegrationHost,
		grpcOpts...)

	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &integrationServiceClient{
		cfg:                config,
		logger:             logger,
		sendgridClient:     integration_api.NewSendgridServiceClient(conn),
		auditLoggingClient: integration_api.NewAuditLoggingServiceClient(conn),
	}
}

func (client *integrationServiceClient) WelcomeEmail(c context.Context, userId uint64, name, email string) (*integration_api.WelcomeEmailResponse, error) {
	client.logger.Debugf("sending welcome email from integration client")
	res, err := client.sendgridClient.WelcomeEmail(c, &integration_api.WelcomeEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
	})
	if err != nil {
		client.logger.Errorf("unable to send welcome email error %v", err)
		return nil, err
	}
	return res, nil

}

func (client *integrationServiceClient) GetAuditLog(c context.Context, organizationId, projectId uint64, criterias []*integration_api.Criteria, paginate *integration_api.Paginate) (*integration_api.GetAuditLogResponse, error) {
	client.logger.Debugf("Calling to get audit log with org and project")
	res, err := client.auditLoggingClient.GetAuditLog(c, &integration_api.GetAuditLogRequest{
		OrganizationId: organizationId,
		ProjectId:      projectId,
		Criterias:      criterias,
		Paginate:       paginate,
	})
	if err != nil {
		client.logger.Errorf("error while getting audit log error %v", err)
		return nil, err
	}
	return res, nil
}

func (client *integrationServiceClient) ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*integration_api.ResetPasswordEmailResponse, error) {
	client.logger.Debugf("sending reset password email from integration client")
	res, err := client.sendgridClient.ResetPasswordEmail(c, &integration_api.ResetPasswordEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
		ResetPasswordLink: resetPasswordLink,
	})
	if err != nil {
		client.logger.Errorf("unable to send reset password link error %v", err)
		return nil, err
	}
	return res, nil
}

func (client *integrationServiceClient) InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*integration_api.InviteMemeberEmailResponse, error) {
	client.logger.Debugf("sending invite member email from integration client")
	res, err := client.sendgridClient.InviteMemberEmail(c, &integration_api.InviteMemeberEmailRequest{
		UserId: userId,
		To: &integration_api.Contact{
			Name:  name,
			Email: email,
		},
		OrganizationName: organizationName,
		ProjectName:      projectName,
		InviterName:      inviterName,
	})
	if err != nil {
		client.logger.Errorf("unable to send invite member email error %v", err)
		return nil, err
	}
	return res, nil
}
