// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

//go:build sipintegration && freeswitch

package freeswitch_test

import (
	"context"
	"errors"
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
	snapshot := registrationClient.Snapshot(inboundConfig.registeredDID)
	require.True(t, snapshot.Active)
	require.True(t, snapshot.Healthy)
	require.False(t, snapshot.ExpiresAt.IsZero())
}

func TestFreeSWITCHRegistrationRejectsInvalidPassword(t *testing.T) {
	inboundConfig := loadRegistrationInboundConfig(t)
	harness := newFreeSWITCHHarness(t, inboundConfig.sipCredentialConfig)
	registrationClient := harness.registrationClient()
	invalidConfig := *harness.sipConfig
	invalidConfig.Password = "invalid-integration-password"

	registerContext, cancelRegister := context.WithTimeout(context.Background(), callSetupTimeout)
	defer cancelRegister()

	err := registrationClient.Register(registerContext, &sip_runtime.Registration{
		DID:         inboundConfig.registeredDID,
		Config:      &invalidConfig,
		AssistantID: 1001,
		ExpiresIn:   120,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, sip_runtime.ErrRegistrationFailed)
	require.ErrorIs(t, err, sip_runtime.ErrPermanentFailure)
	var registrationError *sip_runtime.RegistrationError
	require.True(t, errors.As(err, &registrationError))
	require.Equal(t, sip_runtime.RegistrationFailureClassRejected, registrationError.Class)
	require.Equal(t, sip_runtime.RegistrationFailureReasonRegistrarRejected, registrationError.Reason)
	require.Equal(t, 403, registrationError.StatusCode)
	require.False(t, registrationClient.IsRegistered(inboundConfig.registeredDID))
}
