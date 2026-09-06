// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package channel_base provides transport-agnostic streamer plumbing shared by
// concrete channel implementations.
package channel_base

import (
	"context"
	"fmt"
	"io"
	"sync"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultInputChannelCapacity  = 1000
	defaultOutputChannelCapacity = 500
	criticalChannelCapacity      = 16
	lowPriorityChannelCapacity   = 512
)

// BaseStreamer owns common stream channels and lifecycle. Media buffering,
// codec conversion, and playback timing belong to concrete streamers.
type BaseStreamer struct {
	Mu         sync.Mutex
	Logger     commons.Logger
	Ctx        context.Context
	Cancel     context.CancelFunc
	Closed     bool
	CriticalCh chan internal_type.Stream
	InputCh    chan internal_type.Stream
	LowCh      chan internal_type.Stream
	OutputCh   chan internal_type.Stream
}

type options struct {
	logger                commons.Logger
	inputChannelCapacity  int
	outputChannelCapacity int
}

type Option func(*options)

// WithLogger sets the streamer logger.
func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

// WithInputChannelCapacity sets the realtime input queue capacity.
func WithInputChannelCapacity(capacity int) Option {
	return func(options *options) {
		options.inputChannelCapacity = capacity
	}
}

// WithOutputChannelCapacity sets the output queue capacity.
func WithOutputChannelCapacity(capacity int) Option {
	return func(options *options) {
		options.outputChannelCapacity = capacity
	}
}

// New creates transport channels from caller-provided options.
func New(opts ...Option) BaseStreamer {
	options := options{
		inputChannelCapacity:  defaultInputChannelCapacity,
		outputChannelCapacity: defaultOutputChannelCapacity,
	}
	for _, option := range opts {
		if option != nil {
			option(&options)
		}
	}
	if options.inputChannelCapacity <= 0 {
		options.inputChannelCapacity = defaultInputChannelCapacity
	}
	if options.outputChannelCapacity <= 0 {
		options.outputChannelCapacity = defaultOutputChannelCapacity
	}
	ctx, cancel := context.WithCancel(context.Background())
	return BaseStreamer{
		Logger:     options.logger,
		Ctx:        ctx,
		Cancel:     cancel,
		CriticalCh: make(chan internal_type.Stream, criticalChannelCapacity),
		InputCh:    make(chan internal_type.Stream, options.inputChannelCapacity),
		LowCh:      make(chan internal_type.Stream, lowPriorityChannelCapacity),
		OutputCh:   make(chan internal_type.Stream, options.outputChannelCapacity),
	}
}

// Input routes messages into priority channels consumed by Recv.
func (s *BaseStreamer) Input(msg internal_type.Stream) {
	switch message := msg.(type) {
	case *protos.ConversationDisconnection,
		*protos.ConversationToolCallResult,
		*protos.ConversationInitialization,
		*protos.ConversationConfiguration,
		*protos.ConversationError:
		select {
		case s.CriticalCh <- msg:
		default:
			if s.Logger != nil {
				s.Logger.Warnw("Critical input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
			}
		}
		return
	case *protos.ConversationUserMessage:
		if _, isAudio := message.Message.(*protos.ConversationUserMessage_Audio); !isAudio {
			select {
			case s.CriticalCh <- msg:
			default:
				if s.Logger != nil {
					s.Logger.Warnw("Critical input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
				}
			}
			return
		}
	case *protos.ConversationEvent,
		*protos.ConversationMetric,
		*protos.ConversationMetadata,
		*protos.ConversationBridgeUserAudio,
		*protos.ConversationBridgeOperatorAudio:
		select {
		case s.LowCh <- msg:
		default:
			if s.Logger != nil {
				s.Logger.Warnw("Low input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
			}
		}
		return
	}

	var dropped internal_type.Stream
	select {
	case s.InputCh <- msg:
		return
	default:
	}
	select {
	case dropped = <-s.InputCh:
	default:
	}
	select {
	case s.InputCh <- msg:
	default:
		if s.Logger != nil {
			s.Logger.Warnw("Input channel full, dropping latest audio", "type", fmt.Sprintf("%T", msg))
		}
	}
	if dropped != nil && s.Logger != nil {
		s.Logger.Warnw("Input channel full, dropping oldest audio", "type", fmt.Sprintf("%T", dropped))
	}
}

func (s *BaseStreamer) Output(msg internal_type.Stream) {
	select {
	case s.OutputCh <- msg:
	default:
		if s.Logger != nil {
			s.Logger.Warnw("Output channel full, dropping message", "type", fmt.Sprintf("%T", msg))
		}
	}
}
func (s *BaseStreamer) Disconnect(reason protos.ConversationDisconnection_DisconnectionType) *protos.ConversationDisconnection {
	s.Mu.Lock()
	alreadyClosed := s.Closed
	s.Closed = true
	s.Mu.Unlock()
	if alreadyClosed {
		return nil
	}
	return &protos.ConversationDisconnection{
		Type: reason,
		Time: timestamppb.Now(),
	}
}

func (s *BaseStreamer) Context() context.Context {
	return s.Ctx
}

func (s *BaseStreamer) Recv() (internal_type.Stream, error) {
	for {
		select {
		case msg, ok := <-s.CriticalCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		default:
		}

		select {
		case msg, ok := <-s.InputCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		default:
		}

		select {
		case msg, ok := <-s.CriticalCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		case msg, ok := <-s.InputCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		case msg, ok := <-s.LowCh:
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		case <-s.Ctx.Done():
			return nil, io.EOF
		}
	}
}
