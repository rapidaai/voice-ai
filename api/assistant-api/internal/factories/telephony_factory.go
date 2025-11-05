package internal_factories

import (
	"errors"

	internal_telephony "github.com/rapidaai/api/assistant-api/internal/telephony"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

type Telephony string

const (
	Twilio Telephony = "twilio"
	Exotel Telephony = "exotel"
	Vonage Telephony = "vonage"
)

func (at Telephony) String() string {
	return string(at)
}
func GetTelephony(
	at Telephony,
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	opts utils.Option) (internal_telephony.Telephony, error) {
	switch at {
	case Twilio:
		return internal_telephony.NewTwilioTelephony(logger, vaultCredential, opts)
	case Exotel:
		return internal_telephony.NewExotelTelephony(logger, vaultCredential, opts)
	case Vonage:
		return internal_telephony.NewVonageTelephony(logger, vaultCredential, opts)
	default:
		return nil, errors.New("illegal telephony provider")
	}
}
