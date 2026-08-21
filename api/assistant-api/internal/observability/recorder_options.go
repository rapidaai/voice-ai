// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type recorderOptions struct {
	logger           commons.Logger
	auth             *types.Authentication
	globalScope      GlobalScope
	context          Context
	clock            func() time.Time
	closeGracePeriod *time.Duration
	collectors       []Collector
}

type Option interface {
	apply(*recorderOptions)
}

type funcOption struct {
	applyFunc func(*recorderOptions)
}

func (option *funcOption) apply(recorderOptions *recorderOptions) {
	option.applyFunc(recorderOptions)
}

func newOption(applyFunc func(*recorderOptions)) Option {
	return &funcOption{applyFunc: applyFunc}
}

func WithAuth(auth *types.Authentication) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.auth = auth
	})
}

func WithLogger(logger commons.Logger) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.logger = logger
	})
}

func WithGlobalScope(scope GlobalScope) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.globalScope = scope
	})
}

func WithScope(scope GlobalScope) Option {
	return WithGlobalScope(scope)
}

func WithContext(ctx context.Context) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		if traceID, ok := ctx.Value(types.REQUEST_ID_KEY).(string); ok {
			recorderOptions.context.TraceID = traceID
			return
		}
		recorderOptions.context.TraceID = uuid.New().String()
	})
}

func WithClock(clock func() time.Time) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.clock = clock
	})
}

func WithGracePeriod() Option {
	return newOption(func(recorderOptions *recorderOptions) {
		closeGracePeriod := recorderCloseGracePeriod
		recorderOptions.closeGracePeriod = &closeGracePeriod
	})
}

func WithCustomGracePeriod(closeGracePeriod time.Duration) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.closeGracePeriod = &closeGracePeriod
	})
}

func WithCollector(collector Collector) Option {
	return WithCollectors(collector)
}

func WithCollectors(collectors ...Collector) Option {
	return newOption(func(recorderOptions *recorderOptions) {
		recorderOptions.collectors = append(recorderOptions.collectors, collectors...)
	})
}
