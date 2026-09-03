// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/require"
)

type failingTalker struct {
	internal_type.Talking
	err error
}

func (talker failingTalker) Talk(context.Context, *types.Authentication) error {
	return talker.err
}

func TestSIPPreparedCallRuntimeReturnsTalkerError(t *testing.T) {
	expected := errors.New("talker failed")
	runtime := &sipPreparedCallRuntime{
		talkContext: context.Background(),
		talker:      failingTalker{err: expected},
	}

	require.ErrorIs(t, runtime.runTalker(), expected)
}
