package internal_callcontext

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error    { return nil }
func (t *testPostgresConnector) Name() string                         { return "test-postgres" }
func (t *testPostgresConnector) IsConnected(ctx context.Context) bool { return t.db != nil }
func (t *testPostgresConnector) Disconnect(ctx context.Context) error { return nil }
func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.db.WithContext(ctx).Raw(qry).Scan(dest).Error
}
func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB { return t.db.WithContext(ctx) }

func newTestStore(t *testing.T) (Store, context.Context) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE call_contexts (
			id INTEGER PRIMARY KEY,
			context_id TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'pending',
			assistant_id BIGINT NOT NULL,
			conversation_id BIGINT NOT NULL,
			project_id BIGINT NOT NULL DEFAULT 0,
			organization_id BIGINT NOT NULL DEFAULT 0,
			auth_token TEXT NOT NULL DEFAULT '',
			auth_type TEXT NOT NULL DEFAULT '',
			auth_user_id BIGINT,
			auth_actor_type TEXT,
			auth_actor_id INTEGER,
			provider TEXT NOT NULL DEFAULT '',
			direction TEXT NOT NULL DEFAULT '',
			caller_number TEXT NOT NULL DEFAULT '',
			from_number TEXT NOT NULL DEFAULT '',
			created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_date DATETIME DEFAULT NULL,
			assistant_provider_id BIGINT NOT NULL DEFAULT 0,
			channel_uuid TEXT NOT NULL DEFAULT '',
			call_status TEXT NOT NULL DEFAULT '',
			call_error TEXT NOT NULL DEFAULT '',
			failure_class TEXT NOT NULL DEFAULT '',
			failure_reason TEXT NOT NULL DEFAULT '',
			disconnect_reason TEXT NOT NULL DEFAULT '',
			retryable BOOLEAN NOT NULL DEFAULT FALSE,
			provider_status_code INTEGER NOT NULL DEFAULT 0
		)
	`).Error; err != nil {
		t.Fatalf("failed to initialize call contexts schema: %v", err)
	}

	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return NewStore(&testPostgresConnector{db: db}, logger), context.Background()
}

func TestStoreUpdateCallStatus(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "outbound-status-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "sip",
		Direction:      "outbound",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	err = store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		CallStatus:         CallStatusFailed,
		CallError:          "Provider 486 Busy Here",
		FailureClass:       "busy",
		FailureReason:      "Busy Here",
		DisconnectReason:   "outbound_rejected",
		Retryable:          false,
		ProviderStatusCode: 486,
	})
	if err != nil {
		t.Fatalf("UpdateCallStatus returned error: %v", err)
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("expected context status failed, got %q", got.Status)
	}
	if got.CallStatus != CallStatusFailed {
		t.Fatalf("expected call status failed, got %q", got.CallStatus)
	}
	if got.CallError != "Provider 486 Busy Here" {
		t.Fatalf("expected call error, got %q", got.CallError)
	}
	if got.FailureClass != "busy" || got.FailureReason != "Busy Here" {
		t.Fatalf("unexpected failure fields: class=%q reason=%q", got.FailureClass, got.FailureReason)
	}
	if got.DisconnectReason != "outbound_rejected" {
		t.Fatalf("expected disconnect reason outbound_rejected, got %q", got.DisconnectReason)
	}
	if got.Retryable {
		t.Fatal("expected retryable false")
	}
	if got.ProviderStatusCode != 486 {
		t.Fatalf("expected provider status 486, got %d", got.ProviderStatusCode)
	}
}

func TestStoreSaveInitializesPendingContext(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		Status:         StatusFailed,
		AssistantID:    77,
		ConversationID: 1001,
		ProjectID:      22,
		OrganizationID: 33,
		AuthType:       types.AuthTypeService.String(),
		Provider:       "twilio",
		Direction:      "outbound",
		CallerNumber:   "+15550001111",
		FromNumber:     "+15550002222",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}
	if contextID == "" {
		t.Fatal("expected generated context id")
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusPending {
		t.Fatalf("expected saved context status pending, got %q", got.Status)
	}
	if got.Provider != "twilio" || got.Direction != "outbound" {
		t.Fatalf("unexpected provider/direction: provider=%q direction=%q", got.Provider, got.Direction)
	}
	if got.CallerNumber != "+15550001111" || got.FromNumber != "+15550002222" {
		t.Fatalf("unexpected caller/from numbers: caller=%q from=%q", got.CallerNumber, got.FromNumber)
	}
	if got.CreatedDate.IsZero() {
		t.Fatal("expected created date to be initialized")
	}
}

func TestStoreClaimTransitionsPendingContextOnce(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "claim-once-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "asterisk",
		Direction:      "inbound",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	claimed, err := store.Claim(ctx, contextID)
	if err != nil {
		t.Fatalf("Claim returned error: %v", err)
	}
	if claimed.Status != StatusClaimed {
		t.Fatalf("expected claimed status, got %q", claimed.Status)
	}
	if claimed.ContextID != contextID {
		t.Fatalf("expected context id %s, got %s", contextID, claimed.ContextID)
	}

	if _, err := store.Claim(ctx, contextID); err == nil {
		t.Fatal("expected second claim to fail")
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusClaimed {
		t.Fatalf("expected stored context to remain claimed, got %q", got.Status)
	}
	if got.UpdatedDate.IsZero() {
		t.Fatal("expected claim to set updated date")
	}
}

func TestStoreUpdateFieldAllowlist(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "field-allowlist-context",
		AssistantID:    77,
		ConversationID: 1001,
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	if err := store.UpdateField(ctx, contextID, "channel_uuid", "provider-call-uuid"); err != nil {
		t.Fatalf("UpdateField channel_uuid returned error: %v", err)
	}
	if err := store.UpdateField(ctx, contextID, "provider", "vonage"); err != nil {
		t.Fatalf("UpdateField provider returned error: %v", err)
	}
	if err := store.UpdateField(ctx, contextID, "call_error", "hidden update"); err == nil {
		t.Fatal("expected unsupported field update to fail")
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ChannelUUID != "provider-call-uuid" {
		t.Fatalf("expected channel uuid update, got %q", got.ChannelUUID)
	}
	if got.Provider != "vonage" {
		t.Fatalf("expected provider update, got %q", got.Provider)
	}
	if got.CallError != "" {
		t.Fatalf("unsupported field update changed call error to %q", got.CallError)
	}
}

func TestStoreUpdateCallStatus_CancelledMarksContextFailed(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "outbound-cancelled-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "sip",
		Direction:      "outbound",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	err = store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		CallStatus:       CallStatusCancelled,
		FailureClass:     "cancelled",
		DisconnectReason: "outbound_cancelled_before_answer",
	})
	if err != nil {
		t.Fatalf("UpdateCallStatus returned error: %v", err)
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("expected cancelled outbound context to be failed, got %q", got.Status)
	}
	if got.CallStatus != CallStatusCancelled {
		t.Fatalf("expected call status cancelled, got %q", got.CallStatus)
	}
}

func TestStoreUpdateCallStatus_CompletedMarksContextCompleted(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "completed-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "sip",
		Direction:      "inbound",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	err = store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		CallStatus:         CallStatusCompleted,
		DisconnectReason:   "normal_clearing",
		ProviderStatusCode: 16,
	})
	if err != nil {
		t.Fatalf("UpdateCallStatus returned error: %v", err)
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusCompleted {
		t.Fatalf("expected context status completed, got %q", got.Status)
	}
	if got.CallStatus != CallStatusCompleted {
		t.Fatalf("expected call status completed, got %q", got.CallStatus)
	}
	if got.DisconnectReason != "normal_clearing" {
		t.Fatalf("expected disconnect reason normal_clearing, got %q", got.DisconnectReason)
	}
	if got.ProviderStatusCode != 16 {
		t.Fatalf("expected provider status 16, got %d", got.ProviderStatusCode)
	}
}

func TestStoreUpdateCallStatus_ExpectedCallStatus(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "expected-call-status-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "sip",
		Direction:      "outbound",
		CallStatus:     CallStatusNew,
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}
	if err := store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		CallStatus: CallStatusRinging,
	}); err != nil {
		t.Fatalf("UpdateCallStatus returned error: %v", err)
	}
	if err := store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		ExpectedCallStatus: CallStatusNew,
		CallStatus:         CallStatusFailed,
		FailureClass:       "no_answer",
	}); err != nil {
		t.Fatalf("conditional UpdateCallStatus returned error: %v", err)
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.CallStatus != CallStatusRinging {
		t.Fatalf("expected call status ringing to be preserved, got %q", got.CallStatus)
	}
	if got.Status != StatusPending {
		t.Fatalf("expected context status pending, got %q", got.Status)
	}
}

func TestStoreUpdateField_DoesNotClaimTerminalContexts(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  string
		callStatus     string
		expectedStatus string
	}{
		{
			name:           "failed",
			callStatus:     CallStatusFailed,
			expectedStatus: StatusFailed,
		},
		{
			name:           "completed",
			callStatus:     CallStatusCompleted,
			expectedStatus: StatusCompleted,
		},
		{
			name:           "cancelled",
			initialStatus:  StatusCancelled,
			expectedStatus: StatusCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, ctx := newTestStore(t)

			contextID, err := store.Save(ctx, &CallContext{
				ContextID:      "terminal-" + tt.name + "-context",
				AssistantID:    77,
				ConversationID: 1001,
				Provider:       "sip",
				Direction:      "outbound",
			})
			if err != nil {
				t.Fatalf("failed to save call context: %v", err)
			}

			if tt.callStatus != "" {
				err = store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
					CallStatus:       tt.callStatus,
					FailureClass:     "terminal",
					DisconnectReason: "terminal_reason",
				})
				if err != nil {
					t.Fatalf("UpdateCallStatus returned error: %v", err)
				}
			}
			if tt.initialStatus != "" {
				if err := store.UpdateField(ctx, contextID, "status", tt.initialStatus); err != nil {
					t.Fatalf("UpdateField initial status returned error: %v", err)
				}
			}

			if err := store.UpdateField(ctx, contextID, "status", StatusClaimed); err != nil {
				t.Fatalf("UpdateField claimed returned error: %v", err)
			}

			got, err := store.Get(ctx, contextID)
			if err != nil {
				t.Fatalf("Get returned error: %v", err)
			}
			if got.Status != tt.expectedStatus {
				t.Fatalf("expected terminal context to remain %q, got %q", tt.expectedStatus, got.Status)
			}
		})
	}
}

func TestStoreUpdateField_DoesNotClaimFailedContext(t *testing.T) {
	store, ctx := newTestStore(t)

	contextID, err := store.Save(ctx, &CallContext{
		ContextID:      "outbound-failed-claim-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "sip",
		Direction:      "outbound",
	})
	if err != nil {
		t.Fatalf("failed to save call context: %v", err)
	}

	err = store.UpdateCallStatus(ctx, contextID, CallStatusUpdate{
		CallStatus:       CallStatusFailed,
		FailureClass:     "setup",
		DisconnectReason: "outbound_setup_failed",
	})
	if err != nil {
		t.Fatalf("UpdateCallStatus returned error: %v", err)
	}

	if err := store.UpdateField(ctx, contextID, "status", StatusClaimed); err != nil {
		t.Fatalf("UpdateField returned error: %v", err)
	}

	got, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("expected failed context not to be claimed, got %q", got.Status)
	}
}

func TestStoreGetByChannelUUID(t *testing.T) {
	store, ctx := newTestStore(t)

	older := &CallContext{
		ContextID:      "older-context",
		AssistantID:    77,
		ConversationID: 1001,
		Provider:       "vonage",
		ChannelUUID:    "call-uuid-1",
		Direction:      "outbound",
		CreatedDate:    time.Now().Add(-time.Minute),
	}
	newer := &CallContext{
		ContextID:      "newer-context",
		AssistantID:    77,
		ConversationID: 1002,
		Provider:       "vonage",
		ChannelUUID:    "call-uuid-1",
		Direction:      "outbound",
		CreatedDate:    time.Now(),
	}
	otherProvider := &CallContext{
		ContextID:      "twilio-context",
		AssistantID:    77,
		ConversationID: 1003,
		Provider:       "twilio",
		ChannelUUID:    "call-uuid-1",
		CreatedDate:    time.Now(),
	}
	otherAssistant := &CallContext{
		ContextID:      "other-assistant-context",
		AssistantID:    88,
		ConversationID: 1004,
		Provider:       "vonage",
		ChannelUUID:    "call-uuid-1",
		CreatedDate:    time.Now(),
	}

	for _, cc := range []*CallContext{older, newer, otherProvider, otherAssistant} {
		if _, err := store.Save(ctx, cc); err != nil {
			t.Fatalf("failed to save call context %s: %v", cc.ContextID, err)
		}
	}

	got, err := store.GetByChannelUUID(ctx, "vonage", 77, "call-uuid-1")
	if err != nil {
		t.Fatalf("GetByChannelUUID returned error: %v", err)
	}
	if got.ContextID != "newer-context" {
		t.Fatalf("expected newest vonage context, got %s", got.ContextID)
	}

	if _, err := store.GetByChannelUUID(ctx, "vonage", 99, "call-uuid-1"); err == nil {
		t.Fatal("expected error for unknown assistant")
	}
	if _, err := store.GetByChannelUUID(ctx, "exotel", 77, "call-uuid-1"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if _, err := store.GetByChannelUUID(ctx, " ", 77, "call-uuid-1"); err == nil {
		t.Fatal("expected error for blank provider")
	}
	if _, err := store.GetByChannelUUID(ctx, "vonage", 0, "call-uuid-1"); err == nil {
		t.Fatal("expected error for zero assistant id")
	}
	if _, err := store.GetByChannelUUID(ctx, "vonage", 77, ""); err == nil {
		t.Fatal("expected error for empty channel uuid")
	}
	if _, err := store.GetByChannelUUID(ctx, "vonage", 77, " "); err == nil {
		t.Fatal("expected error for blank channel uuid")
	}
}

func TestStoreAuthenticationSnapshotRoundTripDoesNotWriteLegacyToken(t *testing.T) {
	store, ctx := newTestStore(t)
	auth := &types.Authentication{
		AuthType:          types.AuthTypeUser,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeUser, ID: math.MaxInt64},
		UserValue:         &types.UserContext{UserID: math.MaxInt64},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 33},
		ProjectValue:      &types.ProjectContext{OrganizationID: 33, ProjectID: 22},
	}
	callContext := &CallContext{AssistantID: 77, ConversationID: 1001, Provider: "sip", Direction: "outbound"}
	if err := callContext.SetAuthentication(auth); err != nil {
		t.Fatalf("SetAuthentication() error = %v", err)
	}
	contextID, err := store.Save(ctx, callContext)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	stored, err := store.Get(ctx, contextID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	reconstructed, err := stored.ToAuthentication()
	if err != nil {
		t.Fatalf("ToAuthentication() error = %v", err)
	}
	userContext, err := reconstructed.UserContext()
	if err != nil || userContext.UserID != math.MaxInt64 {
		t.Fatalf("UserContext() = %+v, %v", userContext, err)
	}
	actor := reconstructed.Actor()
	if actor.ID != math.MaxInt64 {
		t.Fatalf("Actor() = %+v", actor)
	}
	var authToken string
	testStore := store.(*postgresStore)
	if err := testStore.postgres.DB(ctx).Raw("SELECT auth_token FROM call_contexts WHERE context_id = ?", contextID).Scan(&authToken).Error; err != nil {
		t.Fatalf("read legacy auth_token: %v", err)
	}
	if authToken != "" {
		t.Fatalf("legacy auth_token = %q, want empty", authToken)
	}
}
