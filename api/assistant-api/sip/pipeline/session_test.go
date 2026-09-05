// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
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

func TestEnsureCallContextOutboundFallbackMapsPhoneValues(t *testing.T) {
	auth := newIdentityTestAuthentication()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         sip_runtime.TransportUDP,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionOutbound),
		sip_runtime.WithSessionCallID("call-outbound"),
		sip_runtime.WithSessionAuth(auth),
	)
	require.NoError(t, err)
	dispatcher := &Dispatcher{}

	call, err := dispatcher.ensureCallContext(context.Background(), SessionEstablishedPipeline{
		ID:          "call-outbound",
		Session:     session,
		Direction:   sip_runtime.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_runtime.CallAddress{
			From: "+15557654321",
			To:   "+15551234567",
		},
	}, 84)

	require.NoError(t, err)
	assert.Equal(t, "+15551234567", call.CallerNumber)
	assert.Equal(t, "+15557654321", call.FromNumber)
	assert.Equal(t, uint64(7), call.OrganizationID)
	assert.Equal(t, uint64(8), call.ProjectID)
	require.NotNil(t, call.AuthActorType)
	assert.Equal(t, string(types.ActorTypeService), *call.AuthActorType)
	require.NotNil(t, call.AuthActorID)
	assert.Equal(t, uint64(9), *call.AuthActorID)
}

func TestEnsureCallContextOutboundFallbackKeepsAliasPhonesEmpty(t *testing.T) {
	auth := newIdentityTestAuthentication()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         sip_runtime.TransportUDP,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionOutbound),
		sip_runtime.WithSessionCallID("call-outbound-alias"),
		sip_runtime.WithSessionAuth(auth),
	)
	require.NoError(t, err)

	call, err := (&Dispatcher{}).ensureCallContext(context.Background(), SessionEstablishedPipeline{
		ID:          "call-outbound-alias",
		Session:     session,
		Direction:   sip_runtime.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_runtime.CallAddress{
			FromURI: "sip:assistant-line@sip.example.com",
			ToURI:   "sip:customer-alias@sip.example.com",
		},
	}, 84)

	require.NoError(t, err)
	assert.Empty(t, call.CallerNumber)
	assert.Empty(t, call.FromNumber)
}

func TestEnsureCallContextOutboundClaimedContextUsesCallAddressPhones(t *testing.T) {
	auth := newIdentityTestAuthentication()
	session := newOutboundContextTestSession(t, auth, "claimed-context")
	stored := &callcontext.CallContext{
		ContextID:      "claimed-context",
		Status:         callcontext.StatusClaimed,
		AssistantID:    91,
		ConversationID: 92,
		Provider:       "sip",
		Direction:      string(sip_runtime.CallDirectionOutbound),
		CallerNumber:   "customer-alias",
		FromNumber:     "assistant-alias",
		ChannelUUID:    "stored-channel-id",
	}
	store := &outboundContextTestStore{claimResult: stored}
	dispatcher := &Dispatcher{callContextStore: store}

	call, err := dispatcher.ensureCallContext(context.Background(), SessionEstablishedPipeline{
		ID:          "call-claimed-context",
		Session:     session,
		Direction:   sip_runtime.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_runtime.CallAddress{
			From: "+15557654321",
			To:   "+15551234567",
		},
	}, 84)

	require.NoError(t, err)
	assert.Same(t, stored, call)
	assert.Equal(t, "+15551234567", call.CallerNumber)
	assert.Equal(t, "+15557654321", call.FromNumber)
	assert.Equal(t, uint64(91), call.AssistantID)
	assert.Equal(t, uint64(92), call.ConversationID)
	assert.Equal(t, "stored-channel-id", call.ChannelUUID)
	assert.Equal(t, 1, store.claimCalls)
	assert.Zero(t, store.getCalls)
}

func TestEnsureCallContextOutboundLoadedContextClearsAliasPhones(t *testing.T) {
	auth := newIdentityTestAuthentication()
	session := newOutboundContextTestSession(t, auth, "loaded-context")
	stored := &callcontext.CallContext{
		ContextID:      "loaded-context",
		Status:         callcontext.StatusClaimed,
		AssistantID:    101,
		ConversationID: 102,
		Provider:       "sip",
		Direction:      string(sip_runtime.CallDirectionOutbound),
		CallerNumber:   "customer-alias",
		FromNumber:     "assistant-alias",
		ChannelUUID:    "loaded-channel-id",
	}
	store := &outboundContextTestStore{
		claimErr:  errors.New("context already claimed"),
		getResult: stored,
	}
	dispatcher := &Dispatcher{callContextStore: store}

	call, err := dispatcher.ensureCallContext(context.Background(), SessionEstablishedPipeline{
		ID:          "call-loaded-context",
		Session:     session,
		Direction:   sip_runtime.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_runtime.CallAddress{
			FromURI: "sip:assistant-alias@sip.example.com",
			ToURI:   "sip:customer-alias@sip.example.com",
		},
	}, 84)

	require.NoError(t, err)
	assert.Same(t, stored, call)
	assert.Empty(t, call.CallerNumber)
	assert.Empty(t, call.FromNumber)
	assert.Equal(t, uint64(101), call.AssistantID)
	assert.Equal(t, uint64(102), call.ConversationID)
	assert.Equal(t, "loaded-channel-id", call.ChannelUUID)
	assert.Equal(t, 1, store.claimCalls)
	assert.Equal(t, 1, store.getCalls)
}

func newOutboundContextTestSession(t *testing.T, auth *types.Authentication, contextID string) *sip_runtime.Session {
	t.Helper()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         sip_runtime.TransportUDP,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionOutbound),
		sip_runtime.WithSessionCallID("call-"+contextID),
		sip_runtime.WithSessionContextID(contextID),
		sip_runtime.WithSessionAuth(auth),
	)
	require.NoError(t, err)
	return session
}

type outboundContextTestStore struct {
	callcontext.Store
	claimResult *callcontext.CallContext
	claimErr    error
	getResult   *callcontext.CallContext
	getErr      error
	claimCalls  int
	getCalls    int
}

func (s *outboundContextTestStore) Claim(_ context.Context, _ string) (*callcontext.CallContext, error) {
	s.claimCalls++
	return s.claimResult, s.claimErr
}

func (s *outboundContextTestStore) Get(_ context.Context, _ string) (*callcontext.CallContext, error) {
	s.getCalls++
	return s.getResult, s.getErr
}
