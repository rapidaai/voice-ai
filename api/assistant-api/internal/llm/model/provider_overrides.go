// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_model

import (
	"fmt"
	"strings"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/pkg/utils"
)

const (
	modelOptionCredentialID        = "rapida.credential_id" // #nosec G101, this is a metadata key, not a secret.
	modelOptionConnectionPrefix    = "connection."
	modelOptionConnectionTransport = "connection.transport"
	modelOptionConnectionEndpoint  = "connection.endpoint"
)

func withModelOverrides(providerOptions utils.Option, conversationOptions utils.Option) utils.Option {
	resolved := utils.MergeMaps(providerOptions, conversationOptions)
	if len(conversationOptions) > 0 {
		if value, ok := conversationOptions[internal_options.ModelOptionCredentialID]; ok && value != nil {
			resolved[modelOptionCredentialID] = value
		}
		if value, ok := conversationOptions[internal_options.ModelOptionConnectionTransport]; ok && value != nil {
			resolved[modelOptionConnectionTransport] = value
		}
		if value, ok := conversationOptions[internal_options.ModelOptionConnectionEndpoint]; ok && value != nil {
			resolved[modelOptionConnectionEndpoint] = value
		}
	}

	delete(resolved, internal_options.ModelOptionCredentialID)
	delete(resolved, internal_options.ModelOptionConnectionTransport)
	delete(resolved, internal_options.ModelOptionConnectionEndpoint)
	return resolved
}

func modelConnectionOptions(options utils.Option) map[string]string {
	connectionOptions := make(map[string]string)
	for key, value := range options {
		if !strings.HasPrefix(key, modelOptionConnectionPrefix) || value == nil {
			continue
		}
		connectionOptions[key] = fmt.Sprintf("%v", value)
	}
	return connectionOptions
}
