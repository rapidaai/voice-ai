// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_artifact_storage

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

const (
	awsOptionCredentialIDKey      = "rapida.credential_id"
	awsOptionBucketKey            = "s3_bucket_name"
	awsOptionPrefixKey            = "prefix"
	awsOptionTimeoutSecondsKey    = "timeout_seconds"
	awsDefaultArtifactPushTimeout = 30 * time.Second
)

type awsExecutor struct {
	ctx           context.Context
	logger        commons.Logger
	configuration *internal_assistant_entity.AssistantConfiguration
	caller        internal_type.InternalCaller
	auth          types.SimplePrinciple
	onPacket      func(context.Context, ...internal_type.Packet) error
}

type AWSOption func(*awsExecutor)

func WithAWSContext(ctx context.Context) AWSOption {
	return func(executor *awsExecutor) {
		executor.ctx = ctx
	}
}

func WithAWSLogger(logger commons.Logger) AWSOption {
	return func(executor *awsExecutor) {
		executor.logger = logger
	}
}

func WithAWSConfiguration(configuration *internal_assistant_entity.AssistantConfiguration) AWSOption {
	return func(executor *awsExecutor) {
		executor.configuration = configuration
	}
}

func WithAWSCaller(caller internal_type.InternalCaller) AWSOption {
	return func(executor *awsExecutor) {
		executor.caller = caller
	}
}

func WithAWSAuth(auth types.SimplePrinciple) AWSOption {
	return func(executor *awsExecutor) {
		executor.auth = auth
	}
}

func WithAWSOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) AWSOption {
	return func(executor *awsExecutor) {
		executor.onPacket = onPacket
	}
}

func NewAWS(opts ...AWSOption) (internal_type.ArtifactPushExecutor, error) {
	executor := &awsExecutor{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(executor)
		}
	}
	if executor.ctx == nil {
		executor.ctx = context.Background()
	}
	start := time.Now()
	if executor.configuration == nil {
		return nil, fmt.Errorf("artifact push storage aws: configuration is required")
	}
	if executor.onPacket == nil {
		return nil, fmt.Errorf("artifact push storage aws: onPacket is required")
	}
	credentialID, _ := executor.configuration.GetOptions().GetUint64(awsOptionCredentialIDKey)
	if credentialID != 0 {
		if executor.caller == nil {
			return nil, fmt.Errorf("artifact push storage aws: caller is required when rapida.credential_id is configured")
		}
		if executor.auth == nil {
			return nil, fmt.Errorf("artifact push storage aws: auth is required when rapida.credential_id is configured")
		}
	}
	_ = executor.onPacket(executor.ctx,
		internal_type.ObservabilityMetricRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordMetric{
				Attributes: observability.Attributes{
					"provider":         executor.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", executor.configuration.Id),
					"executor":         executor.Name(),
				},
				Metrics: []*protos.Metric{{
					Name:        observability.MetricStorageInitLatencyMs,
					Value:       fmt.Sprintf("%d", time.Since(start).Milliseconds()),
					Description: "Storage initialization latency in milliseconds",
				}},
			},
		},
		internal_type.ObservabilityLogRecordPacket{
			Scope: internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: fmt.Sprintf("%s: initialization completed", executor.Name()),
				Attributes: observability.Attributes{
					"component":        observability.ComponentStorage.String(),
					"operation":        "initialize_executor",
					"provider":         executor.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", executor.configuration.Id),
					"options":          observability.AttributeValue(executor.configuration.GetOptions()),
				},
				OccurredAt: time.Now(),
			},
		},
	)
	return executor, nil
}

func (e *awsExecutor) Name() string {
	configurationName, _ := e.configuration.GetOptions().GetString("name")
	if configurationName == "" {
		configurationName = fmt.Sprintf("%d", e.configuration.Id)
	}
	return fmt.Sprintf("artifact-push-%s-%s", e.configuration.Provider, configurationName)
}

func (e *awsExecutor) Options() utils.Option {
	return e.configuration.GetOptions()
}

func (e *awsExecutor) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (e *awsExecutor) Close(context.Context) error {
	return nil
}

func (e *awsExecutor) Execute(ctx context.Context, input internal_type.ArtifactPushInput) (*internal_type.ArtifactPushOutput, error) {
	pushStartedAt := time.Now()
	options := e.Options()
	output := internal_type.ArtifactPushOutput{
		Provider:        e.configuration.Provider,
		ConfigurationID: e.configuration.Id,
		Results:         make([]internal_type.ArtifactPushResult, 0, len(input.Artifacts)),
	}
	artifacts := filterArtifactsToPush(input.Artifacts, options)

	bucketName, _ := options.GetString(awsOptionBucketKey)
	region, _ := options.GetString("region")
	assumeRole, _ := options.GetString("assume_role")
	accessKeyID, _ := options.GetString("access_key_id")
	secretKey, _ := options.GetString("secret_access_key")

	credentialID, _ := options.GetUint64(awsOptionCredentialIDKey)
	if credentialID != 0 {
		credential, err := e.caller.VaultCaller().GetCredential(ctx, e.auth, credentialID)
		if err != nil {
			executeErr := fmt.Errorf("artifact push storage: get credential %d for %s: %w", credentialID, input.ContextID, err)
			_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "External artifact push failed",
					Attributes: observability.Attributes{
						"component":        observability.ComponentStorage.String(),
						"operation":        "push_artifact",
						"provider":         e.configuration.Provider,
						"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
						"context_id":       input.ContextID,
						"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
						"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
						"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
						"error":            executeErr.Error(),
						"error_type":       fmt.Sprintf("%T", executeErr),
					},
				},
			})
			return nil, executeErr
		}
		credentialValues := credential.GetValue().AsMap()
		if value, ok := credentialValues[awsOptionBucketKey]; bucketName == "" && ok {
			bucketName = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["region"]; region == "" && ok {
			region = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["assume_role"]; assumeRole == "" && ok {
			assumeRole = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["access_key_id"]; accessKeyID == "" && ok {
			accessKeyID = fmt.Sprintf("%v", value)
		}
		if value, ok := credentialValues["secret_access_key"]; secretKey == "" && ok {
			secretKey = fmt.Sprintf("%v", value)
		}
	}
	destinationAssetStoreConfig := configs.AssetStoreConfig{
		StorageType:       string(configs.S3),
		StoragePathPrefix: bucketName,
		Auth: &configs.AwsConfig{
			Region:      region,
			AssumeRole:  assumeRole,
			AccessKeyId: accessKeyID,
			SecretKey:   secretKey,
		},
	}
	if !validator.NotBlank(destinationAssetStoreConfig.StoragePathPrefix) {
		executeErr := fmt.Errorf("artifact push storage: bucket is required for %s", e.configuration.Provider)
		_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "External artifact push failed",
				Attributes: observability.Attributes{
					"component":        observability.ComponentStorage.String(),
					"operation":        "push_artifact",
					"provider":         e.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
					"context_id":       input.ContextID,
					"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
					"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
					"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
					"error":            executeErr.Error(),
					"error_type":       fmt.Sprintf("%T", executeErr),
				},
			},
		})
		return nil, executeErr
	}
	if !validator.NotBlank(destinationAssetStoreConfig.Auth.Region) {
		executeErr := fmt.Errorf("artifact push storage: region is required for %s", e.configuration.Provider)
		_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
			ContextID: input.ContextID,
			Scope:     internal_type.ObservabilityRecordScopeConversation,
			Record: observability.RecordLog{
				Level:   observability.LevelError,
				Message: "External artifact push failed",
				Attributes: observability.Attributes{
					"component":        observability.ComponentStorage.String(),
					"operation":        "push_artifact",
					"provider":         e.configuration.Provider,
					"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
					"context_id":       input.ContextID,
					"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
					"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
					"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
					"error":            executeErr.Error(),
					"error_type":       fmt.Sprintf("%T", executeErr),
				},
			},
		})
		return nil, executeErr
	}

	pushTimeout := awsDefaultArtifactPushTimeout
	if configuredTimeoutSeconds, _ := options.GetUint32(awsOptionTimeoutSecondsKey); configuredTimeoutSeconds > 0 {
		pushTimeout = time.Duration(configuredTimeoutSeconds) * time.Second
	}
	pushContext, cancelPushContext := context.WithTimeout(ctx, pushTimeout)
	defer cancelPushContext()

	destinationStorage := storage_files.NewStorage(destinationAssetStoreConfig, e.logger)
	configuredPrefix, _ := options.GetString(awsOptionPrefixKey)

	for _, artifact := range artifacts {
		artifactFileName := artifact.Name
		if filepath.Ext(artifactFileName) == "" {
			switch artifact.ContentType {
			case "audio/wav":
				artifactFileName += ".wav"
			case "application/json":
				artifactFileName += ".json"
			case "text/plain":
				artifactFileName += ".txt"
			}
		}
		destinationObjectKey := path.Join(fmt.Sprintf("%d", input.ConversationID), artifactFileName)
		if validator.NotBlank(configuredPrefix) {
			destinationObjectKey = path.Join(configuredPrefix, destinationObjectKey)
		}

		storageResult := destinationStorage.Store(pushContext, destinationObjectKey, artifact.Content)
		if storageResult.Error != nil {
			executeErr := fmt.Errorf("artifact push storage: push artifact %q to %q: %w", artifact.Name, destinationObjectKey, storageResult.Error)
			_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
				ContextID: input.ContextID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordLog{
					Level:   observability.LevelError,
					Message: "External artifact push failed",
					Attributes: observability.Attributes{
						"component":        observability.ComponentStorage.String(),
						"operation":        "push_artifact",
						"provider":         e.configuration.Provider,
						"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
						"context_id":       input.ContextID,
						"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
						"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
						"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
						"error":            executeErr.Error(),
						"error_type":       fmt.Sprintf("%T", executeErr),
					},
				},
			})
			return nil, executeErr
		}
		output.Results = append(output.Results, internal_type.ArtifactPushResult{
			Name:           artifact.Name,
			Type:           artifact.Type,
			ContentType:    artifact.ContentType,
			DestinationKey: destinationObjectKey,
			CompletePath:   storageResult.CompletePath,
			StorageType:    string(storageResult.StorageType),
		})
	}

	_ = e.onPacket(ctx, internal_type.ObservabilityLogRecordPacket{
		ContextID: input.ContextID,
		Scope:     internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "External artifact push completed",
			Attributes: observability.Attributes{
				"component":        observability.ComponentStorage.String(),
				"operation":        "push_artifact",
				"provider":         e.configuration.Provider,
				"configuration_id": fmt.Sprintf("%d", e.configuration.Id),
				"context_id":       input.ContextID,
				"artifact_count":   fmt.Sprintf("%d", len(input.Artifacts)),
				"pushed_count":     fmt.Sprintf("%d", len(output.Results)),
				"duration_ms":      fmt.Sprintf("%d", time.Since(pushStartedAt).Milliseconds()),
			},
		},
	})
	return &output, nil
}
