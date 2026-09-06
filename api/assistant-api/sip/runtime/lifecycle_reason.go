// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

type LifecycleReason string

func (r LifecycleReason) String() string {
	return string(r)
}

type LifecycleController interface {
	TransitionCall(session *Session, next CallState, reason LifecycleReason) bool
	EndCallWithReason(session *Session, reason LifecycleReason) error
	FailCall(session *Session, reason LifecycleReason, err error) error
	CancelCall(session *Session, reason LifecycleReason) error
}
