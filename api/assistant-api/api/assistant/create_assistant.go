// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	agentkit "github.com/rapidaai/api/assistant-api/internal/llm/agentkit"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

// CreateAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistant(ctx context.Context, cer *assistant_api.CreateAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantUnauthenticated.Code),
				ErrorMessage: pkg_errors.CreateAssistantUnauthenticated.Error,
				HumanMessage: pkg_errors.CreateAssistantUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantUnauthenticated.Error)
	}
	if _, authErr := types.RequireUser(iAuth); authErr != nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if _, authErr := types.RequireProject(iAuth); authErr != nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if _, authErr := types.RequireOrganization(iAuth); authErr != nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if !validator.NotBlank(cer.GetName()) {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingName.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingName.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingName.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingName.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingName.Error)
	}
	if cer.GetAssistantProvider() == nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingProvider.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingProvider.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingProvider.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingProvider.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingProvider.Error)
	}
	if cer.GetAssistantProvider().GetAssistantProvider() == nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingProvider.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingProvider.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingProvider.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingProvider.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingProvider.Error)
	}
	if cer.GetAssistantProvider().GetModel() != nil {
		if !validator.NotBlank(cer.GetAssistantProvider().GetModel().GetModelProviderName()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingModelProviderName.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingModelProviderName.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingModelProviderName.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingModelProviderName.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingModelProviderName.Error)
		}
	}
	var agentkitTransportSecurity *string
	var agentkitTLSVerification *string
	var agentkitTLSServerName *string
	var agentkitConnectTimeoutMs *uint32
	var agentkitKeepaliveTimeMs *uint32
	var agentkitKeepaliveTimeoutMs *uint32
	var agentkitMaxRecvMessageBytes *uint32
	var agentkitMaxSendMessageBytes *uint32
	if cer.GetAssistantProvider().GetAgentkit() != nil {
		agentkitProviderRequest := cer.GetAssistantProvider().GetAgentkit()
		if !validator.NotBlank(agentkitProviderRequest.GetAgentKitUrl()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingAgentKitURL.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAgentKitURL.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingAgentKitURL.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingAgentKitURL.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingAgentKitURL.Error)
		}
		if validator.OneOf(agentkitProviderRequest.GetCertificate(), agentkit.CertificateInsecure, agentkit.CertificateSkipVerify) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantInvalidAgentKitCertificate.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidAgentKitCertificate.Code),
					ErrorMessage: pkg_errors.CreateAssistantInvalidAgentKitCertificate.Error,
					HumanMessage: pkg_errors.CreateAssistantInvalidAgentKitCertificate.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantInvalidAgentKitCertificate.Error)
		}
		if agentkitProviderRequest.GetTransportSecurity() != "" {
			agentkitTransportSecurity = utils.Ptr(agentkitProviderRequest.GetTransportSecurity())
			if !validator.OneOf(*agentkitTransportSecurity, agentkit.TransportSecurityTLS, agentkit.TransportSecurityPlaintext) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetTlsVerification() != "" {
			agentkitTLSVerification = utils.Ptr(agentkitProviderRequest.GetTlsVerification())
			if !validator.OneOf(*agentkitTLSVerification, agentkit.TLSVerificationVerify, agentkit.TLSVerificationSkipVerify) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetTlsServerName() != "" {
			agentkitTLSServerName = utils.Ptr(agentkitProviderRequest.GetTlsServerName())
		}
		if agentkitProviderRequest.GetConnectTimeoutMs() != 0 {
			agentkitConnectTimeoutMs = utils.Ptr(agentkitProviderRequest.GetConnectTimeoutMs())
			if !validator.Between(int(*agentkitConnectTimeoutMs), agentkit.MinConnectTimeoutMs, agentkit.MaxConnectTimeoutMs) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetKeepaliveTimeMs() != 0 {
			agentkitKeepaliveTimeMs = utils.Ptr(agentkitProviderRequest.GetKeepaliveTimeMs())
			if !validator.Between(int(*agentkitKeepaliveTimeMs), agentkit.MinKeepaliveTimeMs, agentkit.MaxKeepaliveTimeMs) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetKeepaliveTimeoutMs() != 0 {
			agentkitKeepaliveTimeoutMs = utils.Ptr(agentkitProviderRequest.GetKeepaliveTimeoutMs())
			if !validator.Between(int(*agentkitKeepaliveTimeoutMs), agentkit.MinKeepaliveTimeoutMs, agentkit.MaxKeepaliveTimeoutMs) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetMaxRecvMessageBytes() != 0 {
			agentkitMaxRecvMessageBytes = utils.Ptr(agentkitProviderRequest.GetMaxRecvMessageBytes())
			if !validator.Between(int(*agentkitMaxRecvMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				return &assistant_api.GetAssistantResponse{
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
		if agentkitProviderRequest.GetMaxSendMessageBytes() != 0 {
			agentkitMaxSendMessageBytes = utils.Ptr(agentkitProviderRequest.GetMaxSendMessageBytes())
			if !validator.Between(int(*agentkitMaxSendMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				return &assistant_api.GetAssistantResponse{
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
	}
	if cer.GetAssistantProvider().GetWebsocket() != nil {
		if !validator.NotBlank(cer.GetAssistantProvider().GetWebsocket().GetWebsocketUrl()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingWebsocketURL.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingWebsocketURL.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingWebsocketURL.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingWebsocketURL.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingWebsocketURL.Error)
		}
	}
	if cer.GetAssistantProvider().GetAgentflow() != nil {
		agentflowProviderRequest := cer.GetAssistantProvider().GetAgentflow()
		definition := map[string]interface{}{}
		if agentflowProviderRequest.GetDefinition() != nil {
			definition = agentflowProviderRequest.GetDefinition().AsMap()
		}
		agentflowValidationError := ""
		if !validator.NotBlank(agentflowProviderRequest.GetSchemaVersion()) {
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
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantInvalidRequest.Code),
					ErrorMessage: pkg_errors.CreateAssistantInvalidRequest.Error,
					HumanMessage: agentflowValidationError,
				},
			}, errors.New(agentflowValidationError)
		}
	}

	// creating assistant
	assistant, err := assistantApi.
		assistantService.
		CreateAssistant(
			ctx,
			iAuth,
			cer.GetName(),
			cer.GetDescription(),
			cer.GetVisibility(),
			cer.GetSource(),
			&cer.SourceIdentifier,
			cer.GetLanguage(),
		)
	if err != nil {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantCreateAssistant.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantCreateAssistant.Code),
				ErrorMessage: pkg_errors.CreateAssistantCreateAssistant.Error,
				HumanMessage: pkg_errors.CreateAssistantCreateAssistant.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantCreateAssistant.Error)
	}

	prd := cer.GetAssistantProvider().GetAssistantProvider()
	switch provider := prd.(type) {
	case *assistant_api.CreateAssistantProviderRequest_Model:
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			protojson.Format(provider.Model.GetTemplate()),
			provider.Model.GetModelProviderName(),
			provider.Model.GetAssistantModelOptions(),
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderModel.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.MODEL,
			providerModel.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderModel.Error)
		}

	case *assistant_api.CreateAssistantProviderRequest_Agentkit:
		agentKitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Agentkit.GetAgentKitUrl(),
			provider.Agentkit.GetCertificate(),
			provider.Agentkit.GetMetadata(),
			agentkitTransportSecurity,
			agentkitTLSVerification,
			agentkitTLSServerName,
			agentkitConnectTimeoutMs,
			agentkitKeepaliveTimeMs,
			agentkitKeepaliveTimeoutMs,
			agentkitMaxRecvMessageBytes,
			agentkitMaxSendMessageBytes,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderAgentkit.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderAgentkit.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderAgentkit.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderAgentkit.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderAgentkit.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.AGENTKIT,
			agentKitProvider.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderAgentkit.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderAgentkit.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderAgentkit.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderAgentkit.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderAgentkit.Error)
		}

	case *assistant_api.CreateAssistantProviderRequest_Websocket:
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Websocket.GetWebsocketUrl(),
			provider.Websocket.GetHeaders(),
			provider.Websocket.GetConnectionParameters(),
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderWebsocket.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderWebsocket.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderWebsocket.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderWebsocket.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderWebsocket.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.WEBSOCKET,
			websocketProvider.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderWebsocket.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderWebsocket.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderWebsocket.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderWebsocket.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderWebsocket.Error)
		}

	case *assistant_api.CreateAssistantProviderRequest_Agentflow:
		definition := map[string]interface{}{}
		if provider.Agentflow.GetDefinition() != nil {
			definition = provider.Agentflow.GetDefinition().AsMap()
		}
		agentflowProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentflow(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Agentflow.GetSchemaVersion(),
			definition,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderModel.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.AGENTFLOW,
			agentflowProvider.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderModel.Error)
		}

	}

	for _, tl := range cer.GetAssistantTools() {
		_, err := assistantApi.createAssistantTool(
			ctx,
			iAuth,
			assistant.Id,
			tl)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant tools, please try again later with error %+v", err)
		}
	}

	for _, ak := range cer.GetAssistantKnowledges() {
		_, err := assistantApi.createAssistantKnowledge(
			ctx,
			iAuth,
			assistant.Id,
			ak)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant knowledge, please try again later with error %+v", err)
		}
	}

	_, err = assistantApi.assistantService.CreateOrUpdateAssistantTag(ctx, iAuth, assistant.Id, cer.GetTags())
	if err != nil {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantCreateTags.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantCreateTags.Code),
				ErrorMessage: pkg_errors.CreateAssistantCreateTags.Error,
				HumanMessage: pkg_errors.CreateAssistantCreateTags.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantCreateTags.Error)
	}

	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)
}
