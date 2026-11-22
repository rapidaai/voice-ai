// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

// NewMux creates a connection multiplexer whose protocol detection is bounded.
func NewMux(listener net.Listener, readTimeout time.Duration) cmux.CMux {
	mux := cmux.New(listener)
	mux.SetReadTimeout(readTimeout)
	return mux
}

// Shutdown drains HTTP and gRPC requests until the context expires.
func Shutdown(ctx context.Context, grpcServer *grpc.Server, httpServers ...*http.Server) error {
	result := make(chan error, len(httpServers))
	for _, server := range httpServers {
		go func(server *http.Server) {
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

func forceStop(grpcServer *grpc.Server, httpServers []*http.Server) {
	for _, server := range httpServers {
		_ = server.Close()
	}
	go grpcServer.Stop()
}
