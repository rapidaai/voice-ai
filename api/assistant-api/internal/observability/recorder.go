// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

type Recorder interface {
	Record(ctx context.Context, scope Scope, record ...Record) error
	AddCollectors(collectors ...Collector) error
	Close(ctx context.Context) error
}

const (
	// Queue capacity is fixed by platform contract.
	recorderQueueSize                      = 2048
	recorderReplayBufferSize               = 2048
	collectorQueueSize                     = 2048
	defaultCloseGracePeriod  time.Duration = 0
	recorderCloseGracePeriod               = 10 * time.Second
)

type recorder struct {
	logger           commons.Logger
	auth             *types.Authentication
	globalScope      GlobalScope
	context          Context
	clock            func() time.Time
	closeGracePeriod time.Duration
	collectorWorkers map[string]*collectorWorker
	replayBuffer     []observation
	operationQueue   chan recorderOperation
	done             chan struct{}
	closeRequested   bool
	closed           bool
	mu               sync.RWMutex
	errMu            sync.Mutex
	errs             []error
}

type recorderOperation interface {
	Type() string
}

type addCollectorOperation struct {
	collector Collector
}

func (addCollectorOperation) Type() string {
	return "addCollectorOperation"
}

type recordOperation struct {
	observation observation
}

func (recordOperation) Type() string {
	return "recordOperation"
}

type closeOperation struct{}

func (closeOperation) Type() string {
	return "closeOperation"
}

type observation struct {
	scope   Scope
	context Context
	record  Record
}

var (
	ErrRecorderClosed      = errors.New("observability: recorder is closed")
	ErrBufferFull          = errors.New("observability: recorder buffer is full")
	ErrCollectorBufferFull = errors.New("observability: collector buffer is full")
)

func New(options ...Option) Recorder {
	var resolvedOptions recorderOptions
	for _, option := range options {
		if !validator.NonNil(option) {
			continue
		}
		option.apply(&resolvedOptions)
	}

	globalScope := resolvedOptions.globalScope
	if validator.NonNil(resolvedOptions.auth) {
		if projectContext, err := resolvedOptions.auth.ProjectContext(); err == nil {
			globalScope.ProjectID = projectContext.ProjectID
			globalScope.OrganizationID = projectContext.OrganizationID
		} else if organizationContext, err := resolvedOptions.auth.OrganizationContext(); err == nil {
			globalScope.OrganizationID = organizationContext.OrganizationID
		}
	}
	clock := resolvedOptions.clock
	if clock == nil {
		clock = time.Now
	}
	closeGracePeriod := defaultCloseGracePeriod
	if resolvedOptions.closeGracePeriod != nil {
		closeGracePeriod = *resolvedOptions.closeGracePeriod
	}
	r := &recorder{
		logger:           resolvedOptions.logger,
		auth:             resolvedOptions.auth,
		globalScope:      globalScope,
		context:          resolvedOptions.context,
		clock:            clock,
		closeGracePeriod: closeGracePeriod,
		collectorWorkers: make(map[string]*collectorWorker),
		replayBuffer:     make([]observation, 0, recorderReplayBufferSize),
		operationQueue:   make(chan recorderOperation, recorderQueueSize),
		done:             make(chan struct{}),
	}
	go r.run()
	for _, collector := range resolvedOptions.collectors {
		r.operationQueue <- addCollectorOperation{collector: collector}
	}
	return r
}

func (r *recorder) Record(ctx context.Context, scope Scope, records ...Record) error {
	for _, record := range records {
		preparedObservation, err := r.prepareObservation(scope, record)
		if err != nil {
			if r.logger != nil {
				r.logger.Errorf("error while prepareObservation %s", err.Error())
			}
			return err
		}
		preparedObservation.context = r.context
		preparedObservation.context.Auth = r.auth
		if traceID, ok := ctx.Value(types.REQUEST_ID_KEY).(string); ok && traceID != "" {
			preparedObservation.context.TraceID = traceID
		}
		if preparedObservation.context.TraceID == "" {
			preparedObservation.context.TraceID = uuid.New().String()
		}

		r.mu.RLock()
		if r.closed || (r.closeRequested && r.closeGracePeriod <= 0) {
			r.mu.RUnlock()
			return ErrRecorderClosed
		}
		select {
		case r.operationQueue <- recordOperation{observation: preparedObservation}:
			r.mu.RUnlock()
		default:
			r.mu.RUnlock()
			return ErrBufferFull
		}
	}
	return nil
}

func (r *recorder) AddCollectors(collectors ...Collector) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closeRequested || r.closed {
		return ErrRecorderClosed
	}
	for _, collector := range collectors {
		select {
		case r.operationQueue <- addCollectorOperation{collector: collector}:
		default:
			return ErrBufferFull
		}
	}
	return nil
}

func (r *recorder) Close(ctx context.Context) error {
	ownsClose := false
	r.mu.Lock()
	if !r.closeRequested && !r.closed {
		r.closeRequested = true
		ownsClose = true
	}
	r.mu.Unlock()

	if !ownsClose {
		return nil
	}

	go func() {
		if r.closeGracePeriod > 0 {
			closeGracePeriodTimer := time.NewTimer(r.closeGracePeriod)
			<-closeGracePeriodTimer.C
		}

		r.mu.Lock()
		if !r.closed {
			r.closed = true
		}
		r.mu.Unlock()

		select {
		case r.operationQueue <- closeOperation{}:
		case <-r.done:
		}
	}()
	return nil
}

func (r *recorder) addError(err error) {
	// Observability failures are absorbed by contract and logged when possible.
	r.errMu.Lock()
	defer r.errMu.Unlock()
	r.errs = append(r.errs, err)
	if validator.NonNil(r.logger) {
		r.logger.Warnf("observability recorder absorbed failure: %v", err)
	}
}

func (r *recorder) errors() error {
	r.errMu.Lock()
	defer r.errMu.Unlock()
	return errors.Join(r.errs...)
}
