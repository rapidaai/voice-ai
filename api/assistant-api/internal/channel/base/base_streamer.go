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
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rapidaai/pkg/channel"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultInputChannelCapacity  = 1000
	defaultOutputChannelCapacity = 500
	criticalChannelCapacity      = 16
	lowPriorityChannelCapacity   = 512
	disconnectionDeliveryTimeout = time.Second
)

// BaseStreamer owns common stream channels and lifecycle. Media buffering,
// codec conversion, and playback timing belong to concrete streamers.
type BaseStreamer struct {
	Mu         sync.Mutex
	Logger     commons.Logger
	Ctx        context.Context
	Cancel     context.CancelFunc
	Closed     bool
	CriticalCh *channel.Channel[proto.Message]
	InputCh    *channel.Channel[proto.Message]
	LowCh      *channel.Channel[proto.Message]
	OutputCh   *channel.Channel[proto.Message]
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
	criticalCh, err := channel.New[proto.Message](channel.Config{
		CapacityPolicy: channel.FixedCapacity(criticalChannelCapacity),
		OverflowPolicy: channel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	inputCh, err := channel.New[proto.Message](channel.Config{
		CapacityPolicy: channel.FixedCapacity(options.inputChannelCapacity),
		OverflowPolicy: channel.ReplaceOldestWhenFull,
	})
	if err != nil {
		panic(err)
	}
	lowCh, err := channel.New[proto.Message](channel.Config{
		CapacityPolicy: channel.FixedCapacity(lowPriorityChannelCapacity),
		OverflowPolicy: channel.RejectNewestWhenFull,
	})
	if err != nil {
		panic(err)
	}
	outputCh, err := channel.New[proto.Message](channel.Config{
		CapacityPolicy: channel.FixedCapacity(options.outputChannelCapacity),
		OverflowPolicy: channel.BlockWhenFull,
	})
	if err != nil {
		panic(err)
	}
	return BaseStreamer{
		Logger:     options.logger,
		Ctx:        ctx,
		Cancel:     cancel,
		CriticalCh: criticalCh,
		InputCh:    inputCh,
		LowCh:      lowCh,
		OutputCh:   outputCh,
	}
}

// Input routes messages into priority channels consumed by Recv.
func (s *BaseStreamer) Input(msg proto.Message) {
	switch message := msg.(type) {
	case *protos.ConversationDisconnection:
		// Transport teardown must not wait indefinitely for its final notification.
		ctx, cancel := context.WithTimeout(s.Ctx, disconnectionDeliveryTimeout)
		defer cancel()
		if _, err := s.CriticalCh.Send(ctx, msg); errors.Is(err, context.DeadlineExceeded) && s.Logger != nil {
			s.Logger.Warnw("Disconnection input delivery timed out; proceeding with shutdown")
		}
		return
	case *protos.ConversationPlaybackComplete,
		*protos.ConversationToolCallResult,
		*protos.ConversationInitialization,
		*protos.ConversationConfiguration,
		*protos.ConversationError:
		_, _ = s.CriticalCh.Send(s.Ctx, msg)
		return
	case *protos.ConversationUserMessage:
		if _, isAudio := message.Message.(*protos.ConversationUserMessage_Audio); !isAudio {
			_, _ = s.CriticalCh.Send(s.Ctx, msg)
			return
		}
	case *protos.ConversationEvent,
		*protos.ConversationMetric,
		*protos.ConversationMetadata,
		*protos.ConversationBridgeUserAudio,
		*protos.ConversationBridgeOperatorAudio:
		result, err := s.LowCh.Send(s.Ctx, msg)
		if err == nil && result.Status == channel.Rejected && s.Logger != nil {
			s.Logger.Warnw("Low input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
		}
		return
	}

	result, err := s.InputCh.Send(s.Ctx, msg)
	if err != nil {
		return
	}
	if result.Status == channel.ReplacedOldest && s.Logger != nil {
		s.Logger.Warnw("Input channel full, replacing oldest audio", "type", fmt.Sprintf("%T", result.Discarded))
	}
}

func (s *BaseStreamer) Output(msg proto.Message) {
	if _, isDisconnection := msg.(*protos.ConversationDisconnection); isDisconnection {
		ctx, cancel := context.WithTimeout(s.Ctx, disconnectionDeliveryTimeout)
		defer cancel()
		if _, err := s.OutputCh.Send(ctx, msg); errors.Is(err, context.DeadlineExceeded) && s.Logger != nil {
			s.Logger.Warnw("Disconnection output delivery timed out; proceeding with shutdown")
		}
		return
	}
	_, _ = s.OutputCh.Send(s.Ctx, msg)
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

func (s *BaseStreamer) Recv() (proto.Message, error) {
	for {
		msg, err := s.CriticalCh.TryReceive()
		if err == nil {
			return msg, nil
		}
		if errors.Is(err, channel.ErrClosed) {
			return nil, io.EOF
		}

		msg, err = s.InputCh.TryReceive()
		if err == nil {
			return msg, nil
		}
		if errors.Is(err, channel.ErrClosed) {
			return nil, io.EOF
		}

		select {
		case <-s.CriticalCh.Ready():
			continue
		case <-s.InputCh.Ready():
			continue
		case <-s.LowCh.Ready():
			msg, err := s.LowCh.TryReceive()
			if errors.Is(err, channel.ErrClosed) {
				return nil, io.EOF
			}
			if err == nil {
				return msg, nil
			}
		case <-s.Ctx.Done():
			return nil, io.EOF
		}
	}
}
