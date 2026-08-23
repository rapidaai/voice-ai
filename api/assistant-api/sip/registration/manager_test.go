// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_registration

import (
	"context"
	"testing"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/types"
	"google.golang.org/grpc/metadata"
)

func TestServiceAuthenticationMintsDelegatedServiceActor(t *testing.T) {
	m, _, _ := newTestManager(t)
	auth := m.serviceAuthentication(7, 11)
	if auth.Type() != types.AuthTypeService {
		t.Fatalf("Type() = %q, want %q", auth.Type(), types.AuthTypeService)
	}
	actor, err := auth.Actor()
	if err != nil || actor != (types.ActorIdentity{Type: types.ActorTypeService, ID: 9007}) {
		t.Fatalf("Actor() = %+v, %v", actor, err)
	}

	internalClient := clients.NewInternalClient(&m.assistantConfig.AppConfig, nil, nil)
	outgoingContext, err := internalClient.WithAuth(context.Background(), auth)
	if err != nil {
		t.Fatalf("WithAuth() error = %v", err)
	}
	outgoingMetadata, ok := metadata.FromOutgoingContext(outgoingContext)
	if !ok || len(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)) != 1 {
		t.Fatalf("service scope metadata = %v", outgoingMetadata)
	}
	scope, err := types.ExtractServiceScope(outgoingMetadata.Get(types.SERVICE_SCOPE_KEY)[0], m.assistantConfig.Secret)
	if err != nil {
		t.Fatalf("ExtractServiceScope() error = %v", err)
	}
	forwardedAuth, err := scope.Authentication()
	if err != nil {
		t.Fatalf("Authentication() error = %v", err)
	}
	forwardedActor, err := forwardedAuth.Actor()
	if err != nil || forwardedActor != actor {
		t.Fatalf("Actor() = %+v, %v; want %+v", forwardedActor, err, actor)
	}
	projectContext, err := forwardedAuth.ProjectContext()
	if err != nil || projectContext.OrganizationID != 7 || projectContext.ProjectID != 11 {
		t.Fatalf("ProjectContext() = %+v, %v", projectContext, err)
	}
}

func TestStartRunsImmediateReconcile(t *testing.T) {
	m, db, _ := newTestManager(t)
	m.regClient = sip_infra.NewRegistrationClient(nil, &sip_infra.ListenConfig{}, m.logger)

	insertSIPDeploymentWithOptions(t, db, 4001, 801, map[string]string{
		OptKeyCredentialID: "101",
		OptKeySIPInbound:   "true",
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		m.Start(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("expected immediate reconcile to write missing DID status")
		case <-ticker.C:
			var count int64
			if err := db.Table("assistant_deployment_telephony_options").
				Where("assistant_deployment_telephony_id = ? AND key = ? AND value = ?", 4001, OptKeySIPFailureReason, string(RegistrationFailureReasonMissingDID)).
				Count(&count).Error; err != nil {
				t.Fatalf("failed checking registration status: %v", err)
			}
			if count > 0 {
				cancel()
				return
			}
		}
	}
}

func TestRegistrationRenewalFailedWritesDurableMetadata(t *testing.T) {
	m, db, ctx := newTestManager(t)
	insertSIPDeployment(t, db, 5001, 901, "+14155550120", StatusActive)

	m.RegistrationRenewalFailed(ctx, sip_infra.RegistrationEvent{
		DID:           "+14155550120",
		DeploymentID:  5001,
		Error:         sip_infra.ErrRegistrationFailed,
		FailureClass:  RegistrationFailureClassRenewal,
		FailureReason: RegistrationFailureReasonRenewalFailed,
		StatusCode:    503,
		StatusText:    "Service Unavailable",
		RetryCount:    3,
		NextRetryAt:   time.Now().Add(time.Minute),
	})

	if status := getOptionValue(t, db, 5001, OptKeySIPStatus); status != string(StatusActive) {
		t.Fatalf("expected status=%s, got %s", StatusActive, status)
	}
	if failureClass := getOptionValue(t, db, 5001, OptKeySIPFailureClass); failureClass != string(RegistrationFailureClassRenewal) {
		t.Fatalf("expected failure_class=%s, got %s", RegistrationFailureClassRenewal, failureClass)
	}
	if failureReason := getOptionValue(t, db, 5001, OptKeySIPFailureReason); failureReason != string(RegistrationFailureReasonRenewalFailed) {
		t.Fatalf("expected failure_reason=%s, got %s", RegistrationFailureReasonRenewalFailed, failureReason)
	}
	if responseCode := getOptionValue(t, db, 5001, OptKeySIPResponseCode); responseCode != "503" {
		t.Fatalf("expected response code 503, got %s", responseCode)
	}
	if retryCount := getOptionValue(t, db, 5001, OptKeySIPRetry); retryCount != "3" {
		t.Fatalf("expected retry count 3, got %s", retryCount)
	}
}

func TestRegistrationRenewedClearsFailureMetadata(t *testing.T) {
	m, db, ctx := newTestManager(t)
	insertSIPDeployment(t, db, 5002, 902, "+14155550121", StatusActive)
	m.RegistrationRenewalFailed(ctx, sip_infra.RegistrationEvent{
		DID:           "+14155550121",
		DeploymentID:  5002,
		Error:         sip_infra.ErrRegistrationFailed,
		FailureClass:  RegistrationFailureClassRenewal,
		FailureReason: RegistrationFailureReasonRenewalFailed,
		RetryCount:    2,
	})

	m.RegistrationRenewed(ctx, sip_infra.RegistrationEvent{
		DID:          "+14155550121",
		DeploymentID: 5002,
	})

	if failureClass := getOptionValue(t, db, 5002, OptKeySIPFailureClass); failureClass != "" {
		t.Fatalf("expected failure class to be cleared, got %s", failureClass)
	}
	if failureReason := getOptionValue(t, db, 5002, OptKeySIPFailureReason); failureReason != "" {
		t.Fatalf("expected failure reason to be cleared, got %s", failureReason)
	}
	if retryCount := getOptionValue(t, db, 5002, OptKeySIPRetry); retryCount != "0" {
		t.Fatalf("expected retry count 0, got %s", retryCount)
	}
}

func TestRegistrationExpiredMarksUnreachable(t *testing.T) {
	m, db, ctx := newTestManager(t)
	insertSIPDeployment(t, db, 5003, 903, "+14155550122", StatusActive)

	m.RegistrationExpired(ctx, sip_infra.RegistrationEvent{
		DID:           "+14155550122",
		DeploymentID:  5003,
		Error:         sip_infra.ErrRegistrationFailed,
		FailureClass:  RegistrationFailureClassRenewal,
		FailureReason: RegistrationFailureReasonRenewalFailed,
		RetryCount:    10,
	})

	if status := getOptionValue(t, db, 5003, OptKeySIPStatus); status != string(StatusUnreachable) {
		t.Fatalf("expected status=%s, got %s", StatusUnreachable, status)
	}
	if failureReason := getOptionValue(t, db, 5003, OptKeySIPFailureReason); failureReason != string(RegistrationFailureReasonRenewalFailed) {
		t.Fatalf("expected failure_reason=%s, got %s", RegistrationFailureReasonRenewalFailed, failureReason)
	}
}
