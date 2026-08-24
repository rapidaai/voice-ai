// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"testing"

	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
)

func requestWithReason(value string) *sip.Request {
	request := sip.NewRequest(sip.BYE, sip.Uri{Host: "example.com"})
	request.AppendHeader(sip.NewHeader("Reason", value))
	return request
}

func TestSIPReason_Q850NormalClearing(t *testing.T) {
	metadata := NewSIPReason(requestWithReason(`Q.850;cause=16;text="Normal call clearing"`)).DisconnectMetadata()

	assert.Equal(t, DisconnectReasonNormalClearing, metadata.Reason)
	assert.Equal(t, 16, metadata.ProviderStatusCode)
	assert.Equal(t, "Normal call clearing", metadata.Text)
}

func TestSIPReason_SIPBusy(t *testing.T) {
	metadata := NewSIPReason(requestWithReason(`SIP;cause=486;text="Busy Here"`)).DisconnectMetadata()

	assert.Equal(t, DisconnectReasonBusy, metadata.Reason)
	assert.Equal(t, 486, metadata.ProviderStatusCode)
	assert.Equal(t, "Busy Here", metadata.Text)
}

func TestSIPReason_QuotedTextWithSemicolon(t *testing.T) {
	metadata := NewSIPReason(requestWithReason(`Q.850;cause=41;text="Temporary failure; upstream"`)).DisconnectMetadata()

	assert.Equal(t, DisconnectReasonNetworkFailure, metadata.Reason)
	assert.Equal(t, "Temporary failure; upstream", metadata.Text)
}

func TestSIPReason_DefaultsToRemoteHangup(t *testing.T) {
	metadata := NewSIPReason(nil).DisconnectMetadata()

	assert.Equal(t, DisconnectReasonRemoteHangup, metadata.Reason)
	assert.Zero(t, metadata.ProviderStatusCode)
}
