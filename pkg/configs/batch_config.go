// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package configs

type BatchConfig struct {
	BatchType   string     `mapstructure:"batch_type"`
	BatchScript string     `mapstructure:"batch_script"`
	Auth        *AwsConfig `mapstructure:"auth"`
}

func (b *BatchConfig) IsLocal() bool {
	return b.BatchType != "aws"
}
