// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_media

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// MediaEngine defines shared telephony media semantics independent of transport.
type MediaEngine interface {
	ProcessProviderAudioFrame(frame ProviderAudioFrame) (InputAudioFrame, error)
	ProcessAssistantAudio(audio []byte, completed bool) error
	NextOutputFrame() (AssistantOutputFrame, bool)
	IdleOutputFrame() (AssistantOutputFrame, bool)
	ClearOutputBuffer()
	ConfigureAmbient(ambientConfig internal_ambient.Config) error
	OutputFrameDuration() time.Duration
	OutputHealthSnapshot() internal_output.HealthSnapshot
	internal_output.HealthObserver
}

// MediaSessionConfig carries all transport-independent session dependencies.
type MediaSessionConfig struct {
	Context           context.Context
	Logger            commons.Logger
	MediaEngine       MediaEngine
	SendProviderClear func() error
	StreamSink        StreamSink
	OutputSink        OutputSink
	Record            func(...observability.Record) error
}

// MediaSession owns telephony media lifecycle for a channel transport.
// Transport implementations only need to feed provider audio in and send clear commands.
type MediaSession struct {
	logger      commons.Logger
	mediaEngine MediaEngine

	sendProviderClear func() error

	sinkMu     sync.RWMutex
	streamSink StreamSink
	outputSink OutputSink
	record     func(...observability.Record) error

	outputFrameMu         sync.Mutex
	currentOutputFrame    AssistantOutputFrame
	hasCurrentOutputFrame bool

	started atomic.Bool
	closed  atomic.Bool

	startMu sync.Mutex
	cancel  context.CancelFunc
	ctx     context.Context
}

// ProviderAudioFrame carries provider audio at the websocket receive boundary.
type ProviderAudioFrame struct {
	Audio      []byte
	ReceivedAt time.Time
}

// InputAudioFrame separates recording audio from realtime AI input.
type InputAudioFrame struct {
	BridgeAudio   []byte
	PipelineAudio []byte
	ReceivedAt    time.Time
}

// AssistantOutputFrame carries paired provider audio and bridge audio.
type AssistantOutputFrame struct {
	ProviderAudio []byte
	BridgeAudio   []byte
	Idle          bool
}

// StreamSink pushes conversation streams back into the channel input path.
type StreamSink func(internal_type.Stream)

// OutputSink writes a paced provider frame to the transport.
type OutputSink func(frame AssistantOutputFrame) error
