// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import "errors"

const (
	Provider     = "asterisk"
	WebhookEvent = "webhook"
)

type OutboundFailureReason string

const (
	OutboundFailureReasonRequestCancelled       OutboundFailureReason = "asterisk_outbound_request_cancelled"
	OutboundFailureReasonMissingVaultCredential OutboundFailureReason = "asterisk_outbound_missing_vault_credential"
	OutboundFailureReasonARIURLMissing          OutboundFailureReason = "asterisk_outbound_ari_url_missing"
	OutboundFailureReasonRequestPayloadFailed   OutboundFailureReason = "asterisk_outbound_request_payload_failed"
	OutboundFailureReasonRequestCreateFailed    OutboundFailureReason = "asterisk_outbound_request_create_failed"
	OutboundFailureReasonProviderAPIError       OutboundFailureReason = "asterisk_outbound_provider_api_error"
	OutboundFailureReasonHTTPStatusFailed       OutboundFailureReason = "asterisk_outbound_provider_http_status_failed"
)

func (r OutboundFailureReason) String() string {
	return string(r)
}

var (
	ErrRequestBodyReadFailed   = errors.New("failed to read request body")
	ErrRequestBodyParseFailed  = errors.New("failed to parse request body")
	ErrVaultCredentialMissing  = errors.New("missing vault credential for Asterisk ARI")
	ErrVaultARIURLMissing      = errors.New("missing ari_url in vault credential")
	ErrInboundCallerMissing    = errors.New("missing caller information in query params")
	ErrProviderARIStatusFailed = errors.New("ARI returned failed status")
)

type AsteriskMediaEvent struct {
	Event            string `json:"event,omitempty"`
	Command          string `json:"command,omitempty"`
	Channel          string `json:"channel,omitempty"`
	OptimalFrameSize int    `json:"optimal_frame_size,omitempty"`
	CorrelationID    string `json:"correlation_id,omitempty"`
	RawMessage       string `json:"-"`
}

type AsteriskARIEvent struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	Channel   *AsteriskChannel       `json:"channel,omitempty"`
	Bridge    *AsteriskBridge        `json:"bridge,omitempty"`
	Peer      *AsteriskChannel       `json:"peer,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

type AsteriskChannel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	State       string            `json:"state"`
	Caller      *AsteriskEndpoint `json:"caller,omitempty"`
	Connected   *AsteriskEndpoint `json:"connected,omitempty"`
	Dialplan    *AsteriskDialplan `json:"dialplan,omitempty"`
	ChannelVars map[string]string `json:"channelvars,omitempty"`
}

type AsteriskEndpoint struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

type AsteriskDialplan struct {
	Context string `json:"context"`
	Exten   string `json:"exten"`
	AppData string `json:"app_data"`
}

type AsteriskBridge struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	BridgeType string   `json:"bridge_type"`
	Channels   []string `json:"channels"`
}

type AsteriskRESTRequest struct {
	Type         string              `json:"type"`
	RequestID    string              `json:"request_id"`
	Method       string              `json:"method"`
	URI          string              `json:"uri"`
	QueryStrings []map[string]string `json:"query_strings,omitempty"`
}

type AsteriskRESTResponse struct {
	Type         string `json:"type"`
	RequestID    string `json:"request_id"`
	StatusCode   int    `json:"status_code"`
	ReasonPhrase string `json:"reason_phrase"`
	MessageBody  string `json:"message_body,omitempty"`
}
