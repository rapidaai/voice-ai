// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_factory

import (
	"errors"

	"github.com/rapidaai/api/assistant-api/config"
	internal_exotel_telephony "github.com/rapidaai/api/assistant-api/internal/telephony/internal/exotel"
	internal_telnyx_telephony "github.com/rapidaai/api/assistant-api/internal/telephony/internal/telnyx"
	internal_twilio_telephony "github.com/rapidaai/api/assistant-api/internal/telephony/internal/twilio"
	internal_vonage_telephony "github.com/rapidaai/api/assistant-api/internal/telephony/internal/vonage"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type Telephony string

const (
	Twilio Telephony = "twilio"
	Exotel Telephony = "exotel"
	Vonage Telephony = "vonage"
	Telnyx Telephony = "telnyx"
)

func (at Telephony) String() string {
	return string(at)
}

func GetTelephony(at Telephony, cfg *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	switch at {
	case Twilio:
		return internal_twilio_telephony.NewTwilioTelephony(cfg, logger)
	case Exotel:
		return internal_exotel_telephony.NewExotelTelephony(cfg, logger)
	case Vonage:
		return internal_vonage_telephony.NewVonageTelephony(cfg, logger)
	case Telnyx:
		return internal_telnyx_telephony.NewTelnyxTelephony(cfg, logger)
	default:
		return nil, errors.New("illegal telephony provider")
	}
}
