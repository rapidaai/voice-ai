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
	inner, err := internal_core.NewServer(ctx, cfg)
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
	s.inner.SetMiddlewares(middlewares)
}

func (s *Server) IsRunning() bool {
	return s.inner.IsRunning()
}

func (s *Server) NegotiatedSDPConfig(localIP string, rtpPort int, codec *Codec) *SDPConfig {
	return sdpConfigFromCore(s.inner.NegotiatedSDPConfig(localIP, rtpPort, codecToCore(codec)))
}

func (s *Server) GenerateSDP(config *SDPConfig) string {
	return s.inner.GenerateSDP(sdpConfigToCore(config))
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
	return s.inner.GetListenConfig()
}

func (s *Server) SessionCount() int {
	return s.inner.SessionCount()
}

// SetOnApplicationReady sets the legacy application-ready callback.
// Deprecated: use SetOnApplicationReadyIdentity.
func (s *Server) SetOnApplicationReady(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnApplicationReady(nil)
		return
	}
	s.inner.SetOnApplicationReady(func(session *internal_core.Session, _ string, callAddress internal_core.CallAddress) error {
		return fn(wrapSession(session), callAddress.FromURI, callAddress.ToURI)
	})
}

// SetOnApplicationReadyIdentity sets the application-ready callback with explicit SIP identities.
func (s *Server) SetOnApplicationReadyIdentity(fn func(session *Session, identity SIPRequestIdentity) error) {
	if fn == nil {
		s.inner.SetOnApplicationReady(nil)
		return
	}
	s.inner.SetOnApplicationReady(func(session *internal_core.Session, requestURI string, callAddress internal_core.CallAddress) error {
		return fn(wrapSession(session), SIPRequestIdentity{
			RequestURI:   requestURI,
			CallAddress:  callAddress,
			FromIdentity: callAddress.FromURI,
			ToIdentity:   callAddress.ToURI,
		})
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

// SetOnInvite sets the legacy answered-INVITE callback.
// Deprecated: use SetOnInviteIdentity.
func (s *Server) SetOnInvite(fn func(session *Session, fromURI, toURI string) error) {
	if fn == nil {
		s.inner.SetOnInvite(nil)
		return
	}
	s.inner.SetOnInvite(func(session *internal_core.Session, _ string, callAddress internal_core.CallAddress) error {
		return fn(wrapSession(session), callAddress.FromURI, callAddress.ToURI)
	})
}

// SetOnInviteIdentity sets the answered-INVITE callback with explicit SIP identities.
func (s *Server) SetOnInviteIdentity(fn func(session *Session, identity SIPRequestIdentity) error) {
	if fn == nil {
		s.inner.SetOnInvite(nil)
		return
	}
	s.inner.SetOnInvite(func(session *internal_core.Session, requestURI string, callAddress internal_core.CallAddress) error {
		return fn(wrapSession(session), SIPRequestIdentity{
			RequestURI:   requestURI,
			CallAddress:  callAddress,
			FromIdentity: callAddress.FromURI,
			ToIdentity:   callAddress.ToURI,
		})
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
	return s.inner.HealthSnapshot()
}
