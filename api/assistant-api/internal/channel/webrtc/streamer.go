// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/dtls/v3/pkg/crypto/elliptic"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// webrtcStreamer implements Streamer using Pion WebRTC for media and gRPC for signaling.
type webrtcStreamer struct {
	channel_base.BaseStreamer

	peerConfig   *webrtc_internal.Config
	serverConfig *assistant_config.WebRTCConfig
	grpcStream   grpc.BidiStreamingServer[protos.WebTalkRequest, protos.WebTalkResponse]

	sessionID                        string
	signalingSessionID               string
	signalOfferSent                  bool
	signalPendingLocalICECandidates  []*protos.ICECandidate
	signalPendingRemoteICECandidates []pionwebrtc.ICECandidateInit

	peerConnection      *pionwebrtc.PeerConnection
	assistantAudioTrack *pionwebrtc.TrackLocalStaticSample
	assistantRTPSender  *pionwebrtc.RTPSender
	resampler           internal_type.AudioResampler
	opusCodec           *webrtc_internal.OpusCodec

	mediaCtx     context.Context
	cancelMedia  context.CancelFunc
	mediaWorkers sync.WaitGroup

	streamModeTransitionMu sync.Mutex
	currentMode            protos.StreamMode

	sessionState      webrtc_internal.SessionState
	mediaHealthState  webrtc_internal.MediaHealthState
	peerEventCh       chan webrtc_internal.PeerEvent
	mediaLifecycleCh  chan webrtc_internal.MediaLifecycleEvent
	webrtcOperationCh chan webrtc_internal.WebRTCOperation

	ambientMixer internal_ambient.Mixer
	outputHealth *internal_output.HealthStats

	outputAudioQueueMu sync.Mutex
	outputAudioQueue   []webrtc_internal.OutputAudioFrame

	audioBufferState webrtc_internal.WebRTCAudioBufferState
	flushAudioCh     chan struct{}

	observer observability.Recorder

	auth                 types.SimplePrinciple
	configurationService internal_services.AssistantConfigurationService
	httpLogService       internal_services.AssistantHTTPLogService
	assistantToolService internal_services.AssistantToolService
}

type StreamerOptions struct {
	Context              context.Context
	Logger               commons.Logger
	GRPCStream           grpc.BidiStreamingServer[protos.WebTalkRequest, protos.WebTalkResponse]
	ServerConfig         *assistant_config.WebRTCConfig
	Observer             observability.Recorder
	Auth                 types.SimplePrinciple
	ConfigurationService internal_services.AssistantConfigurationService
	HTTPLogService       internal_services.AssistantHTTPLogService
	AssistantToolService internal_services.AssistantToolService
}

type FuncOption func(*StreamerOptions)

func WithContext(ctx context.Context) FuncOption {
	return func(options *StreamerOptions) {
		options.Context = ctx
	}
}

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

func WithServer(server grpc.BidiStreamingServer[protos.WebTalkRequest, protos.WebTalkResponse]) FuncOption {
	return func(options *StreamerOptions) {
		options.GRPCStream = server
	}
}

func WithServerConfig(serverConfig *assistant_config.WebRTCConfig) FuncOption {
	return func(options *StreamerOptions) {
		options.ServerConfig = serverConfig
	}
}

func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *StreamerOptions) {
		options.Observer = observer
	}
}

func WithAuth(auth types.SimplePrinciple) FuncOption {
	return func(options *StreamerOptions) {
		options.Auth = auth
	}
}

func WithAssistantConfigurationService(configurationService internal_services.AssistantConfigurationService) FuncOption {
	return func(options *StreamerOptions) {
		options.ConfigurationService = configurationService
	}
}

func WithHTTPLogService(httpLogService internal_services.AssistantHTTPLogService) FuncOption {
	return func(options *StreamerOptions) {
		options.HTTPLogService = httpLogService
	}
}

func WithAssistantToolService(assistantToolService internal_services.AssistantToolService) FuncOption {
	return func(options *StreamerOptions) {
		options.AssistantToolService = assistantToolService
	}
}

func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	resampler, err := internal_audio_resampler.GetResampler(options.Logger)
	if err != nil {
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "WebRTC streamer initialization failed",
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "resampler",
				"error":     err.Error(),
			},
		})
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCFailed,
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "resampler",
				"error":     err.Error(),
			},
		})
		return nil, fmt.Errorf("failed to create resampler: %w", err)
	}

	opusCodec, err := webrtc_internal.NewOpusCodec()
	if err != nil {
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "WebRTC streamer initialization failed",
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "opus_codec",
				"error":     err.Error(),
			},
		})
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCFailed,
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "opus_codec",
				"error":     err.Error(),
			},
		})
		return nil, fmt.Errorf("failed to create Opus codec: %w", err)
	}

	peerConfig := webrtc_internal.DefaultConfig()
	if options.ServerConfig != nil {
		if len(options.ServerConfig.ICEServers) > 0 {
			iceServers := make([]webrtc_internal.ICEServer, 0, len(options.ServerConfig.ICEServers))
			for _, server := range options.ServerConfig.ICEServers {
				if len(server.URLs) == 0 {
					continue
				}
				urls := make([]string, 0, len(server.URLs))
				for _, url := range server.URLs {
					url = strings.TrimSpace(os.ExpandEnv(url))
					if url != "" {
						urls = append(urls, url)
					}
				}
				if len(urls) == 0 {
					continue
				}
				iceServers = append(iceServers, webrtc_internal.ICEServer{
					URLs:       urls,
					Username:   os.ExpandEnv(server.Username),
					Credential: os.ExpandEnv(server.Credential),
				})
			}
			if len(iceServers) > 0 {
				peerConfig.ICEServers = iceServers
			}
		}

		switch strings.ToLower(strings.TrimSpace(options.ServerConfig.ICETransportPolicy)) {
		case "", webrtc_internal.ICETransportPolicyAll:
			peerConfig.ICETransportPolicy = webrtc_internal.ICETransportPolicyAll
		case webrtc_internal.ICETransportPolicyRelay:
			peerConfig.ICETransportPolicy = webrtc_internal.ICETransportPolicyRelay
		default:
			_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Invalid WebRTC ICE transport policy, using all",
				Attributes: observability.Attributes{
					"component": observability.ComponentWebRTC.String(),
					"policy":    options.ServerConfig.ICETransportPolicy,
				},
			})
			peerConfig.ICETransportPolicy = webrtc_internal.ICETransportPolicyAll
		}
	}
	ambientMixer, err := internal_ambient.NewLoopMixer(internal_ambient.MixerSpec{
		Logger:            options.Logger,
		Resampler:         resampler,
		TargetAudioConfig: internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG,
		FrameBytes:        webrtc_internal.WebRTCOutputPCM16kFrameBytes,
	})
	if err != nil {
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "WebRTC streamer initialization failed",
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "ambient_mixer",
				"error":     err.Error(),
			},
		})
		_ = options.Observer.Record(options.Context, observability.ProjectScope{}, observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCFailed,
			Attributes: observability.Attributes{
				"component": observability.ComponentWebRTC.String(),
				"stage":     "ambient_mixer",
				"error":     err.Error(),
			},
		})
		return nil, fmt.Errorf("failed to create ambient mixer: %w", err)
	}
	s := &webrtcStreamer{
		BaseStreamer: channel_base.NewBaseStreamerWithChannelCapacity(
			options.Logger,
			webrtc_internal.InputChannelSize,
			webrtc_internal.OutputChannelSize,
		),
		peerConfig:           peerConfig,
		serverConfig:         options.ServerConfig,
		grpcStream:           options.GRPCStream,
		sessionID:            uuid.New().String(),
		resampler:            resampler,
		opusCodec:            opusCodec,
		currentMode:          protos.StreamMode_STREAM_MODE_TEXT,
		sessionState:         webrtc_internal.SessionState{Scope: observability.ProjectScope{}},
		peerEventCh:          make(chan webrtc_internal.PeerEvent, webrtc_internal.PeerEventChannelSize),
		mediaLifecycleCh:     make(chan webrtc_internal.MediaLifecycleEvent, webrtc_internal.MediaLifecycleChannelSize),
		webrtcOperationCh:    make(chan webrtc_internal.WebRTCOperation, webrtc_internal.WebRTCOperationChannelSize),
		outputHealth:         internal_output.NewHealthStats(),
		audioBufferState:     newWebRTCAudioBufferState(),
		flushAudioCh:         make(chan struct{}, 1),
		observer:             options.Observer,
		auth:                 options.Auth,
		configurationService: options.ConfigurationService,
		httpLogService:       options.HTTPLogService,
		assistantToolService: options.AssistantToolService,
		ambientMixer:         ambientMixer,
	}
	_ = options.Observer.Record(options.Context, s.sessionState.Scope, observability.RecordEvent{
		Component: observability.ComponentWebRTC,
		Event:     observability.WebRTCConnecting,
		Attributes: observability.Attributes{
			"component":  observability.ComponentWebRTC.String(),
			"session_id": s.sessionID,
		},
	})
	go s.runGrpcReader()
	go s.runPeerEventLoop()
	go s.runMediaLifecycleLoop()
	go s.runWebRTCOperationLoop()
	go s.runOutputWriter()
	go s.runAudioPacer()
	go s.runOutputHealthReporter()
	go s.runHealthWatchdog()
	go s.watchCallerContext(options.Context)
	return s, nil
}

func (s *webrtcStreamer) Observer() observability.Recorder {
	return s.observer
}

func (s *webrtcStreamer) stopMediaSession() {
	s.sessionState.InvalidateMediaSession()

	s.Mu.Lock()
	peerConnection := s.peerConnection
	if s.cancelMedia != nil {
		s.cancelMedia()
		s.cancelMedia = nil
	}
	s.mediaCtx = nil
	s.peerConnection = nil
	s.assistantAudioTrack = nil
	s.assistantRTPSender = nil
	s.Mu.Unlock()

	if peerConnection != nil {
		peerConnection.Close()
	}
	s.mediaWorkers.Wait()
}

func (s *webrtcStreamer) createPeer(mediaSessionID uint64) error {
	s.Mu.Lock()
	s.mediaCtx, s.cancelMedia = context.WithCancel(s.Ctx)
	s.Mu.Unlock()
	s.sessionState.SetPeerConnected(false)

	mediaEngine := &pionwebrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:    pionwebrtc.MimeTypeOpus,
			ClockRate:   webrtc_internal.OpusSampleRate,
			Channels:    webrtc_internal.OpusChannels,
			SDPFmtpLine: webrtc_internal.OpusSDPFmtpLine,
			RTCPFeedback: []pionwebrtc.RTCPFeedback{
				{Type: webrtc_internal.RTCPFeedbackNACK},
			},
		},
		PayloadType: webrtc_internal.OpusPayloadType,
	}, pionwebrtc.RTPCodecTypeAudio); err != nil {
		return fmt.Errorf("failed to register Opus codec: %w", err)
	}

	registry := &interceptor.Registry{}
	if err := pionwebrtc.RegisterDefaultInterceptors(mediaEngine, registry); err != nil {
		return fmt.Errorf("failed to register interceptors: %w", err)
	}

	settingEngine := pionwebrtc.SettingEngine{}
	settingEngine.SetDTLSEllipticCurves(elliptic.X25519, elliptic.P384, elliptic.P256)
	settingEngine.DisableCloseByDTLS(true)
	settingEngine.SetFireOnTrackBeforeFirstRTP(true)
	settingEngine.SetDTLSRetransmissionInterval(webrtc_internal.DTLSRetransmissionInterval)
	settingEngine.SetDTLSConnectContextMaker(func() (context.Context, func()) {
		return context.WithTimeout(context.Background(), webrtc_internal.DTLSHandshakeTimeout)
	})
	settingEngine.SetICETimeouts(
		webrtc_internal.ICEDisconnectedTimeout,
		webrtc_internal.ICEFailedTimeout,
		webrtc_internal.ICEKeepaliveInterval,
	)
	if s.serverConfig != nil {
		if s.serverConfig.ExternalIP != "" {
			if err := settingEngine.SetICEAddressRewriteRules(pionwebrtc.ICEAddressRewriteRule{
				External:        []string{s.serverConfig.ExternalIP},
				AsCandidateType: pionwebrtc.ICECandidateTypeHost,
			}); err != nil {
				return fmt.Errorf("failed to set ICE address rewrite rules: %w", err)
			}
		}
		if s.serverConfig.UDPPortRangeStart > 0 && s.serverConfig.UDPPortRangeEnd > 0 {
			if err := settingEngine.SetEphemeralUDPPortRange(
				uint16(s.serverConfig.UDPPortRangeStart),
				uint16(s.serverConfig.UDPPortRangeEnd),
			); err != nil {
				return fmt.Errorf("failed to set UDP port range: %w", err)
			}
		}
	}

	api := pionwebrtc.NewAPI(
		pionwebrtc.WithMediaEngine(mediaEngine),
		pionwebrtc.WithInterceptorRegistry(registry),
		pionwebrtc.WithSettingEngine(settingEngine),
	)

	iceServers := make([]pionwebrtc.ICEServer, len(s.peerConfig.ICEServers))
	for i, srv := range s.peerConfig.ICEServers {
		iceServers[i] = pionwebrtc.ICEServer{
			URLs:       srv.URLs,
			Username:   srv.Username,
			Credential: srv.Credential,
		}
	}

	peerConnectionConfig := pionwebrtc.Configuration{ICEServers: iceServers}
	if s.peerConfig.ICETransportPolicy == webrtc_internal.ICETransportPolicyRelay {
		peerConnectionConfig.ICETransportPolicy = pionwebrtc.ICETransportPolicyRelay
	}

	peerConnection, err := api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	s.Mu.Lock()
	s.peerConnection = peerConnection
	s.Mu.Unlock()

	s.bindPeerHandlers(peerConnection, mediaSessionID)
	return s.createAssistantAudioTrack(peerConnection, mediaSessionID)
}

func (s *webrtcStreamer) bindPeerHandlers(peerConnection *pionwebrtc.PeerConnection, mediaSessionID uint64) {
	peerConnection.OnICECandidate(func(candidate *pionwebrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		s.queueLocalICECandidate(candidate.ToJSON(), mediaSessionID)
	})

	peerConnection.OnICEGatheringStateChange(func(state pionwebrtc.ICEGatheringState) {
		if state != pionwebrtc.ICEGatheringStateComplete || !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}
		s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
			Kind:           webrtc_internal.WebRTCOperationICEGatheringComplete,
			MediaSessionID: mediaSessionID,
		})
	})

	peerConnection.OnConnectionStateChange(func(state pionwebrtc.PeerConnectionState) {
		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}

		s.enqueuePeerEvent(webrtc_internal.PeerEvent{
			Kind:               webrtc_internal.PeerEventStateChanged,
			MediaSessionID:     mediaSessionID,
			PeerState:          state,
			PeerStateChangedAt: time.Now(),
		})
	})

	peerConnection.OnICEConnectionStateChange(func(state pionwebrtc.ICEConnectionState) {
		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}

		s.enqueuePeerEvent(webrtc_internal.PeerEvent{
			Kind:                  webrtc_internal.PeerEventICEConnectionStateChanged,
			MediaSessionID:        mediaSessionID,
			PeerICEState:          state,
			PeerICEStateChangedAt: time.Now(),
		})
	})

	peerConnection.OnTrack(func(track *pionwebrtc.TrackRemote, rtpReceiver *pionwebrtc.RTPReceiver) {
		if track.Kind() != pionwebrtc.RTPCodecTypeAudio {
			return
		}

		remoteAudioTrack := webrtc_internal.WebRTCRemoteAudioTrack{
			TrackCodec: track.Codec(),
		}
		if rtpReceiver != nil {
			remoteAudioTrack.ReceiverCodecs = rtpReceiver.GetParameters().Codecs
		}
		remoteAudioCodec, ok := remoteAudioTrack.SelectedCodec()
		if !ok {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "No negotiated audio codec for WebRTC remote track",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
				},
			})
			return
		}
		if !strings.EqualFold(remoteAudioCodec.MimeType, pionwebrtc.MimeTypeOpus) {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Unsupported codec, only Opus is supported",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataCodec:     remoteAudioCodec.MimeType,
				},
			})
			return
		}
		if !s.tryStartRemoteAudioReader(peerConnection, mediaSessionID) {
			return
		}

		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordEvent{
			Component: observability.ComponentWebRTC,
			Event:     observability.WebRTCAudioTrackReceived,
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:      "audio_track_received",
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataCodec:     remoteAudioCodec.MimeType,
			},
		})
		if s.Logger != nil {
			s.Logger.Debugw("webrtc webhook record call site",
				"call_site", "audio_track_received",
				"event", observability.WebRTCAudioTrackReceived.String(),
				webrtc_internal.DataSessionID, s.sessionID,
				webrtc_internal.DataMediaSessionID, mediaSessionID,
				webrtc_internal.DataCodec, remoteAudioCodec.MimeType,
			)
		}
		if err := s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordWebhook{
			Event: observability.WebRTCAudioTrackReceived,
			Payload: map[string]interface{}{
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: mediaSessionID,
				webrtc_internal.DataCodec:          remoteAudioCodec.MimeType,
			},
		}); err != nil && s.Logger != nil {
			s.Logger.Warnw("webrtc webhook record failed",
				"call_site", "audio_track_received",
				"event", observability.WebRTCAudioTrackReceived.String(),
				webrtc_internal.DataSessionID, s.sessionID,
				webrtc_internal.DataMediaSessionID, mediaSessionID,
				"error", err,
			)
		}
		go s.readRemoteAudio(track, mediaSessionID, remoteAudioCodec)
	})
}

func (s *webrtcStreamer) tryStartRemoteAudioReader(peerConnection *pionwebrtc.PeerConnection, mediaSessionID uint64) bool {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if !s.sessionState.IsActiveMediaSession(mediaSessionID) || s.peerConnection != peerConnection {
		return false
	}
	if !s.sessionState.TryStartRemoteAudioReader(mediaSessionID) {
		return false
	}
	s.mediaWorkers.Add(1)
	return true
}

func (s *webrtcStreamer) queueLocalICECandidate(candidateInit pionwebrtc.ICECandidateInit, mediaSessionID uint64) {
	if candidateInit.Candidate == "" || !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendLocalICECandidate,
		MediaSessionID:    mediaSessionID,
		LocalICECandidate: candidateInit,
	})
}

func (s *webrtcStreamer) createAssistantAudioTrack(peerConnection *pionwebrtc.PeerConnection, mediaSessionID uint64) error {
	assistantAudioTrack, err := pionwebrtc.NewTrackLocalStaticSample(
		pionwebrtc.RTPCodecCapability{
			MimeType:  pionwebrtc.MimeTypeOpus,
			ClockRate: webrtc_internal.OpusSampleRate,
			Channels:  webrtc_internal.OpusChannels,
			RTCPFeedback: []pionwebrtc.RTCPFeedback{
				{Type: webrtc_internal.RTCPFeedbackNACK},
			},
		},
		"audio",
		"rapida-audio",
	)
	if err != nil {
		return fmt.Errorf("failed to create local audio track: %w", err)
	}

	assistantRTPSender, err := peerConnection.AddTrack(assistantAudioTrack)
	if err != nil {
		return fmt.Errorf("failed to add track: %w", err)
	}

	s.Mu.Lock()
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) || s.peerConnection != peerConnection {
		s.Mu.Unlock()
		return nil
	}
	s.assistantAudioTrack = assistantAudioTrack
	s.assistantRTPSender = assistantRTPSender
	s.mediaWorkers.Add(1)
	s.Mu.Unlock()

	go s.readAssistantRTCP(assistantRTPSender, mediaSessionID)
	return nil
}

// readRemoteAudio converts browser Opus/48k inputAudioBuffer into Rapida PCM16k audio.
func (s *webrtcStreamer) readRemoteAudio(track *pionwebrtc.TrackRemote, mediaSessionID uint64, remoteAudioCodec pionwebrtc.RTPCodecParameters) {
	defer s.mediaWorkers.Done()

	s.Mu.Lock()
	mediaCtx := s.mediaCtx
	s.Mu.Unlock()

	if mediaCtx == nil {
		return
	}
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	if !strings.EqualFold(remoteAudioCodec.MimeType, pionwebrtc.MimeTypeOpus) {
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Unsupported codec, only Opus is supported",
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
				webrtc_internal.DataCodec:          remoteAudioCodec.MimeType,
			},
		})
		return
	}

	opusDecoder, err := webrtc_internal.NewOpusDecoder()
	if err != nil {
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "Failed to create Opus decoder",
			Attributes: observability.Attributes{
				"component":                        observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
				"error":                            err.Error(),
			},
		})
		return
	}

	buf := make([]byte, webrtc_internal.RTPBufferSize)
	pkt := &rtp.Packet{}
	consecutiveErrors := 0

	for {
		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}
		select {
		case <-mediaCtx.Done():
			return
		default:
		}

		n, _, err := track.Read(buf)
		if err != nil {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			if errors.Is(err, io.EOF) {
				return
			}
			consecutiveErrors++
			s.Mu.Lock()
			s.mediaHealthState.RecordUserAudioReadError(consecutiveErrors)
			s.Mu.Unlock()
			if consecutiveErrors >= webrtc_internal.MaxConsecutiveReadErrors {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
					Level:   observability.LevelError,
					Message: "Too many consecutive read errors, stopping audio reader",
					Attributes: observability.Attributes{
						"component":                        observability.ComponentWebRTC.String(),
						webrtc_internal.DataSessionID:      s.sessionID,
						webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
						"consecutive_errors":               fmt.Sprintf("%d", consecutiveErrors),
						"error":                            err.Error(),
					},
				})
				return
			}
			continue
		}
		consecutiveErrors = 0
		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}
		s.Mu.Lock()
		s.mediaHealthState.RecordUserAudioReadRecovered()
		s.Mu.Unlock()

		if err := pkt.Unmarshal(buf[:n]); err != nil {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			s.Mu.Lock()
			s.mediaHealthState.RecordUserAudioRTPUnmarshalFailure()
			s.Mu.Unlock()
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Failed to unmarshal RTP packet",
				Attributes: observability.Attributes{
					"component":                        observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID:      s.sessionID,
					webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
					"error":                            err.Error(),
				},
			})
			continue
		}
		if len(pkt.Payload) == 0 {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			s.Mu.Lock()
			s.mediaHealthState.RecordUserAudioEmptyRTPPayload()
			s.Mu.Unlock()
			continue
		}

		userPCM48k, err := opusDecoder.Decode(pkt.Payload)
		if err != nil {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			s.Mu.Lock()
			s.mediaHealthState.RecordUserAudioOpusDecodeFailure()
			s.Mu.Unlock()
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Opus decode failed",
				Attributes: observability.Attributes{
					"component":                        observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID:      s.sessionID,
					webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
					"payload_size":                     fmt.Sprintf("%d", len(pkt.Payload)),
					"error":                            err.Error(),
				},
			})
			continue
		}
		userPCM16k, err := s.resampler.Resample(userPCM48k, internal_audio.WEBRTC_AUDIO_CONFIG, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG)
		if err != nil {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			s.Mu.Lock()
			s.mediaHealthState.RecordUserAudioResampleFailure()
			s.Mu.Unlock()
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Audio resample failed",
				Attributes: observability.Attributes{
					"component":                        observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID:      s.sessionID,
					webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
					"error":                            err.Error(),
				},
			})
			continue
		}
		userAudioReceivedAt := time.Now()

		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}
		s.Mu.Lock()
		s.mediaHealthState.RecordUserAudioReceived(userAudioReceivedAt)
		s.Mu.Unlock()
		s.bufferAndSendInput(userPCM16k, userAudioReceivedAt)
	}
}

func (s *webrtcStreamer) readAssistantRTCP(assistantRTPSender *pionwebrtc.RTPSender, mediaSessionID uint64) {
	defer s.mediaWorkers.Done()

	s.Mu.Lock()
	mediaCtx := s.mediaCtx
	s.Mu.Unlock()

	if mediaCtx == nil {
		return
	}
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	for {
		if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
			return
		}
		select {
		case <-mediaCtx.Done():
			return
		default:
		}

		rtcpPackets, _, err := assistantRTPSender.ReadRTCP()
		if err != nil {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return
			}
			select {
			case <-mediaCtx.Done():
				return
			default:
			}
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Failed to read WebRTC RTCP feedback",
				Attributes: observability.Attributes{
					"component":                        observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID:      s.sessionID,
					webrtc_internal.DataMediaSessionID: fmt.Sprintf("%d", mediaSessionID),
					"error":                            err.Error(),
				},
			})
			continue
		}

		receiverReportReceivedAt := time.Now()
		for _, rtcpPacket := range rtcpPackets {
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
				return
			}
			receiverReport, ok := rtcpPacket.(*rtcp.ReceiverReport)
			if !ok {
				continue
			}
			for _, receptionReport := range receiverReport.Reports {
				if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
					return
				}
				s.Mu.Lock()
				s.mediaHealthState.RecordReceiverReport(
					receiverReportReceivedAt,
					receptionReport.FractionLost,
					receptionReport.TotalLost,
					receptionReport.Jitter,
					receptionReport.LastSenderReport,
					receptionReport.Delay,
				)
				s.Mu.Unlock()
			}
		}
	}
}

// writeAudioFrame writes encoded Opus to the assistant WebRTC track.
func (s *webrtcStreamer) writeAudioFrame(data []byte) error {
	s.Mu.Lock()
	assistantAudioTrack := s.assistantAudioTrack
	s.Mu.Unlock()

	if assistantAudioTrack == nil {
		return errors.New("WebRTC assistant audio track is not ready")
	}
	if err := assistantAudioTrack.WriteSample(media.Sample{
		Data:     data,
		Duration: webrtc_internal.OpusFrameDuration * time.Millisecond,
	}); err != nil {
		return fmt.Errorf("failed to write WebRTC assistant audio frame: %w", err)
	}
	return nil
}

func (s *webrtcStreamer) signalConfig() {
	iceServers := make([]*protos.ICEServer, len(s.peerConfig.ICEServers))
	for i, srv := range s.peerConfig.ICEServers {
		iceServers[i] = &protos.ICEServer{
			Urls:       srv.URLs,
			Username:   srv.Username,
			Credential: srv.Credential,
		}
	}

	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	s.Mu.Unlock()
	if signalingSessionID == "" {
		signalingSessionID = s.sessionID
	}

	s.Output(&protos.ServerSignaling{
		SessionId: signalingSessionID,
		Message: &protos.ServerSignaling_Config{
			Config: &protos.WebRTCConfig{
				IceServers: iceServers,
				AudioCodec: "opus",
				SampleRate: int32(webrtc_internal.OpusSampleRate),
			},
		},
	})
}

func (s *webrtcStreamer) signalOffer(sdp string) {
	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	s.Mu.Unlock()
	if signalingSessionID == "" {
		signalingSessionID = s.sessionID
	}

	s.Output(&protos.ServerSignaling{
		SessionId: signalingSessionID,
		Message: &protos.ServerSignaling_Sdp{
			Sdp: &protos.WebRTCSDP{
				Type: protos.WebRTCSDP_OFFER,
				Sdp:  sdp,
			},
		},
	})
}

func (s *webrtcStreamer) signalReady() {
	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	s.Mu.Unlock()
	if signalingSessionID == "" {
		signalingSessionID = s.sessionID
	}

	s.Output(&protos.ServerSignaling{
		SessionId: signalingSessionID,
		Message:   &protos.ServerSignaling_Ready{Ready: true},
	})
}

func (s *webrtcStreamer) signalClear() {
	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	s.Mu.Unlock()
	if signalingSessionID == "" {
		signalingSessionID = s.sessionID
	}

	s.Output(&protos.ServerSignaling{
		SessionId: signalingSessionID,
		Message:   &protos.ServerSignaling_Clear{Clear: true},
	})
}

func (s *webrtcStreamer) applyAmbientConfig(cfg internal_ambient.Config, source string) {
	if s.ambientMixer == nil {
		return
	}
	if err := s.ambientMixer.Configure(cfg); err != nil {
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC ambient configuration ignored",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				"source":                      source,
				"error":                       err.Error(),
			},
		})
		return
	}
}

func (s *webrtcStreamer) applyAmbientToFrame(primary []byte) []byte {
	if s.ambientMixer == nil {
		return primary
	}
	out, err := s.ambientMixer.Mix(primary)
	if err != nil {
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC ambient mix failed",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				"error":                       err.Error(),
			},
		})
		return primary
	}
	return out
}

func (s *webrtcStreamer) enqueueOutputAudio(frame []byte) {
	if len(frame) == 0 {
		return
	}
	assistantAudioQueuedAt := time.Now()
	outputFrame := webrtc_internal.OutputAudioFrame{
		Audio:    append([]byte(nil), frame...),
		QueuedAt: assistantAudioQueuedAt,
	}

	droppedFrames := 0
	queueDepth := 0

	s.outputAudioQueueMu.Lock()
	limit := webrtc_internal.OutputAudioQueueMaxFrames
	if limit > 0 && len(s.outputAudioQueue) >= limit {
		s.outputAudioQueue[0] = webrtc_internal.OutputAudioFrame{}
		copy(s.outputAudioQueue[0:], s.outputAudioQueue[1:])
		s.outputAudioQueue[len(s.outputAudioQueue)-1] = outputFrame
		droppedFrames = webrtc_internal.OutputAudioDropOldestSize
	} else {
		s.outputAudioQueue = append(s.outputAudioQueue, outputFrame)
	}
	queueDepth = len(s.outputAudioQueue)
	s.outputAudioQueueMu.Unlock()

	s.Mu.Lock()
	s.mediaHealthState.RecordAssistantAudioQueued(assistantAudioQueuedAt)
	s.Mu.Unlock()

	if droppedFrames > 0 {
		totalDropped := s.sessionState.AddOutputAudioDroppedFrames(droppedFrames)
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "WebRTC output queue overflow dropped the oldest assistant audio frame; this keeps playback current when audio is produced faster than WebRTC can send it.",
			Attributes: observability.Attributes{
				"component":                            observability.ComponentWebRTC.String(),
				webrtc_internal.DataType:               webrtc_internal.EventOutputQueueOverflow,
				webrtc_internal.DataSessionID:          s.sessionID,
				webrtc_internal.DataPolicy:             webrtc_internal.OutputQueuePolicyDropOldest,
				webrtc_internal.DataDroppedFrames:      fmt.Sprintf("%d", droppedFrames),
				webrtc_internal.DataLimitFrames:        fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueMaxFrames),
				webrtc_internal.DataQueueDepthFrames:   fmt.Sprintf("%d", queueDepth),
				webrtc_internal.DataTotalDroppedFrames: fmt.Sprintf("%d", totalDropped),
			},
		}, observability.RecordMetric{
			Metrics: []*protos.Metric{{
				Name:        observability.MetricWebRTCOutputQueueDrops,
				Value:       fmt.Sprintf("%d", droppedFrames),
				Description: "WebRTC output queue dropped frames",
			}},
		})
	}
}

func (s *webrtcStreamer) popOutputAudio() []byte {
	s.outputAudioQueueMu.Lock()
	defer s.outputAudioQueueMu.Unlock()
	if len(s.outputAudioQueue) == 0 {
		return nil
	}
	outputFrame := s.outputAudioQueue[0]
	s.outputAudioQueue[0] = webrtc_internal.OutputAudioFrame{}
	s.outputAudioQueue = s.outputAudioQueue[1:]
	return outputFrame.Audio
}

func (s *webrtcStreamer) clearOutputAudio() int {
	s.outputAudioQueueMu.Lock()
	clearedFrames := len(s.outputAudioQueue)
	for i := range s.outputAudioQueue {
		s.outputAudioQueue[i] = webrtc_internal.OutputAudioFrame{}
	}
	s.outputAudioQueue = s.outputAudioQueue[:0]
	s.outputAudioQueueMu.Unlock()
	return clearedFrames
}

func (s *webrtcStreamer) NextFrame() []byte {
	if !s.sessionState.PeerConnected() {
		return nil
	}
	mediaSessionID := s.sessionState.ActiveMediaSessionID()
	if mediaSessionID == 0 {
		return nil
	}
	frame := s.popOutputAudio()
	if len(frame) == 0 {
		return nil
	}
	s.sessionState.StampPacedAssistantFrame(mediaSessionID)
	return s.applyAmbientToFrame(frame)
}

func (s *webrtcStreamer) IdleFrame() []byte {
	if !s.sessionState.PeerConnected() {
		return nil
	}
	mediaSessionID := s.sessionState.ActiveMediaSessionID()
	if mediaSessionID == 0 {
		return nil
	}
	s.sessionState.StampPacedAssistantFrame(mediaSessionID)
	return s.applyAmbientToFrame(nil)
}

func (s *webrtcStreamer) ConsumeFrame(assistantPCM16k []byte) error {
	if !s.sessionState.CanWritePacedAssistantFrame() {
		return nil
	}

	assistantPCM48k, err := s.resampler.Resample(assistantPCM16k, internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG, internal_audio.WEBRTC_AUDIO_CONFIG)
	if err != nil {
		return err
	}
	assistantOpus, err := s.opusCodec.Encode(assistantPCM48k)
	if err != nil {
		return err
	}

	if !s.sessionState.CanWritePacedAssistantFrame() {
		return nil
	}
	if err := s.writeAudioFrame(assistantOpus); err != nil {
		s.Mu.Lock()
		s.mediaHealthState.RecordAssistantFrameWriteFailure(time.Now())
		s.Mu.Unlock()
		return err
	}
	assistantFrameSentAt := time.Now()
	s.Input(&protos.ConversationBridgeOperatorAudio{
		Audio: assistantPCM16k,
		Time:  timestamppb.New(assistantFrameSentAt),
	})
	s.Mu.Lock()
	s.mediaHealthState.RecordAssistantFrameSent(assistantFrameSentAt)
	s.Mu.Unlock()
	return nil
}

// handleConfigurationMessage switches the WebRTC media mode without changing the conversation.
func (s *webrtcStreamer) handleConfigurationMessage(mode protos.StreamMode) {
	s.streamModeTransitionMu.Lock()
	defer s.streamModeTransitionMu.Unlock()

	s.Mu.Lock()
	currentMode := s.currentMode
	s.Mu.Unlock()

	if mode == protos.StreamMode_STREAM_MODE_UNSPECIFIED {
		return
	}

	mediaState := s.sessionState.MediaState()
	switch mode {
	case protos.StreamMode_STREAM_MODE_AUDIO:
		if mediaState == webrtc_internal.MediaStateAudioNegotiating || mediaState == webrtc_internal.MediaStateAudioConnected {
			return
		}
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
		s.clearBufferedOutputAudio()
		s.signalClear()
		s.sessionState.ResetMediaRestartAttempts()
		if err := s.startMediaSession(); err != nil {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Error while starting WebRTC media session",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"error":                       err.Error(),
				},
			})
			if s.Logger != nil {
				s.Logger.Debugw("webrtc webhook record call site",
					"call_site", "media_session_start_failed",
					"event", observability.WebRTCFailed.String(),
					webrtc_internal.DataSessionID, s.sessionID,
					webrtc_internal.DataReason, "start_media_session",
					"error", err.Error(),
				)
			}
			if recordErr := s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordWebhook{
				Event: observability.WebRTCFailed,
				Payload: map[string]interface{}{
					webrtc_internal.DataType:      "media_session_start_failed",
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataReason:    "start_media_session",
					"error":                       err.Error(),
					"fallback":                    "text",
				},
			}); recordErr != nil && s.Logger != nil {
				s.Logger.Warnw("webrtc webhook record failed",
					"call_site", "media_session_start_failed",
					"event", observability.WebRTCFailed.String(),
					webrtc_internal.DataSessionID, s.sessionID,
					"error", recordErr,
				)
			}
			s.stopMediaSessionAndFallbackToText()
		}
	case protos.StreamMode_STREAM_MODE_TEXT:
		if currentMode == protos.StreamMode_STREAM_MODE_TEXT && mediaState == webrtc_internal.MediaStateText {
			return
		}
		s.clearBufferedOutputAudio()
		s.signalClear()
		s.stopMediaSessionAndFallbackToText()
	}
}

func (s *webrtcStreamer) queueClientSignal(signaling *protos.ClientSignaling) {
	if signaling == nil {
		return
	}
	s.enqueuePeerEvent(webrtc_internal.PeerEvent{
		Kind:                webrtc_internal.SignalEventClientMessage,
		SignalClientMessage: signaling,
	})
}

func (s *webrtcStreamer) stopMediaSessionAndFallbackToText() {
	s.clearBufferedOutputAudio()
	s.clearOutputAudio()
	if s.ambientMixer != nil {
		s.ambientMixer.Reset()
	}

	s.stopMediaSession()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.signalingSessionID = ""
	s.signalOfferSent = false
	s.signalPendingLocalICECandidates = nil
	s.signalPendingRemoteICECandidates = nil
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT
	s.sessionState.SetMediaState(webrtc_internal.MediaStateText)
	s.mediaHealthState.Reset()
	s.sessionState.SetPeerConnected(false)
}

// startMediaSession creates a fresh WebRTC peer connection and starts SDP negotiation.
func (s *webrtcStreamer) startMediaSession() error {
	s.stopMediaSession()

	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.ResetICERestartAttempts()
	s.Mu.Lock()
	s.signalingSessionID = uuid.New().String()
	s.signalOfferSent = false
	s.signalPendingLocalICECandidates = nil
	s.signalPendingRemoteICECandidates = nil
	s.mediaHealthState.StartICE(time.Now())
	s.Mu.Unlock()

	if err := s.createPeer(mediaSessionID); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
		Kind:              webrtc_internal.WebRTCOperationSendOffer,
		MediaSessionID:    mediaSessionID,
		SignalMediaConfig: true,
	})
	go s.runMediaSessionDeadlines(mediaSessionID)
	return nil
}

// sendWebRTCOffer sends an SDP offer; initial media negotiation also sends media config.
func (s *webrtcStreamer) sendWebRTCOffer(operation webrtc_internal.WebRTCOperation) (bool, error) {
	mediaSessionID := operation.MediaSessionID
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return false, nil
	}

	s.Mu.Lock()
	peerConnection := s.peerConnection
	if peerConnection == nil {
		s.Mu.Unlock()
		return false, fmt.Errorf("WebRTC media session is not ready")
	}
	offerOptions := operation.OfferOptions
	iceRestart := offerOptions != nil && offerOptions.ICERestart
	negotiationStarted, retryPending := s.sessionState.BeginNegotiation(iceRestart)
	if !negotiationStarted {
		s.Mu.Unlock()
		if retryPending {
			s.emitWebRTCNegotiationEvent(webrtc_internal.EventNegotiationRetryQueued, operation, iceRestart, true, time.Now())
		}
		return false, nil
	}
	s.signalOfferSent = false
	if iceRestart {
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
		s.mediaHealthState.StartICERestart(time.Now())
	}
	s.Mu.Unlock()

	if operation.SignalMediaConfig {
		s.signalConfig()
	}

	offer, err := peerConnection.CreateOffer(offerOptions)
	if err != nil {
		s.clearNegotiationState(peerConnection)
		return false, fmt.Errorf("failed to create offer: %w", err)
	}
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return false, nil
	}
	if err := peerConnection.SetLocalDescription(offer); err != nil {
		s.clearNegotiationState(peerConnection)
		return false, fmt.Errorf("failed to set local description: %w", err)
	}
	s.sessionState.SetICEGatheringActive(true)
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return false, nil
	}
	localDescription := peerConnection.LocalDescription()
	if localDescription == nil {
		return false, fmt.Errorf("local description is nil after local offer")
	}
	offerSentAt := time.Now()
	s.Mu.Lock()
	if s.sessionState.IsActiveMediaSession(mediaSessionID) {
		s.mediaHealthState.RecordOfferSent(offerSentAt)
	}
	s.Mu.Unlock()
	s.signalOffer(localDescription.SDP)

	s.Mu.Lock()
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		s.Mu.Unlock()
		return false, nil
	}
	signalingSessionID := s.signalingSessionID
	s.signalOfferSent = true
	signalPendingLocalICECandidates := append([]*protos.ICECandidate(nil), s.signalPendingLocalICECandidates...)
	s.signalPendingLocalICECandidates = nil
	s.Mu.Unlock()
	if signalingSessionID == "" {
		signalingSessionID = s.sessionID
	}

	for _, candidate := range signalPendingLocalICECandidates {
		s.Output(&protos.ServerSignaling{
			SessionId: signalingSessionID,
			Message: &protos.ServerSignaling_IceCandidate{
				IceCandidate: candidate,
			},
		})
	}

	s.emitWebRTCNegotiationEvent(webrtc_internal.EventNegotiationOfferSent, operation, iceRestart, false, offerSentAt)
	return true, nil
}

func (s *webrtcStreamer) clearNegotiationState(peerConnection *pionwebrtc.PeerConnection) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.peerConnection != peerConnection {
		return
	}
	s.sessionState.ResetNegotiation()
	s.sessionState.SetICEGatheringActive(false)
	s.signalPendingLocalICECandidates = nil
	s.signalOfferSent = true
}

func (s *webrtcStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			s.bufferAndSendOutput(content.Audio)
			return nil
		case *protos.ConversationAssistantMessage_Text:
			s.Output(data)
		}
	case *protos.ConversationConfiguration:
		s.handleConfigurationMessage(data.GetStreamMode())
		s.Output(data)
	case *protos.ConversationInitialization:
		s.sessionState.ChangeScope(observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: data.GetAssistant().GetAssistantId()},
			ConversationID: data.GetAssistantConversationId(),
		})
		s.handleConfigurationMessage(data.GetStreamMode())
		if ambientCfg, ok := internal_ambient.ParseFromInitialization(data); ok {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Parsed ambient configuration from initialization message",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"source":                      "server_initialization",
				},
			})
			s.applyAmbientConfig(ambientCfg, "server_initialization")
		}
		s.Output(data)
	case *protos.ConversationUserMessage:
		s.Output(data)
	case *protos.ConversationInterruption:
		s.clearBufferedOutputAudio()
		clearedFrames := s.clearOutputAudio()
		if clearedFrames > 0 {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "WebRTC output queue cleared after user interruption; this drops queued assistant audio so the response stops promptly when the user speaks.",
				Attributes: observability.Attributes{
					"component":                              observability.ComponentWebRTC.String(),
					webrtc_internal.DataType:                 webrtc_internal.EventOutputQueueCleared,
					webrtc_internal.DataSessionID:            s.sessionID,
					webrtc_internal.DataReason:               webrtc_internal.OutputQueueClearReasonInterruption,
					webrtc_internal.DataClearedFrames:        fmt.Sprintf("%d", clearedFrames),
					webrtc_internal.DataRemainingQueueFrames: fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueEmptySize),
				},
			})
		}
		s.signalClear()
		s.Output(data)
	case *protos.ConversationToolCall:
		s.Output(data)
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			s.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// WebRTC has no PSTN/SIP leg to transfer.
			s.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for WebRTC", "next_action": "end_call"},
			})
		}
	case *protos.ConversationError:
		s.Output(data)
	case *protos.ConversationEvent:
		s.Output(data)
	case *protos.ConversationMetadata:
		s.Output(data)
	case *protos.ConversationDisconnection:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "WebRTC streamer closing from ConversationDisconnection",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				"disconnection_type":          data.GetType().String(),
			},
		})
		_ = s.Disconnect(data.GetType())
		s.Output(data)
		s.Close()
	case *protos.ConversationMetric:
		s.Output(data)
	default:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "Unknown WebRTC send message type, skipping",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataType:      fmt.Sprintf("%T", response),
			},
		})
	}
	return nil
}

// Close releases WebRTC media and conversation resources once.
func (s *webrtcStreamer) Close() error {
	if !s.sessionState.BeginClose() {
		return nil
	}
	mediaSessionID := s.sessionState.ActiveMediaSessionID()
	if mediaSessionID != 0 {
		if s.Logger != nil {
			s.Logger.Debugw("webrtc webhook record call site",
				"call_site", "close_disconnected",
				"event", observability.WebRTCDisconnected.String(),
				webrtc_internal.DataSessionID, s.sessionID,
				webrtc_internal.DataMediaSessionID, mediaSessionID,
			)
		}
		if err := s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordWebhook{
			Event: observability.WebRTCDisconnected,
			Payload: map[string]interface{}{
				webrtc_internal.DataSessionID:      s.sessionID,
				webrtc_internal.DataMediaSessionID: mediaSessionID,
				webrtc_internal.DataReason:         "closed",
			},
		}); err != nil && s.Logger != nil {
			s.Logger.Warnw("webrtc webhook record failed",
				"call_site", "close_disconnected",
				"event", observability.WebRTCDisconnected.String(),
				webrtc_internal.DataSessionID, s.sessionID,
				webrtc_internal.DataMediaSessionID, mediaSessionID,
				"error", err,
			)
		}
	}
	s.stopMediaSession()

	s.Cancel()
	return nil
}
