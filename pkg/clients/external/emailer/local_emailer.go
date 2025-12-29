// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package external_emailer

import (
	"context"

	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	"github.com/rapidaai/pkg/commons"
)

type localEmailer struct {
	logger commons.Logger
}

func NewLocalEmailer(logger commons.Logger) external_clients.Emailer {
	return &localEmailer{
		logger: logger,
	}
}

func (sg *localEmailer) EmailText(ctx context.Context, to external_clients.Contact, subject string, content string) error {
	sg.logger.Info(ctx, "Sending text email", map[string]interface{}{
		"to":      to,
		"subject": subject,
		"content": content,
	})
	return nil
}

func (sg *localEmailer) EmailRichText(ctx context.Context, to external_clients.Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error {
	sg.logger.Info(ctx, "Sending template email", map[string]interface{}{
		"to":       to,
		"subject":  subject,
		"template": template,
		"args":     args,
	})
	return nil
}

func (sg *localEmailer) EmailTemplate(ctx context.Context, to external_clients.Contact, subject string, templateId string, args map[string]string) error {
	sg.logger.Info(ctx, "Sending template email", map[string]interface{}{
		"to":         to,
		"subject":    subject,
		"templateId": templateId,
		"args":       args,
	})
	return nil
}
