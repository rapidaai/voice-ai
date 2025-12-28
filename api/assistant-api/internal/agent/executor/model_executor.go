// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_agent_tool "github.com/rapidaai/api/assistant-api/internal/agent/tool"
	internal_executors "github.com/rapidaai/api/assistant-api/internal/agent/tool"
	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/errgroup"
)

type modelAssistantExecutor struct {
	logger             commons.Logger
	toolExecutor       internal_executors.ToolExecutor
	providerCredential *protos.VaultCredential
	inputBuilder       integration_client_builders.InputChatBuilder
	history            []*protos.Message
}

func NewModelAssistantExecutor(
	logger commons.Logger,
) AssistantExecutor {
	return &modelAssistantExecutor{
		logger:       logger,
		inputBuilder: integration_client_builders.NewChatInputBuilder(logger),
		toolExecutor: internal_agent_tool.NewToolExecutor(logger),
		history:      make([]*protos.Message, 0),
	}

}

func (executor *modelAssistantExecutor) Name() string {
	return "model"
}

func (executor *modelAssistantExecutor) Initialize(
	ctx context.Context,
	communication internal_adapter_requests.Communication,
) error {
	start := time.Now()
	ctx, span, _ := communication.
		Tracer().
		StartSpan(
			ctx,
			utils.AssistantAgentConnectStage,
			internal_adapter_telemetry.KV{
				K: "executor",
				V: internal_adapter_telemetry.StringValue(executor.Name()),
			},
		)
	defer span.EndSpan(ctx, utils.AssistantAgentConnectStage)
	g, ctx := errgroup.WithContext(ctx)
	var providerCredential *protos.VaultCredential

	// g.Go(func() error {
	// 	behavior, err := communication.GetBehavior()
	// 	if err != nil {
	// 		executor.logger.Errorf("error while fetching deployment behavior: %v", err)
	// 		return nil
	// 	}

	// 	if behavior.Greeting == nil {
	// 		executor.logger.Errorf("error while fetching deployment behavior: %v", err)
	// 		return nil
	// 	}
	// 	greetingCnt := executor.templateParser.Parse(*behavior.Greeting, communication.GetArgs())
	// 	if strings.TrimSpace(greetingCnt) == "" {
	// 		executor.logger.Warnf("empty greeting message, could be space in the table or argument contains space")
	// 		return nil
	// 	}
	// 	greetings := types.NewMessage(
	// 		"assistant", &types.Content{
	// 			ContentType:   commons.TEXT_CONTENT.String(),
	// 			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
	// 			Content:       []byte(greetingCnt),
	// 		},
	// 	)
	// 	communication.OnGeneration(ctx, greetings.Id, greetings)
	// 	communication.OnGenerationComplete(ctx, greetings.Id, greetings, nil)
	// 	executor.history = append(executor.history, greetings.ToProto())

	// 	return nil
	// })
	// Goroutine to get the provider credential
	g.Go(func() error {
		credentialId, err := communication.Assistant().AssistantProviderModel.GetOptions().GetUint64("rapida.credential_id")
		if err != nil {
			executor.logger.Errorf("Error while getting provider model credential ID: %v", err)
			return fmt.Errorf("failed to get credential ID: %w", err)
		}

		span.AddAttributes(ctx, internal_adapter_telemetry.KV{
			K: "vault_id",
			V: internal_adapter_telemetry.IntValue(credentialId),
		})
		providerCredential, err = communication.
			VaultCaller().
			GetCredential(
				ctx, communication.Auth(), credentialId,
			)
		if err != nil {
			executor.logger.Errorf("Error while getting provider model credentials: %v", err)
			return fmt.Errorf("failed to get provider credential: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		executor.history = append(executor.history, communication.GetConversationLogs()...)
		span.AddAttributes(
			ctx,
			internal_adapter_telemetry.KV{
				K: "history_length", V: internal_adapter_telemetry.IntValue(len(executor.history)),
			},
		)
		return nil

	})
	// Goroutine to initialize tool executor
	g.Go(func() error {
		err := executor.toolExecutor.Initialize(ctx, communication)
		if err != nil {
			executor.logger.Errorf("Error initializing tool executor: %v", err)
			return fmt.Errorf("failed to initialize tool executor: %w", err)
		}
		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		executor.logger.Errorf("Error during initialization: %v", err)
		return err
	}
	executor.providerCredential = providerCredential
	executor.logger.Benchmark("DefaultAssistantExecutor.Init", time.Since(start))
	return nil
}

func (executor *modelAssistantExecutor) chat(
	ctx context.Context,
	//
	messageid string,
	// for communication
	communication internal_adapter_requests.Communication,
	// current messages
	in *types.Message,
	// histories or older conversation
	histories ...*protos.Message,
) error {

	start := time.Now()
	var (
		output  *protos.Message
		metrics []*protos.Metric
	)
	request := executor.inputBuilder.
		Chat(
			&protos.Credential{
				Id:    executor.providerCredential.GetId(),
				Value: executor.providerCredential.GetValue(),
			},
			executor.
				inputBuilder.
				Options(
					utils.MergeMaps(communication.
						Assistant().AssistantProviderModel.
						GetOptions(),
						communication.
							GetOptions()), nil,
				),
			executor.toolExecutor.GetFunctionDefinitions(),
			map[string]string{
				"assistant_id":                fmt.Sprintf("%d", communication.Assistant().Id),
				"assistant_provider_model_id": fmt.Sprintf("%d", communication.Assistant().AssistantProviderModel.Id),
			},
			append(append(
				executor.
					inputBuilder.
					Message(
						communication.
							Assistant().AssistantProviderModel.
							Template.
							GetTextChatCompleteTemplate().
							Prompt,
						utils.MergeMaps(
							executor.inputBuilder.PromptArguments(
								communication.
									Assistant().AssistantProviderModel.
									Template.
									GetTextChatCompleteTemplate().
									Variables,
							),
							communication.
								GetArgs()),
					),
				histories...), in.ToProto())...,
		)

	res, err := communication.
		IntegrationCaller().
		StreamChat(
			ctx,
			communication.
				Auth(),
			communication.
				Assistant().AssistantProviderModel.
				ModelProviderName,
			request)
	if err != nil {
		executor.logger.Errorf("error while streaming chat request: %v", err)
		return err
	}
	for {
		msg, err := res.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				executor.logger.Benchmark("executor.chat", time.Since(start))
				executor.history = append(
					executor.history,
					in.ToProto(),
					output,
				)
				executor.llm(
					messageid,
					communication,
					in,
					types.ToMessage(output),
					types.ToMetrics(metrics))

				//tool call resolve
				toolCalls := output.GetToolCalls()
				if len(toolCalls) > 0 {
					toolExecution := executor.toolExecutor.ExecuteAll(
						ctx,
						messageid,
						toolCalls,
						communication,
					)
					return executor.chat(
						ctx,
						messageid,
						communication,
						&types.Message{Contents: toolExecution, Role: "tool"},
						append(histories, in.ToProto(), output)...,
					)
				}
				communication.OnGenerationComplete(
					ctx,
					messageid,
					types.ToMessage(output).WithMetadata(in.Meta),
					types.ToMetrics(metrics),
				)
				return nil
			}
			return err
		}
		metrics = msg.GetMetrics()
		output = msg.GetData()
		if output != nil && metrics == nil && len(output.GetContents()) > 0 {
			communication.OnGeneration(
				ctx,
				messageid,
				types.ToMessage(msg.GetData()).WithMetadata(in.Meta),
			)
		}

	}
}

func (executor *modelAssistantExecutor) llm(
	messageid string,
	communication internal_adapter_requests.Communication,
	in, out *types.Message,
	metrics []*types.Metric) error {
	utils.Go(context.Background(), func() {
		communication.
			CreateConversationMessageLog(
				messageid, in, out, metrics,
			)
	})
	return nil
}

func (executor *modelAssistantExecutor) Talk(
	ctx context.Context,
	messageid string,
	msg *types.Message,
	communication internal_adapter_requests.Communication) error {
	ctx, span, _ := communication.Tracer().StartSpan(ctx,
		utils.AssistantAgentTextGenerationStage,
		internal_adapter_telemetry.MessageKV(messageid))
	defer span.EndSpan(ctx, utils.AssistantAgentTextGenerationStage)
	return executor.chat(
		ctx,
		messageid,
		communication,
		msg,
		executor.history...)

}

func (a *modelAssistantExecutor) Close(
	ctx context.Context,
	communication internal_adapter_requests.Communication,
) error {
	return nil
}
