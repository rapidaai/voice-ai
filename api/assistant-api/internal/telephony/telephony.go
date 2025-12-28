// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telephony

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// any telephony integration must impliment this interface to provide consistent behaviour
type Telephony interface {
	// streamer
	Streamer(c *gin.Context, connection *websocket.Conn, assistantID uint64, assistantVersion string, assistantConversationID uint64) internal_streamers.Streamer

	// for creating call throght telephony
	MakeCall(
		auth types.SimplePrinciple,
		toPhone string,
		fromPhone string,
		assistantId, assistantConversationId uint64,
		vaultCredential *protos.VaultCredential,
		opts utils.Option,
	) ([]*types.Metadata, []*types.Metric, []*types.Event, error)

	//  callback for a conversation
	Callback(ctx *gin.Context, auth types.SimplePrinciple, assistantId, assistantConversationId uint64) ([]*types.Metric, []*types.Event, error)

	// conversation reference, metrics, events
	CatchAllCallback(ctx *gin.Context) (*string, []*types.Metric, []*types.Event, error)

	//
	ReceiveCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error

	//
	GetCaller(c *gin.Context) (string, bool)

	// let telephony provider handle and allow to choose what will be best codecs and format
	// streaming config
	// added to support multiple audio codecs and format
	InputStreamConfig() *protos.StreamConfig

	// streaming config
	// added to support multiple audio codecs and format
	OutputStreamConfig() *protos.StreamConfig
}

func GetAnswerPath(provider string, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64, toPhone string) string {
	switch auth.Type() {
	case "project":
		return fmt.Sprintf("v1/talk/%s/prj/%d/%s/%d/%s",
			provider,
			assistantId,
			toPhone,
			assistantConversationId,
			auth.GetCurrentToken())
	default:
		return fmt.Sprintf("v1/talk/%s/usr/%d/%s/%d/%s/%d/%d",
			provider,
			assistantId,
			toPhone,
			assistantConversationId,
			auth.GetCurrentToken(),
			*auth.GetUserId(),
			*auth.GetCurrentProjectId())
	}
}

func GetEventPath(provider string, auth types.SimplePrinciple, assistantId, assistantConversationId uint64) string {
	switch auth.Type() {
	case "project":
		return fmt.Sprintf("v1/talk/%s/prj/event/%d/%d/%s",
			provider,
			assistantId,
			assistantConversationId,
			auth.GetCurrentToken())
	default:
		return fmt.Sprintf("v1/talk/%s/usr/event/%d/%d/%s/%d/%d",
			provider,
			assistantId,
			assistantConversationId,
			auth.GetCurrentToken(),
			*auth.GetUserId(),
			*auth.GetCurrentProjectId())
	}
}
