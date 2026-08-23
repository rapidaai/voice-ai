// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"

	"github.com/emiago/sipgo"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
)

func NewServer(ctx context.Context, cfg *ServerConfig) (*Server, error) {
	inner, err := internal_core.NewServer(ctx, cfg.toCore())
	if err != nil {
		return nil, err
	}
	return &Server{inner: inner}, nil
}

func (s *Server) Start() error {
	return s.inner.Start()
}

func (s *Server) Stop() {
	s.inner.Stop()
}

func (s *Server) SetMiddlewares(middlewares []Middleware) {
	coreMiddlewares := make([]internal_core.Middleware, 0, len(middlewares))
	for _, middleware := range middlewares {
		if middleware == nil {
			continue
		}
		current := middleware
		coreMiddlewares = append(coreMiddlewares, func(ctx *internal_core.SIPRequestContext) error {
			var config *Config
			if ctx.Config != nil {
				converted := configFromCore(ctx.Config)
				config = &converted
			}
			infraCtx := &SIPRequestContext{
				Method:          ctx.Method,
				CallID:          ctx.CallID,
				FromURI:         ctx.FromURI,
				ToURI:           ctx.ToURI,
				SDPInfo:         sdpInfoFromCore(ctx.SDPInfo),
				APIKey:          ctx.APIKey,
				AssistantID:     ctx.AssistantID,
				Auth:            ctx.Auth,
				Assistant:       ctx.Assistant,
				VaultCredential: ctx.VaultCredential,
				Config:          config,
			}
			err := current(infraCtx)
			ctx.APIKey = infraCtx.APIKey
			ctx.AssistantID = infraCtx.AssistantID
			ctx.Auth = infraCtx.Auth
			ctx.Assistant = infraCtx.Assistant
			ctx.VaultCredential = infraCtx.VaultCredential
			if infraCtx.Config != nil {
				ctx.Config = infraCtx.Config.toCore()
			} else {
				ctx.Config = nil
			}
			return err
		})
	}
	s.inner.SetMiddlewares(coreMiddlewares)
}

func (s *Server) IsRunning() bool {
	return s.inner.IsRunning()
}

func (s *Server) NegotiatedSDPConfig(localIP string, rtpPort int, codec *Codec) *SDPConfig {
	return sdpConfigFromCore(s.inner.NegotiatedSDPConfig(localIP, rtpPort, codec.toCore()))
}

func (s *Server) GenerateSDP(config *SDPConfig) string {
	return s.inner.GenerateSDP(config.toCore())
}

func (s *Server) ParseSDP(sdpBody []byte) (*SDPMediaInfo, error) {
	info, err := s.inner.ParseSDP(sdpBody)
	if err != nil {
		return nil, err
	}
	return sdpInfoFromCore(info), nil
}

func (s *Server) Client() *sipgo.Client {
	return s.inner.Client()
}

func (s *Server) GetListenConfig() *ListenConfig {
	return listenConfigFromCore(s.inner.GetListenConfig())
}

func (s *Server) SessionCount() int {
	return s.inner.SessionCount()
}

func (s *Server) SetOnApplicationReady(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnApplicationReady(nil)
		return
	}
	s.inner.SetOnApplicationReady(func(session *internal_core.Session, fromURI, toURI string) error {
		return fn(wrapSession(session), fromURI, toURI)
	})
}

func (s *Server) SetOnApplicationCleanup(fn func(session *Session)) {
	if fn == nil {
		s.inner.SetOnApplicationCleanup(nil)
		return
	}
	s.inner.SetOnApplicationCleanup(func(session *internal_core.Session) {
		fn(wrapSession(session))
	})
}

func (s *Server) SetOnInvite(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnInvite(nil)
		return
	}
	s.inner.SetOnInvite(func(session *internal_core.Session, fromURI, toURI string) error {
		return fn(wrapSession(session), fromURI, toURI)
	})
}

func (s *Server) SetOnBye(fn func(session *Session) error) {
	if fn == nil {
		s.inner.SetOnBye(nil)
		return
	}
	s.inner.SetOnBye(func(session *internal_core.Session) error {
		return fn(wrapSession(session))
	})
}

func (s *Server) SetOnCancel(fn func(session *Session) error) {
	if fn == nil {
		s.inner.SetOnCancel(nil)
		return
	}
	s.inner.SetOnCancel(func(session *internal_core.Session) error {
		return fn(wrapSession(session))
	})
}

func (s *Server) SetOnError(fn func(session *Session, err error)) {
	if fn == nil {
		s.inner.SetOnError(nil)
		return
	}
	s.inner.SetOnError(func(session *internal_core.Session, err error) {
		fn(wrapSession(session), err)
	})
}

func (s *Server) HealthSnapshot() ServerHealthSnapshot {
	return serverHealthSnapshotFromCore(s.inner.HealthSnapshot())
}
