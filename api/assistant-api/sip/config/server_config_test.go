// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfigValidate_AcceptsSocketOwnedRTPConfig(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableFile(false))
	require.NoError(t, err)

	cfg := &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:   "0.0.0.0",
			Port:      5060,
			Transport: TransportUDP,
		},
		Logger:            logger,
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10010,
	}

	require.NoError(t, cfg.Validate())
}

func TestServerConfigValidate_AcceptsUnlimitedCallAdmission(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableFile(false))
	require.NoError(t, err)

	cfg := &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:   "0.0.0.0",
			Port:      5060,
			Transport: TransportUDP,
		},
		Logger:             logger,
		RTPPortRangeStart:  10000,
		RTPPortRangeEnd:    10010,
		MaxConcurrentCalls: 0,
	}

	require.NoError(t, cfg.Validate())
}

func TestServerConfigValidate_RejectsNegativeCallAdmission(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableFile(false))
	require.NoError(t, err)

	cfg := &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:   "0.0.0.0",
			Port:      5060,
			Transport: TransportUDP,
		},
		Logger:             logger,
		RTPPortRangeStart:  10000,
		RTPPortRangeEnd:    10010,
		MaxConcurrentCalls: -1,
	}

	require.Error(t, cfg.Validate())
}

func TestServerConfigValidate_RejectsPartialCallRateAdmission(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableFile(false))
	require.NoError(t, err)

	cfg := &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:   "0.0.0.0",
			Port:      5060,
			Transport: TransportUDP,
		},
		Logger:             logger,
		RTPPortRangeStart:  10000,
		RTPPortRangeEnd:    10010,
		CallAdmissionCPS:   1,
		CallAdmissionBurst: 0,
	}
	require.Error(t, cfg.Validate())

	cfg.CallAdmissionCPS = 0
	cfg.CallAdmissionBurst = 1
	require.Error(t, cfg.Validate())
}

func TestServerConfigValidateRejectsNil(t *testing.T) {
	var config *ServerConfig

	require.Error(t, config.Validate())
}

func TestListenConfigSIPContactHeaderTransport(t *testing.T) {
	tests := []struct {
		name           string
		transport      Transport
		expectedScheme string
		expectedParam  string
	}{
		{name: "udp", transport: TransportUDP, expectedScheme: "sip"},
		{name: "tcp", transport: TransportTCP, expectedScheme: "sip", expectedParam: "tcp"},
		{name: "tls", transport: TransportTLS, expectedScheme: "sips", expectedParam: "tls"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contact := (&ListenConfig{
				ExternalIP: "203.0.113.10",
				Port:       5061,
				Transport:  test.transport,
			}).SIPContactHeader()

			assert.Equal(t, test.expectedScheme, contact.Address.Scheme)
			assert.Equal(t, "203.0.113.10", contact.Address.Host)
			assert.Equal(t, 5061, contact.Address.Port)
			if test.expectedParam == "" {
				assert.Nil(t, contact.Address.UriParams)
				return
			}
			transport, ok := contact.Address.UriParams.Get("transport")
			require.True(t, ok)
			assert.Equal(t, test.expectedParam, transport)
		})
	}
}
