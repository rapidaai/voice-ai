// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_callcontext

import (
	"time"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"gorm.io/gorm"
)

// Call context status constants.
//
//	PENDING → CLAIMED
//
// Save creates with PENDING. Claim transitions to CLAIMED when the media path
// consumes the context (typically at session establishment). Get reads
// regardless of status, since async callbacks may arrive after claim.
const (
	StatusPending   = "pending"   // Context created, awaiting media-path consumption
	StatusClaimed   = "claimed"   // Context consumed (media-path bound, or call ended unclaimed)
	StatusCompleted = "completed" // Provider reported a terminal completed state
	StatusFailed    = "failed"    // Provider reported terminal failure before media claim
	StatusCancelled = "cancelled" // Provider reported terminal cancellation before media claim
)

const (
	CallStatusNew       = "new"
	CallStatusInitiated = "initiated"
	CallStatusRinging   = "ringing"
	CallStatusAnswered  = "answered"
	CallStatusFailed    = "failed"
	CallStatusCompleted = "completed"
	CallStatusCancelled = "cancelled"
)

// CallContext holds all the information needed to resolve a call session.
// It bridges the gap between the HTTP call-setup request (inbound webhook or outbound gRPC)
// and the AudioSocket/WebSocket connection that follows.
//
// Stored in Postgres (call_contexts table). The status field provides atomic
// claiming: only one caller can transition pending→claimed.
type CallContext struct {
	Id             uint64  `json:"id" gorm:"type:bigint;primaryKey;<-:create"`
	ContextID      string  `json:"contextId" gorm:"column:context_id;type:varchar(36);not null;uniqueIndex"`
	Status         string  `json:"status" gorm:"column:status;type:varchar(20);not null;default:pending"`
	AssistantID    uint64  `json:"assistantId" gorm:"column:assistant_id;type:bigint;not null"`
	ConversationID uint64  `json:"conversationId" gorm:"column:conversation_id;type:bigint;not null"`
	ProjectID      uint64  `json:"projectId" gorm:"column:project_id;type:bigint;not null;default:0"`
	OrganizationID uint64  `json:"organizationId" gorm:"column:organization_id;type:bigint;not null;default:0"`
	AuthType       string  `json:"authType" gorm:"column:auth_type;type:varchar(50);not null;default:''"`
	AuthUserID     *uint64 `json:"authUserId,omitempty" gorm:"column:auth_user_id;type:bigint"`
	AuthActorType  *string `json:"authActorType,omitempty" gorm:"column:auth_actor_type;type:varchar(50)"`
	AuthActorID    *uint64 `json:"authActorId,omitempty" gorm:"column:auth_actor_id;type:bigint"`

	AuthPrincipalVersion    int16     `json:"authPrincipalVersion" gorm:"column:auth_principal_version;type:smallint;not null;default:1"`
	AuthSubjectType         string    `json:"authSubjectType" gorm:"column:auth_subject_type;type:varchar(32)"`
	AuthSubjectID           uint64    `json:"authSubjectId" gorm:"column:auth_subject_id;type:bigint"`
	AuthSubjectIssuer       string    `json:"authSubjectIssuer" gorm:"column:auth_subject_issuer;type:varchar(200)"`
	AuthServiceActorID      *uint64   `json:"authServiceActorId,omitempty" gorm:"column:auth_service_actor_id;type:bigint"`
	AuthServiceActorIssuer  *string   `json:"authServiceActorIssuer,omitempty" gorm:"column:auth_service_actor_issuer;type:varchar(200)"`
	AuthProofMethod         string    `json:"authProofMethod" gorm:"column:auth_proof_method;type:varchar(50)"`
	AuthProofAudience       string    `json:"authProofAudience" gorm:"column:auth_proof_audience;type:varchar(100)"`
	AuthProofReferenceType  string    `json:"authProofReferenceType" gorm:"column:auth_proof_reference_type;type:varchar(50)"`
	AuthProofReferenceValue string    `json:"authProofReferenceValue" gorm:"column:auth_proof_reference_value;type:varchar(200)"`
	AuthAuthorities         []string  `json:"authAuthorities" gorm:"column:auth_authorities;type:jsonb;not null;default:'[]';serializer:json"`
	Provider                string    `json:"provider" gorm:"column:provider;type:varchar(50);not null;default:''"`
	Direction               string    `json:"direction" gorm:"column:direction;type:varchar(20);not null;default:''"`
	CallerNumber            string    `json:"callerNumber" gorm:"column:caller_number;type:varchar(50);not null;default:''"`
	FromNumber              string    `json:"fromNumber" gorm:"column:from_number;type:varchar(50);not null;default:''"`
	CreatedDate             time.Time `json:"createdDate" gorm:"type:timestamp;not null;default:NOW();<-:create"`
	UpdatedDate             time.Time `json:"updatedDate" gorm:"type:timestamp;default:null"`

	// AssistantProviderId is the version identifier for the assistant provider.
	AssistantProviderId uint64 `json:"assistantProviderId" gorm:"column:assistant_provider_id;type:bigint;not null;default:0"`

	// ChannelUUID is the provider-specific call identifier (Twilio CallSid, Vonage UUID,
	// Asterisk channel ID, SIP Call-ID, etc.).
	ChannelUUID        string `json:"channelUuid" gorm:"column:channel_uuid;type:varchar(200);not null;default:''"`
	CallStatus         string `json:"callStatus" gorm:"column:call_status;type:varchar(30);not null;default:''"`
	CallError          string `json:"callError" gorm:"column:call_error;type:text;not null;default:''"`
	FailureClass       string `json:"failureClass" gorm:"column:failure_class;type:varchar(80);not null;default:''"`
	FailureReason      string `json:"failureReason" gorm:"column:failure_reason;type:text;not null;default:''"`
	DisconnectReason   string `json:"disconnectReason" gorm:"column:disconnect_reason;type:varchar(120);not null;default:''"`
	Retryable          bool   `json:"retryable" gorm:"column:retryable;type:boolean;not null;default:false"`
	ProviderStatusCode int    `json:"providerStatusCode" gorm:"column:provider_status_code;type:int;not null;default:0"`
}

func (CallContext) TableName() string {
	return "call_contexts"
}

func (cc *CallContext) BeforeCreate(tx *gorm.DB) (err error) {
	if cc.Id <= 0 {
		cc.Id = gorm_generator.ID()
	}
	if cc.CreatedDate.IsZero() {
		cc.CreatedDate = time.Now()
	}
	return nil
}

func (cc *CallContext) SetAuthentication(auth authentication.Principal) error {
	if auth.Subject().ID == 0 {
		return pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	subject := auth.Subject()
	tenant := auth.Tenant()
	proof := auth.Proof()
	cc.AuthPrincipalVersion = 1
	cc.AuthSubjectType = string(subject.Type)
	cc.AuthSubjectID = subject.ID
	cc.AuthSubjectIssuer = subject.Issuer
	cc.OrganizationID = tenant.OrganizationID
	cc.ProjectID = tenant.ProjectID
	cc.AuthProofMethod = string(proof.Method)
	cc.AuthProofAudience = string(proof.Audience)
	cc.AuthProofReferenceType = string(proof.Reference.Type)
	cc.AuthProofReferenceValue = proof.Reference.Value
	cc.AuthAuthorities = make([]string, 0, len(auth.Authorities()))
	for _, authority := range auth.Authorities() {
		cc.AuthAuthorities = append(cc.AuthAuthorities, string(authority))
	}
	if actor, delegated := auth.Actor(); delegated {
		cc.AuthServiceActorID = &actor.ID
		cc.AuthServiceActorIssuer = &actor.Issuer
	} else {
		cc.AuthServiceActorID = nil
		cc.AuthServiceActorIssuer = nil
	}
	return nil
}

func (cc *CallContext) ToAuthentication() (authentication.Principal, error) {
	if cc == nil || cc.AuthPrincipalVersion != 1 {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	authorities := make([]authentication.PermissionKey, 0, len(cc.AuthAuthorities))
	for _, authority := range cc.AuthAuthorities {
		authorities = append(authorities, authentication.PermissionKey(authority))
	}
	subject := authentication.Identity{Type: authentication.IdentityType(cc.AuthSubjectType), ID: cc.AuthSubjectID, Issuer: cc.AuthSubjectIssuer}
	tenant := authentication.Tenant{OrganizationID: cc.OrganizationID, ProjectID: cc.ProjectID}
	proof := authentication.Proof{
		Method:   authentication.AuthenticationMethod(cc.AuthProofMethod),
		Audience: authentication.Audience(cc.AuthProofAudience),
		Reference: authentication.ProofReference{
			Type:  authentication.ProofReferenceType(cc.AuthProofReferenceType),
			Value: cc.AuthProofReferenceValue,
		},
	}
	if cc.AuthServiceActorID != nil && cc.AuthServiceActorIssuer != nil {
		return authentication.NewDelegatedPrincipal(subject, authentication.ServiceActor{ID: *cc.AuthServiceActorID, Issuer: *cc.AuthServiceActorIssuer}, tenant, proof, authorities)
	}
	if cc.AuthServiceActorID != nil || cc.AuthServiceActorIssuer != nil {
		return authentication.Principal{}, pkg_errors.AuthenticationPrincipalInvalidError{}
	}
	return authentication.NewDirectPrincipal(subject, tenant, proof, authorities)
}
