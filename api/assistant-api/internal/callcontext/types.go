// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_callcontext

import (
	"errors"
	"fmt"
	"strings"
	"time"

	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
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
	Id             uint64    `json:"id" gorm:"type:bigint;primaryKey;<-:create"`
	ContextID      string    `json:"contextId" gorm:"column:context_id;type:varchar(36);not null;uniqueIndex"`
	Status         string    `json:"status" gorm:"column:status;type:varchar(20);not null;default:pending"`
	AssistantID    uint64    `json:"assistantId" gorm:"column:assistant_id;type:bigint;not null"`
	ConversationID uint64    `json:"conversationId" gorm:"column:conversation_id;type:bigint;not null"`
	ProjectID      uint64    `json:"projectId" gorm:"column:project_id;type:bigint;not null;default:0"`
	OrganizationID uint64    `json:"organizationId" gorm:"column:organization_id;type:bigint;not null;default:0"`
	AuthType       string    `json:"authType" gorm:"column:auth_type;type:varchar(50);not null;default:''"`
	AuthUserID     *uint64   `json:"authUserId,omitempty" gorm:"column:auth_user_id;type:bigint"`
	AuthActorType  *string   `json:"authActorType,omitempty" gorm:"column:auth_actor_type;type:varchar(50)"`
	AuthActorID    *string   `json:"authActorId,omitempty" gorm:"column:auth_actor_id;type:varchar(255)"`
	Provider       string    `json:"provider" gorm:"column:provider;type:varchar(50);not null;default:''"`
	Direction      string    `json:"direction" gorm:"column:direction;type:varchar(20);not null;default:''"`
	CallerNumber   string    `json:"callerNumber" gorm:"column:caller_number;type:varchar(50);not null;default:''"`
	FromNumber     string    `json:"fromNumber" gorm:"column:from_number;type:varchar(50);not null;default:''"`
	CreatedDate    time.Time `json:"createdDate" gorm:"type:timestamp;not null;default:NOW();<-:create"`
	UpdatedDate    time.Time `json:"updatedDate" gorm:"type:timestamp;default:null"`

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

func (cc *CallContext) SetAuthentication(auth *types.Authentication) error {
	if auth == nil || !auth.IsAuthenticated() {
		return types.ErrUnauthenticated
	}

	cc.AuthType = auth.AuthType.String()
	cc.AuthUserID = nil
	cc.OrganizationID = 0
	cc.ProjectID = 0
	cc.AuthActorType = nil
	cc.AuthActorID = nil

	if userContext, err := auth.UserContext(); err == nil {
		cc.AuthUserID = &userContext.UserID
	}
	if organizationContext, err := auth.OrganizationContext(); err == nil {
		cc.OrganizationID = organizationContext.OrganizationID
	}
	if projectContext, err := auth.ProjectContext(); err == nil {
		cc.ProjectID = projectContext.ProjectID
	}
	if actor, err := auth.Actor(); err == nil {
		actorType := string(actor.Type)
		cc.AuthActorType = &actorType
		cc.AuthActorID = &actor.ID
	}
	return nil
}

func (cc *CallContext) ToAuthentication() (*types.Authentication, error) {
	if cc == nil {
		return nil, types.ErrUnauthenticated
	}

	auth := &types.Authentication{AuthType: types.AuthType(cc.AuthType)}
	if cc.AuthUserID != nil && *cc.AuthUserID != 0 {
		auth.UserValue = &types.UserContext{UserID: *cc.AuthUserID}
	}
	if cc.OrganizationID != 0 {
		auth.OrganizationValue = &types.OrganizationContext{OrganizationID: cc.OrganizationID}
	}
	if cc.ProjectID != 0 {
		auth.ProjectValue = &types.ProjectContext{OrganizationID: cc.OrganizationID, ProjectID: cc.ProjectID}
	}

	actorType := ""
	actorID := ""
	if cc.AuthActorType != nil {
		actorType = strings.TrimSpace(*cc.AuthActorType)
	}
	if cc.AuthActorID != nil {
		actorID = strings.TrimSpace(*cc.AuthActorID)
	}
	if (actorType == "") != (actorID == "") {
		return nil, errors.New("call context authentication actor is incomplete")
	}
	if actorType != "" {
		auth.ActorValue = &types.ActorIdentity{Type: types.ActorType(actorType), ID: actorID}
	}

	if !auth.IsAuthenticated() {
		return nil, fmt.Errorf("call context authentication is invalid for auth type %q: %w", cc.AuthType, types.ErrUnauthenticated)
	}
	return auth, nil
}
