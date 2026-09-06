// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_sip

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	sip_pipeline "github.com/rapidaai/api/assistant-api/sip/pipeline"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	app_config "github.com/rapidaai/config"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSIPEngineRetainsRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}
	engine := &SIPEngine{rapidaClient: client}

	require.Same(t, client, engine.rapidaClient)
}

func TestSIPEngineDoesNotConstructInternalClients(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "sip.go", nil, 0)
	require.NoError(t, err)

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "NewVaultClientGRPC" {
			t.Fatal("SIP engine constructs a Vault client")
		}
		return true
	})
}

func TestSIPEngineDoesNotPassPostgresToMiddleware(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "sip.go", nil, 0)
	require.NoError(t, err)

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "WithPostgres" {
			return true
		}
		packageName, ok := selector.X.(*ast.Ident)
		if ok && packageName.Name == "sip_middleware" {
			t.Fatal("SIP engine passes PostgreSQL to middleware")
		}
		return true
	})
}

func TestSIPEnginePassesCallAdmissionConfigToRuntime(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "sip.go", nil, 0)
	require.NoError(t, err)

	expectedFields := map[string]bool{
		"MaxConcurrentCalls": false,
		"CallAdmissionCPS":   false,
		"CallAdmissionBurst": false,
	}

	ast.Inspect(file, func(node ast.Node) bool {
		composite, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		selector, ok := composite.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ServerConfig" {
			return true
		}
		for _, element := range composite.Elts {
			keyValue, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			field, ok := keyValue.Key.(*ast.Ident)
			if ok {
				if _, exists := expectedFields[field.Name]; exists {
					expectedFields[field.Name] = true
				}
			}
		}
		return false
	})

	for field, found := range expectedFields {
		assert.True(t, found, "runtime ServerConfig missing %s", field)
	}
}

func TestSIPEngineUsesConfiguredServiceID(t *testing.T) {
	engine := &SIPEngine{cfg: &assistant_config.AssistantConfig{AppConfig: app_config.AppConfig{ServiceID: 9007}}}
	assert.Equal(t, uint64(9007), engine.cfg.ServiceID)
}

func TestSessionEstablishedStagePreservesCallAddress(t *testing.T) {
	auth := &types.Authentication{}
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionInbound),
		sip_runtime.WithSessionCallID("call-address"),
		sip_runtime.WithSessionAuth(auth),
		sip_runtime.WithSessionAssistant(&internal_assistant_entity.Assistant{}),
	)
	require.NoError(t, err)
	address := sip_runtime.CallAddress{
		From:    "+14155550100",
		To:      "agent-42",
		FromURI: "sip:+14155550100@carrier.example.com",
		ToURI:   "sip:agent-42@sip.rapida.ai",
		Headers: map[string]string{"x-original-called-number": "+14155550200"},
	}

	var stage sip_pipeline.SessionEstablishedPipeline
	stage, err = (&SIPEngine{}).sessionEstablishedStage(session, "sip:agent-42@sip.rapida.ai", address)

	require.NoError(t, err)
	assert.Equal(t, address, stage.CallAddress)
}

func TestPersistRemoteByeCallStatus_UpdatesCompletedDisconnectMetadata(t *testing.T) {
	store := newSIPCallStatusTestStore(&callcontext.CallContext{
		ContextID:  "ctx-bye",
		Status:     callcontext.StatusClaimed,
		CallStatus: callcontext.CallStatusAnswered,
	})
	engine := &SIPEngine{
		logger:           newSIPTestLogger(t),
		callContextStore: store,
	}
	session := newSIPCallStatusTestSession(t, "ctx-bye")

	engine.persistRemoteByeCallStatus(session, sip_runtime.DisconnectMetadata{
		Reason:             sip_runtime.DisconnectReasonNormalClearing,
		ProviderStatusCode: 16,
	})

	require.NotNil(t, store.lastStatus)
	assert.Equal(t, callcontext.StatusCompleted, store.callContext.Status)
	assert.Equal(t, callcontext.CallStatusCompleted, store.lastStatus.CallStatus)
	assert.Equal(t, sip_runtime.DisconnectReasonNormalClearing, store.lastStatus.DisconnectReason)
	assert.Equal(t, 16, store.lastStatus.ProviderStatusCode)
}

func TestPersistRemoteByeCallStatus_DoesNotDowngradeFailure(t *testing.T) {
	store := newSIPCallStatusTestStore(&callcontext.CallContext{
		ContextID:  "ctx-failed",
		Status:     callcontext.StatusFailed,
		CallStatus: callcontext.CallStatusFailed,
	})
	engine := &SIPEngine{
		logger:           newSIPTestLogger(t),
		callContextStore: store,
	}
	session := newSIPCallStatusTestSession(t, "ctx-failed")

	engine.persistRemoteByeCallStatus(session, sip_runtime.DisconnectMetadata{
		Reason: sip_runtime.DisconnectReasonRemoteHangup,
	})

	assert.Nil(t, store.lastStatus)
	assert.Equal(t, callcontext.StatusFailed, store.callContext.Status)
}

type sipCallStatusTestStore struct {
	callContext *callcontext.CallContext
	lastStatus  *callcontext.CallStatusUpdate
}

func newSIPCallStatusTestStore(callContext *callcontext.CallContext) *sipCallStatusTestStore {
	return &sipCallStatusTestStore{callContext: callContext}
}

func (s *sipCallStatusTestStore) Save(_ context.Context, cc *callcontext.CallContext) (string, error) {
	s.callContext = cc
	return cc.ContextID, nil
}

func (s *sipCallStatusTestStore) Get(_ context.Context, contextID string) (*callcontext.CallContext, error) {
	if s.callContext == nil || s.callContext.ContextID != contextID {
		return nil, fmt.Errorf("call context not found: %s", contextID)
	}
	return s.callContext, nil
}

func (s *sipCallStatusTestStore) GetByChannelUUID(_ context.Context, _ string, _ uint64, channelUUID string) (*callcontext.CallContext, error) {
	if s.callContext == nil || s.callContext.ChannelUUID != channelUUID {
		return nil, fmt.Errorf("call context not found for channel uuid: %s", channelUUID)
	}
	return s.callContext, nil
}

func (s *sipCallStatusTestStore) Claim(_ context.Context, contextID string) (*callcontext.CallContext, error) {
	return s.Get(context.Background(), contextID)
}

func (s *sipCallStatusTestStore) UpdateField(_ context.Context, contextID, field, value string) error {
	if _, err := s.Get(context.Background(), contextID); err != nil {
		return err
	}
	if field == "status" {
		s.callContext.Status = value
	}
	return nil
}

func (s *sipCallStatusTestStore) UpdateCallStatus(_ context.Context, contextID string, status callcontext.CallStatusUpdate) error {
	if _, err := s.Get(context.Background(), contextID); err != nil {
		return err
	}
	s.lastStatus = &status
	s.callContext.CallStatus = status.CallStatus
	s.callContext.DisconnectReason = status.DisconnectReason
	s.callContext.ProviderStatusCode = status.ProviderStatusCode
	if status.CallStatus == callcontext.CallStatusCompleted {
		s.callContext.Status = callcontext.StatusCompleted
	}
	if status.CallStatus == callcontext.CallStatusFailed || status.CallStatus == callcontext.CallStatusCancelled {
		s.callContext.Status = callcontext.StatusFailed
	}
	return nil
}

func newSIPCallStatusTestSession(t *testing.T, contextID string) *sip_runtime.Session {
	t.Helper()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionOutbound),
		sip_runtime.WithSessionCallID("sip-call-id"),
		sip_runtime.WithSessionContextID(contextID),
	)
	require.NoError(t, err)
	return session
}

func newSIPTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
		commons.Level("error"),
	)
	require.NoError(t, err)
	return logger
}
