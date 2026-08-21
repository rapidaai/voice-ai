// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentkit "github.com/rapidaai/api/assistant-api/internal/llm/agentkit"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantApi) CreateAssistantRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.CreateAssistantUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !hasUserProjectCapability(auth) {
		c.JSON(pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	var createAssistantRequest openapi.CreateAssistantRequest
	if err := c.ShouldBindJSON(&createAssistantRequest); err != nil {

		c.JSON(pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidRequest.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidRequest.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidRequest.ErrorMessage),
			},
		})
		return
	}

	if !validator.NotBlank(createAssistantRequest.Name) {
		c.JSON(pkg_errors.CreateAssistantMissingName.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingName.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingName.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingName.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingName.ErrorMessage),
			},
		})
		return
	}

	assistantProviderRequest := createAssistantRequest.AssistantProvider
	providerDescription := ""
	if validator.NonNil(assistantProviderRequest.Description) {
		providerDescription = *assistantProviderRequest.Description
	}
	agentkitCertificate := ""
	var agentkitTransportSecurity *string
	var agentkitTLSVerification *string
	var agentkitTLSServerName *string
	var agentkitConnectTimeoutMs *uint32
	var agentkitKeepaliveTimeMs *uint32
	var agentkitKeepaliveTimeoutMs *uint32
	var agentkitMaxRecvMessageBytes *uint32
	var agentkitMaxSendMessageBytes *uint32
	hasModelProvider := validator.NonNil(assistantProviderRequest.Model)
	hasAgentKitProvider := validator.NonNil(assistantProviderRequest.Agentkit)
	hasWebsocketProvider := validator.NonNil(assistantProviderRequest.Websocket)
	hasAgentflowProvider := validator.NonNil(assistantProviderRequest.Agentflow)
	providerCount := 0
	for _, hasProvider := range []bool{hasModelProvider, hasAgentKitProvider, hasWebsocketProvider, hasAgentflowProvider} {
		if hasProvider {
			providerCount++
		}
	}
	if providerCount == 0 {
		c.JSON(pkg_errors.CreateAssistantMissingProvider.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingProvider.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingProvider.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingProvider.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingProvider.ErrorMessage),
			},
		})
		return
	}
	if providerCount > 1 {
		c.JSON(pkg_errors.CreateAssistantInvalidProvider.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidProvider.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidProvider.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidProvider.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidProvider.ErrorMessage),
			},
		})
		return
	}

	if hasModelProvider && !validator.NotBlank(assistantProviderRequest.Model.ModelProviderName) {
		c.JSON(pkg_errors.CreateAssistantMissingModelProviderName.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingModelProviderName.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingModelProviderName.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingModelProviderName.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingModelProviderName.ErrorMessage),
			},
		})
		return
	}
	if hasAgentKitProvider && !validator.NotBlank(assistantProviderRequest.Agentkit.AgentKitUrl) {
		c.JSON(pkg_errors.CreateAssistantMissingAgentKitURL.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingAgentKitURL.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingAgentKitURL.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingAgentKitURL.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingAgentKitURL.ErrorMessage),
			},
		})
		return
	}
	if hasAgentKitProvider {
		agentkitProviderRequest := assistantProviderRequest.Agentkit
		if validator.NonNil(agentkitProviderRequest.Certificate) {
			agentkitCertificate = *agentkitProviderRequest.Certificate
		}
		if validator.OneOf(agentkitCertificate, agentkit.CertificateInsecure, agentkit.CertificateSkipVerify) {
			c.JSON(pkg_errors.CreateAssistantInvalidAgentKitCertificate.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitCertificate.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitCertificate.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitCertificate.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitCertificate.ErrorMessage),
				},
			})
			return
		}
		if validator.NonNil(agentkitProviderRequest.TransportSecurity) {
			agentkitTransportSecurity = agentkitProviderRequest.TransportSecurity
			if !validator.OneOf(*agentkitTransportSecurity, agentkit.TransportSecurityTLS, agentkit.TransportSecurityPlaintext) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitTransport.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTransport.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitTransport.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTransport.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTransport.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.TlsVerification) {
			agentkitTLSVerification = agentkitProviderRequest.TlsVerification
			if !validator.OneOf(*agentkitTLSVerification, agentkit.TLSVerificationVerify, agentkit.TLSVerificationSkipVerify) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitTLSVerification.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.TlsServerName) {
			agentkitTLSServerName = agentkitProviderRequest.TlsServerName
		}
		if validator.NonNil(agentkitProviderRequest.ConnectTimeoutMs) {
			agentkitConnectTimeoutMs = agentkitProviderRequest.ConnectTimeoutMs
			if !validator.Between(int(*agentkitConnectTimeoutMs), agentkit.MinConnectTimeoutMs, agentkit.MaxConnectTimeoutMs) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitConnectTimeout.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.KeepaliveTimeMs) {
			agentkitKeepaliveTimeMs = agentkitProviderRequest.KeepaliveTimeMs
			if !validator.Between(int(*agentkitKeepaliveTimeMs), agentkit.MinKeepaliveTimeMs, agentkit.MaxKeepaliveTimeMs) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTime.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.KeepaliveTimeoutMs) {
			agentkitKeepaliveTimeoutMs = agentkitProviderRequest.KeepaliveTimeoutMs
			if !validator.Between(int(*agentkitKeepaliveTimeoutMs), agentkit.MinKeepaliveTimeoutMs, agentkit.MaxKeepaliveTimeoutMs) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitKeepaliveTimeout.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.MaxRecvMessageBytes) {
			agentkitMaxRecvMessageBytes = agentkitProviderRequest.MaxRecvMessageBytes
			if !validator.Between(int(*agentkitMaxRecvMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxRecvMessageBytes.ErrorMessage),
					},
				})
				return
			}
		}
		if validator.NonNil(agentkitProviderRequest.MaxSendMessageBytes) {
			agentkitMaxSendMessageBytes = agentkitProviderRequest.MaxSendMessageBytes
			if !validator.Between(int(*agentkitMaxSendMessageBytes), agentkit.MinMessageBytes, agentkit.MaxMessageBytes) {
				c.JSON(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidAgentKitMaxSendMessageBytes.ErrorMessage),
					},
				})
				return
			}
		}
	}
	if hasWebsocketProvider && !validator.NotBlank(assistantProviderRequest.Websocket.WebsocketUrl) {
		c.JSON(pkg_errors.CreateAssistantMissingWebsocketURL.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantMissingWebsocketURL.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantMissingWebsocketURL.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantMissingWebsocketURL.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantMissingWebsocketURL.ErrorMessage),
			},
		})
		return
	}
	if hasAgentflowProvider {
		agentflowProviderRequest := assistantProviderRequest.Agentflow
		agentflowValidationError := ""
		if !validator.NotBlank(agentflowProviderRequest.SchemaVersion) {
			agentflowValidationError = "agentflow schemaVersion is required"
		} else if len(agentflowProviderRequest.Definition) == 0 {
			agentflowValidationError = "agentflow definition is required"
		} else {
			entryNodeID, ok := agentflowProviderRequest.Definition["entryNodeId"].(string)
			if !ok || !validator.NotBlank(entryNodeID) {
				agentflowValidationError = "agentflow definition entryNodeId is required"
			}
			nodes, ok := agentflowProviderRequest.Definition["nodes"].([]interface{})
			if agentflowValidationError == "" && !ok {
				agentflowValidationError = "agentflow definition nodes must be an array"
			}
			edges, ok := agentflowProviderRequest.Definition["edges"].([]interface{})
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
			c.JSON(pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidRequest.Error),
					HumanMessage: utils.Ptr(agentflowValidationError),
				},
			})
			return
		}
	}

	description := ""
	if validator.NonNil(createAssistantRequest.Description) {
		description = *createAssistantRequest.Description
	}
	visibility := ""
	if validator.NonNil(createAssistantRequest.Visibility) {
		visibility = *createAssistantRequest.Visibility
	}
	source := ""
	if validator.NonNil(createAssistantRequest.Source) {
		source = *createAssistantRequest.Source
	}
	language := ""
	if validator.NonNil(createAssistantRequest.Language) {
		language = *createAssistantRequest.Language
	}
	var sourceIdentifier *uint64
	if validator.NonNil(createAssistantRequest.SourceIdentifier) {
		parsedSourceIdentifier, err := utils.StringToUint64(*createAssistantRequest.SourceIdentifier)
		if err != nil {

			c.JSON(pkg_errors.CreateAssistantInvalidSourceIdentifier.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidSourceIdentifier.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidSourceIdentifier.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidSourceIdentifier.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidSourceIdentifier.ErrorMessage),
				},
			})
			return
		}
		sourceIdentifier = &parsedSourceIdentifier
	}

	assistant, err := assistantApi.assistantService.CreateAssistant(c, auth, createAssistantRequest.Name, description, visibility, source, sourceIdentifier, language)
	if err != nil {

		c.JSON(pkg_errors.CreateAssistantCreateAssistant.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantCreateAssistant.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateAssistant.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateAssistant.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateAssistant.ErrorMessage),
			},
		})
		return
	}

	if hasModelProvider {
		modelProviderRequest := assistantProviderRequest.Model
		template := "{}"
		if validator.NonNil(modelProviderRequest.Template) {
			templateBytes, err := json.Marshal(modelProviderRequest.Template)
			if err != nil {

				c.JSON(pkg_errors.CreateAssistantInvalidProviderTemplate.HTTPStatusCode, openapi.ErrorResponse{
					Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidProviderTemplate.HTTPStatusCodeInt32()),
					Success: utils.Ptr(false),
					Error: &openapi.Error{
						ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidProviderTemplate.CodeString())),
						ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidProviderTemplate.Error),
						HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidProviderTemplate.ErrorMessage),
					},
				})
				return
			}
			template = string(templateBytes)
		}
		modelOptions := []*protos.Metadata{}
		if validator.NonNil(modelProviderRequest.AssistantModelOptions) {
			for _, modelOption := range *modelProviderRequest.AssistantModelOptions {
				key := ""
				if validator.NonNil(modelOption.Key) {
					key = *modelOption.Key
				}
				value := ""
				if validator.NonNil(modelOption.Value) {
					value = *modelOption.Value
				}
				modelOptions = append(modelOptions, &protos.Metadata{Key: key, Value: value})
			}
		}
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			c, auth, assistant.Id, providerDescription, template, modelProviderRequest.ModelProviderName, modelOptions,
		)
		if err != nil {

			c.JSON(pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateProviderModel.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.ErrorMessage),
				},
			})
			return
		}
		if _, err = assistantApi.assistantService.AttachProviderModelToAssistant(c, auth, assistant.Id, type_enums.MODEL, providerModel.Id); err != nil {

			c.JSON(pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantAttachProviderModel.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.ErrorMessage),
				},
			})
			return
		}
	}

	if hasAgentKitProvider {
		agentkitProviderRequest := assistantProviderRequest.Agentkit
		metadata := map[string]string{}
		if validator.NonNil(agentkitProviderRequest.Metadata) {
			metadata = *agentkitProviderRequest.Metadata
		}
		agentkitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			c, auth, assistant.Id, providerDescription, agentkitProviderRequest.AgentKitUrl, agentkitCertificate, metadata,
			agentkitTransportSecurity, agentkitTLSVerification, agentkitTLSServerName, agentkitConnectTimeoutMs, agentkitKeepaliveTimeMs,
			agentkitKeepaliveTimeoutMs, agentkitMaxRecvMessageBytes, agentkitMaxSendMessageBytes,
		)
		if err != nil {

			c.JSON(pkg_errors.CreateAssistantCreateProviderAgentkit.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantCreateProviderAgentkit.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateProviderAgentkit.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderAgentkit.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderAgentkit.ErrorMessage),
				},
			})
			return
		}
		if _, err = assistantApi.assistantService.AttachProviderModelToAssistant(c, auth, assistant.Id, type_enums.AGENTKIT, agentkitProvider.Id); err != nil {

			c.JSON(pkg_errors.CreateAssistantAttachProviderAgentkit.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantAttachProviderAgentkit.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantAttachProviderAgentkit.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderAgentkit.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderAgentkit.ErrorMessage),
				},
			})
			return
		}
	}

	if hasWebsocketProvider {
		websocketProviderRequest := assistantProviderRequest.Websocket
		headers := map[string]string{}
		if validator.NonNil(websocketProviderRequest.Headers) {
			headers = *websocketProviderRequest.Headers
		}
		connectionParameters := map[string]string{}
		if validator.NonNil(websocketProviderRequest.ConnectionParameters) {
			connectionParameters = *websocketProviderRequest.ConnectionParameters
		}
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			c, auth, assistant.Id, providerDescription, websocketProviderRequest.WebsocketUrl, headers, connectionParameters,
		)
		if err != nil {

			c.JSON(pkg_errors.CreateAssistantCreateProviderWebsocket.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantCreateProviderWebsocket.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateProviderWebsocket.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderWebsocket.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderWebsocket.ErrorMessage),
				},
			})
			return
		}
		if _, err = assistantApi.assistantService.AttachProviderModelToAssistant(c, auth, assistant.Id, type_enums.WEBSOCKET, websocketProvider.Id); err != nil {

			c.JSON(pkg_errors.CreateAssistantAttachProviderWebsocket.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantAttachProviderWebsocket.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantAttachProviderWebsocket.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderWebsocket.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderWebsocket.ErrorMessage),
				},
			})
			return
		}
	}

	if hasAgentflowProvider {
		agentflowProviderRequest := assistantProviderRequest.Agentflow
		agentflowProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentflow(
			c, auth, assistant.Id, providerDescription, agentflowProviderRequest.SchemaVersion, agentflowProviderRequest.Definition,
		)
		if err != nil {

			c.JSON(pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateProviderModel.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateProviderModel.ErrorMessage),
				},
			})
			return
		}
		if _, err = assistantApi.assistantService.AttachProviderModelToAssistant(c, auth, assistant.Id, type_enums.AGENTFLOW, agentflowProvider.Id); err != nil {

			c.JSON(pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantAttachProviderModel.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.Error),
					HumanMessage: utils.Ptr(pkg_errors.CreateAssistantAttachProviderModel.ErrorMessage),
				},
			})
			return
		}
	}

	if validator.NonNil(createAssistantRequest.AssistantTools) {
		for _, assistantToolRequest := range *createAssistantRequest.AssistantTools {
			fields := map[string]interface{}{}
			if validator.NonNil(assistantToolRequest.Fields) {
				fields = *assistantToolRequest.Fields
			}
			toolOptions := []*protos.Metadata{}
			if validator.NonNil(assistantToolRequest.ExecutionOptions) {
				for _, toolOption := range *assistantToolRequest.ExecutionOptions {
					key := ""
					if validator.NonNil(toolOption.Key) {
						key = *toolOption.Key
					}
					value := ""
					if validator.NonNil(toolOption.Value) {
						value = *toolOption.Value
					}
					toolOptions = append(toolOptions, &protos.Metadata{Key: key, Value: value})
				}
			}
			toolName := ""
			if validator.NonNil(assistantToolRequest.Name) {
				toolName = *assistantToolRequest.Name
			}
			executionMethod := ""
			if validator.NonNil(assistantToolRequest.ExecutionMethod) {
				executionMethod = *assistantToolRequest.ExecutionMethod
			}
			if _, err := assistantApi.assistantToolService.Create(
				c, auth, assistant.Id, toolName, assistantToolRequest.Description, fields, executionMethod, toolOptions,
			); err != nil {
				assistantApi.logger.Errorf("%s with error %+v", pkg_errors.CreateAssistantCreateTools.ErrorMessage, err)
			}
		}
	}

	if validator.NonNil(createAssistantRequest.AssistantKnowledges) {
		for _, assistantKnowledgeRequest := range *createAssistantRequest.AssistantKnowledges {
			knowledgeID := uint64(0)
			if validator.NonNil(assistantKnowledgeRequest.KnowledgeId) && validator.NotBlank(*assistantKnowledgeRequest.KnowledgeId) {
				parsedKnowledgeID, err := utils.StringToUint64(*assistantKnowledgeRequest.KnowledgeId)
				if err != nil {

					c.JSON(pkg_errors.CreateAssistantInvalidKnowledgeID.HTTPStatusCode, openapi.ErrorResponse{
						Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidKnowledgeID.HTTPStatusCodeInt32()),
						Success: utils.Ptr(false),
						Error: &openapi.Error{
							ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidKnowledgeID.CodeString())),
							ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidKnowledgeID.Error),
							HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidKnowledgeID.ErrorMessage),
						},
					})
					return
				}
				knowledgeID = parsedKnowledgeID
			}
			rerankerModelProviderID := uint64(0)
			if validator.NonNil(assistantKnowledgeRequest.RerankerModelProviderId) && validator.NotBlank(*assistantKnowledgeRequest.RerankerModelProviderId) {
				parsedRerankerModelProviderID, err := utils.StringToUint64(*assistantKnowledgeRequest.RerankerModelProviderId)
				if err != nil {

					c.JSON(pkg_errors.CreateAssistantInvalidRerankerModelProviderID.HTTPStatusCode, openapi.ErrorResponse{
						Code:    utils.Ptr(pkg_errors.CreateAssistantInvalidRerankerModelProviderID.HTTPStatusCodeInt32()),
						Success: utils.Ptr(false),
						Error: &openapi.Error{
							ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantInvalidRerankerModelProviderID.CodeString())),
							ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidRerankerModelProviderID.Error),
							HumanMessage: utils.Ptr(pkg_errors.CreateAssistantInvalidRerankerModelProviderID.ErrorMessage),
						},
					})
					return
				}
				rerankerModelProviderID = parsedRerankerModelProviderID
			}
			knowledgeOptions := []*protos.Metadata{}
			if validator.NonNil(assistantKnowledgeRequest.AssistantKnowledgeRerankerOptions) {
				for _, knowledgeOption := range *assistantKnowledgeRequest.AssistantKnowledgeRerankerOptions {
					key := ""
					if validator.NonNil(knowledgeOption.Key) {
						key = *knowledgeOption.Key
					}
					value := ""
					if validator.NonNil(knowledgeOption.Value) {
						value = *knowledgeOption.Value
					}
					knowledgeOptions = append(knowledgeOptions, &protos.Metadata{Key: key, Value: value})
				}
			}
			rerankerEnable := false
			if validator.NonNil(assistantKnowledgeRequest.RerankerEnable) {
				rerankerEnable = *assistantKnowledgeRequest.RerankerEnable
			}
			scoreThreshold := float32(0)
			if validator.NonNil(assistantKnowledgeRequest.ScoreThreshold) {
				scoreThreshold = *assistantKnowledgeRequest.ScoreThreshold
			}
			topK := uint32(0)
			if validator.NonNil(assistantKnowledgeRequest.TopK) {
				topK = *assistantKnowledgeRequest.TopK
			}
			retrievalMethod := ""
			if validator.NonNil(assistantKnowledgeRequest.RetrievalMethod) {
				retrievalMethod = *assistantKnowledgeRequest.RetrievalMethod
			}
			if _, err := assistantApi.assistantKnowledgeService.Create(
				c,
				auth,
				assistant.Id,
				knowledgeID,
				gorm_types.RetrievalMethod(retrievalMethod),
				rerankerEnable,
				scoreThreshold,
				topK,
				&rerankerModelProviderID,
				assistantKnowledgeRequest.RerankerModelProviderName,
				knowledgeOptions,
			); err != nil {
				assistantApi.logger.Errorf("%s with error %+v", pkg_errors.CreateAssistantCreateKnowledge.ErrorMessage, err)
			}
		}
	}

	tags := []string{}
	if validator.NonNil(createAssistantRequest.Tags) {
		tags = *createAssistantRequest.Tags
	}
	if _, err = assistantApi.assistantService.CreateOrUpdateAssistantTag(c, auth, assistant.Id, tags); err != nil {

		c.JSON(pkg_errors.CreateAssistantCreateTags.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.CreateAssistantCreateTags.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.CreateAssistantCreateTags.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.CreateAssistantCreateTags.Error),
				HumanMessage: utils.Ptr(pkg_errors.CreateAssistantCreateTags.ErrorMessage),
			},
		})
		return
	}

	c.JSON(http.StatusOK, openapi.GetAssistantResponse{
		Code:    utils.Ptr(int32(200)),
		Success: utils.Ptr(true),
		Data: &openapi.Assistant{
			Id:          utils.Ptr(openapi.Uint64String(strconv.FormatUint(assistant.Id, 10))),
			Name:        utils.Ptr(assistant.Name),
			Description: utils.Ptr(assistant.Description),
			Visibility:  utils.Ptr(assistant.Visibility),
		},
	})
}
