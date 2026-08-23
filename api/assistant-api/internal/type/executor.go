// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_type

import (
	"context"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// Executor is the generic contract for session-scoped packet handlers.
// Construction is fully wired by each implementation's New<X>Executor.
type Executor[P Packet] interface {
	Name() string
	Options() utils.Option
	Arguments() (map[string]string, error)
	Execute(ctx context.Context, packet P) error
	Close(ctx context.Context) error
}

// SyncExecutor is for session-scoped executors that must return data to the
// caller before the next packet in the flow can run.
type SyncExecutor[I any, O any] interface {
	Name() string
	Options() utils.Option
	Arguments() (map[string]string, error)
	Execute(ctx context.Context, input I) (O, error)
	Close(ctx context.Context) error
}

type AnalysisInput struct {
	ContextID string
	Arguments map[string]interface{}
	Auth      *types.Authentication
}

type AnalysisOutput struct {
	Metadata *protos.Metadata
}

type AuthenticationInput struct {
	ContextID      string
	Arguments      map[string]interface{}
	Initialization *protos.ConversationInitialization
}

type AuthenticationOutput struct {
	Authenticated bool
	Arguments     map[string]interface{}
	Metadata      map[string]interface{}
	Options       map[string]interface{}
}

type ArtifactPushArtifact struct {
	Name        string
	Type        string
	ContentType string
	Content     []byte
}

type ArtifactPushInput struct {
	ContextID      string
	ConversationID uint64
	Artifacts      []ArtifactPushArtifact
}

type ArtifactPushResult struct {
	Name           string
	Type           string
	ContentType    string
	DestinationKey string
	CompletePath   string
	StorageType    string
}

type ArtifactPushOutput struct {
	Provider        string
	ConfigurationID uint64
	Results         []ArtifactPushResult
}

// Typed interfaces for each concrete executor.
type LLMExecutor interface {
	Name() string
	Execute(ctx context.Context, communication Communication, packet Packet) error
	Close(ctx context.Context) error
}

type AnalysisExecutor interface {
	SyncExecutor[AnalysisInput, *AnalysisOutput]
}

type AuthenticationExecutor interface {
	SyncExecutor[AuthenticationInput, *AuthenticationOutput]
}

type ArtifactPushExecutor interface {
	SyncExecutor[ArtifactPushInput, *ArtifactPushOutput]
}

type EndOfSpeechExecutor interface {
	Executor[Packet]
}

type VoiceActivityDetectorExecutor interface {
	Executor[UserAudioReceivedPacket]
}

type VoiceDenoiserExecutor interface {
	Executor[DenoiseAudioPacket]
}

type ConversationRecordingAudio struct {
	UserAudio      []byte
	AssistantAudio []byte
	MixedAudio     []byte
}

type ConversationRecordingExecutor interface {
	Executor[Packet]
}
