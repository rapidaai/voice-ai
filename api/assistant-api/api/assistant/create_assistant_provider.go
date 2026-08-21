// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentkit "github.com/rapidaai/api/assistant-api/internal/llm/agentkit"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

// CreateAssistantProviderModel implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistantProvider(ctx context.Context,
	iRequest *assistant_api.CreateAssistantProviderRequest) (*assistant_api.GetAssistantProviderResponse, error) {
	auth, authErr := types.Authorize(ctx)
	if authErr != nil {
		return nil, status.Error(codes.Unauthenticated, authErr.Error())
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if scopeErr != nil {
		return nil, status.Error(codes.PermissionDenied, scopeErr.Error())
	}
	assistant, err := assistantApi.assistantService.Get(ctx,
		iAuth,
		iRequest.GetAssistantId(), nil, internal_services.NewDefaultGetAssistantOption())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantProviderResponse](
			err,
			"Unable to identify assistant version, please try again later",
		)
	}

	prd := iRequest.GetAssistantProvider()
	switch provider := prd.(type) {
	case *assistant_api.CreateAssistantProviderRequest_Model:
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			protojson.Format(provider.Model.GetTemplate()),
			provider.Model.GetModelProviderName(),
			provider.Model.GetAssistantModelOptions(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderModel{}
		err = utils.Cast(providerModel, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderModel](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderModel{
				AssistantProviderModel: aProviderModel,
			})
	case *assistant_api.CreateAssistantProviderRequest_Agentkit:
		var transportSecurity *string
		var tlsVerification *string
		var tlsServerName *string
		var connectTimeoutMs *uint32
		var keepaliveTimeMs *uint32
		var keepaliveTimeoutMs *uint32
		var maxRecvMessageBytes *uint32
		var maxSendMessageBytes *uint32
		if validator.OneOf(provider.Agentkit.GetCertificate(), agentkit.CertificateInsecure, agentkit.CertificateSkipVerify) {
			return &assistant_api.GetAssistantProviderResponse{
				Code:    pkg_errors.CreateAssistantInvalidAgentKitCertificate.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitCertificate.Code),
					ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitCertificate.Error,
					HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitCertificate.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitCertificate.Error)
		}
		if provider.Agentkit.GetTransportSecurity() != "" {
			transportSecurity = utils.Ptr(provider.Agentkit.GetTransportSecurity())
			if !validator.OneOf(*transportSecurity, agentkit.TransportSecurityTLS, agentkit.TransportSecurityPlaintext) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitTransport.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitTransport.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitTransport.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitTransport.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitTransport.Error)
			}
		}
		if provider.Agentkit.GetTlsVerification() != "" {
			tlsVerification = utils.Ptr(provider.Agentkit.GetTlsVerification())
			if !validator.OneOf(*tlsVerification, agentkit.TLSVerificationVerify, agentkit.TLSVerificationSkipVerify) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.Error)
			}
		}
		if provider.Agentkit.GetTlsServerName() != "" {
			tlsServerName = utils.Ptr(provider.Agentkit.GetTlsServerName())
		}
		if provider.Agentkit.GetConnectTimeoutMs() != 0 {
			connectTimeoutMs = utils.Ptr(provider.Agentkit.GetConnectTimeoutMs())
			if !validator.Between(int(*connectTimeoutMs), agentkit.MinConnectTimeoutMs, agentkit.MaxConnectTimeoutMs) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.Error)
			}
		}
		if provider.Agentkit.GetKeepaliveTimeMs() != 0 {
			keepaliveTimeMs = utils.Ptr(provider.Agentkit.GetKeepaliveTimeMs())
			if !validator.Between(int(*keepaliveTimeMs), agentkit.MinKeepaliveTimeMs, agentkit.MaxKeepaliveTimeMs) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.Error)
			}
		}
		if provider.Agentkit.GetKeepaliveTimeoutMs() != 0 {
			keepaliveTimeoutMs = utils.Ptr(provider.Agentkit.GetKeepaliveTimeoutMs())
			if !validator.Between(int(*keepaliveTimeoutMs), agentkit.MinKeepaliveTimeoutMs, agentkit.MaxKeepaliveTimeoutMs) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.Error)
			}
		}
		if provider.Agentkit.GetMaxRecvMessageBytes() != 0 {
			maxRecvMessageBytes = utils.Ptr(provider.Agentkit.GetMaxRecvMessageBytes())
			if !validator.Between(int(*maxRecvMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.Error)
			}
		}
		if provider.Agentkit.GetMaxSendMessageBytes() != 0 {
			maxSendMessageBytes = utils.Ptr(provider.Agentkit.GetMaxSendMessageBytes())
			if !validator.Between(int(*maxSendMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				return &assistant_api.GetAssistantProviderResponse{
					Code:    pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.HTTPStatusCodeInt32(),
					Success: false,
					Error: &assistant_api.Error{
						ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.Code),
						ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.Error,
						HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.ErrorMessage,
					},
				}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.Error)
			}
		}
		agentKitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			provider.Agentkit.GetAgentKitUrl(),
			provider.Agentkit.GetCertificate(),
			provider.Agentkit.GetMetadata(),
			transportSecurity,
			tlsVerification,
			tlsServerName,
			connectTimeoutMs,
			keepaliveTimeMs,
			keepaliveTimeoutMs,
			maxRecvMessageBytes,
			maxSendMessageBytes,
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderAgentkit{}
		err = utils.Cast(agentKitProvider, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderAgentkit](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderAgentkit{
				AssistantProviderAgentkit: aProviderModel,
			})
	case *assistant_api.CreateAssistantProviderRequest_Websocket:
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			provider.Websocket.GetWebsocketUrl(),
			provider.Websocket.GetHeaders(),
			provider.Websocket.GetConnectionParameters(),
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider model, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderWebsocket{}
		err = utils.Cast(websocketProvider, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderWebsocket](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderWebsocket{
				AssistantProviderWebsocket: aProviderModel,
			})
	case *assistant_api.CreateAssistantProviderRequest_Agentflow:
		definition := map[string]interface{}{}
		if provider.Agentflow.GetDefinition() != nil {
			definition = provider.Agentflow.GetDefinition().AsMap()
		}
		agentflowValidationError := ""
		if !validator.NotBlank(provider.Agentflow.GetSchemaVersion()) {
			agentflowValidationError = "agentflow schemaVersion is required"
		} else if len(definition) == 0 {
			agentflowValidationError = "agentflow definition is required"
		} else {
			entryNodeID, ok := definition["entryNodeId"].(string)
			if !ok || !validator.NotBlank(entryNodeID) {
				agentflowValidationError = "agentflow definition entryNodeId is required"
			}
			nodes, ok := definition["nodes"].([]interface{})
			if agentflowValidationError == "" && !ok {
				agentflowValidationError = "agentflow definition nodes must be an array"
			}
			edges, ok := definition["edges"].([]interface{})
			if agentflowValidationError == "" && !ok {
				agentflowValidationError = "agentflow definition edges must be an array"
			}
			if agentflowValidationError == "" {
				nodeIDs := make(map[string]struct{}, len(nodes))
				for _, node := range nodes {
					nodeMap, ok := node.(map[string]interface{})
					if !ok {
						agentflowValidationError = "agentflow definition node must be an object"
						break
					}
					nodeID, ok := nodeMap["id"].(string)
					if !ok || !validator.NotBlank(nodeID) {
						agentflowValidationError = "agentflow definition node id is required"
						break
					}
					nodeIDs[nodeID] = struct{}{}
				}
				if agentflowValidationError == "" {
					if _, ok := nodeIDs[entryNodeID]; !ok {
						agentflowValidationError = "agentflow entryNodeId does not match a node"
					}
				}
				for _, edge := range edges {
					if agentflowValidationError != "" {
						break
					}
					edgeMap, ok := edge.(map[string]interface{})
					if !ok {
						agentflowValidationError = "agentflow definition edge must be an object"
						break
					}
					source, ok := edgeMap["source"].(string)
					if !ok || !validator.NotBlank(source) {
						agentflowValidationError = "agentflow edge source is required"
						break
					}
					target, ok := edgeMap["target"].(string)
					if !ok || !validator.NotBlank(target) {
						agentflowValidationError = "agentflow edge target is required"
						break
					}
					if _, ok := nodeIDs[source]; !ok {
						agentflowValidationError = "agentflow edge source does not match a node"
						break
					}
					if _, ok := nodeIDs[target]; !ok {
						agentflowValidationError = "agentflow edge target does not match a node"
						break
					}
				}
			}
		}
		if agentflowValidationError != "" {
			return &assistant_api.GetAssistantProviderResponse{
				Code:    pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidRequest.Code),
					ErrorMessage: pkg_errors.CreateAssistantInvalidRequest.Error,
					HumanMessage: agentflowValidationError,
				},
			}, errors.New(agentflowValidationError)
		}
		agentflowProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentflow(
			ctx,
			iAuth,
			assistant.Id,
			iRequest.GetDescription(),
			provider.Agentflow.GetSchemaVersion(),
			definition,
		)
		if err != nil {
			return utils.Error[assistant_api.GetAssistantProviderResponse](
				err,
				"Unable to create assistant provider agentflow, please check the argument and try again.",
			)
		}
		aProviderModel := &assistant_api.AssistantProviderAgentflow{}
		err = utils.Cast(agentflowProvider, aProviderModel)
		if err != nil {
			assistantApi.logger.Errorf("unable to cast the assistant provider agentflow to the response object")
		}
		return utils.Success[
			assistant_api.GetAssistantProviderResponse,
			*assistant_api.
				GetAssistantProviderResponse_AssistantProviderAgentflow](
			&assistant_api.GetAssistantProviderResponse_AssistantProviderAgentflow{
				AssistantProviderAgentflow: aProviderModel,
			})
	}
	return utils.Error[assistant_api.GetAssistantProviderResponse](
		fmt.Errorf("illegal request for creating new assistant provider"),
		"illegal request for creating new assistant provider",
	)
}
