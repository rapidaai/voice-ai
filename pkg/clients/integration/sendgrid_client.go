package integration_client

import (
	"context"
	"math"

	"github.com/rapidaai/config"
	commons "github.com/rapidaai/pkg/commons"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SendgridServiceClient interface {
	WelcomeEmail(c context.Context, userId uint64, name, email string) (*integration_api.WelcomeEmailResponse, error)
	ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*integration_api.ResetPasswordEmailResponse, error)
	InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*integration_api.InviteMemeberEmailResponse, error)
}

type sendgridServiceClient struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	sendgridClient integration_api.SendgridServiceClient
}

func NewSendgridServiceClientGRPC(config *config.AppConfig, logger commons.Logger) SendgridServiceClient {
	logger.Debugf("conntecting to integration client with %s", config.IntegrationHost)

	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt64),
			grpc.MaxCallSendMsgSize(math.MaxInt64),
		),
	}
	conn, err := grpc.NewClient(config.IntegrationHost,
		grpcOpts...)

	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &sendgridServiceClient{
		cfg:            config,
		logger:         logger,
		sendgridClient: integration_api.NewSendgridServiceClient(conn),
	}
}

func (client *sendgridServiceClient) WelcomeEmail(c context.Context, userId uint64, name, email string) (*integration_api.WelcomeEmailResponse, error) {
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
func (client *sendgridServiceClient) ResetPasswordEmail(c context.Context, userId uint64, name, email, resetPasswordLink string) (*integration_api.ResetPasswordEmailResponse, error) {
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

func (client *sendgridServiceClient) InviteMemberEmail(c context.Context, userId uint64, name, email, organizationName, projectName, inviterName string) (*integration_api.InviteMemeberEmailResponse, error) {
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
