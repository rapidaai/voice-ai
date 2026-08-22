// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"testing"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestReconstructCallContextInboundPreservesSIPIdentities(t *testing.T) {
	auth := &types.ProjectScope{}

	call := reconstructCallContext(
		auth,
		42,
		84,
		string(sip_infra.CallDirectionInbound),
		"call-inbound",
		"context-inbound",
		"sip:alice@example.com;user=phone",
		"sip:agent-42@sip.rapida.ai",
	)

	assert.Equal(t, "sip:alice@example.com;user=phone", call.CallerNumber)
	assert.Equal(t, "sip:agent-42@sip.rapida.ai", call.FromNumber)
}

func TestReconstructCallContextOutboundPreservesResolvedIdentities(t *testing.T) {
	auth := &types.ProjectScope{}

	call := reconstructCallContext(
		auth,
		42,
		84,
		string(sip_infra.CallDirectionOutbound),
		"call-outbound",
		"context-outbound",
		"sip:assistant@example.com",
		"sip:bob@example.net",
	)

	assert.Equal(t, "sip:bob@example.net", call.CallerNumber)
	assert.Equal(t, "sip:assistant@example.com", call.FromNumber)
}
