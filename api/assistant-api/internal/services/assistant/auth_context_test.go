package internal_assistant_service

import (
	"context"
	"strconv"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type auditTestPostgresConnector struct {
	db *gorm.DB
}

func (connector *auditTestPostgresConnector) Connect(context.Context) error { return nil }
func (connector *auditTestPostgresConnector) Name() string                  { return "audit-test-postgres" }
func (connector *auditTestPostgresConnector) IsConnected(context.Context) bool {
	return connector.db != nil
}
func (connector *auditTestPostgresConnector) Disconnect(context.Context) error { return nil }
func (connector *auditTestPostgresConnector) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *auditTestPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

type auditTestStorage struct{}

func (auditTestStorage) Name() string { return "audit-test-storage" }
func (auditTestStorage) Store(context.Context, string, []byte) storages.StorageOutput {
	return storages.StorageOutput{}
}
func (auditTestStorage) Get(context.Context, string) storages.GetStorageOutput {
	return storages.GetStorageOutput{}
}
func (auditTestStorage) GetUrl(context.Context, string) storages.StorageOutput {
	return storages.StorageOutput{}
}

func TestScopeReadRequiresProjectCapability(t *testing.T) {
	organizationID := uint64(11)
	auth := directOrganizationPrincipal(organizationID, organizationID)

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 22, 33); err == nil {
		t.Fatal("GetLog() error = nil without project capability")
	}
}

func TestScopeReadRejectsMismatchedProject(t *testing.T) {
	organizationID := uint64(11)
	projectID := uint64(22)
	auth := directProjectPrincipal(projectID, organizationID, projectID)

	service := &assistantToolService{}
	if _, err := service.GetLog(context.Background(), auth, 23, 33); err == nil {
		t.Fatal("GetLog() error = nil for mismatched project")
	}
}

func TestAssistantCreatesSetOnlyCreatedActor(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun: true,
		Logger: logger.Discard.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	applicationLogger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	auth := directProjectPrincipal(22, 11, 22)
	service := &assistantConversationService{
		logger:   applicationLogger,
		postgres: &auditTestPostgresConnector{db: database},
		storage:  auditTestStorage{},
	}

	conversation, err := service.CreateConversation(
		context.Background(), auth, "conversation", 33, 44, type_enums.DIRECTION_INBOUND, utils.SDK,
	)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if conversation.CreatedActorType != "project" || conversation.CreatedActorID != 22 {
		t.Fatalf("CreateConversation() created actor = (%v, %v)", conversation.CreatedActorType, conversation.CreatedActorID)
	}
	if conversation.UpdatedActorType != "" || conversation.UpdatedActorID != 0 {
		t.Fatalf("CreateConversation() updated actor = (%v, %v), want zero values", conversation.UpdatedActorType, conversation.UpdatedActorID)
	}

	message, err := service.CreateConversationMessage(
		context.Background(), auth, utils.SDK, 33, 44, conversation.Id, "message", "user", "hello",
	)
	if err != nil {
		t.Fatalf("CreateConversationMessage() error = %v", err)
	}
	if message.CreatedActorType != "project" || message.CreatedActorID != 22 {
		t.Fatalf("CreateConversationMessage() created actor = (%v, %v)", message.CreatedActorType, message.CreatedActorID)
	}
	if message.UpdatedActorType != "" || message.UpdatedActorID != 0 {
		t.Fatalf("CreateConversationMessage() updated actor = (%v, %v), want zero values", message.UpdatedActorType, message.UpdatedActorID)
	}

	httpLogService := &assistantHTTPLogService{
		logger:   applicationLogger,
		postgres: &auditTestPostgresConnector{db: database},
		storage:  auditTestStorage{},
	}
	httpLog, err := httpLogService.CreateLog(
		context.Background(), auth, "webhook", 55, "request", "context", 33, &conversation.Id,
		"https://example.com", "POST", 200, 10, 0, type_enums.RECORD_COMPLETE, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("CreateLog() error = %v", err)
	}
	if httpLog.CreatedActorType != "project" || httpLog.CreatedActorID != 22 {
		t.Fatalf("CreateLog() created actor = (%v, %v)", httpLog.CreatedActorType, httpLog.CreatedActorID)
	}
	if httpLog.UpdatedActorType != "" || httpLog.UpdatedActorID != 0 {
		t.Fatalf("CreateLog() updated actor = (%v, %v), want zero values", httpLog.UpdatedActorType, httpLog.UpdatedActorID)
	}
}

func TestConversationMetadataUpsertSetsUpdatedActor(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.Exec(`CREATE TABLE assistant_conversation_metadata (
		id integer primary key,
		created_date datetime,
		updated_date datetime,
		status text default 'ACTIVE' not null,
		created_actor_type text,
		created_actor_id integer,
		updated_actor_type text,
		updated_actor_id integer,
		key text not null,
		value text not null,
		assistant_id integer not null,
		assistant_conversation_id integer not null,
		unique (assistant_conversation_id, key)
	)`).Error; err != nil {
		t.Fatalf("create metadata table: %v", err)
	}
	applicationLogger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	auth := directUserPrincipal(41, authentication.Tenant{OrganizationID: 11})
	service := &assistantConversationService{
		logger:   applicationLogger,
		postgres: &auditTestPostgresConnector{db: database},
	}

	if _, err := service.CreateOrUpdateConversationMetadata(
		context.Background(), auth, 33, 44, []*protos.Metadata{{Key: "language", Value: "en"}},
	); err != nil {
		t.Fatalf("initial metadata create: %v", err)
	}
	auth = directServicePrincipal(52, authentication.Tenant{OrganizationID: 11})
	if _, err := service.CreateOrUpdateConversationMetadata(
		context.Background(), auth, 33, 44, []*protos.Metadata{{Key: "language", Value: "fr"}},
	); err != nil {
		t.Fatalf("metadata update: %v", err)
	}

	var row struct {
		Value            string
		CreatedActorType string
		CreatedActorID   uint64
		UpdatedActorType string
		UpdatedActorID   uint64
	}
	if err := database.Table("assistant_conversation_metadata").First(&row).Error; err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if row.Value != "fr" {
		t.Fatalf("metadata value = %q, want fr", row.Value)
	}
	if row.CreatedActorType != "user" || row.CreatedActorID != 41 {
		t.Fatalf("created actor = (%v, %v)", row.CreatedActorType, row.CreatedActorID)
	}
	if row.UpdatedActorType != "service" || row.UpdatedActorID != 52 {
		t.Fatalf("updated actor = (%v, %v)", row.UpdatedActorType, row.UpdatedActorID)
	}
}

func directUserPrincipal(userID uint64, tenant authentication.Tenant) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: userID}, tenant,
		authentication.Proof{Method: authentication.AuthenticationUserSession, Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: strconv.FormatUint(userID, 10)}}, nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}

func directProjectPrincipal(credentialID, organizationID, projectID uint64) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityProjectCredential, ID: credentialID},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{Method: authentication.AuthenticationProjectCredential, Audience: authentication.AudienceManagement, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: strconv.FormatUint(credentialID, 10)}}, nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}

func directOrganizationPrincipal(credentialID, organizationID uint64) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityOrganizationCredential, ID: credentialID},
		authentication.Tenant{OrganizationID: organizationID},
		authentication.Proof{Method: authentication.AuthenticationOrganizationCredential, Reference: authentication.ProofReference{Type: authentication.ProofReferenceCredential, Value: strconv.FormatUint(credentialID, 10)}}, nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}

func directServicePrincipal(serviceID uint64, tenant authentication.Tenant) authentication.Principal {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: serviceID, Issuer: "assistant-api"}, tenant,
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "assistant-test-1"}}, nil,
	)
	if err != nil {
		panic(err)
	}
	return principal
}
