// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package connectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	commons "github.com/rapidaai/pkg/commons"
	configs "github.com/rapidaai/pkg/configs"
)

type DynamoConnector interface {
	Connector
	DB() *dynamodb.DynamoDB
}

type dynamoConnector struct {
	cfg    *configs.DynamoConfig
	logger commons.Logger
	db     *dynamodb.DynamoDB
}

func NewDynamoConnector(config *configs.DynamoConfig, logger commons.Logger) DynamoConnector {
	return &dynamoConnector{cfg: config, logger: logger}
}

func (dynamo *dynamoConnector) DB() *dynamodb.DynamoDB {
	return dynamo.db
}

func (dynamo *dynamoConnector) Connect(ctx context.Context) error {
	db, err := dynamo.createClient()
	if err != nil {
		dynamo.logger.Errorf("connecting to dynamo db with ends with error %v", err)
		return err
	}
	dynamo.db = db
	return nil
}
func (dynamo *dynamoConnector) Name() string {
	return "dynamodb"
}
func (dynamo *dynamoConnector) IsConnected(ctx context.Context) bool {
	dynamo.logger.Debugf("Calling info for dynamo, yet to impliment")
	return true
}
func (dynamo *dynamoConnector) Disconnect(ctx context.Context) error {
	dynamo.logger.Debug("Disconnecting with opensearch client.")
	dynamo.db = nil
	return nil
}

func (dynamo *dynamoConnector) createClient() (*dynamodb.DynamoDB, error) {
	config := aws.Config{
		Region:     aws.String(dynamo.cfg.Auth.Region),
		MaxRetries: &dynamo.cfg.MaxRetries,
	}
	if dynamo.cfg.Auth.AccessKeyId != "" && dynamo.cfg.Auth.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(
			dynamo.cfg.Auth.AccessKeyId,
			dynamo.cfg.Auth.SecretKey,
			"",
		)
	}
	sessionOptions := awsSession.Options{
		Config:            config,
		SharedConfigState: awsSession.SharedConfigEnable,
	}
	awsSession, err := awsSession.NewSessionWithOptions(sessionOptions)
	if err != nil {
		dynamo.logger.Errorf("failed to get session from given option %v due to %s", sessionOptions, err)
		return nil, fmt.Errorf("failed to get session from given option %v due to %s", sessionOptions, err)
	}
	return dynamodb.New(awsSession), nil
}
