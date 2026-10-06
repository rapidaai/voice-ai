package internal_transformer_aws

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/tests/testutil"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type awsTestTransport func(*http.Request) (*http.Response, error)

func (transport awsTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type awsTestBody struct {
	read       func([]byte) (int, error)
	closed     chan struct{}
	closeCount atomic.Int32
}

func (body *awsTestBody) Read(buffer []byte) (int, error) { return body.read(buffer) }

func (body *awsTestBody) Close() error {
	if body.closeCount.Add(1) == 1 {
		close(body.closed)
	}
	return nil
}

func newAWSTTSTest(t *testing.T, transport awsTestTransport) (*awsTTS, *testutil.PacketCollector) {
	t.Helper()
	client := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() { http.DefaultClient = client })
	collector := testutil.NewPacketCollector()
	transformer, err := NewAWSTextToSpeech(context.Background(), testutil.NewTestLogger(),
		testutil.BuildCredential(map[string]string{"access_key_id": "test-access", "secret_access_key": "test-secret"}),
		collector.OnPacket, utils.Option{})
	require.NoError(t, err)
	tts := transformer.(*awsTTS)
	t.Cleanup(func() { require.NoError(t, tts.Close(context.Background())) })
	return tts, collector
}

func TestAWSTTSTerminalResult(t *testing.T) {
	for _, scenario := range []string{"eof", "read_error", "transport_error", "status_error"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				body := &awsTestBody{closed: make(chan struct{})}
				body.read = func(buffer []byte) (int, error) {
					if scenario == "read_error" {
						return copy(buffer, []byte{1, 2}), io.ErrUnexpectedEOF
					}
					return copy(buffer, []byte{1, 2}), io.EOF
				}
				var calls atomic.Int32
				tts, collector := newAWSTTSTest(t, func(request *http.Request) (*http.Response, error) {
					calls.Add(1)
					assert.Equal(t, http.MethodPost, request.Method)
					assert.Equal(t, "https://polly.us-east-1.amazonaws.com/v1/speech", request.URL.String())
					assert.Contains(t, request.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=test-access/")
					assert.Contains(t, request.Header.Get("Authorization"), "/us-east-1/polly/aws4_request")
					assert.NotEmpty(t, request.Header.Get("X-Amz-Date"))
					assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
					var payload map[string]interface{}
					assert.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
					assert.Equal(t, map[string]interface{}{
						"Text": "hello world", "VoiceId": AWS_DEFAULT_VOICE,
						"LanguageCode": AWS_DEFAULT_LANGUAGE, "OutputFormat": "pcm", "SampleRate": "16000",
						"Engine": AWS_DEFAULT_ENGINE, "TextType": "text",
					}, payload)
					if scenario == "transport_error" {
						return nil, errors.New("connection failed")
					}
					status := http.StatusOK
					if scenario == "status_error" {
						status = http.StatusBadGateway
					}
					return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
				})
				tts.onPacket = func(packets ...internal_type.Packet) error {
					for _, packet := range packets {
						switch packet.(type) {
						case internal_type.TextToSpeechEndPacket, internal_type.TextToSpeechErrorPacket:
							if scenario != "transport_error" {
								assert.EqualValues(t, 1, body.closeCount.Load(), "body must close before terminal callback")
							}
						}
					}
					return collector.OnPacket(packets...)
				}
				require.NoError(t, tts.Initialize())
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "hello "}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "stale"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "stale"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TurnChangePacket{}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "world"}))
				require.Zero(t, calls.Load())
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"}))
				synctest.Wait()
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "late"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"}))
				require.EqualValues(t, 1, calls.Load())
				var failures []internal_type.TextToSpeechErrorPacket
				for _, packet := range collector.GetPackets() {
					if failure, ok := packet.(internal_type.TextToSpeechErrorPacket); ok {
						failures = append(failures, failure)
					}
				}
				if scenario == "eof" {
					require.Empty(t, failures)
					require.Len(t, collector.EndPackets(), 1)
				} else {
					require.Len(t, failures, 1)
					require.Empty(t, collector.EndPackets())
				}
				if scenario == "eof" || scenario == "read_error" {
					require.Len(t, collector.AudioPackets(), 1)
					require.Equal(t, []byte{1, 2}, collector.AudioPackets()[0].AudioChunk)
				} else {
					require.Empty(t, collector.AudioPackets())
				}
				if scenario != "transport_error" {
					require.EqualValues(t, 1, body.closeCount.Load())
				}
			})
		})
	}
}

func TestAWSTTSCancellation(t *testing.T) {
	for _, phase := range []string{"headers", "body"} {
		for _, action := range []string{"interrupt", "replacement", "caller", "session", "close"} {
			t.Run(phase+"/"+action, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					started := make(chan context.Context, 1)
					body := &awsTestBody{closed: make(chan struct{})}
					var calls atomic.Int32
					tts, collector := newAWSTTSTest(t, func(request *http.Request) (*http.Response, error) {
						if calls.Add(1) > 1 {
							return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("new")), Header: make(http.Header)}, nil
						}
						if phase == "headers" {
							started <- request.Context()
							<-request.Context().Done()
							return nil, request.Context().Err()
						}
						body.read = func(buffer []byte) (int, error) {
							started <- request.Context()
							<-body.closed
							return copy(buffer, "stale"), io.EOF
						}
						return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
					})
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
					require.NoError(t, tts.Transform(ctx, internal_type.TextToSpeechDonePacket{ContextID: "old"}))
					requestCtx := <-started
					switch action {
					case "interrupt":
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
					case "replacement":
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
						require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
					case "caller":
						cancel()
					case "session":
						tts.ctxCancel()
					case "close":
						require.NoError(t, tts.Close(context.Background()))
					}
					synctest.Wait()
					require.ErrorIs(t, requestCtx.Err(), context.Canceled)
					for _, packet := range collector.GetPackets() {
						_, failed := packet.(internal_type.TextToSpeechErrorPacket)
						require.False(t, failed, "cancellation must not report a provider failure")
					}
					if action == "replacement" {
						require.Len(t, collector.EndPackets(), 1)
						require.Equal(t, "new", collector.EndPackets()[0].ContextID)
						require.Len(t, collector.AudioPackets(), 1)
						require.Equal(t, "new", collector.AudioPackets()[0].ContextID)
					} else {
						require.Empty(t, collector.EndPackets())
						require.Empty(t, collector.AudioPackets())
					}
					if phase == "body" {
						require.EqualValues(t, 1, body.closeCount.Load())
					}
				})
			})
		}
	}
}

func TestAWSTTSCloseJoinsAudioCallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector := newAWSTTSTest(t, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("audio")), Header: make(http.Header)}, nil
		})
		entered, release := make(chan struct{}), make(chan struct{})
		t.Cleanup(func() {
			select {
			case <-release:
			default:
				close(release)
			}
		})
		tts.onPacket = func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if _, ok := packet.(internal_type.TextToSpeechAudioPacket); ok {
					close(entered)
					<-release
				}
			}
			return collector.OnPacket(packets...)
		}
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "text"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"}))
		<-entered
		closed := make(chan error, 1)
		go func() { closed <- tts.Close(context.Background()) }()
		synctest.Wait()
		require.Empty(t, closed, "Close must join the active callback")
		close(release)
		require.NoError(t, <-closed)
		require.Empty(t, collector.EndPackets())
		require.ErrorIs(t, tts.Initialize(), context.Canceled)
	})
}

func TestAWSTTSTerminalClaimPrecedesLaterInterruption(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tts, collector := newAWSTTSTest(t, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("audio")), Header: make(http.Header)}, nil
		})
		entered, release := make(chan struct{}), make(chan struct{})
		t.Cleanup(func() {
			select {
			case <-release:
			default:
				close(release)
			}
		})
		tts.onPacket = func(packets ...internal_type.Packet) error {
			for _, packet := range packets {
				if end, ok := packet.(internal_type.TextToSpeechEndPacket); ok && end.ContextID == "old" {
					close(entered)
					<-release
				}
			}
			return collector.OnPacket(packets...)
		}
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		<-entered
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
		synctest.Wait()
		require.Len(t, collector.EndPackets(), 1)
		require.Equal(t, "new", collector.EndPackets()[0].ContextID)
		close(release)
		synctest.Wait()
		require.Len(t, collector.EndPackets(), 2)
		require.Equal(t, "old", collector.EndPackets()[1].ContextID)
		for _, packet := range collector.GetPackets() {
			if event, ok := packet.(internal_type.ObservabilityEventRecordPacket); ok {
				require.NotEqual(t, observability.TTSInterrupted, event.Record.Event)
			}
		}
	})
}

func TestAWSTTSInterruptDiscardsBufferedText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		tts, collector := newAWSTTSTest(t, func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("new")), Header: make(http.Header)}, nil
		})
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechInterruptPacket{ContextID: "old"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "old", Text: "late"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "old"}))
		synctest.Wait()
		require.Zero(t, calls.Load())
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "new", Text: "new"}))
		require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "new"}))
		synctest.Wait()
		require.EqualValues(t, 1, calls.Load())
		require.Len(t, collector.EndPackets(), 1)
		require.Equal(t, "new", collector.EndPackets()[0].ContextID)
	})
}

func TestAWSTTSConfiguredRequest(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		options  utils.Option
		text     string
		textType string
	}{
		{name: "plain", text: "hello and goodbye", textType: "text"},
		{
			name: "ssml",
			options: utils.Option{
				"speaker.conjunction.boundaries": "and", "speaker.conjunction.break": "250",
			},
			text: `<speak>hello and<break time="250ms"/> goodbye</speak>`, textType: "ssml",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := http.DefaultClient
				http.DefaultClient = &http.Client{Transport: awsTestTransport(func(request *http.Request) (*http.Response, error) {
					assert.Equal(t, "https://polly.eu-west-1.amazonaws.com/v1/speech", request.URL.String())
					assert.Contains(t, request.Header.Get("Authorization"), "/eu-west-1/polly/aws4_request")
					var payload map[string]interface{}
					assert.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
					assert.Equal(t, map[string]interface{}{
						"Text": scenario.text, "TextType": scenario.textType,
						"VoiceId": "Amy", "Engine": "standard", "LanguageCode": "en-GB",
						"OutputFormat": "pcm", "SampleRate": "16000",
					}, payload)
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("audio")), Header: make(http.Header)}, nil
				})}
				t.Cleanup(func() { http.DefaultClient = client })
				options := utils.Option{
					"speak.voice.id": "Amy", "speak.model": "standard", "speak.language": "en-GB",
				}
				for key, value := range scenario.options {
					options[key] = value
				}
				collector := testutil.NewPacketCollector()
				tts, err := NewAWSTextToSpeech(context.Background(), testutil.NewTestLogger(),
					testutil.BuildCredential(map[string]string{
						"access_key_id": "test-access", "secret_access_key": "test-secret", "region": "eu-west-1",
					}), collector.OnPacket, options)
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, tts.Close(context.Background())) })
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechTextPacket{ContextID: "active", Text: "hello and goodbye"}))
				require.NoError(t, tts.Transform(context.Background(), internal_type.TextToSpeechDonePacket{ContextID: "active"}))
				synctest.Wait()
				require.Len(t, collector.EndPackets(), 1)
			})
		})
	}
}
