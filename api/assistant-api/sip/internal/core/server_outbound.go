// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"fmt"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// MakeCall initiates an outbound SIP call and registers the dialog for routing.
func (s *Server) MakeCall(ctx context.Context, cfg *Config, toUser, fromUser string, opts MakeCallOptions) (*Session, error) {
	outboundCall, err := s.prepareOutboundCallLeg(ctx, cfg, toUser, fromUser, outboundCallLegOptions{
		purpose:         OutboundLegPurposePrimary,
		makeCallOptions: opts,
	})
	if err != nil {
		return nil, err
	}

	outboundCall.ReportStatus(internal_type.ProviderCallStatusUpdate{CallStatus: string(OutboundCallStatusInitiated)})
	outboundCall.Start()

	return outboundCall.session, nil
}

type outboundCallLegOptions struct {
	purpose         OutboundLegPurpose
	makeCallOptions MakeCallOptions
	parentCallID    string
	parentContextID string
	parentConvID    uint64
	transferTarget  string
	transferAttempt int
	transferTotal   int
}

func (s *Server) prepareOutboundCallLeg(ctx context.Context, cfg *Config, toUser, fromUser string, opts outboundCallLegOptions) (*Outbound, error) {
	if s.state.Load() != int32(ServerStateRunning) {
		return nil, fmt.Errorf("SIP server is not running")
	}
	setupContext := ctx
	// Primary outbound calls outlive the API request that dispatches them.
	callLifecycleContext := context.WithoutCancel(ctx)

	request, err := NewOutboundInviteRequest(cfg, toUser, fromUser)
	if err != nil {
		return nil, err
	}

	session, err := s.createAndRegisterOutboundSession(callLifecycleContext, cfg, "", opts.makeCallOptions)
	if err != nil {
		return nil, err
	}
	s.applyOutboundLegMetadata(session, opts)

	media := NewOutboundMedia(s, session, request)
	if err := media.Prepare(); err != nil {
		failure := NewOutboundSetupFailure(err)
		s.failOutboundCallLegSetup(session, media, failure, opts.makeCallOptions.CallStatusObserver)
		return nil, err
	}

	dialog := NewOutboundDialog(s, session, request)
	sdpOffer, err := media.SDPOffer()
	if err != nil {
		failure := NewOutboundSetupFailure(err)
		s.failOutboundCallLegSetup(session, media, failure, opts.makeCallOptions.CallStatusObserver)
		return nil, err
	}
	if err := dialog.Invite(setupContext, sdpOffer); err != nil {
		failure := NewOutboundSetupFailure(err)
		s.failOutboundCallLegSetup(session, media, failure, opts.makeCallOptions.CallStatusObserver)
		return nil, err
	}

	outboundCall := NewOutbound(s, session, dialog, media, request)
	outboundCall.statusObserver = opts.makeCallOptions.CallStatusObserver
	return outboundCall, nil
}

func (s *Server) failOutboundCallLegSetup(
	session *Session,
	media *outboundMedia,
	failure OutboundFailure,
	statusObserver internal_type.ProviderCallStatusReporter,
) {
	failure.Record(session)
	if statusObserver != nil {
		statusObserver(failure.StatusUpdate(session.GetCallID()))
	}
	_ = s.FailCall(session, failure.LifecycleReason, failure.Err)
}

func (s *Server) applyOutboundLegMetadata(session *Session, opts outboundCallLegOptions) {
	if session == nil {
		return
	}
	purpose := opts.purpose
	if purpose == "" {
		purpose = OutboundLegPurposePrimary
	}
	session.SetMetadata(MetadataOutboundLegPurpose, string(purpose))
	if opts.parentCallID != "" {
		session.SetMetadata(MetadataOutboundParentCallID, opts.parentCallID)
	}
	if opts.parentContextID != "" {
		session.SetMetadata(MetadataOutboundParentContextID, opts.parentContextID)
	}
	if opts.parentConvID > 0 {
		session.SetMetadata(MetadataOutboundParentConversationID, opts.parentConvID)
	}
	if opts.transferTarget != "" {
		session.SetMetadata(MetadataOutboundTransferTarget, opts.transferTarget)
	}
	if opts.transferAttempt > 0 {
		session.SetMetadata(MetadataOutboundTransferAttempt, opts.transferAttempt)
	}
	if opts.transferTotal > 0 {
		session.SetMetadata(MetadataOutboundTransferTotal, opts.transferTotal)
	}
}

func (s *Server) createAndRegisterOutboundSession(ctx context.Context, cfg *Config, callID string, opts MakeCallOptions) (*Session, error) {
	session, err := NewSession(ctx,
		WithSessionConfig(cfg),
		WithSessionDirection(CallDirectionOutbound),
		WithSessionCallID(callID),
		WithSessionCodec(&CodecPCMU),
		WithSessionAuth(opts.Auth),
		WithSessionAssistant(opts.Assistant),
		WithSessionConversationID(opts.ConversationID),
		WithSessionContextID(opts.ContextID),
		WithSessionVaultCredential(opts.VaultCredential),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbound session: %w", err)
	}
	s.registerSession(session, session.GetCallID())
	return session, nil
}
