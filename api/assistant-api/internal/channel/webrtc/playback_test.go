package channel_webrtc

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/pkg/channel"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type playbackTrack struct {
	packets [][]byte
	err     error
	onWrite func()
}

func (p *playbackTrack) CodecParameters() []pionwebrtc.RTPCodecParameters {
	return []pionwebrtc.RTPCodecParameters{{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		PayloadType:        111,
	}}
}

func (p *playbackTrack) HeaderExtensions() []pionwebrtc.RTPHeaderExtensionParameter { return nil }
func (p *playbackTrack) SSRC() pionwebrtc.SSRC                                      { return 1 }
func (p *playbackTrack) SSRCRetransmission() pionwebrtc.SSRC                        { return 0 }
func (p *playbackTrack) SSRCForwardErrorCorrection() pionwebrtc.SSRC                { return 0 }
func (p *playbackTrack) WriteStream() pionwebrtc.TrackLocalWriter                   { return p }
func (p *playbackTrack) ID() string                                                 { return "playback-test" }
func (p *playbackTrack) RTCPReader() interceptor.RTCPReader                         { return nil }
func (p *playbackTrack) Write(payload []byte) (int, error)                          { return len(payload), p.err }
func (p *playbackTrack) WriteRTP(_ *rtp.Header, payload []byte) (int, error) {
	if p.onWrite != nil {
		p.onWrite()
	}
	if p.err != nil {
		return 0, p.err
	}
	p.packets = append(p.packets, bytes.Clone(payload))
	return len(payload), nil
}

func TestPlaybackInvalidControlDoesNotSendOrChangeState(t *testing.T) {
	for _, kind := range []protos.ConversationPlaybackControl_Kind{
		protos.ConversationPlaybackControl_KIND_UNSPECIFIED, protos.ConversationPlaybackControl_Kind(99),
	} {
		t.Run(kind.String(), func(t *testing.T) {
			s := newTestStreamer(t)
			t.Cleanup(s.Cancel)
			s.sessionState.StartMediaSession()
			s.sessionState.SetPeerConnected(true)
			invalid := &protos.ConversationPlaybackControl{Id: "response", Kind: kind}
			require.Error(t, s.Send(invalid))
			assert.False(t, s.outputPaused)
			assert.False(t, s.outputFlushed)
			assert.False(t, s.outputClearPending)
			assert.Zero(t, s.CriticalCh.Len())
			assert.Zero(t, s.LowCh.Len())

			frame := bytes.Repeat([]byte{0x31}, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
			require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
				Id: "response", Message: &protos.ConversationAssistantMessage_Audio{Audio: frame},
			}))
			staged := s.NextFrame()
			require.Equal(t, frame, staged)
			require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_PAUSE}))
			criticalBefore, lowBefore := s.CriticalCh.Len(), s.LowCh.Len()
			require.Error(t, s.Send(invalid))
			assert.True(t, s.outputPaused)
			assert.False(t, s.outputFlushed)
			assert.False(t, s.outputClearPending)
			assert.Equal(t, criticalBefore, s.CriticalCh.Len())
			assert.Equal(t, lowBefore, s.LowCh.Len())
			assert.Equal(t, frame, s.currentOutputFrame)
			assert.Empty(t, s.NextFrame())

			require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_CONTINUE}))
			assert.Equal(t, frame, s.NextFrame())
			require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Id: "response", Kind: protos.ConversationPlaybackControl_FLUSH}))
			assert.Empty(t, s.currentOutputFrame)
			assert.True(t, s.outputClearPending)
		})
	}
}

func TestPlaybackBufferedOutput(t *testing.T) {
	type expectedCompletion struct {
		id string
	}

	for _, scenario := range []string{
		"empty_audio_terminal", "payloadless_terminal", "audio_terminal", "text_is_not_terminal", "gap", "pause_continue",
		"flush", "flush_then_new", "peer_generation", "disconnect", "write_error", "nil_transport", "conversion_error", "cancel",
		"consecutive_responses", "overflow", "terminal_without_audio", "uncorrelated_terminal",
		"same_id_queued", "same_id_drained", "same_id_disconnect", "unavailable_terminal",
		"idless_after_terminal", "idless_after_drain",
		"stale_terminal",
	} {
		t.Run(scenario, func(t *testing.T) {
			// Keep native fixtures serial; concurrent cold initialization can crash libsoxr.
			s := newTestStreamer(t)
			t.Cleanup(s.Cancel)
			mediaSessionID := s.sessionState.StartMediaSession()
			s.sessionState.SetPeerConnected(true)
			track := &playbackTrack{}
			var err error
			s.assistantAudioTrack, err = pionwebrtc.NewTrackLocalStaticSample(track.CodecParameters()[0].RTPCodecCapability, "audio", "test")
			require.NoError(t, err)
			_, err = s.assistantAudioTrack.Bind(track)
			require.NoError(t, err)
			s.assistantWriter, err = resampler_soxr.NewWriter(
				internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG, internal_audio.WEBRTC_AUDIO_CONFIG,
				func(pcm []byte) error {
					s.assistantPCM48k = append(s.assistantPCM48k, pcm...)
					return nil
				}, resampler_soxr.WithHighQuality(),
			)
			require.NoError(t, err)
			t.Cleanup(s.assistantWriter.Close)
			frameBytes := webrtc_internal.WebRTCOutputPCM16kFrameBytes
			audio := bytes.Repeat([]byte{0x12, 0x34}, (3*frameBytes+4)/2)
			wantComplete := scenario == "empty_audio_terminal" || scenario == "audio_terminal" || scenario == "pause_continue" || scenario == "consecutive_responses" || scenario == "flush_then_new" || scenario == "same_id_queued" || scenario == "same_id_drained" || scenario == "idless_after_terminal" || scenario == "idless_after_drain" || scenario == "stale_terminal"
			if scenario != "terminal_without_audio" {
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-a", Message: &protos.ConversationAssistantMessage_Audio{Audio: audio},
					Completed: scenario == "audio_terminal",
				}))
			}
			if scenario == "conversion_error" {
				s.assistantWriter.Close()
			}
			if scenario == "write_error" {
				track.err = errors.New("local send failed")
			}
			if scenario == "nil_transport" {
				s.assistantAudioTrack = nil
			}
			if scenario != "audio_terminal" {
				for range 16 {
					frame := s.NextFrame()
					if len(frame) == 0 {
						break
					}
					err := s.ConsumeFrame(frame)
					if scenario != "write_error" && scenario != "nil_transport" && scenario != "conversion_error" {
						require.NoError(t, err)
					}
				}
				require.Zero(t, s.CriticalCh.Len(), "a queue gap cannot complete playback")
			}
			switch scenario {
			case "uncorrelated_terminal":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{Completed: true}))
			case "payloadless_terminal":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{Id: "response-a", Completed: true}))
			case "text_is_not_terminal":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-a", Message: &protos.ConversationAssistantMessage_Text{Text: "done"}, Completed: true,
				}))
			case "gap":
				assert.Nil(t, s.IdleFrame())
			case "audio_terminal":
			case "stale_terminal":
			default:
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-a", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}))
			}
			packetsBeforeScenarioAction := len(track.packets)
			switch scenario {
			case "same_id_queued", "same_id_drained", "same_id_disconnect", "idless_after_terminal", "idless_after_drain":
				if scenario == "same_id_drained" || scenario == "idless_after_drain" {
					for range 16 {
						frame := s.NextFrame()
						if len(frame) == 0 {
							break
						}
						require.NoError(t, s.ConsumeFrame(frame))
					}
					require.Equal(t, 1, s.CriticalCh.Len())
					require.Len(t, track.packets, 4)
				}
				id := "response-a"
				if scenario == "idless_after_terminal" || scenario == "idless_after_drain" {
					id = ""
				}
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: id, Message: &protos.ConversationAssistantMessage_Audio{Audio: audio}, Completed: true,
				}))
				if scenario == "same_id_disconnect" {
					s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateDisconnected, time.Now())
					assert.Nil(t, s.NextFrame())
					assert.Zero(t, s.CriticalCh.Len())
					s.sessionState.SetPeerConnected(true)
				}
			case "unavailable_terminal":
				frame := s.NextFrame()
				require.NotEmpty(t, frame)
				s.sessionState.SetPeerConnected(false)
				require.NoError(t, s.ConsumeFrame(frame))
				assert.Zero(t, s.CriticalCh.Len())
				s.sessionState.SetPeerConnected(true)
			case "pause_continue":
				frame := s.NextFrame()
				require.NotEmpty(t, frame)
				require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE}))
				before := len(track.packets)
				require.NoError(t, s.ConsumeFrame(frame))
				assert.Len(t, track.packets, before)
				assert.Zero(t, s.CriticalCh.Len())
				require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_CONTINUE}))
				assert.Equal(t, frame, s.NextFrame())
			case "flush", "flush_then_new":
				require.NotEmpty(t, s.NextFrame())
				require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH}))
				assert.Empty(t, s.assistantPCM48k)
				if scenario == "flush_then_new" {
					s.grpcStream = &failingGRPCStream{}
					done := make(chan struct{})
					go func() {
						s.runOutputWriter()
						close(done)
					}()
					t.Cleanup(func() { s.Cancel(); <-done })
					require.Eventually(t, func() bool {
						s.outputStateMu.Lock()
						defer s.outputStateMu.Unlock()
						return !s.outputClearPending
					}, time.Second, time.Millisecond)
					track.packets = nil
					require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
						Id: "response-b", Message: &protos.ConversationAssistantMessage_Audio{Audio: audio}, Completed: true,
					}))
					require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
						Id: "response-a", Message: &protos.ConversationAssistantMessage_Audio{Audio: audio}, Completed: true,
					}))
				}
			case "peer_generation":
				s.sessionState.StartMediaSession()
				s.sessionState.SetPeerConnected(true)
			case "disconnect":
				s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateDisconnected, time.Now())
				s.sessionState.SetPeerConnected(true)
			case "cancel":
				s.Cancel()
			case "consecutive_responses":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-b", Message: &protos.ConversationAssistantMessage_Audio{Audio: audio}, Completed: true,
				}))
			case "stale_terminal":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-a", Message: &protos.ConversationAssistantMessage_Audio{Audio: audio}, Completed: true,
				}))
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-a", Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}))
			case "overflow":
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: "response-b", Message: &protos.ConversationAssistantMessage_Audio{
						Audio: make([]byte, frameBytes*(webrtc_internal.OutputAudioQueueMaxFrames+1)),
					}, Completed: true,
				}))
			}
			for range webrtc_internal.OutputAudioQueueMaxFrames + 32 {
				frame := s.NextFrame()
				if len(frame) == 0 {
					break
				}
				before := len(track.packets)
				err := s.ConsumeFrame(frame)
				if scenario != "write_error" && scenario != "nil_transport" && scenario != "conversion_error" {
					require.NoError(t, err)
				}
				assert.LessOrEqual(t, len(track.packets)-before, 1, "one RTP frame per paced tick")
			}
			if wantComplete {
				wantCompletions := []expectedCompletion{{id: "response-a"}}
				if scenario == "flush_then_new" {
					wantCompletions = []expectedCompletion{{id: "response-b"}}
				}
				if scenario == "consecutive_responses" {
					wantCompletions = append(wantCompletions, expectedCompletion{id: "response-b"})
				}
				for _, wantCompletion := range wantCompletions {
					select {
					case <-s.CriticalCh.Ready():
						msg, err := s.CriticalCh.TryReceive()
						require.NoError(t, err)
						complete, ok := msg.(*protos.ConversationPlaybackComplete)
						require.True(t, ok, "unexpected observation %T", msg)
						assert.Equal(t, wantCompletion.id, complete.GetId())
						require.NotNil(t, complete.GetTime())
						require.NoError(t, complete.GetTime().CheckValid())
					default:
						t.Fatal("missing local output completion")
					}
				}
				wantPackets := 4 * len(wantCompletions)
				if scenario == "idless_after_terminal" || scenario == "idless_after_drain" {
					wantPackets += 4
				}
				if scenario == "stale_terminal" {
					assert.GreaterOrEqual(t, len(track.packets), packetsBeforeScenarioAction, "partial input and converter tails must be sent")
				} else {
					assert.Len(t, track.packets, wantPackets, "partial input and converter tails must be sent")
				}
				lastCompletion := wantCompletions[len(wantCompletions)-1]
				require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
					Id: lastCompletion.id, Completed: true, Message: &protos.ConversationAssistantMessage_Audio{},
				}))
				assert.Nil(t, s.NextFrame(), "duplicate terminal must not queue another completion")
				assert.Empty(t, s.assistantPCM48k)
			}
			assert.Zero(t, s.CriticalCh.Len())
			decoder, err := webrtc_internal.NewOpusDecoder()
			require.NoError(t, err)
			for _, packet := range track.packets {
				pcm, err := decoder.Decode(packet)
				require.NoError(t, err)
				assert.Len(t, pcm, frameBytes*3)
			}
		})
	}
}

func TestPlaybackCompletionWaitsForInflightWriteAndReleasesLocks(t *testing.T) {
	s := newTestStreamer(t)
	t.Cleanup(s.Cancel)
	s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	track := &playbackTrack{}
	var err error
	s.assistantAudioTrack, err = pionwebrtc.NewTrackLocalStaticSample(track.CodecParameters()[0].RTPCodecCapability, "audio", "test")
	require.NoError(t, err)
	_, err = s.assistantAudioTrack.Bind(track)
	require.NoError(t, err)
	s.assistantWriter, err = resampler_soxr.NewWriter(
		internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG, internal_audio.WEBRTC_AUDIO_CONFIG,
		func(pcm []byte) error {
			s.assistantPCM48k = append(s.assistantPCM48k, pcm...)
			return nil
		}, resampler_soxr.WithHighQuality(),
	)
	require.NoError(t, err)
	t.Cleanup(s.assistantWriter.Close)
	frameBytes := webrtc_internal.WebRTCOutputPCM16kFrameBytes
	require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
		Id: "inflight", Message: &protos.ConversationAssistantMessage_Audio{Audio: bytes.Repeat([]byte{0x23}, 3*frameBytes+4)}, Completed: true,
	}))
	writing := make(chan struct{})
	release := make(chan struct{})
	track.onWrite = func() {
		if len(track.packets) == 3 {
			close(writing)
			<-release
		}
	}
	done := make(chan error, 1)
	go func() {
		for range 32 {
			frame := s.NextFrame()
			if len(frame) == 0 {
				break
			}
			if err := s.ConsumeFrame(frame); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case <-writing:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("last RTP write did not start")
	}
	assert.Zero(t, s.CriticalCh.Len(), "completion must wait for the local write to return")
	for range s.CriticalCh.Capacity() {
		result, err := s.CriticalCh.Send(t.Context(), &protos.ConversationMetadata{})
		require.NoError(t, err)
		require.Equal(t, channel.Enqueued, result.Status)
	}
	close(release)
	require.Eventually(t, func() bool {
		if !s.outputWriteMu.TryLock() {
			return false
		}
		defer s.outputWriteMu.Unlock()
		if !s.outputStateMu.TryLock() {
			return false
		}
		defer s.outputStateMu.Unlock()
		return s.currentOutputFrame == nil && s.outputPlayback.Sent && len(s.assistantPCM48k) == 0
	}, time.Second, time.Millisecond)
	require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_PAUSE}))
	select {
	case <-done:
		t.Fatal("completion was dropped instead of waiting for input capacity")
	default:
	}
	s.Cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("completion delivery did not unblock on cancellation")
	}
}

func TestPlaybackClearSendFailureClosesWithoutDeadlock(t *testing.T) {
	s := newTestStreamer(t)
	t.Cleanup(s.Cancel)
	s.grpcStream = &failingGRPCStream{sendErr: errors.New("clear signaling failed")}
	require.NoError(t, s.Send(&protos.ConversationPlaybackControl{Kind: protos.ConversationPlaybackControl_FLUSH}))
	done := make(chan struct{})
	go func() {
		s.runOutputWriter()
		close(done)
	}()
	select {
	case <-done:
		assert.Error(t, s.Ctx.Err())
	case <-time.After(time.Second):
		t.Fatal("clear send failure deadlocked WebRTC teardown")
	}
}

func TestPlaybackShutdownCancelsBlockedWriteBeforeClearingOutput(t *testing.T) {
	for _, shutdown := range []string{"close", "text_fallback", "text_mode"} {
		t.Run(shutdown, func(t *testing.T) {
			s := newTestStreamer(t)
			t.Cleanup(s.Cancel)
			s.sessionState.StartMediaSession()
			s.sessionState.SetPeerConnected(true)
			s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
			mediaCtx, cancelMedia := context.WithCancel(s.Ctx)
			t.Cleanup(cancelMedia)
			s.mediaCtx, s.cancelMedia = mediaCtx, cancelMedia
			writing := make(chan struct{})
			track := &playbackTrack{err: context.Canceled, onWrite: func() {
				close(writing)
				<-mediaCtx.Done()
			}}
			var err error
			s.assistantAudioTrack, err = pionwebrtc.NewTrackLocalStaticSample(track.CodecParameters()[0].RTPCodecCapability, "audio", "test")
			require.NoError(t, err)
			_, err = s.assistantAudioTrack.Bind(track)
			require.NoError(t, err)
			s.assistantWriter, err = resampler_soxr.NewWriter(
				internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG, internal_audio.WEBRTC_AUDIO_CONFIG,
				func(pcm []byte) error {
					s.assistantPCM48k = append(s.assistantPCM48k, pcm...)
					return nil
				}, resampler_soxr.WithHighQuality(),
			)
			require.NoError(t, err)
			t.Cleanup(s.assistantWriter.Close)
			require.NoError(t, s.Send(&protos.ConversationAssistantMessage{
				Id: "blocked", Completed: true,
				Message: &protos.ConversationAssistantMessage_Audio{Audio: make([]byte, 3*webrtc_internal.WebRTCOutputPCM16kFrameBytes)},
			}))
			writeDone := make(chan error, 1)
			go func() {
				for frame := s.NextFrame(); len(frame) > 0; frame = s.NextFrame() {
					if err := s.ConsumeFrame(frame); err != nil {
						writeDone <- err
						return
					}
				}
				writeDone <- nil
			}()
			select {
			case <-writing:
			case <-time.After(time.Second):
				cancelMedia()
				<-writeDone
				t.Fatal("RTP write did not start")
			}
			stopped := make(chan struct{})
			go func() {
				defer close(stopped)
				if shutdown == "close" {
					_ = s.Close()
				} else if shutdown == "text_mode" {
					s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)
				} else {
					s.stopMediaSessionAndFallbackToText()
				}
			}()
			select {
			case <-stopped:
			case <-time.After(time.Second):
				cancelMedia()
				<-stopped
				t.Error("shutdown waited for the output lock before canceling media")
			}
			require.ErrorIs(t, <-writeDone, context.Canceled)
			assert.Zero(t, s.CriticalCh.Len())
			assert.Empty(t, s.assistantPCM48k)
			assert.Nil(t, s.currentOutputFrame)
		})
	}
}
