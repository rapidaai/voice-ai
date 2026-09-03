// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package sip_integration

import (
	"context"
	"testing"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/stretchr/testify/require"
)

func TestFreeSWITCHRegistrationUsernamePassword(t *testing.T) {
	inboundConfig := loadRegistrationInboundConfig(t)
	harness := newFreeSWITCHHarness(t, inboundConfig.sipCredentialConfig)
	registrationClient := harness.registrationClient()

	registerContext, cancelRegister := context.WithTimeout(context.Background(), callSetupTimeout)
	defer cancelRegister()

	registration := &sip_runtime.Registration{
		DID:         inboundConfig.registeredDID,
		Config:      harness.sipConfig,
		AssistantID: 1001,
		ExpiresIn:   120,
	}
	require.NoError(t, registrationClient.Register(registerContext, registration))
	t.Cleanup(func() {
		unregisterContext, cancelUnregister := context.WithTimeout(context.Background(), callTeardownTimeout)
		defer cancelUnregister()
		require.NoError(t, registrationClient.Unregister(unregisterContext, inboundConfig.registeredDID))
	})

	require.True(t, registrationClient.IsRegistered(inboundConfig.registeredDID))
	require.Contains(t, registrationClient.GetRegisteredDIDs(), inboundConfig.registeredDID)
}
