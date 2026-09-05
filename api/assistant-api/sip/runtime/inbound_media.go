// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"errors"
)

var (
	ErrInboundMediaNotPrepared = errors.New("inbound media is not prepared")
	ErrInboundMediaNoSession   = errors.New("inbound media requires a session")
)

// inboundMedia prepares RTP for an inbound INVITE and configures it before
// the session adopts the handler. Session.End owns final RTP teardown.
type inboundMedia struct {
	server     *Server
	session    *Session
	mediaOffer inboundMediaOffer

	rtpHandler   *RTPHandler
	localRTPPort int
	externalIP   string
	started      bool
}

// newInboundMedia creates the inbound media preparer for a SIP INVITE.
func newInboundMedia(server *Server, session *Session, mediaOffer inboundMediaOffer) *inboundMedia {
	return &inboundMedia{
		server:     server,
		session:    session,
		mediaOffer: mediaOffer,
	}
}

func (media *inboundMedia) Prepare() error {
	if media.session == nil {
		return ErrInboundMediaNoSession
	}

	rtpHandler, err := NewRTPHandler(media.server.ctx, &RTPConfig{
		LocalIP:             media.server.listenConfig.GetBindAddress(),
		RTPPortRangeStart:   media.server.rtpPortRangeStart,
		RTPPortRangeEnd:     media.server.rtpPortRangeEnd,
		PayloadType:         media.mediaOffer.negotiatedCodec.PayloadType,
		ClockRate:           media.mediaOffer.negotiatedCodec.ClockRate,
		MediaTimeoutInitial: media.session.config.MediaTimeoutInitial,
		MediaTimeout:        media.session.config.MediaTimeout,
		SymmetricRTP:        media.server.useSymmetricRTPForRemoteIP(media.mediaOffer.sdpInfo.ConnectionIP),
		portStats:           media.server.rtpPortStats,
	})
	if err != nil {
		return err
	}

	_, media.localRTPPort = rtpHandler.LocalAddr()
	rtpHandler.SetRemoteAddr(media.mediaOffer.sdpInfo.ConnectionIP, media.mediaOffer.sdpInfo.AudioPort)
	if media.mediaOffer.sdpInfo.RTCPPort > 0 {
		rtcpIP := media.mediaOffer.sdpInfo.RTCPIP
		if rtcpIP == "" {
			rtcpIP = media.mediaOffer.sdpInfo.ConnectionIP
		}
		rtpHandler.SetRemoteRTCPAddr(rtcpIP, media.mediaOffer.sdpInfo.RTCPPort)
	}
	rtpHandler.SetCodec(media.mediaOffer.negotiatedCodec)
	rtpHandler.SetOnFirstPacket(func() {
		if media.session != nil {
			media.session.MarkInboundFirstRTPReceived()
		}
	})

	media.rtpHandler = rtpHandler
	media.externalIP = media.server.listenConfig.GetExternalIP()

	media.session.SetRemoteRTP(media.mediaOffer.sdpInfo.ConnectionIP, media.mediaOffer.sdpInfo.AudioPort)
	media.session.SetLocalRTP(media.externalIP, media.localRTPPort)
	media.session.SetNegotiatedCodec(media.mediaOffer.negotiatedCodec.Name, int(media.mediaOffer.negotiatedCodec.ClockRate))
	media.session.SetRTPHandler(rtpHandler)
	return nil
}

func (media *inboundMedia) Start(onMediaTimeout func()) error {
	if media.rtpHandler == nil {
		return ErrInboundMediaNotPrepared
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

func (media *inboundMedia) SDPConfig() *SDPConfig {
	config := media.server.NegotiatedSDPConfig(
		media.externalIP,
		media.localRTPPort,
		media.mediaOffer.negotiatedCodec,
	)
	if media.rtpHandler != nil {
		config.RTCPPort = media.rtpHandler.LocalRTCPPort()
	}
	return config
}
