// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type agentkitExecutor struct {
	logger   commons.Logger
	agentKit protos.AgentTalkClient
	talker   grpc.BidiStreamingClient[protos.AgentTalkRequest, protos.AgentTalkResponse]
}

// Init implements AssistantExecutor.
func (executor *agentkitExecutor) Initialize(ctx context.Context, communication internal_adapter_requests.Communication) error {
	ctx, span, _ := communication.Tracer().StartSpan(
		ctx,
		utils.AssistantAgentConnectStage,
		internal_adapter_telemetry.KV{
			K: "executor",
			V: internal_adapter_telemetry.StringValue(executor.Name()),
		},
	)
	defer span.EndSpan(ctx, utils.AssistantAgentConnectStage)
	g, ctx := errgroup.WithContext(ctx)

	providerDefinition := communication.
		Assistant().
		AssistantProviderAgentkit

	g.Go(func() error {
		// Prepare HTTP headers
		// providerDefinition.Certificate
		// providerDefinition.Metadata

		grpcOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(math.MaxInt64),
				grpc.MaxCallSendMsgSize(math.MaxInt64),
			),
		}
		conn, err := grpc.NewClient(providerDefinition.Url,
			grpcOpts...)

		span.AddAttributes(ctx, internal_adapter_telemetry.KV{
			K: "url",
			V: internal_adapter_telemetry.StringValue(providerDefinition.Url),
		})
		if err != nil {
			span.AddAttributes(ctx, internal_adapter_telemetry.KV{
				K: "error",
				V: internal_adapter_telemetry.StringValue(fmt.Sprintf("Error while connect agentkit to provided host and port: %v", err)),
			})
			return fmt.Errorf("failed to connect agentkit : %w", err)
		}
		executor.agentKit = protos.NewAgentTalkClient(conn)
		talker, err := executor.agentKit.Talk(context.Background())
		if err != nil {
			span.AddAttributes(ctx, internal_adapter_telemetry.KV{
				K: "error",
				V: internal_adapter_telemetry.StringValue(fmt.Sprintf("Error while connect agentkit to provided host and port: %v", err)),
			})
			return fmt.Errorf("failed to connect agentkit : %w", err)
		}
		executor.talker = talker
		return err

	})
	if err := g.Wait(); err != nil {
		executor.logger.Errorf("Error during initialization of agentkit: %v", err)
		return err
	}

	utils.Go(ctx, func() {
		if err := executor.TalkListener(ctx, communication); err != nil {
			executor.logger.Errorf("Error in TalkListener: %v", err)
		}
	})

	if executor.talker != nil {
		executor.talker.Send(&protos.AgentTalkRequest{
			Request: &protos.AgentTalkRequest_Configuration{
				Configuration: &protos.AssistantConversationConfiguration{
					AssistantConversationId: communication.Conversation().Id,
					Assistant: &protos.AssistantDefinition{
						AssistantId: communication.Assistant().Id,
					},
				},
			},
		})
	}
	return nil
}

// Name implements AssistantExecutor.
func (a *agentkitExecutor) Name() string {
	return "agentkit"
}

func (executor *agentkitExecutor) TalkListener(ctx context.Context, communication internal_adapter_requests.Communication) error {
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		req, err := executor.talker.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			// Log and return unrecoverable errors
			return fmt.Errorf("stream.Recv error: %w", err)
		}
		if req.GetError() != nil {
			executor.logger.Debugf("stream.Recv error: %w", err)
			continue
		}

		if req.GetSuccess() {
			switch msg := req.GetData().(type) {
			case *protos.AgentTalkResponse_Assistant:
				switch in := msg.Assistant.GetMessage().(type) {
				case *protos.AssistantConversationAssistantMessage_Audio:
					continue
				case *protos.AssistantConversationAssistantMessage_Text:
					communication.OnGeneration(
						ctx,
						msg.Assistant.Id,
						&types.Message{
							Id:   msg.Assistant.Id,
							Role: "assistant",
							Contents: []*types.Content{&types.Content{
								ContentType:   commons.TEXT_CONTENT.String(),
								ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
								Content:       []byte(in.Text.Content),
							}},
						},
					)
					if msg.Assistant.GetCompleted() {
						communication.OnGenerationComplete(ctx, msg.Assistant.Id, nil, nil)
					}

				}
			}
		}
	}
	return nil
}

// Talk implements AssistantExecutor.
func (executor *agentkitExecutor) Talk(ctx context.Context, messageid string, msg *types.Message, communcation internal_adapter_requests.Communication) error {
	executor.logger.Debugf("sending communication request")
	if executor.talker != nil {
		executor.talker.Send(&protos.AgentTalkRequest{
			Request: &protos.AgentTalkRequest_Message{
				Message: &protos.AssistantConversationUserMessage{
					Message: &protos.AssistantConversationUserMessage_Text{
						Text: &protos.AssistantConversationMessageTextContent{
							Content: msg.String(),
						},
					},
					Id:        messageid,
					Completed: true,
					Time:      timestamppb.New(time.Now()),
				},
			},
		})
	}
	return nil
}

func (executor *agentkitExecutor) Close(
	ctx context.Context,
	communication internal_adapter_requests.Communication,
) error {
	if executor.talker != nil {
		executor.logger.Debugf("calling disconnect to agentkit")
		// executor.talker.Send(&protos.AgentTalkRequest{
		// 	Request: &protos.AgentTalkRequest_Configuration{
		// 		Configuration: &protos.AssistantConversationConfiguration{
		// 			AssistantConversationId: assistantConversationId,
		// 			Assistant: &protos.AssistantDefinition{
		// 				AssistantId: assistantId,
		// 			},
		// 		},
		// 	},
		// })
	}
	return nil
}

func NewAgentKitAssistantExecutor(
	logger commons.Logger,
) AssistantExecutor {
	return &agentkitExecutor{
		logger: logger,
	}

}
