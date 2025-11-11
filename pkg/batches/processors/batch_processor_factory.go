package batch_processors

import (
	"github.com/rapidaai/pkg/batches"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
)

func NewBatchProcessor(config configs.BatchConfig, logger commons.Logger, opts map[string]string) batches.BatchProcessor {
	if !config.IsLocal() {
		return NewAWSBatchProcessor(config, logger, opts)
	}
	return NewLocalBatchProcessor(config, logger, opts)
}
