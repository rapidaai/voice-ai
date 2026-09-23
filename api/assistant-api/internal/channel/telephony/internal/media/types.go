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
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"google.golang.org/protobuf/proto"
)

// MediaEngine defines shared telephony media semantics independent of transport.
type MediaEngine interface {
	ProcessProviderAudioFrame(frame ProviderAudioFrame) (InputAudioFrame, error)
	ProcessAssistantAudio(audio []byte, completed bool) error
	NextOutputFrame() (AssistantOutputFrame, bool)
	// OutputDrained excludes partial frames and transport suspension.
	OutputDrained() bool
	IdleOutputFrame() (AssistantOutputFrame, bool)
	ClearOutputBuffer()
	ConfigureAmbient(ambientConfig internal_ambient.Config) error
	OutputFrameDuration() time.Duration
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
	outputPaused          bool
	outputFlushed         bool
	currentOutputID       string
	blockedOutputID       string
	responses             []*responsePlayback
	flushedOutputIDs      map[string]struct{}
	closedOutputIDs       map[string]struct{}
	discardedOutputIDs    map[string]struct{}

	started atomic.Bool
	closed  atomic.Bool

	startMu sync.Mutex
	cancel  context.CancelFunc
	ctx     context.Context
}

type responsePlayback struct {
	id        string
	audio     []byte
	completed bool
	processed bool
	failed    bool
	sent      bool
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
type StreamSink func(proto.Message)

// OutputSink writes a paced provider frame to the transport.
type OutputSink func(frame AssistantOutputFrame) error
