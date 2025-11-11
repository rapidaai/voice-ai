package internal_emailers

import "context"

type Emailer interface {
	EmailText(ctx context.Context, to Contact, subject string, content string) error
	EmailTemplate(ctx context.Context, to Contact, subject string, templateId string, args map[string]string) error
}
