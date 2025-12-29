// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package external_emailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/utils"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go/aws"
)

type sesEmailer struct {
	cfg       *configs.EmailerConfig
	logger    commons.Logger
	sesClient *ses.Client
}

func NewSESEmailer(logger commons.Logger, config *configs.EmailerConfig) external_clients.Emailer {
	cfg, _ := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(config.Auth.Region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.Auth.AccessKeyId,
			config.Auth.SecretKey,
			"",
		)),
	)
	return &sesEmailer{
		logger:    logger,
		cfg:       config,
		sesClient: ses.NewFromConfig(cfg),
	}
}

func (s *sesEmailer) EmailText(ctx context.Context, to external_clients.Contact, subject string, content string) error {
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to.Email},
		},
		Message: &types.Message{
			Body: &types.Body{
				Text: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(content),
				},
			},
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)),
	}
	_, err := s.sesClient.SendEmail(ctx, input)
	if err != nil {
		s.logger.Errorf("send email error %+v", err)
		return fmt.Errorf("error while sending email from ses")
	}
	return nil
}
func (s *sesEmailer) EmailRichText(ctx context.Context, to external_clients.Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error {
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
	input := &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to.Email},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(bodyBuffer.String()),
				},
				Text: &types.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(txtBuffer.String()),
				},
			},
			Subject: &types.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)),
	}
	_, err = s.sesClient.SendEmail(ctx, input)
	if err != nil {
		s.logger.Errorf("send email error %+v", err)
		return fmt.Errorf("error while sending email from ses")
	}
	return nil
}

func (s *sesEmailer) EmailTemplate(ctx context.Context, to external_clients.Contact, subject string, templateId string, args map[string]string) error {

	templateData, err := json.Marshal(args)
	if err != nil {
		s.logger.Errorf("error marshaling template data: %+v", err)
		return fmt.Errorf("error while preparing template data")
	}
	input := &ses.SendTemplatedEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to.Email},
		},
		Template:     utils.Ptr(templateId),
		TemplateData: utils.Ptr(string(templateData)),
		Source:       aws.String(fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)),
	}
	_, err = s.sesClient.SendTemplatedEmail(ctx, input)
	if err != nil {
		s.logger.Errorf("send email error %+v", err)
		return fmt.Errorf("error while sending email from ses")
	}
	return nil
}
