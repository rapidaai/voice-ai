package batch_processors

import (
	"context"

	"github.com/rapidaai/pkg/batches"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
)

type localBatchProcessor struct {
	config configs.BatchConfig
	logger commons.Logger
}

func NewLocalBatchProcessor(config configs.BatchConfig, logger commons.Logger, opts map[string]string) batches.BatchProcessor {
	return &localBatchProcessor{
		config: config,
		logger: logger,
	}
}

func (localProcessor *localBatchProcessor) Process(ctx context.Context, args map[string]string) batches.BatchOutput {
	localProcessor.logger.Infof("Starting execution of job : %v with local framework", args)

	// cmd := exec.CommandContext(ctx, "/bin/sh", fmt.Sprintf("%s/testsuite-with-s3-dataset-runner/dev/test-run.sh", homePath), fmt.Sprintf("%d", lLauncher.Job.TestId), fmt.Sprintf("%d", lLauncher.Job.OrgId), fmt.Sprintf("%d", lLauncher.Job.ProjectId), fmt.Sprintf("%s/testsuite-with-s3-dataset-runner", homePath))
	// logDirPath := fmt.Sprintf("%s/logs/%d/%d", homePath, lLauncher.Job.OrgId, lLauncher.Job.TestId)

	// lLauncher.Logger.Infof("Location of job logs : %s", logDirPath)
	// if err := lLauncher.executeCmd(ctx, cmd, logDirPath); err != nil {
	// 	lLauncher.Logger.Errorf("Unable to execute test job for testId : %d", lLauncher.Job.TestId)
	// 	return &jobs.JobResponse{
	// 		Success: false,
	// 		Error:   err,
	// 	}
	// } else {
	// 	lLauncher.Logger.Errorf("Successfull completed test job for testId : %d", lLauncher.Job.TestId)
	// 	return &jobs.JobResponse{
	// 		Success: true,
	// 	}
	// }
	return batches.BatchOutput{
		Success: true,
	}
}
