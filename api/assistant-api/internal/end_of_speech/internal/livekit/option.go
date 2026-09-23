// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type options struct {
	ctx      context.Context
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	options  utils.Option
}

// Option configures the EOS provider at construction time.
type Option func(*options)

// WithContext supplies initialization telemetry context; packet contexts scope inference.
func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

// WithLogger enables optional prediction debug logging.
func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithOnPacket supplies the required packet sink, which must be safe for concurrent calls.
// Callback errors do not retry or roll back an emitted turn.
func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

// WithOptions supplies provider parameters; New validates supplied values and defaults missing values.
// Parameter keys, units, and limits are documented in README.md.
func WithOptions(opts utils.Option) Option {
	return func(options *options) {
		options.options = opts
	}
}
