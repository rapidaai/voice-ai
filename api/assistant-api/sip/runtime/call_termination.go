// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

type CallTerminationResult string

const (
	CallTerminationSuccess     CallTerminationResult = "success"
	CallTerminationClientError CallTerminationResult = "client_error"
	CallTerminationServerError CallTerminationResult = "server_error"
)

type CallTermination struct {
	Result CallTerminationResult
	Reason string
}
