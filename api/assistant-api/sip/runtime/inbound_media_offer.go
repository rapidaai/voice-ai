// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"net"
	"strings"

	"github.com/emiago/sipgo/sip"
)

// inboundMediaOffer is the validated remote SDP offer for inbound media.
// It owns the negotiated codec selected from the caller's payload list.
type inboundMediaOffer struct {
	sdpInfo         *SDPMediaInfo
	negotiatedCodec *Codec
}

// newInboundMediaOffer validates the remote SDP offer and negotiates audio.
// Call setup receives inboundFailure so it can reject SIP with the right class.
func newInboundMediaOffer(
	server *Server,
	request *sip.Request,
	requestName string,
	lifecycleReason LifecycleReason,
	allowDisabledRTPAddress bool,
) (inboundMediaOffer, *inboundFailure) {
	if len(request.Body()) > 0 {
		contentTypeHeader := request.GetHeader("Content-Type")
		if contentTypeHeader == nil {
			err := fmt.Errorf("%w: %s missing Content-Type", ErrSDPParseFailed, requestName)
			return inboundMediaOffer{}, &inboundFailure{
				statusCode:      415,
				class:           inboundFailureUnsupportedMedia,
				responseClass:   inboundFailureMedia,
				reason:          err.Error(),
				termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"},
				lifecycleReason: lifecycleReason,
				err:             err,
			}
		}
		contentType := strings.ToLower(strings.TrimSpace(contentTypeHeader.Value()))
		if semicolon := strings.Index(contentType, ";"); semicolon >= 0 {
			contentType = strings.TrimSpace(contentType[:semicolon])
		}
		if contentType != sdpContentType {
			err := fmt.Errorf("%w: unsupported %s media type %q", ErrSDPParseFailed, requestName, contentTypeHeader.Value())
			return inboundMediaOffer{}, &inboundFailure{
				statusCode:      415,
				class:           inboundFailureUnsupportedMedia,
				responseClass:   inboundFailureMedia,
				reason:          err.Error(),
				termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"},
				lifecycleReason: lifecycleReason,
				err:             err,
			}
		}
	}

	sdpInfo, err := server.ParseSDP(request.Body())
	if err != nil {
		err = fmt.Errorf("%w: %v", ErrSDPParseFailed, err)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      400,
			class:           inboundFailureMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_sdp_error"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}

	if sdpInfo == nil {
		err := fmt.Errorf("%w: inbound offer missing SDP media", ErrSDPParseFailed)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      400,
			class:           inboundFailureMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_sdp_error"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}
	if strings.TrimSpace(sdpInfo.ConnectionIP) == "" {
		err := fmt.Errorf("%w: inbound offer missing RTP address", ErrSDPParseFailed)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      400,
			class:           inboundFailureMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_sdp_error"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}
	remoteIP := net.ParseIP(sdpInfo.ConnectionIP)
	if remoteIP == nil {
		err := fmt.Errorf("%w: inbound offer invalid RTP address %q", ErrSDPParseFailed, sdpInfo.ConnectionIP)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      400,
			class:           inboundFailureMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_sdp_error"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}
	if remoteIP.IsUnspecified() && !allowDisabledRTPAddress {
		err := fmt.Errorf("%w: inbound offer disabled RTP address %q", ErrSDPParseFailed, sdpInfo.ConnectionIP)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      488,
			class:           inboundFailureUnsupportedMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}
	if sdpInfo.AudioPort <= 0 || sdpInfo.AudioPort > 65535 {
		err := fmt.Errorf("%w: inbound offer invalid RTP port %d", ErrSDPParseFailed, sdpInfo.AudioPort)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      400,
			class:           inboundFailureMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_sdp_error"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}
	if len(sdpInfo.PayloadTypes) == 0 {
		err := fmt.Errorf("%w: inbound offer has no RTP payload types", ErrCodecNotSupported)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      488,
			class:           inboundFailureUnsupportedMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}

	if sdpInfo.PreferredCodec == nil {
		err := fmt.Errorf("%w: %s payload types %v", ErrCodecNotSupported, requestName, sdpInfo.PayloadTypes)
		return inboundMediaOffer{}, &inboundFailure{
			statusCode:      488,
			class:           inboundFailureUnsupportedMedia,
			responseClass:   inboundFailureMedia,
			reason:          err.Error(),
			termination:     CallTermination{Result: CallTerminationClientError, Reason: "inbound_unsupported_media"},
			lifecycleReason: lifecycleReason,
			err:             err,
		}
	}

	return inboundMediaOffer{
		sdpInfo:         sdpInfo,
		negotiatedCodec: sdpInfo.PreferredCodec,
	}, nil
}
