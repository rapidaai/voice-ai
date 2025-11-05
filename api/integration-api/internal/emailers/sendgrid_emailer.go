package internal_emailers

import (
	"context"

	"github.com/rapidaai/api/integration-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type sendgridEmailer struct {
	logger commons.Logger
	cfg    *config.IntegrationConfig
	client *sendgrid.Client
}

func NewSendgridEmailer(logger commons.Logger, cfg *config.IntegrationConfig) Emailer {
	return &sendgridEmailer{
		logger: logger,
		cfg:    cfg,
		client: sendgrid.NewSendClient(cfg.SendgridApiKey),
	}
}

type Contact struct {
	Name  string
	Email string
}

func (sg *sendgridEmailer) From() *mail.Email {
	return mail.NewEmail("Mukesh Singh", "mukesh@rapida.ai")
}

func (sg *sendgridEmailer) EmailText(ctx context.Context, to Contact, subject string, content string) error {
	message := mail.NewSingleEmailPlainText(sg.From(), subject, mail.NewEmail(to.Name, to.Email), content)
	response, err := sg.client.Send(message)
	if err != nil {
		sg.logger.Errorf("got an error while sending email %v", err)
		return err
	} else {
		sg.logger.Debugf("send email successful %v", response)
	}
	return nil
}

func (sg *sendgridEmailer) EmailTemplate(ctx context.Context, to Contact, subject string, templateId string, args map[string]string) error {
	personalization := mail.NewPersonalization()
	personalization.AddTos(mail.NewEmail(to.Name, to.Email))
	for k, v := range args {
		personalization.SetDynamicTemplateData(k, v)
	}

	response, err := sg.client.Send(mail.NewV3Mail().SetFrom(sg.From()).SetTemplateID(templateId).AddPersonalizations(personalization))
	if err != nil {
		sg.logger.Errorf("got an error while sending email %v", err)
		return err
	} else {
		sg.logger.Debugf("send email successful %v", response)
	}
	return nil
}
