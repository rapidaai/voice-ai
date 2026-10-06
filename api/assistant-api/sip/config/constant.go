// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import "time"

const (
	// DefaultProviderPort is used when a provider vault credential omits sip_port.
	DefaultProviderPort = 5060

	defaultRegisterTimeout = 10 * time.Second
	defaultInboundACKTime  = 5 * time.Second
	maxPort                = 65535
)
