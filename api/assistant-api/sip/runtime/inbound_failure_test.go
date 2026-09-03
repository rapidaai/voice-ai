// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"testing"

	"github.com/emiago/sipgo/sip"
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundFailureError(t *testing.T) {
	cause := fmt.Errorf("%w: unsupported payload", ErrCodecNotSupported)
	failure := inboundFailure{
		statusCode:      488,
		responseClass:   internal_inbound.FailureMedia,
		lifecycleReason: LifecycleReasonInboundInviteFailed,
		err:             cause,
	}

	assert.Equal(t, "codec not supported: unsupported payload", failure.Error())
	assert.ErrorIs(t, failure, ErrCodecNotSupported)
}

func TestNewInboundMediaOfferFailure(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-unsupported-content")
	for request.RemoveHeader("Content-Type") {
	}
	request.AppendHeader(sip.NewHeader("Content-Type", "application/json"))
	request.SetBody([]byte("{}"))

	_, failure := NewInboundMediaOffer(
		server,
		request,
		"inbound INVITE",
		LifecycleReasonInboundInviteFailed,
		false,
	)

	require.NotNil(t, failure)
	assert.Equal(t, 415, failure.statusCode)
	assert.Equal(t, inboundFailureUnsupportedMedia, failure.class)
	assert.Equal(t, internal_inbound.FailureMedia, failure.responseClass)
	assert.Equal(t, CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"}, failure.termination)
	assert.Equal(t, LifecycleReasonInboundInviteFailed, failure.lifecycleReason)
	assert.ErrorIs(t, failure, ErrSDPParseFailed)
}

func TestNewInboundMediaOfferUnsupportedCodec(t *testing.T) {
	server := newServerForCommandTests(t)
	request := newInboundInviteRequest("inbound-unsupported-codec")
	request.SetBody([]byte(inboundOfferSDPWithMedia("127.0.0.1", 19000, "18 101")))

	_, failure := NewInboundMediaOffer(
		server,
		request,
		"inbound INVITE",
		LifecycleReasonInboundInviteFailed,
		false,
	)

	require.NotNil(t, failure)
	assert.Equal(t, 488, failure.statusCode)
	assert.Equal(t, inboundFailureUnsupportedMedia, failure.class)
	assert.Equal(t, internal_inbound.FailureMedia, failure.responseClass)
	assert.Equal(t, CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"}, failure.termination)
	assert.Equal(t, LifecycleReasonInboundInviteFailed, failure.lifecycleReason)
	assert.ErrorIs(t, failure, ErrCodecNotSupported)
}
