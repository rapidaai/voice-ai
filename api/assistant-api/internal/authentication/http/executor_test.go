package internal_authentication_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/stretchr/testify/require"
)

type testCallback struct{}

func (testCallback) OnPacket(context.Context, ...internal_type.Packet) error {
	return nil
}

type captureCallback struct {
	packets []internal_type.Packet
}

func (c *captureCallback) OnPacket(_ context.Context, packets ...internal_type.Packet) error {
	c.packets = append(c.packets, packets...)
	return nil
}

func testConfiguration(serverURL string, failBehavior string) *internal_assistant_entity.AssistantConfiguration {
	options := []*internal_assistant_entity.AssistantConfigurationOption{
		{Metadata: gorm_model.Metadata{Key: OptionHTTPURLKey, Value: serverURL}},
		{Metadata: gorm_model.Metadata{Key: OptionHTTPBodyKey, Value: `{"token":"token"}`}},
	}
	if failBehavior != "" {
		options = append(options, &internal_assistant_entity.AssistantConfigurationOption{
			Metadata: gorm_model.Metadata{Key: "fail_behavior", Value: failBehavior},
		})
	}
	return &internal_assistant_entity.AssistantConfiguration{
		Provider: "http",
		Options:  options,
	}
}

func TestExecute_UnauthenticatedBlocksByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
	}))
	defer server.Close()

	executor, err := New(
		WithConfiguration(testConfiguration(server.URL, "")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-block",
	})

	require.Nil(t, output)
	require.EqualError(t, err, "authentication: unauthenticated")
}

func TestExecute_UnauthenticatedReturnsOutputWhenDoNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
	}))
	defer server.Close()

	executor, err := New(
		WithConfiguration(testConfiguration(server.URL, "DO_NOTHING")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-do-nothing",
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	require.False(t, output.Authenticated)
}

func TestExecute_TransportErrorReturnsNilOutput(t *testing.T) {
	executor, err := New(
		WithConfiguration(testConfiguration("http://127.0.0.1:1", "DO_NOTHING")),
		WithCallback(testCallback{}),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-error",
	})

	require.Nil(t, output)
	require.Error(t, err)
}

func TestExecute_RecordsAuthenticationLatencyMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"arguments":{"token":"updated"}}`))
	}))
	defer server.Close()

	callback := &captureCallback{}
	executor, err := New(
		WithConfiguration(testConfiguration(server.URL, "")),
		WithCallback(callback),
	)
	require.NoError(t, err)

	output, err := executor.Execute(context.Background(), internal_type.AuthenticationInput{
		ContextID: "ctx-auth-success",
	})

	require.NoError(t, err)
	require.True(t, output.Authenticated)

	var latencyMetric *internal_type.ObservabilityMetricRecordPacket
	for i := range callback.packets {
		packet, ok := callback.packets[i].(internal_type.ObservabilityMetricRecordPacket)
		if !ok || len(packet.Record.Metrics) == 0 {
			continue
		}
		if packet.Record.Metrics[0].Name == observability.MetricAuthenticationLatencyMs {
			latencyMetric = &packet
			break
		}
	}
	require.NotNil(t, latencyMetric)
	require.Equal(t, "ctx-auth-success", latencyMetric.ContextID)
	require.Equal(t, internal_type.ObservabilityRecordScopeConversation, latencyMetric.Scope)
	require.Equal(t, "http", latencyMetric.Record.Attributes["provider"])
	require.Equal(t, "POST", latencyMetric.Record.Attributes["method"])
	require.Equal(t, "200", latencyMetric.Record.Attributes["response_status"])
	require.Equal(t, "COMPLETE", latencyMetric.Record.Attributes["status"])
	latencyMs, err := strconv.ParseInt(latencyMetric.Record.Metrics[0].Value, 10, 64)
	require.NoError(t, err)
	require.GreaterOrEqual(t, latencyMs, int64(0))
}
