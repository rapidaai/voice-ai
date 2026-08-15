// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// Conversation metadata names mirror the current metadata keys.
const (
	MetadataClientPhone          = "client.phone"
	MetadataClientAssistantPhone = "client.assistant_phone"
	MetadataClientDirection      = "client.direction"
	MetadataClientChannel        = "client.channel"
	MetadataClientProviderCallID = "client.provider_call_id"
	MetadataClientContextID      = "client.context_id"
	MetadataClientCodec          = "client.codec"
	MetadataClientSampleRate     = "client.sample_rate"

	MetadataClientTimezone       = "client.timezone"
	MetadataClientPlatform       = "client.platform"
	MetadataClientLanguage       = "client.language"
	MetadataClientUserAgent      = "client.user_agent"
	MetadataClientReferrer       = "client.referrer"
	MetadataClientConnectionType = "client.connection_type"
	MetadataClientLatitude       = "client.latitude"
	MetadataClientLongitude      = "client.longitude"
)

// Current metadata constant names kept as aliases for the existing observe names.
const (
	ClientPhone          = MetadataClientPhone
	ClientAssistantPhone = MetadataClientAssistantPhone
	ClientDirection      = MetadataClientDirection
	ClientChannel        = MetadataClientChannel
	ClientProviderCallID = MetadataClientProviderCallID
	ClientContextID      = MetadataClientContextID
	ClientCodec          = MetadataClientCodec
	ClientSampleRate     = MetadataClientSampleRate
)

// Current conversation, call-context, and SIP lifecycle metadata names.
const (
	MetadataLanguage     = "language"
	MetadataLanguageCode = "language_code"

	MetadataDisconnectReason    = "disconnect_reason"
	MetadataDisconnectRawReason = "disconnect_raw_reason"

	MetadataCallError          = "call_error"
	MetadataFailureClass       = "failure_class"
	MetadataFailureReason      = "failure_reason"
	MetadataSLIResult          = "sli_result"
	MetadataSLIReason          = "sli_reason"
	MetadataRetryable          = "retryable"
	MetadataProviderStatusCode = "provider_status_code"
	MetadataTelephonyError     = "telephony.error"
	MetadataSTTRequestID       = "stt.request_id"
)

// Current SIP transfer metadata names.
const (
	MetadataBridgeTransferTarget         = "bridge_transfer_target"
	MetadataBridgeTransferStatus         = "bridge_transfer_status"
	MetadataBridgeTransferDuration       = "bridge_transfer_duration"
	MetadataBridgeTransferOutboundCallID = "bridge_transfer_outbound_call_id"
)

// ClientMetadata returns standardized client metadata for a conversation.
func ClientMetadata(phone, assistantPhone, direction, provider, providerCallID, contextID, codec, sampleRate string) []*protos.Metadata {
	metadata := make([]*protos.Metadata, 0, 8)
	metadata = appendMetadata(metadata, MetadataClientDirection, direction)
	metadata = appendMetadata(metadata, MetadataClientChannel, provider)
	metadata = appendMetadata(metadata, MetadataClientPhone, phone)
	metadata = appendMetadata(metadata, MetadataClientAssistantPhone, assistantPhone)
	metadata = appendMetadata(metadata, MetadataClientProviderCallID, providerCallID)
	metadata = appendMetadata(metadata, MetadataClientContextID, contextID)
	metadata = appendMetadata(metadata, MetadataClientCodec, codec)
	metadata = appendMetadata(metadata, MetadataClientSampleRate, sampleRate)
	return metadata
}

// DisconnectMetadata returns standardized terminal disconnect metadata.
func DisconnectMetadata(reason, rawReason string) []*protos.Metadata {
	metadata := make([]*protos.Metadata, 0, 2)
	metadata = appendMetadata(metadata, MetadataDisconnectReason, reason)
	metadata = appendMetadata(metadata, MetadataDisconnectRawReason, rawReason)
	return metadata
}

// CallStatusMetric returns the canonical call status metric shape.
func CallStatusMetric(status, reason string) []*protos.Metric {
	return []*protos.Metric{{
		Name:        MetricCallStatus,
		Value:       status,
		Description: reason,
	}}
}

func appendMetadata(metadata []*protos.Metadata, key, value string) []*protos.Metadata {
	if validator.NotBlank(value) {
		return append(metadata, &protos.Metadata{Key: key, Value: value})
	}
	return metadata
}
