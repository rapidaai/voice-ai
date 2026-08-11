// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_exotel

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

type StatusCallback struct {
	Event        internal_type.TelephonyEvent
	Status       string
	ChannelUUID  string
	Duration     *time.Duration
	Price        string
	Cause        string
	ErrorCode    string
	ErrorMessage string
	RawPayload   string
	Payload      utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	status, err := eventDetails.GetString("Status")
	if err != nil {
		return nil, ErrStatusCallbackStatusMissing
	}

	callback := &StatusCallback{
		Event:      StatusEvent(status),
		Status:     status,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}
	duration, err := eventDetails.GetDuration("ConversationDuration")
	if err == nil {
		callback.Duration = utils.Ptr(duration)
	}
	if channelUUID, err := eventDetails.GetString("CallSid"); err == nil {
		callback.ChannelUUID = channelUUID
	}
	if price, err := eventDetails.GetString("Price"); err == nil {
		callback.Price = price
	}
	if cause, err := eventDetails.GetString("Cause"); err == nil {
		callback.Cause = cause
	}
	if errorCode, err := eventDetails.GetString("ErrorCode"); err == nil {
		callback.ErrorCode = errorCode
	}
	if errorMessage, err := eventDetails.GetString("ErrorMessage"); err == nil {
		callback.ErrorMessage = errorMessage
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
	return s.Status == "completed" && !s.Failed()
}

func (s *StatusCallback) Failed() bool {
	switch s.Status {
	case "failed", "busy", "no-answer", "canceled":
		return true
	}
	return s.Cause != "" || s.ErrorMessage != "" || s.ErrorCode != ""
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}

	if s.Cause != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Cause}
	}
	if s.ErrorMessage != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.ErrorMessage}
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
	case "queued", "ringing":
		return internal_type.TelephonyEventRinging
	case "in-progress":
		return internal_type.TelephonyEventAnswered
	case "completed", "no-answer", "busy", "failed", "canceled":
		return internal_type.TelephonyEventCompleted
	default:
		return internal_type.TelephonyEvent(status)
	}
}
