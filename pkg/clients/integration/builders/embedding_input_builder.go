// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package integration_client_builders

import (
	"github.com/rapidaai/pkg/commons"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type embeddingInputBuilder struct {
	logger commons.Logger
}

func NewEmbeddingInputBuilder(logger commons.Logger) InputEmbeddingBuilder {
	return &embeddingInputBuilder{
		logger: logger,
	}
}

func (in *embeddingInputBuilder) Credential(i uint64, dp *structpb.Struct) *lexatic_backend.Credential {
	return &lexatic_backend.Credential{
		Id:    i,
		Value: dp,
	}
}
func (in *embeddingInputBuilder) Embedding(
	credential *lexatic_backend.Credential,
	modelOpts map[string]*anypb.Any,
	additionalData map[string]string,
	contents map[int32]string,
) *lexatic_backend.EmbeddingRequest {
	return &lexatic_backend.EmbeddingRequest{
		Credential:      credential,
		ModelParameters: modelOpts,
		Content:         contents,
		AdditionalData:  additionalData,
	}

}

func (in *embeddingInputBuilder) Arguments(
	variables []*gorm_types.PromptVariable,
	arguments map[string]*anypb.Any) map[string]interface{} {
	args, err := utils.AnyMapToInterfaceMap(arguments)
	if err != nil {
	}
	existing := make(map[string]interface{}, 0)
	for _, v := range variables {
		existing[v.Name] = v.DefaultValue
	}
	return utils.MergeMaps(existing, args)
}

func (in *embeddingInputBuilder) Options(
	opts map[string]interface{},
	options map[string]*anypb.Any) map[string]*anypb.Any {

	// If options is nil, initialize it
	if options == nil {
		options = make(map[string]*anypb.Any)
	}

	// Iterate through the opts map and add them to options
	for key, value := range opts {
		// Convert the value to *structpb.Value
		structValue, err := structpb.NewValue(value)
		if err != nil {
			// Handle error (you might want to log it or handle it according to your error handling strategy)
			continue
		}

		// Convert the *structpb.Value to *anypb.Any
		anyValue, err := anypb.New(structValue)
		if err != nil {
			// Handle error
			continue
		}

		options[key] = anyValue
	}

	return options
}
