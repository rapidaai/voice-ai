// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package config

import "errors"

// ErrInvalidConfig marks invalid SIP provider or server configuration.
var ErrInvalidConfig = errors.New("invalid SIP configuration")
