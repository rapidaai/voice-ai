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
	"sync/atomic"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultInputChannelCapacity  = 100
	defaultOutputChannelCapacity = 500
	criticalChannelCapacity      = 16
	observabilityChannelCapacity = 512
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

	criticalInputDropped atomic.Uint64
	normalInputDropped   atomic.Uint64
	lowInputDropped      atomic.Uint64
	outputDropped        atomic.Uint64
}

// StreamerDropStats reports messages dropped by BaseStreamer priority queues.
type StreamerDropStats struct {
	CriticalInputDropped uint64
	NormalInputDropped   uint64
	LowInputDropped      uint64
	OutputDropped        uint64
}

// NewBaseStreamer creates transport channels with default capacities.
func NewBaseStreamer(logger commons.Logger) BaseStreamer {
	return NewBaseStreamerWithChannelCapacity(logger, defaultInputChannelCapacity, defaultOutputChannelCapacity)
}

// NewBaseStreamerWithChannelCapacity creates transport channels with caller-defined capacities.
func NewBaseStreamerWithChannelCapacity(
	logger commons.Logger,
	inputChannelCapacity int,
	outputChannelCapacity int,
) BaseStreamer {
	ctx, cancel := context.WithCancel(context.Background())
	return BaseStreamer{
		Logger:     logger,
		Ctx:        ctx,
		Cancel:     cancel,
		CriticalCh: make(chan internal_type.Stream, criticalChannelCapacity),
		InputCh:    make(chan internal_type.Stream, inputChannelCapacity),
		LowCh:      make(chan internal_type.Stream, observabilityChannelCapacity),
		OutputCh:   make(chan internal_type.Stream, outputChannelCapacity),
	}
}

// Input routes messages into priority channels consumed by Recv.
func (s *BaseStreamer) Input(msg internal_type.Stream) {
	switch msg.(type) {
	case *protos.ConversationDisconnection,
		*protos.ConversationToolCallResult:
		select {
		case s.CriticalCh <- msg:
		default:
			s.criticalInputDropped.Add(1)
			if s.Logger != nil {
				s.Logger.Warnw("Critical input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
			}
		}
	case *protos.ConversationEvent,
		*protos.ConversationMetric,
		*protos.ConversationMetadata:
		select {
		case s.LowCh <- msg:
		default:
			s.lowInputDropped.Add(1)
			if s.Logger != nil {
				s.Logger.Warnw("Low input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
			}
		}
	default:
		select {
		case s.InputCh <- msg:
		default:
			s.normalInputDropped.Add(1)
			if s.Logger != nil {
				s.Logger.Warnw("Normal input channel full, dropping message", "type", fmt.Sprintf("%T", msg))
			}
		}
	}
}

func (s *BaseStreamer) Output(msg internal_type.Stream) {
	select {
	case s.OutputCh <- msg:
	default:
		s.outputDropped.Add(1)
		if s.Logger != nil {
			s.Logger.Warnw("Output channel full, dropping message", "type", fmt.Sprintf("%T", msg))
		}
	}
}

// DropStats returns a lock-free snapshot of BaseStreamer queue drops.
func (s *BaseStreamer) DropStats() StreamerDropStats {
	if s == nil {
		return StreamerDropStats{}
	}
	return StreamerDropStats{
		CriticalInputDropped: s.criticalInputDropped.Load(),
		NormalInputDropped:   s.normalInputDropped.Load(),
		LowInputDropped:      s.lowInputDropped.Load(),
		OutputDropped:        s.outputDropped.Load(),
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
