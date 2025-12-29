// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package external_clients

import (
	"context"

	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
)

type Contact struct {
	Name  string
	Email string
}

type Emailer interface {
	EmailText(ctx context.Context, to Contact, subject string, content string) error
	EmailRichText(ctx context.Context, to Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error
	EmailTemplate(ctx context.Context, to Contact, subject string, templateId string, args map[string]string) error
}
