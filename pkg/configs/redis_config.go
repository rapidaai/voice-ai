// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type RedisConfig struct {
	Host               string    `mapstructure:"host" validate:"required"`
	Port               int       `mapstructure:"port" validate:"required"`
	Db                 int       `mapstructure:"db"`
	MaxConnection      int       `mapstructure:"max_connection" validate:"required"`
	Auth               BasicAuth `mapstructure:"auth"`
	InsecureSkipVerify bool      `mapstructure:"insecure_skip_verify"`
}
