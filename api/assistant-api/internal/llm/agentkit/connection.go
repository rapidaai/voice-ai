// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentkit

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

type AgentConnectionOption struct {
	transportSecurity   string
	tlsVerification     string
	tlsServerName       string
	certificate         string
	connectTimeoutMs    uint32
	keepaliveTimeMs     uint32
	keepaliveTimeoutMs  uint32
	maxRecvMessageBytes uint32
	maxSendMessageBytes uint32
}

func NewAgentConnectionOption(provider *internal_assistant_entity.AssistantProviderAgentkit) AgentConnectionOption {
	option := AgentConnectionOption{
		transportSecurity:   DefaultTransportSecurity,
		tlsVerification:     DefaultTLSVerification,
		connectTimeoutMs:    DefaultConnectTimeoutMs,
		keepaliveTimeMs:     DefaultKeepaliveTimeMs,
		keepaliveTimeoutMs:  DefaultKeepaliveTimeoutMs,
		maxRecvMessageBytes: DefaultMaxRecvMessageBytes,
		maxSendMessageBytes: DefaultMaxSendMessageBytes,
	}
	if provider == nil {
		return option
	}
	if provider.TransportSecurity != nil && *provider.TransportSecurity != "" {
		option.transportSecurity = *provider.TransportSecurity
	}
	if provider.TLSVerification != nil && *provider.TLSVerification != "" {
		option.tlsVerification = *provider.TLSVerification
	}
	if provider.TLSServerName != nil {
		option.tlsServerName = *provider.TLSServerName
	}
	if provider.Certificate != "" {
		option.certificate = provider.Certificate
	}
	if provider.ConnectTimeoutMs != nil {
		option.connectTimeoutMs = *provider.ConnectTimeoutMs
	}
	if provider.KeepaliveTimeMs != nil {
		option.keepaliveTimeMs = *provider.KeepaliveTimeMs
	}
	if provider.KeepaliveTimeoutMs != nil {
		option.keepaliveTimeoutMs = *provider.KeepaliveTimeoutMs
	}
	if provider.MaxRecvMessageBytes != nil {
		option.maxRecvMessageBytes = *provider.MaxRecvMessageBytes
	}
	if provider.MaxSendMessageBytes != nil {
		option.maxSendMessageBytes = *provider.MaxSendMessageBytes
	}
	return option
}

func (o AgentConnectionOption) GetOption() map[string]interface{} {
	return map[string]interface{}{
		"transportSecurity":          o.transportSecurity,
		"tlsVerification":            o.tlsVerification,
		"tlsServerName":              o.tlsServerName,
		"caCertificatePemConfigured": o.certificate != "",
		"connectTimeoutMs":           o.connectTimeoutMs,
		"keepaliveTimeMs":            o.keepaliveTimeMs,
		"keepaliveTimeoutMs":         o.keepaliveTimeoutMs,
		"maxRecvMessageBytes":        o.maxRecvMessageBytes,
		"maxSendMessageBytes":        o.maxSendMessageBytes,
	}
}

func (o AgentConnectionOption) DialTimeout() time.Duration {
	return time.Duration(o.connectTimeoutMs) * time.Millisecond
}

func (o AgentConnectionOption) GetGrpcOptions() ([]grpc.DialOption, error) {
	maxRecvMessageBytes, err := utils.Uint32ToInt(o.maxRecvMessageBytes)
	if err != nil {
		return nil, fmt.Errorf("agentkit max receive message bytes: %w", err)
	}
	maxSendMessageBytes, err := utils.Uint32ToInt(o.maxSendMessageBytes)
	if err != nil {
		return nil, fmt.Errorf("agentkit max send message bytes: %w", err)
	}
	dialTimeout := o.DialTimeout()
	options := []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: dialTimeout}
			return dialer.DialContext(ctx, "tcp", addr)
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMessageBytes),
			grpc.MaxCallSendMsgSize(maxSendMessageBytes),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Duration(o.keepaliveTimeMs) * time.Millisecond,
			Timeout: time.Duration(o.keepaliveTimeoutMs) * time.Millisecond,
		}),
	}

	switch o.transportSecurity {
	case TransportSecurityPlaintext:
		return append(options, grpc.WithTransportCredentials(insecure.NewCredentials())), nil
	case TransportSecurityTLS:
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if o.tlsServerName != "" {
			tlsConfig.ServerName = o.tlsServerName
		}
		if o.tlsVerification == TLSVerificationSkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}
		if o.certificate != "" {
			certificatePool := x509.NewCertPool()
			if !certificatePool.AppendCertsFromPEM([]byte(o.certificate)) {
				return nil, fmt.Errorf("%w: failed to parse PEM", ErrAgentkitConnectionInvalidCertificate)
			}
			tlsConfig.RootCAs = certificatePool
		}
		return append(options, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrAgentkitConnectionInvalidTransportSecurity, o.transportSecurity)
	}
}

type AgentkitConnection struct {
	url      string
	metadata map[string]string
	option   AgentConnectionOption

	mu     sync.RWMutex
	sendMu sync.Mutex

	conn   *grpc.ClientConn
	stream grpc.BidiStreamingClient[protos.TalkInput, protos.TalkOutput]
}

func NewAgentkitConnection(provider *internal_assistant_entity.AssistantProviderAgentkit) *AgentkitConnection {
	connection := &AgentkitConnection{
		option:   NewAgentConnectionOption(provider),
		metadata: make(map[string]string),
	}
	if provider == nil {
		return connection
	}
	connection.url = provider.Url
	for key, value := range provider.Metadata {
		connection.metadata[key] = value
	}
	return connection
}

func (c *AgentkitConnection) Connect() error {
	options, err := c.option.GetGrpcOptions()
	if err != nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionDialOptions, c.url, err)
	}
	conn, err := grpc.NewClient(c.url, options...)
	if err != nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionConnect, c.url, err)
	}

	// Do not wait for Ready here; Talk initialization is the lifecycle's first
	// connectivity gate and will surface dial/handshake failures.
	conn.Connect()
	if err := c.Close(); err != nil {
		_ = conn.Close()
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.stream = nil
	c.mu.Unlock()
	return nil
}

func (c *AgentkitConnection) OpenTalkStream(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	existingStream := c.stream
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionOpenTalkStream, c.url, ErrAgentkitConnectionNotConnected)
	}
	if existingStream != nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionOpenTalkStream, c.url, ErrAgentkitConnectionStreamAlreadyOpen)
	}

	streamCtx := ctx
	if len(c.metadata) > 0 {
		streamCtx = metadata.NewOutgoingContext(ctx, metadata.New(c.metadata))
	}
	stream, err := protos.NewAgentKitClient(conn).Talk(streamCtx)
	if err != nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionOpenTalkStream, c.url, err)
	}

	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		_ = stream.CloseSend()
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionOpenTalkStream, c.url, ErrAgentkitConnectionChanged)
	}
	if c.stream != nil {
		c.mu.Unlock()
		_ = stream.CloseSend()
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionOpenTalkStream, c.url, ErrAgentkitConnectionStreamAlreadyOpen)
	}
	c.stream = stream
	c.mu.Unlock()
	return nil
}

func (c *AgentkitConnection) Send(req *protos.TalkInput) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.RLock()
	stream := c.stream
	c.mu.RUnlock()
	if stream == nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionSend, c.url, ErrAgentkitConnectionNotConnected)
	}
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionSend, c.url, err)
	}
	return nil
}

func (c *AgentkitConnection) Recv() (*protos.TalkOutput, error) {
	c.mu.RLock()
	stream := c.stream
	c.mu.RUnlock()
	if stream == nil {
		return nil, fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionRecv, c.url, ErrAgentkitConnectionNotConnected)
	}
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionRecv, c.url, err)
	}
	return response, nil
}

func (c *AgentkitConnection) Close() error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.Lock()
	stream := c.stream
	conn := c.conn
	c.stream = nil
	c.conn = nil
	c.mu.Unlock()

	var closeErrors []error
	if stream != nil {
		if err := stream.CloseSend(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionCloseStream, c.url, err))
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("%w for %s: %w", ErrAgentkitConnectionCloseConn, c.url, err))
		}
	}
	return errors.Join(closeErrors...)
}

func (c *AgentkitConnection) GetOption() map[string]interface{} {
	return c.option.GetOption()
}

func (c *AgentkitConnection) DialTimeout() time.Duration {
	return c.option.DialTimeout()
}
