package internal_assistant_service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDashboardTestService(t *testing.T) (*assistantService, sqlmock.Sqlmock) {
	t.Helper()

	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = database.Close()
	})

	gormDatabase, err := gorm.Open(postgres.New(postgres.Config{Conn: database}), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	return &assistantService{
		postgres: &auditTestPostgresConnector{db: gormDatabase},
	}, mock
}

func TestAssistantDashboardMessageSummaryUsesAgentTTFT(t *testing.T) {
	service, mock := newDashboardTestService(t)
	mock.ExpectQuery(`(?s)SELECT.*agent_ttft_value.*FROM assistant_conversation_messages`).
		WithArgs(
			observability.MetricSTTLatencyMs,
			observability.MetricEOSLatencyMs,
			observability.MetricTTSLatencyMs,
			observability.MetricAgentTTFTMs,
			observability.MetricAgentTotalToken,
			observability.MetricSTTLatencyMs,
			observability.MetricEOSLatencyMs,
			observability.MetricTTSLatencyMs,
			observability.MetricAgentTTFTMs,
			observability.MetricAgentTotalToken,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_messages",
			"user_messages",
			"average_stt_ms",
			"average_eos_ms",
			"average_tts_ms",
			"average_agent_ttft_ms",
			"total_tokens",
		}).AddRow(2, 1, 10, 20, 30, 40, 50))

	latency, _, _, _, err := service.getAssistantDashboardMessageSummary(
		context.Background(),
		&assistantDashboardSQLFilters{messageConditions: []string{"1 = 1"}},
	)

	require.NoError(t, err)
	require.Equal(t, float64(40), latency.GetLlmMs())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAssistantDashboardBucketsUseAgentTTFT(t *testing.T) {
	service, mock := newDashboardTestService(t)
	rangeStart := time.Date(2026, time.August, 24, 10, 0, 0, 0, time.UTC)
	rangeEnd := rangeStart.Add(30 * time.Minute)
	mock.ExpectQuery(`(?s)SELECT.*agent_ttft_value.*FROM assistant_conversation_messages`).
		WithArgs(
			rangeStart,
			int64(30*time.Minute/time.Second),
			observability.MetricSTTLatencyMs,
			observability.MetricEOSLatencyMs,
			observability.MetricTTSLatencyMs,
			observability.MetricAgentTTFTMs,
			observability.MetricSTTLatencyMs,
			observability.MetricEOSLatencyMs,
			observability.MetricTTSLatencyMs,
			observability.MetricAgentTTFTMs,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_index",
			"message_count",
			"average_stt_ms",
			"average_eos_ms",
			"average_tts_ms",
			"average_agent_ttft_ms",
		}).AddRow(0, 2, 10, 20, 30, 40))

	buckets, err := service.getAssistantDashboardBuckets(
		context.Background(),
		&assistantDashboardSQLFilters{
			messageConditions: []string{"1 = 1"},
			rangeStart:        rangeStart,
			rangeEnd:          rangeEnd,
		},
	)

	require.NoError(t, err)
	require.Len(t, buckets, 1)
	require.Equal(t, float64(40), buckets[0].GetLlmLatencyMs())
	require.NoError(t, mock.ExpectationsWereMet())
}
