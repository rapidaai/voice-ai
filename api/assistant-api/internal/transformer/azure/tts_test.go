package internal_transformer_azure

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type azureTTSFakeClient struct {
	start func(string, bool) (azureSynthesisStream, error)
	stop  func() error
	close func()
}

func (c *azureTTSFakeClient) StartSpeaking(text string, ssml bool) (azureSynthesisStream, error) {
	return c.start(text, ssml)
}

func (c *azureTTSFakeClient) StopSpeaking() error {
	if c.stop != nil {
		return c.stop()
	}
	return nil
}

func (c *azureTTSFakeClient) Close() {
	if c.close != nil {
		c.close()
	}
}

type azureTTSRead struct {
	audio []byte
	err   error
}

type azureTTSFakeStream struct {
	chunks    chan azureTTSRead
	reading   chan struct{}
	stopped   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
	status    common.StreamStatus
	statusErr error
}

func newAzureTTSStream(chunks ...azureTTSRead) *azureTTSFakeStream {
	stream := &azureTTSFakeStream{
		chunks: make(chan azureTTSRead, 16), reading: make(chan struct{}, 16),
		stopped: make(chan struct{}), closed: make(chan struct{}), status: common.StreamStatusAllData,
	}
	for _, chunk := range chunks {
		stream.chunks <- chunk
	}
	return stream
}

func (s *azureTTSFakeStream) Read(buffer []byte) (int, error) {
	s.reading <- struct{}{}
	select {
	case chunk := <-s.chunks:
		return copy(buffer, chunk.audio), chunk.err
	case <-s.stopped:
		return 0, context.Canceled
	}
}

func (s *azureTTSFakeStream) GetStatus() (common.StreamStatus, error) {
	return s.status, s.statusErr
}

func (s *azureTTSFakeStream) Close() { s.closeOnce.Do(func() { close(s.closed) }) }

func newAzureTTSFixture(t *testing.T, client *azureTTSFakeClient) (*azureTextToSpeech, *testutil.PacketCollector) {
	t.Helper()
	collector := testutil.NewPacketCollector()
	transformer, err := NewAzureTextToSpeech(context.Background(), newTestLogger(), newVaultCredential(map[string]interface{}{
		"subscription_key": "test-sub-key", "endpoint": "https://test.cognitiveservices.azure.com",
	}), collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*azureTextToSpeech)
	if client != nil {
		tts.client = client
	}
	t.Cleanup(func() { require.NoError(t, tts.Close(context.Background())) })
	return tts, collector
}

func azureTTSErrors(collector *testutil.PacketCollector) []internal_type.TextToSpeechErrorPacket {
	var result []internal_type.TextToSpeechErrorPacket
	for _, packet := range collector.GetPackets() {
		if packet, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
			result = append(result, packet)
		}
	}
	return result
}

func TestAzureTTSMultipleRequestsReuseClient(t *testing.T) {
	var texts []string
	var streams []*azureTTSFakeStream
	client := &azureTTSFakeClient{start: func(text string, ssml bool) (azureSynthesisStream, error) {
		texts = append(texts, text)
		stream := newAzureTTSStream(azureTTSRead{audio: []byte{1, 2}}, azureTTSRead{err: io.EOF})
		streams = append(streams, stream)
		return stream, nil
	}}
	tts, collector := newAzureTTSFixture(t, client)
	require.NoError(t, tts.Initialize())
	require.NoError(t, tts.Initialize())
	for index, id := range []string{"old", "fresh"} {
		for _, text := range []string{"hello", "more"} {
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: text}))
		}
		require.Len(t, collector.EndPackets(), index)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: id}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: id, Text: "late"}))
		require.Len(t, collector.EndPackets(), index+1)
		require.Equal(t, id, collector.EndPackets()[index].ContextID)
		require.Same(t, client, tts.client)
	}
	require.Equal(t, []string{"hello", "more", "hello", "more"}, texts)
	require.Len(t, collector.AudioPackets(), 4)
	require.Equal(t, "old", collector.AudioPackets()[0].ContextID)
	require.Equal(t, "fresh", collector.AudioPackets()[2].ContextID)
	for _, stream := range streams {
		select {
		case <-stream.closed:
		default:
			t.Fatal("request stream was not disposed")
		}
	}
	require.Empty(t, azureTTSErrors(collector))
}

func TestAzureTTSRequestFailures(t *testing.T) {
	for _, failure := range []string{"start", "read", "status", "partial", "canceled"} {
		t.Run(failure, func(t *testing.T) {
			stream := newAzureTTSStream(azureTTSRead{err: io.EOF})
			if failure == "read" {
				<-stream.chunks
				stream.chunks <- azureTTSRead{err: errors.New("read failed")}
			}
			if failure == "status" {
				stream.statusErr = errors.New("status failed")
			}
			if failure == "partial" {
				stream.status = common.StreamStatusPartialData
			}
			if failure == "canceled" {
				stream.status = common.StreamStatusCanceled
			}
			starts := 0
			client := &azureTTSFakeClient{start: func(string, bool) (azureSynthesisStream, error) {
				starts++
				if failure == "start" {
					return nil, errors.New("start failed")
				}
				return stream, nil
			}}
			tts, collector := newAzureTTSFixture(t, client)
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
			require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
			require.Equal(t, 1, starts)
			require.Len(t, azureTTSErrors(collector), 1)
			require.Equal(t, "old", azureTTSErrors(collector)[0].ContextID)
			require.Empty(t, collector.EndPackets())
			if failure != "start" {
				select {
				case <-stream.closed:
				default:
					t.Fatal("failed stream was not disposed")
				}
			}
		})
	}
}

func TestAzureTTSCancellationJoinsStopBeforeReuse(t *testing.T) {
	for _, phase := range []string{"start", "read"} {
		for _, cancellation := range []string{"interrupt", "caller", "session", "close"} {
			t.Run(phase+"/"+cancellation, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					stream := newAzureTTSStream()
					started := make(chan struct{})
					stopEntered := make(chan struct{})
					stopRelease := make(chan struct{})
					releaseStop := sync.OnceFunc(func() { close(stopRelease) })
					stopped := sync.OnceFunc(func() { close(stream.stopped); close(stopEntered) })
					starts := 0
					client := &azureTTSFakeClient{
						start: func(string, bool) (azureSynthesisStream, error) {
							starts++
							if starts > 1 {
								return newAzureTTSStream(azureTTSRead{audio: []byte{3, 4}}, azureTTSRead{err: io.EOF}), nil
							}
							close(started)
							if phase == "start" {
								<-stream.stopped
								return nil, context.Canceled
							}
							return stream, nil
						},
						stop: func() error { stopped(); <-stopRelease; return nil },
					}
					tts, collector := newAzureTTSFixture(t, client)
					t.Cleanup(releaseStop)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					result := make(chan error, 1)
					go func() {
						result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "blocked"})
					}()
					<-started
					if phase == "read" {
						<-stream.reading
					}
					synctest.Wait()
					require.Empty(t, result)
					closed := make(chan error, 1)
					switch cancellation {
					case "interrupt":
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					case "caller":
						cancel()
					case "session":
						tts.ctxCancel()
					case "close":
						go func() { closed <- tts.Close(context.Background()) }()
					}
					<-stopEntered
					synctest.Wait()
					require.Empty(t, result, "Transform must join the in-flight StopSpeaking call")
					select {
					case <-stream.closed:
						t.Fatal("stream disposed before StopSpeaking returned")
					default:
					}
					fresh := make(chan error, 1)
					if cancellation == "interrupt" || cancellation == "caller" {
						go func() {
							fresh <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
						}()
						synctest.Wait()
						require.Equal(t, 1, starts, "replacement must wait until the previous stop completes")
					}
					releaseStop()
					if cancellation == "interrupt" {
						require.NoError(t, <-result)
					} else {
						require.ErrorIs(t, <-result, context.Canceled)
					}
					if cancellation == "close" {
						require.NoError(t, <-closed)
					}
					if cancellation == "interrupt" || cancellation == "caller" {
						require.NoError(t, <-fresh)
						require.Len(t, collector.AudioPackets(), 1)
						require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
					}
					require.Empty(t, azureTTSErrors(collector))
					require.Empty(t, collector.EndPackets())
				})
			})
		}
	}
}

func TestAzureTTSStopAfterLateStartAcknowledgment(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newAzureTTSStream(azureTTSRead{audio: []byte{1, 2}}, azureTTSRead{err: io.EOF})
		started := make(chan struct{})
		acknowledge := make(chan struct{})
		releaseStart := sync.OnceFunc(func() { close(acknowledge) })
		stops := make(chan struct{}, 4)
		starts := 0
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) {
				starts++
				if starts == 1 {
					close(started)
					<-acknowledge
					return stream, nil
				}
				return newAzureTTSStream(azureTTSRead{audio: []byte{3, 4}}, azureTTSRead{err: io.EOF}), nil
			},
			stop: func() error { stops <- struct{}{}; return nil },
		}
		tts, collector := newAzureTTSFixture(t, client)
		t.Cleanup(releaseStart)
		result := make(chan error, 1)
		go func() {
			result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		<-started
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		<-stops
		synctest.Wait()
		require.Empty(t, result)
		releaseStart()
		require.NoError(t, <-result)
		require.Len(t, stops, 1, "a late start acknowledgment needs a settled follow-up StopSpeaking")
		select {
		case <-stream.closed:
		default:
			t.Fatal("late start result was not disposed")
		}
		require.Empty(t, collector.AudioPackets())
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"}))
		require.Len(t, collector.AudioPackets(), 1)
		require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
		require.Empty(t, azureTTSErrors(collector))
	})
}

func TestAzureTTSStaleInterruptAndLateCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		old, fresh := newAzureTTSStream(), newAzureTTSStream()
		requests := make(chan *azureTTSFakeStream, 2)
		requests <- old
		requests <- fresh
		var active *azureTTSFakeStream
		var gateMu sync.Mutex
		stops := 0
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) {
				stream := <-requests
				gateMu.Lock()
				active = stream
				gateMu.Unlock()
				return stream, nil
			},
			stop: func() error {
				gateMu.Lock()
				defer gateMu.Unlock()
				stops++
				// Return queued audio after cancellation to exercise the post-read ownership check.
				active.chunks <- azureTTSRead{audio: []byte{9, 9}, err: io.EOF}
				return nil
			},
		}
		tts, collector := newAzureTTSFixture(t, client)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result := make(chan error, 1)
		go func() {
			result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		<-old.reading
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, <-result)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Len(t, requests, 1)
		go func() {
			result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
		}()
		<-fresh.reading
		cancel()
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		synctest.Wait()
		require.Empty(t, result)
		require.Equal(t, 1, stops)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
		require.NoError(t, <-result)
		require.Empty(t, collector.AudioPackets())
		require.Empty(t, collector.EndPackets())
		require.Empty(t, azureTTSErrors(collector))
	})
}

func TestAzureTTSCloseJoinsBlockedAudio(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newAzureTTSStream(azureTTSRead{audio: []byte{1, 2}})
		stop := sync.OnceFunc(func() { close(stream.stopped) })
		clientClosed := make(chan struct{})
		closeClient := sync.OnceFunc(func() { close(clientClosed) })
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) { return stream, nil },
			stop:  func() error { stop(); return nil }, close: closeClient,
		}
		tts, collector := newAzureTTSFixture(t, client)
		entered, unblock := make(chan struct{}), make(chan struct{})
		release := sync.OnceFunc(func() { close(unblock) })
		t.Cleanup(release)
		tts.onPacket = func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
					close(entered)
					<-unblock
				}
			}
			return collector.OnPacket(packets...)
		}
		result := make(chan error, 1)
		go func() {
			result <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		<-entered
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		closed := make(chan error, 1)
		go func() { closed <- tts.Close(context.Background()) }()
		synctest.Wait()
		require.Empty(t, closed)
		select {
		case <-clientClosed:
			t.Fatal("client closed before the active audio callback returned")
		default:
		}
		release()
		require.ErrorIs(t, <-result, context.Canceled)
		require.NoError(t, <-closed)
		require.Empty(t, collector.EndPackets())
		require.Empty(t, azureTTSErrors(collector))
	})
}

func TestAzureTTSQueuedCallerCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newAzureTTSStream()
		stop := sync.OnceFunc(func() { close(stream.stopped) })
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) { return stream, nil },
			stop:  func() error { stop(); return nil },
		}
		tts, collector := newAzureTTSFixture(t, client)
		owner := make(chan error, 1)
		go func() {
			owner <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		<-stream.reading
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		queued := make(chan error, 1)
		go func() { queued <- tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}) }()
		synctest.Wait()
		require.Empty(t, queued)
		cancel()
		require.ErrorIs(t, <-queued, context.Canceled)
		require.Empty(t, owner)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, <-owner)
		require.Empty(t, azureTTSErrors(collector))
	})
}

func TestAzureTTSStopFailureReturnedWithoutSyntheticError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stream := newAzureTTSStream()
		stopError := errors.New("stop failed")
		stop := sync.OnceFunc(func() { close(stream.stopped) })
		client := &azureTTSFakeClient{
			start: func(string, bool) (azureSynthesisStream, error) { return stream, nil },
			stop:  func() error { stop(); return stopError },
		}
		tts, collector := newAzureTTSFixture(t, client)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result := make(chan error, 1)
		go func() {
			result <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
		}()
		<-stream.reading
		cancel()
		err := <-result
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, stopError)
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		require.Empty(t, collector.EndPackets())
		require.Empty(t, azureTTSErrors(collector))
	})
}

func TestAzureTTSNativeInitializeIsIdempotent(t *testing.T) {
	tts, collector := newAzureTTSFixture(t, nil)
	require.NoError(t, tts.Initialize())
	client := tts.client
	require.NotNil(t, client)
	require.NoError(t, tts.Initialize())
	require.Same(t, client, tts.client)
	require.Empty(t, collector.AudioPackets())
	require.NoError(t, tts.Close(context.Background()))
	require.ErrorIs(t, tts.Initialize(), context.Canceled)
}

func TestAzureTTSUninitializedRejectsInput(t *testing.T) {
	tts, collector := newAzureTTSFixture(t, nil)
	require.Nil(t, tts.client)
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "hello"}))
	require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
	require.Len(t, azureTTSErrors(collector), 1)
	require.Empty(t, collector.EndPackets())
}

func TestAzureTTSInitializeHonorsCancellation(t *testing.T) {
	tts, collector := newAzureTTSFixture(t, nil)
	tts.ctxCancel()
	require.ErrorIs(t, tts.Initialize(), context.Canceled)
	require.Nil(t, tts.client)
	require.Empty(t, azureTTSErrors(collector))
}

func TestAzureTTSReplacementDuringBlockedStop(t *testing.T) {
	for _, name := range []string{"replacement", "interrupted replacement", "canceled replacement", "superseded replacement"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				old := newAzureTTSStream()
				stopEntered := make(chan struct{})
				stopRelease := make(chan struct{})
				release := sync.OnceFunc(func() { close(stopRelease) })
				stop := sync.OnceFunc(func() { close(old.stopped); close(stopEntered) })
				starts := 0
				client := &azureTTSFakeClient{
					start: func(string, bool) (azureSynthesisStream, error) {
						starts++
						if starts == 1 {
							return old, nil
						}
						return newAzureTTSStream(azureTTSRead{audio: []byte{3, 4}}, azureTTSRead{err: io.EOF}), nil
					},
					stop: func() error { stop(); <-stopRelease; return nil },
				}
				tts, collector := newAzureTTSFixture(t, client)
				t.Cleanup(release)
				oldResult := make(chan error, 1)
				go func() {
					oldResult <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"})
				}()
				<-old.reading
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				freshResult := make(chan error, 1)
				go func() {
					freshResult <- tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "fresh"})
				}()
				<-stopEntered
				synctest.Wait()
				require.Equal(t, 1, starts)
				require.Empty(t, freshResult)
				newestResult := make(chan error, 1)
				switch name {
				case "interrupted replacement":
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "fresh"}))
				case "canceled replacement":
					cancel()
				case "superseded replacement":
					go func() {
						newestResult <- tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "newest", Text: "newest"})
					}()
					synctest.Wait()
					require.Equal(t, 1, starts)
				}
				release()
				require.NoError(t, <-oldResult)
				if name == "canceled replacement" {
					require.ErrorIs(t, <-freshResult, context.Canceled)
				} else {
					require.NoError(t, <-freshResult)
				}
				switch name {
				case "interrupted replacement", "canceled replacement":
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "fresh", Text: "late"}))
					require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "fresh"}))
					require.Equal(t, 1, starts, "a canceled replacement must not start after the old stop finishes")
					require.Empty(t, collector.AudioPackets())
				case "superseded replacement":
					require.NoError(t, <-newestResult)
					require.Equal(t, 2, starts)
					require.Len(t, collector.AudioPackets(), 1)
					require.Equal(t, "newest", collector.AudioPackets()[0].ContextID)
				default:
					require.Equal(t, 2, starts)
					require.Len(t, collector.AudioPackets(), 1)
					require.Equal(t, "fresh", collector.AudioPackets()[0].ContextID)
				}
				require.Empty(t, azureTTSErrors(collector))
			})
		})
	}
}
