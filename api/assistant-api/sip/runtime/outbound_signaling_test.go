// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"testing"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInviteHeaders_Deterministic(t *testing.T) {
	request := testOutboundInviteRequest()
	request.Config.Headers = map[string]string{
		"X-Zeta":              "last",
		"X-Alpha":             "first",
		"Allow":               "bad-override",
		"P-Asserted-Identity": "bad-override",
	}

	headers, err := buildInviteHeaders(request)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"From",
		"P-Asserted-Identity",
		"Allow",
		"User-Agent",
		"X-Alpha",
		"X-Zeta",
	}, headerNames(headers))
	assert.Equal(t, outboundAllowHeaderValue, headers[2].Value())
	assert.Equal(t, sipUserAgent, headers[3].Value())
	assert.Equal(t, "first", headers[4].Value())
	assert.Equal(t, "last", headers[5].Value())
}

func TestBuildInviteHeaders_AllowsRouteHeader(t *testing.T) {
	request := testOutboundInviteRequest()
	request.Config.Headers = map[string]string{
		"Route": "<sip:proxy.example.com;lr>",
	}

	headers, err := buildInviteHeaders(request)
	require.NoError(t, err)

	require.Len(t, headers, 5)
	assert.Equal(t, "Route", headers[4].Name())
	assert.Equal(t, "<sip:proxy.example.com;lr>", headers[4].Value())
}

func TestNormalizeDialogRouteSet_UsesRecordRoute(t *testing.T) {
	inviteRequest := sip.NewRequest(sip.INVITE, sip.Uri{Scheme: "sip", User: "callee", Host: "trunk.example.com"})
	inviteRequest.AppendHeader(sip.NewHeader("Route", "<sip:initial.example.com;lr>"))
	inviteResponse := sip.NewResponseFromRequest(inviteRequest, 200, "OK", nil)
	inviteResponse.AppendHeader(sip.NewHeader("Contact", "<sip:uas@carrier.example.com>"))
	inviteResponse.AppendHeader(sip.NewHeader("Record-Route", "<sip:p2.example.com;lr>"))
	inviteResponse.AppendHeader(sip.NewHeader("Record-Route", "<sip:p1.example.com;lr>"))
	dialogSession := &sipgo.DialogClientSession{
		Dialog: sipgo.Dialog{
			InviteRequest:  inviteRequest,
			InviteResponse: inviteResponse,
		},
	}

	normalizeDialogRouteSet(dialogSession)

	assert.Empty(t, inviteRequest.GetHeaders("Route"))
}

func testOutboundInviteRequest() OutboundInviteRequest {
	return OutboundInviteRequest{
		Config: &OutboundConfig{
			Mode:      OutboundModeTrunkTermination,
			Address:   "trunk.example.com",
			Port:      5060,
			Transport: sip_config.TransportUDP,
			Domain:    "example.com",
		},
		Address: CallAddress{
			To:   "+15551234567",
			From: "+15557654321",
		},
	}
}

func headerNames(headers []sip.Header) []string {
	names := make([]string, 0, len(headers))
	for _, header := range headers {
		names = append(names, header.Name())
	}
	return names
}
