// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerStartWaitsForListenerReadiness(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	server, err := NewServer(context.Background(), &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:                 "127.0.0.1",
			ExternalIP:              "127.0.0.1",
			AllowLoopbackExternalIP: true,
			Port:                    port,
			Transport:               TransportUDP,
		},
		Logger:            bridgeTestLogger(),
		RTPPortRangeStart: 19000,
		RTPPortRangeEnd:   19010,
	})
	require.NoError(t, err)
	require.NoError(t, server.Start())
	assert.True(t, server.IsRunning())

	duplicate, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if duplicate != nil {
		_ = duplicate.Close()
	}
	require.Error(t, err)

	server.Stop()
	assert.False(t, server.IsRunning())

	reopened, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	require.NoError(t, err)
	require.NoError(t, reopened.Close())
}

func TestServerStartReturnsListenerBindFailure(t *testing.T) {
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	defer reserved.Close()

	server, err := NewServer(context.Background(), &ServerConfig{
		ListenConfig: &ListenConfig{
			Address:                 "127.0.0.1",
			ExternalIP:              "127.0.0.1",
			AllowLoopbackExternalIP: true,
			Port:                    reserved.LocalAddr().(*net.UDPAddr).Port,
			Transport:               TransportUDP,
		},
		Logger:            bridgeTestLogger(),
		RTPPortRangeStart: 19000,
		RTPPortRangeEnd:   19010,
	})
	require.NoError(t, err)

	err = server.Start()
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to start SIP listener")
	assert.False(t, server.IsRunning())
}
