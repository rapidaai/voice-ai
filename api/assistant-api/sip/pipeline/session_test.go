// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"errors"
	"testing"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestReconstructCallContextInboundMapsPhoneValues(t *testing.T) {
	auth := newIdentityTestAuthentication()

	call, err := reconstructCallContext(
		auth,
		42,
		84,
		string(sip_infra.CallDirectionInbound),
		"call-inbound",
		"context-inbound",
		"07249994778",
		"+447249994778",
	)
	require.NoError(t, err)

	assert.Equal(t, "07249994778", call.CallerNumber)
	assert.Equal(t, "+447249994778", call.FromNumber)
}

func TestEnsureCallContextOutboundFallbackMapsPhoneValues(t *testing.T) {
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
			From: "+15557654321",
			To:   "+15551234567",
		},
	}, 84)

	require.NoError(t, err)
	assert.Equal(t, "+15551234567", call.CallerNumber)
	assert.Equal(t, "+15557654321", call.FromNumber)
}

func TestEnsureCallContextOutboundFallbackKeepsAliasPhonesEmpty(t *testing.T) {
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
		sip_infra.WithSessionCallID("call-outbound-alias"),
		sip_infra.WithSessionAuth(auth),
	)
	require.NoError(t, err)

	call, err := (&Dispatcher{}).ensureCallContext(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-outbound-alias",
		Session:     session,
		Direction:   sip_infra.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_infra.CallAddress{
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
		Direction:      string(sip_infra.CallDirectionOutbound),
		CallerNumber:   "customer-alias",
		FromNumber:     "assistant-alias",
		ChannelUUID:    "stored-channel-id",
	}
	store := &outboundContextTestStore{claimResult: stored}
	dispatcher := &Dispatcher{callContextStore: store}

	call, err := dispatcher.ensureCallContext(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-claimed-context",
		Session:     session,
		Direction:   sip_infra.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_infra.CallAddress{
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
		Direction:      string(sip_infra.CallDirectionOutbound),
		CallerNumber:   "customer-alias",
		FromNumber:     "assistant-alias",
		ChannelUUID:    "loaded-channel-id",
	}
	store := &outboundContextTestStore{
		claimErr:  errors.New("context already claimed"),
		getResult: stored,
	}
	dispatcher := &Dispatcher{callContextStore: store}

	call, err := dispatcher.ensureCallContext(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-loaded-context",
		Session:     session,
		Direction:   sip_infra.CallDirectionOutbound,
		AssistantID: 42,
		Auth:        auth,
		CallAddress: sip_infra.CallAddress{
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

func newOutboundContextTestSession(t *testing.T, auth *types.Authentication, contextID string) *sip_infra.Session {
	t.Helper()
	session, err := sip_infra.NewSession(context.Background(),
		sip_infra.WithSessionConfig(&sip_infra.Config{
			Server:            "sip.example.com",
			Port:              5060,
			Transport:         sip_infra.TransportUDP,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_infra.WithSessionDirection(sip_infra.CallDirectionOutbound),
		sip_infra.WithSessionCallID("call-"+contextID),
		sip_infra.WithSessionContextID(contextID),
		sip_infra.WithSessionAuth(auth),
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
