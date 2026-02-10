// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telnyx

type TelnyxMediaEvent struct {
	Event     string `json:"event"`
	StreamSid string `json:"stream_sid"`
	Media     *TelnyxMedia `json:"media,omitempty"`
}

type TelnyxMedia struct {
	Payload string `json:"payload"`
}

type TelnyxCallResponse struct {
	CallID     string `json:"call_id"`
	Status     string `json:"status"`
	ResultCode int    `json:"result_code"`
	Message    string `json:"message"`
}