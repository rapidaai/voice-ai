// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type WeaviateConfig struct {
	Host   string     `mapstructure:"host"`
	Scheme string     `mapstructure:"scheme"`
	Auth   ApiKeyAuth `mapstructure:"auth"`
}
