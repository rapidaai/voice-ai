// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type OpenSearchConfig struct {
	Schema        string    `mapstructure:"schema" validate:"required"`
	Host          string    `mapstructure:"host" validate:"required"`
	Port          *int      `mapstructure:"port"`
	Auth          BasicAuth `mapstructure:"auth"`
	MaxRetries    int       `mapstructure:"max_retries" validate:"required"`
	MaxConnection int       `mapstructure:"max_connection" validate:"required"`
}
