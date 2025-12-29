// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type DynamoConfig struct {
	Auth       AwsConfig `mapstructure:"auth"`
	MaxRetries int       `mapstructure:"max_retries" validate:"required"`
}
