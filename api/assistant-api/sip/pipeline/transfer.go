// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"strings"
	"time"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
)

func (d *Dispatcher) handleTransferInitiated(ctx context.Context, v TransferInitiatedPipeline) {
	go d.executeTransfer(ctx, v)
}

func (d *Dispatcher) executeTransfer(ctx context.Context, v TransferInitiatedPipeline) {
	d.logger.Infow("SIP transfer initiated",
		"call_id", v.ID, "target", v.TargetURI)

	if d.server == nil {
		d.logger.Errorw("SIP transfer failed",
			"call_id", v.ID, "target", v.TargetURI, "reason", "server_nil")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		return
	}

	config := v.Config
	if config == nil {
		config = v.Session.GetConfig()
	}
	if config == nil {
		d.logger.Errorw("SIP transfer failed", "call_id", v.ID, "target", v.TargetURI, "reason", "config_missing")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		return
	}
	transferConfig := *config

	if transferConfig.CallerID == "" {
		if assistant := v.Session.GetAssistant(); assistant != nil && assistant.AssistantPhoneDeployment != nil {
			if did, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone"); err == nil && did != "" {
				transferConfig.CallerID = strings.TrimPrefix(did, "+")
			}
		}
	}

	targets := v.Targets
	if len(targets) == 0 {
		targets = []string{v.TargetURI}
	}

	var outboundSession *sip_runtime.Session
	var connectedTarget string
	var lastError error
	for i, target := range targets {
		attempt := i + 1
		d.logger.Debugw("SIP transfer attempt",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "total", len(targets))

		if v.OnAttempt != nil {
			v.OnAttempt(target, attempt, len(targets))
		}

		// Each target gets its own bridge timeout. Transfer policy remains here;
		// SIP infra owns only the outbound B-leg lifecycle.
		perTargetCtx, perTargetCancel := context.WithTimeout(v.Session.Context(), sip_runtime.BridgeCallTimeout)
		session, err := d.server.MakeTransferBridgeCall(perTargetCtx, &transferConfig, target, transferConfig.CallerID, sip_runtime.TransferBridgeCallOptions{
			ParentCallID:    v.Session.GetCallID(),
			Attempt:         attempt,
			TotalAttempts:   len(targets),
			Auth:            v.Session.GetAuth(),
			Assistant:       v.Session.GetAssistant(),
			ConversationID:  v.Session.GetConversationID(),
			ContextID:       v.Session.GetContextID(),
			VaultCredential: v.Session.GetVaultCredential(),
		})
		perTargetCancel()
		if err == nil {
			outboundSession = session
			connectedTarget = target
			break
		}
		lastError = err

		d.logger.Debugw("SIP transfer attempt failed",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "error", err)

		// Caller hung up or session ended — stop trying further targets.
		if v.Session.Context().Err() != nil {
			break
		}
	}

	if outboundSession == nil {
		d.logger.Warnw("SIP transfer failed",
			"call_id", v.ID,
			"targets", targets,
			"error", lastError)
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		return
	}

	v.TargetURI = connectedTarget
	outboundCallID := outboundSession.GetCallID()

	// Store outbound call ID in session metadata for observability
	v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferOutboundCallID, outboundCallID)
	d.logger.Infow("SIP transfer connected",
		"call_id", v.ID,
		"outbound_call_id", outboundCallID,
		"target", connectedTarget,
		"codec", outboundSession.GetInfo().Codec)

	if v.OnConnected != nil {
		v.OnConnected(outboundSession.GetRTPHandler())
	}

	// Track bridge duration from the moment the transfer target answered
	bridgeStart := time.Now()

	endReason, err := d.server.BridgeTransfer(ctx, v.Session, outboundSession, v.OnOperatorAudio)
	bridgeDuration := time.Since(bridgeStart)

	if err != nil {
		d.logger.Errorw("SIP transfer bridge failed",
			"call_id", v.ID,
			"target", v.TargetURI,
			"outbound_call_id", outboundCallID,
			"status", "failed",
			"bridge_duration", bridgeDuration,
			"error", err)
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferDuration, bridgeDuration.String())
	} else {
		d.logger.Infow("SIP transfer completed",
			"call_id", v.ID,
			"target", v.TargetURI,
			"outbound_call_id", outboundCallID,
			"status", "completed",
			"end_reason", endReason,
			"bridge_duration", bridgeDuration)
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "completed")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferDuration, bridgeDuration.String())
	}

	// SIP layer owns transfer transport only. Policy decisions (continue vs end_call)
	// are handled upstream via tool-result handling.
	//
	// Teardown order matters:
	//   1. OnTeardown — calls streamer.DisconnectTransferMedia which blocks until any
	//      in-flight ForwardUserAudio write to the outbound RTP has finished.
	//   2. lifecycle ends outbound session, closing outbound RTP only after
	//      channels; no streamer goroutine still holds a reference.
	//   3. OnResumeAI — bridge state is fully torn down; AI resumes.
	if v.OnTeardown != nil {
		v.OnTeardown()
	}
	if !outboundSession.IsEnded() {
		d.endCall(outboundSession, sip_runtime.LifecycleReasonTransferOutboundEnded)
	}
	if v.OnResumeAI != nil {
		v.OnResumeAI()
	}
}
