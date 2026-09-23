package internal_transformer_google

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	google_internal "github.com/rapidaai/api/assistant-api/internal/transformer/google/internal"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type googleTTSCall struct {
	stream     texttospeechpb.TextToSpeech_StreamingSynthesizeServer
	requests   chan *texttospeechpb.StreamingSynthesizeRequest
	halfClosed chan struct{}
	finish     chan error
	done       chan struct{}
}

type googleTTSServer struct {
	texttospeechpb.UnimplementedTextToSpeechServer
	calls   chan *googleTTSCall
	workers sync.WaitGroup
}

func (s *googleTTSServer) StreamingSynthesize(stream texttospeechpb.TextToSpeech_StreamingSynthesizeServer) error {
	call := &googleTTSCall{
		stream: stream, requests: make(chan *texttospeechpb.StreamingSynthesizeRequest, 32),
		halfClosed: make(chan struct{}), finish: make(chan error, 1), done: make(chan struct{}),
	}
	defer close(call.done)
	s.workers.Go(func() {
		for {
			request, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					close(call.halfClosed)
				}
				return
			}
			select {
			case call.requests <- request:
			case <-stream.Context().Done():
				return
			}
		}
	})
	s.calls <- call
	select {
	case err := <-call.finish:
		return err
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
}

type googleTTSWriteGate struct {
	operation string
	entered   chan struct{}
	result    chan error
}

type googleTTSClientStream struct {
	grpc.ClientStream
	rpcCtx      context.Context
	gateMu      sync.Mutex
	gate        *googleTTSWriteGate
	received    chan error
	readEntered chan struct{}
	readRelease chan struct{}
}

func (s *googleTTSClientStream) gateRead() (<-chan struct{}, chan<- struct{}) {
	s.gateMu.Lock()
	defer s.gateMu.Unlock()
	s.readEntered, s.readRelease = make(chan struct{}), make(chan struct{})
	return s.readEntered, s.readRelease
}

func (s *googleTTSClientStream) RecvMsg(message any) error {
	err := s.ClientStream.RecvMsg(message)
	s.gateMu.Lock()
	entered, release := s.readEntered, s.readRelease
	s.readEntered, s.readRelease = nil, nil
	s.gateMu.Unlock()
	if entered != nil {
		close(entered)
		<-release
	}
	s.received <- err
	return err
}

func (s *googleTTSClientStream) gateWrite(operation string) *googleTTSWriteGate {
	s.gateMu.Lock()
	defer s.gateMu.Unlock()
	s.gate = &googleTTSWriteGate{operation: operation, entered: make(chan struct{}), result: make(chan error, 1)}
	return s.gate
}

func (s *googleTTSClientStream) waitForWrite(operation string) error {
	s.gateMu.Lock()
	gate := s.gate
	if gate == nil || gate.operation != operation {
		s.gateMu.Unlock()
		return nil
	}
	s.gate = nil
	s.gateMu.Unlock()
	close(gate.entered)
	select {
	case err := <-gate.result:
		return err
	case <-s.rpcCtx.Done():
		return s.rpcCtx.Err()
	}
}

func (s *googleTTSClientStream) SendMsg(message any) error {
	operation := "text"
	if request, ok := message.(*texttospeechpb.StreamingSynthesizeRequest); ok && request.GetStreamingConfig() != nil {
		operation = "config"
	}
	if err := s.waitForWrite(operation); err != nil {
		return err
	}
	return s.ClientStream.SendMsg(message)
}

func (s *googleTTSClientStream) CloseSend() error {
	if err := s.waitForWrite("close"); err != nil {
		return err
	}
	return s.ClientStream.CloseSend()
}

type googleTTSFixture struct {
	tts       *googleTextToSpeech
	collector *testutil.PacketCollector
	calls     chan *googleTTSCall
	streams   chan *googleTTSClientStream
	nextGate  chan *googleTTSWriteGate
}

type googleTTSDelayedSessionContext struct {
	context.Context
	registered         chan struct{}
	resumeRegistration chan struct{}
	forward            chan struct{}
	forwarded          chan struct{}
}

func (*googleTTSDelayedSessionContext) Value(any) any { return nil }

func (ctx *googleTTSDelayedSessionContext) AfterFunc(callback func()) func() bool {
	stop := context.AfterFunc(ctx.Context, func() {
		<-ctx.forward
		callback()
		close(ctx.forwarded)
	})
	close(ctx.registered)
	<-ctx.resumeRegistration
	return stop
}

func newGoogleTTSFixture(t *testing.T) *googleTTSFixture {
	t.Helper()
	f := &googleTTSFixture{
		collector: testutil.NewPacketCollector(), calls: make(chan *googleTTSCall, 16),
		streams: make(chan *googleTTSClientStream, 16), nextGate: make(chan *googleTTSWriteGate, 1),
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	service := &googleTTSServer{calls: f.calls}
	texttospeechpb.RegisterTextToSpeechServer(server, service)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///google-tts-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			stream, err := streamer(ctx, desc, cc, method, opts...)
			if err != nil {
				return nil, err
			}
			wrapped := &googleTTSClientStream{ClientStream: stream, rpcCtx: ctx, received: make(chan error, 32)}
			select {
			case wrapped.gate = <-f.nextGate:
			default:
			}
			f.streams <- wrapped
			return wrapped, nil
		}),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	client, err := texttospeech.NewClient(ctx, option.WithGRPCConn(conn))
	require.NoError(t, err)
	logger := newTestLogger()
	options, err := NewGoogleOption(logger, newVaultCredential(map[string]interface{}{"key": "test-key"}), utils.Option{})
	require.NoError(t, err)
	f.tts = &googleTextToSpeech{
		ctx: ctx, ctxCancel: cancel, client: client, googleOption: options,
		writeLock: semaphore.NewWeighted(1), logger: logger, onPacket: f.collector.OnPacket,
		normalizer: google_internal.NewGoogleNormalizer(logger, utils.Option{}),
	}
	t.Cleanup(func() {
		cancel()
		require.NoError(t, f.tts.Close(context.Background()))
		server.Stop()
		_ = conn.Close()
		_ = listener.Close()
		<-served
		service.workers.Wait()
	})
	return f
}

func googleTTSReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Google TTS test event")
		var zero T
		return zero
	}
}

func googleTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var result []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			result = append(result, packet)
		}
	}
	return result
}

func TestGoogleTTSSeparateRPCsReuseClient(t *testing.T) {
	f := newGoogleTTSFixture(t)
	client := f.tts.client
	require.NoError(t, f.tts.Initialize())
	require.NoError(t, f.tts.Initialize())
	require.Empty(t, f.streams, "Initialize must not open an idle synthesis RPC")
	for index, contextID := range []string{"first", "second"} {
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "hello & goodbye"}))
		call := googleTTSReceive(t, f.calls)
		googleTTSReceive(t, f.streams)
		config := googleTTSReceive(t, call.requests)
		require.NotNil(t, config.GetStreamingConfig())
		require.Equal(t, DefaultVoice, config.GetStreamingConfig().GetVoice().GetName())
		require.Equal(t, "hello &amp; goodbye", googleTTSReceive(t, call.requests).GetInput().GetText())
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "more"}))
		require.Equal(t, "more", googleTTSReceive(t, call.requests).GetInput().GetText())
		require.Empty(t, f.streams)
		require.NoError(t, call.stream.Send(&texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{1, 2}}))
		f.collector.WaitFor(t, time.Second, "first audio", func() bool { return len(f.collector.AudioPackets()) == index*2+1 })
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		googleTTSReceive(t, call.halfClosed)
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: contextID}))
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "late"}))
		require.Empty(t, call.requests)
		require.Len(t, f.collector.EndPackets(), index)
		require.NoError(t, call.stream.Send(&texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{3, 4}}))
		call.finish <- nil
		f.collector.WaitFor(t, time.Second, "RPC completion", func() bool { return len(f.collector.EndPackets()) == index+1 })
		require.Len(t, f.collector.AudioPackets(), (index+1)*2)
		require.Equal(t, contextID, f.collector.EndPackets()[index].ContextID)
		require.Equal(t, contextID, f.collector.AudioPackets()[index*2+1].ContextID)
		require.Same(t, client, f.tts.client)
		require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: contextID, Text: "retired"}))
		require.Empty(t, f.streams)
	}
	require.Empty(t, googleTTSErrors(f.collector))
}

func TestGoogleTTSEarlyEOFAndFailure(t *testing.T) {
	for _, failure := range []string{"early EOF", "provider error", "error after Done"} {
		t.Run(failure, func(t *testing.T) {
			f := newGoogleTTSFixture(t)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
			call := googleTTSReceive(t, f.calls)
			googleTTSReceive(t, f.streams)
			googleTTSReceive(t, call.requests)
			googleTTSReceive(t, call.requests)
			if failure == "error after Done" {
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				googleTTSReceive(t, call.halfClosed)
			}
			if failure == "early EOF" {
				call.finish <- nil
			} else {
				call.finish <- status.Error(codes.Unavailable, "stream timed out")
			}
			f.collector.WaitFor(t, time.Second, "RPC failure", func() bool { return len(googleTTSErrors(f.collector)) == 1 })
			require.Empty(t, f.collector.EndPackets())
			require.Equal(t, "old", googleTTSErrors(f.collector)[0].ContextID)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.Empty(t, f.streams, "a failed response must not be replayed")
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			fresh := googleTTSReceive(t, f.calls)
			googleTTSReceive(t, fresh.requests)
			require.Equal(t, "fresh", googleTTSReceive(t, fresh.requests).GetInput().GetText())
			require.Len(t, googleTTSErrors(f.collector), 1)
		})
	}
}

func TestGoogleTTSGatedWriteCancellation(t *testing.T) {
	for _, operation := range []string{"config", "text", "close"} {
		for _, cancellation := range []string{"interrupt", "caller", "session", "close"} {
			t.Run(operation+"/"+cancellation, func(t *testing.T) {
				f := newGoogleTTSFixture(t)
				client := f.tts.client
				var call *googleTTSCall
				var stream *googleTTSClientStream
				var gate *googleTTSWriteGate
				if operation == "config" {
					gate = &googleTTSWriteGate{operation: operation, entered: make(chan struct{}), result: make(chan error, 1)}
					f.nextGate <- gate
				} else {
					require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "first"}))
					call = googleTTSReceive(t, f.calls)
					stream = googleTTSReceive(t, f.streams)
					googleTTSReceive(t, call.requests)
					googleTTSReceive(t, call.requests)
					gate = stream.gateWrite(operation)
				}
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if operation == "close" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				go func() { result <- f.tts.Transform(ctx, packet) }()
				googleTTSReceive(t, gate.entered)
				if operation == "config" {
					call = googleTTSReceive(t, f.calls)
					stream = googleTTSReceive(t, f.streams)
				}
				require.Empty(t, result, "the RPC write must remain blocked at its gate")
				switch cancellation {
				case "interrupt":
					require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				case "caller":
					cancel()
				case "session":
					f.tts.ctxCancel()
				case "close":
					require.NoError(t, f.tts.Close(context.Background()))
				}
				err := googleTTSReceive(t, result)
				if cancellation == "interrupt" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, context.Canceled)
				}
				googleTTSReceive(t, call.done)
				require.Empty(t, googleTTSErrors(f.collector))
				require.Empty(t, f.collector.EndPackets())
				if cancellation == "session" || cancellation == "close" {
					return
				}
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Empty(t, f.streams)
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := googleTTSReceive(t, f.calls)
				freshStream := googleTTSReceive(t, f.streams)
				googleTTSReceive(t, fresh.requests)
				require.Equal(t, "fresh", googleTTSReceive(t, fresh.requests).GetInput().GetText())
				require.Same(t, client, f.tts.client)
				gate = freshStream.gateWrite("text")
				go func() {
					result <- f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "blocked again"})
				}()
				googleTTSReceive(t, gate.entered)
				cancel()
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				require.Empty(t, result)
				require.NoError(t, freshStream.Context().Err(), "late cancellation must not cancel the replacement RPC")
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
				require.NoError(t, googleTTSReceive(t, result))
				require.Empty(t, googleTTSErrors(f.collector))
			})
		}
	}
}

func TestGoogleTTSEOFWaitsForCloseSend(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			f := newGoogleTTSFixture(t)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
			call := googleTTSReceive(t, f.calls)
			stream := googleTTSReceive(t, f.streams)
			googleTTSReceive(t, call.requests)
			googleTTSReceive(t, call.requests)
			gate := stream.gateWrite("close")
			t.Cleanup(func() {
				select {
				case gate.result <- errors.New("test cleanup"):
				default:
				}
			})
			result := make(chan error, 1)
			go func() {
				result <- f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"})
			}()
			googleTTSReceive(t, gate.entered)
			call.finish <- nil
			require.ErrorIs(t, googleTTSReceive(t, stream.received), io.EOF)
			require.Empty(t, f.collector.EndPackets(), "EOF must not complete while CloseSend is unresolved")
			if failed {
				gate.result <- errors.New("CloseSend failed")
			} else {
				gate.result <- nil
			}
			require.NoError(t, googleTTSReceive(t, result))
			f.collector.WaitFor(t, time.Second, "terminal packet", func() bool { return len(f.collector.EndPackets())+len(googleTTSErrors(f.collector)) == 1 })
			if failed {
				require.Empty(t, f.collector.EndPackets())
				require.Contains(t, googleTTSErrors(f.collector)[0].ErrMessage(), "CloseSend failed")
			} else {
				require.Len(t, f.collector.EndPackets(), 1)
				require.Empty(t, googleTTSErrors(f.collector))
			}
		})
	}
}

func TestGoogleTTSStaleReadAfterReplacement(t *testing.T) {
	for _, response := range []string{"audio", "EOF", "error"} {
		t.Run(response, func(t *testing.T) {
			f := newGoogleTTSFixture(t)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
			old := googleTTSReceive(t, f.calls)
			oldStream := googleTTSReceive(t, f.streams)
			googleTTSReceive(t, old.requests)
			googleTTSReceive(t, old.requests)
			entered, unblock := oldStream.gateRead()
			release := sync.OnceFunc(func() { close(unblock) })
			t.Cleanup(release)
			switch response {
			case "audio":
				require.NoError(t, old.stream.Send(&texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{1, 2}}))
			case "EOF":
				old.finish <- nil
			case "error":
				old.finish <- status.Error(codes.Unavailable, "old stream failed")
			}
			googleTTSReceive(t, entered)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
			fresh := googleTTSReceive(t, f.calls)
			googleTTSReceive(t, fresh.requests)
			googleTTSReceive(t, fresh.requests)
			release()
			require.NoError(t, fresh.stream.Send(&texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{3, 4}}))
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
			googleTTSReceive(t, fresh.halfClosed)
			fresh.finish <- nil
			f.collector.WaitForTTSEnd(t, time.Second)
			require.NoError(t, f.tts.Close(context.Background()))
			require.Len(t, f.collector.AudioPackets(), 1)
			require.Equal(t, "fresh", f.collector.AudioPackets()[0].ContextID)
			require.Equal(t, []byte{3, 4}, f.collector.AudioPackets()[0].AudioChunk)
			require.Len(t, f.collector.EndPackets(), 1)
			require.Equal(t, "fresh", f.collector.EndPackets()[0].ContextID)
			require.Empty(t, googleTTSErrors(f.collector))
		})
	}
}

func TestGoogleTTSReaderFailureUnblocksWriterBeforeCallback(t *testing.T) {
	for _, failure := range []string{"early EOF", "provider error"} {
		for _, operation := range []string{"text", "close"} {
			if failure == "early EOF" && operation == "close" {
				continue
			}
			t.Run(failure+"/"+operation, func(t *testing.T) {
				f := newGoogleTTSFixture(t)
				entered := make(chan struct{}, 2)
				unblock := make(chan struct{})
				release := sync.OnceFunc(func() { close(unblock) })
				t.Cleanup(release)
				f.tts.onPacket = func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						if _, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
							f.collector.OnPacket(packet)
							entered <- struct{}{}
							<-unblock
							return nil
						}
					}
					return f.collector.OnPacket(packets...)
				}
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
				call := googleTTSReceive(t, f.calls)
				stream := googleTTSReceive(t, f.streams)
				googleTTSReceive(t, call.requests)
				googleTTSReceive(t, call.requests)
				gate := stream.gateWrite(operation)
				var packet internal_type.Packet = internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"}
				if operation == "close" {
					packet = internal_type.TextToSpeechDonePacket{ContextID: "old"}
				}
				result := make(chan error, 1)
				go func() { result <- f.tts.Transform(context.Background(), packet) }()
				googleTTSReceive(t, gate.entered)
				if failure == "early EOF" {
					call.finish <- nil
				} else {
					call.finish <- status.Error(codes.Unavailable, "provider failed")
				}
				googleTTSReceive(t, entered)
				require.ErrorIs(t, stream.rpcCtx.Err(), context.Canceled, "reader must cancel the RPC before publishing failure")
				require.NoError(t, googleTTSReceive(t, result), "stalled writer must exit while the reader callback is blocked")
				require.Len(t, googleTTSErrors(f.collector), 1)
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
				require.Empty(t, f.collector.EndPackets())
				release()
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
				fresh := googleTTSReceive(t, f.calls)
				googleTTSReceive(t, fresh.requests)
				require.Equal(t, "fresh", googleTTSReceive(t, fresh.requests).GetInput().GetText())
				require.Len(t, googleTTSErrors(f.collector), 1)
			})
		}
	}
}

func TestGoogleTTSCloseJoinsBlockedAudio(t *testing.T) {
	f := newGoogleTTSFixture(t)
	entered := make(chan struct{})
	unblock := make(chan struct{})
	release := sync.OnceFunc(func() { close(unblock) })
	t.Cleanup(release)
	f.tts.onPacket = func(packets ...internal_type.Packet) error {
		for _, packet := range packets {
			if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
				close(entered)
				<-unblock
			}
		}
		return f.collector.OnPacket(packets...)
	}
	require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
	call := googleTTSReceive(t, f.calls)
	stream := googleTTSReceive(t, f.streams)
	googleTTSReceive(t, call.requests)
	googleTTSReceive(t, call.requests)
	require.NoError(t, call.stream.Send(&texttospeechpb.StreamingSynthesizeResponse{AudioContent: []byte{1, 2}}))
	googleTTSReceive(t, entered)
	require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	googleTTSReceive(t, call.halfClosed)
	closed := make(chan error, 1)
	go func() { closed <- f.tts.Close(context.Background()) }()
	googleTTSReceive(t, stream.rpcCtx.Done())
	require.Empty(t, closed, "Close must wait for the reader callback")
	release()
	require.NoError(t, googleTTSReceive(t, closed))
	require.Empty(t, f.collector.EndPackets())
	require.Empty(t, googleTTSErrors(f.collector))
}

func TestGoogleTTSWaitingWriterCancellation(t *testing.T) {
	for _, cancellation := range []string{"caller", "session"} {
		t.Run(cancellation, func(t *testing.T) {
			f := newGoogleTTSFixture(t)
			require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
			call := googleTTSReceive(t, f.calls)
			stream := googleTTSReceive(t, f.streams)
			googleTTSReceive(t, call.requests)
			googleTTSReceive(t, call.requests)
			gate := stream.gateWrite("text")
			owner := make(chan error, 1)
			go func() {
				owner <- f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
			}()
			googleTTSReceive(t, gate.entered)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			waiting := make(chan error, 1)
			go func() { waiting <- f.tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}) }()
			if cancellation == "caller" {
				cancel()
			} else {
				f.tts.ctxCancel()
			}
			require.ErrorIs(t, googleTTSReceive(t, waiting), context.Canceled)
			if cancellation == "caller" {
				require.Empty(t, owner)
				require.NoError(t, stream.rpcCtx.Err(), "canceling a waiter must not cancel the current writer")
				require.NoError(t, f.tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
				require.NoError(t, googleTTSReceive(t, owner))
			} else {
				require.ErrorIs(t, googleTTSReceive(t, owner), context.Canceled)
			}
			require.Empty(t, googleTTSErrors(f.collector))
		})
	}
}

func TestGoogleTTSRechecksSessionAfterCloseBeforeForwarding(t *testing.T) {
	f := newGoogleTTSFixture(t)
	session := &googleTTSDelayedSessionContext{
		Context: f.tts.ctx, registered: make(chan struct{}), resumeRegistration: make(chan struct{}),
		forward: make(chan struct{}), forwarded: make(chan struct{}),
	}
	f.tts.ctx = session
	resumeRegistration := sync.OnceFunc(func() { close(session.resumeRegistration) })
	forward := sync.OnceFunc(func() { close(session.forward) })
	t.Cleanup(func() {
		f.tts.ctxCancel()
		resumeRegistration()
		forward()
		googleTTSReceive(t, session.forwarded)
	})
	require.NoError(t, f.tts.writeLock.Acquire(context.Background(), 1))
	releaseWriter := sync.OnceFunc(func() { f.tts.writeLock.Release(1) })
	t.Cleanup(releaseWriter)
	result := make(chan error, 1)
	go func() {
		result <- f.tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "late", Text: "late"})
	}()
	googleTTSReceive(t, session.registered)

	closed := make(chan error, 1)
	go func() { closed <- f.tts.Close(context.Background()) }()
	googleTTSReceive(t, session.Done())
	require.ErrorIs(t, session.Err(), context.Canceled)
	require.Empty(t, session.forwarded, "cancellation forwarding must still be blocked")
	releaseWriter()
	require.NoError(t, googleTTSReceive(t, closed))
	require.Nil(t, f.tts.client)

	// Let Transform acquire the permit only after Close has disposed of the client.
	resumeRegistration()
	require.ErrorIs(t, googleTTSReceive(t, result), context.Canceled)
	require.Empty(t, session.forwarded)
	require.Empty(t, f.streams)
	require.Empty(t, f.collector.EndPackets())
	require.Empty(t, googleTTSErrors(f.collector))
}
