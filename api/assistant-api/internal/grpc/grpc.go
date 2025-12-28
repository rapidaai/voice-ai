// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_grpc

import (
	"context"

	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

type unidirectionalStreamer struct {
	server grpc.BidiStreamingServer[protos.AssistantMessagingRequest, protos.AssistantMessagingResponse]
}

func NewGrpcUnidirectionalStreamer(
	server protos.TalkService_AssistantTalkServer) internal_streamers.Streamer {
	return &unidirectionalStreamer{
		server: server,
	}
}

func (uds *unidirectionalStreamer) Context() context.Context {
	return uds.server.Context()
}

func (uds *unidirectionalStreamer) Recv() (*protos.AssistantMessagingRequest, error) {
	return uds.server.Recv()
}

// Send sends an output value to the stream.
// It returns an error if the send operation fails.
func (uds *unidirectionalStreamer) Send(out *protos.AssistantMessagingResponse) error {
	return uds.server.Send(out)
}
