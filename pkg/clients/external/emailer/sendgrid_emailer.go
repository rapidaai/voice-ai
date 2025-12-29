// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package external_emailer

import (
	"bytes"
	"context"
	"fmt"

	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type sendgridEmailer struct {
	logger commons.Logger
	cfg    *configs.EmailerConfig
	client *sendgrid.Client
}

func NewSendgridEmailer(logger commons.Logger, cfg *configs.EmailerConfig) external_clients.Emailer {
	return &sendgridEmailer{
		logger: logger,
		cfg:    cfg,
		client: sendgrid.NewSendClient(*cfg.SendgridKey),
	}
}

func (sg *sendgridEmailer) From() *mail.Email {
	return mail.NewEmail(sg.cfg.FromName, sg.cfg.FromEmail)
}

func (sg *sendgridEmailer) EmailText(ctx context.Context, to external_clients.Contact, subject string, content string) error {
	sg.logger.Infof("sending email text to user %s with subject %s", to.Email, subject)
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

func (sg *sendgridEmailer) EmailRichText(ctx context.Context, to external_clients.Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error {
	sg.logger.Infof("sending email rich text to user %s with subject %s", to.Email, subject)
	tmpl, err := external_emailer_template.GetHTMLTemplate(template)
	if err != nil {
		return fmt.Errorf("error parsing email template: %w", err)
	}
	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, args); err != nil {
		return fmt.Errorf("error executing email template: %w", err)
	}

	tmpl2, err := external_emailer_template.GetTextTemplate(template)
	if err != nil {
		return fmt.Errorf("error parsing email template: %w", err)
	}

	var txtBuffer bytes.Buffer
	if err := tmpl2.Execute(&txtBuffer, args); err != nil {
		return fmt.Errorf("error executing email template: %w", err)
	}

	message := mail.NewSingleEmail(sg.From(), subject, mail.NewEmail(to.Name, to.Email), txtBuffer.String(), bodyBuffer.String())
	response, err := sg.client.Send(message)
	if err != nil {
		sg.logger.Errorf("got an error while sending email %v", err)
		return err
	} else {
		sg.logger.Debugf("send email successful %v", response)
	}
	return nil
}

func (sg *sendgridEmailer) EmailTemplate(ctx context.Context, to external_clients.Contact, subject string, templateId string, args map[string]string) error {
	sg.logger.Infof("sending email template to user %s with subject %s and template %s", to.Email, subject, templateId)
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
