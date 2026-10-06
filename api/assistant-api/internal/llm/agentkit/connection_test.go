package internal_llm_agentkit

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"testing"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type connectionTestAgentKitServer struct {
	protos.UnimplementedAgentKitServer
	talkFn func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error
}

func (s *connectionTestAgentKitServer) Talk(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
	if s.talkFn == nil {
		return nil
	}
	return s.talkFn(stream)
}

func startConnectionTestServer(
	t *testing.T,
	talkFn func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error,
) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	protos.RegisterAgentKitServer(server, &connectionTestAgentKitServer{talkFn: talkFn})

	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}

func connectionStringPtr(value string) *string {
	return &value
}

func connectionUint32Ptr(value uint32) *uint32 {
	return &value
}

func TestAgentConnectionOption_UsesDefaultsWhenProviderIsNil(t *testing.T) {
	option := NewAgentConnectionOption(nil)

	assert.Equal(t, time.Duration(DefaultConnectTimeoutMs)*time.Millisecond, option.DialTimeout())
	assert.Equal(t, map[string]interface{}{
		"transportSecurity":          DefaultTransportSecurity,
		"tlsVerification":            DefaultTLSVerification,
		"tlsServerName":              "",
		"caCertificatePemConfigured": false,
		"connectTimeoutMs":           DefaultConnectTimeoutMs,
		"keepaliveTimeMs":            DefaultKeepaliveTimeMs,
		"keepaliveTimeoutMs":         DefaultKeepaliveTimeoutMs,
		"maxRecvMessageBytes":        DefaultMaxRecvMessageBytes,
		"maxSendMessageBytes":        DefaultMaxSendMessageBytes,
	}, option.GetOption())
}

func TestAgentConnectionOption_UsesProviderOverrides(t *testing.T) {
	option := NewAgentConnectionOption(&internal_assistant_entity.AssistantProviderAgentkit{
		Certificate:         "certificate",
		TransportSecurity:   connectionStringPtr(TransportSecurityPlaintext),
		TLSVerification:     connectionStringPtr(TLSVerificationSkipVerify),
		TLSServerName:       connectionStringPtr("agentkit.local"),
		ConnectTimeoutMs:    connectionUint32Ptr(1500),
		KeepaliveTimeMs:     connectionUint32Ptr(20000),
		KeepaliveTimeoutMs:  connectionUint32Ptr(2500),
		MaxRecvMessageBytes: connectionUint32Ptr(4096),
		MaxSendMessageBytes: connectionUint32Ptr(8192),
	})

	assert.Equal(t, 1500*time.Millisecond, option.DialTimeout())
	assert.Equal(t, map[string]interface{}{
		"transportSecurity":          TransportSecurityPlaintext,
		"tlsVerification":            TLSVerificationSkipVerify,
		"tlsServerName":              "agentkit.local",
		"caCertificatePemConfigured": true,
		"connectTimeoutMs":           uint32(1500),
		"keepaliveTimeMs":            uint32(20000),
		"keepaliveTimeoutMs":         uint32(2500),
		"maxRecvMessageBytes":        uint32(4096),
		"maxSendMessageBytes":        uint32(8192),
	}, option.GetOption())
}

func TestAgentConnectionOption_UsesConversationOverrides(t *testing.T) {
	provider := &internal_assistant_entity.AssistantProviderAgentkit{
		Certificate:         "saved-certificate",
		Metadata:            map[string]string{"x-saved": "saved", "x-override": "saved"},
		TransportSecurity:   connectionStringPtr(TransportSecurityTLS),
		TLSVerification:     connectionStringPtr(TLSVerificationVerify),
		TLSServerName:       connectionStringPtr("saved.local"),
		ConnectTimeoutMs:    connectionUint32Ptr(1500),
		KeepaliveTimeMs:     connectionUint32Ptr(20000),
		KeepaliveTimeoutMs:  connectionUint32Ptr(2500),
		MaxRecvMessageBytes: connectionUint32Ptr(4096),
		MaxSendMessageBytes: connectionUint32Ptr(8192),
	}
	resolved, _ := withAgentkitOverrides(provider, utils.Option{
		internal_options.AgentkitOptionTransportSecurity:   TransportSecurityPlaintext,
		internal_options.AgentkitOptionTLSVerification:     TLSVerificationSkipVerify,
		internal_options.AgentkitOptionTLSServerName:       "override.local",
		internal_options.AgentkitOptionCertificate:         "override-certificate",
		internal_options.AgentkitOptionConnectTimeoutMs:    "3000",
		internal_options.AgentkitOptionKeepaliveTimeMs:     float64(40000),
		internal_options.AgentkitOptionKeepaliveTimeoutMs:  uint32(5000),
		internal_options.AgentkitOptionMaxRecvMessageBytes: uint64(16384),
		internal_options.AgentkitOptionMaxSendMessageBytes: uint32(32768),
		internal_options.AgentkitOptionMetadata:            `{"x-override":"conversation","x-extra":"extra"}`,
	})
	option := NewAgentConnectionOption(resolved)

	assert.Equal(t, 3000*time.Millisecond, option.DialTimeout())
	assert.Equal(t, map[string]interface{}{
		"transportSecurity":          TransportSecurityPlaintext,
		"tlsVerification":            TLSVerificationSkipVerify,
		"tlsServerName":              "override.local",
		"caCertificatePemConfigured": true,
		"connectTimeoutMs":           uint32(3000),
		"keepaliveTimeMs":            uint32(40000),
		"keepaliveTimeoutMs":         uint32(5000),
		"maxRecvMessageBytes":        uint32(16384),
		"maxSendMessageBytes":        uint32(32768),
	}, option.GetOption())
	connection := NewAgentkitConnection(resolved)
	assert.Equal(t, map[string]string{
		"x-saved":    "saved",
		"x-override": "conversation",
		"x-extra":    "extra",
	}, connection.metadata)
	assert.Equal(t, "saved-certificate", provider.Certificate)
	assert.Equal(t, "saved.local", *provider.TLSServerName)
	assert.Equal(t, uint32(1500), *provider.ConnectTimeoutMs)
	assert.Equal(t, map[string]string{"x-saved": "saved", "x-override": "saved"}, map[string]string(provider.Metadata))
}

func TestAgentConnectionOption_GetGrpcOptionsReturnsTypedErrors(t *testing.T) {
	invalidTransport := NewAgentConnectionOption(&internal_assistant_entity.AssistantProviderAgentkit{
		TransportSecurity: connectionStringPtr("INVALID"),
	})
	_, err := invalidTransport.GetGrpcOptions()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionInvalidTransportSecurity))

	invalidCertificate := NewAgentConnectionOption(&internal_assistant_entity.AssistantProviderAgentkit{
		TransportSecurity: connectionStringPtr(TransportSecurityTLS),
		Certificate:       "not a pem certificate",
	})
	_, err = invalidCertificate.GetGrpcOptions()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionInvalidCertificate))

	maximumMessageSize := AgentConnectionOption{
		transportSecurity:   TransportSecurityPlaintext,
		maxRecvMessageBytes: math.MaxUint32,
		maxSendMessageBytes: math.MaxUint32,
	}
	_, err = maximumMessageSize.GetGrpcOptions()
	_, conversionErr := utils.Uint32ToInt(math.MaxUint32)
	if conversionErr != nil {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
}

func TestAgentkitConnection_ReturnsNotConnectedWhenStreamIsMissing(t *testing.T) {
	connection := NewAgentkitConnection(&internal_assistant_entity.AssistantProviderAgentkit{})

	err := connection.OpenTalkStream(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionOpenTalkStream))
	assert.True(t, errors.Is(err, ErrAgentkitConnectionNotConnected))

	err = connection.Send(&protos.TalkInput{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionSend))
	assert.True(t, errors.Is(err, ErrAgentkitConnectionNotConnected))

	_, err = connection.Recv()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionRecv))
	assert.True(t, errors.Is(err, ErrAgentkitConnectionNotConnected))
}

func TestAgentkitConnection_OpenTalkStreamRejectsDuplicateStream(t *testing.T) {
	addr, shutdown := startConnectionTestServer(t, func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
		<-stream.Context().Done()
		return nil
	})
	defer shutdown()

	connection := NewAgentkitConnection(&internal_assistant_entity.AssistantProviderAgentkit{
		Url:               addr,
		TransportSecurity: connectionStringPtr(TransportSecurityPlaintext),
	})
	require.NoError(t, connection.Connect())
	defer func() {
		assert.NoError(t, connection.Close())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, connection.OpenTalkStream(ctx))

	err := connection.OpenTalkStream(ctx)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentkitConnectionOpenTalkStream))
	assert.True(t, errors.Is(err, ErrAgentkitConnectionStreamAlreadyOpen))
}

func TestAgentkitConnection_SendRecvAndMetadata(t *testing.T) {
	receivedMetadata := make(chan metadata.MD, 1)
	receivedInput := make(chan *protos.TalkInput, 1)

	addr, shutdown := startConnectionTestServer(t, func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
		md, _ := metadata.FromIncomingContext(stream.Context())
		receivedMetadata <- md

		req, err := stream.Recv()
		if err != nil {
			return err
		}
		receivedInput <- req

		return stream.Send(&protos.TalkOutput{
			Data: &protos.TalkOutput_Initialization{
				Initialization: &protos.ConversationInitialization{
					AssistantConversationId: req.GetInitialization().GetAssistantConversationId(),
				},
			},
		})
	})
	defer shutdown()

	connection := NewAgentkitConnection(&internal_assistant_entity.AssistantProviderAgentkit{
		Url:               addr,
		TransportSecurity: connectionStringPtr(TransportSecurityPlaintext),
		Metadata: map[string]string{
			"x-agentkit-test": "connection",
		},
	})
	require.NoError(t, connection.Connect())
	defer func() {
		assert.NoError(t, connection.Close())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, connection.OpenTalkStream(ctx))
	require.NoError(t, connection.Send(&protos.TalkInput{
		Request: &protos.TalkInput_Initialization{
			Initialization: &protos.ConversationInitialization{
				AssistantConversationId: 9009,
			},
		},
	}))

	response, err := connection.Recv()
	require.NoError(t, err)
	assert.Equal(t, uint64(9009), response.GetInitialization().GetAssistantConversationId())

	select {
	case md := <-receivedMetadata:
		assert.Equal(t, []string{"connection"}, md.Get("x-agentkit-test"))
	case <-time.After(time.Second):
		t.Fatal("server did not receive metadata")
	}

	select {
	case req := <-receivedInput:
		assert.Equal(t, uint64(9009), req.GetInitialization().GetAssistantConversationId())
	case <-time.After(time.Second):
		t.Fatal("server did not receive request")
	}
}

func TestAgentkitConnection_CloseIsIdempotent(t *testing.T) {
	addr, shutdown := startConnectionTestServer(t, func(stream grpc.BidiStreamingServer[protos.TalkInput, protos.TalkOutput]) error {
		for {
			if _, err := stream.Recv(); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
		}
	})
	defer shutdown()

	connection := NewAgentkitConnection(&internal_assistant_entity.AssistantProviderAgentkit{
		Url:               addr,
		TransportSecurity: connectionStringPtr(TransportSecurityPlaintext),
	})
	require.NoError(t, connection.Connect())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, connection.OpenTalkStream(ctx))

	require.NoError(t, connection.Close())
	require.NoError(t, connection.Close())
}
