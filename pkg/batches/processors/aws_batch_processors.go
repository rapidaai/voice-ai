package batch_processors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/rapidaai/pkg/batches"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
)

type awsBatchProcessor struct {
	config configs.BatchConfig
	logger commons.Logger
	opts   map[string]string
}

func NewAWSBatchProcessor(config configs.BatchConfig, logger commons.Logger, opts map[string]string) batches.BatchProcessor {
	return &awsBatchProcessor{
		config: config,
		logger: logger,
		opts:   opts,
	}
}

const (
	BatchTestId    = "BATCH_TEST_ID"
	BatchOrgId     = "BATCH_ORG_ID"
	BatchProjectId = "BATCH_PROJECT_ID"
)

func (awsProcessor *awsBatchProcessor) Process(ctx context.Context, args map[string]string) batches.BatchOutput {
	awsProcessor.logger.Infof("Starting execution of job with args : %v with aws framework", args)
	if awsProcessor.config.Auth == nil {
		return batches.BatchOutput{
			Success: false,
			Error:   fmt.Errorf("aws batch configuration is missing"),
		}
	}

	config := aws.Config{
		Region: aws.String(awsProcessor.config.Auth.Region),
		Credentials: credentials.NewStaticCredentials(
			awsProcessor.config.Auth.AccessKeyId,
			awsProcessor.config.Auth.SecretKey, "",
		),
	}
	sessionOptions := awsSession.Options{
		Config:            config,
		SharedConfigState: awsSession.SharedConfigEnable,
	}

	awsSession, err := awsSession.NewSessionWithOptions(sessionOptions)
	if err != nil {
		awsProcessor.logger.Errorf("unable to acquire the session from aws with err %v", err)
		return batches.BatchOutput{
			Success: false,
			Error:   err,
		}
	}
	batchClient := batch.New(awsSession)
	joboutout, err := batchClient.SubmitJobWithContext(ctx, awsProcessor.input(args))

	if err != nil {
		awsProcessor.logger.Errorf("Unable to launch aws batch job error %v", err)
		return batches.BatchOutput{
			Success: false,
			Error:   err,
		}
	}

	awsProcessor.logger.Infof("Successfully launched remote job err %v", err)
	awsProcessor.logger.Infof("Job info is : %v", joboutout)
	return batches.BatchOutput{
		Success: true,
	}
}

func (awsProcessor *awsBatchProcessor) input(args map[string]string) *batch.SubmitJobInput {
	ji := &batch.SubmitJobInput{
		ContainerOverrides: &batch.ContainerOverrides{
			Environment: awsProcessor.args(args),
		},
	}
	if value, ok := awsProcessor.opts["jobName"]; ok {
		ji.JobName = aws.String(value)
	}
	if value, ok := awsProcessor.opts["jobDefinitionName"]; ok {
		ji.JobDefinition = aws.String(value)
	}
	if value, ok := awsProcessor.opts["jobQueue"]; ok {
		ji.JobQueue = aws.String(value)
	}

	return ji

	// return {
	// 	"jobDefinitionName": "knowledge-dataset-runner-jd-01",
	// 	"jobDefinitionArn": "arn:aws:batch:ap-south-1:424737338319:job-definition/knowledge-dataset-runner-jd-01:6",
	// 	"revision": 6,
	// 	"status": "ACTIVE",
	// 	"type": "container",
	// 	"parameters": {},
	// 	"containerProperties": {
	// 	  "image": "424737338319.dkr.ecr.ap-south-1.amazonaws.com/knowledge-dataset-runner:latest",
	// 	  "command": [
	// 		"echo",
	// 		"hello world"
	// 	  ],
	// 	  "executionRoleArn": "arn:aws:iam::424737338319:role/AWSTestsuiteTestingBatchJobRole",
	// 	  "volumes": [],
	// 	  "environment": [
	// 		{
	// 		  "name": "env",
	// 		  "value": "Production"
	// 		}
	// 	  ],
	// 	  "mountPoints": [],
	// 	  "ulimits": [],
	// 	  "resourceRequirements": [
	// 		{
	// 		  "value": "2.0",
	// 		  "type": "VCPU"
	// 		},
	// 		{
	// 		  "value": "4096",
	// 		  "type": "MEMORY"
	// 		}
	// 	  ],
	// 	  "logConfiguration": {
	// 		"logDriver": "awslogs",
	// 		"options": {},
	// 		"secretOptions": []
	// 	  },
	// 	  "secrets": [],
	// 	  "networkConfiguration": {
	// 		"assignPublicIp": "ENABLED",
	// 		"interfaceConfigurations": []
	// 	  },
	// 	  "fargatePlatformConfiguration": {
	// 		"platformVersion": "LATEST"
	// 	  },
	// 	  "runtimePlatform": {
	// 		"operatingSystemFamily": "LINUX",
	// 		"cpuArchitecture": "X86_64"
	// 	  }
	// 	},
	// 	"tags": {},
	// 	"platformCapabilities": [
	// 	  "FARGATE"
	// 	],
	// 	"containerOrchestrationType": "ECS"
	//   }
}
func (awsProcessor *awsBatchProcessor) args(args map[string]string) []*batch.KeyValuePair {
	kv := make([]*batch.KeyValuePair, 0)
	for k, v := range args {
		kv = append(kv, &batch.KeyValuePair{Name: aws.String(k), Value: aws.String(v)})
	}
	return kv
}
