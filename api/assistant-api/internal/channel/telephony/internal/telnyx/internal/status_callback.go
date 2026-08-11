// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

type StatusCallback struct {
	Event        internal_type.TelephonyEvent
	EventType    string
	ChannelUUID  string
	Duration     *time.Duration
	Price        string
	Reason       string
	ErrorCode    string
	ErrorMessage string
	RawPayload   string
	Payload      utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	rawData, ok := eventDetails["data"].(map[string]interface{})
	if !ok {
		if data, ok := eventDetails["data"].(utils.Option); ok {
			rawData = data
			ok = true
		}
	}
	if !ok {
		return nil, ErrStatusCallbackDataMissing
	}
	data := utils.Option(rawData)

	eventType, err := data.GetString("event_type")
	if err != nil || eventType == "" {
		return nil, ErrStatusCallbackEventTypeMissing
	}

	payloadData := utils.Option{}
	if rawPayloadData, ok := data["payload"].(map[string]interface{}); ok {
		payloadData = utils.Option(rawPayloadData)
	} else if rawPayloadData, ok := data["payload"].(utils.Option); ok {
		payloadData = rawPayloadData
	}

	callback := &StatusCallback{
		Event:      StatusEvent(eventType),
		EventType:  eventType,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}

	if channelUUID, err := payloadData.GetString("call_control_id"); err == nil {
		callback.ChannelUUID = channelUUID
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := payloadData.GetString("call_session_id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := data.GetString("call_control_id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := data.GetString("id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}

	duration, err := payloadData.GetDuration("duration")
	if err != nil {
		duration, err = payloadData.GetDuration("duration_secs")
	}
	if err != nil {
		duration, err = payloadData.GetDuration("call_duration")
	}
	if err == nil {
		callback.Duration = utils.Ptr(duration)
	}

	if price, err := payloadData.GetString("price"); err == nil {
		callback.Price = price
	}
	if callback.Price == "" {
		if price, err := payloadData.GetString("cost"); err == nil {
			callback.Price = price
		}
	}

	if reason, err := payloadData.GetString("hangup_cause"); err == nil {
		callback.Reason = reason
	}
	if callback.Reason == "" {
		if reason, err := payloadData.GetString("cause"); err == nil {
			callback.Reason = reason
		}
	}
	if callback.Reason == "" {
		if reason, err := payloadData.GetString("sip_hangup_cause"); err == nil {
			callback.Reason = reason
		}
	}
	if errorCode, err := payloadData.GetString("error_code"); err == nil {
		callback.ErrorCode = errorCode
	}
	if errorMessage, err := payloadData.GetString("error_message"); err == nil {
		callback.ErrorMessage = errorMessage
	}
	return callback, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	statusInfo := &internal_type.StatusInfo{
		Event:       s.Event,
		ChannelUUID: s.ChannelUUID,
		Completed:   s.IsCompleted(),
		Duration:    s.Duration,
		Price:       s.Price,
		RawPayload:  s.RawPayload,
		Payload:     s.Payload,
	}
	if statusError := s.StatusError(); statusError != nil {
		statusInfo.Error = statusError
	}
	return statusInfo
}

func (s *StatusCallback) IsCompleted() bool {
	return s.EventType == "call.hangup" && !s.Failed()
}

func (s *StatusCallback) Failed() bool {
	switch s.EventType {
	case "call.failed", "call.rejected", "call.bridging.failed":
		return true
	}
	if s.ErrorCode != "" || s.ErrorMessage != "" {
		return true
	}
	if s.EventType == "call.hangup" {
		switch s.Reason {
		case "busy", "no_answer", "no-answer", "rejected", "failed", "timeout":
			return true
		}
	}
	return false
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}
	if s.Reason != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Reason}
	}
	if s.ErrorMessage != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.ErrorMessage}
	}
	if s.ErrorCode != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.ErrorCode}
	}
	if s.EventType != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.EventType}
	}
	return &internal_type.StatusError{Error: "failed", Reason: s.Event.String()}
}

func StatusEvent(eventType string) internal_type.TelephonyEvent {
	switch eventType {
	case "call.initiated", "call.ringing", "initiated", "queued", "ringing":
		return internal_type.TelephonyEventRinging
	case "call.answered", "answered":
		return internal_type.TelephonyEventAnswered
	case "call.hangup", "call.failed", "call.rejected", "call.bridging.failed", "completed", "failed", "busy", "no-answer", "canceled":
		return internal_type.TelephonyEventCompleted
	default:
		return internal_type.TelephonyEvent(eventType)
	}
}
