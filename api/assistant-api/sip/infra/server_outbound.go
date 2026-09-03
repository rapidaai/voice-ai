// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import "context"

func (s *Server) MakeCall(ctx context.Context, cfg *Config, toUser, fromUser string, opts MakeCallOptions) (*Session, error) {
	session, err := s.inner.MakeCall(ctx, cfg, toUser, fromUser, opts)
	if err != nil {
		return nil, err
	}
	return wrapSession(session), nil
}

func (s *Server) MakeTransferBridgeCall(ctx context.Context, cfg *Config, toUser, fromUser string, opts TransferBridgeCallOptions) (*Session, error) {
	session, err := s.inner.MakeTransferBridgeCall(ctx, cfg, toUser, fromUser, opts)
	if err != nil {
		return nil, err
	}
	return wrapSession(session), nil
}
