// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"errors"
	"fmt"
)

var (
	ErrOutboundMediaNotPrepared = errors.New("outbound media is not prepared")
	ErrOutboundMediaNoSession   = errors.New("outbound media requires a session")
)

// outboundMedia prepares RTP for an outbound call and configures it before
// the session adopts the handler. Session.End owns final RTP teardown.
type outboundMedia struct {
	server  *Server
	session *Session
	request OutboundInviteRequest

	rtpHandler   *RTPHandler
	localRTPPort int
	externalIP   string
	started      bool
}

// OutboundMediaAnswer is the validated remote SDP answer from outbound 200 OK.
// It is parsed before ACK so the call can fail cleanly if media is unusable.
type OutboundMediaAnswer struct {
	negotiatedCodec *Codec
	remoteIP        string
	remotePort      int
}

// NewOutboundMedia creates the outbound RTP preparer for a SIP call.
func NewOutboundMedia(server *Server, session *Session, request OutboundInviteRequest) *outboundMedia {
	return &outboundMedia{
		server:  server,
		session: session,
		request: request,
	}
}

func (media *outboundMedia) Prepare() error {
	if media.session == nil {
		return ErrOutboundMediaNoSession
	}
	if err := media.request.Validate(); err != nil {
		return err
	}

	rtpHandler, err := NewRTPHandler(media.server.ctx, &RTPConfig{
		LocalIP:             media.server.listenConfig.GetBindAddress(),
		RTPPortRangeStart:   media.server.rtpPortRangeStart,
		RTPPortRangeEnd:     media.server.rtpPortRangeEnd,
		PayloadType:         CodecPCMU.PayloadType,
		ClockRate:           CodecPCMU.ClockRate,
		Logger:              media.server.logger,
		MediaTimeoutInitial: media.request.Config.MediaTimeoutInitial,
		MediaTimeout:        media.request.Config.MediaTimeout,
		SymmetricRTP:        media.server.symmetricRTP,
		portStats:           media.server.rtpPortStats,
	})
	if err != nil {
		return err
	}

	_, media.localRTPPort = rtpHandler.LocalAddr()
	media.rtpHandler = rtpHandler
	media.externalIP = media.server.listenConfig.GetExternalIP()
	media.session.SetLocalRTP(media.externalIP, media.localRTPPort)
	media.session.SetRTPHandler(rtpHandler)
	return nil
}

func (media *outboundMedia) SDPOffer() (string, error) {
	if media.rtpHandler == nil {
		return "", ErrOutboundMediaNotPrepared
	}
	return media.server.GenerateSDP(DefaultSDPConfig(media.externalIP, media.localRTPPort)), nil
}

func NewOutboundMediaAnswer(server *Server, dialog *outboundDialog) (OutboundMediaAnswer, error) {
	if dialog == nil || dialog.InviteResponse() == nil {
		return OutboundMediaAnswer{}, fmt.Errorf("%w: outbound 200 OK response is missing", ErrSDPParseFailed)
	}

	body := dialog.InviteResponse().Body()
	if len(body) == 0 {
		return OutboundMediaAnswer{}, fmt.Errorf("%w: outbound 200 OK SDP body is missing", ErrSDPParseFailed)
	}

	if server.logger != nil {
		server.logger.Debugw("Outbound call 200 OK SDP answer",
			"call_id", dialog.session.GetCallID(),
			"sdp_body", string(body))
	}

	sdpInfo, err := server.ParseSDP(body)
	if err != nil {
		return OutboundMediaAnswer{}, fmt.Errorf("%w: %v", ErrSDPParseFailed, err)
	}
	if sdpInfo.ConnectionIP == "" || sdpInfo.AudioPort <= 0 {
		return OutboundMediaAnswer{}, fmt.Errorf("%w: outbound answer missing RTP address", ErrSDPParseFailed)
	}
	if sdpInfo.PreferredCodec == nil {
		return OutboundMediaAnswer{}, fmt.Errorf("%w: outbound answer payload types %v", ErrCodecNotSupported, sdpInfo.PayloadTypes)
	}

	return OutboundMediaAnswer{
		negotiatedCodec: sdpInfo.PreferredCodec,
		remoteIP:        sdpInfo.ConnectionIP,
		remotePort:      sdpInfo.AudioPort,
	}, nil
}

func (media *outboundMedia) ApplyAnswer(answer OutboundMediaAnswer) error {
	if media.rtpHandler == nil {
		return ErrOutboundMediaNotPrepared
	}
	if media.server != nil {
		media.rtpHandler.SetSymmetricRTP(media.server.useSymmetricRTPForRemoteIP(answer.remoteIP))
	}
	media.rtpHandler.SetRemoteAddr(answer.remoteIP, answer.remotePort)
	media.rtpHandler.SetCodec(answer.negotiatedCodec)
	media.session.SetRemoteRTP(answer.remoteIP, answer.remotePort)
	if answer.negotiatedCodec != nil {
		media.session.SetNegotiatedCodec(answer.negotiatedCodec.Name, int(answer.negotiatedCodec.ClockRate))
	}
	return nil
}

func (media *outboundMedia) Start(onMediaTimeout func()) error {
	if media.rtpHandler == nil {
		return ErrOutboundMediaNotPrepared
	}
	if media.started {
		return nil
	}
	media.rtpHandler.Start()
	media.rtpHandler.EnableMediaTimeout(true)

	mediaTimeout := media.rtpHandler.MediaTimeout()
	if media.session != nil && mediaTimeout != nil && onMediaTimeout != nil {
		sessionContext := media.session.Context()
		go func() {
			select {
			case <-sessionContext.Done():
				return
			case <-mediaTimeout:
			}
			onMediaTimeout()
		}()
	}

	media.started = true
	return nil
}

func (media *outboundMedia) Stop() {
	if media == nil || media.rtpHandler == nil {
		return
	}
	rtpHandler := media.rtpHandler
	media.rtpHandler = nil
	media.started = false
	if media.session != nil && media.session.GetRTPHandler() == rtpHandler {
		return
	}
	_ = rtpHandler.Stop()
}

func (media *outboundMedia) LocalAddr() (string, int) {
	if media == nil || media.rtpHandler == nil {
		return "", 0
	}
	return media.rtpHandler.LocalAddr()
}

func (media *outboundMedia) RemoteAddrConfigured() bool {
	if media == nil || media.rtpHandler == nil {
		return false
	}
	return media.rtpHandler.GetRemoteAddr() != nil
}
