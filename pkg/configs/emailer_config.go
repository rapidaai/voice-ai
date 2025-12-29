// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type EmailProvider string

const (
	SES      EmailProvider = "ses"
	SENDGRID EmailProvider = "sendgrid"
)

// asset_upload_bucket

type EmailerConfig struct {
	EmailProvider string     `mapstructure:"provider" validate:"required"`
	FromEmail     string     `mapstructure:"from_email" validate:"required"`
	FromName      string     `mapstructure:"from_name" validate:"required"`
	Auth          *AwsConfig `mapstructure:"auth"`
	SendgridKey   *string    `mapstructure:"sendgrid_key"`
}

func (cfg *EmailerConfig) Provider() EmailProvider {
	if cfg.EmailProvider == string(SES) {
		return SES
	}
	if cfg.EmailProvider == string(SENDGRID) {
		return SENDGRID
	}
	return SENDGRID
}
