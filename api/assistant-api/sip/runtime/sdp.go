// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

// Codec represents an audio codec with its RTP configuration
type Codec struct {
	Name        string
	PayloadType uint8
	ClockRate   uint32
	Channels    int
}

// Common codecs used in telephony
var (
	CodecPCMU = Codec{Name: "PCMU", PayloadType: 0, ClockRate: 8000, Channels: 1}
	CodecPCMA = Codec{Name: "PCMA", PayloadType: 8, ClockRate: 8000, Channels: 1}
	CodecG722 = Codec{Name: "G722", PayloadType: 9, ClockRate: 8000, Channels: 1}

	// CodecTelephoneEvent is RFC 4733 DTMF telephone-event.
	// Nearly all SIP endpoints (Asterisk, FreeSWITCH, Twilio, Zoiper) require
	// this in the SDP offer/answer or they report "remote codecs: None" and
	// refuse to bridge media. It MUST be advertised even if we never send DTMF.
	CodecTelephoneEvent = Codec{Name: "telephone-event", PayloadType: 101, ClockRate: 8000, Channels: 1}
)

// SupportedCodecs lists audio codecs in order of preference (excludes telephone-event)
var SupportedCodecs = []Codec{CodecPCMU, CodecPCMA}

// SDPDirection represents the media direction attribute in SDP
type SDPDirection string

// SDPMediaInfo contains parsed media information from SDP
type SDPMediaInfo struct {
	ConnectionIP   string
	AudioPort      int
	RTCPIP         string
	RTCPPort       int
	PayloadTypes   []uint8
	PreferredCodec *Codec
	PTime          int
	Direction      SDPDirection // sendrecv, sendonly, recvonly, inactive
}

// IsHold returns true if the SDP indicates a hold condition.
// Hold is signalled by: direction=sendonly/inactive, or connection IP 0.0.0.0 (RFC 3264)
func (s *SDPMediaInfo) IsHold() bool {
	if s == nil {
		return false
	}
	if s.Direction == SDPDirectionSendOnly || s.Direction == SDPDirectionInactive {
		return true
	}
	if s.ConnectionIP == "0.0.0.0" {
		return true
	}
	return false
}

func (s *SDPMediaInfo) PacketizationDuration() time.Duration {
	if s == nil || !validator.Between(s.PTime, sdpMinPTimeMS, sdpMaxPTimeMS) {
		return rtpDefaultPacketizationTime
	}
	return time.Duration(s.PTime) * time.Millisecond
}

// SDPConfig holds configuration for SDP generation
type SDPConfig struct {
	SessionID   string
	SessionName string
	LocalIP     string
	RTPPort     int
	RTCPPort    int
	Codecs      []Codec
	PTime       int // Packetization time in milliseconds
}

// DefaultSDPConfig returns a default SDP configuration
func DefaultSDPConfig(localIP string, rtpPort int) *SDPConfig {
	return &SDPConfig{
		SessionID:   "0",
		SessionName: "Rapida Voice AI",
		LocalIP:     localIP,
		RTPPort:     rtpPort,
		Codecs:      SupportedCodecs,
		PTime:       20,
	}
}

// NegotiatedSDPConfig returns an SDP configuration advertising only the
// negotiated codec. This MUST be used for re-INVITE / UPDATE responses
// and any post-answer SDP where a codec has already been agreed upon.
// Advertising multiple codecs in a response confuses some PBXes (Asterisk,
// FreeSWITCH) because it looks like a new offer instead of a confirmation.
func (s *Server) NegotiatedSDPConfig(localIP string, rtpPort int, codec *Codec) *SDPConfig {
	if codec == nil {
		codec = &CodecPCMU
	}
	return &SDPConfig{
		SessionID:   "0",
		SessionName: "Rapida Voice AI",
		LocalIP:     localIP,
		RTPPort:     rtpPort,
		Codecs:      []Codec{*codec},
		PTime:       20,
	}
}

// GenerateSDP creates an SDP body for SIP responses.
// Always includes telephone-event (PT 101) per RFC 4733. Nearly all SIP
// endpoints (Asterisk, FreeSWITCH, Zoiper, Twilio) require telephone-event
// in the m= line. Without it, they report "remote codecs: None" and refuse
// to bridge/accept media even when audio codecs match.
func (s *Server) GenerateSDP(cfg *SDPConfig) string {
	var sb strings.Builder

	// Version
	sb.WriteString("v=0\r\n")

	// Origin: o=<username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
	sb.WriteString(fmt.Sprintf("o=rapida %s 0 IN IP4 %s\r\n", cfg.SessionID, cfg.LocalIP))

	// Session Name
	sb.WriteString(fmt.Sprintf("s=%s\r\n", cfg.SessionName))

	// Connection Information
	sb.WriteString(fmt.Sprintf("c=IN IP4 %s\r\n", cfg.LocalIP))

	// Time (0 0 = session is permanent)
	sb.WriteString("t=0 0\r\n")

	// Media description: build payload type list with audio codecs and telephone-event (101)
	payloadTypes := make([]string, 0, len(cfg.Codecs)+1)
	for _, codec := range cfg.Codecs {
		payloadTypes = append(payloadTypes, strconv.Itoa(int(codec.PayloadType)))
	}
	// Always include telephone-event PT 101 in the m= line (RFC 4733 DTMF).
	// Check it's not already in the codec list to avoid duplicates.
	hasTelEvent := false
	for _, codec := range cfg.Codecs {
		if codec.PayloadType == CodecTelephoneEvent.PayloadType {
			hasTelEvent = true
			break
		}
	}
	if !hasTelEvent {
		payloadTypes = append(payloadTypes, strconv.Itoa(int(CodecTelephoneEvent.PayloadType)))
	}
	sb.WriteString(fmt.Sprintf("m=audio %d RTP/AVP %s\r\n", cfg.RTPPort, strings.Join(payloadTypes, " ")))
	if cfg.RTCPPort > 0 {
		sb.WriteString(fmt.Sprintf("a=rtcp:%d IN IP4 %s\r\n", cfg.RTCPPort, cfg.LocalIP))
	}

	// Codec attributes (rtpmap for each audio codec)
	for _, codec := range cfg.Codecs {
		sb.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\r\n", codec.PayloadType, codec.Name, codec.ClockRate))
	}

	// telephone-event rtpmap + fmtp (required by Asterisk, Zoiper, etc.)
	if !hasTelEvent {
		sb.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\r\n",
			CodecTelephoneEvent.PayloadType, CodecTelephoneEvent.Name, CodecTelephoneEvent.ClockRate))
		sb.WriteString(fmt.Sprintf("a=fmtp:%d 0-16\r\n", CodecTelephoneEvent.PayloadType))
	}

	// Packetization time
	sb.WriteString(fmt.Sprintf("a=ptime:%d\r\n", cfg.PTime))

	// Direction (send and receive)
	sb.WriteString("a=sendrecv\r\n")

	return sb.String()
}

// ParseSDP extracts media information from an SDP body
func (s *Server) ParseSDP(sdpBody []byte) (*SDPMediaInfo, error) {
	if len(sdpBody) == 0 {
		return nil, fmt.Errorf("empty SDP body")
	}

	info := &SDPMediaInfo{
		PayloadTypes: make([]uint8, 0),
		PTime:        sdpDefaultPTimeMS,
		Direction:    SDPDirectionSendRecv, // default per RFC 3264
	}

	sdpStr := string(sdpBody)
	lines := strings.Split(sdpStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, "\r")

		switch {
		case strings.HasPrefix(line, sdpConnectionIPPrefix):
			// Connection line: c=IN IP4 192.168.1.5
			info.ConnectionIP = strings.TrimSpace(strings.TrimPrefix(line, sdpConnectionIPPrefix))

		case strings.HasPrefix(line, sdpAudioPrefix):
			// Media line: m=audio 10000 RTP/AVP 0 8 101
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				port, err := strconv.Atoi(parts[1])
				if err == nil {
					info.AudioPort = port
				}
				for i := 3; i < len(parts); i++ {
					pt, err := strconv.Atoi(parts[i])
					if err == nil && pt >= 0 && pt <= 127 {
						payloadType, convertErr := utils.IntToUint8(pt)
						if convertErr == nil {
							info.PayloadTypes = append(info.PayloadTypes, payloadType)
						}
					}
				}
			}

		case strings.HasPrefix(line, sdpRTCPPrefix):
			parts := strings.Fields(strings.TrimPrefix(line, sdpRTCPPrefix))
			if len(parts) >= 1 {
				port, err := strconv.Atoi(parts[0])
				if err == nil && port > 0 && port <= rtpMaxPort {
					info.RTCPPort = port
				}
			}
			if len(parts) >= 4 && parts[1] == "IN" && parts[2] == "IP4" {
				info.RTCPIP = parts[3]
			}

		case strings.HasPrefix(line, sdpPTimePrefix):
			ptime, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, sdpPTimePrefix)))
			if err == nil && validator.Between(ptime, sdpMinPTimeMS, sdpMaxPTimeMS) {
				info.PTime = ptime
			}

		case strings.HasPrefix(line, sdpRTPMapPrefix):
			// RTP map: a=rtpmap:0 PCMU/8000
			// We use this to confirm codec selection

		// SDP direction attributes (RFC 3264)
		// Used by all providers for hold/resume:
		//   - Twilio/Telnyx: sendonly or inactive when putting call on hold
		//   - Asterisk/FreeSWITCH: sendonly or 0.0.0.0 connection IP
		//   - Vonage: inactive for hold
		case line == "a=sendrecv":
			info.Direction = SDPDirectionSendRecv
		case line == "a=sendonly":
			info.Direction = SDPDirectionSendOnly
		case line == "a=recvonly":
			info.Direction = SDPDirectionRecvOnly
		case line == "a=inactive":
			info.Direction = SDPDirectionInactive
		}
	}

	// Determine preferred codec based on first matching payload type.
	// Skip telephone-event (PT 101). It is not an audio codec.
	for _, pt := range info.PayloadTypes {
		if pt == CodecTelephoneEvent.PayloadType {
			continue // telephone-event is not an audio codec
		}
		for _, codec := range SupportedCodecs {
			if codec.PayloadType == pt {
				info.PreferredCodec = &codec
				break
			}
		}
		if info.PreferredCodec != nil {
			break
		}
	}
	if info.RTCPPort > 0 && info.RTCPIP == "" {
		info.RTCPIP = info.ConnectionIP
	}

	return info, nil
}

// GetCodecByPayloadType returns a codec by its payload type
func GetCodecByPayloadType(pt uint8) *Codec {
	for _, codec := range SupportedCodecs {
		if codec.PayloadType == pt {
			return &codec
		}
	}
	return nil
}

// GetCodecByName returns a codec by its name
func GetCodecByName(name string) *Codec {
	name = strings.ToUpper(name)
	for _, codec := range SupportedCodecs {
		if codec.Name == name {
			return &codec
		}
	}
	return nil
}
