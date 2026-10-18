package internal_transformer_google

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/speech/apiv2/speechpb"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- mock stream ---

type mockStream struct {
	grpc.ClientStream
	recvFunc      func() (*speechpb.StreamingRecognizeResponse, error)
	sendFunc      func(*speechpb.StreamingRecognizeRequest) error
	closeSendFunc func() error
}

func (m *mockStream) Recv() (*speechpb.StreamingRecognizeResponse, error) { return m.recvFunc() }
func (m *mockStream) Send(req *speechpb.StreamingRecognizeRequest) error {
	if m.sendFunc != nil {
		return m.sendFunc(req)
	}
	return nil
}
func (m *mockStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockStream) Trailer() metadata.MD         { return nil }
func (m *mockStream) CloseSend() error {
	if m.closeSendFunc != nil {
		return m.closeSendFunc()
	}
	return nil
}
func (m *mockStream) Context() context.Context  { return context.Background() }
func (m *mockStream) SendMsg(interface{}) error { return nil }
func (m *mockStream) RecvMsg(interface{}) error { return nil }

// --- helpers ---

func newTestSTT(ctx context.Context, factory func(context.Context) (speechpb.Speech_StreamingRecognizeClient, error), onPacket func(...internal_type.Packet) error) *googleSpeechToText {
	logger := newTestLogger()
	cred := newVaultCredential(map[string]interface{}{"key": "k", "project_id": "p"})
	opt, _ := NewGoogleOption(logger, cred, utils.Option{})

	xctx, cancel := context.WithCancel(ctx)
	return &googleSpeechToText{
		ctx:           xctx,
		ctxCancel:     cancel,
		logger:        logger,
		googleOption:  opt,
		onPacket:      onPacket,
		streamFactory: factory,
	}
}

type googleSTTRecordingLogger struct {
	commons.Logger
	errorMessage string
}

func (logger *googleSTTRecordingLogger) Errorf(format string, args ...interface{}) {
	logger.errorMessage = fmt.Sprintf(format, args...)
}

func TestClose_LogsStreamErrorsContainingFormatDirectives(t *testing.T) {
	logger := &googleSTTRecordingLogger{}
	_, cancel := context.WithCancel(context.Background())
	transformer := &googleSpeechToText{
		ctxCancel: cancel,
		logger:    logger,
		stream: &mockStream{
			closeSendFunc: func() error {
				return errors.New("stream is 100% unavailable")
			},
		},
		onPacket: func(...internal_type.Packet) error { return nil },
	}

	err := transformer.Close(context.Background())

	require.EqualError(t, err, "error closing StreamClient: stream is 100% unavailable")
	assert.Equal(t, "error closing StreamClient: stream is 100% unavailable", logger.errorMessage)
}

// TestRecvLoop_ReinitOnTimeout proves that when the Google stream returns
// an Aborted error (the "Stream timed out" error), recvLoop acquires the lock,
// reinitializes the stream via streamFactory, and does NOT emit an error event.
func TestRecvLoop_ReinitOnTimeout(t *testing.T) {
	var factoryCalls atomic.Int32
	recvOnce := sync.Once{}
	allowFirstRecv := make(chan struct{})

	// The first stream returns a timeout error on Recv.
	firstStream := &mockStream{
		recvFunc: func() (*speechpb.StreamingRecognizeResponse, error) {
			<-allowFirstRecv
			var err error
			recvOnce.Do(func() {
				err = status.Error(codes.Aborted, "Stream timed out after receiving no more client requests.")
			})
			if err != nil {
				return nil, err
			}
			// Block forever after the first error (should not be called — recvLoop exits)
			select {}
		},
	}

	// The second stream (after reinit) blocks on Recv forever (healthy idle).
	secondStream := &mockStream{
		recvFunc: func() (*speechpb.StreamingRecognizeResponse, error) {
			select {} // block — healthy stream waiting for audio
		},
	}

	factory := func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error) {
		n := factoryCalls.Add(1)
		if n == 1 {
			return firstStream, nil
		}
		return secondStream, nil
	}

	var errorEvents atomic.Int32
	onPacket := func(pkts ...internal_type.Packet) error {
		for _, p := range pkts {
			if evt, ok := p.(internal_type.ObservabilityEventRecordPacket); ok {
				if evt.Record.Attributes["type"] == "error" {
					errorEvents.Add(1)
				}
			}
		}
		return nil
	}

	g := newTestSTT(context.Background(), factory, onPacket)

	// Initialize — opens the first stream
	err := g.Initialize()
	require.NoError(t, err)
	assert.EqualValues(t, 1, factoryCalls.Load(), "Initialize should call factory once")
	close(allowFirstRecv)

	// Wait for recvLoop to hit the timeout error and reinitialize
	assert.Eventually(t, func() bool {
		return factoryCalls.Load() >= 2
	}, 2*time.Second, 10*time.Millisecond, "recvLoop should reinitialize the stream via factory")

	// The stream should be non-nil (the new stream)
	g.mu.Lock()
	strm := g.stream
	g.mu.Unlock()
	assert.NotNil(t, strm, "stream should be restored after reinit")
	assert.Same(t, secondStream, strm, "stream should be the second (reinit) stream")

	// No error event should have been emitted — reinit succeeded
	assert.EqualValues(t, 0, errorEvents.Load(), "no error event should be emitted when reinit succeeds")

	g.ctxCancel()
}

// TestTransform_WorksAfterReinit proves that Transform() can send audio
// after recvLoop has transparently reinitialized the stream.
func TestTransform_WorksAfterReinit(t *testing.T) {
	var factoryCalls atomic.Int32
	recvOnce := sync.Once{}

	var secondSendCalled atomic.Int32

	firstStream := &mockStream{
		recvFunc: func() (*speechpb.StreamingRecognizeResponse, error) {
			var err error
			recvOnce.Do(func() {
				err = status.Error(codes.Aborted, "Stream timed out after receiving no more client requests.")
			})
			if err != nil {
				return nil, err
			}
			select {}
		},
	}

	secondStream := &mockStream{
		recvFunc: func() (*speechpb.StreamingRecognizeResponse, error) {
			select {} // healthy idle
		},
		sendFunc: func(req *speechpb.StreamingRecognizeRequest) error {
			if req.GetAudio() != nil {
				secondSendCalled.Add(1)
			}
			return nil
		},
	}

	factory := func(ctx context.Context) (speechpb.Speech_StreamingRecognizeClient, error) {
		n := factoryCalls.Add(1)
		if n == 1 {
			return firstStream, nil
		}
		return secondStream, nil
	}

	onPacket := func(pkts ...internal_type.Packet) error { return nil }

	g := newTestSTT(context.Background(), factory, onPacket)

	err := g.Initialize()
	require.NoError(t, err)

	// Wait for reinit to complete
	require.Eventually(t, func() bool {
		return factoryCalls.Load() >= 2
	}, 2*time.Second, 10*time.Millisecond)

	// Now Transform should send audio on the new stream
	err = g.Transform(context.Background(), internal_type.SpeechToTextAudioPacket{Audio: []byte{0x01, 0x02}})
	require.NoError(t, err)

	assert.EqualValues(t, 1, secondSendCalled.Load(), "audio should be sent on the reinitialized stream")

	g.ctxCancel()
}
