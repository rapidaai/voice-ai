package internal_notification_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
)

type notificationTestPostgres struct {
	db *gorm.DB
}

func (p *notificationTestPostgres) Connect(context.Context) error    { return nil }
func (p *notificationTestPostgres) Name() string                     { return "notification-test" }
func (p *notificationTestPostgres) IsConnected(context.Context) bool { return true }
func (p *notificationTestPostgres) Disconnect(context.Context) error { return nil }
func (p *notificationTestPostgres) Query(ctx context.Context, query string, dest interface{}) error {
	return p.DB(ctx).Raw(query).Scan(dest).Error
}
func (p *notificationTestPostgres) DB(ctx context.Context) *gorm.DB { return p.db.WithContext(ctx) }

var _ connectors.PostgresConnector = (*notificationTestPostgres)(nil)

func TestUpdateNotificationSettingAuditsCreateAndConflictUpdate(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`
		CREATE TABLE notification_settings (
			id INTEGER PRIMARY KEY,
			created_date DATETIME,
			updated_date DATETIME,
			status TEXT,
			created_actor_type TEXT,
			created_actor_id INTEGER,
			updated_actor_type TEXT,
			updated_actor_id INTEGER,
			user_auth_id INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			channel TEXT NOT NULL,
			enabled BOOLEAN NOT NULL,
			UNIQUE(channel, event_type, user_auth_id)
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	service := NewNotificationService(logger, &notificationTestPostgres{db: database})
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 11},
		authentication.Tenant{OrganizationID: 22},
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "notification-test"}}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	setting := &protos.NotificationSetting{EventType: "call.completed", Channel: "email", Enabled: true}
	if _, err := service.UpdateNotificationSetting(context.Background(), auth, 11, []*protos.NotificationSetting{setting}); err != nil {
		t.Fatal(err)
	}
	var createdType string
	var createdID uint64
	var updatedType sql.NullString
	var updatedID sql.NullInt64
	row := database.Raw(`SELECT created_actor_type, created_actor_id, updated_actor_type, updated_actor_id
		FROM notification_settings WHERE user_auth_id = 11`).Row()
	if err := row.Scan(&createdType, &createdID, &updatedType, &updatedID); err != nil {
		t.Fatal(err)
	}
	if createdType != "user" || createdID != 11 || updatedType.Valid || updatedID.Valid {
		t.Fatalf("created audit = %q/%d, updated audit = %v/%v", createdType, createdID, updatedType, updatedID)
	}

	setting.Enabled = false
	if _, err := service.UpdateNotificationSetting(context.Background(), auth, 11, []*protos.NotificationSetting{setting}); err != nil {
		t.Fatal(err)
	}
	row = database.Raw(`SELECT updated_actor_type, updated_actor_id
		FROM notification_settings WHERE user_auth_id = 11`).Row()
	if err := row.Scan(&updatedType, &updatedID); err != nil {
		t.Fatal(err)
	}
	if !updatedType.Valid || updatedType.String != "user" || !updatedID.Valid || updatedID.Int64 != 11 {
		t.Fatalf("updated audit = %v/%v", updatedType, updatedID)
	}

	if _, err := service.UpdateNotificationSetting(context.Background(), auth, 12, []*protos.NotificationSetting{setting}); err == nil {
		t.Fatal("UpdateNotificationSetting() cross-subject error = nil")
	} else {
		var denied pkg_errors.RouteAuthorizationDeniedError
		if !errors.As(err, &denied) {
			t.Fatalf("UpdateNotificationSetting() cross-subject error = %v", err)
		}
	}
	if _, err := service.GetAllNotificationSetting(context.Background(), auth, 12); err == nil {
		t.Fatal("GetAllNotificationSetting() cross-subject error = nil")
	} else {
		var denied pkg_errors.RouteAuthorizationDeniedError
		if !errors.As(err, &denied) {
			t.Fatalf("GetAllNotificationSetting() cross-subject error = %v", err)
		}
	}
}
