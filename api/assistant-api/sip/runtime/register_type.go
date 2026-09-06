// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/rapidaai/pkg/validator"
)

type RegistrationFailureClass string

type RegistrationFailureReason string

type RegistrationError struct {
	Class      RegistrationFailureClass
	Reason     RegistrationFailureReason
	StatusCode int
	StatusText string
	Retryable  bool
	Cause      error
}

func (err *RegistrationError) Error() string {
	if !validator.NonNil(err) {
		return ""
	}
	message := fmt.Sprintf("%s: %s", err.Class, err.Reason)
	if err.StatusCode > 0 {
		message = fmt.Sprintf("%s: %d %s", message, err.StatusCode, err.StatusText)
	}
	if err.Cause != nil {
		message = fmt.Sprintf("%s: %v", message, err.Cause)
	}
	return message
}

func (err *RegistrationError) Unwrap() error {
	if !validator.NonNil(err) {
		return nil
	}
	return err.Cause
}

type RegistrationEvent struct {
	DID           string
	DeploymentID  uint64
	AssistantID   uint64
	Server        string
	ExpiresAt     time.Time
	GrantedExpiry uint32
	RetryCount    int
	NextRetryAt   time.Time
	Error         error
	FailureClass  RegistrationFailureClass
	FailureReason RegistrationFailureReason
	StatusCode    int
	StatusText    string
}

type RegistrationObserver interface {
	RegistrationRenewed(ctx context.Context, event RegistrationEvent)
	RegistrationRenewalFailed(ctx context.Context, event RegistrationEvent)
	RegistrationExpired(ctx context.Context, event RegistrationEvent)
	RegistrationUnregisterFailed(ctx context.Context, event RegistrationEvent)
}

type RegistrationSnapshot struct {
	DID               string
	Active            bool
	Healthy           bool
	ExpiresAt         time.Time
	RenewalRetryCount int
	LastRenewalError  error
	FailureClass      RegistrationFailureClass
	FailureReason     RegistrationFailureReason
}
