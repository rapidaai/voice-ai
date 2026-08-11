// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip

import (
	"strconv"
	"strings"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

type StatusCallback struct {
	Event       internal_type.TelephonyEvent
	Status      string
	ChannelUUID string
	Duration    *time.Duration
	Price       string
	Reason      string
	ErrorCode   string
	RawPayload  string
	Payload     utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	status, err := eventDetails.GetString("event")
	if err != nil || status == "" {
		status, _ = eventDetails.GetString("status")
	}
	if status == "" {
		status, _ = eventDetails.GetString("state")
	}

	callback := &StatusCallback{
		Event:      StatusEvent(status),
		Status:     status,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}
	if channelUUID, err := eventDetails.GetString("call_id"); err == nil {
		callback.ChannelUUID = channelUUID
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("callId"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("call-id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("Call-ID"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("channel_uuid"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}

	duration, err := eventDetails.GetDuration("duration")
	if err != nil {
		duration, err = eventDetails.GetDuration("call_duration")
	}
	if err != nil {
		duration, err = eventDetails.GetDuration("CallDuration")
	}
	if err == nil {
		callback.Duration = utils.Ptr(duration)
	}
	if callback.Duration == nil {
		switch value := eventDetails["duration_ms"].(type) {
		case string:
			if ms, parseErr := strconv.ParseFloat(strings.TrimSpace(value), 64); parseErr == nil {
				duration := time.Duration(ms * float64(time.Millisecond))
				callback.Duration = utils.Ptr(duration)
			}
		case float64:
			duration := time.Duration(value * float64(time.Millisecond))
			callback.Duration = utils.Ptr(duration)
		case int:
			duration := time.Duration(value) * time.Millisecond
			callback.Duration = utils.Ptr(duration)
		case int64:
			duration := time.Duration(value) * time.Millisecond
			callback.Duration = utils.Ptr(duration)
		}
	}

	if price, err := eventDetails.GetString("price"); err == nil {
		callback.Price = price
	}
	if callback.Price == "" {
		if price, err := eventDetails.GetString("cost"); err == nil {
			callback.Price = price
		}
	}
	if reason, err := eventDetails.GetString("reason"); err == nil {
		callback.Reason = reason
	}
	if callback.Reason == "" {
		if reason, err := eventDetails.GetString("disconnect_reason"); err == nil {
			callback.Reason = reason
		}
	}
	if callback.Reason == "" {
		if reason, err := eventDetails.GetString("failure_reason"); err == nil {
			callback.Reason = reason
		}
	}
	if callback.Reason == "" {
		if reason, err := eventDetails.GetString("error_message"); err == nil {
			callback.Reason = reason
		}
	}
	if callback.Reason == "" {
		if reason, err := eventDetails.GetString("error"); err == nil {
			callback.Reason = reason
		}
	}
	if callback.Reason == "" {
		if reason, err := eventDetails.GetString("sip_code"); err == nil {
			callback.Reason = reason
		}
	}
	if errorCode, err := eventDetails.GetString("error_code"); err == nil {
		callback.ErrorCode = errorCode
	}
	return callback, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	info := &internal_type.StatusInfo{
		Event:       s.Event,
		ChannelUUID: s.ChannelUUID,
		Completed:   s.IsCompleted(),
		Duration:    s.Duration,
		Price:       s.Price,
		RawPayload:  s.RawPayload,
		Payload:     s.Payload,
	}
	if statusError := s.StatusError(); statusError != nil {
		info.Error = statusError
	}
	return info
}

func (s *StatusCallback) IsCompleted() bool {
	switch s.Status {
	case "completed", "complete", "ended", "end", "hangup", "bye", "terminated":
		return !s.Failed()
	}
	return false
}

func (s *StatusCallback) Failed() bool {
	switch s.Status {
	case "failed", "failure", "busy", "no-answer", "no_answer", "unanswered", "rejected", "timeout", "error":
		return true
	}
	return s.Reason != "" && s.ErrorCode != ""
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}
	if s.Reason != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Reason}
	}
	if s.ErrorCode != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.ErrorCode}
	}
	if s.Status != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Status}
	}
	return &internal_type.StatusError{Error: "failed", Reason: s.Event.String()}
}

func StatusEvent(status string) internal_type.TelephonyEvent {
	switch status {
	case "initiated", "ringing":
		return internal_type.TelephonyEventRinging
	case "answered":
		return internal_type.TelephonyEventAnswered
	case "completed", "complete", "ended", "end", "hangup", "bye", "terminated",
		"failed", "failure", "busy", "no-answer", "no_answer", "unanswered", "rejected", "timeout", "error":
		return internal_type.TelephonyEventCompleted
	default:
		return internal_type.TelephonyEvent(status)
	}
}
