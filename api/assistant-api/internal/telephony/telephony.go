package internal_telephony

import "github.com/rapidaai/pkg/types"

type Telephony interface {
	CreateCall(
		auth types.SimplePrinciple,
		toPhone string,
		fromPhone string,
		assistantId, sessionId uint64) (map[string]interface{}, error)
}
