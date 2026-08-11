// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

// StatusCallback parses a vobiz call/stream webhook (ring_url/hangup_url and
// Stream statusCallbackUrl events: Ring, StartApp, Hangup, StartStream,
// StopStream).
type StatusCallback struct {
	Event       internal_type.TelephonyEvent
	RawEvent    string
	CallStatus  string
	ChannelUUID string
	Duration    *time.Duration
	RawPayload  string
	Payload     utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	event, eventErr := eventDetails.GetString("Event")
	callStatus, statusErr := eventDetails.GetString("CallStatus")
	if eventErr != nil && statusErr != nil {
		return nil, ErrStatusCallbackStatusMissing
	}

	callback := &StatusCallback{
		Event:      StatusEvent(event, callStatus),
		RawEvent:   event,
		CallStatus: callStatus,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}
	duration, err := eventDetails.GetDuration("Duration")
	if err == nil {
		callback.Duration = utils.Ptr(duration)
	}
	if channelUUID, err := eventDetails.GetString("CallUUID"); err == nil {
		callback.ChannelUUID = channelUUID
	}
	if callback.ChannelUUID == "" {
		if requestUUID, err := eventDetails.GetString("RequestUUID"); err == nil {
			callback.ChannelUUID = requestUUID
		}
	}
	return callback, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	statusInfo := &internal_type.StatusInfo{
		Event:       s.Event,
		ChannelUUID: s.ChannelUUID,
		Completed:   s.IsCompleted(),
		Duration:    s.Duration,
		RawPayload:  s.RawPayload,
		Payload:     s.Payload,
	}
	if statusError := s.StatusError(); statusError != nil {
		statusInfo.Error = statusError
	}
	return statusInfo
}

func (s *StatusCallback) IsCompleted() bool {
	return (s.CallStatus == "completed" || s.RawEvent == "Hangup") && !s.Failed()
}

func (s *StatusCallback) Failed() bool {
	switch s.CallStatus {
	case "failed", "busy", "no-answer", "no_answer", "canceled", "cancelled":
		return true
	}
	return false
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}
	if s.CallStatus != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.CallStatus}
	}
	if s.RawEvent != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.RawEvent}
	}
	return &internal_type.StatusError{Error: "failed", Reason: s.Event.String()}
}

func StatusEvent(event string, status string) internal_type.TelephonyEvent {
	switch event {
	case "Ring":
		return internal_type.TelephonyEventRinging
	case "StartApp":
		return internal_type.TelephonyEventAnswered
	case "StartStream":
		return internal_type.TelephonyEventStreamStarted
	case "Hangup", "StopStream":
		return internal_type.TelephonyEventCompleted
	}

	switch status {
	case "ringing":
		return internal_type.TelephonyEventRinging
	case "in-progress", "answered":
		return internal_type.TelephonyEventAnswered
	case "completed", "failed", "busy", "no-answer", "no_answer", "canceled", "cancelled":
		return internal_type.TelephonyEventCompleted
	}

	if status != "" {
		return internal_type.TelephonyEvent(status)
	}
	if event != "" {
		return internal_type.TelephonyEvent(event)
	}
	return internal_type.TelephonyEvent(WebhookEvent)
}
