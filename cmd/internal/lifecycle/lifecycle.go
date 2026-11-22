// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// DrainableServer is an HTTP serving resource with bounded shutdown behavior.
type DrainableServer interface {
	Shutdown(context.Context) error
	Close() error
}

// HTTPServer tracks active handlers, including hijacked WebSocket requests.
type HTTPServer struct {
	*http.Server
	requests *requestTracker
}

type requestTracker struct {
	mu       sync.Mutex
	active   map[uint64]*trackedRequest
	nextID   uint64
	draining bool
	forced   bool
	drained  chan struct{}
}

type trackedRequest struct {
	cancel     context.CancelFunc
	connection net.Conn
}

type trackedResponseWriter struct {
	http.ResponseWriter
	tracker   *requestTracker
	requestID uint64
}

// NewHTTPServer creates an HTTP server whose upgraded requests can be drained.
func NewHTTPServer(server *http.Server) *HTTPServer {
	tracker := &requestTracker{active: make(map[uint64]*trackedRequest), drained: make(chan struct{})}
	handler := server.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	server.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestContext, cancelRequest := context.WithCancel(request.Context())
		requestID, started := tracker.start(cancelRequest)
		if !started {
			cancelRequest()
			http.Error(response, "server is shutting down", http.StatusServiceUnavailable)
			return
		}
		defer cancelRequest()
		defer tracker.finish(requestID)
		handler.ServeHTTP(&trackedResponseWriter{ResponseWriter: response, tracker: tracker, requestID: requestID}, request.WithContext(requestContext))
	})
	return &HTTPServer{Server: server, requests: tracker}
}

// Close cancels tracked upgraded requests before closing ordinary connections.
func (server *HTTPServer) Close() error {
	server.requests.cancel()
	return server.Server.Close()
}

// Shutdown drains ordinary and upgraded HTTP requests.
func (server *HTTPServer) Shutdown(ctx context.Context) error {
	drained := server.requests.drain()
	shutdownErr := server.Server.Shutdown(ctx)
	select {
	case <-drained:
		return shutdownErr
	case <-ctx.Done():
		return errors.Join(shutdownErr, ctx.Err())
	}
}

// NewMux creates a connection multiplexer whose protocol detection is bounded.
func NewMux(listener net.Listener, readTimeout time.Duration) cmux.CMux {
	mux := cmux.New(listener)
	mux.SetReadTimeout(readTimeout)
	return mux
}

// Shutdown drains HTTP and gRPC requests until the context expires.
func Shutdown(ctx context.Context, grpcServer *grpc.Server, httpServers ...DrainableServer) error {
	result := make(chan error, len(httpServers))
	for _, server := range httpServers {
		go func(server DrainableServer) {
			result <- server.Shutdown(ctx)
		}(server)
	}

	shutdownErrors := make([]error, 0, len(httpServers)+1)
	for range httpServers {
		select {
		case err := <-result:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				shutdownErrors = append(shutdownErrors, err)
			}
		case <-ctx.Done():
			forceStop(grpcServer, httpServers)
			return errors.Join(append(shutdownErrors, ctx.Err())...)
		}
	}

	if ctx.Err() != nil {
		forceStop(grpcServer, httpServers)
		return errors.Join(append(shutdownErrors, ctx.Err())...)
	}

	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
		return errors.Join(shutdownErrors...)
	case <-ctx.Done():
		forceStop(grpcServer, httpServers)
		return errors.Join(append(shutdownErrors, ctx.Err())...)
	}
}

func forceStop(grpcServer *grpc.Server, httpServers []DrainableServer) {
	for _, server := range httpServers {
		_ = server.Close()
	}
	go grpcServer.Stop()
}

// WaitForServers joins serving goroutines without extending completed shutdown.
func WaitForServers(group *errgroup.Group, shutdownDone <-chan struct{}) error {
	groupDone := make(chan error, 1)
	go func() { groupDone <- group.Wait() }()
	select {
	case err := <-groupDone:
		<-shutdownDone
		return err
	case <-shutdownDone:
		select {
		case err := <-groupDone:
			return err
		default:
			return nil
		}
	}
}

func (tracker *requestTracker) start(cancel context.CancelFunc) (uint64, bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.draining {
		return 0, false
	}
	tracker.nextID++
	tracker.active[tracker.nextID] = &trackedRequest{cancel: cancel}
	return tracker.nextID, true
}

func (tracker *requestTracker) finish(requestID uint64) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	delete(tracker.active, requestID)
	if tracker.draining && len(tracker.active) == 0 {
		close(tracker.drained)
	}
}

func (tracker *requestTracker) drain() <-chan struct{} {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.draining {
		return tracker.drained
	}
	tracker.draining = true
	if len(tracker.active) == 0 {
		close(tracker.drained)
	}
	return tracker.drained
}

func (tracker *requestTracker) cancel() {
	tracker.mu.Lock()
	tracker.draining = true
	tracker.forced = true
	cancellations := make([]context.CancelFunc, 0, len(tracker.active))
	connections := make([]net.Conn, 0, len(tracker.active))
	for _, request := range tracker.active {
		cancellations = append(cancellations, request.cancel)
		if request.connection != nil {
			connections = append(connections, request.connection)
		}
	}
	tracker.mu.Unlock()
	for _, cancel := range cancellations {
		cancel()
	}
	for _, connection := range connections {
		_ = connection.Close()
	}
}

func (writer *trackedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := writer.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	connection, buffer, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	writer.tracker.mu.Lock()
	closeConnection := writer.tracker.forced
	if request := writer.tracker.active[writer.requestID]; request != nil {
		request.connection = connection
	}
	writer.tracker.mu.Unlock()
	if closeConnection {
		_ = connection.Close()
	}
	return connection, buffer, nil
}

func (writer *trackedResponseWriter) Flush() {
	_ = http.NewResponseController(writer.ResponseWriter).Flush()
}

func (writer *trackedResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}
