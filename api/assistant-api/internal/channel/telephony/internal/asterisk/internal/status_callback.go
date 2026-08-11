// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
)

type StatusCallback struct {
	Event       internal_type.TelephonyEvent
	EventType   string
	State       string
	DialStatus  string
	Cause       string
	CauseText   string
	Reason      string
	ChannelUUID string
	Duration    *time.Duration
	Price       string
	RawPayload  string
	Payload     utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	eventType, err := eventDetails.GetString("type")
	if err != nil || eventType == "" {
		eventType, _ = eventDetails.GetString("event")
	}

	callback := &StatusCallback{
		EventType:  eventType,
		RawPayload: rawCallbackPayload,
		Payload:    eventDetails,
	}

	if channel, ok := eventDetails["channel"].(map[string]interface{}); ok {
		channelDetails := utils.Option(channel)
		if channelUUID, err := channelDetails.GetString("id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
		if callback.ChannelUUID == "" {
			if channelUUID, err := channelDetails.GetString("name"); err == nil {
				callback.ChannelUUID = channelUUID
			}
		}
		if state, err := channelDetails.GetString("state"); err == nil {
			callback.State = state
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("channel_id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("uniqueid"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}
	if callback.ChannelUUID == "" {
		if channelUUID, err := eventDetails.GetString("id"); err == nil {
			callback.ChannelUUID = channelUUID
		}
	}

	duration, err := eventDetails.GetDuration("duration")
	if err != nil {
		duration, err = eventDetails.GetDuration("billsec")
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
	if dialStatus, err := eventDetails.GetString("dialstatus"); err == nil {
		callback.DialStatus = dialStatus
	}
	if cause, err := eventDetails.GetString("cause"); err == nil {
		callback.Cause = cause
	} else if value, ok := eventDetails["cause"]; ok && value != nil {
		callback.Cause = fmt.Sprintf("%v", value)
	}
	if causeText, err := eventDetails.GetString("cause_txt"); err == nil {
		callback.CauseText = causeText
	} else if value, ok := eventDetails["cause_txt"]; ok && value != nil {
		callback.CauseText = fmt.Sprintf("%v", value)
	}
	if reason, err := eventDetails.GetString("reason"); err == nil {
		callback.Reason = reason
	}

	callback.Event = StatusEvent(callback.EventType, callback.State, callback.DialStatus)
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
	switch s.EventType {
	case "ChannelDestroyed", "ChannelHangupRequest":
		return !s.Failed()
	case "StasisEnd":
		return true
	}
	return false
}

func (s *StatusCallback) Failed() bool {
	if s.Reason != "" {
		return true
	}
	switch strings.ToUpper(s.DialStatus) {
	case "BUSY", "NOANSWER", "NO_ANSWER", "CHANUNAVAIL", "CONGESTION", "FAILED", "FAILURE", "REJECTED", "TIMEOUT":
		return true
	}
	switch strings.ToUpper(s.CauseText) {
	case "USER_BUSY", "BUSY", "NO_ANSWER", "NO ANSWER", "CALL_REJECTED", "REJECTED", "CONGESTION", "NETWORK_OUT_OF_ORDER", "NORMAL_TEMPORARY_FAILURE":
		return true
	}
	switch s.Cause {
	case "", "16", "0":
		return false
	default:
		return s.EventType == "ChannelDestroyed" || s.EventType == "ChannelHangupRequest"
	}
}

func (s *StatusCallback) StatusError() *internal_type.StatusError {
	if !s.Failed() {
		return nil
	}
	if s.Reason != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Reason}
	}
	if s.DialStatus != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.DialStatus}
	}
	if s.CauseText != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.CauseText}
	}
	if s.Cause != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.Cause}
	}
	if s.EventType != "" {
		return &internal_type.StatusError{Error: "failed", Reason: s.EventType}
	}
	return &internal_type.StatusError{Error: "failed", Reason: s.Event.String()}
}

func StatusEvent(eventType string, state string, dialStatus string) internal_type.TelephonyEvent {
	switch eventType {
	case "ChannelStateChange":
		switch strings.ToUpper(state) {
		case "RING", "RINGING":
			return internal_type.TelephonyEventRinging
		case "UP":
			return internal_type.TelephonyEventAnswered
		}
	case "Dial":
		switch strings.ToUpper(dialStatus) {
		case "ANSWER", "ANSWERED":
			return internal_type.TelephonyEventAnswered
		case "RING", "RINGING", "PROGRESS":
			return internal_type.TelephonyEventRinging
		case "CANCEL", "CANCELED", "CANCELLED", "BUSY", "NOANSWER", "NO_ANSWER", "CHANUNAVAIL", "CONGESTION", "FAILED", "FAILURE", "REJECTED", "TIMEOUT":
			return internal_type.TelephonyEventCompleted
		}
	case "ChannelCreated", "channel_created":
		return internal_type.TelephonyEventRinging
	case "StasisStart":
		return internal_type.TelephonyEventAnswered
	case "ChannelDestroyed", "ChannelHangupRequest", "StasisEnd":
		return internal_type.TelephonyEventCompleted
	}
	return internal_type.TelephonyEvent(eventType)
}
