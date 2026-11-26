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

	sip_config "github.com/rapidaai/api/assistant-api/sip/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	"github.com/rapidaai/pkg/authentication"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
)

func TestWithRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}
	var options ManagerOptions
	WithRapidaClient(client)(&options)

	if options.RapidaClient != client {
		t.Fatal("WithRapidaClient() did not retain RapidaClient")
	}
}

func TestServiceAuthenticationCreatesTenantBoundServicePrincipal(t *testing.T) {
	m, _, _ := newTestManager(t)
	auth := m.serviceAuthentication(context.Background(), 7, 11)
	if auth.Subject() != (authentication.Identity{Type: authentication.IdentityService, ID: 1, Issuer: "internal"}) {
		t.Fatalf("Subject() = %+v", auth.Subject())
	}
	if _, delegated := auth.Actor(); delegated {
		t.Fatal("Actor() reports delegation for a direct service principal")
	}
	if auth.Tenant() != (authentication.Tenant{OrganizationID: 7, ProjectID: 11}) {
		t.Fatalf("Tenant() = %+v", auth.Tenant())
	}
	if auth.Proof().Reference.Value == "" {
		t.Fatalf("Proof() = %+v", auth.Proof())
	}
}

func TestServiceAuthenticationRejectsMissingLocalVerifier(t *testing.T) {
	m, _, _ := newTestManager(t)
	m.servicePrincipalResolver = nil

	if auth := m.serviceAuthentication(context.Background(), 7, 11); auth.Subject().ID != 0 {
		t.Fatalf("serviceAuthentication() = %+v", auth)
	}
}

func TestStartRunsImmediateReconcile(t *testing.T) {
	m, db, _ := newTestManager(t)
	m.regClient = sip_runtime.NewRegistrationClient(nil, &sip_config.ListenConfig{}, m.logger)

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

	m.RegistrationRenewalFailed(ctx, sip_runtime.RegistrationEvent{
		DID:           "+14155550120",
		DeploymentID:  5001,
		Error:         sip_runtime.ErrRegistrationFailed,
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
	var insertedAudit struct {
		CreatedActorType string
		CreatedActorID   uint64
	}
	if err := db.Table("assistant_deployment_telephony_options").
		Select("created_actor_type, created_actor_id").
		Where("assistant_deployment_telephony_id = ? AND key = ?", 5001, OptKeySIPFailureClass).
		Scan(&insertedAudit).Error; err != nil {
		t.Fatalf("failed loading created actor: %v", err)
	}
	if insertedAudit.CreatedActorType != string(authentication.IdentityService) || insertedAudit.CreatedActorID != 9007 {
		t.Fatalf("created actor = %q/%d", insertedAudit.CreatedActorType, insertedAudit.CreatedActorID)
	}
	var updatedAudit struct {
		CreatedActorType string
		CreatedActorID   uint64
		UpdatedActorType string
		UpdatedActorID   uint64
	}
	if err := db.Table("assistant_deployment_telephony_options").
		Select("created_actor_type, created_actor_id, updated_actor_type, updated_actor_id").
		Where("assistant_deployment_telephony_id = ? AND key = ?", 5001, OptKeySIPStatus).
		Scan(&updatedAudit).Error; err != nil {
		t.Fatalf("failed loading updated actor: %v", err)
	}
	if updatedAudit.CreatedActorType != string(authentication.IdentityUser) || updatedAudit.CreatedActorID != 1 {
		t.Fatalf("created actor changed to %q/%d", updatedAudit.CreatedActorType, updatedAudit.CreatedActorID)
	}
	if updatedAudit.UpdatedActorType != string(authentication.IdentityService) || updatedAudit.UpdatedActorID != 9007 {
		t.Fatalf("updated actor = %q/%d", updatedAudit.UpdatedActorType, updatedAudit.UpdatedActorID)
	}
}

func TestRegistrationRenewedClearsFailureMetadata(t *testing.T) {
	m, db, ctx := newTestManager(t)
	insertSIPDeployment(t, db, 5002, 902, "+14155550121", StatusActive)
	m.RegistrationRenewalFailed(ctx, sip_runtime.RegistrationEvent{
		DID:           "+14155550121",
		DeploymentID:  5002,
		Error:         sip_runtime.ErrRegistrationFailed,
		FailureClass:  RegistrationFailureClassRenewal,
		FailureReason: RegistrationFailureReasonRenewalFailed,
		RetryCount:    2,
	})

	m.RegistrationRenewed(ctx, sip_runtime.RegistrationEvent{
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

	m.RegistrationExpired(ctx, sip_runtime.RegistrationEvent{
		DID:           "+14155550122",
		DeploymentID:  5003,
		Error:         sip_runtime.ErrRegistrationFailed,
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
