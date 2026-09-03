// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"errors"
	"fmt"

	"github.com/rapidaai/api/assistant-api/config"
	internal_asterisk_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk"
	internal_exotel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel"
	internal_sip_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip"
	internal_telnyx_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx"
	internal_twilio_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio"
	internal_vobiz_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz"
	internal_vonage_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vonage"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/commons"
)

// Telephony is a string type identifying a telephony provider.
type Telephony string

const (
	Twilio   Telephony = "twilio"
	Exotel   Telephony = "exotel"
	Vonage   Telephony = "vonage"
	Telnyx   Telephony = "telnyx"
	Asterisk Telephony = "asterisk"
	SIP      Telephony = "sip"
	Vobiz    Telephony = "vobiz"
)

func (at Telephony) String() string {
	return string(at)
}

// GetTelephony is the factory function that creates a telephony provider for the
// given type. This follows the platform factory pattern — providers are created
// per-request through a switch-based lookup.
//
// For SIP, the caller must supply the SIPServer via TelephonyOption.
func GetTelephony(at Telephony, cfg *config.AssistantConfig, logger commons.Logger, opts ...TelephonyOption) (internal_type.Telephony, error) {
	var opt TelephonyOption
	if len(opts) > 0 {
		opt = opts[0]
	}

	switch at {
	case Twilio:
		return internal_twilio_telephony.NewTwilioTelephony(cfg, logger)
	case Exotel:
		return internal_exotel_telephony.NewExotelTelephony(cfg, logger)
	case Vonage:
		return internal_vonage_telephony.NewVonageTelephony(cfg, logger)
	case Asterisk:
		return internal_asterisk_telephony.NewAsteriskTelephony(cfg, logger)
	case Telnyx:
		return internal_telnyx_telephony.NewTelnyxTelephony(cfg, logger)
	case Vobiz:
		return internal_vobiz_telephony.NewVobizTelephony(cfg, logger)
	case SIP:
		if opt.SIPServer == nil {
			return nil, errors.New("SIP server not available — SIP telephony requires a running SIP server")
		}
		return internal_sip_telephony.NewSIPTelephony(cfg, logger, opt.SIPServer)
	default:
		return nil, fmt.Errorf("unknown telephony provider %q", at)
	}
}

// TelephonyOption configures optional dependencies for telephony providers.
type TelephonyOption struct {
	SIPServer *sip_runtime.Server
}
