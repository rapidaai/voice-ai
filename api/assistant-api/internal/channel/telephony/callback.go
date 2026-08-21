// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"fmt"

	"github.com/gin-gonic/gin"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
)

// HandleStatusCallback resolves the telephony provider and parses a status callback webhook.
func (d *InboundDispatcher) HandleStatusCallback(c *gin.Context, provider string, auth *types.Authentication, assistantId, conversationId uint64) (*internal_type.StatusInfo, error) {
	tel, err := GetTelephony(Telephony(provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return nil, fmt.Errorf("invalid telephony provider %s: %w", provider, err)
	}

	statusInfo, err := tel.StatusCallback(c, auth, assistantId, conversationId)
	if err != nil {
		return nil, fmt.Errorf("status callback failed: %w", err)
	}
	return statusInfo, nil
}

// HandleCatchAllStatusCallback resolves the telephony provider and parses a global status callback webhook.
func (d *InboundDispatcher) HandleCatchAllStatusCallback(c *gin.Context, provider string) (*internal_type.StatusInfo, error) {
	tel, err := GetTelephony(Telephony(provider), d.cfg, d.logger, d.telephonyOpt)
	if err != nil {
		return nil, fmt.Errorf("invalid telephony provider %s: %w", provider, err)
	}

	statusInfo, err := tel.CatchAllStatusCallback(c)
	if err != nil {
		return nil, fmt.Errorf("catch-all status callback failed: %w", err)
	}
	return statusInfo, nil
}
