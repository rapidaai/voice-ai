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
	"sync"
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
	var waitGroup sync.WaitGroup
	result := make(chan error, len(httpServers)+1)

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		grpcServer.GracefulStop()
		result <- nil
	}()

	for _, server := range httpServers {
		waitGroup.Add(1)
		go func(server *http.Server) {
			defer waitGroup.Done()
			result <- server.Shutdown(ctx)
		}(server)
	}

	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		grpcServer.Stop()
		for _, server := range httpServers {
			_ = server.Close()
		}
		<-done
	}
	close(result)

	shutdownErrors := make([]error, 0, len(httpServers)+1)
	for err := range result {
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	return errors.Join(shutdownErrors...)
}
