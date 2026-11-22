package lifecycle

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

type observedListener struct {
	net.Listener
	accepted chan struct{}
	once     sync.Once
}

type blockingHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	started            chan struct{}
	release            chan struct{}
	canceled           chan struct{}
	ignoreCancellation bool
}

func (server *blockingHealthServer) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	close(server.started)
	if server.ignoreCancellation {
		<-server.release
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	}
	select {
	case <-server.release:
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	case <-ctx.Done():
		close(server.canceled)
		return nil, ctx.Err()
	}
}

func (listener *observedListener) Accept() (net.Conn, error) {
	connection, err := listener.Listener.Accept()
	if err == nil {
		listener.once.Do(func() { close(listener.accepted) })
	}
	return connection, err
}

func TestNewMuxBoundsSilentConnectionShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	observed := &observedListener{Listener: listener, accepted: make(chan struct{})}
	mux := NewMux(observed, 50*time.Millisecond)
	mux.Match(cmux.HTTP1Fast())
	mux.Match(cmux.Any())

	serveDone := make(chan error, 1)
	go func() { serveDone <- mux.Serve() }()

	connection, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer connection.Close()
	<-observed.accepted

	mux.Close()
	if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case <-serveDone:
	case <-time.After(time.Second):
		t.Fatal("Mux Serve did not return after the protocol detection timeout")
	}
}

func TestShutdownWaitsForActiveHTTPRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		response.WriteHeader(http.StatusNoContent)
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_, requestErr = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	<-started

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- Shutdown(ctx, grpc.NewServer(), server)
	}()
	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before the active request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := <-requestDone; err != nil {
		t.Fatalf("GET error = %v", err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want http.ErrServerClosed", err)
	}
}

func TestShutdownWaitsForActiveGRPCWebRequest(t *testing.T) {
	grpcServer := grpc.NewServer()
	healthServer := &blockingHealthServer{
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	httpServer, address, serveDone := startGRPCWebServer(t, grpcServer)
	requestDone := sendGRPCWebHealthRequest(t, address)
	<-healthServer.started

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- Shutdown(ctx, grpcServer, httpServer)
	}()
	select {
	case <-healthServer.canceled:
		t.Fatal("Shutdown canceled the active gRPC-Web request before the grace period expired")
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before the active gRPC-Web request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(healthServer.release)
	if err := <-requestDone; err != nil {
		t.Fatalf("gRPC-Web request error = %v", err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want http.ErrServerClosed", err)
	}
}

func TestShutdownReturnsAtDeadlineWithBlockedGRPCHandler(t *testing.T) {
	grpcServer := grpc.NewServer()
	healthServer := &blockingHealthServer{
		started:            make(chan struct{}),
		release:            make(chan struct{}),
		canceled:           make(chan struct{}),
		ignoreCancellation: true,
	}
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- grpcServer.Serve(listener) }()
	connection, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer connection.Close()
	requestDone := make(chan error, 1)
	go func() {
		_, requestErr := grpc_health_v1.NewHealthClient(connection).Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		requestDone <- requestErr
	}()
	<-healthServer.started

	shutdownDone := make(chan error, 1)
	startedAt := time.Now()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		shutdownDone <- Shutdown(ctx, grpcServer)
	}()
	select {
	case err := <-shutdownDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Shutdown() error = %v, want context deadline exceeded", err)
		}
		if elapsed := time.Since(startedAt); elapsed > 200*time.Millisecond {
			t.Fatalf("Shutdown() elapsed = %v, want no more than 200ms", elapsed)
		}
	case <-time.After(200 * time.Millisecond):
		close(healthServer.release)
		<-shutdownDone
		t.Fatal("Shutdown did not return after its deadline")
	}

	close(healthServer.release)
	<-requestDone
	<-serveDone
}

func TestShutdownForcesCloseAfterDeadline(t *testing.T) {
	started := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	go server.Serve(listener)
	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := Shutdown(ctx, grpc.NewServer(), server); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want context deadline exceeded", err)
	}
	<-requestDone
}

func startGRPCWebServer(t *testing.T, grpcServer *grpc.Server) (*http.Server, string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	server := &http.Server{Handler: grpcweb.WrapServer(grpcServer)}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	return server, listener.Addr().String(), serveDone
}

func sendGRPCWebHealthRequest(t *testing.T, address string) <-chan error {
	t.Helper()
	message, err := proto.Marshal(&grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	frame := make([]byte, 5+len(message))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(message)))
	copy(frame[5:], message)
	request, err := http.NewRequest(http.MethodPost, "http://"+address+"/grpc.health.v1.Health/Check", bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Content-Type", "application/grpc-web+proto")
	request.Header.Set("X-Grpc-Web", "1")

	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr == nil {
			_, requestErr = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	return requestDone
}
