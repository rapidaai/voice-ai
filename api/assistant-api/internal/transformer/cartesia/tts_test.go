package internal_transformer_cartesia

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testutil "github.com/rapidaai/api/assistant-api/internal/transformer/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartesiaTTSReadLoopConnectionOwnership(t *testing.T) {
	for _, state := range []string{"current", "interrupted", "replaced"} {
		t.Run(state, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			serverConnections := make(chan *websocket.Conn, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				connection, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade websocket: %v", err)
					return
				}
				serverConnections <- connection
			}))
			t.Cleanup(server.Close)
			connectionURL := "ws" + strings.TrimPrefix(server.URL, "http")
			connection, _, err := websocket.DefaultDialer.Dial(connectionURL, nil)
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			remote := waitCartesiaTTSServerConnection(t, serverConnections)
			t.Cleanup(func() { _ = remote.Close() })

			ctx, cancel := context.WithCancel(context.Background())
			collector := testutil.NewPacketCollector()
			transformer := &cartesiaTTS{
				ctx:           ctx,
				ctxCancel:     cancel,
				connection:    connection,
				contextId:     "original-message",
				ttsStartedAt:  time.Now(),
				textClosed:    true,
				pendingDrains: 1,
				logger:        newTestLogger(),
				onPacket:      collector.OnPacket,
			}
			readerReady := make(chan struct{})
			releaseReader := make(chan struct{})
			release := sync.OnceFunc(func() { close(releaseReader) })
			// Hold the real reader while audio queues and connection ownership changes.
			connection.SetPingHandler(func(string) error {
				close(readerReady)
				<-releaseReader
				return nil
			})
			readerDone := make(chan struct{})
			go func() {
				defer close(readerDone)
				transformer.readLoop(connection)
			}()
			t.Cleanup(func() {
				cancel()
				release()
				_ = connection.Close()
				select {
				case <-readerDone:
				case <-time.After(3 * time.Second):
					t.Error("TTS reader did not exit")
				}
			})
			require.NoError(t, remote.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
			select {
			case <-readerReady:
			case <-time.After(3 * time.Second):
				t.Fatal("TTS reader did not reach the queued frame barrier")
			}
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "chunk", "context_id": "original-message",
				"data": base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
			}))
			require.NoError(t, remote.WriteJSON(map[string]interface{}{
				"type": "done", "context_id": "original-message", "done": true,
			}))
			require.NoError(t, remote.Close())

			switch state {
			case "interrupted":
				transformer.mu.Lock()
				transformer.connection = nil
				transformer.resetTurnLocked("")
				transformer.mu.Unlock()
			case "replaced":
				replacement, _, err := websocket.DefaultDialer.Dial(connectionURL, nil)
				require.NoError(t, err)
				t.Cleanup(func() { _ = replacement.Close() })
				replacementRemote := waitCartesiaTTSServerConnection(t, serverConnections)
				t.Cleanup(func() { _ = replacementRemote.Close() })
				transformer.mu.Lock()
				transformer.connection = replacement
				transformer.resetTurnLocked("replacement-message")
				transformer.ttsStartedAt = time.Now()
				transformer.mu.Unlock()
			}
			release()
			select {
			case <-readerDone:
			case <-time.After(3 * time.Second):
				t.Fatal("TTS reader did not finish queued frames")
			}

			if state != "current" {
				assert.Empty(t, collector.AudioPackets(), "obsolete connection must not emit audio")
				assert.Empty(t, collector.MetricPackets(), "obsolete connection must not emit latency")
				assert.Empty(t, collector.EndPackets(), "obsolete connection must not finish the replacement message")
				return
			}
			require.Len(t, collector.AudioPackets(), 1)
			assert.Equal(t, "original-message", collector.AudioPackets()[0].ContextID)
			assert.Equal(t, []byte{1, 2, 3, 4}, collector.AudioPackets()[0].AudioChunk)
			require.Len(t, collector.MetricPackets(), 1)
			assert.Equal(t, "original-message", collector.MetricPackets()[0].ContextID)
			require.Len(t, collector.EndPackets(), 1)
			assert.Equal(t, "original-message", collector.EndPackets()[0].ContextID)
		})
	}
}
