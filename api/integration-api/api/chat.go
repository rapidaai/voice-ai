// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"errors"
	"io"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internal_caller_factory "github.com/rapidaai/api/integration-api/internal/caller"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func (iApi *integrationApi) Chat(
	c context.Context,
	irRequest *protos.ChatRequest,
	tag string,
) (*protos.ChatResponse, error) {
	tag = strings.ToLower(tag)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	projectContext, authErr := requireProjectContext(iAuth)
	if !isAuthenticated || authErr != nil {
		return utils.Error[protos.ChatResponse](
			errors.New("unauthenticated request for chat"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	if irRequest.AdditionalData == nil {
		irRequest.AdditionalData = map[string]string{}
	}

	caller, err := internal_caller_factory.GetChat(
		iApi.logger,
		tag,
		irRequest.GetCredential(),
		irRequest.GetConnectionOptions(),
	)
	if err != nil {
		return utils.Error[protos.ChatResponse](err, err.Error())
	}

	irRequest.AdditionalData["provider_name"] = tag
	model, ok := irRequest.ModelParameters["model.name"]
	if ok {
		mdl, err := utils.AnyToString(model)
		if err == nil {
			irRequest.AdditionalData["model_name"] = mdl
		}
	}

	modelID, ok := irRequest.ModelParameters["model.id"]
	if ok {
		mdlID, err := utils.AnyToString(modelID)
		if err == nil {
			irRequest.AdditionalData["model_id"] = mdlID
		}
	}
	source, ok := utils.GetClientSource(c)
	if ok {
		irRequest.AdditionalData["source"] = source.Get()
	}

	clientEnv, ok := utils.GetClientEnvironment(c)
	if ok {
		irRequest.AdditionalData["env"] = clientEnv.Get()
	}

	clientRegion, ok := utils.GetClientRegion(c)
	if ok {
		irRequest.AdditionalData["region"] = clientRegion.Get()
	}

	requestID := iApi.RequestId()
	completions, metrics, err := caller.ChatComplete(
		c,
		irRequest.GetConversations(),
		internal_callers.NewChatOptions(
			requestID,
			irRequest,
			iApi.PreHook(c, projectContext, irRequest, requestID, tag),
			iApi.PostHook(c, projectContext, irRequest, requestID, tag),
		),
	)
	if err != nil {
		return utils.Error[protos.ChatResponse](err, err.Error())
	}
	return &protos.ChatResponse{
		Code:    200,
		Success: true,
		Data:    completions,
		Metrics: metrics,
	}, nil
}

// StreamChatBidirectionalUnified handles bidirectional streaming chat for the unified provider service.
// The first frame must be configuration; all later chat frames reuse the same caller/client.
func (iApi *integrationApi) StreamChatBidirectionalUnified(
	context context.Context,
	logger commons.Logger,
	stream grpc.BidiStreamingServer[protos.StreamChatRequest, protos.StreamChatResponse],
) error {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(context)
	projectContext, authErr := requireProjectContext(iAuth)
	if !isAuthenticated || authErr != nil {
		iApi.logger.Errorf("unauthenticated request for bidirectional stream chat")
		return status.Error(codes.Unauthenticated, "Please provide valid service credentials to perform invoke.")
	}
	var (
		chatStream                internal_callers.ChatStream
		streamConnectionTransport string
	)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			iApi.logger.Infof("Client closed bidirectional stream for unified provider")
			if chatStream != nil {
				_ = chatStream.Close(stream.Context())
			}
			return nil
		}
		if err != nil {
			iApi.logger.Errorf("Error receiving from bidirectional stream: %v", err)
			return status.Errorf(codes.Internal, "Error receiving chat request from stream: %v", err)
		}
		switch payload := req.GetRequest().(type) {
		case *protos.StreamChatRequest_Configuration:
			streamConnectionTransport = payload.Configuration.GetConnectionOptions()["connection.transport"]
			if streamConnectionTransport == "" {
				streamConnectionTransport = "default"
			}

			chatStream, err = internal_caller_factory.GetChatStream(
				logger,
				payload.Configuration.GetProviderName(),
				payload.Configuration.GetCredential(),
				payload.Configuration.GetConnectionOptions(),
			)
			if err != nil {
				_ = stream.Send(&protos.StreamChatResponse{
					Response: &protos.StreamChatResponse_Configuration{
						Configuration: &protos.StreamChatConfigured{
							ProviderName: payload.Configuration.GetProviderName(),
							Success:      false,
							Error: &protos.Error{
								ErrorCode:    400,
								ErrorMessage: err.Error(),
								HumanMessage: err.Error(),
							},
						},
					},
				})
				continue
			}

			if err = chatStream.Connect(stream.Context(), payload.Configuration); err != nil {
				_ = stream.Send(&protos.StreamChatResponse{
					Response: &protos.StreamChatResponse_Configuration{
						Configuration: &protos.StreamChatConfigured{
							ProviderName: payload.Configuration.GetProviderName(),
							Success:      false,
							Error: &protos.Error{
								ErrorCode:    400,
								ErrorMessage: err.Error(),
								HumanMessage: err.Error(),
							},
						},
					},
				})
				continue
			}

			_ = stream.Send(&protos.StreamChatResponse{
				Response: &protos.StreamChatResponse_Configuration{
					Configuration: &protos.StreamChatConfigured{
						ProviderName: payload.Configuration.GetProviderName(),
						Success:      true,
					},
				},
			})
		case *protos.StreamChatRequest_Chat:
			if chatStream == nil {
				_ = stream.Send(&protos.StreamChatResponse{
					Response: &protos.StreamChatResponse_Chat{
						Chat: &protos.StreamChatOutput{
							RequestId: payload.Chat.GetRequestId(),
							Error: &protos.Error{
								ErrorCode:    412,
								ErrorMessage: "stream is not configured; send configuration first",
								HumanMessage: "stream is not configured; send configuration first",
							},
						},
					},
				})
				continue
			}

			if payload.Chat.AdditionalData == nil {
				payload.Chat.AdditionalData = map[string]string{}
			}
			payload.Chat.AdditionalData["connection.transport"] = streamConnectionTransport
			payload.Chat.AdditionalData["provider_name"] = payload.Chat.GetProviderName()
			model, ok := payload.Chat.ModelParameters["model.name"]
			if ok {
				mdl, err := utils.AnyToString(model)
				if err == nil {
					payload.Chat.AdditionalData["model_name"] = mdl
				}
			}

			modelID, ok := payload.Chat.ModelParameters["model.id"]
			if ok {
				mdlID, err := utils.AnyToString(modelID)
				if err == nil {
					payload.Chat.AdditionalData["model_id"] = mdlID
				}
			}

			source, ok := utils.GetClientSource(context)
			if ok {
				payload.Chat.AdditionalData["source"] = source.Get()
			}

			clientEnv, ok := utils.GetClientEnvironment(context)
			if ok {
				payload.Chat.AdditionalData["env"] = clientEnv.Get()
			}

			clientRegion, ok := utils.GetClientRegion(context)
			if ok {
				payload.Chat.AdditionalData["region"] = clientRegion.Get()
			}

			auditRequest := &protos.ChatRequest{
				Credential:      chatStream.GetCredential(),
				RequestId:       payload.Chat.GetRequestId(),
				Conversations:   payload.Chat.GetConversations(),
				AdditionalData:  payload.Chat.GetAdditionalData(),
				ModelParameters: payload.Chat.GetModelParameters(),
				ToolDefinitions: payload.Chat.GetToolDefinitions(),
				ProviderName:    payload.Chat.GetProviderName(),
			}
			requestID := iApi.RequestId()

			err = chatStream.Chat(
				stream.Context(),
				payload.Chat.GetConversations(),
				internal_callers.NewChatStreamOptions(
					requestID,
					payload.Chat,
					iApi.PreHook(stream.Context(), projectContext, auditRequest, requestID, payload.Chat.GetProviderName()),
					iApi.PostHook(stream.Context(), projectContext, auditRequest, requestID, payload.Chat.GetProviderName()),
				),
				func(rID string, content *protos.Message) error {
					return stream.Send(&protos.StreamChatResponse{
						Response: &protos.StreamChatResponse_Chat{
							Chat: &protos.StreamChatOutput{
								RequestId: rID,
								Data:      content,
							},
						},
					})
				},
				func(rID string, content *protos.Message, mtx []*protos.Metric) error {
					return stream.Send(&protos.StreamChatResponse{
						Response: &protos.StreamChatResponse_Chat{
							Chat: &protos.StreamChatOutput{
								RequestId: rID,
								Data:      content,
								Metrics:   mtx,
							},
						},
					})
				},
				func(rID string, err error) {
					_ = stream.Send(&protos.StreamChatResponse{
						Response: &protos.StreamChatResponse_Chat{
							Chat: &protos.StreamChatOutput{
								RequestId: rID,
								Error: &protos.Error{
									ErrorCode:    400,
									ErrorMessage: err.Error(),
									HumanMessage: err.Error(),
								},
							},
						},
					})
				},
			)

			if err != nil {
				iApi.logger.Warnf("Error processing chat request in bidirectional stream: %v", err)
				_ = stream.Send(&protos.StreamChatResponse{
					Response: &protos.StreamChatResponse_Chat{
						Chat: &protos.StreamChatOutput{
							RequestId: payload.Chat.GetRequestId(),
							Error: &protos.Error{
								ErrorCode:    500,
								ErrorMessage: err.Error(),
								HumanMessage: "Internal server error processing your request",
							},
						},
					},
				})
			}
		case *protos.StreamChatRequest_Close:
			if chatStream != nil {
				_ = chatStream.Close(stream.Context())
			}
			_ = stream.Send(&protos.StreamChatResponse{
				Response: &protos.StreamChatResponse_Close{
					Close: &protos.StreamChatClose{
						Reason: payload.Close.GetReason(),
					},
				},
			})
			return nil
		default:
			_ = stream.Send(&protos.StreamChatResponse{
				Response: &protos.StreamChatResponse_Chat{
					Chat: &protos.StreamChatOutput{
						Error: &protos.Error{
							ErrorCode:    400,
							ErrorMessage: "invalid stream request; expected configuration, chat, or close",
							HumanMessage: "invalid stream request; expected configuration, chat, or close",
						},
					},
				},
			})
		}
	}
}
