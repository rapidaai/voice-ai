// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
)

func (d *Dispatcher) handleTransferInitiated(ctx context.Context, v sip_runtime.TransferInitiatedPipeline) {
	go d.executeTransfer(ctx, v)
}

func (d *Dispatcher) executeTransfer(ctx context.Context, v sip_runtime.TransferInitiatedPipeline) {
	d.logger.Infow("Pipeline: transfer_initiated",
		"call_id", v.ID, "target", v.TargetURI)

	if d.server == nil {
		d.logger.Errorw("Pipeline: transfer_failed — SIP server not available",
			"call_id", v.ID, "target", v.TargetURI, "reason", "server_nil")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		return
	}

	cfg := v.Config
	if cfg == nil {
		cfg = v.Session.GetConfig()
	}

	if cfg.CallerID == "" {
		if assistant := v.Session.GetAssistant(); assistant != nil && assistant.AssistantPhoneDeployment != nil {
			if did, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone"); err == nil && did != "" {
				cfg.CallerID = strings.TrimPrefix(did, "+")
			}
		}
	}

	targets := v.Targets
	if len(targets) == 0 {
		targets = []string{v.TargetURI}
	}

	var outboundSession *sip_runtime.Session
	var connectedTarget string
	for i, target := range targets {
		attempt := i + 1
		d.logger.Infow("Pipeline: transfer_attempt",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "total", len(targets))

		if v.OnAttempt != nil {
			v.OnAttempt(target, attempt, len(targets))
		}

		// Each target gets its own bridge timeout. Transfer policy remains here;
		// SIP infra owns only the outbound B-leg lifecycle.
		perTargetCtx, perTargetCancel := context.WithTimeout(v.Session.Context(), sip_runtime.BridgeCallTimeout)
		session, err := d.server.MakeTransferBridgeCall(perTargetCtx, cfg, target, cfg.CallerID, sip_runtime.TransferBridgeCallOptions{
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

		d.logger.Warnw("Pipeline: transfer_target_failed",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "error", err)

		// Caller hung up or session ended — stop trying further targets.
		if v.Session.Context().Err() != nil {
			break
		}
	}

	if outboundSession == nil {
		d.logger.Errorw("Pipeline: transfer_failed — all targets exhausted",
			"call_id", v.ID, "targets", targets)
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		d.OnPipeline(ctx, sip_runtime.TransferFailedPipeline{
			ID:     v.ID,
			Error:  fmt.Errorf("all %d transfer targets failed", len(targets)),
			Reason: "outbound_failed",
		})
		return
	}

	v.TargetURI = connectedTarget
	outboundCallID := outboundSession.GetCallID()

	d.logger.Infow("Pipeline: transfer_connected",
		"call_id", v.ID,
		"outbound_call_id", outboundCallID,
		"target", v.TargetURI)

	// Store outbound call ID in session metadata for observability
	v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferOutboundCallID, outboundCallID)

	if v.OnConnected != nil {
		v.OnConnected(outboundSession.GetRTPHandler())
	}

	d.OnPipeline(ctx, sip_runtime.TransferConnectedPipeline{
		ID:              v.ID,
		InboundSession:  v.Session,
		OutboundSession: outboundSession,
	})

	// Track bridge duration from the moment the transfer target answered
	bridgeStart := time.Now()

	endReason, err := d.server.BridgeTransfer(ctx, v.Session, outboundSession, v.OnOperatorAudio)
	bridgeDuration := time.Since(bridgeStart)

	if err != nil {
		d.logger.Errorw("Pipeline: transfer_completed — bridge failed",
			"call_id", v.ID,
			"target", v.TargetURI,
			"outbound_call_id", outboundCallID,
			"status", "failed",
			"bridge_duration", bridgeDuration,
			"error", err)
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferStatus, "failed")
		v.Session.SetMetadata(sip_runtime.MetadataBridgeTransferDuration, bridgeDuration.String())
	} else {
		d.logger.Infow("Pipeline: transfer_completed",
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

func (d *Dispatcher) handleTransferConnected(ctx context.Context, v sip_runtime.TransferConnectedPipeline) {
	outboundInfo := v.OutboundSession.GetInfo()
	d.logger.Infow("Pipeline: transfer_connected",
		"call_id", v.ID,
		"outbound_call_id", v.OutboundSession.GetCallID(),
		"target_uri", outboundInfo.RemoteURI,
		"codec", outboundInfo.Codec)
}

func (d *Dispatcher) handleTransferFailed(ctx context.Context, v sip_runtime.TransferFailedPipeline) {
	// Categorize the failure for structured alerting
	category := categorizeTransferError(v.Reason, v.Error)
	d.logger.Warnw("Pipeline: transfer_failed",
		"call_id", v.ID,
		"reason", v.Reason,
		"category", category,
		"error", v.Error)
}

// categorizeTransferError maps raw transfer failure reasons into high-level
// categories for structured logging and alerting. Categories:
//   - "setup": server unavailable or config errors before dialing
//   - "network": outbound call could not be placed (timeout, DNS, network)
//   - "rejected": callee rejected the call (busy, declined)
//   - "bridge": bridge was established but broke during media relay
//   - "unknown": could not determine category
func categorizeTransferError(reason string, err error) string {
	switch {
	case reason == "server_nil" || reason == "config_error":
		return "setup"
	case reason == "outbound_failed":
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
				return "network"
			}
			if strings.Contains(errMsg, "486") || strings.Contains(errMsg, "603") ||
				strings.Contains(errMsg, "busy") || strings.Contains(errMsg, "declined") {
				return "rejected"
			}
		}
		return "network"
	case reason == "bridge_failed":
		return "bridge"
	default:
		return "unknown"
	}
}
