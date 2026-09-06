// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"context"
	"testing"
	"time"

	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMediaPortTestSession(t *testing.T) *sip_runtime.Session {
	t.Helper()
	session, err := sip_runtime.NewSession(context.Background(),
		sip_runtime.WithSessionConfig(&sip_runtime.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10010,
		}),
		sip_runtime.WithSessionDirection(sip_runtime.CallDirectionInbound),
		sip_runtime.WithSessionCallID("media-port-test"),
		sip_runtime.WithSessionCodec(&sip_runtime.CodecPCMU),
	)
	require.NoError(t, err)
	return session
}

func newMediaPortTestRTP(t *testing.T) (*fakeRTPHandler, chan sip_runtime.InboundAudioFrame, chan []byte) {
	t.Helper()
	rtpHandler := newTestRTPHandler(&sip_runtime.CodecPCMU)
	return rtpHandler, rtpHandler.audioIn, rtpHandler.audioOut
}

func newMediaPortForTest(
	t *testing.T,
	streamSink func(internal_type.Stream),
	recorders ...func(...observability.Record) error,
) (*MediaPort, chan sip_runtime.InboundAudioFrame, chan []byte) {
	t.Helper()
	rtpHandler, audioIn, audioOut := newMediaPortTestRTP(t)
	var record func(...observability.Record) error
	if len(recorders) > 0 {
		record = recorders[0]
	}
	mediaPort, err := NewMediaPort(MediaPortConfig{
		Context:    context.Background(),
		Session:    newMediaPortTestSession(t),
		RTPHandler: rtpHandler,
		StreamSink: streamSink,
		Record:     record,
	})
	require.NoError(t, err)
	return mediaPort, audioIn, audioOut
}

func TestMediaPort_StartForwardsProviderAudio(t *testing.T) {
	streams := make(chan internal_type.Stream, 4)
	mediaPort, audioIn, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()

	for i := 0; i < 2; i++ {
		audioIn <- sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()}
	}

	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				if userMessage, ok := stream.(*protos.ConversationUserMessage); ok {
					return len(userMessage.GetAudio()) == BridgeOutputFrameSize
				}
			default:
				return false
			}
		}
	}, time.Second, 10*time.Millisecond)
}

func TestMediaPort_SeparatesRecognitionAndRecordingDelivery(tester *testing.T) {
	rtpHandler, audioIn, _ := newMediaPortTestRTP(tester)
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

	audioIn <- sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()}

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
	mediaPort, audioIn, _ := newMediaPortForTest(tester, func(stream internal_type.Stream) {
		streams <- stream
	})
	mediaPort.StartInput()
	tester.Cleanup(func() { require.NoError(tester, mediaPort.Close()) })

	receivedAt := time.Unix(123, 456)
	audioIn <- sip_runtime.InboundAudioFrame{
		Audio:      make([]byte, MulawFrameSize),
		ReceivedAt: receivedAt,
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
	mediaPort, audioIn, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()

	for i := 0; i < 2; i++ {
		audioIn <- sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()}
	}

	var bridgeUserAudioCount int
	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				switch message := stream.(type) {
				case *protos.ConversationBridgeUserAudio:
					bridgeUserAudioCount++
					if len(message.GetAudio()) != BridgeOutputFrameSize {
						t.Fatalf("unexpected bridge user audio length: %d", len(message.GetAudio()))
					}
				case *protos.ConversationUserMessage:
					return bridgeUserAudioCount == 1 && len(message.GetAudio()) == BridgeOutputFrameSize
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
	require.NoError(t, mediaPort.HandleAssistantAudio(make([]byte, BridgeOutputFrameSize), false))

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
	mediaPort, audioIn, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	audioIn <- sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()}
	mediaPort.HandleInterrupt()
	audioIn <- sip_runtime.InboundAudioFrame{Audio: make([]byte, MulawFrameSize), ReceivedAt: time.Now()}

	receivedAudioCount := 0
	require.Eventually(t, func() bool {
		for {
			select {
			case stream := <-streams:
				if userMessage, ok := stream.(*protos.ConversationUserMessage); ok {
					require.Len(t, userMessage.GetAudio(), BridgeOutputFrameSize)
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
	mediaPort, audioIn, _ := newMediaPortForTest(t, func(stream internal_type.Stream) {
		streams <- stream
	})
	bridgeRTP, _, bridgeAudioOut := newMediaPortTestRTP(t)

	mediaPort.Start()
	defer func() { require.NoError(t, mediaPort.Close()) }()
	mediaPort.ConnectTransferMedia(bridgeRTP, sip_runtime.CodecPCMU.Name)
	audioIn <- sip_runtime.InboundAudioFrame{Audio: []byte{0x01, 0x02, 0x03}, ReceivedAt: time.Now()}

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
