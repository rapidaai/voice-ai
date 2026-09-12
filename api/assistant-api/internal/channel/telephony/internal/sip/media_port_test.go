// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"net"
	"testing"
	"time"

	"github.com/pion/rtp"
	resampler_soxr "github.com/rapidaai/api/assistant-api/internal/audio/resampler/soxr"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaf/g711"
)

func newMediaPortTestSession(t *testing.T) *sip_runtime.Session {
	return newMediaPortTestSessionWithCodec(t, &sip_runtime.CodecPCMU)
}

func newMediaPortTestSessionWithCodec(t *testing.T, codec *sip_runtime.Codec) *sip_runtime.Session {
	t.Helper()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_config.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10010,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionInbound),
		sip_runtime.WithSessionCallID("media-port-test"),
		sip_runtime.WithSessionCodec(codec),
	)
	require.NoError(t, err)
	return session
}

func newMediaPortTestRTPHandler(t *testing.T, codec sip_runtime.Codec) *sip_runtime.RTPHandler {
	t.Helper()
	reserved, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	port := reserved.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, reserved.Close())

	handler, err := sip_runtime.NewRTPHandler(t.Context(), &sip_runtime.RTPConfig{
		LocalAddress: sip_runtime.RTPAddress{
			IP:   "127.0.0.1",
			Port: port,
		},
		PayloadType:       codec.PayloadType,
		ClockRate:         codec.ClockRate,
		PacketizationTime: ChunkDuration,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, handler.Stop())
	})
	return handler
}

func newMediaPortTestRTP(t *testing.T) (*fakeRTPHandler, func(sip_runtime.InboundAudioFrame), chan []byte) {
	t.Helper()
	rtpHandler := newTestRTPHandler(&sip_runtime.CodecPCMU)
	return rtpHandler, rtpHandler.emitInboundAudio, rtpHandler.audioOut
}

func newMediaPortForTest(
	t *testing.T,
	streamSink func(internal_type.Stream),
	recorders ...func(...observability.Record) error,
) (*MediaPort, func(sip_runtime.InboundAudioFrame), chan []byte) {
	t.Helper()
	rtpHandler, emitInboundAudio, audioOut := newMediaPortTestRTP(t)
	var record func(...observability.Record) error
	if len(recorders) > 0 {
		record = recorders[0]
	}
	mediaPort, err := NewMediaPort(MediaPortConfig{
		Context:    context.Background(),
		Session:    newMediaPortTestSession(t),
		RTPHandler: rtpHandler,
		StreamSink: streamSink,
		RecordSink: record,
	})
	require.NoError(t, err)
	return mediaPort, emitInboundAudio, audioOut
}

func TestMediaPort_StartForwardsProviderAudio(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, emitInboundAudio, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()

	for i := 0; i < 8; i++ {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}

	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				if userMessage, ok := stream.(*protos.ConversationUserMessage); ok {
					return len(userMessage.GetAudio()) > 0 &&
						len(userMessage.GetAudio())%2 == 0
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_RealUDPInputMatchesReferencePCM(t *testing.T) {
	const frameCount = 10

	for _, codec := range []sip_runtime.Codec{sip_runtime.CodecPCMU, sip_runtime.CodecPCMA} {
		t.Run(codec.Name, func(t *testing.T) {
			receiver := newMediaPortTestRTPHandler(t, codec)
			pipelineAudio := make(chan []byte, frameCount)
			recordingAudio := make(chan []byte, frameCount)
			mediaPort, err := NewMediaPort(MediaPortConfig{
				Context:    t.Context(),
				Session:    newMediaPortTestSessionWithCodec(t, &codec),
				RTPHandler: receiver,
				StreamSink: func(stream internal_type.Stream) {
					switch message := stream.(type) {
					case *protos.ConversationUserMessage:
						pipelineAudio <- append([]byte(nil), message.GetAudio()...)
					case *protos.ConversationBridgeUserAudio:
						recordingAudio <- append([]byte(nil), message.GetAudio()...)
					}
				},
			})
			require.NoError(t, err)
			mediaPort.StartInput()
			receiver.Start()
			t.Cleanup(func() {
				require.NoError(t, mediaPort.Close())
			})

			sender := newMediaPortTestRTPHandler(t, codec)
			sender.Start()
			sender.SetRemoteAddress(receiver.LocalAddress())

			pcm := make([]byte, frameCount*MulawFrameSize*2)
			for sampleIndex := 0; sampleIndex < frameCount*MulawFrameSize; sampleIndex++ {
				sample := int16(12000 * math.Sin(2*math.Pi*440*float64(sampleIndex)/8000))
				binary.LittleEndian.PutUint16(pcm[sampleIndex*2:], uint16(sample))
			}

			reference := resampler_soxr.New(resampler_soxr.WithHighQuality())
			var expectedFrames [][]byte
			for frameIndex := range frameCount {
				start := frameIndex * MulawFrameSize * 2
				end := start + MulawFrameSize*2
				encoded := g711.EncodeUlaw(pcm[start:end])
				if codec.Name == sip_runtime.CodecPCMA.Name {
					encoded = g711.EncodeAlaw(pcm[start:end])
				}
				expectedFrame, err := reference.Resample(
					decodeG711ToLinear8k(encoded, codec.Name),
					Linear8kConfig,
					Rapida16kConfig,
				)
				require.NoError(t, err)
				if len(expectedFrame) > 0 {
					expectedFrames = append(expectedFrames, expectedFrame)
				}
				require.NoError(t, sender.WriteAudio(encoded))
			}

			var actualPipeline []byte
			var actualRecording []byte
			for frameIndex, expected := range expectedFrames {
				select {
				case audio := <-pipelineAudio:
					require.Equalf(t, expected, audio, "pipeline frame %d", frameIndex)
					actualPipeline = append(actualPipeline, audio...)
				case <-time.After(time.Second):
					t.Fatalf("timed out waiting for pipeline frame %d", frameIndex)
				}
				select {
				case audio := <-recordingAudio:
					require.Equalf(t, expected, audio, "recording frame %d", frameIndex)
					actualRecording = append(actualRecording, audio...)
				case <-time.After(time.Second):
					t.Fatalf("timed out waiting for recording frame %d", frameIndex)
				}
			}

			expectedPCM := bytes.Join(expectedFrames, nil)
			require.Equal(t, expectedPCM, actualPipeline)
			require.Equal(t, expectedPCM, actualRecording)
			require.Eventually(t, func() bool {
				return receiver.GetDetailedStats().PacketsDelivered == uint64(frameCount)
			}, time.Second, time.Millisecond)
			stats := receiver.GetDetailedStats()
			assert.Equal(t, uint64(frameCount), stats.PacketsReceived)
			assert.Equal(t, uint64(frameCount), stats.PacketsDelivered)
			assert.Zero(t, stats.PacketsLost)
			assert.Zero(t, stats.PacketsDropped)
		})
	}
}

func TestMediaPort_RealUDPInputHandlesReorderingAndLoss(t *testing.T) {
	const samplesPerFrame = MulawFrameSize
	testCases := []struct {
		name            string
		sendOrder       []int
		expectLoss      bool
		packetsReceived uint64
	}{
		{name: "reordered", sendOrder: []int{0, 2, 1, 3, 4, 5}, packetsReceived: 6},
		{name: "missing packet", sendOrder: []int{0, 2, 3, 4, 5}, expectLoss: true, packetsReceived: 5},
	}

	for _, codec := range []sip_runtime.Codec{sip_runtime.CodecPCMU, sip_runtime.CodecPCMA} {
		for _, testCase := range testCases {
			t.Run(codec.Name+"/"+testCase.name, func(t *testing.T) {
				receiver := newMediaPortTestRTPHandler(t, codec)
				pipelineAudio := make(chan []byte, 6)
				recordingAudio := make(chan []byte, 6)
				mediaPort, err := NewMediaPort(MediaPortConfig{
					Context:    t.Context(),
					Session:    newMediaPortTestSessionWithCodec(t, &codec),
					RTPHandler: receiver,
					StreamSink: func(stream internal_type.Stream) {
						switch message := stream.(type) {
						case *protos.ConversationUserMessage:
							pipelineAudio <- append([]byte(nil), message.GetAudio()...)
						case *protos.ConversationBridgeUserAudio:
							recordingAudio <- append([]byte(nil), message.GetAudio()...)
						}
					},
				})
				require.NoError(t, err)
				mediaPort.StartInput()
				receiver.Start()
				t.Cleanup(func() {
					require.NoError(t, mediaPort.Close())
				})

				sender, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, sender.Close())
				})
				destination := &net.UDPAddr{
					IP:   net.ParseIP(receiver.LocalAddress().IP),
					Port: receiver.LocalAddress().Port,
				}

				encodedFrames := make([][]byte, 6)
				for frameIndex := range encodedFrames {
					pcm := make([]byte, samplesPerFrame*2)
					for sampleIndex := range samplesPerFrame {
						continuousIndex := frameIndex*samplesPerFrame + sampleIndex
						sample := int16(12000 * math.Sin(2*math.Pi*440*float64(continuousIndex)/8000))
						binary.LittleEndian.PutUint16(pcm[sampleIndex*2:], uint16(sample))
					}
					encodedFrames[frameIndex] = g711.EncodeUlaw(pcm)
					if codec.Name == sip_runtime.CodecPCMA.Name {
						encodedFrames[frameIndex] = g711.EncodeAlaw(pcm)
					}
				}

				expectedEncoded := append([][]byte(nil), encodedFrames...)
				if testCase.expectLoss {
					expectedEncoded = [][]byte{encodedFrames[0], encodedFrames[2], encodedFrames[3], encodedFrames[4], encodedFrames[5]}
				}

				reference := resampler_soxr.New(resampler_soxr.WithHighQuality())
				var expectedPCM [][]byte
				for _, encoded := range expectedEncoded {
					expectedFrame, err := reference.Resample(
						decodeG711ToLinear8k(encoded, codec.Name),
						Linear8kConfig,
						Rapida16kConfig,
					)
					require.NoError(t, err)
					if len(expectedFrame) > 0 {
						expectedPCM = append(expectedPCM, expectedFrame)
					}
				}

				for _, frameIndex := range testCase.sendOrder {
					packet := &rtp.Packet{
						Header: rtp.Header{
							Version:        2,
							PayloadType:    codec.PayloadType,
							SequenceNumber: uint16(100 + frameIndex),
							Timestamp:      uint32(frameIndex * samplesPerFrame),
							SSRC:           42,
						},
						Payload: encodedFrames[frameIndex],
					}
					packetData, marshalErr := packet.Marshal()
					require.NoError(t, marshalErr)
					_, writeErr := sender.WriteToUDP(packetData, destination)
					require.NoError(t, writeErr)
				}

				for frameIndex, expected := range expectedPCM {
					select {
					case audio := <-pipelineAudio:
						require.Equalf(t, expected, audio, "pipeline frame %d", frameIndex)
					case <-time.After(time.Second):
						t.Fatalf("timed out waiting for pipeline frame %d", frameIndex)
					}
					select {
					case audio := <-recordingAudio:
						require.Equalf(t, expected, audio, "recording frame %d", frameIndex)
					case <-time.After(time.Second):
						t.Fatalf("timed out waiting for recording frame %d", frameIndex)
					}
				}

				require.Eventually(t, func() bool {
					return receiver.GetDetailedStats().PacketsDelivered == uint64(len(expectedEncoded))
				}, time.Second, time.Millisecond)
				stats := receiver.GetDetailedStats()
				assert.Equal(t, testCase.packetsReceived, stats.PacketsReceived)
				assert.Equal(t, uint64(len(expectedEncoded)), stats.PacketsDelivered)
				if testCase.expectLoss {
					assert.Equal(t, uint64(1), stats.PacketsLost)
				} else {
					assert.Zero(t, stats.PacketsLost)
				}
				assert.Zero(t, stats.PacketsDropped)
				assert.Zero(t, stats.InvalidPackets)
			})
		}
	}
}

func TestMediaPort_SeparatesRecognitionAndRecordingDelivery(tester *testing.T) {
	rtpHandler, emitInboundAudio, _ := newMediaPortTestRTP(tester)
	realtime := make(chan internal_type.Stream, 1)
	recording := make(chan internal_type.Stream, 1)
	mediaPort, err := NewMediaPort(MediaPortConfig{
		Context:    context.Background(),
		Session:    newMediaPortTestSession(tester),
		RTPHandler: rtpHandler,
		StreamSink: func(stream internal_type.Stream) {
			switch stream.(type) {
			case *protos.ConversationBridgeUserAudio:
				recording <- stream
			default:
				realtime <- stream
			}
		},
	})
	require.NoError(tester, err)
	mediaPort.StartInput()
	tester.Cleanup(func() { require.NoError(tester, mediaPort.Close()) })

	for range 4 {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}

	select {
	case stream := <-realtime:
		require.IsType(tester, &protos.ConversationUserMessage{}, stream)
	case <-time.After(time.Second):
		tester.Fatal("recognition audio was not delivered")
	}
	select {
	case stream := <-recording:
		require.IsType(tester, &protos.ConversationBridgeUserAudio{}, stream)
	case <-time.After(time.Second):
		tester.Fatal("recording audio was not delivered")
	}
}

func TestMediaPort_PreservesRTPFrameReceivedAt(tester *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, emitInboundAudio, _ := newMediaPortForTest(tester, func(stream internal_type.Stream) {
		streams <- stream
	})
	mediaPort.StartInput()
	tester.Cleanup(func() { require.NoError(tester, mediaPort.Close()) })

	receivedAt := time.Unix(123, 456)
	for range 4 {
		emitInboundAudio(sip_runtime.InboundAudioFrame{
			Audio:      make([]byte, MulawFrameSize),
			ReceivedAt: receivedAt,
		})
	}

	for range 2 {
		select {
		case stream := <-streams:
			switch message := stream.(type) {
			case *protos.ConversationUserMessage:
				require.Equal(tester, receivedAt.UnixNano(), message.GetTime().AsTime().UnixNano())
				return
			}
		case <-time.After(time.Second):
			tester.Fatal("provider audio was not delivered")
		}
	}
	tester.Fatal("user audio was not delivered")
}

func TestMediaPort_LocalAddrReturnsRTPAddress(t *testing.T) {
	mediaPort, _, _ := newMediaPortForTest(t, nil)
	mediaPort.rtpHandler.(*fakeRTPHandler).localAddress = sip_runtime.RTPAddress{
		IP:   "127.0.0.1",
		Port: 12000,
	}

	localIP, localPort := mediaPort.LocalAddr()

	assert.Equal(t, "127.0.0.1", localIP)
	assert.Equal(t, 12000, localPort)
}

func TestMediaPort_ProviderAudioRecordsBeforePipelineAudio(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, emitInboundAudio, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()

	for i := 0; i < 8; i++ {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}

	var bridgeUserAudioCount int
	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				switch message := stream.(type) {
				case *protos.ConversationBridgeUserAudio:
					bridgeUserAudioCount++
					if len(message.GetAudio()) == 0 ||
						len(message.GetAudio())%2 != 0 {
						return false
					}
				case *protos.ConversationUserMessage:
					return bridgeUserAudioCount == 1 &&
						len(message.GetAudio()) > 0 &&
						len(message.GetAudio())%2 == 0
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_AssistantAudioReachesRTPOutput(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, _, audioOut := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	assert.True(t, mediaPort.session.GetInboundSetupTimings().FirstAssistantAudioSentAt.IsZero())
	require.NoError(t, mediaPort.HandleAssistantAudio(make([]byte, BridgeOutputFrameSize), true))

	select {
	case frame := <-audioOut:
		assert.Len(t, frame, MulawFrameSize)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RTP output")
	}
	select {
	case stream := <-streams:
		operatorAudio, ok := stream.(*protos.ConversationBridgeOperatorAudio)
		require.True(t, ok, "expected ConversationBridgeOperatorAudio, got %T", stream)
		assert.Len(t, operatorAudio.GetAudio(), BridgeOutputFrameSize)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivered assistant recording")
	}
	require.Eventually(t, func() bool {
		return !mediaPort.session.GetInboundSetupTimings().FirstAssistantAudioSentAt.IsZero()
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_StartInputDoesNotStartAssistantOutput(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, _, audioOut := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.StartInput()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	require.NoError(t, mediaPort.HandleAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	select {
	case frame := <-audioOut:
		t.Fatalf("pre-answer assistant audio was sent to RTP: %v", frame)
	case stream := <-streams:
		t.Fatalf("pre-answer assistant audio was recorded as delivered: %T", stream)
	case <-time.After(50 * time.Millisecond):
	}

	mediaPort.StartOutput()
	select {
	case frame := <-audioOut:
		assert.Len(t, frame, MulawFrameSize)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RTP output after output start")
	}
}

func TestMediaPort_DroppedAssistantAudioIsNotRecorded(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	records := make(chan observability.Record, 4)
	mediaPort, _, audioOut := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	}, func(record ...observability.Record) error {
		for _, item := range record {
			records <- item
		}
		return nil
	})

	for i := 0; i < cap(audioOut); i++ {
		audioOut <- []byte{byte(i)}
	}
	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()

	require.NoError(t, mediaPort.HandleAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				if _, ok := stream.(*protos.ConversationBridgeOperatorAudio); ok {
					t.Fatalf("dropped RTP frame was recorded as delivered assistant audio")
				}
			case record := <-records:
				if _, ok := record.(observability.RecordLog); ok {
					return true
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
	assert.True(t, mediaPort.session.GetInboundSetupTimings().FirstAssistantAudioSentAt.IsZero())
}

func TestMediaPort_TransferModeSuppressesAssistantAudio(t *testing.T) {
	mediaPort, _, audioOut := newMediaPortForTest(t, nil)

	require.True(t, mediaPort.EnterTransferMode(DefaultRingtone))
	require.NoError(t, mediaPort.HandleAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

	select {
	case frame := <-audioOut:
		t.Fatalf("assistant audio was queued during transfer mode: %v", frame)
	default:
	}
	require.True(t, mediaPort.ResumeAssistant())
	require.NoError(t, mediaPort.Close())
}

func TestMediaPort_InterruptPreservesInputAudio(t *testing.T) {
	streams := make(chan internal_type.Stream, 8)
	mediaPort, emitInboundAudio, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	for range 4 {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}
	mediaPort.HandleInterrupt()
	for range 4 {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}

	receivedAudioCount := 0
	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				if userMessage, ok := stream.(*protos.ConversationUserMessage); ok {
					require.NotEmpty(t, userMessage.GetAudio())
					require.Zero(t, len(userMessage.GetAudio())%2)
					receivedAudioCount++
					if receivedAudioCount == 2 {
						return true
					}
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_ConnectTransferMediaForwardsCallerAudio(t *testing.T) {
	streams := make(chan internal_type.Stream, 1)
	mediaPort, emitInboundAudio, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})
	bridgeRTP, _, bridgeAudioOut := newMediaPortTestRTP(t)

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	mediaPort.ConnectTransferMedia(bridgeRTP, sip_runtime.CodecPCMU.Name)
	emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: []byte{0x01, 0x02, 0x03}, ReceivedAt: time.Now()})
	for range 8 {
		emitInboundAudio(sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()})
	}

	select {
	case frame := <-bridgeAudioOut:
		assert.Equal(t, []byte{0x01, 0x02, 0x03}, frame)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridged caller audio")
	}

	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				_, ok := stream.(*protos.ConversationBridgeUserAudio)
				if ok {
					return true
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_CloseIsIdempotent(t *testing.T) {
	mediaPort, _, _ := newMediaPortForTest(t, nil)

	mediaPort.Start()

	require.NoError(t, mediaPort.Close())
	require.NoError(t, mediaPort.Close())
	_, err := mediaPort.audioProcessor.resamplers.provider.Resample(
		make([]byte, MulawFrameSize*2),
		Linear8kConfig,
		Rapida16kConfig,
	)
	require.ErrorIs(t, err, resampler_soxr.ErrResamplerClosed)
}

func TestMediaPort_DeliverAssistantFrameAfterCloseReturnsSessionClosed(t *testing.T) {
	mediaPort, _, _ := newMediaPortForTest(t, nil)
	mediaPort.Start()
	require.NoError(t, mediaPort.Close())

	require.NotPanics(t, func() {
		err := mediaPort.deliverAssistantFrame(internal_telephony_media.AssistantOutputFrame{
			ProviderAudio: make([]byte, BridgeOutputFrameSize),
		})
		assert.ErrorIs(t, err, sip_runtime.ErrSessionClosed)
	})
}
