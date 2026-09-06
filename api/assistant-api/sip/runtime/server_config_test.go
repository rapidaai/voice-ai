// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerConfigValidate_AcceptsSocketOwnedRTPConfig(t *testing.T) {
	cfg := validServerConfigForValidation()

	require.NoError(t, cfg.Validate())
}

func TestServerConfigValidate_AcceptsUnlimitedCallAdmission(t *testing.T) {
	cfg := validServerConfigForValidation()
	cfg.MaxConcurrentCalls = 0

	require.NoError(t, cfg.Validate())
}

func TestServerConfigValidate_RejectsNegativeCallAdmission(t *testing.T) {
	cfg := validServerConfigForValidation()
	cfg.MaxConcurrentCalls = -1

	require.Error(t, cfg.Validate())
}

func TestServerConfigValidate_RejectsPartialCallRateAdmission(t *testing.T) {
	cfg := validServerConfigForValidation()
	cfg.CallAdmissionCPS = 1

	require.Error(t, cfg.Validate())

	cfg = validServerConfigForValidation()
	cfg.CallAdmissionBurst = 1

	require.Error(t, cfg.Validate())
}

func TestServerConfigValidateRejectsNil(t *testing.T) {
	var config *ServerConfig

	require.Error(t, config.Validate())
}

func TestNewServerRejectsNilConfig(t *testing.T) {
	server, err := NewServer(nil, nil)

	require.Error(t, err)
	require.Nil(t, server)
}

func TestServerGetListenConfigReturnsCopy(t *testing.T) {
	server := &Server{listenConfig: &ListenConfig{Address: "127.0.0.1", Port: 5060}}

	config := server.GetListenConfig()
	config.Address = "0.0.0.0"
	config.Port = 5090

	stored := server.GetListenConfig()
	require.Equal(t, "127.0.0.1", stored.Address)
	require.Equal(t, 5060, stored.Port)
}

func TestServerUseSymmetricRTPForRemoteIP(t *testing.T) {
	require.True(t, (&Server{symmetricRTP: true}).useSymmetricRTPForRemoteIP("203.0.113.10"))
	require.True(t, (&Server{ignoreLocalAddrInSDP: true}).useSymmetricRTPForRemoteIP("10.0.0.10"))
	require.False(t, (&Server{ignoreLocalAddrInSDP: true}).useSymmetricRTPForRemoteIP("203.0.113.10"))
	require.False(t, (&Server{ignoreLocalAddrInSDP: true}).useSymmetricRTPForRemoteIP("not-an-ip"))
	require.False(t, (&Server{}).useSymmetricRTPForRemoteIP("10.0.0.10"))
}

func TestServerSetMiddlewaresSkipsNil(t *testing.T) {
	server := &Server{}
	middleware := func(*SIPRequestContext) error { return nil }

	server.SetMiddlewares([]Middleware{nil, middleware})

	require.Len(t, server.middlewares, 1)
}

func validServerConfigForValidation() *ServerConfig {
	return &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:   "0.0.0.0",
			Port:      5060,
			Transport: TransportUDP,
		},
		Logger:            bridgeTestLogger(),
		RTPPortRangeStart: 10000,
		RTPPortRangeEnd:   10010,
	}
}
