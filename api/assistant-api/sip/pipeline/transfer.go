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

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
)

func (d *Dispatcher) handleTransferInitiated(ctx context.Context, v sip_infra.TransferInitiatedPipeline) {
	if v.RoutingMode == "" {
		v.RoutingMode = "sequential"
	}
	if v.TransferID == "" {
		v.TransferID = fmt.Sprintf("transfer:%s", v.ID)
	}
	targets := v.Targets
	if len(targets) == 0 && v.TargetURI != "" {
		targets = []string{v.TargetURI}
	}
	d.OnPipeline(ctx, sip_infra.TransferRequestedPipeline{
		ID:                 v.ID,
		Session:            v.Session,
		TransferID:         v.TransferID,
		Targets:            targets,
		RoutingMode:        v.RoutingMode,
		PostTransferAction: v.PostTransferAction,
	})
	go d.executeTransfer(ctx, v)
}

func (d *Dispatcher) executeTransfer(ctx context.Context, v sip_infra.TransferInitiatedPipeline) {
	if v.RoutingMode == "" {
		v.RoutingMode = "sequential"
	}
	if v.TransferID == "" {
		v.TransferID = fmt.Sprintf("transfer:%s", v.ID)
	}
	d.logger.Infow("Pipeline: transfer_initiated",
		"call_id", v.ID, "target", v.TargetURI)

	if d.server == nil {
		d.logger.Errorw("Pipeline: transfer_failed — SIP server not available",
			"call_id", v.ID, "target", v.TargetURI, "reason", "server_nil")
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		d.OnPipeline(ctx, sip_infra.TransferFailedPipeline{
			ID:          v.ID,
			Session:     v.Session,
			TransferID:  v.TransferID,
			RoutingMode: v.RoutingMode,
			Error:       fmt.Errorf("sip server unavailable"),
			Reason:      "server_nil",
		})
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

	var outboundSession *sip_infra.Session
	var connectedTarget string
	for i, target := range targets {
		attempt := i + 1
		d.logger.Infow("Pipeline: transfer_attempt",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "total", len(targets))

		if v.OnAttempt != nil {
			v.OnAttempt(target, attempt, len(targets))
		}
		d.OnPipeline(ctx, sip_infra.TransferAttemptStartedPipeline{
			ID:          v.ID,
			Session:     v.Session,
			TransferID:  v.TransferID,
			TargetURI:   target,
			Attempt:     attempt,
			Total:       len(targets),
			RoutingMode: v.RoutingMode,
		})
		// TODO: emit TransferTargetRingingPipeline when bridge dialing exposes
		// a reliable per-target ringing callback from SIP 180/183 progress.

		// Each target gets its own BridgeCallTimeout. The overall budget is
		// bounded by the inbound session context — if the caller hangs up
		// mid-failover, we stop trying.
		perTargetCtx, perTargetCancel := context.WithTimeout(v.Session.Context(), sip_infra.BridgeCallTimeout)
		session, err := d.server.MakeBridgeCall(perTargetCtx, cfg, target, cfg.CallerID)
		perTargetCancel()
		if err == nil {
			outboundSession = session
			connectedTarget = target
			d.OnPipeline(ctx, sip_infra.TransferAttemptEndedPipeline{
				ID:             v.ID,
				Session:        v.Session,
				TransferID:     v.TransferID,
				AttemptID:      fmt.Sprintf("%s:%d", v.TransferID, attempt),
				TargetURI:      target,
				OutboundCallID: session.GetCallID(),
				Attempt:        attempt,
				Total:          len(targets),
				RoutingMode:    v.RoutingMode,
				State:          "connected",
				Reason:         "connected",
			})
			break
		}

		d.logger.Warnw("Pipeline: transfer_target_failed",
			"call_id", v.ID, "target", target,
			"attempt", attempt, "error", err)

		d.OnPipeline(ctx, sip_infra.EventEmittedPipeline{
			ID:    v.ID,
			Event: "transfer_target_failed",
			Data: map[string]string{
				"target":  target,
				"attempt": fmt.Sprintf("%d", attempt),
				"error":   err.Error(),
			},
		})
		attemptState, attemptReason := classifyTransferAttemptFailure(err)
		d.OnPipeline(ctx, sip_infra.TransferAttemptEndedPipeline{
			ID:          v.ID,
			Session:     v.Session,
			TransferID:  v.TransferID,
			AttemptID:   fmt.Sprintf("%s:%d", v.TransferID, attempt),
			TargetURI:   target,
			Attempt:     attempt,
			Total:       len(targets),
			RoutingMode: v.RoutingMode,
			State:       attemptState,
			Reason:      attemptReason,
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
		if attemptState == "cancelled" {
			d.OnPipeline(ctx, sip_infra.TransferCancelledPipeline{
				ID:          v.ID,
				Session:     v.Session,
				TransferID:  v.TransferID,
				TargetURI:   target,
				Attempt:     attempt,
				Total:       len(targets),
				RoutingMode: v.RoutingMode,
				Reason:      attemptReason,
			})
		}

		// Caller hung up or session ended — stop trying further targets.
		if v.Session.Context().Err() != nil {
			break
		}
	}

	if outboundSession == nil {
		d.logger.Errorw("Pipeline: transfer_failed — all targets exhausted",
			"call_id", v.ID, "targets", targets)
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferStatus, "failed")
		if v.OnFailed != nil {
			v.OnFailed()
		}
		d.OnPipeline(ctx, sip_infra.TransferFailedPipeline{
			ID:          v.ID,
			Session:     v.Session,
			TransferID:  v.TransferID,
			RoutingMode: v.RoutingMode,
			Error:       fmt.Errorf("all %d transfer targets failed", len(targets)),
			Reason:      "outbound_failed",
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
	v.Session.SetMetadata(sip_infra.MetadataBridgeTransferOutboundCallID, outboundCallID)

	if v.OnConnected != nil {
		v.OnConnected(outboundSession.GetRTPHandler())
	}

	v.Session.SetState(sip_infra.CallStateBridgeConnected)

	d.OnPipeline(ctx, sip_infra.TransferConnectedPipeline{
		ID:              v.ID,
		InboundSession:  v.Session,
		OutboundSession: outboundSession,
		TargetURI:       connectedTarget,
		Attempt:         indexOfTarget(targets, connectedTarget) + 1,
		TotalAttempts:   len(targets),
		TransferID:      v.TransferID,
		RoutingMode:     v.RoutingMode,
	})
	connectedIndex := indexOfTarget(targets, connectedTarget)
	for i, target := range targets {
		if target == connectedTarget {
			continue
		}
		if v.RoutingMode != "parallel" && connectedIndex >= 0 && i < connectedIndex {
			continue
		}
		d.OnPipeline(ctx, sip_infra.TransferCancelledPipeline{
			ID:          v.ID,
			Session:     v.Session,
			TransferID:  v.TransferID,
			TargetURI:   target,
			Attempt:     i + 1,
			Total:       len(targets),
			RoutingMode: v.RoutingMode,
			Reason:      reasonAnsweredOther,
			AnsweredBy:  connectedTarget,
		})
	}

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
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferStatus, "failed")
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferDuration, bridgeDuration.String())
	} else {
		d.logger.Infow("Pipeline: transfer_completed",
			"call_id", v.ID,
			"target", v.TargetURI,
			"outbound_call_id", outboundCallID,
			"status", "completed",
			"end_reason", endReason,
			"bridge_duration", bridgeDuration)
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferStatus, "completed")
		v.Session.SetMetadata(sip_infra.MetadataBridgeTransferDuration, bridgeDuration.String())
	}

	// SIP layer owns transfer transport only. Policy decisions (continue vs end_call)
	// are handled upstream via tool-result handling.
	//
	// Teardown order matters:
	//   1. OnTeardown — calls streamer.ClearBridgeTarget which blocks until any
	//      in-flight ForwardUserAudio write to the outbound RTP has finished.
	//   2. outboundSession.End() — only now safe to close the outbound RTP
	//      channels; no streamer goroutine still holds a reference.
	//   3. OnResumeAI — bridge state is fully torn down; AI resumes.
	if v.OnTeardown != nil {
		v.OnTeardown()
	}
	if !outboundSession.IsEnded() {
		outboundSession.End()
	}
	if v.OnResumeAI != nil {
		v.OnResumeAI()
	}
}

func indexOfTarget(targets []string, target string) int {
	for i, t := range targets {
		if t == target {
			return i
		}
	}
	return -1
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
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
				return "network"
			}
			if strings.Contains(errMsg, "486") || strings.Contains(errMsg, "600") ||
				strings.Contains(errMsg, "busy") {
				return "rejected"
			}
			if strings.Contains(errMsg, "603") || strings.Contains(errMsg, "403") ||
				strings.Contains(errMsg, "declined") || strings.Contains(errMsg, "rejected") {
				return "rejected"
			}
			if strings.Contains(errMsg, "480") || strings.Contains(errMsg, "408") ||
				strings.Contains(errMsg, "no answer") || strings.Contains(errMsg, "unavailable") {
				return "network"
			}
		}
		return "network"
	case reason == "bridge_failed":
		return "bridge"
	default:
		return "unknown"
	}
}
