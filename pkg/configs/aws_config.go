// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type AwsConfig struct {
	Region      string `mapstructure:"region" validate:"required"`
	AssumeRole  string `mapstructure:"assume_role"`
	AccessKeyId string `mapstructure:"access_key_id"`
	SecretKey   string `mapstructure:"secret_key"`
}

func (cfg *AwsConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"region":        cfg.Region,
		"assume_role":   cfg.AssumeRole,
		"access_key_id": cfg.AccessKeyId,
		"secret_key":    cfg.SecretKey,
	}
}
