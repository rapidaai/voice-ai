// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type BasicAuth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type ApiKeyAuth struct {
	ApiKey string `mapstructure:"api_key"`
}
