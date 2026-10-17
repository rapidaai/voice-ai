package internal_dashboard_service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type dashboardTestPostgres struct {
	db *gorm.DB
}

func (connector *dashboardTestPostgres) Connect(context.Context) error { return nil }
func (connector *dashboardTestPostgres) Name() string                  { return "dashboard-test" }
func (connector *dashboardTestPostgres) IsConnected(context.Context) bool {
	return true
}
func (connector *dashboardTestPostgres) Disconnect(context.Context) error { return nil }
func (connector *dashboardTestPostgres) Query(ctx context.Context, query string, destination interface{}) error {
	return connector.DB(ctx).Raw(query).Scan(destination).Error
}
func (connector *dashboardTestPostgres) DB(ctx context.Context) *gorm.DB {
	if transaction, ok := connectors.PostgresTxFromContext(ctx); ok {
		return transaction.WithContext(ctx)
	}
	return connector.db.WithContext(ctx)
}

func projectDashboardAnalyticsTestService(t *testing.T) (*projectDashboardAnalyticsService, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE assistants (id integer primary key, created_date datetime, status text, organization_id integer, project_id integer, name text)`,
		`CREATE TABLE assistant_conversations (id integer primary key, created_date datetime, status text, organization_id integer, project_id integer, assistant_id integer, identifier text, source text)`,
		`CREATE TABLE assistant_conversation_metrics (id integer primary key, created_date datetime, status text, name text, value text, description text, assistant_id integer, assistant_conversation_id integer)`,
		`CREATE TABLE assistant_conversation_metadata (id integer primary key, created_date datetime, status text, key text, value text, assistant_id integer, assistant_conversation_id integer)`,
		`CREATE TABLE assistant_conversation_messages (
			id integer primary key, created_date datetime, updated_date datetime, status text,
			created_actor_type text, created_actor_id integer, updated_actor_type text, updated_actor_id integer,
			message_id text, assistant_conversation_id integer, assistant_id integer,
			assistant_provider_model_id integer, source text, role text, body text
		)`,
		`CREATE TABLE assistant_conversation_message_metrics (id integer primary key, created_date datetime, status text, name text, value text, description text, assistant_conversation_id integer, assistant_conversation_message_id text)`,
	}
	for _, statement := range statements {
		require.NoError(t, database.Exec(statement).Error)
	}
	return &projectDashboardAnalyticsService{postgres: &dashboardTestPostgres{db: database}}, database
}

func dashboardTestProjectContext(organizationID, projectID uint64) types.ProjectContext {
	return types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID}
}

func dashboardRequest(fromDate, toDate time.Time) *protos.GetProjectDashboardAnalyticsRequest {
	duration := toDate.Sub(fromDate)
	return &protos.GetProjectDashboardAnalyticsRequest{
		FromDate:           timestamppb.New(fromDate),
		ToDate:             timestamppb.New(toDate),
		ComparisonFromDate: timestamppb.New(fromDate.Add(-duration)),
		ComparisonToDate:   timestamppb.New(fromDate),
		AssistantLimit:     3,
		RecentFailureLimit: 5,
	}
}

func insertDashboardConversation(t *testing.T, database *gorm.DB, id, organizationID, projectID, assistantID uint64, createdAt time.Time, status, cause string) {
	t.Helper()
	require.NoError(t, database.Table("assistant_conversations").Create(map[string]interface{}{
		"id": id, "created_date": createdAt, "status": type_enums.RECORD_ACTIVE, "organization_id": organizationID,
		"project_id": projectID, "assistant_id": assistantID, "identifier": id, "source": "phone",
	}).Error)
	require.NoError(t, database.Table("assistant_conversation_metrics").Create(map[string]interface{}{
		"id": id + 1000, "created_date": createdAt, "status": type_enums.RECORD_ACTIVE, "name": observability.MetricConversationStatus,
		"value": status, "assistant_id": assistantID, "assistant_conversation_id": id,
	}).Error)
	if cause != "" {
		require.NoError(t, database.Table("assistant_conversation_metadata").Create(map[string]interface{}{
			"id": id + 2000, "created_date": createdAt, "status": type_enums.RECORD_ACTIVE, "key": "disconnect_reason",
			"value": cause, "assistant_id": assistantID, "assistant_conversation_id": id,
		}).Error)
	}
	messageID := "message-" + createdAt.Format(time.RFC3339Nano)
	require.NoError(t, database.Table("assistant_conversation_messages").Create(map[string]interface{}{
		"id": id + 3000, "created_date": createdAt, "status": type_enums.RECORD_ACTIVE, "message_id": messageID,
		"assistant_conversation_id": id, "assistant_id": assistantID, "role": "assistant", "source": "assistant",
	}).Error)
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Create(map[string]interface{}{
		"id": id + 4000, "created_date": createdAt, "status": type_enums.RECORD_ACTIVE, "name": observability.MetricSTTLatencyMs,
		"value": "800", "assistant_conversation_id": id, "assistant_conversation_message_id": messageID,
	}).Error)
}

func TestGetProjectDashboardAnalyticsRejectsNilRequest(t *testing.T) {
	service, _ := projectDashboardAnalyticsTestService(t)
	_, err := service.GetProjectDashboardAnalytics(context.Background(), dashboardTestProjectContext(10, 100), nil)
	require.ErrorIs(t, err, ErrInvalidProjectDashboardAnalyticsRequest)
}

func TestGetProjectDashboardAnalyticsAggregatesTenantData(t *testing.T) {
	service, database := projectDashboardAnalyticsTestService(t)
	fromDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	toDate := fromDate.Add(24 * time.Hour)
	require.NoError(t, database.Table("assistants").Create(map[string]interface{}{"id": 1, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Support bot"}).Error)
	require.NoError(t, database.Table("assistants").Create(map[string]interface{}{"id": 2, "status": type_enums.RECORD_ACTIVE, "organization_id": 20, "project_id": 200, "name": "Foreign bot"}).Error)
	require.NoError(t, database.Table("assistants").Create(map[string]interface{}{"id": 3, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 200, "name": "Other project bot"}).Error)
	require.NoError(t, database.Table("assistants").Create(map[string]interface{}{"id": 4, "status": type_enums.RECORD_ACTIVE, "organization_id": 20, "project_id": 100, "name": "Other organization bot"}).Error)
	insertDashboardConversation(t, database, 11, 10, 100, 1, fromDate.Add(time.Hour), "FAILED", "provider_upstream_failure")
	insertDashboardConversation(t, database, 12, 10, 100, 1, fromDate.Add(2*time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 13, 10, 100, 1, fromDate.Add(-time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 14, 20, 200, 2, fromDate.Add(time.Hour), "FAILED", "network_failure")
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id = ?", 14).Update("value", "9000").Error)
	insertDashboardConversation(t, database, 15, 10, 200, 3, fromDate.Add(time.Hour), "FAILED", "network_failure")
	insertDashboardConversation(t, database, 16, 20, 100, 4, fromDate.Add(time.Hour), "FAILED", "network_failure")
	insertDashboardConversation(t, database, 17, 10, 100, 1, toDate, "FAILED", "network_failure")

	analytics, err := service.GetProjectDashboardAnalytics(context.Background(), dashboardTestProjectContext(10, 100), dashboardRequest(fromDate, toDate))
	require.NoError(t, err)
	require.Equal(t, uint64(2), analytics.GetSummary().GetSessions().GetValue())
	require.Equal(t, uint64(1), analytics.GetSummary().GetFailedSessions().GetValue())
	require.InDelta(t, 50, analytics.GetSummary().GetSuccessRate().GetValuePercent(), 0.01)
	require.Equal(t, uint64(800), analytics.GetSummary().GetResponseLatency().GetValueMs())
	require.Equal(t, uint64(1), analytics.GetSummary().GetSessions().GetPreviousValue())
	require.InDelta(t, 100, analytics.GetSummary().GetSessions().GetChangePercent(), 0.01)
	require.InDelta(t, 100, analytics.GetSummary().GetSuccessRate().GetPreviousValuePercent(), 0.01)
	require.InDelta(t, -50, analytics.GetSummary().GetSuccessRate().GetChangePoints(), 0.01)
	require.Equal(t, uint64(800), analytics.GetSummary().GetResponseLatency().GetPreviousValueMs())
	require.Equal(t, int64(0), analytics.GetSummary().GetResponseLatency().GetChangeMs())
	require.Equal(t, uint32(1), analytics.GetProjectState().GetAssistantCount())
	require.Len(t, analytics.GetAssistants(), 1)
	require.Equal(t, protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NEEDS_ATTENTION, analytics.GetAssistants()[0].GetHealth())
	require.Len(t, analytics.GetRecentFailures(), 1)
	require.Equal(t, protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_PROVIDER, analytics.GetRecentFailures()[0].GetFailureCode())
	require.Len(t, analytics.GetActivity(), 48)
	var activitySessions, activityFailures uint64
	for _, bucket := range analytics.GetActivity() {
		activitySessions += bucket.GetSessions()
		activityFailures += bucket.GetFailedSessions()
	}
	require.Equal(t, analytics.GetSummary().GetSessions().GetValue(), activitySessions)
	require.Equal(t, analytics.GetSummary().GetFailedSessions().GetValue(), activityFailures)
}

func TestGetProjectDashboardAnalyticsIgnoresInvalidAndNonPositiveMetrics(t *testing.T) {
	service, database := projectDashboardAnalyticsTestService(t)
	fromDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	toDate := fromDate.Add(24 * time.Hour)
	require.NoError(t, database.Table("assistants").Create(map[string]interface{}{
		"id": 1, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Support bot",
	}).Error)
	insertDashboardConversation(t, database, 1, 10, 100, 1, fromDate.Add(time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 2, 10, 100, 1, fromDate.Add(2*time.Hour), "FAILED", "provider_failure")
	insertDashboardConversation(t, database, 3, 10, 100, 1, fromDate.Add(3*time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 4, 10, 100, 1, fromDate.Add(4*time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 5, 10, 100, 1, fromDate.Add(5*time.Hour), "COMPLETE", "")
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id = ?", 2).Update("value", "not-a-number").Error)
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id = ?", 3).Update("value", "0").Error)
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id = ?", 4).Update("value", "-100").Error)
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id = ?", 5).Update("value", "9999999999999999999999999999999999999999999999999999999999999999").Error)
	require.NoError(t, database.Table("assistant_conversation_metrics").Create([]map[string]interface{}{
		{"id": 5001, "name": observability.MetricConversationDuration, "value": "1500", "assistant_id": 1, "assistant_conversation_id": 1},
		{"id": 5002, "name": observability.MetricConversationDuration, "value": "not-a-number", "assistant_id": 1, "assistant_conversation_id": 2},
	}).Error)

	analytics, err := service.GetProjectDashboardAnalytics(context.Background(), dashboardTestProjectContext(10, 100), dashboardRequest(fromDate, toDate))
	require.NoError(t, err)
	require.Equal(t, uint64(800), analytics.GetSummary().GetResponseLatency().GetValueMs())
	require.Equal(t, uint64(800), analytics.GetAssistants()[0].GetResponseLatencyMs())
	require.Equal(t, uint64(1500), analytics.GetAssistants()[0].GetConversationDurationMs())
	require.Equal(t, uint64(0), analytics.GetHealth().GetSlowSessions())
	require.Len(t, analytics.GetRecentFailures(), 1)
	require.Equal(t, uint64(0), analytics.GetRecentFailures()[0].GetDurationMs())
}

func TestProjectDashboardQueriesUseGuardedPostgresMetricValues(t *testing.T) {
	sqlDatabase, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, sqlDatabase.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})
	database, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDatabase}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	service := &projectDashboardAnalyticsService{postgres: &dashboardTestPostgres{db: database}}
	fromDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT COUNT.*FROM assistant_conversations`).WillReturnRows(sqlmock.NewRows([]string{"sessions", "failed"}).AddRow(1, 0))
	mock.ExpectQuery(`SELECT AVG.*BTRIM\(assistant_conversation_message_metrics.value\).*DOUBLE PRECISION.*assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id`).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(800))

	sessions, failed, latency, err := service.getProjectDashboardSummary(context.Background(), dashboardTestProjectContext(10, 100), fromDate, fromDate.Add(24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, uint64(1), sessions)
	require.Equal(t, uint64(0), failed)
	require.Equal(t, uint64(800), *latency)

	mock.ExpectQuery(`SELECT assistants.id.*FROM assistants.*BIGINT`).
		WillReturnRows(sqlmock.NewRows([]string{"assistant_id", "name", "sessions", "failed", "conversation_duration", "last_activity"}))
	mock.ExpectQuery(`SELECT assistant_latency.assistant_id.*DOUBLE PRECISION.*assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id`).
		WillReturnRows(sqlmock.NewRows([]string{"assistant_id", "value"}))
	mock.ExpectQuery(`SELECT COUNT.*DOUBLE PRECISION.*assistant_conversation_messages.assistant_conversation_id = assistant_conversation_message_metrics.assistant_conversation_id`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	_, _, _, err = service.getProjectDashboardAssistants(context.Background(), dashboardTestProjectContext(10, 100), fromDate, fromDate.Add(24*time.Hour), 0, 3)
	require.NoError(t, err)

	mock.ExpectQuery(`SELECT assistant_conversations.id.*FROM assistant_conversations.*BIGINT.*LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{"conversation_id", "assistant_id", "assistant_name", "occurred_at", "channel", "duration"}))
	_, err = service.getProjectDashboardRecentFailures(context.Background(), dashboardTestProjectContext(10, 100), fromDate, fromDate.Add(24*time.Hour), 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildActivityUsesRangeCadenceAndHalfOpenBounds(t *testing.T) {
	fromDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		duration time.Duration
		buckets  int
	}{
		{name: "24 hours", duration: 24 * time.Hour, buckets: 48},
		{name: "7 days", duration: 7 * 24 * time.Hour, buckets: 42},
		{name: "30 days", duration: 30 * 24 * time.Hour, buckets: 30},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dashboardService, database := projectDashboardAnalyticsTestService(t)
			toDate := fromDate.Add(test.duration)
			insertDashboardConversation(t, database, 1, 10, 100, 1, fromDate, "FAILED", "")
			insertDashboardConversation(t, database, 2, 10, 100, 1, toDate.Add(-time.Second), "COMPLETE", "")
			insertDashboardConversation(t, database, 3, 10, 100, 1, toDate, "FAILED", "")
			buckets, err := dashboardService.getProjectDashboardActivity(context.Background(), dashboardTestProjectContext(10, 100), fromDate, toDate)
			require.NoError(t, err)
			require.Len(t, buckets, test.buckets)
			require.Equal(t, fromDate, buckets[0].GetFromDate().AsTime())
			require.Equal(t, toDate, buckets[len(buckets)-1].GetToDate().AsTime())
			var sessions, failures uint64
			for _, bucket := range buckets {
				sessions += bucket.GetSessions()
				failures += bucket.GetFailedSessions()
			}
			require.Equal(t, uint64(2), sessions)
			require.Equal(t, uint64(1), failures)
		})
	}
}

func TestBuildAssistantRowsUsesComponentWeightedLatency(t *testing.T) {
	fromDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	dashboardService, database := projectDashboardAnalyticsTestService(t)
	require.NoError(t, database.Table("assistants").Create([]map[string]interface{}{
		{"id": 1, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Support bot"},
		{"id": 2, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "No data bot"},
	}).Error)
	insertDashboardConversation(t, database, 1, 10, 100, 1, fromDate, "COMPLETE", "")
	insertDashboardConversation(t, database, 2, 10, 100, 1, fromDate.Add(time.Hour), "COMPLETE", "")
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Where("assistant_conversation_id IN ?", []uint64{1, 2}).Update("value", "100").Error)
	require.NoError(t, database.Table("assistant_conversation_message_metrics").Create(map[string]interface{}{
		"id": 5002, "created_date": fromDate, "status": type_enums.RECORD_ACTIVE, "name": observability.MetricTTSLatencyMs,
		"value": "1900", "assistant_conversation_id": 2, "assistant_conversation_message_id": "message-" + fromDate.Add(time.Hour).Format(time.RFC3339Nano),
	}).Error)

	rows, health, assistantCount, err := dashboardService.getProjectDashboardAssistants(context.Background(), dashboardTestProjectContext(10, 100), fromDate, fromDate.Add(24*time.Hour), 0, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(2), assistantCount)
	require.Len(t, rows, 1)
	require.Equal(t, uint64(1000), rows[0].GetResponseLatencyMs())
	require.Equal(t, protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_WATCH, rows[0].GetHealth())
	require.Equal(t, uint32(1), health.GetWatchAssistants())

	rows, health, _, err = dashboardService.getProjectDashboardAssistants(context.Background(), dashboardTestProjectContext(10, 100), fromDate, fromDate.Add(24*time.Hour), 0, 2)
	require.NoError(t, err)
	require.Nil(t, rows[1].SuccessRatePercent)
	require.Nil(t, rows[1].ResponseLatencyMs)
	require.Equal(t, protos.DashboardAssistantHealth_DASHBOARD_ASSISTANT_HEALTH_NO_DATA, rows[1].GetHealth())
	require.Equal(t, uint32(1), health.GetNoDataAssistants())
}

func TestGetProjectDashboardAnalyticsAppliesStableLimitsAndOrdering(t *testing.T) {
	service, database := projectDashboardAnalyticsTestService(t)
	fromDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	toDate := fromDate.Add(24 * time.Hour)
	for _, assistant := range []map[string]interface{}{
		{"id": 10, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Ten"},
		{"id": 2, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Two"},
		{"id": 3, "status": type_enums.RECORD_ACTIVE, "organization_id": 10, "project_id": 100, "name": "Three"},
	} {
		require.NoError(t, database.Table("assistants").Create(assistant).Error)
	}
	insertDashboardConversation(t, database, 20, 10, 100, 10, fromDate.Add(2*time.Hour), "FAILED", "network_failure")
	insertDashboardConversation(t, database, 19, 10, 100, 2, fromDate.Add(2*time.Hour), "FAILED", "provider_failure")
	insertDashboardConversation(t, database, 21, 10, 100, 3, fromDate.Add(time.Hour), "COMPLETE", "")
	insertDashboardConversation(t, database, 22, 10, 100, 3, fromDate.Add(3*time.Hour), "COMPLETE", "")
	request := dashboardRequest(fromDate, toDate)
	request.AssistantLimit = 2
	request.RecentFailureLimit = 1

	analytics, err := service.GetProjectDashboardAnalytics(context.Background(), dashboardTestProjectContext(10, 100), request)
	require.NoError(t, err)
	require.Len(t, analytics.GetAssistants(), 2)
	require.Equal(t, "3", analytics.GetAssistants()[0].GetAssistantId())
	require.Equal(t, "2", analytics.GetAssistants()[1].GetAssistantId())
	require.Len(t, analytics.GetRecentFailures(), 1)
	require.Equal(t, "19", analytics.GetRecentFailures()[0].GetConversationId())
}

func TestGetProjectDashboardAnalyticsRejectsInvalidRequest(t *testing.T) {
	service, _ := projectDashboardAnalyticsTestService(t)
	toDate := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	fromDate := toDate.Add(-24 * time.Hour)

	invalidTimestamp := dashboardRequest(fromDate, toDate)
	invalidTimestamp.FromDate = &timestamppb.Timestamp{Seconds: 253402300800}
	zeroDuration := dashboardRequest(fromDate, toDate)
	zeroDuration.FromDate = zeroDuration.ToDate
	unequalComparison := dashboardRequest(fromDate, toDate)
	unequalComparison.ComparisonFromDate = timestamppb.New(fromDate.Add(-2 * time.Hour))
	unanchoredComparison := dashboardRequest(fromDate, toDate)
	unanchoredComparison.ComparisonFromDate = timestamppb.New(fromDate.Add(-48 * time.Hour))
	unanchoredComparison.ComparisonToDate = timestamppb.New(fromDate.Add(-24 * time.Hour))
	zeroAssistantLimit := dashboardRequest(fromDate, toDate)
	zeroAssistantLimit.AssistantLimit = 0
	highAssistantLimit := dashboardRequest(fromDate, toDate)
	highAssistantLimit.AssistantLimit = 11
	zeroFailureLimit := dashboardRequest(fromDate, toDate)
	zeroFailureLimit.RecentFailureLimit = 0
	highFailureLimit := dashboardRequest(fromDate, toDate)
	highFailureLimit.RecentFailureLimit = 21

	tests := []struct {
		name    string
		request *protos.GetProjectDashboardAnalyticsRequest
	}{
		{name: "missing timestamps", request: &protos.GetProjectDashboardAnalyticsRequest{}},
		{name: "invalid timestamp", request: invalidTimestamp},
		{name: "zero duration", request: zeroDuration},
		{name: "range exceeds 90 days", request: dashboardRequest(toDate.Add(-91*24*time.Hour), toDate)},
		{name: "unequal comparison", request: unequalComparison},
		{name: "unanchored comparison", request: unanchoredComparison},
		{name: "zero assistant limit", request: zeroAssistantLimit},
		{name: "high assistant limit", request: highAssistantLimit},
		{name: "zero failure limit", request: zeroFailureLimit},
		{name: "high failure limit", request: highFailureLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.GetProjectDashboardAnalytics(context.Background(), dashboardTestProjectContext(10, 100), test.request)
			require.ErrorIs(t, err, ErrInvalidProjectDashboardAnalyticsRequest)
		})
	}
}

func TestFailureCodeMappings(t *testing.T) {
	tests := map[string]protos.DashboardFailureCode{
		"authentication failed": protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_AUTHENTICATION,
		"invalid config":        protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_CONFIGURATION,
		"upstream provider":     protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_PROVIDER,
		"network transport":     protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_NETWORK,
		"cancelled":             protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_CANCELLED,
		"no_answer":             protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_NO_ANSWER,
		"rejected":              protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_REJECTED,
		"other":                 protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_UNKNOWN,
	}
	for cause, want := range tests {
		require.Equal(t, want, projectDashboardFailureCode([]string{cause}), cause)
	}
}

func TestFailureCodeUsesPersistedStatusReason(t *testing.T) {
	require.Equal(t, protos.DashboardFailureCode_DASHBOARD_FAILURE_CODE_PROVIDER, projectDashboardFailureCode([]string{"provider response failed"}))
}
