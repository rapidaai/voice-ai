// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"strconv"
	"strings"

	"github.com/emiago/sipgo/sip"
)

// SIPReason contains the parsed reason reported by a SIP peer.
type SIPReason struct {
	Protocol string
	Cause    int
	Text     string
	Raw      string
}

func NewSIPReason(request *sip.Request) SIPReason {
	if request == nil {
		return SIPReason{}
	}

	for _, header := range request.GetHeaders("Reason") {
		reason := SIPReason{Raw: strings.TrimSpace(header.Value())}
		if reason.Raw == "" {
			continue
		}

		parts := make([]string, 0, 3)
		var part strings.Builder
		inQuotes := false
		for _, char := range reason.Raw {
			switch char {
			case '"':
				inQuotes = !inQuotes
				part.WriteRune(char)
			case ';':
				if inQuotes {
					part.WriteRune(char)
					continue
				}
				if value := strings.TrimSpace(part.String()); value != "" {
					parts = append(parts, value)
				}
				part.Reset()
			default:
				part.WriteRune(char)
			}
		}
		if value := strings.TrimSpace(part.String()); value != "" {
			parts = append(parts, value)
		}
		if len(parts) == 0 {
			continue
		}

		reason.Protocol = strings.ToLower(strings.TrimSpace(parts[0]))
		for _, parameter := range parts[1:] {
			key, value, ok := strings.Cut(parameter, "=")
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "cause":
				reason.Cause, _ = strconv.Atoi(strings.TrimSpace(value))
			case "text":
				value = strings.TrimSpace(value)
				if unquoted, err := strconv.Unquote(value); err == nil {
					reason.Text = unquoted
				} else {
					reason.Text = value
				}
			}
		}
		return reason
	}
	return SIPReason{}
}

func (reason SIPReason) DisconnectMetadata() DisconnectMetadata {
	metadata := DisconnectMetadata{
		Reason:             DisconnectReasonRemoteHangup,
		Text:               reason.Text,
		Raw:                reason.Raw,
		ProviderStatusCode: reason.Cause,
	}
	switch reason.Protocol {
	case "q.850":
		switch reason.Cause {
		case 16, 31:
			metadata.Reason = DisconnectReasonNormalClearing
		case 17:
			metadata.Reason = DisconnectReasonBusy
		case 18, 19:
			metadata.Reason = DisconnectReasonNoAnswer
		case 21:
			metadata.Reason = DisconnectReasonRejected
		case 34, 38, 41, 42, 47:
			metadata.Reason = DisconnectReasonNetworkFailure
		}
	case "sip":
		switch reason.Cause {
		case 200:
			metadata.Reason = DisconnectReasonNormalClearing
		case 408, 480:
			metadata.Reason = DisconnectReasonNoAnswer
		case 486, 600:
			metadata.Reason = DisconnectReasonBusy
		case 487:
			metadata.Reason = DisconnectReasonCancelled
		case 403, 603:
			metadata.Reason = DisconnectReasonRejected
		case 500, 502, 503, 504:
			metadata.Reason = DisconnectReasonNetworkFailure
		}
	}
	if metadata.Reason == DisconnectReasonRemoteHangup && reason.Cause > 0 {
		metadata.Reason = DisconnectReasonRemoteError
	}
	return metadata
}
