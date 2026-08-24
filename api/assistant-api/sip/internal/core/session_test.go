// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionAppliesOptions(t *testing.T) {
	config := testSessionConfig()

	session, err := NewSession(context.Background(),
		WithSessionConfig(config),
		WithSessionDirection(CallDirectionOutbound),
		WithSessionCallID("call-123"),
		WithSessionCodec(&CodecPCMA),
		WithSessionConversationID(42),
		WithSessionContextID("context-123"),
	)

	require.NoError(t, err)
	assert.Same(t, config, session.GetConfig())
	assert.Equal(t, "call-123", session.GetCallID())
	assert.Equal(t, CallDirectionOutbound, session.GetInfo().Direction)
	assert.Equal(t, CodecPCMA.Name, session.GetNegotiatedCodec().Name)
	assert.Equal(t, uint64(42), session.GetConversationID())
	assert.Equal(t, "context-123", session.GetContextID())
	assert.Equal(t, OutboundDialogPhaseInviting, session.GetOutboundDialogPhase())
}

func TestNewSessionRequiresConfig(t *testing.T) {
	session, err := NewSession(context.Background())

	require.ErrorIs(t, err, ErrInvalidConfig)
	assert.Nil(t, session)
}
