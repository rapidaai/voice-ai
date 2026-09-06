// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

// inboundMedia prepares RTP for an inbound INVITE and configures it before
// the session adopts the handler. Session.End owns final RTP teardown.
type inboundMedia struct {
	server     *Server
	session    *Session
	mediaOffer inboundMediaOffer

	rtpHandler   *RTPHandler
	localAddress RTPAddress
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
		LocalAddress:        RTPAddress{IP: media.server.listenConfig.GetBindAddress()},
		RTPPortRangeStart:   media.server.rtpPortRangeStart,
		RTPPortRangeEnd:     media.server.rtpPortRangeEnd,
		PayloadType:         media.mediaOffer.negotiatedCodec.PayloadType,
		ClockRate:           media.mediaOffer.negotiatedCodec.ClockRate,
		MediaTimeoutInitial: media.session.config.MediaTimeoutInitial,
		MediaTimeout:        media.session.config.MediaTimeout,
		PacketizationTime:   media.mediaOffer.sdpInfo.PacketizationDuration(),
		SymmetricRTP:        media.server.useSymmetricRTPForRemoteIP(media.mediaOffer.sdpInfo.RTPAddress.IP),
		portStats:           media.server.rtpPortStats,
	})
	if err != nil {
		return err
	}

	localRTPAddress := rtpHandler.LocalAddress()
	rtpHandler.setRemoteMediaAddress(remoteMediaAddress{
		remoteRTPAddress:  media.mediaOffer.sdpInfo.RTPAddress,
		remoteRTCPAddress: media.mediaOffer.sdpInfo.RTCPAddress,
	})
	rtpHandler.SetOnFirstPacket(func() {
		if media.session != nil {
			media.session.MarkInboundFirstRTPReceived()
		}
	})

	media.rtpHandler = rtpHandler
	media.localAddress = RTPAddress{IP: media.server.listenConfig.GetExternalIP(), Port: localRTPAddress.Port}

	media.session.SetRemoteRTPAddress(media.mediaOffer.sdpInfo.RTPAddress)
	media.session.SetLocalRTPAddress(media.localAddress)
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
	config := media.server.NegotiatedSDPConfig(media.localAddress, media.mediaOffer.negotiatedCodec)
	if media.rtpHandler != nil {
		config.RTCPPort = media.rtpHandler.LocalRTCPPort()
	}
	config.PTime = sdpDefaultPTimeMS
	return config
}
