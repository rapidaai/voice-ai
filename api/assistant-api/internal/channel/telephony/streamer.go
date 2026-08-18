// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_telephony

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_asterisk_audiosocket "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk/audiosocket"
	internal_asterisk_websocket "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/asterisk/websocket"
	internal_exotel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel"
	internal_plivo_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/plivo"
	internal_sip_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip"
	internal_telnyx_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx"
	internal_twilio_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/twilio"
	internal_vobiz_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz"
	internal_vonage_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vonage"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type StreamerOptions struct {
	WebSocketConn *websocket.Conn

	AudioSocketConn   net.Conn
	AudioSocketReader *bufio.Reader
	AudioSocketWriter *bufio.Writer

	Context      context.Context
	SIPSession   *sip_infra.Session
	SIPLifecycle sip_infra.LifecycleController
	Observer     observability.Recorder
}

type StreamerFuncOption func(*StreamerOptions)

// WithWebSocketStreamer configures websocket media transport.
func WithWebSocketStreamer(conn *websocket.Conn) StreamerFuncOption {
	return func(streamerOptions *StreamerOptions) {
		streamerOptions.WebSocketConn = conn
	}
}

// WithAudioSocketStreamer configures Asterisk AudioSocket media transport.
func WithAudioSocketStreamer(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) StreamerFuncOption {
	return func(streamerOptions *StreamerOptions) {
		streamerOptions.AudioSocketConn = conn
		streamerOptions.AudioSocketReader = reader
		streamerOptions.AudioSocketWriter = writer
	}
}

// WithSIPStreamer configures SIP session-owned media transport.
func WithSIPStreamer(ctx context.Context, session *sip_infra.Session, lifecycle sip_infra.LifecycleController) StreamerFuncOption {
	return func(streamerOptions *StreamerOptions) {
		streamerOptions.Context = ctx
		streamerOptions.SIPSession = session
		streamerOptions.SIPLifecycle = lifecycle
	}
}

func WithObserver(observer observability.Recorder) StreamerFuncOption {
	return func(streamerOptions *StreamerOptions) {
		streamerOptions.Observer = observer
	}
}

func (at Telephony) NewStreamer(
	logger commons.Logger,
	callContext *callcontext.CallContext,
	vaultCredential *protos.VaultCredential,
	options ...StreamerFuncOption,
) (internal_type.Streamer, error) {
	var resolvedOptions StreamerOptions
	for _, option := range options {
		option(&resolvedOptions)
	}
	switch at {
	case Twilio:
		return internal_twilio_telephony.New(
			internal_twilio_telephony.WithLogger(logger),
			internal_twilio_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_twilio_telephony.WithCallContext(callContext),
			internal_twilio_telephony.WithVaultCredential(vaultCredential),
			internal_twilio_telephony.WithObserver(resolvedOptions.Observer),
		)
	case Exotel:
		return internal_exotel_telephony.New(
			internal_exotel_telephony.WithLogger(logger),
			internal_exotel_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_exotel_telephony.WithCallContext(callContext),
			internal_exotel_telephony.WithVaultCredential(vaultCredential),
			internal_exotel_telephony.WithObserver(resolvedOptions.Observer),
		)
	case Vonage:
		return internal_vonage_telephony.New(
			internal_vonage_telephony.WithLogger(logger),
			internal_vonage_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_vonage_telephony.WithCallContext(callContext),
			internal_vonage_telephony.WithVaultCredential(vaultCredential),
			internal_vonage_telephony.WithObserver(resolvedOptions.Observer),
		)
	case Asterisk:
		if resolvedOptions.AudioSocketConn != nil {
			return internal_asterisk_audiosocket.New(
				internal_asterisk_audiosocket.WithLogger(logger),
				internal_asterisk_audiosocket.WithConnection(resolvedOptions.AudioSocketConn),
				internal_asterisk_audiosocket.WithReader(resolvedOptions.AudioSocketReader),
				internal_asterisk_audiosocket.WithWriter(resolvedOptions.AudioSocketWriter),
				internal_asterisk_audiosocket.WithCallContext(callContext),
				internal_asterisk_audiosocket.WithVaultCredential(vaultCredential),
				internal_asterisk_audiosocket.WithObserver(resolvedOptions.Observer),
			)
		}
		return internal_asterisk_websocket.New(
			internal_asterisk_websocket.WithLogger(logger),
			internal_asterisk_websocket.WithConnection(resolvedOptions.WebSocketConn),
			internal_asterisk_websocket.WithCallContext(callContext),
			internal_asterisk_websocket.WithVaultCredential(vaultCredential),
			internal_asterisk_websocket.WithObserver(resolvedOptions.Observer),
		)
	case Telnyx:
		return internal_telnyx_telephony.New(
			internal_telnyx_telephony.WithLogger(logger),
			internal_telnyx_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_telnyx_telephony.WithCallContext(callContext),
			internal_telnyx_telephony.WithVaultCredential(vaultCredential),
			internal_telnyx_telephony.WithObserver(resolvedOptions.Observer),
		)
	case Vobiz:
		return internal_vobiz_telephony.New(
			internal_vobiz_telephony.WithLogger(logger),
			internal_vobiz_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_vobiz_telephony.WithCallContext(callContext),
			internal_vobiz_telephony.WithVaultCredential(vaultCredential),
			internal_vobiz_telephony.WithObserver(resolvedOptions.Observer),
		)
	case Plivo:
		return internal_plivo_telephony.New(
			internal_plivo_telephony.WithLogger(logger),
			internal_plivo_telephony.WithConnection(resolvedOptions.WebSocketConn),
			internal_plivo_telephony.WithCallContext(callContext),
			internal_plivo_telephony.WithVaultCredential(vaultCredential),
			internal_plivo_telephony.WithObserver(resolvedOptions.Observer),
		)
	case SIP:
		return New(
			WithSIPContext(resolvedOptions.Context),
			WithSIPLogger(logger),
			WithSIPSession(resolvedOptions.SIPSession),
			WithSIPLifecycle(resolvedOptions.SIPLifecycle),
			WithSIPCallContext(callContext),
			WithSIPVaultCredential(vaultCredential),
			WithSIPObserver(resolvedOptions.Observer),
		)
	default:
		return nil, fmt.Errorf("streamer not supported for provider %q", at)
	}
}

type SIPStreamerOptions struct {
	Context         context.Context
	Logger          commons.Logger
	Session         *sip_infra.Session
	Lifecycle       sip_infra.LifecycleController
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

type SIPStreamerFuncOption func(*SIPStreamerOptions)

func WithSIPContext(ctx context.Context) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.Context = ctx
	}
}

func WithSIPLogger(logger commons.Logger) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.Logger = logger
	}
}

func WithSIPSession(session *sip_infra.Session) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.Session = session
	}
}

func WithSIPLifecycle(lifecycle sip_infra.LifecycleController) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.Lifecycle = lifecycle
	}
}

func WithSIPCallContext(callContext *callcontext.CallContext) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.CallContext = callContext
	}
}

func WithSIPVaultCredential(vaultCredential *protos.VaultCredential) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.VaultCredential = vaultCredential
	}
}

func WithSIPObserver(observer observability.Recorder) SIPStreamerFuncOption {
	return func(options *SIPStreamerOptions) {
		options.Observer = observer
	}
}

func New(opts ...SIPStreamerFuncOption) (internal_type.SIPCallStreamer, error) {
	var options SIPStreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return internal_sip_telephony.New(
		internal_sip_telephony.WithContext(options.Context),
		internal_sip_telephony.WithLogger(options.Logger),
		internal_sip_telephony.WithSession(options.Session),
		internal_sip_telephony.WithLifecycle(options.Lifecycle),
		internal_sip_telephony.WithCallContext(options.CallContext),
		internal_sip_telephony.WithVaultCredential(options.VaultCredential),
		internal_sip_telephony.WithObserver(options.Observer),
	)
}
