// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}
	dispatcher := New(WithRapidaClient(client))

	require.Same(t, client, dispatcher.rapidaClient)
}

func TestSessionBillingUsesProductUsage(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "session.go", nil, 0)
	require.NoError(t, err)

	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "ProductUsage" {
			found = true
		}
		return true
	})
	require.True(t, found, "SIP billing does not use RapidaClient.ProductUsage")
}

func TestSIPTalkerReceivesRapidaClient(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "runtime.go", nil, 0)
	require.NoError(t, err)

	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "WithRapidaClient" {
			found = true
		}
		return true
	})
	require.True(t, found, "SIP talker does not receive RapidaClient")
}

func newIdentityTestAuthentication() *types.Authentication {
	organizationID := uint64(7)
	projectID := uint64(8)
	serviceID := uint64(9)
	return &types.Authentication{
		AuthType:          types.AuthTypeService,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeService, ID: serviceID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}

func TestReconstructCallContextInboundPreservesSIPIdentities(t *testing.T) {
	auth := newIdentityTestAuthentication()

	call, err := reconstructCallContext(
		auth,
		42,
		84,
		string(sip_infra.CallDirectionInbound),
		"call-inbound",
		"context-inbound",
		"sip:alice@example.com;user=phone",
		"sip:agent-42@sip.rapida.ai",
	)
	require.NoError(t, err)

	assert.Equal(t, "sip:alice@example.com;user=phone", call.CallerNumber)
	assert.Equal(t, "sip:agent-42@sip.rapida.ai", call.FromNumber)
}

func TestEnsureCallContextOutboundFallbackPreservesResolvedRequestIdentities(t *testing.T) {
	auth := newIdentityTestAuthentication()
	session, err := sip_infra.NewSession(context.Background(),
		sip_infra.WithSessionConfig(&sip_infra.Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         sip_infra.TransportUDP,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_infra.WithSessionDirection(sip_infra.CallDirectionOutbound),
		sip_infra.WithSessionCallID("call-outbound"),
		sip_infra.WithSessionAuth(auth),
	)
	require.NoError(t, err)
	dispatcher := &Dispatcher{}

	call, err := dispatcher.ensureCallContext(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-outbound",
		Session:     session,
		Direction:   sip_infra.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_infra.CallAddress{
			From: "assistant-line",
			To:   "customer-alias",
		},
	}, 84)

	require.NoError(t, err)
	assert.Equal(t, "customer-alias", call.CallerNumber)
	assert.Equal(t, "assistant-line", call.FromNumber)
}
