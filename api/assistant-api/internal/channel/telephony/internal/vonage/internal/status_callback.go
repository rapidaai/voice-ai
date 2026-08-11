// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage

import (
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

type StatusCallback struct {
	Event          internal_type.TelephonyEvent
	Status         string
	ChannelUUID    string
	Duration       *time.Duration
	Price          string
	Detail         string
	SIPCode        string
	Reason         string
	DisconnectedBy string
	RawPayload     string
	Payload        utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	status, err := eventDetails.GetString("status")
	if err != nil || status == "" {
		return nil, ErrStatusCallbackStatusMissing
	}

	callback := &StatusCallback{
		Event:      StatusEvent(status),
		Status:     status,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}
	duration, err := eventDetails.GetDuration("duration")
	if err == nil {
		callback.Duration = utils.Ptr(duration)
	}
	if channelUUID, err := eventDetails.GetString("uuid"); err == nil {
		callback.ChannelUUID = channelUUID
	}
	if price, err := eventDetails.GetString("price"); err == nil {
		callback.Price = price
	}
	if detail, err := eventDetails.GetString("detail"); err == nil {
		callback.Detail = detail
	}
	if sipCode, err := eventDetails.GetString("sip_code"); err == nil {
		callback.SIPCode = sipCode
	}
	if reason, err := eventDetails.GetString("reason"); err == nil {
		callback.Reason = reason
	}
	if disconnectedBy, err := eventDetails.GetString("disconnected_by"); err == nil {
		callback.DisconnectedBy = disconnectedBy
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
	return s.Status == "completed" && !s.Failed()
}

func (s *StatusCallback) Failed() bool {
	switch s.Status {
	case "failed", "busy", "timeout", "unanswered", "rejected", "cancelled", "canceled":
		return true
	case "completed":
		return s.Detail != "" && s.Duration != nil && *s.Duration == 0
	}
	return false
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}
	if s.Detail != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Detail}
	}
	if s.Reason != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Reason}
	}
	if s.DisconnectedBy != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.DisconnectedBy}
	}
	if s.SIPCode != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.SIPCode}
	}
	if s.Status != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Status}
	}
	return &internal_type.StatusError{Error: "failed", Reason: s.Event.String()}
}

func StatusEvent(status string) internal_type.TelephonyEvent {
	switch status {
	case "started", "ringing":
		return internal_type.TelephonyEventRinging
	case "answered":
		return internal_type.TelephonyEventAnswered
	case "completed", "failed", "busy", "timeout", "unanswered", "rejected", "cancelled", "canceled":
		return internal_type.TelephonyEventCompleted
	default:
		return internal_type.TelephonyEvent(status)
	}
}
