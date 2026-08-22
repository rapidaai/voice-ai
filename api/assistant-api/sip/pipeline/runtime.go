// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"
	"strconv"
	"time"

	internal_adapter "github.com/rapidaai/api/assistant-api/internal/adapters"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type sipPreparedCallRuntime struct {
	logger      commons.Logger
	session     *sip_infra.Session
	auth        *types.Authentication
	callContext *callcontext.CallContext
	talkContext context.Context
	cancelTalk  context.CancelFunc
	streamer    internal_type.SIPStreamer
	talker      internal_type.Talking
	direction   string
	talkDone    chan error
}

func (runtime *sipPreparedCallRuntime) Start(_ context.Context) error {
	if runtime.session.IsEnded() {
		runtime.logger.Warnw("Session already ended before runtime start", "call_id", runtime.session.GetCallID())
		return fmt.Errorf("session_ended_before_start")
	}
	runtime.streamer.StartAssistantOutput()
	if runtime.talkDone != nil {
		return <-runtime.talkDone
	}
	return runtime.runTalker()
}

func (runtime *sipPreparedCallRuntime) runTalker() error {
	runtime.logger.Infow("SIP call started",
		"call_id", runtime.session.GetCallID(),
		"assistant_id", runtime.callContext.AssistantID,
		"conversation_id", runtime.callContext.ConversationID,
		"direction", runtime.direction)
	if err := runtime.talker.Talk(runtime.talkContext, runtime.auth); err != nil {
		runtime.logger.Warnw("SIP talker exited", "error", err, "call_id", runtime.session.GetCallID())
	}
	runtime.logger.Infow("SIP call ended", "call_id", runtime.session.GetCallID())
	return nil
}

func (runtime *sipPreparedCallRuntime) Close(_ context.Context) {
	if runtime == nil {
		return
	}
	if runtime.cancelTalk != nil {
		runtime.cancelTalk()
	}
	_ = runtime.streamer.Close()
}

func (d *Dispatcher) prepareSIPCallRuntime(ctx context.Context, session *sip_infra.Session, setup *CallSetupResult, observer observability.Recorder, vaultCred interface{}, sipConfig *sip_infra.Config, direction, fromIdentity, toIdentity string) (*sipPreparedCallRuntime, error) {
	callID := session.GetCallID()
	if session.IsEnded() {
		d.logger.Warnw("Session already ended before call runtime preparation", "call_id", callID)
		return nil, fmt.Errorf("session_ended_before_start")
	}
	auth := session.GetAuth()
	resolvedCallContext, err := d.resolveSIPCallContext(session, setup, direction, fromIdentity, toIdentity)
	if err != nil {
		return nil, fmt.Errorf("resolve SIP call context: %w", err)
	}
	talkContext, cancelTalk := context.WithCancel(session.Context())

	go func() {
		select {
		case <-session.ByeReceived():
			cancelTalk()
		case <-talkContext.Done():
		}
	}()

	select {
	case <-session.ByeReceived():
		cancelTalk()
		d.logger.Infow("BYE received before call runtime preparation", "call_id", callID)
		return nil, fmt.Errorf("bye_before_start")
	default:
	}

	var vaultCredential *protos.VaultCredential
	if v, ok := vaultCred.(*protos.VaultCredential); ok {
		vaultCredential = v
	} else {
		vaultCredential = session.GetVaultCredential()
	}
	streamer, err := internal_telephony.New(
		internal_telephony.WithSIPContext(talkContext),
		internal_telephony.WithSIPLogger(d.logger),
		internal_telephony.WithSIPSession(session),
		internal_telephony.WithSIPLifecycle(d.server),
		internal_telephony.WithSIPCallContext(resolvedCallContext),
		internal_telephony.WithSIPVaultCredential(vaultCredential),
		internal_telephony.WithSIPObserver(observer),
	)
	if err != nil {
		cancelTalk()
		d.logger.Error("Failed to create SIP streamer", "error", err, "call_id", callID)
		return nil, fmt.Errorf("streamer_failed: %w", err)
	}
	if session.IsEnded() {
		cancelTalk()
		_ = streamer.Close()
		return nil, fmt.Errorf("session_ended_after_streamer")
	}

	d.configureSIPTransfer(ctx, session, sipConfig, resolvedCallContext, streamer)
	talker, err := internal_adapter.New(
		internal_adapter.WithSource(utils.PhoneCall),
		internal_adapter.WithContext(talkContext),
		internal_adapter.WithConfig(d.assistantConfig),
		internal_adapter.WithLogger(d.logger),
		internal_adapter.WithPostgres(d.postgres),
		internal_adapter.WithOpenSearch(d.opensearch),
		internal_adapter.WithRedis(d.redis),
		internal_adapter.WithStorage(d.storage),
		internal_adapter.WithStreamer(streamer),
		internal_adapter.WithObserver(observer),
	)
	if err != nil {
		cancelTalk()
		_ = streamer.Close()
		d.logger.Error("Failed to create SIP talker", "error", err, "call_id", callID)
		return nil, fmt.Errorf("talker_failed: %w", err)
	}
	return &sipPreparedCallRuntime{
		logger:      d.logger,
		session:     session,
		auth:        auth,
		callContext: resolvedCallContext,
		talkContext: talkContext,
		cancelTalk:  cancelTalk,
		streamer:    streamer,
		talker:      talker,
		direction:   direction,
	}, nil
}

func (runtime *sipPreparedCallRuntime) StartBeforeAnswer(ctx context.Context, timeout time.Duration) error {
	if runtime.talkDone != nil {
		return nil
	}
	runtime.talkDone = make(chan error, 1)
	go func() {
		runtime.talkDone <- runtime.runTalker()
	}()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-runtime.talkDone:
			runtime.talkDone <- err
			if err != nil {
				return err
			}
			return fmt.Errorf("SIP runtime exited before answer")
		case <-ctx.Done():
			return ctx.Err()
		case <-runtime.session.Context().Done():
			return runtime.session.Context().Err()
		case <-timer.C:
			return fmt.Errorf("SIP runtime readiness timeout")
		case <-ticker.C:
			return nil
		}
	}
}

func inboundRuntimeReadyTimeout(config *sip_infra.Config) time.Duration {
	if config != nil && config.InboundMaxRingDuration > 0 {
		return config.InboundMaxRingDuration
	}
	return 30 * time.Second
}

func (d *Dispatcher) resolveSIPCallContext(session *sip_infra.Session, setup *CallSetupResult, direction, fromIdentity, toIdentity string) (*callcontext.CallContext, error) {
	callID := session.GetCallID()
	if setup.CallContext != nil {
		call := setup.CallContext
		if call.AssistantProviderId == 0 {
			call.AssistantProviderId = setup.AssistantProviderId
		}
		if call.ProjectID == 0 {
			call.ProjectID = setup.ProjectID
		}
		if call.OrganizationID == 0 {
			call.OrganizationID = setup.OrganizationID
		}
		return call, nil
	}

	d.logger.Warnw("setup.CallContext missing - reconstructing from session", "call_id", callID)
	call := &callcontext.CallContext{
		AssistantID:         setup.AssistantID,
		ConversationID:      setup.ConversationID,
		AssistantProviderId: setup.AssistantProviderId,
		Direction:           direction,
		Provider:            "sip",
		ChannelUUID:         callID,
		ContextID:           callID,
		ProjectID:           setup.ProjectID,
		OrganizationID:      setup.OrganizationID,
	}
	if err := call.SetAuthentication(setup.Auth); err != nil {
		return nil, err
	}
	if direction == string(sip_infra.CallDirectionOutbound) {
		call.CallerNumber = toIdentity
		call.FromNumber = fromIdentity
	} else {
		call.CallerNumber = fromIdentity
		call.FromNumber = toIdentity
	}
	return call, nil
}

func (d *Dispatcher) configureSIPTransfer(ctx context.Context, session *sip_infra.Session, sipConfig *sip_infra.Config, call *callcontext.CallContext, transferStreamer internal_type.SIPTransferStreamer) {
	callID := session.GetCallID()
	transferStreamer.SetTransferRequestHandler(func(targets []string, postTransferAction string) {
		toolID, _ := session.GetMetadata("tool_id")
		toolIDStr, _ := toolID.(string)
		toolCtxID, _ := session.GetMetadata("tool_context_id")
		toolCtxIDStr, _ := toolCtxID.(string)
		primaryTarget := targets[0]
		conversationID := strconv.FormatUint(call.ConversationID, 10)
		legID := string(session.GetInfo().Direction)
		emitTransferEvent := func(eventType string, data map[string]string) {
			payload := map[string]string{
				"event_type":  eventType,
				"call_id":     callID,
				"leg_id":      legID,
				"transfer_id": toolCtxIDStr,
				"attempt_id":  "",
				"from_state":  "",
				"to_state":    "",
				"reason":      "",
				"sip_method":  "",
				"sip_status":  "",
				"target":      "",
				"duration_ms": "",
				"error":       "",
			}
			for k, v := range data {
				payload[k] = v
			}
			transferStreamer.SendTransferEvent(&protos.ConversationEvent{
				Id:   conversationID,
				Name: observability.ComponentSIP.String(),
				Data: payload,
				Time: timestamppb.Now(),
			})
		}
		d.OnPipeline(ctx, sip_infra.TransferInitiatedPipeline{
			ID:                 callID,
			Session:            session,
			TargetURI:          primaryTarget,
			Targets:            targets,
			Config:             sipConfig,
			PostTransferAction: postTransferAction,
			OnAttempt: func(target string, attempt int, total int) {
				emitTransferEvent("transfer_attempt", map[string]string{
					"type":       observability.SIPTransferring.String(),
					"provider":   "sip",
					"target":     target,
					"attempt_id": strconv.Itoa(attempt),
					"reason":     "dial_attempt",
					"total":      strconv.Itoa(total),
				})
			},
			OnConnected: func(outboundRTP *sip_infra.RTPHandler) {
				outputCodecName := ""
				if outboundRTP != nil {
					if codec := outboundRTP.GetCodec(); codec != nil {
						outputCodecName = codec.Name
					}
				}
				transferStreamer.StopTransferRingback()
				transferStreamer.ConnectTransferMedia(outboundRTP, outputCodecName)
				emitTransferEvent("transfer_connected", map[string]string{
					"target":     primaryTarget,
					"from_state": "transferring",
					"to_state":   "bridge_connected",
					"reason":     "target_answered",
				})
			},
			OnFailed: func() {
				transferStreamer.ResumeAssistant()
				emitTransferEvent("transfer_failed", map[string]string{
					"target": primaryTarget,
					"reason": "dial_failed",
					"error":  fmt.Sprintf("transfer to %s failed", primaryTarget),
				})
				if toolIDStr != "" {
					transferStreamer.SendTransferToolResult(toolCtxIDStr, toolIDStr, "transfer_call", protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, map[string]string{
						"status":      "failed",
						"reason":      fmt.Sprintf("Transfer to %s failed", primaryTarget),
						"next_action": postTransferAction,
					})
				}
			},
			OnTeardown: func() {
				transferStreamer.DisconnectTransferMedia()
				durationMs := ""
				if d, ok := session.GetMetadata(sip_infra.MetadataBridgeTransferDuration); ok {
					if duration, ok := d.(string); ok && duration != "" {
						if parsed, err := time.ParseDuration(duration); err == nil {
							durationMs = strconv.FormatInt(parsed.Milliseconds(), 10)
						}
					}
				}
				transferStreamer.RecordTransferDurationMetric(durationMs)
				emitTransferEvent("transfer_teardown", map[string]string{
					"target":      primaryTarget,
					"from_state":  "bridge_connected",
					"to_state":    "connected",
					"reason":      "bridge_end",
					"duration_ms": durationMs,
				})
				if toolIDStr != "" {
					status, _ := session.GetMetadata(sip_infra.MetadataBridgeTransferStatus)
					statusStr, _ := status.(string)
					transferStreamer.SendTransferToolResult(toolCtxIDStr, toolIDStr, "transfer_call", protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, map[string]string{
						"status":      statusStr,
						"reason":      fmt.Sprintf("Transfer to %s %s", primaryTarget, statusStr),
						"next_action": postTransferAction,
					})
				}
			},
			OnResumeAI: func() {
				transferStreamer.ResumeAssistant()
				emitTransferEvent("transfer_resume_ai", map[string]string{
					"target":     primaryTarget,
					"from_state": "bridge_connected",
					"to_state":   "connected",
					"reason":     "handoff_to_ai",
				})
			},
			OnOperatorAudio: func(audio []byte) { transferStreamer.RecordTransferOperatorAudio(audio) },
		})
	})
}
