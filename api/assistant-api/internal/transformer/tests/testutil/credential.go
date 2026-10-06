// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transformer_testutil

import (
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

// BuildCredential converts a flat string map (from YAML config) into a VaultCredential proto.
func BuildCredential(creds map[string]string) *protos.VaultCredential {
	fields := make(map[string]interface{}, len(creds))
	for k, v := range creds {
		fields[k] = v
	}
	s, err := structpb.NewStruct(fields)
	if err != nil {
		// Only happens with unsupported types, which won't occur with string values.
		panic("failed to build structpb.Struct from credentials: " + err.Error())
	}
	return &protos.VaultCredential{Value: s}
}

// BuildOptions converts the YAML options map into a utils.Option.
func BuildOptions(opts map[string]interface{}) utils.Option {
	if opts == nil {
		return utils.Option{}
	}
	return utils.Option(opts)
}

// NewTestLogger creates a lightweight logger suitable for integration tests.
func NewTestLogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}
